package apitools

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/OpenUdon/apitools/catalog"
	"gopkg.in/yaml.v3"
)

const (
	CatalogRefreshDownloaded              = "downloaded"
	CatalogRefreshValidOpenAPI            = "valid-openapi"
	CatalogRefreshValidSwagger            = "valid-swagger"
	CatalogRefreshParseableOpenAPIInvalid = "parseable-openapi-invalid"
	CatalogRefreshParseableSwaggerInvalid = "parseable-swagger-invalid"
	CatalogRefreshValidStructured         = "valid-structured"
	CatalogRefreshSkippedValidation       = "skipped-non-openapi"
	CatalogRefreshInvalid                 = "invalid"
)

// CatalogSpecRefreshOptions controls a selected, opt-in catalog spec refresh.
// It never promotes catalog metadata or executes provider API operations.
type CatalogSpecRefreshOptions struct {
	CacheDir string
}

// CatalogSpecRefreshReport records deterministic metadata about refreshed
// catalog review artifacts and the manual follow-ups still required.
type CatalogSpecRefreshReport struct {
	Results []CatalogSpecRefreshResult `json:"results,omitempty"`
}

// CatalogSpecRefreshResult records one downloaded catalog review artifact.
type CatalogSpecRefreshResult struct {
	ProviderID       string               `json:"provider_id"`
	SpecRefID        string               `json:"spec_ref_id"`
	Kind             catalog.SpecKind     `json:"kind"`
	Protocol         catalog.SpecProtocol `json:"protocol"`
	ProtocolVersion  string               `json:"protocol_version,omitempty"`
	URL              string               `json:"url"`
	FinalURL         string               `json:"final_url,omitempty"`
	DownloadStatus   string               `json:"download_status"`
	ValidationStatus string               `json:"validation_status"`
	ValidationError  string               `json:"validation_error,omitempty"`
	ArtifactPath     string               `json:"artifact_path,omitempty"`
	SavedPath        string               `json:"saved_path,omitempty"`
	SHA256           string               `json:"sha256,omitempty"`
	Bytes            int64                `json:"bytes,omitempty"`
	Metadata         SpecMetadata         `json:"metadata,omitempty"`
	CorrectionNotes  []string             `json:"correction_notes,omitempty"`
	ManualFollowUps  []string             `json:"manual_follow_ups,omitempty"`
}

// RefreshCatalogSpecReferences downloads selected catalog spec references into
// an ignored curation cache directory using the same safe HTTP(S) download path
// as imports. OpenAPI/Swagger roots are validated; non-OpenAPI machine specs
// are saved as review artifacts without pretending they are OpenAPI.
func (c *Client) RefreshCatalogSpecReferences(ctx context.Context, refs []catalog.RefreshableSpecReference, opts CatalogSpecRefreshOptions) (CatalogSpecRefreshReport, error) {
	c = c.effective()
	cacheDir := strings.TrimSpace(opts.CacheDir)
	if cacheDir == "" {
		return CatalogSpecRefreshReport{}, fmt.Errorf("cache directory is required")
	}
	var report CatalogSpecRefreshReport
	for _, ref := range refs {
		result, err := c.refreshCatalogSpecReference(ctx, ref, cacheDir)
		if err != nil {
			return CatalogSpecRefreshReport{}, err
		}
		report.Results = append(report.Results, result)
	}
	return report, nil
}

