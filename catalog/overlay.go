package catalog

import (
	"fmt"
	"sort"
	"strings"
)

// SecurityOverlay records supplemental auth/security metadata separately from
// upstream provider specs. Overlays are advisory catalog metadata, not provider
// truth, and never contain credential values.
type SecurityOverlay struct {
	ID                string                 `json:"id"`
	ProviderID        string                 `json:"provider_id"`
	SpecRefID         string                 `json:"spec_ref_id,omitempty"`
	Status            AuthCompletenessStatus `json:"status"`
	SecuritySchemes   []SecurityScheme       `json:"security_schemes,omitempty"`
	RootSecurity      []SecurityRequirement  `json:"root_security,omitempty"`
	OperationSecurity []OperationSecurity    `json:"operation_security,omitempty"`
	SourceRefs        []string               `json:"source_refs,omitempty"`
	SourceNote        string                 `json:"source_note"`
}

// BuiltInSecurityOverlays returns built-in security overlays in deterministic
// order. Callers receive independent copies.
func BuiltInSecurityOverlays() []SecurityOverlay {
	out := cloneSecurityOverlays(builtInSecurityOverlays)
	sortSecurityOverlays(out)
	return out
}

// SecurityOverlaysForProvider returns built-in security overlays for a provider
// in deterministic order.
func SecurityOverlaysForProvider(providerID string) []SecurityOverlay {
	normalized := normalizeKey(providerID)
	var out []SecurityOverlay
	for _, overlay := range BuiltInSecurityOverlays() {
		if normalizeKey(overlay.ProviderID) == normalized {
			out = append(out, overlay)
		}
	}
	return out
}

// ValidateSecurityOverlays validates security overlays against provider catalog
// metadata.
func ValidateSecurityOverlays(overlays []SecurityOverlay, providers []Provider) error {
	providerByID := map[string]Provider{}
	for _, provider := range providers {
		providerByID[provider.ID] = provider
	}
	seenIDs := map[string]struct{}{}
	for i, overlay := range overlays {
		if err := validateSecurityOverlay(overlay, providerByID); err != nil {
			return fmt.Errorf("security overlay[%d]: %w", i, err)
		}
		if _, exists := seenIDs[overlay.ID]; exists {
			return fmt.Errorf("security overlay %q: duplicate id", overlay.ID)
		}
		seenIDs[overlay.ID] = struct{}{}
	}
	return nil
}

func validateSecurityOverlay(overlay SecurityOverlay, providers map[string]Provider) error {
	if strings.TrimSpace(overlay.ID) == "" {
		return fmt.Errorf("missing id")
	}
	if !validID(overlay.ID) {
		return fmt.Errorf("invalid id %q", overlay.ID)
	}
	if strings.TrimSpace(overlay.ProviderID) == "" {
		return fmt.Errorf("overlay %q: missing provider id", overlay.ID)
	}
	if !validID(overlay.ProviderID) {
		return fmt.Errorf("overlay %q: invalid provider id %q", overlay.ID, overlay.ProviderID)
	}
	provider, ok := providers[overlay.ProviderID]
	if !ok {
		return fmt.Errorf("overlay %q: unknown provider %q", overlay.ID, overlay.ProviderID)
	}
	if strings.TrimSpace(overlay.SpecRefID) != "" {
		if !validID(overlay.SpecRefID) {
			return fmt.Errorf("overlay %q: invalid spec ref id %q", overlay.ID, overlay.SpecRefID)
		}
		if !providerHasSpecRef(provider, overlay.SpecRefID) {
			return fmt.Errorf("overlay %q: unknown spec ref %q for provider %q", overlay.ID, overlay.SpecRefID, overlay.ProviderID)
		}
	}
	if !validAuthCompletenessStatus(overlay.Status) {
		return fmt.Errorf("overlay %q: invalid status %q", overlay.ID, overlay.Status)
	}
	if strings.TrimSpace(overlay.SourceNote) == "" {
		return fmt.Errorf("overlay %q: missing source note", overlay.ID)
	}
	if len(overlay.SourceRefs) == 0 {
		return fmt.Errorf("overlay %q: missing source refs", overlay.ID)
	}
	for i, ref := range overlay.SourceRefs {
		if !validHTTPSURL(ref) {
			return fmt.Errorf("overlay %q source ref[%d]: must be https", overlay.ID, i)
		}
	}
	if err := validateUniqueStrings("source_ref", overlay.SourceRefs); err != nil {
		return fmt.Errorf("overlay %q: %w", overlay.ID, err)
	}

	schemes := map[string]struct{}{}
	for i, scheme := range overlay.SecuritySchemes {
		if err := validateSecurityScheme(scheme); err != nil {
			return fmt.Errorf("overlay %q scheme[%d]: %w", overlay.ID, i, err)
		}
		if _, exists := schemes[scheme.Name]; exists {
			return fmt.Errorf("overlay %q scheme[%d]: duplicate security scheme %q", overlay.ID, i, scheme.Name)
		}
		schemes[scheme.Name] = struct{}{}
	}
	for i, requirement := range overlay.RootSecurity {
		if err := validateSecurityRequirement(requirement, schemes); err != nil {
			return fmt.Errorf("overlay %q root security[%d]: %w", overlay.ID, i, err)
		}
	}
	for i, operation := range overlay.OperationSecurity {
		if err := validateOperationSecurity(operation, schemes); err != nil {
			return fmt.Errorf("overlay %q operation security[%d]: %w", overlay.ID, i, err)
		}
	}
	return nil
}

