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
	catalogpkg "github.com/OpenUdon/apitools/catalog"
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
	case "catalog":
		return runCatalog(args[1:], out, errOut)
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
	fmt.Fprintln(out, "  search   search APIs.guru with public-apis fallback")
	fmt.Fprintln(out, "  import   download and validate an OpenAPI document")
	fmt.Fprintln(out, "  catalog  inspect built-in provider catalog metadata")
}

func runCatalog(args []string, out, errOut io.Writer) int {
	if len(args) == 0 {
		catalogUsage(out)
		return 2
	}
	switch args[0] {
	case "list":
		return runCatalogList(args[1:], out, errOut)
	case "inspect":
		return runCatalogInspect(args[1:], out, errOut)
	case "security-report":
		return runCatalogSecurityReport(args[1:], out, errOut)
	case "-h", "--help", "help":
		catalogUsage(out)
		return 0
	default:
		fmt.Fprintf(errOut, "unknown catalog command %q\n", args[0])
		catalogUsage(errOut)
		return 2
	}
}

func catalogUsage(out io.Writer) {
	fmt.Fprintln(out, "Usage: apitools catalog <command>")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Commands:")
	fmt.Fprintln(out, "  list             list built-in provider catalog metadata")
	fmt.Fprintln(out, "  inspect          inspect one provider and resolution status")
	fmt.Fprintln(out, "  security-report  report auth/security status across providers")
}

func runCatalogList(args []string, out, errOut io.Writer) int {
	fs := flag.NewFlagSet("apitools catalog list", flag.ContinueOnError)
	fs.SetOutput(errOut)
	jsonOut := fs.Bool("json", false, "Write JSON output")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: apitools catalog list [--json]")
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
	if fs.NArg() > 0 {
		fmt.Fprintf(errOut, "unexpected argument %q\n", fs.Arg(0))
		fs.Usage()
		return 2
	}
	report, err := catalogpkg.BuiltInSecurityReport()
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	statusByProvider := map[string]catalogpkg.AuthCompletenessStatus{}
	for _, row := range catalogpkg.SecurityReportRows(report) {
		statusByProvider[row.ProviderID] = row.Status
	}
	rows := make([]catalogListRow, 0, len(catalogpkg.BuiltInProviders()))
	for _, provider := range catalogpkg.BuiltInProviders() {
		rows = append(rows, catalogListRow{
			ID:                  provider.ID,
			DisplayName:         provider.DisplayName,
			Category:            provider.Category,
			OpenAPIAvailability: provider.OfficialOpenAPIAvailability,
			MachineAvailability: provider.OfficialMachineSpecAvailability,
			UserOpenAPINeed:     provider.UserOpenAPINeed,
			AuthStatus:          statusByProvider[provider.ID],
		})
	}
	if *jsonOut {
		if err := writeJSON(out, rows); err != nil {
			fmt.Fprintln(errOut, err)
			return 1
		}
		return 0
	}
	fmt.Fprintf(out, "%-18s %-20s %-18s %-18s %-18s %s\n", "ID", "NAME", "OPENAPI", "MACHINE", "USER_OPENAPI", "AUTH")
	for _, row := range rows {
		fmt.Fprintf(out, "%-18s %-20s %-18s %-18s %-18s %s\n", row.ID, row.DisplayName, row.OpenAPIAvailability, row.MachineAvailability, row.UserOpenAPINeed, row.AuthStatus)
	}
	return 0
}

func runCatalogInspect(args []string, out, errOut io.Writer) int {
	var provider string
	parseArgs := args
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		provider = args[0]
		parseArgs = args[1:]
	}
	fs := flag.NewFlagSet("apitools catalog inspect", flag.ContinueOnError)
	fs.SetOutput(errOut)
	userOpenAPI := fs.String("openapi", "", "User-provided OpenAPI path or URL; overrides built-in specs")
	userOverlay := fs.String("security-overlay", "", "User-provided security overlay path or URL; overrides built-in security overlays")
	localOpenAPI := fs.String("local-openapi", "", "Project-local OpenAPI path; used before built-in specs")
	jsonOut := fs.Bool("json", false, "Write JSON output")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: apitools catalog inspect <provider> [--openapi path-or-url] [--security-overlay path-or-url] [--local-openapi path] [--json]")
		fmt.Fprintln(fs.Output())
		fs.PrintDefaults()
	}
	if hasHelpFlag(args) {
		fs.SetOutput(out)
		fs.Usage()
		return 0
	}
	if err := fs.Parse(parseArgs); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if provider == "" {
		if fs.NArg() > 0 {
			provider = fs.Arg(0)
		}
		if fs.NArg() > 1 {
			fmt.Fprintf(errOut, "unexpected argument %q\n", fs.Arg(1))
			fs.Usage()
			return 2
		}
	} else if fs.NArg() > 0 {
		fmt.Fprintf(errOut, "unexpected argument %q\n", fs.Arg(0))
		fs.Usage()
		return 2
	}
	if strings.TrimSpace(provider) == "" {
		fmt.Fprintln(errOut, "missing provider")
		fs.Usage()
		return 2
	}
	resolved, err := catalogpkg.ResolveProvider(catalogpkg.ResolveProviderOptions{
		ProviderKey:         provider,
		UserOpenAPI:         *userOpenAPI,
		UserSecurityOverlay: *userOverlay,
		ProjectLocalOpenAPI: *localOpenAPI,
	})
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	if *jsonOut {
		if err := writeJSON(out, resolved); err != nil {
			fmt.Fprintln(errOut, err)
			return 1
		}
		return 0
	}
	writeCatalogInspect(out, resolved)
	return 0
}

