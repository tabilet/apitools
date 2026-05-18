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
		ProviderID: "box",
		SpecRefID:  "box-platform-openapi-v3",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://raw.githubusercontent.com/box/box-openapi/main/openapi.json"},
		SourceNote: "Box's official OpenAPI document includes an OAuth2 authorization code security scheme and root security requirement.",
	},
	{
		ProviderID: "calendly",
		SpecRefID:  "calendly-public-api-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://developer.calendly.com/api-docs"},
		SourceNote: "Calendly's official API reference is human documentation backed by Stoplight hosting in this catalog entry; no downloadable official OpenAPI document is recorded, so endpoint and security metadata come from advisory overlays.",
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
		ProviderID: "dropbox",
		SpecRefID:  "dropbox-api-stone-spec",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://github.com/dropbox/dropbox-api-spec"},
		SourceNote: "Dropbox's official machine-readable source is a Stone spec, not OpenAPI; OAuth security metadata needs advisory OpenAPI overlay mapping for OpenAPI-only consumers.",
	},
	{
		ProviderID: "github",
		SpecRefID:  "github-rest-api-openapi",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://raw.githubusercontent.com/github/rest-api-description/main/descriptions/api.github.com/api.github.com.json"},
		SourceNote: "GitHub's official OpenAPI document omits security schemes and requirements; official REST authentication docs need advisory bearer/basic overlay mapping.",
	},
	{
		ProviderID: "gitlab",
		SpecRefID:  "gitlab-openapi-v2",
		Status:     AuthStatusPresentIncomplete,
		SourceRefs: []string{"https://gitlab.com/gitlab-org/gitlab/-/raw/master/doc/api/openapi/openapi_v2.yaml"},
		SourceNote: "GitLab's official Swagger 2.0 document includes token security definitions but no root security; official REST auth docs describe additional bearer-token forms.",
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
		ProviderID: "jira-cloud",
		SpecRefID:  "jira-cloud-platform-openapi-v3",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://dac-static.atlassian.com/cloud/jira/platform/swagger-v3.v3.json"},
		SourceNote: "Atlassian's Jira Cloud OpenAPI document includes OAuth2/basic security schemes and operation-level security metadata.",
	},
	{
		ProviderID: "microsoft-graph",
		SpecRefID:  "microsoft-graph-v1-openapi",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://raw.githubusercontent.com/microsoftgraph/msgraph-metadata/master/openapi/v1.0/openapi.yaml"},
		SourceNote: "Microsoft Graph's official OpenAPI v1.0 document lacks OpenAPI security schemes; official auth docs need advisory bearer-token overlay mapping.",
	},
	{
		ProviderID: "notion",
		SpecRefID:  "notion-api-openapi",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://developers.notion.com/openapi.json"},
		SourceNote: "Notion's official OpenAPI document includes bearer auth security scheme metadata and a root security requirement.",
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
		ProviderID: "salesforce",
		SpecRefID:  "salesforce-rest-docs",
		Status:     AuthStatusOverlayRequired,
		SourceRefs: []string{"https://developer.salesforce.com/docs/atlas.en-us.api_rest.meta/api_rest/intro_rest.htm"},
		SourceNote: "Salesforce has no recorded stable public downloadable OpenAPI document for the core REST API in the built-in catalog; OAuth bearer security metadata must come from advisory overlay notes.",
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
		ProviderID: "stripe",
		SpecRefID:  "stripe-latest-openapi-spec3",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://raw.githubusercontent.com/stripe/openapi/master/latest/openapi.spec3.json"},
		SourceNote: "Stripe's official latest OpenAPI document includes basic and bearer HTTP security schemes plus root security requirements.",
	},
	{
		ProviderID: "trello",
		SpecRefID:  "trello-cloud-openapi-v3",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://dac-static.atlassian.com/cloud/trello/swagger.v3.json"},
		SourceNote: "Atlassian's Trello OpenAPI document includes query API key/token security schemes and root security requirements.",
	},
	{
		ProviderID: "zendesk",
		SpecRefID:  "zendesk-sunshine-conversations-openapi",
		Status:     AuthStatusComplete,
		SourceRefs: []string{"https://raw.githubusercontent.com/zendesk/sunshine-conversations-api-spec/master/openapi.yaml"},
		SourceNote: "Zendesk's official Sunshine Conversations OpenAPI document includes basic and bearer HTTP security schemes plus root and operation security requirements for the messaging API surface.",
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
