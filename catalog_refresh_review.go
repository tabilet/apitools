package apitools

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/OpenUdon/apitools/catalog"
)

const (
	CatalogRefreshMissingRegistration   = "missing-registration"
	CatalogRefreshMissingFile           = "missing-file"
	CatalogRefreshReviewDefaultMaxBytes = 128 * 1024 * 1024
)

// CatalogSpecRefreshReviewOptions controls offline review of saved catalog
// refresh artifacts. It never fetches remote documents or creates cache files.
type CatalogSpecRefreshReviewOptions struct {
	CacheDir              string
	Artifacts             []catalog.CatalogSpecArtifact
	AsOf                  time.Time
	StaleVerificationDays int
	MaxBytes              int64
}

// CatalogSpecRefreshReviewReport records local refresh artifact evidence and
// deterministic manual follow-ups for catalog maintainers.
type CatalogSpecRefreshReviewReport struct {
	Results []CatalogSpecRefreshReviewResult `json:"results,omitempty"`
}

// CatalogSpecRefreshReviewResult records one offline refresh artifact review
// row for a built-in refreshable spec reference.
type CatalogSpecRefreshReviewResult struct {
	ProviderID             string                  `json:"provider_id"`
	ProviderName           string                  `json:"provider_name,omitempty"`
	SpecRefID              string                  `json:"spec_ref_id"`
	Kind                   catalog.SpecKind        `json:"kind"`
	URL                    string                  `json:"url"`
	SourceAuthority        catalog.SourceAuthority `json:"source_authority,omitempty"`
	VerifiedAt             string                  `json:"verified_at,omitempty"`
	RegisteredArtifactPath string                  `json:"registered_artifact_path,omitempty"`
	SavedPath              string                  `json:"saved_path,omitempty"`
	Exists                 bool                    `json:"exists"`
	Bytes                  int64                   `json:"bytes,omitempty"`
	SHA256                 string                  `json:"sha256,omitempty"`
	ValidationStatus       string                  `json:"validation_status"`
	ValidationError        string                  `json:"validation_error,omitempty"`
	VerificationStale      bool                    `json:"verification_stale,omitempty"`
	Metadata               SpecMetadata            `json:"metadata,omitempty"`
	ManualFollowUps        []string                `json:"manual_follow_ups,omitempty"`
}

// BuiltInCatalogSpecRefreshReviewReport reviews built-in refreshable catalog
// spec references against existing local artifact registrations and files.
func BuiltInCatalogSpecRefreshReviewReport(opts CatalogSpecRefreshReviewOptions) (CatalogSpecRefreshReviewReport, error) {
	refs := catalog.BuiltInRefreshableSpecReferences(opts.Artifacts)
	return BuildCatalogSpecRefreshReviewReport(refs, opts)
}

// BuildCatalogSpecRefreshReviewReport reviews refreshable spec references
// without network access or cache creation.
func BuildCatalogSpecRefreshReviewReport(refs []catalog.RefreshableSpecReference, opts CatalogSpecRefreshReviewOptions) (CatalogSpecRefreshReviewReport, error) {
	opts = normalizeCatalogSpecRefreshReviewOptions(opts)
	cacheDir := strings.TrimSpace(opts.CacheDir)
	if cacheDir == "" {
		return CatalogSpecRefreshReviewReport{}, fmt.Errorf("cache directory is required")
	}
	refs = append([]catalog.RefreshableSpecReference(nil), refs...)
	sort.SliceStable(refs, func(i, j int) bool {
		if refs[i].ProviderID == refs[j].ProviderID {
			return refs[i].SpecRefID < refs[j].SpecRefID
		}
		return refs[i].ProviderID < refs[j].ProviderID
	})

	var report CatalogSpecRefreshReviewReport
	for _, ref := range refs {
		result, err := reviewCatalogRefreshArtifact(ref, opts)
		if err != nil {
			return CatalogSpecRefreshReviewReport{}, err
		}
		report.Results = append(report.Results, result)
	}
	return report, nil
}

func normalizeCatalogSpecRefreshReviewOptions(opts CatalogSpecRefreshReviewOptions) CatalogSpecRefreshReviewOptions {
	if opts.AsOf.IsZero() {
		opts.AsOf = time.Now().UTC()
	}
	if opts.StaleVerificationDays <= 0 {
		opts.StaleVerificationDays = catalog.DefaultStaleVerificationDays
	}
	if opts.MaxBytes <= 0 {
		opts.MaxBytes = CatalogRefreshReviewDefaultMaxBytes
	}
	return opts
}

