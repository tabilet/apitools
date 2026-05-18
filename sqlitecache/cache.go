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
	_ "modernc.org/sqlite"
)

const schemaVersion = 2

type Cache struct {
	db      *sql.DB
	baseDir string
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
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, fmt.Errorf("cache path is required")
	}
	if path != ":memory:" {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return nil, err
		}
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	cache := &Cache{db: db, baseDir: cacheBaseDir(path)}
	if err := cache.migrate(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}
	return cache, nil
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
	var report apitools.SearchReport
	if err := json.Unmarshal(reportJSON, &report); err != nil {
		return apitools.SearchReport{}, false, err
	}
	return report, true, nil
}

func (c *Cache) StoreSearch(ctx context.Context, key apitools.SearchCacheKey, report apitools.SearchReport) error {
	if c == nil || c.db == nil {
		return fmt.Errorf("cache is closed")
	}
	now := time.Now().UTC().Unix()
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
  landing_url, score, validated, provenance, first_seen_at, last_seen_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
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
  last_seen_at = excluded.last_seen_at`,
			result.ID, result.Source, result.Provider, result.Title, result.Description,
			result.Version, string(categories), result.SpecURL, result.LandingURL, result.Score,
			boolInt(result.Validated), result.Provenance, now, now); err != nil {
			return err
		}
		resultIDs = append(resultIDs, result.ID)
	}
	resultIDsJSON, err := json.Marshal(resultIDs)
	if err != nil {
		return err
	}
	reportJSON, err := json.Marshal(report)
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `
INSERT INTO search_queries (
  key_hash, query, source, limit_value, public_probe, result_ids_json,
  report_json, first_seen_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(key_hash) DO UPDATE SET
  query = excluded.query,
  source = excluded.source,
  limit_value = excluded.limit_value,
  public_probe = excluded.public_probe,
  result_ids_json = excluded.result_ids_json,
  report_json = excluded.report_json,
  updated_at = excluded.updated_at`,
		searchKeyHash(key), key.Query, string(key.Source), key.Limit, key.PublicProbe,
		string(resultIDsJSON), reportJSON, now, now); err != nil {
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
	if err := json.Unmarshal(metadataJSON, &spec.Metadata); err != nil {
		return apitools.CachedSpec{}, false, err
	}
	if contentPath.Valid && strings.TrimSpace(contentPath.String) != "" {
		spec.ContentPath = filepath.ToSlash(filepath.Clean(contentPath.String))
		resolved, err := c.resolveArtifactPath(spec.ContentPath)
		if err != nil {
			return apitools.CachedSpec{}, false, err
		}
		content, err = os.ReadFile(resolved)
		if err != nil {
			return apitools.CachedSpec{}, false, fmt.Errorf("%w: read cached spec file %q: %v", apitools.ErrCachedSpecIntegrity, spec.ContentPath, err)
		}
	}
	if len(content) == 0 {
		return apitools.CachedSpec{}, false, fmt.Errorf("%w: cached spec %q has no content or content path", apitools.ErrCachedSpecIntegrity, strings.TrimSpace(rawURL))
	}
	spec.Content = append([]byte(nil), content...)
	if spec.SHA256 != "" {
		digest := sha256.Sum256(spec.Content)
		if got := hex.EncodeToString(digest[:]); got != spec.SHA256 {
			return apitools.CachedSpec{}, false, fmt.Errorf("%w: cached spec SHA256 mismatch for %q", apitools.ErrCachedSpecIntegrity, strings.TrimSpace(rawURL))
		}
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
	contentPath := filepath.ToSlash(filepath.Clean(strings.TrimSpace(spec.ContentPath)))
	if contentPath == "." {
		contentPath = ""
	}
	content := append([]byte(nil), spec.Content...)
	if len(content) == 0 && contentPath != "" {
		resolved, err := c.resolveArtifactPath(contentPath)
		if err != nil {
			return err
		}
		content, err = os.ReadFile(resolved)
		if err != nil {
			return fmt.Errorf("read cached spec file %q: %w", contentPath, err)
		}
	}
	if len(content) == 0 {
		return fmt.Errorf("spec content or content path is required")
	}
	digest := sha256.Sum256(content)
	spec.SHA256 = hex.EncodeToString(digest[:])
	if spec.Bytes == 0 {
		spec.Bytes = int64(len(content))
	}
	metadataJSON, err := json.Marshal(spec.Metadata)
	if err != nil {
		return err
	}
	now := time.Now().UTC().Unix()
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
  url, original_url, final_url, sha256, bytes, metadata_json, content_path, content, first_seen_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(url) DO UPDATE SET
  original_url = excluded.original_url,
  final_url = excluded.final_url,
  sha256 = excluded.sha256,
  bytes = excluded.bytes,
  metadata_json = excluded.metadata_json,
  content_path = excluded.content_path,
  content = excluded.content,
  updated_at = excluded.updated_at`,
			urlValue, originalURL, finalURL, spec.SHA256, spec.Bytes, string(metadataJSON), nullableString(contentPath), nullableContent(contentPath, content), now, now); err != nil {
			return err
		}
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
	path := filepath.ToSlash(filepath.Clean(strings.TrimSpace(artifact.Path)))
	if path == "." {
		path = ""
	}
	if providerID == "" {
		return fmt.Errorf("provider id is required")
	}
	if artifactID == "" {
		return fmt.Errorf("artifact id is required")
	}
	if kind == "" {
		return fmt.Errorf("artifact kind is required")
	}
	if path == "" {
		return fmt.Errorf("artifact path is required")
	}
	resolved, err := c.resolveArtifactPath(path)
	if err != nil {
		return err
	}
	content, err := os.ReadFile(resolved)
	if err != nil {
		return fmt.Errorf("read catalog artifact %q: %w", path, err)
	}
	digest := sha256.Sum256(content)
	metadataJSON, err := json.Marshal(artifact.Metadata)
	if err != nil {
		return err
	}
	now := time.Now().UTC().Unix()
	_, err = c.db.ExecContext(ctx, `
INSERT INTO catalog_artifacts (
  provider_id, artifact_id, kind, path, source_url, overlay_path, builder_path,
  sha256, bytes, metadata_json, first_seen_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(provider_id, artifact_id) DO UPDATE SET
  kind = excluded.kind,
  path = excluded.path,
  source_url = excluded.source_url,
  overlay_path = excluded.overlay_path,
  builder_path = excluded.builder_path,
  sha256 = excluded.sha256,
  bytes = excluded.bytes,
  metadata_json = excluded.metadata_json,
  updated_at = excluded.updated_at`,
		providerID,
		artifactID,
		kind,
		path,
		strings.TrimSpace(artifact.SourceURL),
		cleanOptionalPath(artifact.OverlayPath),
		cleanOptionalPath(artifact.BuilderPath),
		hex.EncodeToString(digest[:]),
		int64(len(content)),
		string(metadataJSON),
		now,
		now,
	)
	return err
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
			if err := json.Unmarshal(metadataJSON, &artifact.Metadata); err != nil {
				return nil, err
			}
		}
		artifact.StoredAt = time.Unix(updatedAt, 0).UTC()
		out = append(out, artifact)
	}
	if err := rows.Err(); err != nil {
		return nil, err
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
			last_seen_at INTEGER NOT NULL
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
			updated_at INTEGER NOT NULL
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
	for _, stmt := range []string{
		`CREATE INDEX IF NOT EXISTS idx_search_results_spec_url ON search_results(spec_url)`,
		`CREATE INDEX IF NOT EXISTS idx_spec_documents_sha256 ON spec_documents(sha256)`,
		`CREATE INDEX IF NOT EXISTS idx_catalog_artifacts_kind ON catalog_artifacts(kind)`,
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

func cacheBaseDir(path string) string {
	if path == ":memory:" {
		return ""
	}
	return filepath.Dir(path)
}

func (c *Cache) resolveArtifactPath(path string) (string, error) {
	cleaned := filepath.Clean(filepath.FromSlash(strings.TrimSpace(path)))
	if cleaned == "." || cleaned == "" {
		return "", fmt.Errorf("artifact path is required")
	}
	if filepath.IsAbs(cleaned) {
		return cleaned, nil
	}
	if c.baseDir != "" {
		basePath := filepath.Join(c.baseDir, cleaned)
		if _, err := os.Stat(basePath); err == nil || !errorsIsNotExist(err) {
			return basePath, err
		}
	}
	return cleaned, nil
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

func cleanOptionalPath(path string) string {
	cleaned := filepath.ToSlash(filepath.Clean(strings.TrimSpace(path)))
	if cleaned == "." {
		return ""
	}
	return cleaned
}

func errorsIsNotExist(err error) bool {
	return err != nil && os.IsNotExist(err)
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
