package catalog

import (
	"fmt"
	"sort"
	"strings"
)

// SecurityClassification records source-backed auth/security completeness for
// providers whose upstream machine specs are already sufficient or still need
// review.
type SecurityClassification struct {
	ProviderID string                 `json:"provider_id"`
	SpecRefID  string                 `json:"spec_ref_id,omitempty"`
	Status     AuthCompletenessStatus `json:"status"`
	SourceRefs []string               `json:"source_refs,omitempty"`
	SourceNote string                 `json:"source_note"`
}

// SecurityReport is a deterministic metadata-only auth/security report for a
// provider catalog.
type SecurityReport struct {
	Providers []ProviderSecurityReport `json:"providers,omitempty"`
}

// ProviderSecurityReport records the effective catalog security status for one
// provider.
type ProviderSecurityReport struct {
	ProviderID  string                 `json:"provider_id"`
	Status      AuthCompletenessStatus `json:"status"`
	OverlayIDs  []string               `json:"overlay_ids,omitempty"`
	SpecRefIDs  []string               `json:"spec_ref_ids,omitempty"`
	SourceRefs  []string               `json:"source_refs,omitempty"`
	SourceNotes []string               `json:"source_notes,omitempty"`
}

// BuiltInSecurityClassifications returns source-backed security status records
// for built-in providers. Callers receive independent copies.
func BuiltInSecurityClassifications() []SecurityClassification {
	out := cloneSecurityClassifications(builtInSecurityClassifications)
	sortSecurityClassifications(out)
	return out
}

// BuiltInSecurityReport returns the built-in provider security report.
func BuiltInSecurityReport() (SecurityReport, error) {
	return BuildSecurityReport(BuiltInProviders(), BuiltInSecurityOverlays(), BuiltInSecurityClassifications())
}

// BuildSecurityReport combines provider metadata, security overlays, and
// source-backed classifications into a deterministic report.
func BuildSecurityReport(providers []Provider, overlays []SecurityOverlay, classifications []SecurityClassification) (SecurityReport, error) {
	if err := ValidateProviders(providers); err != nil {
		return SecurityReport{}, err
	}
	if err := ValidateSecurityOverlays(overlays, providers); err != nil {
		return SecurityReport{}, err
	}
	overlays = cloneSecurityOverlays(overlays)
	sortSecurityOverlays(overlays)
	providerByID := map[string]Provider{}
	for _, provider := range providers {
		providerByID[provider.ID] = provider
	}

	classificationByProvider := map[string]SecurityClassification{}
	for i, classification := range classifications {
		if err := validateSecurityClassification(classification, providerByID); err != nil {
			return SecurityReport{}, fmt.Errorf("security classification[%d]: %w", i, err)
		}
		if _, exists := classificationByProvider[classification.ProviderID]; exists {
			return SecurityReport{}, fmt.Errorf("security classification %q: duplicate provider", classification.ProviderID)
		}
		classificationByProvider[classification.ProviderID] = classification
	}

	overlaysByProvider := map[string][]SecurityOverlay{}
	for _, overlay := range overlays {
		overlaysByProvider[overlay.ProviderID] = append(overlaysByProvider[overlay.ProviderID], overlay)
	}

	var reports []ProviderSecurityReport
	for _, provider := range cloneProviders(providers) {
		report := ProviderSecurityReport{
			ProviderID: provider.ID,
			Status:     AuthStatusUnknown,
		}
		if classification, ok := classificationByProvider[provider.ID]; ok {
			report.Status = classification.Status
			report.SpecRefIDs = appendIfNotEmpty(report.SpecRefIDs, classification.SpecRefID)
			report.SourceRefs = append(report.SourceRefs, classification.SourceRefs...)
			report.SourceNotes = appendIfNotEmpty(report.SourceNotes, classification.SourceNote)
		}
		for _, overlay := range overlaysByProvider[provider.ID] {
			report.Status = overlay.Status
			report.OverlayIDs = appendIfNotEmpty(report.OverlayIDs, overlay.ID)
			report.SpecRefIDs = appendIfNotEmpty(report.SpecRefIDs, overlay.SpecRefID)
			report.SourceRefs = append(report.SourceRefs, overlay.SourceRefs...)
			report.SourceNotes = appendIfNotEmpty(report.SourceNotes, overlay.SourceNote)
		}
		report.OverlayIDs = sortedUniqueStrings(report.OverlayIDs)
		report.SpecRefIDs = sortedUniqueStrings(report.SpecRefIDs)
		report.SourceRefs = sortedUniqueStrings(report.SourceRefs)
		report.SourceNotes = sortedUniqueStrings(report.SourceNotes)
		reports = append(reports, report)
	}
	sort.SliceStable(reports, func(i, j int) bool {
		return reports[i].ProviderID < reports[j].ProviderID
	})
	return SecurityReport{Providers: reports}, nil
}

// FindProvider returns a provider security report by provider ID.
func (r SecurityReport) FindProvider(providerID string) (ProviderSecurityReport, bool) {
	normalized := normalizeKey(providerID)
	for _, provider := range r.Providers {
		if normalizeKey(provider.ProviderID) == normalized {
			return provider, true
		}
	}
	return ProviderSecurityReport{}, false
}

