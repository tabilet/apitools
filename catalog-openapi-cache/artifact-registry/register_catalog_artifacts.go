//go:build ignore

package main

import (
	"context"
	"fmt"
	"os"
	"time"

	apitools "github.com/OpenUdon/apitools"
	"github.com/OpenUdon/apitools/sqlitecache"
)

const cachePath = "catalog-openapi-cache/cache.sqlite"

type specArtifact struct {
	providerID string
	artifactID string
	kind       string
	url        string
	path       string
	title      string
	openapi    string
	swagger    string
}

type overlayArtifact struct {
	providerID  string
	artifactID  string
	path        string
	builderPath string
	sourceURL   string
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	ctx := context.Background()
	cache, err := sqlitecache.Open(cachePath)
	if err != nil {
		return err
	}
	defer cache.Close()

	for _, spec := range specArtifacts {
		if err := registerSpec(ctx, cache, spec); err != nil {
			return err
		}
	}
	for _, overlay := range overlayArtifacts {
		if err := cache.StoreCatalogArtifact(ctx, sqlitecache.CatalogArtifact{
			ProviderID:  overlay.providerID,
			ArtifactID:  overlay.artifactID,
			Kind:        "advisory-overlay",
			Path:        overlay.path,
			SourceURL:   overlay.sourceURL,
			OverlayPath: overlay.path,
			BuilderPath: overlay.builderPath,
			Metadata: map[string]string{
				"official_openapi":  "false",
				"derived_from_docs": "true",
			},
		}); err != nil {
			return err
		}
	}
	return nil
}

func registerSpec(ctx context.Context, cache *sqlitecache.Cache, artifact specArtifact) error {
	cached, ok, err := cache.LoadSpec(ctx, artifact.url, 100*365*24*time.Hour)
	if err != nil {
		return err
	}
	metadata := cached.Metadata
	finalURL := cached.FinalURL
	if !ok {
		finalURL = artifact.url
	}
	if metadata.Title == "" {
		metadata.Title = artifact.title
	}
	if metadata.OpenAPI == "" {
		metadata.OpenAPI = artifact.openapi
	}
	if metadata.Swagger == "" {
		metadata.Swagger = artifact.swagger
	}
	if finalURL == "" {
		finalURL = artifact.url
	}
	if err := cache.StoreSpec(ctx, apitools.CachedSpec{
		OriginalURL: artifact.url,
		FinalURL:    finalURL,
		ContentPath: artifact.path,
		Metadata:    metadata,
	}); err != nil {
		return err
	}
	return cache.StoreCatalogArtifact(ctx, sqlitecache.CatalogArtifact{
		ProviderID: artifact.providerID,
		ArtifactID: artifact.artifactID,
		Kind:       artifact.kind,
		Path:       artifact.path,
		SourceURL:  artifact.url,
		Metadata: map[string]string{
			"official": "true",
			"kind":     artifact.kind,
		},
	})
}