func runCatalogSecurityReport(args []string, out, errOut io.Writer) int {
	fs := flag.NewFlagSet("apitools catalog security-report", flag.ContinueOnError)
	fs.SetOutput(errOut)
	jsonOut := fs.Bool("json", false, "Write JSON output")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: apitools catalog security-report [--json]")
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
	if fs.NArg() > 0 {
		fmt.Fprintf(errOut, "unexpected argument %q\n", fs.Arg(0))
		fs.Usage()
		return 2
	}
	report, err := catalogpkg.BuiltInSecurityReport()
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	rows := catalogpkg.SecurityReportRows(report)
	if *jsonOut {
		if err := writeJSON(out, rows); err != nil {
			fmt.Fprintln(errOut, err)
			return 1
		}
		return 0
	}
	fmt.Fprintf(out, "%-18s %-22s %s\n", "PROVIDER", "AUTH_STATUS", "OVERLAYS")
	for _, row := range rows {
		fmt.Fprintf(out, "%-18s %-22s %s\n", row.ProviderID, row.Status, strings.Join(row.OverlayIDs, ","))
	}
	return 0
}

type catalogListRow struct {
	ID                  string                            `json:"id"`
	DisplayName         string                            `json:"display_name"`
	Category            string                            `json:"category,omitempty"`
	OpenAPIAvailability catalogpkg.SpecAvailability       `json:"openapi_availability"`
	MachineAvailability catalogpkg.SpecAvailability       `json:"machine_availability"`
	UserOpenAPINeed     catalogpkg.UserOpenAPINeed        `json:"user_openapi_need"`
	AuthStatus          catalogpkg.AuthCompletenessStatus `json:"auth_status"`
}

func writeCatalogInspect(out io.Writer, resolved catalogpkg.ResolvedProvider) {
	provider := resolved.Provider
	fmt.Fprintf(out, "Provider: %s (%s)\n", provider.DisplayName, provider.ID)
	if provider.Category != "" {
		fmt.Fprintf(out, "Category: %s\n", provider.Category)
	}
	if provider.WorkflowRelevance != "" {
		fmt.Fprintf(out, "Relevance: %s\n", provider.WorkflowRelevance)
	}
	fmt.Fprintf(out, "OpenAPI availability: %s\n", provider.OfficialOpenAPIAvailability)
	fmt.Fprintf(out, "Machine spec availability: %s\n", provider.OfficialMachineSpecAvailability)
	fmt.Fprintf(out, "User OpenAPI need: %s\n", provider.UserOpenAPINeed)
	fmt.Fprintf(out, "Resolved OpenAPI: %s", resolved.OpenAPI.Source)
	writeResolvedReferenceDetails(out, resolved.OpenAPI)
	fmt.Fprintf(out, "Resolved security: %s", resolved.Security.Source)
	writeResolvedReferenceDetails(out, resolved.Security)
	fmt.Fprintf(out, "Auth status: %s\n", resolved.SecurityStatus)

	if len(provider.SpecReferences) > 0 {
		fmt.Fprintln(out, "Spec references:")
		for _, ref := range provider.SpecReferences {
			fmt.Fprintf(out, "- %s [%s] %s\n", ref.ID, ref.Kind, ref.URL)
			if ref.SourceNote != "" {
				fmt.Fprintf(out, "  note: %s\n", ref.SourceNote)
			}
		}
	}
	if len(resolved.CatalogSecurityOverlays) > 0 {
		fmt.Fprintln(out, "Security overlays:")
		for _, overlay := range resolved.CatalogSecurityOverlays {
			fmt.Fprintf(out, "- %s [%s]\n", overlay.ID, overlay.Status)
			if overlay.SourceNote != "" {
				fmt.Fprintf(out, "  note: %s\n", overlay.SourceNote)
			}
		}
	}
}

func writeResolvedReferenceDetails(out io.Writer, ref catalogpkg.ResolvedReference) {
	var details []string
	if ref.Value != "" {
		details = append(details, ref.Value)
	}
	if ref.SpecRefID != "" {
		details = append(details, "spec="+ref.SpecRefID)
	}
	if ref.OverlayID != "" {
		details = append(details, "overlay="+ref.OverlayID)
	}
	if len(details) > 0 {
		fmt.Fprintf(out, " (%s)", strings.Join(details, ", "))
	}
	fmt.Fprintln(out)
	if ref.SourceNote != "" {
		fmt.Fprintf(out, "  note: %s\n", ref.SourceNote)
	}
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