func reviewCatalogRefreshArtifact(ref catalog.RefreshableSpecReference, opts CatalogSpecRefreshReviewOptions) (CatalogSpecRefreshReviewResult, error) {
	result := CatalogSpecRefreshReviewResult{
		ProviderID:             ref.ProviderID,
		ProviderName:           ref.ProviderName,
		SpecRefID:              ref.SpecRefID,
		Kind:                   ref.Kind,
		URL:                    ref.URL,
		SourceAuthority:        ref.SourceAuthority,
		VerifiedAt:             ref.VerifiedAt,
		RegisteredArtifactPath: strings.TrimSpace(ref.RegisteredArtifactPath),
	}

	path := result.RegisteredArtifactPath
	if path == "" {
		if discovered := discoverUnregisteredCatalogRefreshArtifact(opts.CacheDir, ref); discovered != "" {
			path = discovered
			result.ManualFollowUps = append(result.ManualFollowUps, "Register the saved artifact path in cache.sqlite if it is accepted.")
		} else {
			result.ValidationStatus = CatalogRefreshMissingRegistration
			result.ManualFollowUps = append(result.ManualFollowUps, "Run a selected catalog refresh or register an existing saved artifact path before promoting metadata.")
			result.addCatalogRefreshReviewFollowUps(opts)
			return result, nil
		}
	}

	savedPath, err := resolveCatalogRefreshReviewPath(opts.CacheDir, path)
	if err != nil {
		result.ValidationStatus = CatalogRefreshInvalid
		result.ValidationError = err.Error()
		result.ManualFollowUps = append(result.ManualFollowUps, "Fix the registered artifact path so it points inside the catalog cache directory.")
		result.addCatalogRefreshReviewFollowUps(opts)
		return result, nil
	}
	result.SavedPath = savedPath

	content, size, exists, err := readCatalogRefreshReviewArtifact(savedPath, opts.MaxBytes)
	result.Exists = exists
	result.Bytes = size
	if err != nil {
		if os.IsNotExist(err) {
			result.ValidationStatus = CatalogRefreshMissingFile
			result.ManualFollowUps = append(result.ManualFollowUps, "Restore the registered artifact file or rerun the selected catalog refresh.")
			result.addCatalogRefreshReviewFollowUps(opts)
			return result, nil
		}
		result.ValidationStatus = CatalogRefreshInvalid
		result.ValidationError = err.Error()
		result.ManualFollowUps = append(result.ManualFollowUps, "Review or replace the saved artifact before promoting durable catalog metadata.")
		result.addCatalogRefreshReviewFollowUps(opts)
		return result, nil
	}
	digest := sha256.Sum256(content)
	result.SHA256 = hex.EncodeToString(digest[:])

	status, metadata, err := validateCatalogRefreshContent(context.Background(), ref, content, ref.URL)
	result.ValidationStatus = status
	result.Metadata = metadata
	if err != nil {
		result.ValidationError = err.Error()
		if !catalogRefreshStatusAllowsSave(status) {
			result.ValidationStatus = CatalogRefreshInvalid
			result.ManualFollowUps = append(result.ManualFollowUps, "Review or replace the saved artifact before promoting durable catalog metadata.")
		}
	}
	result.addCatalogRefreshReviewFollowUps(opts)
	return result, nil
}

func (result *CatalogSpecRefreshReviewResult) addCatalogRefreshReviewFollowUps(opts CatalogSpecRefreshReviewOptions) {
	if result.VerifiedAt == "" {
		result.ManualFollowUps = append(result.ManualFollowUps, "Record a verification date if this artifact is accepted.")
	} else {
		verifiedAt, err := time.Parse("2006-01-02", strings.TrimSpace(result.VerifiedAt))
		if err != nil {
			result.ManualFollowUps = append(result.ManualFollowUps, "Fix the verification date format before promoting metadata.")
		} else if opts.AsOf.Sub(verifiedAt) > time.Duration(opts.StaleVerificationDays)*24*time.Hour {
			result.VerificationStale = true
			result.ManualFollowUps = append(result.ManualFollowUps, fmt.Sprintf("Review stale verification date %s; it is more than %d days before %s.", result.VerifiedAt, opts.StaleVerificationDays, opts.AsOf.Format("2006-01-02")))
		}
	}
	switch result.ValidationStatus {
	case CatalogRefreshValidOpenAPI, CatalogRefreshValidSwagger, CatalogRefreshValidStructured, CatalogRefreshSkippedValidation:
		result.ManualFollowUps = append(result.ManualFollowUps, "Review the local artifact before updating durable catalog metadata.")
		result.ManualFollowUps = append(result.ManualFollowUps, "Update spec revision, verification date, and security classification manually if the artifact is accepted.")
	case CatalogRefreshParseableOpenAPIInvalid, CatalogRefreshParseableSwaggerInvalid:
		result.ManualFollowUps = append(result.ManualFollowUps, "Review the local artifact before updating durable catalog metadata.")
		result.ManualFollowUps = append(result.ManualFollowUps, "Review strict validation errors before treating this artifact as import-ready metadata.")
	}
}

