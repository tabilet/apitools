package sqlitecache

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/OpenUdon/apitools"
)

// RegisterCatalogRefreshResults records successfully saved refresh artifacts
// in the bounded SQLite cache. Artifact paths must remain under the cache file's
// directory so later reads retain the cache's path-containment guarantees.
func RegisterCatalogRefreshResults(ctx context.Context, cachePath, cacheDir string, report apitools.CatalogSpecRefreshReport) error {
	cachePath = strings.TrimSpace(cachePath)
	if cachePath == "" {
		return fmt.Errorf("cache path is required")
	}
	cache, err := Open(cachePath)
	if err != nil {
		return err
	}
	defer cache.Close()
	for _, result := range report.Results {
		contentPath, err := cache.catalogRefreshContentPath(cacheDir, result)
		if err != nil {
			return err
		}
		if err := cache.StoreSpec(ctx, apitools.CachedSpec{
			OriginalURL: result.URL,
			FinalURL:    result.FinalURL,
			ContentPath: contentPath,
			SHA256:      result.SHA256,
			Bytes:       result.Bytes,
			Metadata:    result.RawMetadata,
		}); err != nil {
			return err
		}
		if err := cache.StoreCatalogArtifact(ctx, CatalogArtifact{
			ProviderID: result.ProviderID,
			ArtifactID: result.SpecRefID,
			Kind:       string(result.Kind),
			Path:       contentPath,
			SourceURL:  result.URL,
			Metadata: map[string]string{
				"official":                    "true",
				"kind":                        string(result.Kind),
				"raw_validation_status":       result.RawValidationStatus,
				"corrected_validation_status": result.CorrectedValidationStatus,
				"sha256":                      result.SHA256,
			},
		}); err != nil {
			return err
		}
	}
	return nil
}

func (c *Cache) catalogRefreshContentPath(cacheDir string, result apitools.CatalogSpecRefreshResult) (string, error) {
	if c == nil || c.db == nil {
		return "", fmt.Errorf("cache is closed")
	}
	if c.baseDir == "" {
		return "", fmt.Errorf("catalog refresh artifacts require a file-backed cache")
	}
	fullPath := strings.TrimSpace(result.SavedPath)
	if fullPath == "" {
		fullPath = filepath.Join(cacheDir, filepath.FromSlash(result.ArtifactPath))
	}
	fullPath, err := filepath.Abs(fullPath)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(c.baseDir, fullPath)
	if err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel) {
		return filepath.ToSlash(rel), nil
	}
	return "", fmt.Errorf("refreshed artifact %q must stay under SQLite cache directory %q", fullPath, c.baseDir)
}