func (c *Client) refreshCatalogSpecReference(ctx context.Context, ref catalog.RefreshableSpecReference, cacheDir string) (CatalogSpecRefreshResult, error) {
	content, finalURL, err := c.downloadBounded(ctx, ref.URL)
	if err != nil {
		return CatalogSpecRefreshResult{}, err
	}
	validationStatus, metadata, correctionNotes, err := validateCatalogRefreshContentWithCorrections(ctx, ref, content, finalURL.String())
	if err != nil && !catalogRefreshStatusAllowsSave(validationStatus) {
		return CatalogSpecRefreshResult{}, err
	}
	validationError := ""
	if err != nil {
		validationError = err.Error()
	}
	artifactPath, err := catalogRefreshArtifactPath(ref, content)
	if err != nil {
		return CatalogSpecRefreshResult{}, err
	}
	savedPath, err := writeCatalogRefreshArtifact(cacheDir, artifactPath, content)
	if err != nil {
		return CatalogSpecRefreshResult{}, err
	}
	digest := sha256.Sum256(content)
	manualFollowUps := []string{
		"Review the downloaded artifact before updating durable catalog metadata.",
		"Update spec revision, verification date, and security classification manually if the refreshed artifact is accepted.",
	}
	if catalogRefreshStatusNeedsStrictReview(validationStatus) {
		manualFollowUps = append(manualFollowUps, "Review strict validation errors before treating this artifact as import-ready metadata.")
	}
	if len(correctionNotes) > 0 {
		manualFollowUps = append(manualFollowUps, "Review catalog refresh correction notes before treating this artifact as import-ready metadata.")
	}
	protocol := catalogRefreshProtocolClassification(ref, metadata)
	return CatalogSpecRefreshResult{
		ProviderID:       ref.ProviderID,
		SpecRefID:        ref.SpecRefID,
		Kind:             ref.Kind,
		Protocol:         protocol.Protocol,
		ProtocolVersion:  protocol.Version,
		URL:              ref.URL,
		FinalURL:         finalURL.String(),
		DownloadStatus:   CatalogRefreshDownloaded,
		ValidationStatus: validationStatus,
		ValidationError:  validationError,
		ArtifactPath:     artifactPath,
		SavedPath:        savedPath,
		SHA256:           hex.EncodeToString(digest[:]),
		Bytes:            int64(len(content)),
		Metadata:         metadata,
		CorrectionNotes:  correctionNotes,
		ManualFollowUps:  manualFollowUps,
	}, nil
}

func catalogRefreshProtocolClassification(ref catalog.RefreshableSpecReference, metadata SpecMetadata) catalog.SpecProtocolClassification {
	if metadata.Swagger != "" {
		return catalog.SpecProtocolClassification{Protocol: catalog.SpecProtocolSwagger, Version: metadata.Swagger}
	}
	if metadata.OpenAPI != "" {
		return catalog.SpecProtocolClassification{Protocol: catalog.SpecProtocolOpenAPI, Version: metadata.OpenAPI}
	}
	return ref.ProtocolClassification()
}

func validateCatalogRefreshContent(ctx context.Context, ref catalog.RefreshableSpecReference, content []byte, sourceURL string) (string, SpecMetadata, error) {
	status, metadata, _, err := validateCatalogRefreshContentWithCorrections(ctx, ref, content, sourceURL)
	return status, metadata, err
}

func validateCatalogRefreshContentWithCorrections(ctx context.Context, ref catalog.RefreshableSpecReference, content []byte, sourceURL string) (string, SpecMetadata, []string, error) {
	correctedContent, correctionNotes := correctedCatalogRefreshContent(ref, content)
	status, metadata, err := validateCatalogRefreshContentRaw(ctx, ref, correctedContent, sourceURL)
	return status, metadata, correctionNotes, err
}

func validateCatalogRefreshContentRaw(ctx context.Context, ref catalog.RefreshableSpecReference, content []byte, sourceURL string) (string, SpecMetadata, error) {
	switch ref.Kind {
	case catalog.SpecKindOpenAPI:
		metadata, ok := downloadedSpecMetadata(ctx, content, sourceURL)
		if !ok {
			status, metadata, parseable := parseableInvalidCatalogOpenAPIMetadata(ctx, content)
			if parseable {
				return status, metadata, fmt.Errorf("%s/%s: downloaded document is parseable as OpenAPI or Swagger but fails strict semantic validation", ref.ProviderID, ref.SpecRefID)
			}
			return CatalogRefreshInvalid, SpecMetadata{}, fmt.Errorf("%s/%s: downloaded document does not validate as OpenAPI or Swagger", ref.ProviderID, ref.SpecRefID)
		}
		if metadata.Swagger != "" {
			return CatalogRefreshValidSwagger, metadata, nil
		}
		return CatalogRefreshValidOpenAPI, metadata, nil
	case catalog.SpecKindGoogleDiscovery, catalog.SpecKindOpenAPIIndex, catalog.SpecKindSmithyJSON:
		if !validStructuredCatalogArtifact(content) {
			return CatalogRefreshInvalid, SpecMetadata{}, fmt.Errorf("%s/%s: downloaded document is not structured JSON or YAML", ref.ProviderID, ref.SpecRefID)
		}
		return CatalogRefreshValidStructured, structuredMetadata(content), nil
	case catalog.SpecKindDropboxStone:
		if len(bytes.TrimSpace(content)) == 0 {
			return CatalogRefreshInvalid, SpecMetadata{}, fmt.Errorf("%s/%s: downloaded artifact is empty", ref.ProviderID, ref.SpecRefID)
		}
		return CatalogRefreshSkippedValidation, SpecMetadata{}, nil
	default:
		return CatalogRefreshInvalid, SpecMetadata{}, fmt.Errorf("%s/%s: unsupported refresh spec kind %q", ref.ProviderID, ref.SpecRefID, ref.Kind)
	}
}

