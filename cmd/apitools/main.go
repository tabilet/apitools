package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/OpenUdon/apitools"
	"github.com/OpenUdon/apitools/sqlitecache"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, out, errOut io.Writer) int {
	if len(args) == 0 {
		usage(out)
		return 2
	}
	switch args[0] {
	case "search":
		return runSearch(args[1:], out, errOut)
	case "import":
		return runImport(args[1:], out, errOut)
	case "-h", "--help", "help":
		usage(out)
		return 0
	default:
		fmt.Fprintf(errOut, "unknown command %q\n", args[0])
		usage(errOut)
		return 2
	}
}

func usage(out io.Writer) {
	fmt.Fprintln(out, "Usage: apitools <command>")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Commands:")
	fmt.Fprintln(out, "  search  search APIs.guru with public-apis fallback")
	fmt.Fprintln(out, "  import  download and validate an OpenAPI document")
}

func runSearch(args []string, out, errOut io.Writer) int {
	return runSearchWithClient(args, out, errOut, clientForCache)
}

func runSearchWithClient(args []string, out, errOut io.Writer, newClient func(string) (*apitools.Client, func(), error)) int {
	fs := flag.NewFlagSet("apitools search", flag.ContinueOnError)
	fs.SetOutput(errOut)
	query := fs.String("query", "", "Search query")
	limit := fs.Int("limit", 10, "Maximum result count")
	source := fs.String("source", string(apitools.SourceAuto), "Search source: auto, apis-guru, or public-apis")
	publicProbe := fs.Int("public-probe", 0, "Maximum public-apis well-known URL probes; defaults to limit*5, capped at 50")
	probeTimeout := fs.Duration("probe-timeout", apitools.DefaultProbeTimeout, "Timeout for each public-apis OpenAPI probe")
	probeBudget := fs.Duration("probe-budget", apitools.DefaultPublicProbeBudget, "Overall time budget for public-apis probing")
	cachePath := fs.String("cache", "", "SQLite cache path; disabled when empty")
	cacheMode := fs.String("cache-mode", string(apitools.CacheModeReadWrite), "Cache mode: read-write, refresh, offline, or bypass")
	cacheTTL := fs.Duration("cache-ttl", apitools.DefaultCacheMaxAge, "Maximum age for cached search results")
	offline := fs.Bool("offline", false, "Use only cached search results; shorthand for --cache-mode offline")
	jsonOut := fs.Bool("json", false, "Write JSON output")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: apitools search --query <text> [--limit 10] [--source auto|apis-guru|public-apis] [--public-probe 25] [--probe-timeout 5s] [--probe-budget 30s] [--cache cache.sqlite] [--cache-mode read-write|refresh|offline|bypass] [--json]")
		fmt.Fprintln(fs.Output())
		fs.PrintDefaults()
	}
	if hasHelpFlag(args) {
		fs.SetOutput(out)
		fs.Usage()
		return 0
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	client, closeCache, err := newClient(*cachePath)
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	defer closeCache()
	client.ProbeTimeout = durationOrDefault(*probeTimeout, apitools.DefaultProbeTimeout)
	client.PublicProbeBudget = durationOrDefault(*probeBudget, apitools.DefaultPublicProbeBudget)
	mode := apitools.CacheMode(*cacheMode)
	if *offline {
		mode = apitools.CacheModeOffline
	}
	report, err := client.Search(ctx, apitools.SearchOptions{
		Query:       *query,
		Limit:       *limit,
		Source:      apitools.Source(*source),
		PublicProbe: *publicProbe,
		CacheMode:   mode,
		CacheMaxAge: *cacheTTL,
	})
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	if *jsonOut {
		if err := writeJSON(out, report); err != nil {
			fmt.Fprintln(errOut, err)
			return 1
		}
		return 0
	}
	if len(report.Results) == 0 {
		fmt.Fprintln(out, "No OpenAPI documents found.")
		return 0
	}
	for i, result := range report.Results {
		fmt.Fprintf(out, "%d. %s\n", i+1, result.Title)
		if result.Provider != "" {
			fmt.Fprintf(out, "   provider: %s\n", result.Provider)
		}
		fmt.Fprintf(out, "   source:   %s\n", result.Source)
		fmt.Fprintf(out, "   url:      %s\n", result.SpecURL)
		if strings.TrimSpace(result.Description) != "" {
			fmt.Fprintf(out, "   about:    %s\n", singleLine(result.Description))
		}
	}
	return 0
}

func durationOrDefault(value, fallback time.Duration) time.Duration {
	if value <= 0 {
		return fallback
	}
	return value
}

func runImport(args []string, out, errOut io.Writer) int {
	fs := flag.NewFlagSet("apitools import", flag.ContinueOnError)
	fs.SetOutput(errOut)
	rawURL := fs.String("url", "", "OpenAPI document URL")
	dir := fs.String("dir", "", "Directory to write the imported OpenAPI document")
	name := fs.String("name", "", "Suggested filename stem")
	cachePath := fs.String("cache", "", "SQLite cache path; disabled when empty")
	cacheMode := fs.String("cache-mode", string(apitools.CacheModeReadWrite), "Cache mode: read-write, refresh, offline, or bypass")
	cacheTTL := fs.Duration("cache-ttl", apitools.DefaultCacheMaxAge, "Maximum age for cached OpenAPI documents")
	offline := fs.Bool("offline", false, "Use only cached OpenAPI documents; shorthand for --cache-mode offline")
	jsonOut := fs.Bool("json", false, "Write JSON output")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: apitools import --url <openapi-url> --dir <target-dir> [--name <stem>] [--cache cache.sqlite] [--cache-mode read-write|refresh|offline|bypass] [--json]")
		fmt.Fprintln(fs.Output())
		fs.PrintDefaults()
	}
	if hasHelpFlag(args) {
		fs.SetOutput(out)
		fs.Usage()
		return 0
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	client, closeCache, err := clientForCache(*cachePath)
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	defer closeCache()
	mode := apitools.CacheMode(*cacheMode)
	if *offline {
		mode = apitools.CacheModeOffline
	}
	imported, err := client.Import(ctx, apitools.ImportOptions{
		URL:         *rawURL,
		Dir:         *dir,
		Name:        *name,
		CacheMode:   mode,
		CacheMaxAge: *cacheTTL,
	})
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	if *jsonOut {
		if err := writeJSON(out, imported); err != nil {
			fmt.Fprintln(errOut, err)
			return 1
		}
		return 0
	}
	fmt.Fprintf(out, "imported %s\n", imported.Path)
	if imported.Title != "" {
		fmt.Fprintf(out, "title: %s\n", imported.Title)
	}
	fmt.Fprintf(out, "sha256: %s\n", imported.SHA256)
	return 0
}

func writeJSON(out io.Writer, value any) error {
	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func clientForCache(path string) (*apitools.Client, func(), error) {
	if strings.TrimSpace(path) == "" {
		return &apitools.Client{}, func() {}, nil
	}
	cache, err := sqlitecache.Open(path)
	if err != nil {
		return nil, nil, err
	}
	return &apitools.Client{Cache: cache}, func() { _ = cache.Close() }, nil
}

func singleLine(value string) string {
	fields := strings.Fields(value)
	text := strings.Join(fields, " ")
	if len(text) > 160 {
		return text[:157] + "..."
	}
	return text
}

func hasHelpFlag(args []string) bool {
	for _, arg := range args {
		if arg == "-h" || arg == "--help" {
			return true
		}
	}
	return false
}