var builtInSecurityClassifications = []SecurityClassification{
	{
		ProviderID: "action-network",
		SpecRefID:  "action-network-api-v2-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://actionnetwork.org/docs/v2/"},
		SourceNote: "Action Network has official REST API v2 human docs but no recorded stable public official OpenAPI document; OSDI-API-Token metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "adalo",
		SpecRefID:  "adalo-api-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://help.adalo.com/integrations/the-adalo-api/collections", "https://help.adalo.com/integrations/the-adalo-api/push-notifications"},
		SourceNote: "Adalo has official API human docs and app-specific generated collection docs but no recorded stable public official OpenAPI document; bearer API-key metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "affinity",
		SpecRefID:  "affinity-v1-api-reference",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://api-docs.affinity.co/"},
		SourceNote: "Affinity has official V1 API human docs but no recorded stable public official OpenAPI document; basic and bearer API-key metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "agile-crm",
		SpecRefID:  "agile-crm-rest-api-github",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://www.agilecrm.com/api", "https://github.com/agilecrm/rest-api"},
		SourceNote: "Agile CRM has official API human docs and REST API GitHub documentation but no recorded stable public official OpenAPI document; email/API-key basic auth metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "asana",
		SpecRefID:  "asana-openapi-v1",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://raw.githubusercontent.com/Asana/openapi/master/defs/asana_oas.yaml"},
		SourceNote: "Asana's official OpenAPI document includes bearer personal access token and OAuth2 security schemes plus root and operation security requirements.",
	},
	{
		ProviderID: "airtable",
		SpecRefID:  "airtable-web-api-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://airtable.com/developers/web/api/introduction"},
		SourceNote: "Airtable has no recorded official OpenAPI document in the built-in catalog; security metadata must come from advisory overlay notes when importing a user-provided spec.",
	},
	{
		ProviderID: "bannerbear",
		SpecRefID:  "bannerbear-api-v2-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://developers.bannerbear.com/v2/"},
		SourceNote: "Bannerbear has official API v2 docs but no recorded stable public official OpenAPI document; bearer API-key metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "baserow",
		SpecRefID:  "baserow-api-openapi",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://api.baserow.io/api/schema.json", "https://baserow.io/docs/apis/rest-api", "https://baserow.io/user-docs/database-api"},
		SourceNote: "Baserow's official OpenAPI document includes database-token, JWT, and user-source JWT bearer security schemes with operation-level security requirements.",
	},
	{
		ProviderID: "beeminder",
		SpecRefID:  "beeminder-api-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://api.beeminder.com/", "https://www.beeminder.com/api/v1/auth_token.json", "https://www.beeminder.com/apps/new"},
		SourceNote: "Beeminder has official API human docs but no recorded stable public official OpenAPI document; personal auth_token and OAuth bearer metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "aws-s3",
		SpecRefID:  "aws-s3-smithy-model",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://raw.githubusercontent.com/aws/api-models-aws/main/models/s3/service/2006-03-01/s3-2006-03-01.json", "https://docs.aws.amazon.com/AmazonS3/latest/API/sig-v4-authenticating-requests.html"},
		SourceNote: "AWS S3 has an official Smithy JSON service model rather than OpenAPI; SigV4 signing requirements must be represented as advisory metadata only.",
	},
	{
		ProviderID: "aws-lambda",
		SpecRefID:  "aws-lambda-smithy-model",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://raw.githubusercontent.com/aws/api-models-aws/main/models/lambda/service/2015-03-31/lambda-2015-03-31.json", "https://docs.aws.amazon.com/lambda/latest/api/welcome.html"},
		SourceNote: "AWS Lambda has an official Smithy JSON service model rather than OpenAPI; SigV4 signing requirements must be represented as advisory metadata only.",
	},
	{
		ProviderID: "aws-sns",
		SpecRefID:  "aws-sns-smithy-model",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://raw.githubusercontent.com/aws/api-models-aws/main/models/sns/service/2010-03-31/sns-2010-03-31.json", "https://docs.aws.amazon.com/sns/latest/api/welcome.html"},
		SourceNote: "AWS SNS has an official Smithy JSON service model rather than OpenAPI; SigV4 signing requirements must be represented as advisory metadata only.",
	},
	{
		ProviderID: "box",
		SpecRefID:  "box-platform-openapi-v3",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://raw.githubusercontent.com/box/box-openapi/main/openapi.json"},
		SourceNote: "Box's official OpenAPI document includes an OAuth2 authorization code security scheme and root security requirement.",
	},
	{
		ProviderID: "bitbucket",
		SpecRefID:  "bitbucket-cloud-swagger-v2",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://api.bitbucket.org/swagger.json", "https://developer.atlassian.com/cloud/bitbucket/rest/intro/"},
		SourceNote: "Bitbucket Cloud's official Swagger/OpenAPI schema includes OAuth2, basic, and API key security definitions with operation security metadata.",
	},
	{
		ProviderID: "brandfetch",
		SpecRefID:  "brandfetch-brand-api-openapi",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://docs.brandfetch.com/openapi.json", "https://docs.brandfetch.com/brand-api/overview"},
		SourceNote: "Brandfetch's official Brand API OpenAPI document includes bearer HTTP security metadata with operation-level security requirements.",
	},
	{
		ProviderID: "calendly",
		SpecRefID:  "calendly-public-api-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://developer.calendly.com/api-docs"},
		SourceNote: "Calendly's official API reference is human documentation backed by Stoplight hosting in this catalog entry; no downloadable official OpenAPI document is recorded, so endpoint and security metadata come from advisory overlays.",
	},
	{
		ProviderID: "circleci",
		SpecRefID:  "circleci-api-v2-openapi",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://circleci.com/api/v2/openapi.json", "https://circleci.com/docs/api/v2/", "https://circleci.com/docs/guides/toolkit/managing-api-tokens/"},
		SourceNote: "CircleCI's official API v2 OpenAPI document includes Circle-Token header, basic-auth, and deprecated query-token security schemes with root security requirements.",
	},
	{
		ProviderID: "clearbit",
		SpecRefID:  "clearbit-prospector-zapier-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://help.clearbit.com/hc/en-us/articles/6480449602967-Integrate-Clearbit-Prospector-with-Google-Sheets-Using-Zapier", "https://help.clearbit.com/hc/en-us/articles/6045527495191-How-Do-I-Access-My-Clearbit-API-Key"},
		SourceNote: "Clearbit has official API support docs but no recorded stable public official OpenAPI document; secret API-key basic-auth metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "cloudflare",
		SpecRefID:  "cloudflare-api-openapi",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://raw.githubusercontent.com/cloudflare/api-schemas/main/openapi.yaml", "https://developers.cloudflare.com/api/"},
		SourceNote: "Cloudflare's official OpenAPI document includes API token, API key/email, and user service key security schemes with root and operation security requirements.",
	},
	{
		ProviderID: "databricks",
		SpecRefID:  "databricks-api-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://docs.databricks.com/api/workspace/introduction", "https://docs.databricks.com/aws/en/dev-tools/auth"},
		SourceNote: "Databricks has no recorded public official OpenAPI document for the full REST API in this catalog entry; auth metadata comes from advisory bearer-token overlay notes.",
	},
	{
		ProviderID: "clickup",
		SpecRefID:  "clickup-api-v2-openapi",
		Status:     AuthStatusPresentIncomplete,
		SourceRefs: []string{"https://developer.clickup.com/openapi/clickup-api-v2-reference.json"},
		SourceNote: "ClickUp's official OpenAPI document includes an Authorization header API key scheme and root security, but OAuth authorization-code flow details are documented separately rather than fully modeled in the security scheme.",
	},
	{
		ProviderID: "discord",
		SpecRefID:  "discord-api-v10-openapi",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://raw.githubusercontent.com/discord/discord-api-spec/main/specs/openapi.json"},
		SourceNote: "Discord's official OpenAPI v10 preview document includes BotToken and OAuth2 security schemes with operation-level security requirements.",
	},
	{
		ProviderID: "discourse",
		SpecRefID:  "discourse-api-openapi",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://docs.discourse.org/openapi.json", "https://docs.discourse.org/", "https://github.com/discourse/discourse_api_docs"},
		SourceNote: "Discourse's official OpenAPI document is importable but currently lacks OpenAPI securitySchemes; Api-Key and Api-Username header metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "dropbox",
		SpecRefID:  "dropbox-api-stone-spec",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://github.com/dropbox/dropbox-api-spec"},
		SourceNote: "Dropbox's official machine-readable source is a Stone spec, not OpenAPI; OAuth security metadata needs advisory OpenAPI overlay mapping for OpenAPI-only consumers.",
	},
	{
		ProviderID: "copper",
		SpecRefID:  "copper-developer-api-auth-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://developer.copper.com/introduction/authentication.html", "https://developer.copper.com/"},
		SourceNote: "Copper has official Developer API human docs but no recorded stable public official OpenAPI document; token and user-email header metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "elastic",
		SpecRefID:  "elastic-elasticsearch-openapi",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://www.elastic.co/docs/api/doc/elasticsearch.json", "https://www.elastic.co/docs/api/doc/elasticsearch/authentication"},
		SourceNote: "Elastic's official Elasticsearch OpenAPI document includes API key, basic, and bearer security schemes with root security requirements.",
	},
	{
		ProviderID: "github",
		SpecRefID:  "github-rest-api-openapi",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://raw.githubusercontent.com/github/rest-api-description/main/descriptions/api.github.com/api.github.com.json"},
		SourceNote: "GitHub's official OpenAPI document omits security schemes and requirements; official REST authentication docs need advisory bearer/basic overlay mapping.",
	},
	{
		ProviderID: "grafana",
		SpecRefID:  "grafana-http-api-openapi-v3",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://raw.githubusercontent.com/grafana/grafana/main/public/openapi3.json", "https://grafana.com/docs/grafana/latest/developer-resources/api-reference/"},
		SourceNote: "Grafana's official HTTP API OpenAPI v3 document includes API-key Authorization header and basic auth security schemes with root security requirements.",
	},
	{
		ProviderID: "grist",
		SpecRefID:  "grist-rest-api-usage-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://support.getgrist.com/rest-api/", "https://support.getgrist.com/api/"},
		SourceNote: "Grist has official REST API usage and reference docs but no recorded stable public official OpenAPI document; bearer API-key metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "freshservice",
		SpecRefID:  "freshservice-api-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://api.freshservice.com/", "https://support.freshservice.com/support/solutions/articles/50000012704-working-with-apis-in-freshservice"},
		SourceNote: "Freshservice has official API v2 human docs but no recorded stable public official OpenAPI document; API-key Basic auth metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "gitlab",
		SpecRefID:  "gitlab-openapi-v2",
		Status:     AuthStatusPresentIncomplete,
		SourceRefs: []string{"https://gitlab.com/gitlab-org/gitlab/-/raw/master/doc/api/openapi/openapi_v2.yaml"},
		SourceNote: "GitLab's official Swagger 2.0 document includes token security definitions but no root security; official REST auth docs describe additional bearer-token forms.",
	},
	{
		ProviderID: "gong",
		SpecRefID:  "gong-api-access-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://help.gong.io/docs/receive-access-to-the-api", "https://help.gong.io/docs/create-an-app-for-gong", "https://help.gong.io/hc/en-us/articles/360046818511-Uploading-calls-from-a-non-integrated-telephony-system"},
		SourceNote: "Gong has official API human docs but no recorded stable public official OpenAPI document; access key/secret Basic auth and OAuth bearer metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "gmail",
		SpecRefID:  "gmail-discovery-v1",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://gmail.googleapis.com/$discovery/rest?version=v1"},
		SourceNote: "Gmail's official machine-readable source is Google Discovery, not OpenAPI; OAuth security metadata needs advisory OpenAPI overlay mapping.",
	},
	{
		ProviderID: "google-calendar",
		SpecRefID:  "calendar-discovery-v3",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://www.googleapis.com/discovery/v1/apis/calendar/v3/rest"},
		SourceNote: "Google Calendar's official machine-readable source is Google Discovery, not OpenAPI; OAuth security metadata needs advisory OpenAPI overlay mapping.",
	},
	{
		ProviderID: "google-drive",
		SpecRefID:  "drive-discovery-v3",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://www.googleapis.com/discovery/v1/apis/drive/v3/rest"},
		SourceNote: "Google Drive's official machine-readable source is Google Discovery, not OpenAPI; OAuth security metadata needs advisory OpenAPI overlay mapping.",
	},
	{
		ProviderID: "google-sheets",
		SpecRefID:  "sheets-discovery-v4",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://sheets.googleapis.com/$discovery/rest?version=v4"},
		SourceNote: "Google Sheets' official machine-readable source is Google Discovery, not OpenAPI; OAuth security metadata needs advisory OpenAPI overlay mapping.",
	},
	{
		ProviderID: "hubspot",
		SpecRefID:  "hubspot-public-api-spec-index",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://api.hubapi.com/public/api/spec/v1/specs"},
		SourceNote: "HubSpot publishes split official OpenAPI specs through an index; selected specs should be reviewed with auth overlay metadata before downstream use.",
	},
	{
		ProviderID: "intercom",
		SpecRefID:  "intercom-api-v2-15-openapi",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://developers.intercom.com/_bundle/docs/references/%402.15/rest-api/api.intercom.io.json?download=", "https://developers.intercom.com/docs/references/rest-api/api.intercom.io"},
		SourceNote: "Intercom's official API v2.15 OpenAPI document includes bearer-token security schemes and root security requirements.",
	},
	{
		ProviderID: "jira-cloud",
		SpecRefID:  "jira-cloud-platform-openapi-v3",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://dac-static.atlassian.com/cloud/jira/platform/swagger-v3.v3.json"},
		SourceNote: "Atlassian's Jira Cloud OpenAPI document includes OAuth2/basic security schemes and operation-level security metadata.",
	},
	{
		ProviderID: "jenkins",
		SpecRefID:  "jenkins-api-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://www.jenkins.io/doc/book/using/remote-access-api/", "https://www.jenkins.io/blog/2018/07/02/new-api-token-system/"},
		SourceNote: "Jenkins has no recorded official OpenAPI document in this catalog entry; auth metadata comes from advisory basic/API-token and crumb overlay notes.",
	},
	{
		ProviderID: "linear",
		SpecRefID:  "linear-graphql-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://linear.app/developers/graphql", "https://linear.app/docs/api/"},
		SourceNote: "Linear's official public API is GraphQL and no OpenAPI document is recorded in this catalog entry; personal API key and OAuth bearer-token metadata must come from advisory overlay notes.",
	},
	{
		ProviderID: "mailchimp",
		SpecRefID:  "mailchimp-marketing-api-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://mailchimp.com/developer/marketing/api/", "https://mailchimp.com/developer/marketing/guides/quick-start/"},
		SourceNote: "Mailchimp's official Marketing API docs describe OpenAPI-backed endpoint documentation and API-key authentication, but no stable public downloadable OpenAPI document is recorded in this catalog entry; security metadata must come from advisory overlay notes.",
	},
	{
		ProviderID: "microsoft-graph",
		SpecRefID:  "microsoft-graph-v1-openapi",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://raw.githubusercontent.com/microsoftgraph/msgraph-metadata/master/openapi/v1.0/openapi.yaml"},
		SourceNote: "Microsoft Graph's official OpenAPI v1.0 document lacks OpenAPI security schemes; official auth docs need advisory bearer-token overlay mapping.",
	},
	{
		ProviderID: "monday-com",
		SpecRefID:  "monday-api-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://developer.monday.com/api-reference/docs", "https://developer.monday.com/api-reference/docs/getting-started"},
		SourceNote: "Monday.com's official API is GraphQL and no OpenAPI document is recorded in this catalog entry; Authorization-header token metadata must come from advisory overlay notes.",
	},
	{
		ProviderID: "netlify",
		SpecRefID:  "netlify-api-openapi",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://open-api.netlify.com/openapi.json", "https://docs.netlify.com/api/get-started/"},
		SourceNote: "Netlify's official OpenAPI document includes OAuth2 security metadata and root security requirements; official API docs describe bearer personal access tokens for manual requests.",
	},
	{
		ProviderID: "notion",
		SpecRefID:  "notion-api-openapi",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://developers.notion.com/openapi.json"},
		SourceNote: "Notion's official OpenAPI document includes bearer auth security scheme metadata and a root security requirement.",
	},
	{
		ProviderID: "okta",
		SpecRefID:  "okta-management-minimal-openapi",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://raw.githubusercontent.com/okta/okta-management-openapi-spec/master/dist/current/management-minimal.yaml", "https://developer.okta.com/docs/api/openapi/okta-management/guides/overview/"},
		SourceNote: "Okta's official Management OpenAPI document includes SSWS API-token and OAuth2 security schemes with operation-level security metadata; official docs recommend scoped OAuth2 access tokens where possible.",
	},
	{
		ProviderID: "openweathermap",
		SpecRefID:  "openweathermap-api-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://openweathermap.org/api"},
		SourceNote: "OpenWeather has no recorded official OpenAPI document in the built-in catalog; API key placement is captured by advisory overlay metadata.",
	},
	{
		ProviderID: "pagerduty",
		SpecRefID:  "pagerduty-rest-openapi-v3",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://raw.githubusercontent.com/PagerDuty/api-schema/main/reference/REST/openapiv3.json"},
		SourceNote: "PagerDuty's official OpenAPI document includes an Authorization header API key scheme and root security requirement.",
	},
	{
		ProviderID: "paypal",
		SpecRefID:  "paypal-checkout-orders-v2-openapi",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://raw.githubusercontent.com/paypal/paypal-rest-api-specifications/main/openapi/checkout_orders_v2.json"},
		SourceNote: "PayPal's official Checkout Orders v2 OpenAPI document includes OAuth2 security scheme metadata and operation-level security requirements.",
	},
	{
		ProviderID: "pipedrive",
		SpecRefID:  "pipedrive-api-v2-openapi",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://developers.pipedrive.com/docs/api/v1/openapi-v2.yaml"},
		SourceNote: "Pipedrive's official API v2 OpenAPI document includes API-token, OAuth2, and basic auth security schemes plus operation-level security requirements.",
	},
	{
		ProviderID: "quickbooks",
		SpecRefID:  "quickbooks-online-api-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://developer.intuit.com/app/developer/qbo/docs/learn/explore-the-quickbooks-online-api", "https://developer.intuit.com/app/developer/qbo/docs/develop/authentication-and-authorization/oauth-2.0"},
		SourceNote: "QuickBooks Online has official REST API and OAuth 2.0 docs but no stable public downloadable OpenAPI document recorded in this catalog entry; OAuth bearer security metadata must come from advisory overlay notes.",
	},
	{
		ProviderID: "salesforce",
		SpecRefID:  "salesforce-rest-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://developer.salesforce.com/docs/atlas.en-us.api_rest.meta/api_rest/intro_rest.htm"},
		SourceNote: "Salesforce has no recorded stable public downloadable OpenAPI document for the core REST API in the built-in catalog; OAuth bearer security metadata must come from advisory overlay notes.",
	},
	{
		ProviderID: "seatable",
		SpecRefID:  "seatable-authentication-v6-2-openapi",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://api.seatable.com/openapi", "https://api.seatable.com/reference/authentication", "https://api.seatable.com/openapi/69d4dc0c1422831f8d6fbb8e", "https://api.seatable.com/openapi/69d4dc0c1422831f8d6fbb93", "https://api.seatable.com/openapi/69d4dc0c1422831f8d6fbb96", "https://api.seatable.com/openapi/69d4dc0c1422831f8d6fbb95"},
		SourceNote: "SeaTable's official v6.2 OpenAPI documents include separate bearer security schemes and operation security requirements for account, API, and base tokens.",
	},
	{
		ProviderID: "sendgrid",
		SpecRefID:  "sendgrid-mail-v3-openapi",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://raw.githubusercontent.com/twilio/sendgrid-oai/main/spec/json/tsg_mail_v3.json"},
		SourceNote: "Twilio SendGrid's official Mail v3 OpenAPI document includes bearer API-key security metadata and root security requirements.",
	},
	{
		ProviderID: "sentry",
		SpecRefID:  "sentry-api-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://docs.sentry.io/api/", "https://docs.sentry.io/api/auth/"},
		SourceNote: "Sentry has no recorded official OpenAPI document in this catalog entry; auth metadata comes from advisory bearer-token overlay notes.",
	},
	{
		ProviderID: "servicenow",
		SpecRefID:  "servicenow-rest-api-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://www.servicenow.com/docs/r/api-reference/rest-api-explorer/c_RESTAPI.html", "https://www.servicenow.com/docs/r/api-reference/rest-api-explorer/export-openapi-specification.html"},
		SourceNote: "ServiceNow documents per-instance OpenAPI export through REST API Explorer, but this catalog entry has no stable public downloadable OpenAPI document; Basic and OAuth security metadata must come from advisory overlay notes.",
	},
	{
		ProviderID: "shopify",
		SpecRefID:  "shopify-admin-rest-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://shopify.dev/docs/api/admin-rest"},
		SourceNote: "Shopify has no recorded official OpenAPI document in the built-in catalog; Admin REST access-token security metadata must come from advisory overlay notes.",
	},
	{
		ProviderID: "slack",
		SpecRefID:  "slack-web-openapi-v2",
		Status:     AuthStatusPresentIncomplete,
		SourceRefs: []string{"https://raw.githubusercontent.com/slackapi/slack-api-specs/master/web-api/slack_web_openapi_v2_without_examples.json"},
		SourceNote: "Slack's archived official OpenAPI document includes OAuth metadata and operation security but needs freshness review against current Slack docs.",
	},
	{
		ProviderID: "snowflake",
		SpecRefID:  "snowflake-sql-api-openapi",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://raw.githubusercontent.com/snowflakedb/snowflake-rest-api-specs/main/specifications/sqlapi.yaml", "https://docs.snowflake.com/en/developer-guide/snowflake-rest-api/snowflake-rest-api"},
		SourceNote: "Snowflake's official SQL API OpenAPI document includes key-pair JWT, external OAuth, Snowflake OAuth, and programmatic access token security schemes with root and operation security requirements.",
	},
	{
		ProviderID: "splunk",
		SpecRefID:  "splunk-api-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://docs.splunk.com/Documentation/Splunk/latest/RESTREF", "https://docs.splunk.com/Documentation/Splunk/latest/RESTUM/RESTusing"},
		SourceNote: "Splunk has no recorded stable public OpenAPI document for the general Enterprise REST API in this catalog entry; auth metadata comes from advisory token/basic overlay notes.",
	},
	{
		ProviderID: "stripe",
		SpecRefID:  "stripe-latest-openapi-spec3",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://raw.githubusercontent.com/stripe/openapi/master/latest/openapi.spec3.json"},
		SourceNote: "Stripe's official latest OpenAPI document includes basic and bearer HTTP security schemes plus root security requirements.",
	},
	{
		ProviderID: "supabase",
		SpecRefID:  "supabase-management-api-openapi",
		Status:     AuthStatusPresentIncomplete,
		SourceRefs: []string{"https://api.supabase.com/api/v1-json", "https://supabase.com/docs/reference/api/introduction"},
		SourceNote: "Supabase's official Management API OpenAPI document includes a bearer security scheme but no root security requirement; official docs state bearer authorization is required for API requests.",
	},
	{
		ProviderID: "telegram",
		SpecRefID:  "telegram-bot-api-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://core.telegram.org/bots/api"},
		SourceNote: "Telegram has official Bot API docs but no OpenAPI document recorded in this catalog entry; bot-token path authentication needs advisory metadata and cannot be represented exactly as an OpenAPI security scheme.",
	},
	{
		ProviderID: "trello",
		SpecRefID:  "trello-cloud-openapi-v3",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://dac-static.atlassian.com/cloud/trello/swagger.v3.json"},
		SourceNote: "Atlassian's Trello OpenAPI document includes query API key/token security schemes and root security requirements.",
	},
	{
		ProviderID: "twilio",
		SpecRefID:  "twilio-api-v2010-openapi",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://raw.githubusercontent.com/twilio/twilio-oai/main/spec/json/twilio_api_v2010.json"},
		SourceNote: "Twilio's official OpenAPI v2010 document includes an HTTP Basic security scheme and operation-level security requirements for Account SID/Auth Token authentication.",
	},
	{
		ProviderID: "typeform",
		SpecRefID:  "typeform-developer-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://www.typeform.com/developers/get-started/", "https://www.typeform.com/developers/get-started/personal-access-token/", "https://www.typeform.com/developers/get-started/applications/"},
		SourceNote: "Typeform has official REST API and token/OAuth docs but no stable public downloadable OpenAPI document recorded in this catalog entry; bearer-token security metadata must come from advisory overlay notes.",
	},
	{
		ProviderID: "xero",
		SpecRefID:  "xero-accounting-openapi",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://raw.githubusercontent.com/XeroAPI/Xero-OpenAPI/master/xero_accounting.yaml"},
		SourceNote: "Xero's official Accounting OpenAPI document includes OAuth2 authorization code security metadata and operation-level scope requirements.",
	},
	{
		ProviderID: "zoom",
		SpecRefID:  "zoom-api-v2-openapi",
		Status:     AuthStatusPresentIncomplete,
		SourceRefs: []string{"https://raw.githubusercontent.com/zoom/api/master/openapi.v2.json", "https://developers.zoom.us/docs/integrations/oauth/"},
		SourceNote: "Zoom's official GitHub Swagger 2.0 document includes older access_token query security, while current docs describe OAuth bearer access tokens and granular scopes; security metadata should be reviewed with advisory overlay notes.",
	},
	{
		ProviderID: "zendesk",
		SpecRefID:  "zendesk-sunshine-conversations-openapi",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://raw.githubusercontent.com/zendesk/sunshine-conversations-api-spec/master/openapi.yaml"},
		SourceNote: "Zendesk's official Sunshine Conversations OpenAPI document includes basic and bearer HTTP security schemes plus root and operation security requirements for the messaging API surface.",
	},
	{
		ProviderID: "acuity-scheduling",
		SpecRefID:  "acuity-quick-start",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://developers.acuityscheduling.com/docs/quick-start", "https://developers.acuityscheduling.com/reference"},
		SourceNote: "Acuity Scheduling has official API docs but no recorded official OpenAPI document; Basic auth metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "bitly",
		SpecRefID:  "bitly-api-v4-openapi",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://dev.bitly.com/v4/v4.json", "https://dev.bitly.com/api-reference"},
		SourceNote: "Bitly's official API v4 OpenAPI document includes a bearer HTTP security scheme and root security requirement.",
	},
	{
		ProviderID: "brevo",
		SpecRefID:  "brevo-api-v3-openapi",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://api.brevo.com/v3/swagger_definition_v3.yml", "https://developers.brevo.com/docs/api-clients"},
		SourceNote: "Brevo's official API v3 OpenAPI document includes api-key and partner-key header security schemes with root security requirements.",
	},
	{
		ProviderID: "clockify",
		SpecRefID:  "clockify-api-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://docs.clockify.me/", "https://clockify.me/help/getting-started/clockify-api-overview"},
		SourceNote: "Clockify has official REST-shaped API docs but no recorded official OpenAPI document; X-Api-Key and X-Addon-Token metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "coda",
		SpecRefID:  "coda-api-v1-openapi",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://coda.io/apis/v1/openapi.json", "https://coda.io/developers/apis/v1"},
		SourceNote: "Coda's official API v1 OpenAPI document includes a bearer HTTP security scheme and root security requirement.",
	},
	{
		ProviderID: "deepl",
		SpecRefID:  "deepl-api-openapi",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://raw.githubusercontent.com/DeepLcom/openapi/main/openapi.yaml", "https://developers.deepl.com/docs/resources/open-api-spec"},
		SourceNote: "DeepL's official OpenAPI document includes an Authorization header API-key scheme and operation security requirements.",
	},
	{
		ProviderID: "figma",
		SpecRefID:  "figma-rest-api-openapi",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://raw.githubusercontent.com/figma/rest-api-spec/main/openapi/openapi.yaml", "https://developers.figma.com/docs/rest-api/"},
		SourceNote: "Figma's official REST API OpenAPI document includes personal access token, plan access token, and OAuth2 security schemes with operation-level security requirements.",
	},
	{
		ProviderID: "ghost",
		SpecRefID:  "ghost-admin-api-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://docs.ghost.org/admin-api/", "https://docs.ghost.org/content-api/"},
		SourceNote: "Ghost has official Admin and Content API docs but no recorded official OpenAPI document; Admin JWT and Content API key metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "harvest",
		SpecRefID:  "harvest-auth-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://help.getharvest.com/api-v2/", "https://help.getharvest.com/api-v2/authentication-api/authentication/authentication/"},
		SourceNote: "Harvest has official API v2 docs but no recorded official OpenAPI document; bearer token plus Harvest-Account-ID metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "help-scout",
		SpecRefID:  "help-scout-mailbox-api-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://developer.helpscout.com/mailbox-api/", "https://developer.helpscout.com/docs-api/"},
		SourceNote: "Help Scout has official Inbox API and Docs API docs but no recorded official OpenAPI document; OAuth and Basic auth metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "iterable",
		SpecRefID:  "iterable-api-key-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://api.iterable.com/api/docs", "https://support.iterable.com/hc/en-us/articles/360043464871-API-Keys"},
		SourceNote: "Iterable has official API docs but no recorded official OpenAPI document; Api-Key header metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "convertkit",
		SpecRefID:  "kit-api-v4-openapi",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://developers.kit.com/api-reference/v4.json", "https://developers.kit.com/api-reference/authentication"},
		SourceNote: "Kit's official API V4 OpenAPI document includes API-key and OAuth2 security schemes with operation security requirements; official auth docs describe the X-Kit-Api-Key header for API-key access.",
	},
	{
		ProviderID: "mailjet",
		SpecRefID:  "mailjet-api-key-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://documentation.mailjet.com/hc/en-us/articles/360044088173-REST-API", "https://documentation.mailjet.com/hc/en-us/articles/360043225693-What-is-an-API-key"},
		SourceNote: "Mailjet has official REST API docs but no recorded official OpenAPI document; API key and secret key metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "nocodb",
		SpecRefID:  "nocodb-api-v2-openapi",
		Status:     AuthStatusPresentIncomplete,
		SourceRefs: []string{"https://nocodb.com/apis/v2/swagger-v2.json", "https://nocodb.com/apis/v3/swagger-v3.json", "https://nocodb.com/docs/product-docs/developer-resources/rest-apis/accessing-apis"},
		SourceNote: "NocoDB's official OpenAPI documents include API-token and auth-token metadata, but token requirements are split between reusable header parameters and security schemes, with v2/v3 differences that need normalization before treating auth as complete.",
	},
	{
		ProviderID: "posthog",
		SpecRefID:  "posthog-api-openapi",
		Status:     AuthStatusPresentIncomplete,
		SourceRefs: []string{"https://us.posthog.com/api/schema/", "https://posthog.com/docs/api"},
		SourceNote: "PostHog's official OpenAPI schema includes personal API key bearer authentication for private API endpoints; official API docs also describe public project-token ingestion endpoints that are not fully modeled by that schema.",
	},
	{
		ProviderID: "uptimerobot",
		SpecRefID:  "uptimerobot-api-v3-openapi",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://cdn.uptimerobot.com/api/openapi.yaml", "https://uptimerobot.com/api/v3/"},
		SourceNote: "UptimeRobot's official API v3 OpenAPI document includes a bearer security scheme with operation-level security requirements; official v3 docs describe API-key access and rate limits.",
	},
	{
		ProviderID: "coingecko",
		SpecRefID:  "coingecko-auth-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://docs.coingecko.com/reference/authentication", "https://docs.coingecko.com/reference/endpoint-overview"},
		SourceNote: "CoinGecko has official API human docs but no recorded stable public official OpenAPI document; Demo and Pro API-key header/query metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "hackernews",
		SpecRefID:  "hackernews-firebase-api-docs",
		Status:     AuthStatusIntentionallyAnonymous,
		SourceRefs: []string{"https://github.com/HackerNews/API"},
		SourceNote: "Hacker News publishes an official public Firebase API for item, user, story-list, maxitem, and update reads without authentication.",
	},
	{
		ProviderID: "marketstack",
		SpecRefID:  "marketstack-api-v2-openapi",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://api.swaggerhub.com/apis/apilayer-863/MarketstackAPIv2/2.0.0/swagger.json", "https://docs.apilayer.com/marketstack/docs/marketstack-api-v2-v-2-0-0"},
		SourceNote: "Marketstack's official API v2 OpenAPI document includes an access_key query API-key security scheme with a root security requirement.",
	},
	{
		ProviderID: "nasa",
		SpecRefID:  "nasa-api-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://api.nasa.gov/"},
		SourceNote: "NASA Open APIs have official human docs but no recorded stable public official OpenAPI document; shared api_key query metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "onesimpleapi",
		SpecRefID:  "onesimpleapi-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://onesimpleapi.com/docs", "https://onesimpleapi.com/user/api-tokens"},
		SourceNote: "OneSimpleApi has official human docs but no recorded stable public official OpenAPI document; token query metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "openthesaurus",
		SpecRefID:  "openthesaurus-api-docs",
		Status:     AuthStatusIntentionallyAnonymous,
		SourceRefs: []string{"https://www.openthesaurus.de/about/api"},
		SourceNote: "OpenThesaurus official API docs describe anonymous GET synonym lookup in JSON or XML, subject to published usage conditions.",
	},
	{
		ProviderID: "quickchart",
		SpecRefID:  "quickchart-api-docs",
		Status:     AuthStatusIntentionallyAnonymous,
		SourceRefs: []string{"https://quickchart.io/documentation/", "https://quickchart.io/documentation/qr-codes/"},
		SourceNote: "QuickChart official chart and QR-code API docs describe anonymous rendering endpoints in the reviewed scope; paid/account features are outside this catalog row.",
	},
	{
		ProviderID: "reddit",
		SpecRefID:  "reddit-api-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://www.reddit.com/dev/api/", "https://www.reddit.com/dev/api/oauth", "https://developers.reddit.com/docs/capabilities/server/reddit-api"},
		SourceNote: "Reddit has official generated API docs but no recorded stable public official OpenAPI document; OAuth bearer metadata for oauth.reddit.com comes from advisory overlay notes.",
	},
	{
		ProviderID: "autopilot",
		SpecRefID:  "autopilot-api-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://autopilot.docs.apiary.io/", "https://help.ortto.com/a-376-autopilot-how-to-use-autopilots-api"},
		SourceNote: "Autopilot has official human API docs but no recorded stable public official OpenAPI document; legacy API-key header metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "drift",
		SpecRefID:  "drift-auth-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://devdocs.drift.com/docs/authentication-and-scopes", "https://devdocs.drift.com/docs/platform-apis"},
		SourceNote: "Drift has official human backend API docs but no recorded stable public official OpenAPI document; OAuth bearer metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "freshworks-crm",
		SpecRefID:  "freshworks-crm-api-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://developers.freshworks.com/crm/api/"},
		SourceNote: "Freshworks CRM has official human REST API docs but no recorded stable public official OpenAPI document; Authorization token metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "getresponse",
		SpecRefID:  "getresponse-auth-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://apidocs.getresponse.com/v3/authentication", "https://apidocs.getresponse.com/v3"},
		SourceNote: "GetResponse has official human API docs but no recorded stable public official OpenAPI document; API-key and OAuth metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "highlevel",
		SpecRefID:  "highlevel-contacts-openapi",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://github.com/GoHighLevel/highlevel-api-docs", "https://marketplace.gohighlevel.com/docs/", "https://help.gohighlevel.com/support/solutions/articles/48001060529-highlevel-api-documentation"},
		SourceNote: "HighLevel's official API v2 OpenAPI modules include bearer security schemes and operation-level security metadata for agency and location access.",
	},
	{
		ProviderID: "keap",
		SpecRefID:  "keap-rest-api-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://developer.keap.com/docs/rest/1000/"},
		SourceNote: "Keap has official human REST API docs but no recorded stable public official OpenAPI document; OAuth bearer metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "mailerlite",
		SpecRefID:  "mailerlite-api-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://developers.mailerlite.com/docs/", "https://developers-classic.mailerlite.com/docs/authentication", "https://www.mailerlite.com/help/where-to-find-the-mailerlite-api-key-groupid-and-documentation"},
		SourceNote: "MailerLite has official human API docs but no recorded stable public official OpenAPI document; current bearer-token and Classic API-key metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "mautic",
		SpecRefID:  "mautic-api-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://developer.mautic.org/", "https://kb.mautic.org/article/what-is-mautic-039%3Bs-api.html"},
		SourceNote: "Mautic has official human API docs but no recorded stable public official OpenAPI document; OAuth and Basic auth metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "monica-crm",
		SpecRefID:  "monica-crm-api-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://www.monicahq.com/api"},
		SourceNote: "Monica has official human API docs but no recorded stable public official OpenAPI document; OAuth bearer metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "salesmate",
		SpecRefID:  "salesmate-api-auth-example",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://support.salesmate.io/hc/en-us/articles/12864176787609-What-Are-API-s-Usage-and-Working", "https://support.salesmate.io/hc/en-us/articles/360043653751-Auto-enroll-Contacts-to-sequence-whenever-a-Deal-stage-is-updated-via-Webhooks", "https://apidocs.salesmate.io/"},
		SourceNote: "Salesmate has official human API docs but no recorded stable public official OpenAPI document; sessionToken and x-linkname metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "sendy",
		SpecRefID:  "sendy-api-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://sendy.co/api"},
		SourceNote: "Sendy has official human API docs but no recorded stable public official OpenAPI document; api_key form/query metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "vero",
		SpecRefID:  "vero-track-api-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://help.getvero.com/api-reference/track/overview", "https://help.getvero.com/developer-docs/overview"},
		SourceNote: "Vero has official human API docs but no recorded stable public official OpenAPI document; auth_token request-parameter metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "jotform",
		SpecRefID:  "jotform-api-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://api.jotform.com/docs/", "https://api.jotform.com/"},
		SourceNote: "JotForm has official human API docs but no recorded stable public official OpenAPI document; APIKEY metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "formio",
		SpecRefID:  "formio-auth-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://help.form.io/developers/authentication", "https://help.form.io/developers/introduction/api-documentation"},
		SourceNote: "Form.io has official human API docs but no recorded stable public official OpenAPI document; token metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "formstack",
		SpecRefID:  "formstack-auth-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://developers.formstack.com/reference/authorization", "https://developers.formstack.com/v2.0/reference/api-overview"},
		SourceNote: "Formstack has official human API docs but no recorded stable public official OpenAPI document; OAuth bearer metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "surveymonkey",
		SpecRefID:  "surveymonkey-api-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://api.surveymonkey.com/v3/docs", "https://help.surveymonkey.com/en/surveymonkey/integrations/surveymonkey-api/"},
		SourceNote: "SurveyMonkey has official human API docs but no recorded stable public official OpenAPI document; OAuth bearer metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "kobotoolbox",
		SpecRefID:  "kobotoolbox-api-v2-openapi",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://kf.kobotoolbox.org/api/v2/schema/", "https://kf.kobotoolbox.org/api/v2/docs/"},
		SourceNote: "KoBoToolbox official API v2 OpenAPI schema declares TokenAuth, BasicAuth, and SCIM bearer security metadata with operation-level requirements.",
	},
	{
		ProviderID: "bubble",
		SpecRefID:  "bubble-api-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://manual.bubble.io/help-guides/integrations/api/the-bubble-api", "https://manual.bubble.io/core-resources/api/data-api"},
		SourceNote: "Bubble has official human docs and app-specific Swagger metadata but no stable provider-wide public official OpenAPI document; bearer token metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "cal",
		SpecRefID:  "cal-api-v2-openapi",
		Status:     AuthStatusPresentIncomplete,
		SourceRefs: []string{"https://cal.com/docs", "https://cal.com/docs/api-reference/v2/openapi.json"},
		SourceNote: "Cal.com official API v2 OpenAPI metadata requires Authorization headers but does not declare reusable OpenAPI security schemes; bearer metadata comes from a review overlay.",
	},
	{
		ProviderID: "cockpit",
		SpecRefID:  "cockpit-api-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://getcockpit.com/documentation/core/api", "https://getcockpit.com/documentation/core"},
		SourceNote: "Cockpit has official human API docs but no recorded stable public official OpenAPI document; token metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "stackby",
		SpecRefID:  "stackby-api-key-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://help.stackby.com/en/articles/29-developer-api", "https://help.stackby.com/en/articles/124-how-to-get-your-api-key-in-stackby"},
		SourceNote: "Stackby has official human API docs but no recorded stable public official OpenAPI document; api-key header metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "airtop",
		SpecRefID:  "airtop-api-openapi",
		Status:     AuthStatusPresentIncomplete,
		SourceRefs: []string{"https://docs.airtop.ai/openapi.json", "https://docs.airtop.ai/api-reference/airtop-api"},
		SourceNote: "Airtop's official OpenAPI document declares bearer metadata and repeated Authorization header parameters; bearer metadata is carried as a present-incomplete review overlay.",
	},
	{
		ProviderID: "timesaved",
		SpecRefID:  "timesaved-n8n-docs",
		Status:     AuthStatusIntentionallyAnonymous,
		SourceRefs: []string{"https://docs.n8n.io/integrations/builtin/app-nodes/n8n-nodes-base.savedTime/"},
		SourceNote: "TimeSaved is an n8n workflow metadata helper, not an external provider API; there is no credential or auth surface to model.",
	},
	{
		ProviderID: "filemaker",
		SpecRefID:  "filemaker-data-api-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://help.claris.com/en/data-api-guide/content/data-api-reference.html", "https://help.claris.com/en/data-api-guide/"},
		SourceNote: "FileMaker has official human Data API docs but no recorded stable public official OpenAPI document; Basic login and bearer session metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "facebook",
		SpecRefID:  "facebook-graph-api-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://developers.facebook.com/docs/graph-api/", "https://developers.facebook.com/docs/pages-api/"},
		SourceNote: "Facebook has official Graph API human docs but no recorded stable public official OpenAPI document; bearer access-token metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "facebook-lead-ads",
		SpecRefID:  "facebook-lead-ads-retrieval-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://developers.facebook.com/docs/marketing-api/guides/lead-ads/retrieving/", "https://developers.facebook.com/docs/marketing-api/guides/lead-ads/webhooks/"},
		SourceNote: "Facebook Lead Ads has official human docs but no recorded stable public official OpenAPI document; access-token and app-review-sensitive permission metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "linkedin",
		SpecRefID:  "linkedin-marketing-api-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://learn.microsoft.com/en-us/linkedin/marketing/", "https://learn.microsoft.com/en-us/linkedin/shared/authentication/authorization-code-flow"},
		SourceNote: "LinkedIn has official human API docs but no recorded stable public official OpenAPI document; OAuth bearer metadata and version/header requirements come from advisory overlay notes.",
	},
	{
		ProviderID: "line",
		SpecRefID:  "line-messaging-api-openapi",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://raw.githubusercontent.com/line/line-openapi/main/messaging-api.yml", "https://github.com/line/line-openapi"},
		SourceNote: "LINE's official Messaging API OpenAPI document includes bearer channel access-token security metadata with root security requirements.",
	},
	{
		ProviderID: "mattermost",
		SpecRefID:  "mattermost-api-v4-openapi",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://developers.mattermost.com/mattermost-openapi-v4.yaml", "https://developers.mattermost.com/contribute/more-info/server/rest-api/"},
		SourceNote: "Mattermost's official API v4 OpenAPI document includes bearer security scheme metadata and operation-level security requirements.",
	},
	{
		ProviderID: "messagebird",
		SpecRefID:  "messagebird-api-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://developers.messagebird.com/api/"},
		SourceNote: "MessageBird has official human API docs but no recorded stable public official OpenAPI document; Authorization: AccessKey metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "matrix",
		SpecRefID:  "matrix-client-server-api-openapi",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://spec.matrix.org/latest/client-server-api/api.json", "https://spec.matrix.org/latest/client-server-api/"},
		SourceNote: "Matrix's official Client-Server API OpenAPI document includes bearer and deprecated query access-token security schemes with operation-level requirements.",
	},
	{
		ProviderID: "rocket-chat",
		SpecRefID:  "rocket-chat-messaging-openapi",
		Status:     AuthStatusPresentIncomplete,
		SourceRefs: []string{"https://raw.githubusercontent.com/RocketChat/Rocket.Chat-Open-API/main/authentication.yaml", "https://raw.githubusercontent.com/RocketChat/Rocket.Chat-Open-API/main/messaging.yaml", "https://developer.rocket.chat/apidocs"},
		SourceNote: "Rocket.Chat's official OpenAPI modules model x-Auth-Token and x-User-Id as header parameters rather than reusable security schemes; combined header requirements come from advisory overlay notes.",
	},
	{
		ProviderID: "twake",
		SpecRefID:  "twake-developers-api-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://doc.twake.app/developers-api/api-reference/auth", "https://doc.twake.app/developers-api/api-reference/message/post-request"},
		SourceNote: "Twake has official Developers API human docs but no recorded stable public official OpenAPI document; Basic application authentication metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "twist",
		SpecRefID:  "twist-api-v3-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://developer.twist.com/v3/"},
		SourceNote: "Twist has official API v3 human docs but no recorded stable public official OpenAPI document; OAuth/test bearer token metadata comes from advisory overlay notes.",
	},
	{
		ProviderID: "twitter",
		SpecRefID:  "twitter-api-v2-openapi",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://api.twitter.com/2/openapi.json", "https://docs.x.com/x-api"},
		SourceNote: "X/Twitter's official API v2 OpenAPI document includes bearer token, OAuth 2.0 authorization code, and OAuth 1.0a-style security metadata with operation-level requirements.",
	},
	{
		ProviderID: "whatsapp",
		SpecRefID:  "whatsapp-cloud-api-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://developers.facebook.com/docs/whatsapp/cloud-api/", "https://developers.facebook.com/docs/whatsapp/cloud-api/reference/messages"},
		SourceNote: "WhatsApp Cloud API has official human docs but no recorded stable public official OpenAPI document; Meta Graph API bearer access-token metadata comes from advisory overlay notes.",
	},
}