func catalogRefreshStatusAllowsSave(status string) bool {
	switch status {
	case CatalogRefreshValidOpenAPI, CatalogRefreshValidSwagger, CatalogRefreshParseableOpenAPIInvalid, CatalogRefreshParseableSwaggerInvalid, CatalogRefreshValidStructured, CatalogRefreshSkippedValidation:
		return true
	default:
		return false
	}
}

func catalogRefreshStatusNeedsStrictReview(status string) bool {
	switch status {
	case CatalogRefreshParseableOpenAPIInvalid, CatalogRefreshParseableSwaggerInvalid:
		return true
	default:
		return false
	}
}

func parseableInvalidCatalogOpenAPIMetadata(ctx context.Context, content []byte) (string, SpecMetadata, bool) {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return CatalogRefreshInvalid, SpecMetadata{}, false
		}
	}
	trimmed := bytes.TrimSpace(content)
	if len(trimmed) == 0 {
		return CatalogRefreshInvalid, SpecMetadata{}, false
	}
	var root map[string]any
	if trimmed[0] == '{' {
		if err := json.Unmarshal(trimmed, &root); err != nil {
			return CatalogRefreshInvalid, SpecMetadata{}, false
		}
	} else if err := yaml.Unmarshal(trimmed, &root); err != nil {
		return CatalogRefreshInvalid, SpecMetadata{}, false
	}
	if len(root) == 0 || containsExternalRef(root, 0) {
		return CatalogRefreshInvalid, SpecMetadata{}, false
	}
	if metadata, ok := parseableInvalidOpenAPI3Metadata(root); ok {
		return CatalogRefreshParseableOpenAPIInvalid, metadata, true
	}
	if metadata, ok := parseableInvalidSwagger2Metadata(root); ok {
		return CatalogRefreshParseableSwaggerInvalid, metadata, true
	}
	return CatalogRefreshInvalid, SpecMetadata{}, false
}

func parseableInvalidOpenAPI3Metadata(root map[string]any) (SpecMetadata, bool) {
	openapi := strings.TrimSpace(catalogRefreshStringValue(root, "openapi"))
	if !strings.HasPrefix(openapi, "3.0") && !strings.HasPrefix(openapi, "3.1") {
		return SpecMetadata{}, false
	}
	info, ok := root["info"].(map[string]any)
	if !ok {
		return SpecMetadata{}, false
	}
	title := catalogRefreshStringValue(info, "title")
	if title == "" {
		return SpecMetadata{}, false
	}
	paths, ok := root["paths"].(map[string]any)
	if !ok {
		return SpecMetadata{}, false
	}
	for path, value := range paths {
		if !strings.HasPrefix(path, "/") && !strings.HasPrefix(path, "x-") {
			return SpecMetadata{}, false
		}
		if _, ok := value.(map[string]any); !ok {
			return SpecMetadata{}, false
		}
	}
	return SpecMetadata{
		Title:          title,
		Description:    catalogRefreshStringValue(info, "description"),
		OpenAPI:        openapi,
		OperationCount: looseOperationCount(paths),
	}, true
}

func parseableInvalidSwagger2Metadata(root map[string]any) (SpecMetadata, bool) {
	swagger := strings.TrimSpace(catalogRefreshStringValue(root, "swagger"))
	if swagger != "2.0" {
		return SpecMetadata{}, false
	}
	info, ok := root["info"].(map[string]any)
	if !ok {
		return SpecMetadata{}, false
	}
	title := catalogRefreshStringValue(info, "title")
	version := catalogRefreshStringValue(info, "version")
	if title == "" || version == "" {
		return SpecMetadata{}, false
	}
	paths, ok := root["paths"].(map[string]any)
	if !ok {
		return SpecMetadata{}, false
	}
	for path, value := range paths {
		if !strings.HasPrefix(path, "/") && !strings.HasPrefix(path, "x-") {
			return SpecMetadata{}, false
		}
		if _, ok := value.(map[string]any); !ok {
			return SpecMetadata{}, false
		}
	}
	return SpecMetadata{
		Title:          title,
		Description:    catalogRefreshStringValue(info, "description"),
		Swagger:        swagger,
		OperationCount: looseOperationCount(paths),
	}, true
}

