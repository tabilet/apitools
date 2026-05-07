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

const schemaVersion = 1

type Cache struct {
	db *sql.DB
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
	cache := &Cache{db: db}
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
	var updatedAt int64
	err := c.db.QueryRowContext(ctx, `
SELECT original_url, final_url, sha256, bytes, metadata_json, content, updated_at
FROM spec_documents
WHERE url = ?`, strings.TrimSpace(rawURL)).Scan(&spec.OriginalURL, &spec.FinalURL, &spec.SHA256, &spec.Bytes, &metadataJSON, &spec.Content, &updatedAt)
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
	content := append([]byte(nil), spec.Content...)
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
  url, original_url, final_url, sha256, bytes, metadata_json, content, first_seen_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(url) DO UPDATE SET
  original_url = excluded.original_url,
  final_url = excluded.final_url,
  sha256 = excluded.sha256,
  bytes = excluded.bytes,
  metadata_json = excluded.metadata_json,
  content = excluded.content,
  updated_at = excluded.updated_at`,
			urlValue, originalURL, finalURL, spec.SHA256, spec.Bytes, string(metadataJSON), content, now, now); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	committed = true
	return nil
}

func (c *Cache) migrate(ctx context.Context) error {
	stmts := []string{
		`PRAGMA journal_mode = WAL`,
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
		`CREATE TABLE IF NOT EXISTS spec_documents (
			url TEXT PRIMARY KEY,
			original_url TEXT NOT NULL,
			final_url TEXT NOT NULL,
			sha256 TEXT NOT NULL,
			bytes INTEGER NOT NULL,
			metadata_json TEXT NOT NULL,
			content BLOB NOT NULL,
			first_seen_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_search_results_spec_url ON search_results(spec_url)`,
		`CREATE INDEX IF NOT EXISTS idx_spec_documents_sha256 ON spec_documents(sha256)`,
		`DELETE FROM schema_meta`,
		`INSERT INTO schema_meta(version) VALUES (1)`,
	}
	for _, stmt := range stmts {
		if _, err := c.db.ExecContext(ctx, stmt); err != nil {
			return err
		}
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
