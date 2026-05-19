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
	status     string
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
	artifactMetadata := map[string]string{
		"official": "true",
		"kind":     artifact.kind,
	}
	if artifact.status != "" {
		artifactMetadata["validation_status"] = artifact.status
	}
	return cache.StoreCatalogArtifact(ctx, sqlitecache.CatalogArtifact{
		ProviderID: artifact.providerID,
		ArtifactID: artifact.artifactID,
		Kind:       artifact.kind,
		Path:       artifact.path,
		SourceURL:  artifact.url,
		Metadata:   artifactMetadata,
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
		status:     apitools.CatalogRefreshParseableOpenAPIInvalid,
	},
	{
		providerID: "aws-lambda",
		artifactID: "aws-lambda-smithy-model",
		kind:       "smithy-json",
		url:        "https://raw.githubusercontent.com/aws/api-models-aws/main/models/lambda/service/2015-03-31/lambda-2015-03-31.json",
		path:       "openapi/aws-lambda-smithy-model.json",
		title:      "AWS Lambda Smithy Model",
	},
	{
		providerID: "aws-s3",
		artifactID: "aws-s3-smithy-model",
		kind:       "smithy-json",
		url:        "https://raw.githubusercontent.com/aws/api-models-aws/main/models/s3/service/2006-03-01/s3-2006-03-01.json",
		path:       "openapi/aws-s3-smithy-model.json",
		title:      "Amazon S3 Smithy Model",
	},
	{
		providerID: "aws-sns",
		artifactID: "aws-sns-smithy-model",
		kind:       "smithy-json",
		url:        "https://raw.githubusercontent.com/aws/api-models-aws/main/models/sns/service/2010-03-31/sns-2010-03-31.json",
		path:       "openapi/aws-sns-smithy-model.json",
		title:      "Amazon SNS Smithy Model",
	},
	{
		providerID: "bitbucket",
		artifactID: "bitbucket-cloud-swagger-v2",
		kind:       "openapi",
		url:        "https://api.bitbucket.org/swagger.json",
		path:       "openapi/bitbucket-cloud-swagger-v2.json",
		title:      "Bitbucket API",
		swagger:    "2.0",
		status:     apitools.CatalogRefreshParseableSwaggerInvalid,
	},
	{
		providerID: "box",
		artifactID: "box-platform-openapi-v3",
		kind:       "openapi",
		url:        "https://raw.githubusercontent.com/box/box-openapi/main/openapi.json",
		path:       "openapi/box-platform-openapi-v3.json",
		title:      "Box Platform API",
		openapi:    "3.0.2",
		status:     apitools.CatalogRefreshParseableOpenAPIInvalid,
	},
	{
		providerID: "clickup",
		artifactID: "clickup-api-v2-openapi",
		kind:       "openapi",
		url:        "https://developer.clickup.com/openapi/clickup-api-v2-reference.json",
		path:       "openapi/clickup-api-v2-reference.json",
		title:      "ClickUp API v2 Reference",
		openapi:    "3.1.0",
		status:     apitools.CatalogRefreshParseableOpenAPIInvalid,
	},
	{
		providerID: "circleci",
		artifactID: "circleci-api-v2-openapi",
		kind:       "openapi",
		url:        "https://circleci.com/api/v2/openapi.json",
		path:       "openapi/circleci-api-v2-openapi.json",
		title:      "CircleCI API",
		openapi:    "3.0.3",
		status:     apitools.CatalogRefreshParseableOpenAPIInvalid,
	},
	{
		providerID: "cloudflare",
		artifactID: "cloudflare-api-openapi",
		kind:       "openapi",
		url:        "https://raw.githubusercontent.com/cloudflare/api-schemas/main/openapi.yaml",
		path:       "openapi/cloudflare-api-openapi.yaml",
		title:      "Cloudflare API",
		openapi:    "3.0.3",
		status:     apitools.CatalogRefreshParseableOpenAPIInvalid,
	},
	{
		providerID: "chargebee",
		artifactID: "chargebee-api-v2-pc-v2-openapi",
		kind:       "openapi",
		url:        "https://raw.githubusercontent.com/chargebee/openapi/main/spec/chargebee_api_v2_pc_v2_spec.json",
		path:       "openapi/chargebee-api-v2-pc-v2-openapi.json",
		title:      "Chargebee API",
		openapi:    "3.1.0",
	},
	{
		providerID: "customer-io",
		artifactID: "customer-io-journeys-app-openapi",
		kind:       "openapi",
		url:        "https://docs.customer.io/files/journeys-app.json",
		path:       "openapi/customer-io-journeys-app-openapi.json",
		title:      "Journeys App API",
		openapi:    "3.1.0",
		status:     apitools.CatalogRefreshParseableOpenAPIInvalid,
	},
	{
		providerID: "customer-io",
		artifactID: "customer-io-journeys-track-openapi",
		kind:       "openapi",
		url:        "https://docs.customer.io/files/journeys-track.json",
		path:       "openapi/customer-io-journeys-track-openapi.json",
		title:      "Journeys Track API",
		openapi:    "3.1.0",
		status:     apitools.CatalogRefreshParseableOpenAPIInvalid,
	},
	{
		providerID: "customer-io",
		artifactID: "customer-io-pipelines-openapi",
		kind:       "openapi",
		url:        "https://docs.customer.io/files/pipelines.json",
		path:       "openapi/customer-io-pipelines-openapi.json",
		title:      "Pipelines API",
		openapi:    "3.1.0",
		status:     apitools.CatalogRefreshParseableOpenAPIInvalid,
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
		providerID: "grafana",
		artifactID: "grafana-http-api-openapi-v3",
		kind:       "openapi",
		url:        "https://raw.githubusercontent.com/grafana/grafana/main/public/openapi3.json",
		path:       "openapi/grafana-http-api-openapi-v3.json",
		title:      "Grafana HTTP API.",
		openapi:    "3.0.3",
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
		status:     apitools.CatalogRefreshParseableOpenAPIInvalid,
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
		providerID: "google-calendar",
		artifactID: "calendar-discovery-v3",
		kind:       "google-discovery",
		url:        "https://www.googleapis.com/discovery/v1/apis/calendar/v3/rest",
		path:       "google-discovery/google-calendar-discovery-v3.json",
		title:      "Calendar API",
	},
	{
		providerID: "google-drive",
		artifactID: "drive-discovery-v3",
		kind:       "google-discovery",
		url:        "https://www.googleapis.com/discovery/v1/apis/drive/v3/rest",
		path:       "google-discovery/drive-discovery-v3.json",
	},
	{
		providerID: "google-sheets",
		artifactID: "sheets-discovery-v4",
		kind:       "google-discovery",
		url:        "https://sheets.googleapis.com/$discovery/rest?version=v4",
		path:       "google-discovery/google-sheets-discovery-v4.json",
		title:      "Google Sheets API",
	},
	{
		providerID: "hubspot",
		artifactID: "hubspot-public-api-spec-index",
		kind:       "openapi-index",
		url:        "https://api.hubapi.com/public/api/spec/v1/specs",
		path:       "openapi/hubspot-public-api-spec-index.json",
	},
	{
		providerID: "intercom",
		artifactID: "intercom-api-v2-15-openapi",
		kind:       "openapi",
		url:        "https://developers.intercom.com/_bundle/docs/references/%402.15/rest-api/api.intercom.io.json?download=",
		path:       "openapi/intercom-api-v2-15-openapi.json",
		title:      "Intercom API",
		openapi:    "3.0.1",
		status:     apitools.CatalogRefreshParseableOpenAPIInvalid,
	},
	{
		providerID: "mailgun",
		artifactID: "mailgun-send-api-openapi",
		kind:       "openapi",
		url:        "https://documentation.mailgun.com/_bundle/docs/mailgun/api-reference/send/mailgun.json",
		path:       "openapi/mailgun-send-api-openapi.json",
		title:      "Mailgun API",
		openapi:    "3.1.0",
	},
	{
		providerID: "jira-cloud",
		artifactID: "jira-cloud-platform-openapi-v3",
		kind:       "openapi",
		url:        "https://dac-static.atlassian.com/cloud/jira/platform/swagger-v3.v3.json",
		path:       "openapi/jira-cloud-platform-openapi-v3.json",
		status:     apitools.CatalogRefreshParseableOpenAPIInvalid,
	},
	{
		providerID: "microsoft-graph",
		artifactID: "microsoft-graph-v1-openapi",
		kind:       "openapi",
		url:        "https://raw.githubusercontent.com/microsoftgraph/msgraph-metadata/master/openapi/v1.0/openapi.yaml",
		path:       "openapi/microsoft-graph-v1-openapi.yaml",
		title:      "OData Service for namespace microsoft.graph",
		openapi:    "3.0.4",
	},
	{
		providerID: "netlify",
		artifactID: "netlify-api-openapi",
		kind:       "openapi",
		url:        "https://open-api.netlify.com/openapi.json",
		path:       "openapi/netlify-api-openapi.json",
		title:      "Netlify's API documentation",
		openapi:    "3.0.0",
	},
	{
		providerID: "notion",
		artifactID: "notion-api-openapi",
		kind:       "openapi",
		url:        "https://developers.notion.com/openapi.json",
		path:       "openapi/notion-api-openapi.json",
		title:      "Notion API",
		openapi:    "3.1.0",
		status:     apitools.CatalogRefreshParseableOpenAPIInvalid,
	},
	{
		providerID: "okta",
		artifactID: "okta-management-minimal-openapi",
		kind:       "openapi",
		url:        "https://raw.githubusercontent.com/okta/okta-management-openapi-spec/master/dist/current/management-minimal.yaml",
		path:       "openapi/okta-management-minimal-openapi.yaml",
		title:      "Okta Admin Management API",
		openapi:    "3.0.3",
		status:     apitools.CatalogRefreshParseableOpenAPIInvalid,
	},
	{
		providerID: "pagerduty",
		artifactID: "pagerduty-rest-openapi-v3",
		kind:       "openapi",
		url:        "https://raw.githubusercontent.com/PagerDuty/api-schema/main/reference/REST/openapiv3.json",
		path:       "openapi/pagerduty-rest-openapi-v3.json",
		status:     apitools.CatalogRefreshParseableOpenAPIInvalid,
	},
	{
		providerID: "paypal",
		artifactID: "paypal-checkout-orders-v2-openapi",
		kind:       "openapi",
		url:        "https://raw.githubusercontent.com/paypal/paypal-rest-api-specifications/main/openapi/checkout_orders_v2.json",
		path:       "openapi/paypal-checkout-orders-v2-openapi.json",
		title:      "Orders",
		openapi:    "3.0.4",
	},
	{
		providerID: "pipedrive",
		artifactID: "pipedrive-api-v2-openapi",
		kind:       "openapi",
		url:        "https://developers.pipedrive.com/docs/api/v1/openapi-v2.yaml",
		path:       "openapi/pipedrive-api-v2-openapi.yaml",
		title:      "Pipedrive API v2",
		openapi:    "3.0.1",
		status:     apitools.CatalogRefreshParseableOpenAPIInvalid,
	},
	{
		providerID: "sendgrid",
		artifactID: "sendgrid-mail-v3-openapi",
		kind:       "openapi",
		url:        "https://raw.githubusercontent.com/twilio/sendgrid-oai/main/spec/json/tsg_mail_v3.json",
		path:       "openapi/sendgrid-mail-v3-openapi.json",
		title:      "Twilio SendGrid Mail API",
		openapi:    "3.1.0",
	},
	{
		providerID: "slack",
		artifactID: "slack-web-openapi-v2",
		kind:       "swagger",
		url:        "https://raw.githubusercontent.com/slackapi/slack-api-specs/master/web-api/slack_web_openapi_v2_without_examples.json",
		path:       "openapi/slack-web-openapi-v2.json",
		status:     apitools.CatalogRefreshParseableSwaggerInvalid,
	},
	{
		providerID: "snowflake",
		artifactID: "snowflake-sql-api-openapi",
		kind:       "openapi",
		url:        "https://raw.githubusercontent.com/snowflakedb/snowflake-rest-api-specs/main/specifications/sqlapi.yaml",
		path:       "openapi/snowflake-sql-api-openapi.yaml",
		title:      "Snowflake SQL API",
		openapi:    "3.0.0",
	},
	{
		providerID: "stripe",
		artifactID: "stripe-latest-openapi-spec3",
		kind:       "openapi",
		url:        "https://raw.githubusercontent.com/stripe/openapi/master/latest/openapi.spec3.json",
		path:       "openapi/stripe-latest-openapi-spec3.json",
		title:      "Stripe API",
		openapi:    "3.0.0",
	},
	{
		providerID: "supabase",
		artifactID: "supabase-management-api-openapi",
		kind:       "openapi",
		url:        "https://api.supabase.com/api/v1-json",
		path:       "openapi/supabase-management-api-openapi.json",
		title:      "Supabase API (v1)",
		openapi:    "3.0.0",
		status:     apitools.CatalogRefreshParseableOpenAPIInvalid,
	},
	{
		providerID: "trello",
		artifactID: "trello-cloud-openapi-v3",
		kind:       "openapi",
		url:        "https://dac-static.atlassian.com/cloud/trello/swagger.v3.json",
		path:       "openapi/trello-cloud-openapi-v3.json",
		status:     apitools.CatalogRefreshParseableOpenAPIInvalid,
	},
	{
		providerID: "twilio",
		artifactID: "twilio-api-v2010-openapi",
		kind:       "openapi",
		url:        "https://raw.githubusercontent.com/twilio/twilio-oai/main/spec/json/twilio_api_v2010.json",
		path:       "openapi/twilio-api-v2010-openapi.json",
		title:      "Twilio - Api",
		openapi:    "3.0.1",
	},
	{
		providerID: "xero",
		artifactID: "xero-accounting-openapi",
		kind:       "openapi",
		url:        "https://raw.githubusercontent.com/XeroAPI/Xero-OpenAPI/master/xero_accounting.yaml",
		path:       "openapi/xero-accounting-openapi.yaml",
		title:      "Xero Accounting API",
		openapi:    "3.0.0",
	},
	{
		providerID: "zoom",
		artifactID: "zoom-api-v2-openapi",
		kind:       "swagger",
		url:        "https://raw.githubusercontent.com/zoom/api/master/openapi.v2.json",
		path:       "openapi/zoom-api-v2-openapi.json",
		title:      "Zoom API",
		swagger:    "2.0",
	},
	{
		providerID: "zendesk",
		artifactID: "zendesk-sunshine-conversations-openapi",
		kind:       "openapi",
		url:        "https://raw.githubusercontent.com/zendesk/sunshine-conversations-api-spec/master/openapi.yaml",
		path:       "openapi/zendesk-sunshine-conversations-openapi.yaml",
		title:      "Sunshine Conversations API",
		openapi:    "3.0.2",
		status:     apitools.CatalogRefreshParseableOpenAPIInvalid,
	},
}

