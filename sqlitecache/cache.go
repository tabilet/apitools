package sqlitecache

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/OpenUdon/apitools"
	"github.com/OpenUdon/apitools/internal/artifactio"
	_ "modernc.org/sqlite"
)

const (
	schemaVersion               = 3
	DefaultMaxSearchQueries     = 1_000
	DefaultMaxSearchResults     = 5_000
	DefaultMaxSpecDocuments     = 1_000
	DefaultMaxCatalogArtifacts  = 10_000
	DefaultMaxArtifactBytes     = artifactio.DefaultMaxBytes
	DefaultMaxSearchReportBytes = 4 * 1024 * 1024
	DefaultMaxMetadataBytes     = 1024 * 1024
)

// Options sets hard per-table retention and per-record byte bounds. Zero
// values use bounded defaults; cache growth cannot be made unbounded.
type Options struct {
	MaxSearchQueries     int
	MaxSearchResults     int
	MaxSpecDocuments     int
	MaxCatalogArtifacts  int
	MaxArtifactBytes     int64
	MaxSearchReportBytes int64
	MaxMetadataBytes     int64
}

type Cache struct {
	db      *sql.DB
	baseDir string
	options Options
}

// CatalogArtifact records a local catalog curation artifact in the SQLite
// cache. Paths point at local review assets and never contain credential data.
type CatalogArtifact struct {
	ProviderID  string            `json:"provider_id"`
	ArtifactID  string            `json:"artifact_id"`
	Kind        string            `json:"kind"`
	Path        string            `json:"path"`
	SourceURL   string            `json:"source_url,omitempty"`
	OverlayPath string            `json:"overlay_path,omitempty"`
	BuilderPath string            `json:"builder_path,omitempty"`
	SHA256      string            `json:"sha256"`
	Bytes       int64             `json:"bytes"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	StoredAt    time.Time         `json:"stored_at,omitempty"`
}

func Open(path string) (*Cache, error) {
	return OpenWithOptions(path, Options{})
}

// OpenWithOptions opens a bounded SQLite cache with explicit retention limits.
func OpenWithOptions(path string, options Options) (*Cache, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, fmt.Errorf("cache path is required")
	}
	if path != ":memory:" {
		absolute, err := filepath.Abs(path)
		if err != nil {
			return nil, err
		}
		parent, err := artifactio.EnsureRoot(filepath.Dir(absolute), 0o755)
		if err != nil {
			return nil, err
		}
		path = filepath.Join(parent, filepath.Base(absolute))
		if info, err := os.Lstat(path); err == nil {
			if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
				return nil, fmt.Errorf("cache path %q is not a regular file", path)
			}
		} else if !os.IsNotExist(err) {
			return nil, err
		}
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	cache := &Cache{db: db, baseDir: cacheBaseDir(path), options: normalizeOptions(options)}
	if err := cache.migrate(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}
	if _, err := cache.Prune(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}
	return cache, nil
}

func normalizeOptions(options Options) Options {
	if options.MaxSearchQueries <= 0 {
		options.MaxSearchQueries = DefaultMaxSearchQueries
	}
	if options.MaxSearchResults <= 0 {
		options.MaxSearchResults = DefaultMaxSearchResults
	}
	if options.MaxSpecDocuments <= 0 {
		options.MaxSpecDocuments = DefaultMaxSpecDocuments
	}
	if options.MaxCatalogArtifacts <= 0 {
		options.MaxCatalogArtifacts = DefaultMaxCatalogArtifacts
	}
	if options.MaxArtifactBytes <= 0 {
		options.MaxArtifactBytes = DefaultMaxArtifactBytes
	}
	if options.MaxSearchReportBytes <= 0 {
		options.MaxSearchReportBytes = DefaultMaxSearchReportBytes
	}
	if options.MaxMetadataBytes <= 0 {
		options.MaxMetadataBytes = DefaultMaxMetadataBytes
	}
	return options
}

func (c *Cache) Close() error {
	if c == nil || c.db == nil {
		return nil
	}
	return c.db.Close()
}

func (c *Cache) LoadSearch(ctx context.Context, key apitools.SearchCacheKey, maxAge time.Duration) (apitools.SearchReport, bool, error) {
	if c == nil || c.db == nil {
		return apitools.SearchReport{}, false, fmt.Errorf("cache is closed")
	}
	var reportJSON []byte
	var updatedAt int64
	err := c.db.QueryRowContext(ctx, `SELECT report_json, updated_at FROM search_queries WHERE key_hash = ?`, searchKeyHash(key)).Scan(&reportJSON, &updatedAt)
	if err == sql.ErrNoRows {
		return apitools.SearchReport{}, false, nil
	}
	if err != nil {
		return apitools.SearchReport{}, false, err
	}
	if expired(updatedAt, maxAge) {
		return apitools.SearchReport{}, false, nil
	}
	if int64(len(reportJSON)) > c.options.MaxSearchReportBytes {
		return apitools.SearchReport{}, false, fmt.Errorf("cached search report is %d bytes, over limit %d", len(reportJSON), c.options.MaxSearchReportBytes)
	}
	var report apitools.SearchReport
	if err := json.Unmarshal(reportJSON, &report); err != nil {
		return apitools.SearchReport{}, false, err
	}
	if err := c.touchSearch(ctx, searchKeyHash(key), report); err != nil {
		return apitools.SearchReport{}, false, err
	}
	return report, true, nil
}

func (c *Cache) StoreSearch(ctx context.Context, key apitools.SearchCacheKey, report apitools.SearchReport) error {
	if c == nil || c.db == nil {
		return fmt.Errorf("cache is closed")
	}
	reportJSON, err := json.Marshal(report)
	if err != nil {
		return err
	}
	if int64(len(reportJSON)) > c.options.MaxSearchReportBytes {
		return fmt.Errorf("search cache report is %d bytes, over limit %d", len(reportJSON), c.options.MaxSearchReportBytes)
	}
	if err := validateRecordBudget("search cache key", c.options.MaxMetadataBytes, key.Query, string(key.Source), key.ProviderURL); err != nil {
		return err
	}
	now := time.Now().UTC().Unix()
	accessed := time.Now().UTC().UnixNano()
	tx, err := c.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	resultIDs := make([]string, 0, len(report.Results))
	for _, result := range report.Results {
		categories, err := json.Marshal(result.Categories)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `
INSERT INTO search_results (
  id, source, provider, title, description, version, categories_json, spec_url,
  landing_url, score, validated, provenance, first_seen_at, last_seen_at, accessed_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
  source = excluded.source,
  provider = excluded.provider,
  title = excluded.title,
  description = excluded.description,
  version = excluded.version,
  categories_json = excluded.categories_json,
  spec_url = excluded.spec_url,
  landing_url = excluded.landing_url,
  score = excluded.score,
  validated = excluded.validated,
  provenance = excluded.provenance,
  last_seen_at = excluded.last_seen_at,
  accessed_at = excluded.accessed_at`,
			result.ID, result.Source, result.Provider, result.Title, result.Description,
			result.Version, string(categories), result.SpecURL, result.LandingURL, result.Score,
			boolInt(result.Validated), result.Provenance, now, now, accessed); err != nil {
			return err
		}
		resultIDs = append(resultIDs, result.ID)
	}
	resultIDsJSON, err := json.Marshal(resultIDs)
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `
INSERT INTO search_queries (
  key_hash, query, source, limit_value, public_probe, result_ids_json,
  report_json, first_seen_at, updated_at, accessed_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(key_hash) DO UPDATE SET
  query = excluded.query,
  source = excluded.source,
  limit_value = excluded.limit_value,
  public_probe = excluded.public_probe,
  result_ids_json = excluded.result_ids_json,
  report_json = excluded.report_json,
  updated_at = excluded.updated_at,
  accessed_at = excluded.accessed_at`,
		searchKeyHash(key), key.Query, string(key.Source), key.Limit, key.PublicProbe,
		string(resultIDsJSON), reportJSON, now, now, accessed); err != nil {
		return err
	}
	if _, err := c.pruneTx(ctx, tx); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	committed = true
	return nil
}

func (c *Cache) LoadSpec(ctx context.Context, rawURL string, maxAge time.Duration) (apitools.CachedSpec, bool, error) {
	if c == nil || c.db == nil {
		return apitools.CachedSpec{}, false, fmt.Errorf("cache is closed")
	}
	var spec apitools.CachedSpec
	var metadataJSON []byte
	var contentPath sql.NullString
	var content []byte
	var updatedAt int64
	err := c.db.QueryRowContext(ctx, `
SELECT original_url, final_url, sha256, bytes, metadata_json, content_path, content, updated_at
FROM spec_documents
WHERE url = ?`, strings.TrimSpace(rawURL)).Scan(&spec.OriginalURL, &spec.FinalURL, &spec.SHA256, &spec.Bytes, &metadataJSON, &contentPath, &content, &updatedAt)
	if err == sql.ErrNoRows {
		return apitools.CachedSpec{}, false, nil
	}
	if err != nil {
		return apitools.CachedSpec{}, false, err
	}
	if expired(updatedAt, maxAge) {
		return apitools.CachedSpec{}, false, nil
	}
	if int64(len(metadataJSON)) > c.options.MaxMetadataBytes {
		return apitools.CachedSpec{}, false, fmt.Errorf("%w: cached spec metadata is %d bytes, over limit %d", apitools.ErrCachedSpecIntegrity, len(metadataJSON), c.options.MaxMetadataBytes)
	}
	if err := validateRecordBudget("cached spec", c.options.MaxMetadataBytes, spec.OriginalURL, spec.FinalURL, spec.SHA256, contentPath.String); err != nil {
		return apitools.CachedSpec{}, false, fmt.Errorf("%w: %v", apitools.ErrCachedSpecIntegrity, err)
	}
	if err := json.Unmarshal(metadataJSON, &spec.Metadata); err != nil {
		return apitools.CachedSpec{}, false, err
	}
	if contentPath.Valid && strings.TrimSpace(contentPath.String) != "" {
		spec.ContentPath, err = cleanLocalPath(contentPath.String, "cached spec content")
		if err != nil {
			return apitools.CachedSpec{}, false, err
		}
		file, err := c.readArtifact(spec.ContentPath, spec.SHA256, spec.Bytes)
		if err != nil {
			return apitools.CachedSpec{}, false, fmt.Errorf("%w: read cached spec file %q: %v", apitools.ErrCachedSpecIntegrity, spec.ContentPath, err)
		}
		content = file.Data
	}
	if len(content) == 0 {
		return apitools.CachedSpec{}, false, fmt.Errorf("%w: cached spec %q has no content or content path", apitools.ErrCachedSpecIntegrity, strings.TrimSpace(rawURL))
	}
	if int64(len(content)) > c.options.MaxArtifactBytes {
		return apitools.CachedSpec{}, false, fmt.Errorf("%w: cached spec %q is over byte limit %d", apitools.ErrCachedSpecIntegrity, strings.TrimSpace(rawURL), c.options.MaxArtifactBytes)
	}
	spec.Content = append([]byte(nil), content...)
	if !validSHA256(spec.SHA256) {
		return apitools.CachedSpec{}, false, fmt.Errorf("%w: cached spec %q has missing or invalid SHA256", apitools.ErrCachedSpecIntegrity, strings.TrimSpace(rawURL))
	}
	digest := sha256.Sum256(spec.Content)
	if got := hex.EncodeToString(digest[:]); got != strings.ToLower(spec.SHA256) {
		return apitools.CachedSpec{}, false, fmt.Errorf("%w: cached spec SHA256 mismatch for %q", apitools.ErrCachedSpecIntegrity, strings.TrimSpace(rawURL))
	}
	if spec.Bytes <= 0 || spec.Bytes != int64(len(spec.Content)) {
		return apitools.CachedSpec{}, false, fmt.Errorf("%w: cached spec byte count mismatch for %q", apitools.ErrCachedSpecIntegrity, strings.TrimSpace(rawURL))
	}
	if _, err := c.db.ExecContext(ctx, `UPDATE spec_documents SET accessed_at = ? WHERE url = ?`, time.Now().UTC().UnixNano(), strings.TrimSpace(rawURL)); err != nil {
		return apitools.CachedSpec{}, false, err
	}
	spec.StoredAt = time.Unix(updatedAt, 0).UTC()
	return spec, true, nil
}

func (c *Cache) StoreSpec(ctx context.Context, spec apitools.CachedSpec) error {
	if c == nil || c.db == nil {
		return fmt.Errorf("cache is closed")
	}
	originalURL := strings.TrimSpace(spec.OriginalURL)
	finalURL := strings.TrimSpace(spec.FinalURL)
	if originalURL == "" {
		originalURL = finalURL
	}
	if finalURL == "" {
		finalURL = originalURL
	}
	if originalURL == "" {
		return fmt.Errorf("spec URL is required")
	}
	contentPath := ""
	if strings.TrimSpace(spec.ContentPath) != "" {
		var err error
		contentPath, err = cleanLocalPath(spec.ContentPath, "cached spec content")
		if err != nil {
			return err
		}
	}
	content := append([]byte(nil), spec.Content...)
	if contentPath != "" {
		file, err := c.readArtifact(contentPath, "", 0)
		if err != nil {
			return fmt.Errorf("read cached spec file %q: %w", contentPath, err)
		}
		if len(content) > 0 {
			provided := sha256.Sum256(content)
			if hex.EncodeToString(provided[:]) != file.SHA256 {
				return fmt.Errorf("cached spec inline content differs from content path %q", contentPath)
			}
		}
		content = file.Data
	}
	if len(content) == 0 {
		return fmt.Errorf("spec content or content path is required")
	}
	if int64(len(content)) > c.options.MaxArtifactBytes {
		return fmt.Errorf("spec content is %d bytes, over limit %d", len(content), c.options.MaxArtifactBytes)
	}
	digest := sha256.Sum256(content)
	actualDigest := hex.EncodeToString(digest[:])
	if strings.TrimSpace(spec.SHA256) != "" {
		if !validSHA256(spec.SHA256) || strings.ToLower(spec.SHA256) != actualDigest {
			return fmt.Errorf("spec SHA256 does not match content")
		}
	}
	spec.SHA256 = actualDigest
	if spec.Bytes != 0 && spec.Bytes != int64(len(content)) {
		return fmt.Errorf("spec byte count is %d, want %d", spec.Bytes, len(content))
	}
	spec.Bytes = int64(len(content))
	metadataJSON, err := json.Marshal(spec.Metadata)
	if err != nil {
		return err
	}
	if int64(len(metadataJSON)) > c.options.MaxMetadataBytes {
		return fmt.Errorf("spec metadata is %d bytes, over limit %d", len(metadataJSON), c.options.MaxMetadataBytes)
	}
	if err := validateRecordBudget("cached spec", c.options.MaxMetadataBytes, originalURL, finalURL, spec.SHA256, contentPath); err != nil {
		return err
	}
	now := time.Now().UTC().Unix()
	accessed := time.Now().UTC().UnixNano()
	tx, err := c.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	for _, urlValue := range uniqueStrings(originalURL, finalURL) {
		if _, err := tx.ExecContext(ctx, `
INSERT INTO spec_documents (
  url, original_url, final_url, sha256, bytes, metadata_json, content_path, content, first_seen_at, updated_at, accessed_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(url) DO UPDATE SET
  original_url = excluded.original_url,
  final_url = excluded.final_url,
  sha256 = excluded.sha256,
  bytes = excluded.bytes,
  metadata_json = excluded.metadata_json,
  content_path = excluded.content_path,
  content = excluded.content,
	updated_at = excluded.updated_at,
	accessed_at = excluded.accessed_at`,
			urlValue, originalURL, finalURL, spec.SHA256, spec.Bytes, string(metadataJSON), nullableString(contentPath), nullableContent(contentPath, content), now, now, accessed); err != nil {
			return err
		}
	}
	if _, err := c.pruneTx(ctx, tx); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	committed = true
	return nil
}

// StoreCatalogArtifact records a local catalog curation artifact path. It
// hashes the file on disk so the database can be used as an integrity manifest.
func (c *Cache) StoreCatalogArtifact(ctx context.Context, artifact CatalogArtifact) error {
	if c == nil || c.db == nil {
		return fmt.Errorf("cache is closed")
	}
	providerID := strings.TrimSpace(artifact.ProviderID)
	artifactID := strings.TrimSpace(artifact.ArtifactID)
	kind := strings.TrimSpace(artifact.Kind)
	path, pathErr := cleanLocalPath(artifact.Path, "catalog artifact")
	if providerID == "" {
		return fmt.Errorf("provider id is required")
	}
	if artifactID == "" {
		return fmt.Errorf("artifact id is required")
	}
	if kind == "" {
		return fmt.Errorf("artifact kind is required")
	}
	if pathErr != nil {
		return pathErr
	}
	file, err := c.readArtifact(path, "", 0)
	if err != nil {
		return err
	}
	if file.Bytes <= 0 {
		return fmt.Errorf("catalog artifact %q is empty", path)
	}
	if strings.TrimSpace(artifact.SHA256) != "" && (!validSHA256(artifact.SHA256) || strings.ToLower(artifact.SHA256) != file.SHA256) {
		return fmt.Errorf("catalog artifact SHA256 does not match %q", path)
	}
	if artifact.Bytes != 0 && artifact.Bytes != file.Bytes {
		return fmt.Errorf("catalog artifact byte count is %d, want %d", artifact.Bytes, file.Bytes)
	}
	overlayPath, err := cleanOptionalLocalPath(artifact.OverlayPath, "catalog overlay")
	if err != nil {
		return err
	}
	builderPath, err := cleanOptionalLocalPath(artifact.BuilderPath, "catalog builder")
	if err != nil {
		return err
	}
	metadataJSON, err := json.Marshal(artifact.Metadata)
	if err != nil {
		return err
	}
	if int64(len(metadataJSON)) > c.options.MaxMetadataBytes {
		return fmt.Errorf("catalog artifact metadata is %d bytes, over limit %d", len(metadataJSON), c.options.MaxMetadataBytes)
	}
	if err := validateRecordBudget("catalog artifact", c.options.MaxMetadataBytes, providerID, artifactID, kind, path, artifact.SourceURL, overlayPath, builderPath, file.SHA256); err != nil {
		return err
	}
	now := time.Now().UTC().Unix()
	accessed := time.Now().UTC().UnixNano()
	tx, err := c.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	_, err = tx.ExecContext(ctx, `
INSERT INTO catalog_artifacts (
  provider_id, artifact_id, kind, path, source_url, overlay_path, builder_path,
  sha256, bytes, metadata_json, first_seen_at, updated_at, accessed_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(provider_id, artifact_id) DO UPDATE SET
  kind = excluded.kind,
  path = excluded.path,
  source_url = excluded.source_url,
  overlay_path = excluded.overlay_path,
  builder_path = excluded.builder_path,
  sha256 = excluded.sha256,
  bytes = excluded.bytes,
  metadata_json = excluded.metadata_json,
	updated_at = excluded.updated_at,
	accessed_at = excluded.accessed_at`,
		providerID,
		artifactID,
		kind,
		path,
		strings.TrimSpace(artifact.SourceURL),
		overlayPath,
		builderPath,
		file.SHA256,
		file.Bytes,
		string(metadataJSON),
		now,
		now,
		accessed,
	)
	if err != nil {
		return err
	}
	if _, err := c.pruneTx(ctx, tx); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	committed = true
	return nil
}

// ListCatalogArtifacts returns local catalog artifact path records in
// deterministic order.
func (c *Cache) ListCatalogArtifacts(ctx context.Context) ([]CatalogArtifact, error) {
	if c == nil || c.db == nil {
		return nil, fmt.Errorf("cache is closed")
	}
	rows, err := c.db.QueryContext(ctx, `
SELECT provider_id, artifact_id, kind, path, source_url, overlay_path,
  builder_path, sha256, bytes, metadata_json, updated_at
FROM catalog_artifacts
ORDER BY provider_id, artifact_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []CatalogArtifact
	for rows.Next() {
		var artifact CatalogArtifact
		var metadataJSON []byte
		var updatedAt int64
		if err := rows.Scan(
			&artifact.ProviderID,
			&artifact.ArtifactID,
			&artifact.Kind,
			&artifact.Path,
			&artifact.SourceURL,
			&artifact.OverlayPath,
			&artifact.BuilderPath,
			&artifact.SHA256,
			&artifact.Bytes,
			&metadataJSON,
			&updatedAt,
		); err != nil {
			return nil, err
		}
		if len(metadataJSON) > 0 {
			if int64(len(metadataJSON)) > c.options.MaxMetadataBytes {
				return nil, fmt.Errorf("catalog artifact %s/%s metadata is %d bytes, over limit %d", artifact.ProviderID, artifact.ArtifactID, len(metadataJSON), c.options.MaxMetadataBytes)
			}
			if err := json.Unmarshal(metadataJSON, &artifact.Metadata); err != nil {
				return nil, err
			}
		}
		if err := validateRecordBudget("catalog artifact", c.options.MaxMetadataBytes, artifact.ProviderID, artifact.ArtifactID, artifact.Kind, artifact.Path, artifact.SourceURL, artifact.OverlayPath, artifact.BuilderPath, artifact.SHA256); err != nil {
			return nil, err
		}
		cleanedPath, err := cleanLocalPath(artifact.Path, "catalog artifact")
		if err != nil {
			return nil, err
		}
		artifact.Path = cleanedPath
		artifact.OverlayPath, err = cleanOptionalLocalPath(artifact.OverlayPath, "catalog overlay")
		if err != nil {
			return nil, err
		}
		artifact.BuilderPath, err = cleanOptionalLocalPath(artifact.BuilderPath, "catalog builder")
		if err != nil {
			return nil, err
		}
		if !validSHA256(artifact.SHA256) || artifact.Bytes <= 0 {
			return nil, fmt.Errorf("catalog artifact %s/%s has invalid integrity metadata", artifact.ProviderID, artifact.ArtifactID)
		}
		artifact.StoredAt = time.Unix(updatedAt, 0).UTC()
		out = append(out, artifact)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	now := time.Now().UTC().UnixNano()
	for _, artifact := range out {
		if _, err := c.db.ExecContext(ctx, `UPDATE catalog_artifacts SET accessed_at = ? WHERE provider_id = ? AND artifact_id = ?`, now, artifact.ProviderID, artifact.ArtifactID); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func (c *Cache) migrate(ctx context.Context) error {
	stmts := []string{
		`PRAGMA journal_mode = WAL`,
		`PRAGMA busy_timeout = 5000`,
		`CREATE TABLE IF NOT EXISTS schema_meta (
			version INTEGER NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS search_results (
			id TEXT PRIMARY KEY,
			source TEXT NOT NULL,
			provider TEXT,
			title TEXT NOT NULL,
			description TEXT,
			version TEXT,
			categories_json TEXT,
			spec_url TEXT NOT NULL,
			landing_url TEXT,
			score INTEGER NOT NULL,
			validated INTEGER NOT NULL,
			provenance TEXT,
			first_seen_at INTEGER NOT NULL,
			last_seen_at INTEGER NOT NULL,
			accessed_at INTEGER NOT NULL DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS search_queries (
			key_hash TEXT PRIMARY KEY,
			query TEXT NOT NULL,
			source TEXT NOT NULL,
			limit_value INTEGER NOT NULL,
			public_probe INTEGER NOT NULL,
			result_ids_json TEXT NOT NULL,
			report_json BLOB NOT NULL,
			first_seen_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL,
			accessed_at INTEGER NOT NULL DEFAULT 0
		)`,
		createSpecDocumentsSQL,
		createCatalogArtifactsSQL,
	}
	for _, stmt := range stmts {
		if _, err := c.db.ExecContext(ctx, stmt); err != nil {
			return err
		}
	}
	if err := c.migrateSpecDocuments(ctx); err != nil {
		return err
	}
	if err := c.ensureAccessColumns(ctx); err != nil {
		return err
	}
	for _, stmt := range []string{
		`CREATE INDEX IF NOT EXISTS idx_search_results_spec_url ON search_results(spec_url)`,
		`CREATE INDEX IF NOT EXISTS idx_spec_documents_sha256 ON spec_documents(sha256)`,
		`CREATE INDEX IF NOT EXISTS idx_catalog_artifacts_kind ON catalog_artifacts(kind)`,
		`CREATE INDEX IF NOT EXISTS idx_search_queries_accessed_at ON search_queries(accessed_at)`,
		`CREATE INDEX IF NOT EXISTS idx_search_results_accessed_at ON search_results(accessed_at)`,
		`CREATE INDEX IF NOT EXISTS idx_spec_documents_accessed_at ON spec_documents(accessed_at)`,
		`CREATE INDEX IF NOT EXISTS idx_catalog_artifacts_accessed_at ON catalog_artifacts(accessed_at)`,
	} {
		if _, err := c.db.ExecContext(ctx, stmt); err != nil {
			return err
		}
	}
	if _, err := c.db.ExecContext(ctx, `DELETE FROM schema_meta`); err != nil {
		return err
	}
	if _, err := c.db.ExecContext(ctx, `INSERT INTO schema_meta(version) VALUES (?)`, schemaVersion); err != nil {
		return err
	}
	return nil
}

func searchKeyHash(key apitools.SearchCacheKey) string {
	key.Query = strings.TrimSpace(key.Query)
	if key.Source == "" {
		key.Source = apitools.SourceAuto
	}
	if key.Limit <= 0 {
		key.Limit = 10
	}
	data, _ := json.Marshal(key)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

const createSpecDocumentsSQL = `CREATE TABLE IF NOT EXISTS spec_documents (
	url TEXT PRIMARY KEY,
	original_url TEXT NOT NULL,
	final_url TEXT NOT NULL,
	sha256 TEXT NOT NULL,
	bytes INTEGER NOT NULL,
	metadata_json TEXT NOT NULL,
	content_path TEXT,
	content BLOB,
	first_seen_at INTEGER NOT NULL,
	updated_at INTEGER NOT NULL,
	accessed_at INTEGER NOT NULL DEFAULT 0,
	CHECK ((content_path IS NOT NULL AND content_path <> '') OR content IS NOT NULL)
)`

const createCatalogArtifactsSQL = `CREATE TABLE IF NOT EXISTS catalog_artifacts (
	provider_id TEXT NOT NULL,
	artifact_id TEXT NOT NULL,
	kind TEXT NOT NULL,
	path TEXT NOT NULL,
	source_url TEXT,
	overlay_path TEXT,
	builder_path TEXT,
	sha256 TEXT NOT NULL,
	bytes INTEGER NOT NULL,
	metadata_json TEXT NOT NULL,
	first_seen_at INTEGER NOT NULL,
	updated_at INTEGER NOT NULL,
	accessed_at INTEGER NOT NULL DEFAULT 0,
	PRIMARY KEY(provider_id, artifact_id)
)`

type sqliteColumn struct {
	name    string
	notNull bool
}

func (c *Cache) migrateSpecDocuments(ctx context.Context) error {
	columns, err := c.tableColumns(ctx, "spec_documents")
	if err != nil {
		return err
	}
	content, hasContent := columns["content"]
	_, hasContentPath := columns["content_path"]
	if hasContentPath && hasContent && !content.notNull {
		return nil
	}
	tx, err := c.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	if _, err := tx.ExecContext(ctx, `ALTER TABLE spec_documents RENAME TO spec_documents_old`); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, createSpecDocumentsSQL); err != nil {
		return err
	}
	contentPathSelect := `''`
	if hasContentPath {
		contentPathSelect = `content_path`
	}
	if _, err := tx.ExecContext(ctx, fmt.Sprintf(`
INSERT INTO spec_documents (
  url, original_url, final_url, sha256, bytes, metadata_json, content_path,
  content, first_seen_at, updated_at
)
SELECT
  url, original_url, final_url, sha256, bytes, metadata_json, %s,
  content, first_seen_at, updated_at
FROM spec_documents_old`, contentPathSelect)); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DROP TABLE spec_documents_old`); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	committed = true
	return nil
}

func (c *Cache) tableColumns(ctx context.Context, table string) (map[string]sqliteColumn, error) {
	rows, err := c.db.QueryContext(ctx, `PRAGMA table_info(`+table+`)`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	columns := map[string]sqliteColumn{}
	for rows.Next() {
		var cid int
		var name string
		var typ string
		var notNull int
		var defaultValue sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &typ, &notNull, &defaultValue, &pk); err != nil {
			return nil, err
		}
		columns[name] = sqliteColumn{name: name, notNull: notNull != 0}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return columns, nil
}

func (c *Cache) ensureAccessColumns(ctx context.Context) error {
	for _, migration := range []struct {
		table  string
		source string
	}{
		{table: "search_results", source: "last_seen_at"},
		{table: "search_queries", source: "updated_at"},
		{table: "spec_documents", source: "updated_at"},
		{table: "catalog_artifacts", source: "updated_at"},
	} {
		columns, err := c.tableColumns(ctx, migration.table)
		if err != nil {
			return err
		}
		if _, ok := columns["accessed_at"]; !ok {
			if _, err := c.db.ExecContext(ctx, `ALTER TABLE `+migration.table+` ADD COLUMN accessed_at INTEGER NOT NULL DEFAULT 0`); err != nil {
				return err
			}
		}
		if _, err := c.db.ExecContext(ctx, `UPDATE `+migration.table+` SET accessed_at = `+migration.source+` WHERE accessed_at = 0`); err != nil {
			return err
		}
	}
	return nil
}

// PruneReport records bounded least-recently-used cache eviction.
type PruneReport struct {
	SearchQueries    int64 `json:"search_queries"`
	SearchResults    int64 `json:"search_results"`
	SpecDocuments    int64 `json:"spec_documents"`
	CatalogArtifacts int64 `json:"catalog_artifacts"`
}

// Prune applies configured LRU limits in one transaction.
func (c *Cache) Prune(ctx context.Context) (PruneReport, error) {
	if c == nil || c.db == nil {
		return PruneReport{}, fmt.Errorf("cache is closed")
	}
	tx, err := c.db.BeginTx(ctx, nil)
	if err != nil {
		return PruneReport{}, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	report, err := c.pruneTx(ctx, tx)
	if err != nil {
		return PruneReport{}, err
	}
	if err := tx.Commit(); err != nil {
		return PruneReport{}, err
	}
	committed = true
	return report, nil
}

func (c *Cache) pruneTx(ctx context.Context, tx *sql.Tx) (PruneReport, error) {
	var report PruneReport
	for _, target := range []struct {
		limit   int
		query   string
		removed *int64
	}{
		{limit: c.options.MaxSearchQueries, query: `DELETE FROM search_queries WHERE rowid IN (SELECT rowid FROM search_queries ORDER BY accessed_at DESC, key_hash DESC LIMIT -1 OFFSET ?)`, removed: &report.SearchQueries},
		{limit: c.options.MaxSearchResults, query: `DELETE FROM search_results WHERE rowid IN (SELECT rowid FROM search_results ORDER BY accessed_at DESC, id DESC LIMIT -1 OFFSET ?)`, removed: &report.SearchResults},
		{limit: c.options.MaxSpecDocuments, query: `DELETE FROM spec_documents WHERE rowid IN (SELECT rowid FROM spec_documents ORDER BY accessed_at DESC, url DESC LIMIT -1 OFFSET ?)`, removed: &report.SpecDocuments},
		{limit: c.options.MaxCatalogArtifacts, query: `DELETE FROM catalog_artifacts WHERE rowid IN (SELECT rowid FROM catalog_artifacts ORDER BY accessed_at DESC, provider_id DESC, artifact_id DESC LIMIT -1 OFFSET ?)`, removed: &report.CatalogArtifacts},
	} {
		result, err := tx.ExecContext(ctx, target.query, target.limit)
		if err != nil {
			return PruneReport{}, err
		}
		*target.removed, err = result.RowsAffected()
		if err != nil {
			return PruneReport{}, err
		}
	}
	return report, nil
}

func (c *Cache) touchSearch(ctx context.Context, keyHash string, report apitools.SearchReport) error {
	now := time.Now().UTC().UnixNano()
	tx, err := c.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	if _, err := tx.ExecContext(ctx, `UPDATE search_queries SET accessed_at = ? WHERE key_hash = ?`, now, keyHash); err != nil {
		return err
	}
	for _, result := range report.Results {
		if _, err := tx.ExecContext(ctx, `UPDATE search_results SET accessed_at = ? WHERE id = ?`, now, result.ID); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	committed = true
	return nil
}

func cacheBaseDir(path string) string {
	if path == ":memory:" {
		return ""
	}
	return filepath.Dir(path)
}

func (c *Cache) readArtifact(path, sha256Value string, bytes int64) (artifactio.File, error) {
	if c.baseDir == "" {
		return artifactio.File{}, fmt.Errorf("path-backed artifacts require a file-backed cache")
	}
	cleaned, err := cleanLocalPath(path, "artifact")
	if err != nil {
		return artifactio.File{}, err
	}
	return artifactio.ReadFile(c.baseDir, cleaned, artifactio.ReadOptions{
		MaxBytes: c.options.MaxArtifactBytes,
		SHA256:   sha256Value,
		Bytes:    bytes,
	})
}

func nullableString(value string) sql.NullString {
	value = strings.TrimSpace(value)
	return sql.NullString{String: value, Valid: value != ""}
}

func nullableContent(contentPath string, content []byte) []byte {
	if strings.TrimSpace(contentPath) != "" {
		return nil
	}
	return content
}

func cleanLocalPath(path, label string) (string, error) {
	cleaned := filepath.Clean(filepath.FromSlash(strings.TrimSpace(path)))
	if cleaned == "." || !filepath.IsLocal(cleaned) {
		return "", fmt.Errorf("%s path %q must be a local relative path", label, path)
	}
	return filepath.ToSlash(cleaned), nil
}

func cleanOptionalLocalPath(path, label string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", nil
	}
	return cleanLocalPath(path, label)
}

func validSHA256(value string) bool {
	decoded, err := hex.DecodeString(strings.TrimSpace(value))
	return err == nil && len(decoded) == sha256.Size
}

func validateRecordBudget(label string, limit int64, values ...string) error {
	var total int64
	for _, value := range values {
		total += int64(len(value))
		if total > limit {
			return fmt.Errorf("%s metadata is %d bytes, over limit %d", label, total, limit)
		}
	}
	return nil
}

func expired(updatedAt int64, maxAge time.Duration) bool {
	if maxAge <= 0 {
		maxAge = apitools.DefaultCacheMaxAge
	}
	return time.Since(time.Unix(updatedAt, 0)) > maxAge
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func uniqueStrings(values ...string) []string {
	seen := map[string]bool{}
	var out []string
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}
