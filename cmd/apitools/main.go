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
	"path/filepath"
	"sort"
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
	case "advisory":
		return runCatalogAdvisory(args[1:], out, errOut)
	case "check":
		return runCatalogCheck(args[1:], out, errOut)
	case "list":
		return runCatalogList(args[1:], out, errOut)
	case "specs":
		return runCatalogSpecs(args[1:], out, errOut)
	case "stats":
		return runCatalogStats(args[1:], out, errOut)
	case "refresh":
		return runCatalogRefresh(args[1:], out, errOut)
	case "refresh-report":
		return runCatalogRefreshReport(args[1:], out, errOut)
	case "inspect":
		return runCatalogInspect(args[1:], out, errOut)
	case "overlay-view":
		return runCatalogOverlayView(args[1:], out, errOut)
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
	fmt.Fprintln(out, "  advisory         render provider advisory summaries")
	fmt.Fprintln(out, "  check            run offline catalog quality checks")
	fmt.Fprintln(out, "  list             list built-in provider catalog metadata")
	fmt.Fprintln(out, "  specs            list refreshable built-in spec references")
	fmt.Fprintln(out, "  stats            summarize catalog protocol and artifact status")
	fmt.Fprintln(out, "  refresh          refresh one selected built-in spec reference")
	fmt.Fprintln(out, "  refresh-report   review saved refresh artifacts offline")
	fmt.Fprintln(out, "  inspect          inspect one provider and resolution status")
	fmt.Fprintln(out, "  overlay-view     inspect advisory security overlay effects")
	fmt.Fprintln(out, "  security-report  report auth/security status across providers")
}