func providerHasSpecRef(provider Provider, id string) bool {
	for _, ref := range provider.SpecReferences {
		if ref.ID == id {
			return true
		}
	}
	return false
}

var builtInSecurityOverlays = []SecurityOverlay{
	{
		ID:         "airtable-web-api-auth-overlay",
		ProviderID: "airtable",
		SpecRefID:  "airtable-web-api-docs",
		Status:     AuthStatusOverlayRequired,
		SecuritySchemes: []SecurityScheme{
			{
				Name:        "airtableBearer",
				Type:        SecuritySchemeHTTP,
				Scheme:      "bearer",
				Description: "Airtable Web API Personal Access Token or OAuth token carried as a bearer token.",
			},
		},
		RootSecurity: []SecurityRequirement{{Scheme: "airtableBearer"}},
		SourceRefs: []string{
			"https://airtable.com/developers/web/api/introduction",
			"https://support.airtable.com/docs/getting-started-with-airtables-web-api",
		},
		SourceNote: "Airtable has human Web API docs in this catalog entry but no recorded official OpenAPI document; Airtable docs describe PAT or OAuth token authentication, so OpenAPI imports need an advisory bearer-token overlay.",
	},
	{
		ID:         "calendly-public-api-auth-overlay",
		ProviderID: "calendly",
		SpecRefID:  "calendly-public-api-docs",
		Status:     AuthStatusOverlayRequired,
		SecuritySchemes: []SecurityScheme{
			{
				Name:        "calendlyBearer",
				Type:        SecuritySchemeHTTP,
				Scheme:      "bearer",
				Description: "Calendly personal access token or OAuth 2.1 access token carried in the Authorization header.",
			},
		},
		RootSecurity: []SecurityRequirement{{Scheme: "calendlyBearer"}},
		SourceRefs: []string{
			"https://developer.calendly.com/getting-started",
			"https://developer.calendly.com/authentication",
			"https://developer.calendly.com/creating-an-oauth-app",
		},
		SourceNote: "Calendly has human API docs in this catalog entry but no recorded downloadable official OpenAPI document; docs describe personal access tokens and OAuth 2.1, so OpenAPI imports need an advisory bearer-token overlay.",
	},
	{
		ID:         "clickup-api-v2-auth-review",
		ProviderID: "clickup",
		SpecRefID:  "clickup-api-v2-openapi",
		Status:     AuthStatusPresentIncomplete,
		SecuritySchemes: []SecurityScheme{
			{
				Name:          "Authorization_Token",
				Type:          SecuritySchemeAPIKey,
				In:            APIKeyInHeader,
				ParameterName: "Authorization",
				Description:   "ClickUp personal API token or OAuth2 bearer access token carried in the Authorization header.",
			},
			{
				Name: "clickupOAuth2",
				Type: SecuritySchemeOAuth2,
				Flows: []OAuthFlow{
					{
						Type:             OAuthFlowAuthorizationCode,
						AuthorizationURL: "https://app.clickup.com/api",
						TokenURL:         "https://api.clickup.com/api/v2/oauth/token",
					},
				},
				Description: "ClickUp OAuth 2.0 authorization code flow described in official authentication docs.",
			},
		},
		RootSecurity: []SecurityRequirement{{Scheme: "Authorization_Token"}, {Scheme: "clickupOAuth2"}},
		SourceRefs: []string{
			"https://developer.clickup.com/openapi/clickup-api-v2-reference.json",
			"https://developer.clickup.com/docs/authentication",
			"https://developer.clickup.com/reference/getaccesstoken",
		},
		SourceNote: "ClickUp's official OpenAPI document includes an Authorization header API key scheme and root security; OAuth authorization-code endpoints are documented separately, so keep this as a present-incomplete auth review overlay.",
	},
	{
		ID:         "dropbox-api-auth-overlay",
		ProviderID: "dropbox",
		SpecRefID:  "dropbox-api-stone-spec",
		Status:     AuthStatusOverlayRequired,
		SecuritySchemes: []SecurityScheme{
			{
				Name: "dropboxOAuth2",
				Type: SecuritySchemeOAuth2,
				Flows: []OAuthFlow{
					{
						Type:             OAuthFlowAuthorizationCode,
						AuthorizationURL: "https://www.dropbox.com/oauth2/authorize",
						TokenURL:         "https://api.dropboxapi.com/oauth2/token",
						Scopes: []string{
							"account_info.read",
							"files.content.read",
							"files.content.write",
							"files.metadata.read",
							"sharing.read",
							"sharing.write",
						},
					},
				},
				Description: "Dropbox OAuth 2.0 authorization code flow with scoped bearer access tokens.",
			},
		},
		RootSecurity: []SecurityRequirement{{Scheme: "dropboxOAuth2", Scopes: []string{"files.metadata.read"}}},
		SourceRefs: []string{
			"https://developers.dropbox.com/oauth-guide",
			"https://www.dropbox.com/developers/documentation/http/documentation",
			"https://github.com/dropbox/dropbox-api-spec",
		},
		SourceNote: "Dropbox publishes an official Stone machine spec rather than OpenAPI; official OAuth docs describe bearer-token scopes, so OpenAPI-only consumers need an advisory OAuth2 overlay.",
	},
	{
		ID:         "github-rest-api-auth-overlay",
		ProviderID: "github",
		SpecRefID:  "github-rest-api-openapi",
		Status:     AuthStatusOverlayRequired,
		SecuritySchemes: []SecurityScheme{
			{
				Name:        "githubBearer",
				Type:        SecuritySchemeHTTP,
				Scheme:      "bearer",
				Description: "GitHub personal access token, GitHub App token, OAuth token, or GitHub Actions GITHUB_TOKEN carried in the Authorization header.",
			},
			{
				Name:          "githubBasic",
				Type:          SecuritySchemeHTTP,
				Scheme:        "basic",
				Description:   "GitHub app or OAuth app client ID and client secret for app management endpoints that require basic authentication.",
				ParameterName: "Authorization",
			},
		},
		RootSecurity: []SecurityRequirement{{Scheme: "githubBearer"}},
		SourceRefs: []string{
			"https://raw.githubusercontent.com/github/rest-api-description/main/descriptions/api.github.com/api.github.com.json",
			"https://docs.github.com/en/rest/authentication/authenticating-to-the-rest-api",
			"https://github.com/github/rest-api-description",
		},
		SourceNote: "GitHub's official OpenAPI description has no security schemes or security requirements; official REST docs describe Authorization bearer tokens for most requests and basic authentication for some app-management endpoints.",
	},
	{
		ID:         "gitlab-openapi-v2-auth-review",
		ProviderID: "gitlab",
		SpecRefID:  "gitlab-openapi-v2",
		Status:     AuthStatusPresentIncomplete,
		SecuritySchemes: []SecurityScheme{
			{
				Name:          "access_token_header",
				Type:          SecuritySchemeAPIKey,
				In:            APIKeyInHeader,
				ParameterName: "PRIVATE-TOKEN",
				Description:   "GitLab personal, project, group, or impersonation access token carried in the PRIVATE-TOKEN header.",
			},
			{
				Name:          "access_token_query",
				Type:          SecuritySchemeAPIKey,
				In:            APIKeyInQuery,
				ParameterName: "private_token",
				Description:   "GitLab access token carried in the private_token query parameter.",
			},
			{
				Name:        "gitlabBearer",
				Type:        SecuritySchemeHTTP,
				Scheme:      "bearer",
				Description: "GitLab OAuth 2.0 token or access token carried in the Authorization header.",
			},
		},
		RootSecurity: []SecurityRequirement{{Scheme: "access_token_header"}, {Scheme: "gitlabBearer"}},
		SourceRefs: []string{
			"https://gitlab.com/gitlab-org/gitlab/-/raw/master/doc/api/openapi/openapi_v2.yaml",
			"https://docs.gitlab.com/api/rest/authentication/",
			"https://docs.gitlab.com/api/openapi/openapi_interactive/",
		},
		SourceNote: "GitLab's official Swagger 2.0 document includes PRIVATE-TOKEN security definitions but no root security; official docs also describe OAuth bearer tokens and other token forms, so keep this as present-incomplete auth metadata.",
	},
	{
		ID:         "gmail-discovery-auth-overlay",
		ProviderID: "gmail",
		SpecRefID:  "gmail-discovery-v1",
		Status:     AuthStatusOverlayRequired,
		SecuritySchemes: []SecurityScheme{
			googleOAuthScheme("googleOAuth2", []string{
				"https://www.googleapis.com/auth/gmail.readonly",
				"https://www.googleapis.com/auth/gmail.send",
				"https://www.googleapis.com/auth/gmail.modify",
			}),
		},
		RootSecurity: []SecurityRequirement{{Scheme: "googleOAuth2", Scopes: []string{"https://www.googleapis.com/auth/gmail.readonly"}}},
		SourceRefs: []string{
			"https://gmail.googleapis.com/$discovery/rest?version=v1",
			"https://developers.google.com/workspace/gmail/api/auth/scopes",
		},
		SourceNote: "Gmail publishes an official Google Discovery document rather than OpenAPI; Google documents OAuth scopes separately, so OpenAPI-only consumers need an advisory OAuth2 overlay.",
	},
	{
		ID:         "google-calendar-discovery-auth-overlay",
		ProviderID: "google-calendar",
		SpecRefID:  "calendar-discovery-v3",
		Status:     AuthStatusOverlayRequired,
		SecuritySchemes: []SecurityScheme{
			googleOAuthScheme("googleOAuth2", []string{
				"https://www.googleapis.com/auth/calendar.calendarlist.readonly",
				"https://www.googleapis.com/auth/calendar.calendars",
				"https://www.googleapis.com/auth/calendar.events",
				"https://www.googleapis.com/auth/calendar.events.readonly",
			}),
		},
		RootSecurity: []SecurityRequirement{{Scheme: "googleOAuth2", Scopes: []string{"https://www.googleapis.com/auth/calendar.events.readonly"}}},
		SourceRefs: []string{
			"https://www.googleapis.com/discovery/v1/apis/calendar/v3/rest",
			"https://developers.google.com/workspace/calendar/api/auth",
		},
		SourceNote: "Google Calendar publishes an official Google Discovery document rather than OpenAPI; Google documents OAuth scopes separately, so OpenAPI-only consumers need an advisory OAuth2 overlay.",
	},
	{
		ID:         "google-drive-discovery-auth-overlay",
		ProviderID: "google-drive",
		SpecRefID:  "drive-discovery-v3",
		Status:     AuthStatusOverlayRequired,
		SecuritySchemes: []SecurityScheme{
			googleOAuthScheme("googleOAuth2", []string{
				"https://www.googleapis.com/auth/drive.metadata.readonly",
				"https://www.googleapis.com/auth/drive.file",
				"https://www.googleapis.com/auth/drive.readonly",
			}),
		},
		RootSecurity: []SecurityRequirement{{Scheme: "googleOAuth2", Scopes: []string{"https://www.googleapis.com/auth/drive.metadata.readonly"}}},
		SourceRefs: []string{
			"https://www.googleapis.com/discovery/v1/apis/drive/v3/rest",
			"https://developers.google.com/workspace/drive/api/guides/api-specific-auth",
		},
		SourceNote: "Google Drive publishes an official Google Discovery document rather than OpenAPI; Google documents Drive OAuth scopes separately, so OpenAPI-only consumers need an advisory OAuth2 overlay.",
	},
	{
		ID:         "google-sheets-discovery-auth-overlay",
		ProviderID: "google-sheets",
		SpecRefID:  "sheets-discovery-v4",
		Status:     AuthStatusOverlayRequired,
		SecuritySchemes: []SecurityScheme{
			googleOAuthScheme("googleOAuth2", []string{
				"https://www.googleapis.com/auth/drive.file",
				"https://www.googleapis.com/auth/spreadsheets",
				"https://www.googleapis.com/auth/spreadsheets.readonly",
			}),
		},
		RootSecurity: []SecurityRequirement{{Scheme: "googleOAuth2", Scopes: []string{"https://www.googleapis.com/auth/spreadsheets.readonly"}}},
		SourceRefs: []string{
			"https://sheets.googleapis.com/$discovery/rest?version=v4",
			"https://developers.google.com/workspace/sheets/api/scopes",
		},
		SourceNote: "Google Sheets publishes an official Google Discovery document rather than OpenAPI; Google documents OAuth scopes separately, so OpenAPI-only consumers need an advisory OAuth2 overlay.",
	},
	{
		ID:         "hubspot-public-api-auth-overlay",
		ProviderID: "hubspot",
		SpecRefID:  "hubspot-public-api-spec-index",
		Status:     AuthStatusOverlayRequired,
		SecuritySchemes: []SecurityScheme{
			{
				Name:        "hubspotBearer",
				Type:        SecuritySchemeHTTP,
				Scheme:      "bearer",
				Description: "HubSpot OAuth or private app access token carried in the Authorization header.",
			},
		},
		RootSecurity: []SecurityRequirement{{Scheme: "hubspotBearer"}},
		SourceRefs: []string{
			"https://api.hubapi.com/public/api/spec/v1/specs",
			"https://developers.hubspot.com/docs/apps/legacy-apps/authentication/intro-to-auth",
			"https://developers.hubspot.com/docs/api/private-apps",
		},
		SourceNote: "HubSpot exposes an official split OpenAPI spec index; official docs describe OAuth and private app bearer tokens, so selected specs should be checked with an advisory bearer-token overlay.",
	},
	{
		ID:         "mailchimp-marketing-auth-overlay",
		ProviderID: "mailchimp",
		SpecRefID:  "mailchimp-marketing-api-docs",
		Status:     AuthStatusOverlayRequired,
		SecuritySchemes: []SecurityScheme{
			{
				Name:        "mailchimpBasic",
				Type:        SecuritySchemeHTTP,
				Scheme:      "basic",
				Description: "Mailchimp Marketing API key carried with HTTP Basic authentication as anystring:api_key.",
			},
			{
				Name:        "mailchimpOAuth2",
				Type:        SecuritySchemeHTTP,
				Scheme:      "bearer",
				Description: "Mailchimp OAuth 2 access token carried in the Authorization header.",
			},
		},
		RootSecurity: []SecurityRequirement{{Scheme: "mailchimpBasic"}, {Scheme: "mailchimpOAuth2"}},
		SourceRefs: []string{
			"https://mailchimp.com/developer/marketing/guides/quick-start/",
			"https://mailchimp.com/developer/marketing/docs/fundamentals/",
			"https://mailchimp.com/developer/marketing/api/",
		},
		SourceNote: "Mailchimp Marketing API docs describe API key authentication and OAuth for the Marketing API; because this catalog entry has no recorded downloadable OpenAPI document, OpenAPI-only consumers need advisory security metadata.",
	},
	{
		ID:         "linear-graphql-auth-overlay",
		ProviderID: "linear",
		SpecRefID:  "linear-graphql-docs",
		Status:     AuthStatusOverlayRequired,
		SecuritySchemes: []SecurityScheme{
			{
				Name:         "linearBearer",
				Type:         SecuritySchemeHTTP,
				Scheme:       "bearer",
				BearerFormat: "API key or OAuth 2.0 access token",
				Description:  "Linear personal API key or OAuth access token carried as a bearer token in the Authorization header.",
			},
		},
		RootSecurity: []SecurityRequirement{{Scheme: "linearBearer"}},
		SourceRefs: []string{
			"https://linear.app/developers/graphql",
			"https://linear.app/docs/api/",
		},
		SourceNote: "Linear's GraphQL docs describe personal API keys and OAuth2 bearer tokens; OpenAPI-only imports need advisory bearer-token security metadata.",
	},
	{
		ID:         "microsoft-graph-v1-auth-overlay",
		ProviderID: "microsoft-graph",
		SpecRefID:  "microsoft-graph-v1-openapi",
		Status:     AuthStatusOverlayRequired,
		SecuritySchemes: []SecurityScheme{
			{
				Name:        "microsoftGraphBearer",
				Type:        SecuritySchemeHTTP,
				Scheme:      "bearer",
				Description: "Microsoft identity platform access token carried as a bearer token in the Authorization header.",
			},
		},
		RootSecurity: []SecurityRequirement{{Scheme: "microsoftGraphBearer"}},
		SourceRefs: []string{
			"https://raw.githubusercontent.com/microsoftgraph/msgraph-metadata/master/openapi/v1.0/openapi.yaml",
			"https://learn.microsoft.com/en-us/graph/auth/auth-concepts",
			"https://github.com/microsoftgraph/msgraph-metadata",
		},
		SourceNote: "Microsoft Graph's official OpenAPI v1.0 document lacks OpenAPI security schemes; official docs describe Microsoft identity platform access tokens sent as Authorization bearer tokens.",
	},
	{
		ID:         "monday-graphql-auth-overlay",
		ProviderID: "monday-com",
		SpecRefID:  "monday-api-docs",
		Status:     AuthStatusOverlayRequired,
		SecuritySchemes: []SecurityScheme{
			{
				Name:          "mondayAuthorization",
				Type:          SecuritySchemeAPIKey,
				In:            APIKeyInHeader,
				ParameterName: "Authorization",
				Description:   "Monday.com API token or OAuth-generated token carried in the Authorization header.",
			},
		},
		RootSecurity: []SecurityRequirement{{Scheme: "mondayAuthorization"}},
		SourceRefs: []string{
			"https://developer.monday.com/api-reference/docs",
			"https://developer.monday.com/api-reference/docs/getting-started",
			"https://support.monday.com/hc/en-us/articles/360005144659-Does-monday-com-have-an-API",
		},
		SourceNote: "Monday.com GraphQL docs describe Authorization-header API tokens and OAuth-generated tokens; OpenAPI-only imports need advisory token-header security metadata.",
	},
	{
		ID:         "openweathermap-api-key-overlay",
		ProviderID: "openweathermap",
		SpecRefID:  "openweathermap-api-docs",
		Status:     AuthStatusOverlayRequired,
		SecuritySchemes: []SecurityScheme{
			{
				Name:          "openWeatherAPIKey",
				Type:          SecuritySchemeAPIKey,
				In:            APIKeyInQuery,
				ParameterName: "appid",
				Description:   "OpenWeather API key passed as the appid query parameter.",
			},
		},
		RootSecurity: []SecurityRequirement{{Scheme: "openWeatherAPIKey"}},
		SourceRefs: []string{
			"https://openweathermap.org/api",
			"https://openweathermap.org/appid",
		},
		SourceNote: "OpenWeather has human API docs in this catalog entry but no recorded official OpenAPI document; docs describe API key usage, so OpenAPI imports need an advisory appid query-key overlay.",
	},
	{
		ID:         "quickbooks-online-oauth-overlay",
		ProviderID: "quickbooks",
		SpecRefID:  "quickbooks-online-api-docs",
		Status:     AuthStatusOverlayRequired,
		SecuritySchemes: []SecurityScheme{
			{
				Name:         "quickbooksOAuth2",
				Type:         SecuritySchemeHTTP,
				Scheme:       "bearer",
				BearerFormat: "OAuth 2.0 access token",
				Description:  "QuickBooks Online OAuth 2.0 access token carried in the Authorization header.",
			},
		},
		RootSecurity: []SecurityRequirement{{Scheme: "quickbooksOAuth2"}},
		SourceRefs: []string{
			"https://developer.intuit.com/app/developer/qbo/docs/learn/explore-the-quickbooks-online-api",
			"https://developer.intuit.com/app/developer/qbo/docs/develop/authentication-and-authorization/oauth-2.0",
		},
		SourceNote: "QuickBooks Online REST API docs describe OAuth 2.0 authorization for sandbox and production companies; OpenAPI-only imports need advisory bearer-token security metadata.",
	},
	{
		ID:         "salesforce-rest-auth-overlay",
		ProviderID: "salesforce",
		SpecRefID:  "salesforce-rest-docs",
		Status:     AuthStatusOverlayRequired,
		SecuritySchemes: []SecurityScheme{
			{
				Name:        "salesforceBearer",
				Type:        SecuritySchemeHTTP,
				Scheme:      "bearer",
				Description: "Salesforce OAuth 2.0 access token carried in the Authorization header.",
			},
		},
		RootSecurity: []SecurityRequirement{{Scheme: "salesforceBearer"}},
		SourceRefs: []string{
			"https://developer.salesforce.com/docs/atlas.en-us.api_rest.meta/api_rest/intro_rest.htm",
			"https://developer.salesforce.com/docs/atlas.en-us.api_rest.meta/api_rest/quickstart_oauth.htm",
			"https://help.salesforce.com/s/articleView?id=release-notes.rn_api_rest.htm&language=en_US&release=236&type=5",
		},
		SourceNote: "Salesforce has human REST API docs and org-side OpenAPI generation notes in this catalog entry but no recorded stable public OpenAPI document; OpenAPI-only consumers need an advisory bearer-token overlay.",
	},
	{
		ID:         "shopify-admin-rest-auth-overlay",
		ProviderID: "shopify",
		SpecRefID:  "shopify-admin-rest-docs",
		Status:     AuthStatusOverlayRequired,
		SecuritySchemes: []SecurityScheme{
			{
				Name:          "shopifyAccessToken",
				Type:          SecuritySchemeAPIKey,
				In:            APIKeyInHeader,
				ParameterName: "X-Shopify-Access-Token",
				Description:   "Shopify Admin API access token carried in the X-Shopify-Access-Token header.",
			},
		},
		RootSecurity: []SecurityRequirement{{Scheme: "shopifyAccessToken"}},
		SourceRefs: []string{
			"https://shopify.dev/docs/api/admin-rest",
			"https://shopify.dev/api/admin-rest/usage/access-scopes",
			"https://shopify.dev/docs/api/admin-rest/usage/versioning",
		},
		SourceNote: "Shopify has human REST Admin API docs in this catalog entry but no recorded official OpenAPI document; docs describe Admin API access tokens and access scopes, so OpenAPI imports need an advisory header-token overlay.",
	},
	{
		ID:         "servicenow-rest-auth-overlay",
		ProviderID: "servicenow",
		SpecRefID:  "servicenow-rest-api-docs",
		Status:     AuthStatusOverlayRequired,
		SecuritySchemes: []SecurityScheme{
			{
				Name:        "servicenowBasic",
				Type:        SecuritySchemeHTTP,
				Scheme:      "basic",
				Description: "ServiceNow REST API Basic authentication with a user name and password authorized by instance ACLs.",
			},
			{
				Name:         "servicenowOAuth2",
				Type:         SecuritySchemeHTTP,
				Scheme:       "bearer",
				BearerFormat: "OAuth 2.0 access token",
				Description:  "ServiceNow OAuth 2.0 access token carried in the Authorization header.",
			},
		},
		RootSecurity: []SecurityRequirement{{Scheme: "servicenowBasic"}, {Scheme: "servicenowOAuth2"}},
		SourceRefs: []string{
			"https://www.servicenow.com/docs/r/api-reference/rest-api-explorer/c_RESTAPI.html",
			"https://www.servicenow.com/docs/r/api-reference/rest-api-explorer/export-openapi-specification.html",
		},
		SourceNote: "ServiceNow REST API docs describe Basic authentication and OAuth for instance APIs; per-instance OpenAPI exports may need this advisory security overlay when exported metadata is absent or incomplete.",
	},
	{
		ID:         "slack-web-api-auth-review",
		ProviderID: "slack",
		SpecRefID:  "slack-web-openapi-v2",
		Status:     AuthStatusPresentIncomplete,
		SecuritySchemes: []SecurityScheme{
			{
				Name: "slackAuth",
				Type: SecuritySchemeOAuth2,
				Flows: []OAuthFlow{
					{
						Type:             OAuthFlowAuthorizationCode,
						AuthorizationURL: "https://slack.com/oauth/authorize",
						TokenURL:         "https://slack.com/api/oauth.access",
						Scopes:           []string{"channels:read", "chat:write"},
					},
				},
				Description: "Slack Web API OAuth metadata from the archived official OpenAPI document.",
			},
		},
		SourceRefs: []string{
			"https://raw.githubusercontent.com/slackapi/slack-api-specs/master/web-api/slack_web_openapi_v2_without_examples.json",
			"https://api.slack.com/web",
			"https://api.slack.com/authentication/token-types",
		},
		SourceNote: "Slack's official OpenAPI document includes OAuth metadata and operation security, but the repository is archived and lacks root security; keep this as a present-incomplete review overlay rather than treating it as current provider truth.",
	},
	{
		ID:         "telegram-bot-token-auth-overlay",
		ProviderID: "telegram",
		SpecRefID:  "telegram-bot-api-docs",
		Status:     AuthStatusOverlayRequired,
		SecuritySchemes: []SecurityScheme{
			{
				Name:        "telegramBotToken",
				Type:        SecuritySchemeHTTP,
				Scheme:      "bearer",
				Description: "Advisory placeholder for Telegram Bot API token authentication; official requests embed the token in the URL path segment as bot<token> rather than an Authorization header.",
			},
		},
		RootSecurity: []SecurityRequirement{{Scheme: "telegramBotToken"}},
		SourceRefs:   []string{"https://core.telegram.org/bots/api"},
		SourceNote:   "Telegram Bot API docs define calls under https://api.telegram.org/bot<token>/METHOD_NAME; OpenAPI security schemes cannot model path-token auth exactly, so this overlay flags the token requirement for downstream review.",
	},
	{
		ID:         "typeform-rest-auth-overlay",
		ProviderID: "typeform",
		SpecRefID:  "typeform-developer-docs",
		Status:     AuthStatusOverlayRequired,
		SecuritySchemes: []SecurityScheme{
			{
				Name:         "typeformBearer",
				Type:         SecuritySchemeHTTP,
				Scheme:       "bearer",
				BearerFormat: "Personal access token or OAuth 2.0 access token",
				Description:  "Typeform personal access token or OAuth access token carried in the Authorization header.",
			},
		},
		RootSecurity: []SecurityRequirement{{Scheme: "typeformBearer"}},
		SourceRefs: []string{
			"https://www.typeform.com/developers/get-started/",
			"https://www.typeform.com/developers/get-started/personal-access-token/",
			"https://www.typeform.com/developers/get-started/applications/",
		},
		SourceNote: "Typeform docs describe personal access tokens and OAuth 2.0 access tokens passed in the Authorization header; OpenAPI-only imports need advisory bearer-token security metadata.",
	},
	{
		ID:         "zendesk-support-auth-review",
		ProviderID: "zendesk",
		SpecRefID:  "zendesk-support-api-docs",
		Status:     AuthStatusPresentIncomplete,
		SecuritySchemes: []SecurityScheme{
			{
				Name:        "zendeskBasic",
				Type:        SecuritySchemeHTTP,
				Scheme:      "basic",
				Description: "Zendesk Support API token credentials carried with HTTP Basic authentication as email_address/token:api_token.",
			},
			{
				Name:        "zendeskBearer",
				Type:        SecuritySchemeHTTP,
				Scheme:      "bearer",
				Description: "Zendesk OAuth or global OAuth access token carried in the Authorization header.",
			},
		},
		RootSecurity: []SecurityRequirement{{Scheme: "zendeskBasic"}, {Scheme: "zendeskBearer"}},
		SourceRefs: []string{
			"https://developer.zendesk.com/api-reference/introduction/security-and-auth/",
			"https://developer.zendesk.com/api-reference/ticketing/introduction/",
			"https://developer.zendesk.com/documentation/conversations/references/openapi-specification/",
		},
		SourceNote: "Zendesk's official Sunshine Conversations OpenAPI covers messaging security; broader Support API docs describe API-token basic auth and OAuth bearer tokens, so catalog consumers should treat support/ticketing auth as reviewed but endpoint coverage as incomplete without a user OpenAPI document.",
	},
	{
		ID:         "zoom-api-v2-auth-review",
		ProviderID: "zoom",
		SpecRefID:  "zoom-api-v2-openapi",
		Status:     AuthStatusPresentIncomplete,
		SecuritySchemes: []SecurityScheme{
			{
				Name:         "zoomBearer",
				Type:         SecuritySchemeHTTP,
				Scheme:       "bearer",
				BearerFormat: "OAuth access token",
				Description:  "Zoom OAuth access token carried in the Authorization header for current REST API docs.",
			},
		},
		RootSecurity: []SecurityRequirement{{Scheme: "zoomBearer"}},
		SourceRefs: []string{
			"https://raw.githubusercontent.com/zoom/api/master/openapi.v2.json",
			"https://developers.zoom.us/docs/api/meetings/",
			"https://developers.zoom.us/docs/integrations/oauth/",
		},
		SourceNote: "Zoom's saved official Swagger 2.0 artifact uses older access_token query security, while current docs list OAuth scopes and bearer access-token usage; keep this as present-incomplete advisory auth review metadata.",
	},
}