func discoverUnregisteredCatalogRefreshArtifact(cacheDir string, ref catalog.RefreshableSpecReference) string {
	dirs := []string{"openapi"}
	if ref.Kind == catalog.SpecKindGoogleDiscovery {
		dirs = []string{"google-discovery"}
	}
	cacheAbs, err := filepath.Abs(cacheDir)
	if err != nil {
		return ""
	}
	var candidates []catalogRefreshReviewCandidate
	for _, dir := range dirs {
		root, err := resolveCatalogRefreshReviewPath(cacheDir, dir)
		if err != nil {
			continue
		}
		_ = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
			if err != nil || entry.IsDir() {
				return nil
			}
			if entry.Type()&os.ModeSymlink != 0 || !catalogRefreshReviewExtensionSupported(path) {
				return nil
			}
			rel, err := filepath.Rel(cacheAbs, path)
			if err != nil {
				return nil
			}
			rel = filepath.ToSlash(rel)
			if score := catalogRefreshReviewCandidateScore(rel, ref); score > 0 {
				candidates = append(candidates, catalogRefreshReviewCandidate{path: rel, score: score})
			}
			return nil
		})
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].score == candidates[j].score {
			return candidates[i].path < candidates[j].path
		}
		return candidates[i].score > candidates[j].score
	})
	if len(candidates) > 0 {
		return candidates[0].path
	}
	return ""
}

type catalogRefreshReviewCandidate struct {
	path  string
	score int
}

func catalogRefreshReviewCandidateScore(path string, ref catalog.RefreshableSpecReference) int {
	base := strings.TrimSuffix(filepath.Base(path), ".tar.gz")
	base = strings.TrimSuffix(base, ".tgz")
	base = strings.TrimSuffix(base, ".json")
	base = strings.TrimSuffix(base, ".yaml")
	base = strings.TrimSuffix(base, ".yml")
	normalizedBase := normalizeCatalogRefreshReviewName(base)
	normalizedSpec := normalizeCatalogRefreshReviewName(ref.SpecRefID)
	normalizedProvider := normalizeCatalogRefreshReviewName(ref.ProviderID)
	switch {
	case normalizedBase == normalizedSpec:
		return 100
	case normalizedSpec != "" && strings.Contains(normalizedBase, normalizedSpec):
		return 90
	case normalizedProvider != "" && strings.Contains(normalizedBase, normalizedProvider):
		return 80
	default:
		return 0
	}
}

func normalizeCatalogRefreshReviewName(value string) string {
	var out strings.Builder
	for _, r := range strings.ToLower(value) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			out.WriteRune(r)
		}
	}
	return out.String()
}

func catalogRefreshReviewExtensionSupported(path string) bool {
	lower := strings.ToLower(path)
	return strings.HasSuffix(lower, ".json") || strings.HasSuffix(lower, ".yaml") || strings.HasSuffix(lower, ".yml") || strings.HasSuffix(lower, ".tar.gz") || strings.HasSuffix(lower, ".tgz")
}

func resolveCatalogRefreshReviewPath(cacheDir, artifactPath string) (string, error) {
	cacheDir = strings.TrimSpace(cacheDir)
	if cacheDir == "" {
		return "", fmt.Errorf("cache directory is required")
	}
	cleaned := filepath.Clean(filepath.FromSlash(strings.TrimSpace(artifactPath)))
	if cleaned == "." || cleaned == "" {
		return "", fmt.Errorf("artifact path is required")
	}
	cacheAbs, err := filepath.Abs(cacheDir)
	if err != nil {
		return "", err
	}
	var fullPath string
	if filepath.IsAbs(cleaned) {
		fullPath = cleaned
	} else {
		if cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
			return "", fmt.Errorf("artifact path %q must stay under catalog cache directory", artifactPath)
		}
		fullPath = filepath.Join(cacheAbs, cleaned)
	}
	fullAbs, err := filepath.Abs(fullPath)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(cacheAbs, fullAbs)
	if err != nil {
		return "", err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return "", fmt.Errorf("artifact path %q must stay under catalog cache directory", artifactPath)
	}
	return fullAbs, nil
}

func readCatalogRefreshReviewArtifact(path string, maxBytes int64) ([]byte, int64, bool, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, 0, false, err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return nil, 0, true, fmt.Errorf("artifact path %q is a symlink", path)
	}
	if !info.Mode().IsRegular() {
		return nil, 0, true, fmt.Errorf("artifact path %q is not a regular file", path)
	}
	if info.Size() > maxBytes {
		return nil, info.Size(), true, fmt.Errorf("artifact path %q is %d bytes, over limit %d", path, info.Size(), maxBytes)
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, info.Size(), true, err
	}
	defer file.Close()
	content, err := io.ReadAll(io.LimitReader(file, maxBytes+1))
	if err != nil {
		return nil, info.Size(), true, err
	}
	if int64(len(content)) > maxBytes {
		return nil, int64(len(content)), true, fmt.Errorf("artifact path %q is over limit %d", path, maxBytes)
	}
	return content, int64(len(content)), true, nil
}