func runCatalogAdvisory(args []string, out, errOut io.Writer) int {
	var provider string
	parseArgs := args
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		provider = args[0]
		parseArgs = args[1:]
	}
	fs := flag.NewFlagSet("apitools catalog advisory", flag.ContinueOnError)
	fs.SetOutput(errOut)
	cachePath := fs.String("cache", "", "Existing SQLite cache path used to show registered artifact paths")
	jsonOut := fs.Bool("json", false, "Write JSON output")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: apitools catalog advisory [provider] [--cache catalog-openapi-cache/cache.sqlite] [--json]")
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
	artifacts, closeCache, err := catalogSpecArtifactsFromExistingCache(*cachePath)
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	defer closeCache()
	report, err := catalogpkg.BuiltInProviderAdvisoryReport(catalogpkg.ProviderAdvisoryOptions{
		ProviderKey: provider,
		Artifacts:   artifacts,
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
	writeCatalogAdvisoryReport(out, report)
	return 0
}

func runCatalogCheck(args []string, out, errOut io.Writer) int {
	return runCatalogCheckWithReport(args, out, errOut, func(options catalogpkg.CatalogQualityOptions) catalogpkg.CatalogQualityReport {
		return catalogpkg.BuiltInCatalogQualityReport(options)
	})
}

func runCatalogCheckWithReport(args []string, out, errOut io.Writer, build func(catalogpkg.CatalogQualityOptions) catalogpkg.CatalogQualityReport) int {
	fs := flag.NewFlagSet("apitools catalog check", flag.ContinueOnError)
	fs.SetOutput(errOut)
	jsonOut := fs.Bool("json", false, "Write JSON output")
	asOf := fs.String("as-of", "", "Check freshness as of YYYY-MM-DD; defaults to today")
	staleDays := fs.Int("stale-days", catalogpkg.DefaultStaleVerificationDays, "Warn when spec verification is older than this many days")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: apitools catalog check [--as-of YYYY-MM-DD] [--stale-days 365] [--json]")
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
	options := catalogpkg.CatalogQualityOptions{StaleVerificationDays: *staleDays}
	if strings.TrimSpace(*asOf) != "" {
		parsed, err := time.Parse("2006-01-02", strings.TrimSpace(*asOf))
		if err != nil {
			fmt.Fprintf(errOut, "invalid --as-of %q: expected YYYY-MM-DD\n", *asOf)
			return 2
		}
		options.AsOf = parsed
	}
	report := build(options)
	if *jsonOut {
		if err := writeJSON(out, report); err != nil {
			fmt.Fprintln(errOut, err)
			return 1
		}
		return catalogCheckExitCode(report)
	}
	writeCatalogQualityReport(out, report)
	return catalogCheckExitCode(report)
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

func runCatalogSpecs(args []string, out, errOut io.Writer) int {
	fs := flag.NewFlagSet("apitools catalog specs", flag.ContinueOnError)
	fs.SetOutput(errOut)
	cachePath := fs.String("cache", "", "SQLite cache path used to show registered artifact paths")
	jsonOut := fs.Bool("json", false, "Write JSON output")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: apitools catalog specs [--cache catalog-openapi-cache/cache.sqlite] [--json]")
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
	artifacts, closeCache, err := catalogSpecArtifactsFromExistingCache(*cachePath)
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	defer closeCache()
	rows := catalogpkg.BuiltInRefreshableSpecReferences(artifacts)
	if *jsonOut {
		if err := writeJSON(out, rows); err != nil {
			fmt.Fprintln(errOut, err)
			return 1
		}
		return 0
	}
	fmt.Fprintf(out, "%-18s %-34s %-16s %-16s %-18s %-12s %s\n", "PROVIDER", "SPEC_REF", "KIND", "PROTOCOL", "SOURCE", "VERIFIED", "ARTIFACT")
	for _, row := range rows {
		fmt.Fprintf(out, "%-18s %-34s %-16s %-16s %-18s %-12s %s\n", row.ProviderID, row.SpecRefID, row.Kind, protocolLabel(row.Protocol, row.ProtocolVersion), row.SourceAuthority, row.VerifiedAt, row.RegisteredArtifactPath)
	}
	return 0
}

func runCatalogStats(args []string, out, errOut io.Writer) int {
	fs := flag.NewFlagSet("apitools catalog stats", flag.ContinueOnError)
	fs.SetOutput(errOut)
	cacheDir := fs.String("cache-dir", "catalog-openapi-cache", "Catalog cache directory containing saved review artifacts")
	cachePath := fs.String("cache", "catalog-openapi-cache/cache.sqlite", "Existing SQLite cache path used to join registered artifact paths")
	asOf := fs.String("as-of", "", "Check refresh verification freshness as of YYYY-MM-DD; defaults to today")
	staleDays := fs.Int("stale-days", catalogpkg.DefaultStaleVerificationDays, "Warn when spec verification is older than this many days")
	maxBytes := fs.Int64("max-bytes", apitools.CatalogRefreshReviewDefaultMaxBytes, "Maximum bytes to read from each saved artifact")
	jsonOut := fs.Bool("json", false, "Write JSON output")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: apitools catalog stats [--cache-dir catalog-openapi-cache] [--cache catalog-openapi-cache/cache.sqlite] [--as-of YYYY-MM-DD] [--stale-days 365] [--max-bytes N] [--json]")
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
	opts := apitools.CatalogSpecRefreshReviewOptions{
		CacheDir:              *cacheDir,
		StaleVerificationDays: *staleDays,
		MaxBytes:              *maxBytes,
	}
	if strings.TrimSpace(*asOf) != "" {
		parsed, err := time.Parse("2006-01-02", strings.TrimSpace(*asOf))
		if err != nil {
			fmt.Fprintf(errOut, "invalid --as-of %q: expected YYYY-MM-DD\n", *asOf)
			return 2
		}
		opts.AsOf = parsed
	}
	artifacts, closeCache, err := catalogSpecArtifactsFromExistingCache(*cachePath)
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	defer closeCache()
	opts.Artifacts = artifacts
	refreshReport, err := apitools.BuiltInCatalogSpecRefreshReviewReport(opts)
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	report := buildCatalogStatsReport(catalogpkg.BuiltInProviders(), artifacts, refreshReport)
	if *jsonOut {
		if err := writeJSON(out, report); err != nil {
			fmt.Fprintln(errOut, err)
			return 1
		}
		return 0
	}
	writeCatalogStatsReport(out, report)
	return 0
}

type catalogRefreshFunc func(context.Context, []catalogpkg.RefreshableSpecReference, apitools.CatalogSpecRefreshOptions) (apitools.CatalogSpecRefreshReport, error)

func runCatalogRefresh(args []string, out, errOut io.Writer) int {
	return runCatalogRefreshWithClient(args, out, errOut, func(client *apitools.Client) catalogRefreshFunc {
		return client.RefreshCatalogSpecReferences
	})
}

func runCatalogRefreshReport(args []string, out, errOut io.Writer) int {
	fs := flag.NewFlagSet("apitools catalog refresh-report", flag.ContinueOnError)
	fs.SetOutput(errOut)
	cacheDir := fs.String("cache-dir", "catalog-openapi-cache", "Catalog cache directory containing saved review artifacts")
	cachePath := fs.String("cache", "catalog-openapi-cache/cache.sqlite", "Existing SQLite cache path used to join registered artifact paths")
	asOf := fs.String("as-of", "", "Check refresh verification freshness as of YYYY-MM-DD; defaults to today")
	staleDays := fs.Int("stale-days", catalogpkg.DefaultStaleVerificationDays, "Warn when spec verification is older than this many days")
	maxBytes := fs.Int64("max-bytes", apitools.CatalogRefreshReviewDefaultMaxBytes, "Maximum bytes to read from each saved artifact")
	jsonOut := fs.Bool("json", false, "Write JSON output")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: apitools catalog refresh-report [--cache-dir catalog-openapi-cache] [--cache catalog-openapi-cache/cache.sqlite] [--as-of YYYY-MM-DD] [--stale-days 365] [--max-bytes N] [--json]")
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
	options := apitools.CatalogSpecRefreshReviewOptions{
		CacheDir:              *cacheDir,
		StaleVerificationDays: *staleDays,
		MaxBytes:              *maxBytes,
	}
	if strings.TrimSpace(*asOf) != "" {
		parsed, err := time.Parse("2006-01-02", strings.TrimSpace(*asOf))
		if err != nil {
			fmt.Fprintf(errOut, "invalid --as-of %q: expected YYYY-MM-DD\n", *asOf)
			return 2
		}
		options.AsOf = parsed
	}
	artifacts, closeCache, err := catalogSpecArtifactsFromExistingCache(*cachePath)
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	defer closeCache()
	options.Artifacts = artifacts
	report, err := apitools.BuiltInCatalogSpecRefreshReviewReport(options)
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
	writeCatalogRefreshReviewReport(out, report)
	return 0
}

func runCatalogRefreshWithClient(args []string, out, errOut io.Writer, refresherFor func(*apitools.Client) catalogRefreshFunc) int {
	fs := flag.NewFlagSet("apitools catalog refresh", flag.ContinueOnError)
	fs.SetOutput(errOut)
	providerKey := fs.String("provider", "", "Provider ID, display name, or alias to refresh")
	specRefID := fs.String("spec", "", "Spec reference ID to refresh; required when provider has multiple refreshable specs")
	cacheDir := fs.String("cache-dir", "catalog-openapi-cache", "Catalog cache directory for saved review artifacts")
	cachePath := fs.String("cache", "catalog-openapi-cache/cache.sqlite", "SQLite cache path used to register saved artifact paths")
	timeout := fs.Duration("timeout", apitools.DefaultTimeout, "Refresh download timeout")
	maxBytes := fs.Int64("max-bytes", apitools.DefaultMaxBytes, "Maximum bytes to download")
	allowUnsafeHosts := fs.Bool("allow-unsafe-hosts", false, "Allow localhost/private hosts; intended only for local tests")
	jsonOut := fs.Bool("json", false, "Write JSON output")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: apitools catalog refresh --provider <provider> [--spec <spec-ref>] [--cache-dir catalog-openapi-cache] [--cache catalog-openapi-cache/cache.sqlite] [--json]")
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
	if _, err := selectRefreshableSpecRows(*providerKey, *specRefID, nil); err != nil {
		fmt.Fprintln(errOut, err)
		return 2
	}
	artifacts, closeCache, err := catalogSpecArtifactsFromExistingCache(*cachePath)
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	defer closeCache()
	rows, err := selectRefreshableSpecRows(*providerKey, *specRefID, artifacts)
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 2
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	client := &apitools.Client{
		Timeout:          *timeout,
		MaxBytes:         *maxBytes,
		AllowUnsafeHosts: *allowUnsafeHosts,
	}
	report, err := refresherFor(client)(ctx, rows, apitools.CatalogSpecRefreshOptions{CacheDir: *cacheDir})
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	if err := registerCatalogRefreshResults(ctx, *cachePath, *cacheDir, report); err != nil {
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
	writeCatalogRefreshReport(out, report)
	return 0
}

func selectRefreshableSpecRows(providerKey, specRefID string, artifacts []catalogpkg.CatalogSpecArtifact) ([]catalogpkg.RefreshableSpecReference, error) {
	providerKey = strings.TrimSpace(providerKey)
	if providerKey == "" {
		return nil, fmt.Errorf("missing --provider")
	}
	provider, ok := catalogpkg.FindBuiltInProvider(providerKey)
	if !ok {
		return nil, fmt.Errorf("unknown provider %q", providerKey)
	}
	specRefID = strings.TrimSpace(specRefID)
	var rows []catalogpkg.RefreshableSpecReference
	for _, row := range catalogpkg.BuiltInRefreshableSpecReferences(artifacts) {
		if row.ProviderID != provider.ID {
			continue
		}
		if specRefID != "" && row.SpecRefID != specRefID {
			continue
		}
		rows = append(rows, row)
	}
	if len(rows) == 0 {
		if specRefID != "" {
			return nil, fmt.Errorf("provider %q has no refreshable spec %q", provider.ID, specRefID)
		}
		return nil, fmt.Errorf("provider %q has no refreshable spec references", provider.ID)
	}
	if specRefID == "" && len(rows) > 1 {
		var ids []string
		for _, row := range rows {
			ids = append(ids, row.SpecRefID)
		}
		return nil, fmt.Errorf("provider %q has multiple refreshable specs; pass --spec (%s)", provider.ID, strings.Join(ids, ", "))
	}
	return rows, nil
}

func registerCatalogRefreshResults(ctx context.Context, cachePath, cacheDir string, report apitools.CatalogSpecRefreshReport) error {
	cachePath = strings.TrimSpace(cachePath)
	if cachePath == "" {
		return fmt.Errorf("cache path is required")
	}
	cache, err := sqlitecache.Open(cachePath)
	if err != nil {
		return err
	}
	defer cache.Close()
	for _, result := range report.Results {
		contentPath := refreshContentPathForCache(cachePath, cacheDir, result)
		if err := cache.StoreSpec(ctx, apitools.CachedSpec{
			OriginalURL: result.URL,
			FinalURL:    result.FinalURL,
			ContentPath: contentPath,
			Metadata:    result.Metadata,
		}); err != nil {
			return err
		}
		if err := cache.StoreCatalogArtifact(ctx, sqlitecache.CatalogArtifact{
			ProviderID: result.ProviderID,
			ArtifactID: result.SpecRefID,
			Kind:       string(result.Kind),
			Path:       contentPath,
			SourceURL:  result.URL,
			Metadata: map[string]string{
				"official":          "true",
				"kind":              string(result.Kind),
				"validation_status": result.ValidationStatus,
				"sha256":            result.SHA256,
			},
		}); err != nil {
			return err
		}
	}
	return nil
}

func refreshContentPathForCache(cachePath, cacheDir string, result apitools.CatalogSpecRefreshResult) string {
	cacheBase := filepath.Dir(cachePath)
	fullPath := result.SavedPath
	if strings.TrimSpace(fullPath) == "" {
		fullPath = filepath.Join(cacheDir, filepath.FromSlash(result.ArtifactPath))
	}
	rel, err := filepath.Rel(cacheBase, fullPath)
	if err == nil && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != ".." && !filepath.IsAbs(rel) {
		return filepath.ToSlash(rel)
	}
	return fullPath
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

func runCatalogOverlayView(args []string, out, errOut io.Writer) int {
	var provider string
	parseArgs := args
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		provider = args[0]
		parseArgs = args[1:]
	}
	fs := flag.NewFlagSet("apitools catalog overlay-view", flag.ContinueOnError)
	fs.SetOutput(errOut)
	jsonOut := fs.Bool("json", false, "Write JSON output")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: apitools catalog overlay-view <provider> [--json]")
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
	view, err := catalogpkg.BuiltInSecurityInspectionView(provider)
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	if *jsonOut {
		if err := writeJSON(out, view); err != nil {
			fmt.Fprintln(errOut, err)
			return 1
		}
		return 0
	}
	writeCatalogOverlayView(out, view)
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

type catalogStatsReport struct {
	ProviderCount    int                          `json:"provider_count"`
	Protocols        []catalogStatsProtocol       `json:"protocols"`
	ArtifactRegistry catalogStatsArtifactRegistry `json:"artifact_registry"`
	Refresh          catalogStatsRefreshEvidence  `json:"refresh"`
}

type catalogStatsProtocol struct {
	Protocol    catalogpkg.SpecProtocol `json:"protocol"`
	DisplayName string                  `json:"display_name"`
	Count       int                     `json:"count"`
	ProviderIDs []string                `json:"provider_ids,omitempty"`
}

type catalogStatsRefreshEvidence struct {
	RefreshableSpecCount       int                      `json:"refreshable_spec_count"`
	RegisteredArtifactCount    int                      `json:"registered_artifact_count"`
	ExistingArtifactCount      int                      `json:"existing_artifact_count"`
	MissingRegistrationCount   int                      `json:"missing_registration_count"`
	MissingRegisteredFileCount int                      `json:"missing_registered_file_count"`
	InvalidArtifactCount       int                      `json:"invalid_artifact_count"`
	StaleVerificationCount     int                      `json:"stale_verification_count"`
	TotalBytes                 int64                    `json:"total_bytes"`
	ValidationStatuses         []catalogStatsValidation `json:"validation_statuses,omitempty"`
	Protocols                  []catalogStatsProtocol   `json:"protocols,omitempty"`
}

type catalogStatsArtifactRegistry struct {
	ArtifactCount int                        `json:"artifact_count"`
	Kinds         []catalogStatsArtifactKind `json:"kinds,omitempty"`
}

type catalogStatsArtifactKind struct {
	Kind        string   `json:"kind"`
	Count       int      `json:"count"`
	ArtifactIDs []string `json:"artifact_ids,omitempty"`
}

type catalogStatsValidation struct {
	Status     string   `json:"status"`
	Count      int      `json:"count"`
	SpecRefIDs []string `json:"spec_ref_ids,omitempty"`
}

func buildCatalogStatsReport(providers []catalogpkg.Provider, artifacts []catalogpkg.CatalogSpecArtifact, refreshReport apitools.CatalogSpecRefreshReviewReport) catalogStatsReport {
	return catalogStatsReport{
		ProviderCount:    len(providers),
		Protocols:        catalogProviderProtocolStats(providers),
		ArtifactRegistry: catalogArtifactRegistryStats(artifacts),
		Refresh:          catalogRefreshEvidenceStats(refreshReport),
	}
}

func catalogProviderProtocolStats(providers []catalogpkg.Provider) []catalogStatsProtocol {
	providerIDsByProtocol := map[catalogpkg.SpecProtocol][]string{}
	for _, provider := range providers {
		protocol := primaryProviderProtocol(provider).Protocol
		providerIDsByProtocol[protocol] = append(providerIDsByProtocol[protocol], provider.ID)
	}
	out := make([]catalogStatsProtocol, 0, len(catalogProtocolOrder()))
	for _, protocol := range catalogProtocolOrder() {
		providerIDs := providerIDsByProtocol[protocol]
		out = append(out, catalogStatsProtocol{
			Protocol:    protocol,
			DisplayName: catalogProtocolDisplayName(protocol),
			Count:       len(providerIDs),
			ProviderIDs: append([]string(nil), providerIDs...),
		})
	}
	return out
}

func primaryProviderProtocol(provider catalogpkg.Provider) catalogpkg.SpecProtocolClassification {
	for _, ref := range provider.SpecReferences {
		if ref.Kind != catalogpkg.SpecKindHumanDocs {
			return ref.ProtocolClassification()
		}
	}
	for _, ref := range provider.SpecReferences {
		if ref.Kind == catalogpkg.SpecKindHumanDocs {
			return ref.ProtocolClassification()
		}
	}
	return catalogpkg.SpecProtocolClassification{Protocol: catalogpkg.SpecProtocolUnknown}
}

func catalogRefreshEvidenceStats(report apitools.CatalogSpecRefreshReviewReport) catalogStatsRefreshEvidence {
	statusIDs := map[string][]string{}
	providerIDsByProtocol := map[catalogpkg.SpecProtocol][]string{}
	stats := catalogStatsRefreshEvidence{RefreshableSpecCount: len(report.Results)}
	for _, result := range report.Results {
		key := result.ProviderID + "/" + result.SpecRefID
		if strings.TrimSpace(result.RegisteredArtifactPath) != "" {
			stats.RegisteredArtifactCount++
		}
		if result.Exists {
			stats.ExistingArtifactCount++
		}
		if result.ValidationStatus == apitools.CatalogRefreshMissingRegistration {
			stats.MissingRegistrationCount++
		}
		if result.ValidationStatus == apitools.CatalogRefreshMissingFile {
			stats.MissingRegisteredFileCount++
		}
		if result.ValidationStatus == apitools.CatalogRefreshInvalid {
			stats.InvalidArtifactCount++
		}
		if result.VerificationStale {
			stats.StaleVerificationCount++
		}
		stats.TotalBytes += result.Bytes
		statusIDs[result.ValidationStatus] = append(statusIDs[result.ValidationStatus], key)
		providerIDsByProtocol[result.Protocol] = append(providerIDsByProtocol[result.Protocol], key)
	}
	for _, status := range catalogRefreshValidationOrder(statusIDs) {
		ids := statusIDs[status]
		stats.ValidationStatuses = append(stats.ValidationStatuses, catalogStatsValidation{
			Status:     status,
			Count:      len(ids),
			SpecRefIDs: append([]string(nil), ids...),
		})
	}
	for _, protocol := range catalogProtocolOrder() {
		ids := providerIDsByProtocol[protocol]
		if len(ids) == 0 {
			continue
		}
		stats.Protocols = append(stats.Protocols, catalogStatsProtocol{
			Protocol:    protocol,
			DisplayName: catalogProtocolDisplayName(protocol),
			Count:       len(ids),
			ProviderIDs: append([]string(nil), ids...),
		})
	}
	return stats
}

func catalogArtifactRegistryStats(artifacts []catalogpkg.CatalogSpecArtifact) catalogStatsArtifactRegistry {
	idsByKind := map[string][]string{}
	for _, artifact := range artifacts {
		kind := strings.TrimSpace(artifact.Kind)
		if kind == "" {
			kind = "unknown"
		}
		idsByKind[kind] = append(idsByKind[kind], artifact.ProviderID+"/"+artifact.SpecRefID)
	}
	kinds := make([]string, 0, len(idsByKind))
	for kind := range idsByKind {
		kinds = append(kinds, kind)
	}
	sort.Strings(kinds)
	stats := catalogStatsArtifactRegistry{ArtifactCount: len(artifacts)}
	for _, kind := range kinds {
		ids := idsByKind[kind]
		sort.Strings(ids)
		stats.Kinds = append(stats.Kinds, catalogStatsArtifactKind{
			Kind:        kind,
			Count:       len(ids),
			ArtifactIDs: append([]string(nil), ids...),
		})
	}
	return stats
}

func catalogRefreshValidationOrder(statusIDs map[string][]string) []string {
	preferred := []string{
		apitools.CatalogRefreshValidOpenAPI,
		apitools.CatalogRefreshValidSwagger,
		apitools.CatalogRefreshParseableOpenAPIInvalid,
		apitools.CatalogRefreshParseableSwaggerInvalid,
		apitools.CatalogRefreshValidStructured,
		apitools.CatalogRefreshSkippedValidation,
		apitools.CatalogRefreshMissingRegistration,
		apitools.CatalogRefreshMissingFile,
		apitools.CatalogRefreshInvalid,
	}
	seen := map[string]struct{}{}
	var out []string
	for _, status := range preferred {
		if _, ok := statusIDs[status]; ok {
			out = append(out, status)
			seen[status] = struct{}{}
		}
	}
	var extra []string
	for status := range statusIDs {
		if _, ok := seen[status]; !ok {
			extra = append(extra, status)
		}
	}
	sort.Strings(extra)
	return append(out, extra...)
}

func catalogProtocolOrder() []catalogpkg.SpecProtocol {
	return []catalogpkg.SpecProtocol{
		catalogpkg.SpecProtocolOpenAPI,
		catalogpkg.SpecProtocolSwagger,
		catalogpkg.SpecProtocolSmithy,
		catalogpkg.SpecProtocolGoogleDiscovery,
		catalogpkg.SpecProtocolDropboxStone,
		catalogpkg.SpecProtocolOpenAPIIndex,
		catalogpkg.SpecProtocolHumanDocs,
		catalogpkg.SpecProtocolUnknown,
	}
}

func catalogProtocolDisplayName(protocol catalogpkg.SpecProtocol) string {
	switch protocol {
	case catalogpkg.SpecProtocolOpenAPI:
		return "OpenAPI"
	case catalogpkg.SpecProtocolSwagger:
		return "Swagger"
	case catalogpkg.SpecProtocolSmithy:
		return "Smithy"
	case catalogpkg.SpecProtocolGoogleDiscovery:
		return "Google Discovery"
	case catalogpkg.SpecProtocolDropboxStone:
		return "Dropbox Stone"
	case catalogpkg.SpecProtocolOpenAPIIndex:
		return "OpenAPI index"
	case catalogpkg.SpecProtocolHumanDocs:
		return "Human docs"
	default:
		return "Unknown"
	}
}

func writeCatalogStatsReport(out io.Writer, report catalogStatsReport) {
	fmt.Fprintf(out, "Provider protocols: %d provider(s)\n", report.ProviderCount)
	fmt.Fprintf(out, "%-18s %-7s %s\n", "PROTOCOL", "COUNT", "PROVIDERS")
	for _, bucket := range report.Protocols {
		fmt.Fprintf(out, "%-18s %-7d %s\n", bucket.DisplayName, bucket.Count, strings.Join(bucket.ProviderIDs, ", "))
	}
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Artifact registry:")
	fmt.Fprintf(out, "- artifacts: %d\n", report.ArtifactRegistry.ArtifactCount)
	if len(report.ArtifactRegistry.Kinds) > 0 {
		fmt.Fprintf(out, "%-18s %-7s %s\n", "KIND", "COUNT", "ARTIFACTS")
		for _, bucket := range report.ArtifactRegistry.Kinds {
			fmt.Fprintf(out, "%-18s %-7d %s\n", bucket.Kind, bucket.Count, strings.Join(bucket.ArtifactIDs, ", "))
		}
	}
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Refresh artifacts:")
	fmt.Fprintf(out, "- refreshable specs: %d\n", report.Refresh.RefreshableSpecCount)
	fmt.Fprintf(out, "- registered artifacts: %d\n", report.Refresh.RegisteredArtifactCount)
	fmt.Fprintf(out, "- existing artifacts: %d\n", report.Refresh.ExistingArtifactCount)
	fmt.Fprintf(out, "- missing registrations: %d\n", report.Refresh.MissingRegistrationCount)
	fmt.Fprintf(out, "- missing registered files: %d\n", report.Refresh.MissingRegisteredFileCount)
	fmt.Fprintf(out, "- invalid artifacts: %d\n", report.Refresh.InvalidArtifactCount)
	fmt.Fprintf(out, "- stale verifications: %d\n", report.Refresh.StaleVerificationCount)
	fmt.Fprintf(out, "- total bytes: %d\n", report.Refresh.TotalBytes)
	if len(report.Refresh.ValidationStatuses) > 0 {
		fmt.Fprintln(out, "Validation statuses:")
		fmt.Fprintf(out, "%-28s %-7s %s\n", "STATUS", "COUNT", "SPEC_REFS")
		for _, bucket := range report.Refresh.ValidationStatuses {
			fmt.Fprintf(out, "%-28s %-7d %s\n", bucket.Status, bucket.Count, strings.Join(bucket.SpecRefIDs, ", "))
		}
	}
}

func writeCatalogAdvisoryReport(out io.Writer, report catalogpkg.ProviderAdvisoryReport) {
	if len(report.Providers) == 0 {
		fmt.Fprintln(out, "No provider advisory rows.")
		return
	}
	for i, provider := range report.Providers {
		if i > 0 {
			fmt.Fprintln(out)
		}
		fmt.Fprintf(out, "Provider: %s (%s)\n", provider.DisplayName, provider.ProviderID)
		if provider.Category != "" {
			fmt.Fprintf(out, "Category: %s\n", provider.Category)
		}
		fmt.Fprintf(out, "OpenAPI availability: %s\n", provider.OpenAPIAvailability)
		fmt.Fprintf(out, "Machine spec availability: %s\n", provider.MachineSpecAvailability)
		fmt.Fprintf(out, "User OpenAPI need: %s\n", provider.UserOpenAPINeed)
		fmt.Fprintf(out, "Auth status: %s\n", provider.AuthStatus)
		fmt.Fprintf(out, "Resolved OpenAPI: %s", provider.ResolvedOpenAPI.Source)
		writeResolvedReferenceDetails(out, provider.ResolvedOpenAPI)
		fmt.Fprintf(out, "Resolved security: %s", provider.ResolvedSecurity.Source)
		writeResolvedReferenceDetails(out, provider.ResolvedSecurity)
		if len(provider.SpecReferences) > 0 {
			fmt.Fprintln(out, "Spec references:")
			for _, ref := range provider.SpecReferences {
				fmt.Fprintf(out, "- %s [%s/%s] %s\n", ref.ID, ref.Kind, protocolLabel(ref.Protocol, ref.ProtocolVersion), ref.URL)
				if ref.RegisteredArtifactPath != "" {
					fmt.Fprintf(out, "  artifact: %s\n", ref.RegisteredArtifactPath)
				}
				if ref.SourceNote != "" {
					fmt.Fprintf(out, "  note: %s\n", ref.SourceNote)
				}
			}
		}
		if len(provider.SecurityOverlayIDs) > 0 {
			fmt.Fprintf(out, "Security overlays: %s\n", strings.Join(provider.SecurityOverlayIDs, ", "))
		}
		if len(provider.EndpointOverlays) > 0 {
			fmt.Fprintln(out, "Endpoint overlays:")
			for _, overlay := range provider.EndpointOverlays {
				fmt.Fprintf(out, "- %s [%s] %s\n", overlay.ArtifactID, overlay.Kind, overlay.Path)
				if overlay.OverlayPath != "" && overlay.OverlayPath != overlay.Path {
					fmt.Fprintf(out, "  overlay: %s\n", overlay.OverlayPath)
				}
				if overlay.BuilderPath != "" {
					fmt.Fprintf(out, "  builder: %s\n", overlay.BuilderPath)
				}
				if metadata := formatMetadata(overlay.Metadata); metadata != "" {
					fmt.Fprintf(out, "  metadata: %s\n", metadata)
				}
			}
		}
		if len(provider.ManualFollowUps) > 0 {
			fmt.Fprintln(out, "Manual follow-ups:")
			for _, followUp := range provider.ManualFollowUps {
				fmt.Fprintf(out, "- %s\n", followUp)
			}
		}
	}
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
			protocol := ref.ProtocolClassification()
			fmt.Fprintf(out, "- %s [%s/%s] %s\n", ref.ID, ref.Kind, protocolLabel(protocol.Protocol, protocol.Version), ref.URL)
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

func writeCatalogOverlayView(out io.Writer, view catalogpkg.SecurityInspectionView) {
	fmt.Fprintf(out, "Provider: %s (%s)\n", view.DisplayName, view.ProviderID)
	fmt.Fprintf(out, "Auth status: %s\n", view.Status)
	if view.Classification != nil {
		fmt.Fprintf(out, "Classification: %s", view.Classification.Provenance)
		if view.Classification.SpecRefID != "" {
			fmt.Fprintf(out, " spec=%s", view.Classification.SpecRefID)
		}
		fmt.Fprintf(out, " status=%s\n", view.Classification.Status)
	}
	if len(view.SecuritySchemes) > 0 {
		fmt.Fprintln(out, "Effective security schemes:")
		for _, scheme := range view.SecuritySchemes {
			fmt.Fprintf(out, "- %s [%s]", scheme.Scheme.Name, scheme.Provenance)
			if scheme.OverlayID != "" {
				fmt.Fprintf(out, " overlay=%s", scheme.OverlayID)
			}
			if scheme.SpecRefID != "" {
				fmt.Fprintf(out, " spec=%s", scheme.SpecRefID)
			}
			fmt.Fprintln(out)
		}
	}
	if len(view.RootSecurity) > 0 {
		fmt.Fprintln(out, "Root security alternatives:")
		for _, requirement := range view.RootSecurity {
			fmt.Fprintf(out, "- %s [%s]", requirement.Requirement.Scheme, requirement.Provenance)
			if len(requirement.Requirement.Scopes) > 0 {
				fmt.Fprintf(out, " scopes=%s", strings.Join(requirement.Requirement.Scopes, ","))
			}
			if requirement.OverlayID != "" {
				fmt.Fprintf(out, " overlay=%s", requirement.OverlayID)
			}
			fmt.Fprintln(out)
		}
	}
	if len(view.RootSecuritySets) > 0 {
		fmt.Fprintln(out, "Root security requirement sets:")
		for _, set := range view.RootSecuritySets {
			var schemes []string
			for _, requirement := range set.Requirements {
				schemes = append(schemes, requirement.Requirement.Scheme)
			}
			fmt.Fprintf(out, "- %s [%s]", strings.Join(schemes, " + "), set.Provenance)
			if set.OverlayID != "" {
				fmt.Fprintf(out, " overlay=%s", set.OverlayID)
			}
			fmt.Fprintln(out)
		}
	}
	if len(view.OperationSecurity) > 0 {
		fmt.Fprintln(out, "Operation security:")
		for _, operation := range view.OperationSecurity {
			fmt.Fprintf(out, "- %s [%s]", operationMatchLabel(operation.Match), operation.Provenance)
			if operation.OverlayID != "" {
				fmt.Fprintf(out, " overlay=%s", operation.OverlayID)
			}
			fmt.Fprintln(out)
			for _, requirement := range operation.Security {
				fmt.Fprintf(out, "  - %s [%s]\n", requirement.Requirement.Scheme, requirement.Provenance)
			}
			for _, set := range operation.SecuritySets {
				var schemes []string
				for _, requirement := range set.Requirements {
					schemes = append(schemes, requirement.Requirement.Scheme)
				}
				fmt.Fprintf(out, "  - %s [%s]\n", strings.Join(schemes, " + "), set.Provenance)
			}
		}
	}
	if len(view.Conflicts) > 0 {
		fmt.Fprintln(out, "Conflicts:")
		for _, conflict := range view.Conflicts {
			fmt.Fprintf(out, "- %s", conflict.Type)
			if conflict.Scheme != "" {
				fmt.Fprintf(out, " scheme=%s", conflict.Scheme)
			}
			if conflict.OverlayID != "" {
				fmt.Fprintf(out, " overlay=%s", conflict.OverlayID)
			}
			if label := operationMatchLabel(conflict.Match); label != "" {
				fmt.Fprintf(out, " match=%s", label)
			}
			fmt.Fprintf(out, ": %s\n", conflict.Message)
		}
	}
}

func writeCatalogQualityReport(out io.Writer, report catalogpkg.CatalogQualityReport) {
	fmt.Fprintf(out, "Catalog quality: %d error(s), %d warning(s)\n", report.ErrorCount(), report.WarningCount())
	if len(report.Findings) == 0 {
		fmt.Fprintln(out, "No catalog quality findings.")
		return
	}
	fmt.Fprintf(out, "%-8s %-34s %-18s %-28s %s\n", "SEVERITY", "CODE", "PROVIDER", "FIELD", "MESSAGE")
	for _, finding := range report.Findings {
		subject := finding.ProviderID
		if subject == "" {
			subject = finding.CandidateID
		}
		if subject == "" {
			subject = finding.OverlayID
		}
		fmt.Fprintf(out, "%-8s %-34s %-18s %-28s %s\n", finding.Severity, finding.Code, subject, finding.Field, finding.Message)
	}
}

func writeCatalogRefreshReport(out io.Writer, report apitools.CatalogSpecRefreshReport) {
	if len(report.Results) == 0 {
		fmt.Fprintln(out, "No catalog specs refreshed.")
		return
	}
	fmt.Fprintf(out, "%-18s %-34s %-16s %-20s %-12s %s\n", "PROVIDER", "SPEC_REF", "PROTOCOL", "VALIDATION", "BYTES", "ARTIFACT")
	for _, result := range report.Results {
		fmt.Fprintf(out, "%-18s %-34s %-16s %-20s %-12d %s\n", result.ProviderID, result.SpecRefID, protocolLabel(result.Protocol, result.ProtocolVersion), result.ValidationStatus, result.Bytes, result.ArtifactPath)
	}
	fmt.Fprintln(out, "Manual follow-ups:")
	for _, result := range report.Results {
		for _, followUp := range result.ManualFollowUps {
			fmt.Fprintf(out, "- %s/%s: %s\n", result.ProviderID, result.SpecRefID, followUp)
		}
	}
}

func writeCatalogRefreshReviewReport(out io.Writer, report apitools.CatalogSpecRefreshReviewReport) {
	if len(report.Results) == 0 {
		fmt.Fprintln(out, "No catalog refresh artifacts found.")
		return
	}
	fmt.Fprintf(out, "%-18s %-34s %-16s %-24s %-12s %-64s %s\n", "PROVIDER", "SPEC_REF", "PROTOCOL", "VALIDATION", "BYTES", "SHA256", "ARTIFACT")
	for _, result := range report.Results {
		artifact := result.RegisteredArtifactPath
		if artifact == "" {
			artifact = result.SavedPath
		}
		fmt.Fprintf(out, "%-18s %-34s %-16s %-24s %-12d %-64s %s\n", result.ProviderID, result.SpecRefID, protocolLabel(result.Protocol, result.ProtocolVersion), result.ValidationStatus, result.Bytes, result.SHA256, artifact)
	}
	fmt.Fprintln(out, "Manual follow-ups:")
	for _, result := range report.Results {
		for _, followUp := range result.ManualFollowUps {
			fmt.Fprintf(out, "- %s/%s: %s\n", result.ProviderID, result.SpecRefID, followUp)
		}
		if result.ValidationError != "" {
			fmt.Fprintf(out, "- %s/%s: validation error: %s\n", result.ProviderID, result.SpecRefID, result.ValidationError)
		}
	}
}

func catalogCheckExitCode(report catalogpkg.CatalogQualityReport) int {
	if report.HasErrors() {
		return 1
	}
	return 0
}

func operationMatchLabel(match catalogpkg.OperationMatch) string {
	var parts []string
	if match.OperationID != "" {
		parts = append(parts, "operation_id="+match.OperationID)
	}
	if match.Method != "" || match.Path != "" {
		parts = append(parts, strings.ToUpper(match.Method)+" "+match.Path)
	}
	if len(match.Tags) > 0 {
		parts = append(parts, "tags="+strings.Join(match.Tags, ","))
	}
	return strings.Join(parts, " ")
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

func catalogSpecArtifactsFromCache(path string) ([]catalogpkg.CatalogSpecArtifact, func(), error) {
	if strings.TrimSpace(path) == "" {
		return nil, func() {}, nil
	}
	cache, err := sqlitecache.Open(path)
	if err != nil {
		return nil, nil, err
	}
	closeCache := func() { _ = cache.Close() }
	artifacts, err := cache.ListCatalogArtifacts(context.Background())
	if err != nil {
		closeCache()
		return nil, nil, err
	}
	rows := make([]catalogpkg.CatalogSpecArtifact, 0, len(artifacts))
	for _, artifact := range artifacts {
		rows = append(rows, catalogpkg.CatalogSpecArtifact{
			ProviderID:  artifact.ProviderID,
			SpecRefID:   artifact.ArtifactID,
			ArtifactID:  artifact.ArtifactID,
			Kind:        artifact.Kind,
			Path:        artifact.Path,
			OverlayPath: artifact.OverlayPath,
			BuilderPath: artifact.BuilderPath,
			Metadata:    artifact.Metadata,
		})
	}
	return rows, closeCache, nil
}

func formatMetadata(metadata map[string]string) string {
	if len(metadata) == 0 {
		return ""
	}
	keys := make([]string, 0, len(metadata))
	for key := range metadata {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	values := make([]string, 0, len(keys))
	for _, key := range keys {
		values = append(values, key+"="+metadata[key])
	}
	return strings.Join(values, ", ")
}

func catalogSpecArtifactsFromExistingCache(path string) ([]catalogpkg.CatalogSpecArtifact, func(), error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, func() {}, nil
	}
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil, func() {}, nil
		}
		return nil, nil, err
	}
	return catalogSpecArtifactsFromCache(path)
}

func singleLine(value string) string {
	fields := strings.Fields(value)
	text := strings.Join(fields, " ")
	if len(text) > 160 {
		return text[:157] + "..."
	}
	return text
}

func protocolLabel(protocol catalogpkg.SpecProtocol, version string) string {
	value := strings.TrimSpace(string(protocol))
	if value == "" {
		value = string(catalogpkg.SpecProtocolUnknown)
	}
	if version = strings.TrimSpace(version); version != "" {
		return value + " " + version
	}
	return value
}

func hasHelpFlag(args []string) bool {
	for _, arg := range args {
		if arg == "-h" || arg == "--help" {
			return true
		}
	}
	return false
}
