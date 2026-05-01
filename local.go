package apitools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type LocalOptions struct {
	Dir     string
	BaseDir string
	Query   string
}

type LocalResult struct {
	Path         string       `json:"path"`
	RelativePath string       `json:"relative_path"`
	Title        string       `json:"title,omitempty"`
	Description  string       `json:"description,omitempty"`
	Score        int          `json:"score"`
	Metadata     SpecMetadata `json:"metadata"`
}

// LocalFiles discovers OpenAPI or Swagger documents already present on disk.
// Invalid documents are ignored so callers can scan mixed project directories.
func LocalFiles(ctx context.Context, opts LocalOptions) ([]LocalResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	dir := strings.TrimSpace(opts.Dir)
	if dir == "" {
		return nil, fmt.Errorf("local OpenAPI directory is required")
	}
	baseDir := strings.TrimSpace(opts.BaseDir)
	if baseDir == "" {
		baseDir = dir
	}
	var results []LocalResult
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
		if d.IsDir() || !hasOpenAPIFileExt(path) {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		metadata, ok := localSpecMetadata(ctx, content)
		if !ok {
			return nil
		}
		rel, err := filepath.Rel(baseDir, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		results = append(results, LocalResult{
			Path:         path,
			RelativePath: rel,
			Title:        metadata.Title,
			Description:  metadata.Description,
			Score:        ScoreText(opts.Query, metadata.Title+" "+metadata.Description+" "+rel),
			Metadata:     metadata,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.SliceStable(results, func(i, j int) bool {
		if results[i].Score != results[j].Score {
			return results[i].Score > results[j].Score
		}
		return results[i].RelativePath < results[j].RelativePath
	})
	return results, nil
}

func localSpecMetadata(ctx context.Context, content []byte) (SpecMetadata, bool) {
	if metadata, ok := specMetadata(ctx, content); ok {
		return metadata, true
	}
	return looseSpecMetadata(content)
}

func looseSpecMetadata(content []byte) (SpecMetadata, bool) {
	trimmed := bytes.TrimSpace(content)
	if len(trimmed) == 0 {
		return SpecMetadata{}, false
	}
	var root struct {
		OpenAPI string `json:"openapi" yaml:"openapi"`
		Swagger string `json:"swagger" yaml:"swagger"`
		Info    struct {
			Title       string `json:"title" yaml:"title"`
			Description string `json:"description" yaml:"description"`
		} `json:"info" yaml:"info"`
	}
	if trimmed[0] == '{' {
		if err := json.Unmarshal(trimmed, &root); err != nil {
			return SpecMetadata{}, false
		}
	} else if err := yaml.Unmarshal(trimmed, &root); err != nil {
		return SpecMetadata{}, false
	}
	if strings.TrimSpace(root.OpenAPI) == "" && strings.TrimSpace(root.Swagger) == "" {
		return SpecMetadata{}, false
	}
	return SpecMetadata{
		Title:       strings.TrimSpace(root.Info.Title),
		Description: strings.TrimSpace(root.Info.Description),
		OpenAPI:     strings.TrimSpace(root.OpenAPI),
		Swagger:     strings.TrimSpace(root.Swagger),
	}, true
}

func hasOpenAPIFileExt(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".json", ".yaml", ".yml":
		return true
	default:
		return false
	}
}