func validateSecurityClassification(classification SecurityClassification, providers map[string]Provider) error {
	if strings.TrimSpace(classification.ProviderID) == "" {
		return fmt.Errorf("missing provider id")
	}
	if !validID(classification.ProviderID) {
		return fmt.Errorf("invalid provider id %q", classification.ProviderID)
	}
	provider, ok := providers[classification.ProviderID]
	if !ok {
		return fmt.Errorf("unknown provider %q", classification.ProviderID)
	}
	if strings.TrimSpace(classification.SpecRefID) != "" {
		if !validID(classification.SpecRefID) {
			return fmt.Errorf("invalid spec ref id %q", classification.SpecRefID)
		}
		if !providerHasSpecRef(provider, classification.SpecRefID) {
			return fmt.Errorf("unknown spec ref %q for provider %q", classification.SpecRefID, classification.ProviderID)
		}
	}
	if !validAuthCompletenessStatus(classification.Status) {
		return fmt.Errorf("invalid status %q", classification.Status)
	}
	if strings.TrimSpace(classification.SourceNote) == "" {
		return fmt.Errorf("missing source note")
	}
	if len(classification.SourceRefs) == 0 {
		return fmt.Errorf("missing source refs")
	}
	for i, ref := range classification.SourceRefs {
		if !validHTTPSURL(ref) {
			return fmt.Errorf("source ref[%d]: must be https", i)
		}
	}
	return validateUniqueStrings("source_ref", classification.SourceRefs)
}

func sortSecurityClassifications(classifications []SecurityClassification) {
	sort.SliceStable(classifications, func(i, j int) bool {
		return classifications[i].ProviderID < classifications[j].ProviderID
	})
}

func cloneSecurityClassifications(in []SecurityClassification) []SecurityClassification {
	out := make([]SecurityClassification, len(in))
	for i, classification := range in {
		out[i] = classification
		out[i].SourceRefs = append([]string(nil), classification.SourceRefs...)
	}
	return out
}

func appendIfNotEmpty(values []string, value string) []string {
	if strings.TrimSpace(value) == "" {
		return values
	}
	return append(values, value)
}

func sortedUniqueStrings(values []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	sort.Strings(out)
	return out
}