func validStructuredCatalogArtifact(content []byte) bool {
	trimmed := bytes.TrimSpace(content)
	if len(trimmed) == 0 {
		return false
	}
	var root map[string]any
	if trimmed[0] == '{' {
		return json.Unmarshal(trimmed, &root) == nil && len(root) > 0
	}
	return yaml.Unmarshal(trimmed, &root) == nil && len(root) > 0
}

func structuredMetadata(content []byte) SpecMetadata {
	trimmed := bytes.TrimSpace(content)
	var root map[string]any
	if len(trimmed) > 0 && trimmed[0] == '{' {
		_ = json.Unmarshal(trimmed, &root)
	} else {
		_ = yaml.Unmarshal(trimmed, &root)
	}
	title := catalogRefreshStringValue(root, "title")
	if title == "" {
		if info, ok := root["info"].(map[string]any); ok {
			title = catalogRefreshStringValue(info, "title")
		}
	}
	version := catalogRefreshStringValue(root, "version")
	if version == "" {
		if info, ok := root["info"].(map[string]any); ok {
			version = catalogRefreshStringValue(info, "version")
		}
	}
	return SpecMetadata{Title: title, OpenAPI: "", Swagger: "", Description: version}
}

func catalogRefreshStringValue(root map[string]any, key string) string {
	value, _ := root[key].(string)
	return strings.TrimSpace(value)
}

func catalogRefreshArtifactPath(ref catalog.RefreshableSpecReference, content []byte) (string, error) {
	if path := strings.TrimSpace(ref.RegisteredArtifactPath); path != "" {
		cleaned, err := cleanCatalogRefreshArtifactPath(path)
		if err != nil {
			return "", err
		}
		return normalizeCatalogRefreshRegisteredArtifactPath(ref, cleaned), nil
	}
	dir := ref.ProtocolClassification().SourceAlignedArtifactDir()
	if dir == "" {
		dir = "openapi"
	}
	ext := catalogRefreshExtension(ref.URL, content)
	return cleanCatalogRefreshArtifactPath(filepath.ToSlash(filepath.Join(dir, ref.SpecRefID+ext)))
}

func normalizeCatalogRefreshRegisteredArtifactPath(ref catalog.RefreshableSpecReference, path string) string {
	dir := ref.ProtocolClassification().SourceAlignedArtifactDir()
	if dir == "" || dir == "openapi" {
		return path
	}
	if strings.HasPrefix(path, "openapi/") {
		return dir + strings.TrimPrefix(path, "openapi")
	}
	return path
}

func catalogRefreshExtension(rawURL string, content []byte) string {
	lowerPath := strings.ToLower(strings.TrimSpace(rawURL))
	if strings.HasSuffix(lowerPath, ".tar.gz") || strings.HasSuffix(lowerPath, ".tgz") {
		return ".tar.gz"
	}
	switch ext := strings.ToLower(filepath.Ext(lowerPath)); ext {
	case ".json", ".yaml", ".yml":
		return ext
	}
	if len(bytes.TrimSpace(content)) > 0 && bytes.TrimSpace(content)[0] == '{' {
		return ".json"
	}
	return ".yaml"
}

func cleanCatalogRefreshArtifactPath(path string) (string, error) {
	cleaned := filepath.ToSlash(filepath.Clean(strings.TrimSpace(path)))
	if cleaned == "." || cleaned == "" {
		return "", fmt.Errorf("artifact path is required")
	}
	if filepath.IsAbs(cleaned) || strings.HasPrefix(cleaned, "../") || cleaned == ".." || strings.Contains(cleaned, "/../") {
		return "", fmt.Errorf("artifact path %q must stay under catalog cache directory", path)
	}
	if !strings.HasPrefix(cleaned, "openapi/") && !strings.HasPrefix(cleaned, "google-discovery/") && !strings.HasPrefix(cleaned, "aws-smithy/") {
		return "", fmt.Errorf("artifact path %q must be under openapi/, google-discovery/, or aws-smithy/", path)
	}
	return cleaned, nil
}

func writeCatalogRefreshArtifact(cacheDir, artifactPath string, content []byte) (string, error) {
	fullPath := filepath.Join(cacheDir, filepath.FromSlash(artifactPath))
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(fullPath, content, 0o644); err != nil {
		return "", err
	}
	return fullPath, nil
}