func googleOAuthScheme(name string, scopes []string) SecurityScheme {
	return SecurityScheme{
		Name: name,
		Type: SecuritySchemeOAuth2,
		Flows: []OAuthFlow{
			{
				Type:             OAuthFlowAuthorizationCode,
				AuthorizationURL: "https://accounts.google.com/o/oauth2/v2/auth",
				TokenURL:         "https://oauth2.googleapis.com/token",
				Scopes:           sortedStrings(scopes),
			},
		},
		Scopes:      sortedStrings(scopes),
		Description: "Google OAuth 2.0 authorization code flow with provider-specific scopes.",
	}
}

func sortSecurityOverlays(overlays []SecurityOverlay) {
	sort.SliceStable(overlays, func(i, j int) bool {
		if overlays[i].ProviderID == overlays[j].ProviderID {
			return overlays[i].ID < overlays[j].ID
		}
		return overlays[i].ProviderID < overlays[j].ProviderID
	})
}

func cloneSecurityOverlays(in []SecurityOverlay) []SecurityOverlay {
	out := make([]SecurityOverlay, len(in))
	for i, overlay := range in {
		out[i] = cloneSecurityOverlay(overlay)
	}
	return out
}

func cloneSecurityOverlay(overlay SecurityOverlay) SecurityOverlay {
	overlay.SecuritySchemes = cloneSecuritySchemes(overlay.SecuritySchemes)
	overlay.RootSecurity = cloneSecurityRequirements(overlay.RootSecurity)
	overlay.OperationSecurity = cloneOperationSecurity(overlay.OperationSecurity)
	overlay.SourceRefs = append([]string(nil), overlay.SourceRefs...)
	return overlay
}
