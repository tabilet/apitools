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