var overlayArtifacts = []overlayArtifact{
	{
		providerID:  "activecampaign",
		artifactID:  "activecampaign-api-v3-overlay",
		path:        "advisory-overlays/activecampaign-api-v3-overlay.json",
		builderPath: "overlay-builders/build_m21_human_docs_overlays.go",
		sourceURL:   "https://developers.activecampaign.com/reference/overview",
	},
	{
		providerID:  "airtable",
		artifactID:  "airtable-web-api-overlay",
		path:        "advisory-overlays/airtable-web-api-overlay.json",
		builderPath: "overlay-builders/build_airtable_overlay.go",
		sourceURL:   "https://airtable.com/developers/web/api/introduction",
	},
	{
		providerID:  "bamboohr",
		artifactID:  "bamboohr-api-v1-overlay",
		path:        "advisory-overlays/bamboohr-api-v1-overlay.json",
		builderPath: "overlay-builders/build_m21_human_docs_overlays.go",
		sourceURL:   "https://documentation.bamboohr.com/docs",
	},
	{
		providerID:  "calendly",
		artifactID:  "calendly-public-api-overlay",
		path:        "advisory-overlays/calendly-public-api-overlay.json",
		builderPath: "overlay-builders/build_calendly_overlay.go",
		sourceURL:   "https://developer.calendly.com/api-docs",
	},
	{
		providerID:  "contentful",
		artifactID:  "contentful-management-api-overlay",
		path:        "advisory-overlays/contentful-management-api-overlay.json",
		builderPath: "overlay-builders/build_m21_human_docs_overlays.go",
		sourceURL:   "https://www.contentful.com/developers/docs/references/content-management-api/",
	},
	{
		providerID:  "databricks",
		artifactID:  "databricks-workspace-rest-overlay",
		path:        "advisory-overlays/databricks-workspace-rest-overlay.json",
		builderPath: "overlay-builders/build_m20_human_docs_overlays.go",
		sourceURL:   "https://docs.databricks.com/api/workspace/introduction",
	},
	{
		providerID:  "dropbox",
		artifactID:  "dropbox-core-api-overlay",
		path:        "advisory-overlays/dropbox-core-api-overlay.json",
		builderPath: "overlay-builders/build_dropbox_overlay.go",
		sourceURL:   "https://www.dropbox.com/developers/documentation/http/documentation",
	},
	{
		providerID:  "eventbrite",
		artifactID:  "eventbrite-platform-api-v3-overlay",
		path:        "advisory-overlays/eventbrite-platform-api-v3-overlay.json",
		builderPath: "overlay-builders/build_m21_human_docs_overlays.go",
		sourceURL:   "https://www.eventbrite.com/platform/api",
	},
	{
		providerID:  "freshdesk",
		artifactID:  "freshdesk-api-v2-overlay",
		path:        "advisory-overlays/freshdesk-api-v2-overlay.json",
		builderPath: "overlay-builders/build_m21_human_docs_overlays.go",
		sourceURL:   "https://developers.freshdesk.com/api",
	},
	{
		providerID:  "google-calendar",
		artifactID:  "google-calendar-v3-overlay",
		path:        "advisory-overlays/google-calendar-v3-overlay.json",
		builderPath: "overlay-builders/build_google_calendar_overlay.go",
		sourceURL:   "https://developers.google.com/workspace/calendar/api/v3/reference",
	},
	{
		providerID:  "google-sheets",
		artifactID:  "google-sheets-v4-overlay",
		path:        "advisory-overlays/google-sheets-v4-overlay.json",
		builderPath: "overlay-builders/build_google_sheets_overlay.go",
		sourceURL:   "https://developers.google.com/workspace/sheets/api/reference/rest",
	},
	{
		providerID:  "jenkins",
		artifactID:  "jenkins-remote-api-overlay",
		path:        "advisory-overlays/jenkins-remote-api-overlay.json",
		builderPath: "overlay-builders/build_m20_human_docs_overlays.go",
		sourceURL:   "https://www.jenkins.io/doc/book/using/remote-access-api/",
	},
	{
		providerID:  "mailchimp",
		artifactID:  "mailchimp-marketing-api-overlay",
		path:        "advisory-overlays/mailchimp-marketing-api-overlay.json",
		builderPath: "overlay-builders/build_mailchimp_overlay.go",
		sourceURL:   "https://mailchimp.com/developer/marketing/api/",
	},
	{
		providerID:  "openweathermap",
		artifactID:  "openweathermap-one-call-3-overlay",
		path:        "advisory-overlays/openweathermap-one-call-3-overlay.json",
		builderPath: "overlay-builders/build_openweathermap_overlay.go",
		sourceURL:   "https://openweathermap.org/api/one-call-3?collection=one_call_api_3.0",
	},
	{
		providerID:  "postmark",
		artifactID:  "postmark-api-overlay",
		path:        "advisory-overlays/postmark-api-overlay.json",
		builderPath: "overlay-builders/build_m21_human_docs_overlays.go",
		sourceURL:   "https://postmarkapp.com/developer/api/overview",
	},
	{
		providerID:  "quickbooks",
		artifactID:  "quickbooks-online-accounting-api-overlay",
		path:        "advisory-overlays/quickbooks-online-accounting-api-overlay.json",
		builderPath: "overlay-builders/build_quickbooks_overlay.go",
		sourceURL:   "https://developer.intuit.com/app/developer/qbo/docs/learn/explore-the-quickbooks-online-api",
	},
	{
		providerID:  "salesforce",
		artifactID:  "salesforce-rest-core-overlay",
		path:        "advisory-overlays/salesforce-rest-core-overlay.json",
		builderPath: "overlay-builders/build_salesforce_overlay.go",
		sourceURL:   "https://developer.salesforce.com/docs/atlas.en-us.api_rest.meta/api_rest/intro_rest.htm",
	},
	{
		providerID:  "sentry",
		artifactID:  "sentry-rest-api-overlay",
		path:        "advisory-overlays/sentry-rest-api-overlay.json",
		builderPath: "overlay-builders/build_m20_human_docs_overlays.go",
		sourceURL:   "https://docs.sentry.io/api/",
	},
	{
		providerID:  "servicenow",
		artifactID:  "servicenow-rest-api-overlay",
		path:        "advisory-overlays/servicenow-rest-api-overlay.json",
		builderPath: "overlay-builders/build_servicenow_overlay.go",
		sourceURL:   "https://www.servicenow.com/docs/r/api-reference/rest-api-explorer/c_RESTAPI.html",
	},
	{
		providerID:  "shopify",
		artifactID:  "shopify-admin-rest-overlay",
		path:        "advisory-overlays/shopify-admin-rest-overlay.json",
		builderPath: "overlay-builders/build_shopify_overlay.go",
		sourceURL:   "https://shopify.dev/docs/api/admin-rest",
	},
	{
		providerID:  "splunk",
		artifactID:  "splunk-enterprise-rest-overlay",
		path:        "advisory-overlays/splunk-enterprise-rest-overlay.json",
		builderPath: "overlay-builders/build_m20_human_docs_overlays.go",
		sourceURL:   "https://docs.splunk.com/Documentation/Splunk/latest/RESTREF",
	},
	{
		providerID:  "telegram",
		artifactID:  "telegram-bot-api-overlay",
		path:        "advisory-overlays/telegram-bot-api-overlay.json",
		builderPath: "overlay-builders/build_m20_human_docs_overlays.go",
		sourceURL:   "https://core.telegram.org/bots/api",
	},
	{
		providerID:  "todoist",
		artifactID:  "todoist-rest-api-v2-overlay",
		path:        "advisory-overlays/todoist-rest-api-v2-overlay.json",
		builderPath: "overlay-builders/build_m21_human_docs_overlays.go",
		sourceURL:   "https://developer.todoist.com/rest/v2/",
	},
	{
		providerID:  "typeform",
		artifactID:  "typeform-rest-api-overlay",
		path:        "advisory-overlays/typeform-rest-api-overlay.json",
		builderPath: "overlay-builders/build_typeform_overlay.go",
		sourceURL:   "https://www.typeform.com/developers/get-started/",
	},
	{
		providerID:  "webflow",
		artifactID:  "webflow-data-api-v2-overlay",
		path:        "advisory-overlays/webflow-data-api-v2-overlay.json",
		builderPath: "overlay-builders/build_m21_human_docs_overlays.go",
		sourceURL:   "https://developers.webflow.com/data/reference",
	},
}