var specArtifacts = []specArtifact{
	{
		providerID: "asana",
		artifactID: "asana-openapi-v1",
		kind:       "openapi",
		url:        "https://raw.githubusercontent.com/Asana/openapi/master/defs/asana_oas.yaml",
		path:       "openapi/asana-openapi-v1.yaml",
		title:      "Asana",
		openapi:    "3.0.0",
	},
	{
		providerID: "box",
		artifactID: "box-platform-openapi-v3",
		kind:       "openapi",
		url:        "https://raw.githubusercontent.com/box/box-openapi/main/openapi.json",
		path:       "openapi/box-platform-openapi-v3.json",
		title:      "Box Platform API",
		openapi:    "3.0.2",
	},
	{
		providerID: "clickup",
		artifactID: "clickup-api-v2-openapi",
		kind:       "openapi",
		url:        "https://developer.clickup.com/openapi/clickup-api-v2-reference.json",
		path:       "openapi/clickup-api-v2-reference.json",
		title:      "ClickUp API v2 Reference",
		openapi:    "3.1.0",
	},
	{
		providerID: "discord",
		artifactID: "discord-api-v10-openapi",
		kind:       "openapi",
		url:        "https://raw.githubusercontent.com/discord/discord-api-spec/main/specs/openapi.json",
		path:       "openapi/discord-api-v10-openapi.json",
		title:      "Discord HTTP API (Preview)",
		openapi:    "3.1.0",
	},
	{
		providerID: "dropbox",
		artifactID: "dropbox-api-stone-spec",
		kind:       "dropbox-stone",
		url:        "https://github.com/dropbox/dropbox-api-spec/archive/refs/heads/main.tar.gz",
		path:       "openapi/dropbox-api-spec-main.tar.gz",
		title:      "Dropbox API Stone Spec",
	},
	{
		providerID: "github",
		artifactID: "github-rest-api-openapi",
		kind:       "openapi",
		url:        "https://raw.githubusercontent.com/github/rest-api-description/main/descriptions/api.github.com/api.github.com.json",
		path:       "openapi/github-rest-api-openapi.json",
		title:      "GitHub v3 REST API",
		openapi:    "3.0.3",
	},
	{
		providerID: "gitlab",
		artifactID: "gitlab-openapi-v2",
		kind:       "swagger",
		url:        "https://gitlab.com/gitlab-org/gitlab/-/raw/master/doc/api/openapi/openapi_v2.yaml",
		path:       "openapi/gitlab-openapi-v2.yaml",
		title:      "GitLab API",
		swagger:    "2.0",
	},
	{
		providerID: "gmail",
		artifactID: "gmail-discovery-v1",
		kind:       "google-discovery",
		url:        "https://gmail.googleapis.com/$discovery/rest?version=v1",
		path:       "google-discovery/gmail-discovery-v1.json",
	},
	{
		providerID: "google-drive",
		artifactID: "drive-discovery-v3",
		kind:       "google-discovery",
		url:        "https://www.googleapis.com/discovery/v1/apis/drive/v3/rest",
		path:       "google-discovery/drive-discovery-v3.json",
	},
	{
		providerID: "hubspot",
		artifactID: "hubspot-public-api-spec-index",
		kind:       "openapi-index",
		url:        "https://api.hubapi.com/public/api/spec/v1/specs",
		path:       "openapi/hubspot-public-api-spec-index.json",
	},
	{
		providerID: "jira-cloud",
		artifactID: "jira-cloud-platform-openapi-v3",
		kind:       "openapi",
		url:        "https://dac-static.atlassian.com/cloud/jira/platform/swagger-v3.v3.json",
		path:       "openapi/jira-cloud-platform-openapi-v3.json",
	},
	{
		providerID: "pagerduty",
		artifactID: "pagerduty-rest-openapi-v3",
		kind:       "openapi",
		url:        "https://raw.githubusercontent.com/PagerDuty/api-schema/main/reference/REST/openapiv3.json",
		path:       "openapi/pagerduty-rest-openapi-v3.json",
	},
	{
		providerID: "slack",
		artifactID: "slack-web-openapi-v2",
		kind:       "swagger",
		url:        "https://raw.githubusercontent.com/slackapi/slack-api-specs/master/web-api/slack_web_openapi_v2_without_examples.json",
		path:       "openapi/slack-web-openapi-v2.json",
	},
	{
		providerID: "trello",
		artifactID: "trello-cloud-openapi-v3",
		kind:       "openapi",
		url:        "https://dac-static.atlassian.com/cloud/trello/swagger.v3.json",
		path:       "openapi/trello-cloud-openapi-v3.json",
	},
}

var overlayArtifacts = []overlayArtifact{
	{
		providerID:  "airtable",
		artifactID:  "airtable-web-api-overlay",
		path:        "advisory-overlays/airtable-web-api-overlay.json",
		builderPath: "overlay-builders/build_airtable_overlay.go",
		sourceURL:   "https://airtable.com/developers/web/api/introduction",
	},
	{
		providerID:  "calendly",
		artifactID:  "calendly-public-api-overlay",
		path:        "advisory-overlays/calendly-public-api-overlay.json",
		builderPath: "overlay-builders/build_calendly_overlay.go",
		sourceURL:   "https://developer.calendly.com/api-docs",
	},
	{
		providerID:  "dropbox",
		artifactID:  "dropbox-core-api-overlay",
		path:        "advisory-overlays/dropbox-core-api-overlay.json",
		builderPath: "overlay-builders/build_dropbox_overlay.go",
		sourceURL:   "https://www.dropbox.com/developers/documentation/http/documentation",
	},
	{
		providerID:  "openweathermap",
		artifactID:  "openweathermap-one-call-3-overlay",
		path:        "advisory-overlays/openweathermap-one-call-3-overlay.json",
		builderPath: "overlay-builders/build_openweathermap_overlay.go",
		sourceURL:   "https://openweathermap.org/api/one-call-3?collection=one_call_api_3.0",
	},
}
