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
	ID                string                   `json:"id"`
	ProviderID        string                   `json:"provider_id"`
	SpecRefID         string                   `json:"spec_ref_id,omitempty"`
	Status            AuthCompletenessStatus   `json:"status"`
	SecuritySchemes   []SecurityScheme         `json:"security_schemes,omitempty"`
	RootSecurity      []SecurityRequirement    `json:"root_security,omitempty"`
	RootSecuritySets  []SecurityRequirementSet `json:"root_security_sets,omitempty"`
	OperationSecurity []OperationSecurity      `json:"operation_security,omitempty"`
	SourceRefs        []string                 `json:"source_refs,omitempty"`
	SourceNote        string                   `json:"source_note"`
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
	for i, set := range overlay.RootSecuritySets {
		if err := validateSecurityRequirementSet(set, schemes); err != nil {
			return fmt.Errorf("overlay %q root security sets[%d]: %w", overlay.ID, i, err)
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
		ID:         "action-network-api-auth-overlay",
		ProviderID: "action-network",
		SpecRefID:  "action-network-api-v2-docs",
		Status:     AuthStatusOverlayRequired,
		SecuritySchemes: []SecurityScheme{{
			Name:          "actionNetworkAPIToken",
			Type:          SecuritySchemeAPIKey,
			In:            APIKeyInHeader,
			ParameterName: "OSDI-API-Token",
			Description:   "Action Network API key carried in the OSDI-API-Token header for most API v2 endpoints.",
		}},
		RootSecurity: []SecurityRequirement{{Scheme: "actionNetworkAPIToken"}},
		SourceRefs: []string{
			"https://actionnetwork.org/docs/v2/",
		},
		SourceNote: "Action Network has human REST API v2 docs in this catalog entry but no recorded official OpenAPI document; docs describe OSDI-API-Token header authentication for most endpoints.",
	},
	{
		ID:         "beeminder-api-auth-overlay",
		ProviderID: "beeminder",
		SpecRefID:  "beeminder-api-docs",
		Status:     AuthStatusOverlayRequired,
		SecuritySchemes: []SecurityScheme{
			{
				Name:          "beeminderAuthToken",
				Type:          SecuritySchemeAPIKey,
				In:            APIKeyInQuery,
				ParameterName: "auth_token",
				Description:   "Beeminder personal authentication token supplied as auth_token in request parameters.",
			},
			{
				Name:        "beeminderBearer",
				Type:        SecuritySchemeHTTP,
				Scheme:      "bearer",
				Description: "Beeminder OAuth access token supplied as an Authorization bearer token.",
			},
		},
		RootSecurity: []SecurityRequirement{{Scheme: "beeminderAuthToken"}, {Scheme: "beeminderBearer"}},
		SourceRefs: []string{
			"https://api.beeminder.com/",
			"https://www.beeminder.com/api/v1/auth_token.json",
			"https://www.beeminder.com/apps/new",
		},
		SourceNote: "Beeminder has human API docs in this catalog entry but no recorded official OpenAPI document; docs describe personal auth_token parameters and OAuth access tokens, including Authorization: Bearer usage.",
	},
	{
		ID:         "adalo-api-auth-overlay",
		ProviderID: "adalo",
		SpecRefID:  "adalo-api-docs",
		Status:     AuthStatusOverlayRequired,
		SecuritySchemes: []SecurityScheme{{
			Name:        "adaloBearer",
			Type:        SecuritySchemeHTTP,
			Scheme:      "bearer",
			Description: "Adalo app API key carried as an Authorization bearer token.",
		}},
		RootSecurity: []SecurityRequirement{{Scheme: "adaloBearer"}},
		SourceRefs: []string{
			"https://help.adalo.com/integrations/the-adalo-api/collections",
			"https://help.adalo.com/integrations/the-adalo-api/push-notifications",
		},
		SourceNote: "Adalo has human API docs in this catalog entry but no recorded official OpenAPI document; docs describe app API-key authentication and push notification requests with Authorization: Bearer.",
	},
	{
		ID:         "clearbit-api-auth-overlay",
		ProviderID: "clearbit",
		SpecRefID:  "clearbit-prospector-zapier-docs",
		Status:     AuthStatusOverlayRequired,
		SecuritySchemes: []SecurityScheme{{
			Name:        "clearbitBasic",
			Type:        SecuritySchemeHTTP,
			Scheme:      "basic",
			Description: "Clearbit secret API key supplied with HTTP Basic authentication.",
		}},
		RootSecurity: []SecurityRequirement{{Scheme: "clearbitBasic"}},
		SourceRefs: []string{
			"https://help.clearbit.com/hc/en-us/articles/6480449602967-Integrate-Clearbit-Prospector-with-Google-Sheets-Using-Zapier",
			"https://help.clearbit.com/hc/en-us/articles/6045527495191-How-Do-I-Access-My-Clearbit-API-Key",
		},
		SourceNote: "Clearbit has human API support docs in this catalog entry but no recorded official OpenAPI document; docs show Prospector and Enrichment API examples using a secret API key with HTTP Basic authentication.",
	},
	{
		ID:         "affinity-api-auth-overlay",
		ProviderID: "affinity",
		SpecRefID:  "affinity-v1-api-reference",
		Status:     AuthStatusOverlayRequired,
		SecuritySchemes: []SecurityScheme{
			{
				Name:        "affinityBasic",
				Type:        SecuritySchemeHTTP,
				Scheme:      "basic",
				Description: "Affinity API key supplied as the HTTP Basic password with no username.",
			},
			{
				Name:        "affinityBearer",
				Type:        SecuritySchemeHTTP,
				Scheme:      "bearer",
				Description: "Affinity API key supplied as an Authorization bearer token.",
			},
		},
		RootSecurity: []SecurityRequirement{{Scheme: "affinityBasic"}, {Scheme: "affinityBearer"}},
		SourceRefs: []string{
			"https://api-docs.affinity.co/",
			"https://support.affinity.co/s/article/How-to-Create-and-Manage-API-Keys",
		},
		SourceNote: "Affinity has human V1 API docs in this catalog entry but no recorded official OpenAPI document; docs describe HTTP Basic and bearer API-key authentication alternatives.",
	},
	{
		ID:         "agile-crm-api-auth-overlay",
		ProviderID: "agile-crm",
		SpecRefID:  "agile-crm-rest-api-github",
		Status:     AuthStatusOverlayRequired,
		SecuritySchemes: []SecurityScheme{{
			Name:        "agileCRMBasic",
			Type:        SecuritySchemeHTTP,
			Scheme:      "basic",
			Description: "Agile CRM user email and REST API key supplied with HTTP Basic authentication.",
		}},
		RootSecurity: []SecurityRequirement{{Scheme: "agileCRMBasic"}},
		SourceRefs: []string{
			"https://www.agilecrm.com/api",
			"https://github.com/agilecrm/rest-api",
		},
		SourceNote: "Agile CRM has official human/GitHub REST API docs in this catalog entry but no recorded official OpenAPI document; docs describe email plus API key using HTTP Basic authentication.",
	},
	{
		ID:         "copper-api-auth-overlay",
		ProviderID: "copper",
		SpecRefID:  "copper-developer-api-auth-docs",
		Status:     AuthStatusOverlayRequired,
		SecuritySchemes: []SecurityScheme{
			{
				Name:          "copperAccessToken",
				Type:          SecuritySchemeAPIKey,
				In:            APIKeyInHeader,
				ParameterName: "X-PW-AccessToken",
				Description:   "Copper Developer API token carried in the X-PW-AccessToken header.",
			},
			{
				Name:          "copperUserEmail",
				Type:          SecuritySchemeAPIKey,
				In:            APIKeyInHeader,
				ParameterName: "X-PW-UserEmail",
				Description:   "Copper user email for the user who generated the API token.",
			},
			{
				Name:          "copperApplication",
				Type:          SecuritySchemeAPIKey,
				In:            APIKeyInHeader,
				ParameterName: "X-PW-Application",
				Description:   "Copper Developer API application header, commonly developer_api for the legacy developer API.",
			},
		},
		RootSecurity: []SecurityRequirement{{Scheme: "copperAccessToken"}, {Scheme: "copperUserEmail"}, {Scheme: "copperApplication"}},
		RootSecuritySets: []SecurityRequirementSet{{
			Requirements: []SecurityRequirement{{Scheme: "copperAccessToken"}, {Scheme: "copperUserEmail"}, {Scheme: "copperApplication"}},
		}},
		SourceRefs: []string{
			"https://developer.copper.com/introduction/authentication.html",
			"https://developer.copper.com/",
		},
		SourceNote: "Copper has human Developer API docs in this catalog entry but no recorded official OpenAPI document; docs describe token-based authentication with the token and user email included in request headers.",
	},
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
	parseableSpecAuthReviewOverlay(
		"asana-openapi-v1-auth-review",
		"asana",
		"asana-openapi-v1",
		[]SecurityScheme{
			{
				Name:        "personalAccessToken",
				Type:        SecuritySchemeHTTP,
				Scheme:      "bearer",
				Description: "Asana personal access token carried as an Authorization bearer token.",
			},
			{
				Name: "oauth2",
				Type: SecuritySchemeOAuth2,
				Flows: []OAuthFlow{{
					Type:             OAuthFlowAuthorizationCode,
					AuthorizationURL: "https://app.asana.com/-/oauth_authorize",
					TokenURL:         "https://app.asana.com/-/oauth_token",
					RefreshURL:       "https://app.asana.com/-/oauth_token",
					Scopes:           []string{"default", "openid", "email"},
				}},
				Description: "Asana OAuth 2.0 authorization-code flow from the official OpenAPI artifact.",
			},
		},
		[]SecurityRequirement{{Scheme: "personalAccessToken"}, {Scheme: "oauth2"}},
		[]string{
			"https://raw.githubusercontent.com/Asana/openapi/master/defs/asana_oas.yaml",
			"https://developers.asana.com/docs/oauth",
			"https://developers.asana.com/docs/personal-access-token",
		},
		"Asana's official OpenAPI artifact is tracked as parseable-openapi-invalid by strict refresh validation, so bearer and OAuth security metadata is carried as review-only overlay guidance.",
	),
	awsSigV4Overlay("aws-s3-sigv4-auth-overlay", "aws-s3", "aws-s3-smithy-model", "Amazon S3", []string{
		"https://docs.aws.amazon.com/AmazonS3/latest/API/sig-v4-authenticating-requests.html",
		"https://docs.aws.amazon.com/general/latest/gr/signature-version-4.html",
	}),
	awsSigV4Overlay("aws-lambda-sigv4-auth-overlay", "aws-lambda", "aws-lambda-smithy-model", "AWS Lambda", []string{
		"https://docs.aws.amazon.com/lambda/latest/api/welcome.html",
		"https://docs.aws.amazon.com/lambda/latest/api/CommonParameters.html",
	}),
	awsSigV4Overlay("aws-sns-sigv4-auth-overlay", "aws-sns", "aws-sns-smithy-model", "Amazon SNS", []string{
		"https://docs.aws.amazon.com/sns/latest/api/welcome.html",
		"https://docs.aws.amazon.com/sns/latest/api/CommonParameters.html",
	}),
	parseableSpecAuthReviewOverlay(
		"bitbucket-cloud-swagger-v2-auth-review",
		"bitbucket",
		"bitbucket-cloud-swagger-v2",
		[]SecurityScheme{
			{
				Name:        "basic",
				Type:        SecuritySchemeHTTP,
				Scheme:      "basic",
				Description: "Bitbucket Cloud basic authentication metadata from the official Swagger artifact.",
			},
			{
				Name: "oauth2",
				Type: SecuritySchemeOAuth2,
				Flows: []OAuthFlow{{
					Type:             OAuthFlowAuthorizationCode,
					AuthorizationURL: "https://bitbucket.org/site/oauth2/authorize",
					TokenURL:         "https://bitbucket.org/site/oauth2/access_token",
					Scopes:           []string{"repository", "repository:write", "project", "pullrequest", "account"},
				}},
				Description: "Bitbucket Cloud OAuth 2.0 flow and scopes from the official Swagger artifact.",
			},
			{
				Name:          "api_key",
				Type:          SecuritySchemeAPIKey,
				In:            APIKeyInHeader,
				ParameterName: "Authorization",
				Description:   "Bitbucket Cloud API token or app password carried in the Authorization header.",
			},
		},
		[]SecurityRequirement{{Scheme: "api_key"}, {Scheme: "oauth2"}, {Scheme: "basic"}},
		[]string{
			"https://api.bitbucket.org/swagger.json",
			"https://developer.atlassian.com/cloud/bitbucket/rest/intro/",
			"https://support.atlassian.com/bitbucket-cloud/docs/using-api-tokens/",
			"https://developer.atlassian.com/cloud/bitbucket/oauth-2/",
		},
		"Bitbucket Cloud's official Swagger artifact is tracked as parseable-swagger-invalid by strict refresh validation, so API token, basic, and OAuth metadata is carried as review-only overlay guidance.",
	),
	parseableSpecAuthReviewOverlay(
		"box-platform-openapi-v3-auth-review",
		"box",
		"box-platform-openapi-v3",
		[]SecurityScheme{{
			Name: "OAuth2Security",
			Type: SecuritySchemeOAuth2,
			Flows: []OAuthFlow{{
				Type:             OAuthFlowAuthorizationCode,
				AuthorizationURL: "https://account.box.com/api/oauth2/authorize",
				TokenURL:         "https://api.box.com/oauth2/token",
				Scopes:           []string{"root_readonly", "root_readwrite", "manage_app_users", "manage_managed_users"},
			}},
			Description: "Box Platform OAuth 2.0 authorization-code security metadata from the official OpenAPI artifact.",
		}},
		[]SecurityRequirement{{Scheme: "OAuth2Security"}},
		[]string{
			"https://raw.githubusercontent.com/box/box-openapi/main/openapi.json",
			"https://developer.box.com/guides/authentication/oauth2/",
			"https://developer.box.com/reference/",
		},
		"Box's official OpenAPI artifact is tracked as parseable-openapi-invalid by strict refresh validation, so OAuth security metadata is carried as review-only overlay guidance.",
	),
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
	parseableSpecAuthReviewOverlay(
		"circleci-api-v2-auth-review",
		"circleci",
		"circleci-api-v2-openapi",
		[]SecurityScheme{
			{
				Name:          "api_key_header",
				Type:          SecuritySchemeAPIKey,
				In:            APIKeyInHeader,
				ParameterName: "Circle-Token",
				Description:   "CircleCI API token carried in the Circle-Token header.",
			},
			{
				Name:        "basic_auth",
				Type:        SecuritySchemeHTTP,
				Scheme:      "basic",
				Description: "CircleCI API token used as the HTTP Basic username with an empty password.",
			},
			{
				Name:          "api_key_query",
				Type:          SecuritySchemeAPIKey,
				In:            APIKeyInQuery,
				ParameterName: "circle-token",
				Description:   "Deprecated CircleCI query-string token form recorded in the official OpenAPI artifact.",
			},
		},
		[]SecurityRequirement{{Scheme: "api_key_header"}, {Scheme: "basic_auth"}, {Scheme: "api_key_query"}},
		[]string{
			"https://circleci.com/api/v2/openapi.json",
			"https://circleci.com/docs/guides/toolkit/managing-api-tokens/",
			"https://circleci.com/docs/guides/toolkit/api-developers-guide/",
		},
		"CircleCI's official OpenAPI artifact is tracked as parseable-openapi-invalid by strict refresh validation, so header token, basic-auth token, and deprecated query-token metadata is carried as review-only overlay guidance.",
	),
	parseableSpecAuthReviewOverlay(
		"cloudflare-api-auth-review",
		"cloudflare",
		"cloudflare-api-openapi",
		[]SecurityScheme{
			{
				Name:        "api_token",
				Type:        SecuritySchemeHTTP,
				Scheme:      "bearer",
				Description: "Cloudflare scoped API token carried as an Authorization bearer token.",
			},
			{
				Name:          "api_email",
				Type:          SecuritySchemeAPIKey,
				In:            APIKeyInHeader,
				ParameterName: "X-Auth-Email",
				Description:   "Cloudflare account email header used with legacy global API key authentication.",
			},
			{
				Name:          "api_key",
				Type:          SecuritySchemeAPIKey,
				In:            APIKeyInHeader,
				ParameterName: "X-Auth-Key",
				Description:   "Cloudflare legacy global API key header used with X-Auth-Email.",
			},
			{
				Name:          "user_service_key",
				Type:          SecuritySchemeAPIKey,
				In:            APIKeyInHeader,
				ParameterName: "X-Auth-User-Service-Key",
				Description:   "Cloudflare user service key header recorded in the official OpenAPI artifact.",
			},
		},
		[]SecurityRequirement{{Scheme: "api_token"}, {Scheme: "api_email"}, {Scheme: "api_key"}, {Scheme: "user_service_key"}},
		[]string{
			"https://raw.githubusercontent.com/cloudflare/api-schemas/main/openapi.yaml",
			"https://developers.cloudflare.com/fundamentals/api/get-started/create-token/",
			"https://developers.cloudflare.com/fundamentals/api/get-started/keys/",
		},
		"Cloudflare's official OpenAPI artifact is tracked as parseable-openapi-invalid by strict refresh validation; preferred API-token and legacy key/email metadata is carried as review-only overlay guidance.",
	),
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
		SourceNote: "ClickUp's official OpenAPI artifact is tracked as parseable-openapi-invalid by strict refresh validation. It includes an Authorization header API key scheme and root security; OAuth authorization-code endpoints are documented separately, so keep this as a present-incomplete auth review overlay.",
	},
	{
		ID:         "databricks-rest-api-auth-overlay",
		ProviderID: "databricks",
		SpecRefID:  "databricks-api-docs",
		Status:     AuthStatusOverlayRequired,
		SecuritySchemes: []SecurityScheme{
			{
				Name:        "databricksBearer",
				Type:        SecuritySchemeHTTP,
				Scheme:      "bearer",
				Description: "Databricks OAuth access token or legacy personal access token carried in the Authorization header.",
			},
		},
		RootSecurity: []SecurityRequirement{{Scheme: "databricksBearer"}},
		SourceRefs: []string{
			"https://docs.databricks.com/aws/en/dev-tools/auth",
			"https://docs.databricks.com/aws/en/dev-tools/auth/pat",
			"https://docs.databricks.com/api/workspace/introduction",
		},
		SourceNote: "Databricks has human REST API docs in this catalog entry but no recorded downloadable official OpenAPI document; official auth docs describe OAuth access tokens and legacy PATs, so OpenAPI imports need an advisory bearer-token overlay.",
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
		ID:         "discourse-api-auth-overlay",
		ProviderID: "discourse",
		SpecRefID:  "discourse-api-openapi",
		Status:     AuthStatusOverlayRequired,
		SecuritySchemes: []SecurityScheme{
			{
				Name:          "discourseAPIKey",
				Type:          SecuritySchemeAPIKey,
				In:            APIKeyInHeader,
				ParameterName: "Api-Key",
				Description:   "Discourse API key generated from the admin panel.",
			},
			{
				Name:          "discourseAPIUsername",
				Type:          SecuritySchemeAPIKey,
				In:            APIKeyInHeader,
				ParameterName: "Api-Username",
				Description:   "Discourse API username associated with the API key.",
			},
		},
		RootSecurity: []SecurityRequirement{{Scheme: "discourseAPIKey"}, {Scheme: "discourseAPIUsername"}},
		RootSecuritySets: []SecurityRequirementSet{{
			Requirements: []SecurityRequirement{{Scheme: "discourseAPIKey"}, {Scheme: "discourseAPIUsername"}},
		}},
		SourceRefs: []string{
			"https://docs.discourse.org/openapi.json",
			"https://docs.discourse.org/",
			"https://github.com/discourse/discourse_api_docs",
		},
		SourceNote: "Discourse's official OpenAPI document is importable but does not declare OpenAPI securitySchemes; its prose authentication docs describe Api-Key and Api-Username request headers.",
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
		SourceNote: "GitHub's official OpenAPI artifact is tracked as parseable-openapi-invalid by strict refresh validation and has no security schemes or security requirements; official REST docs describe Authorization bearer tokens for most requests and basic authentication for some app-management endpoints.",
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
	parseableSpecAuthReviewOverlay(
		"intercom-api-v2-15-auth-review",
		"intercom",
		"intercom-api-v2-15-openapi",
		[]SecurityScheme{{
			Name:        "bearerAuth",
			Type:        SecuritySchemeHTTP,
			Scheme:      "bearer",
			Description: "Intercom access token carried as an Authorization bearer token.",
		}},
		[]SecurityRequirement{{Scheme: "bearerAuth"}},
		[]string{
			"https://developers.intercom.com/_bundle/docs/references/%402.15/rest-api/api.intercom.io.json?download=",
			"https://developers.intercom.com/docs/references/rest-api/api.intercom.io",
			"https://developers.intercom.com/docs/build-an-integration/learn-more/authentication",
		},
		"Intercom's official v2.15 OpenAPI artifact is tracked as parseable-openapi-invalid by strict refresh validation, so bearer-token security metadata is carried as review-only overlay guidance.",
	),
	parseableSpecAuthReviewOverlay(
		"jira-cloud-platform-openapi-v3-auth-review",
		"jira-cloud",
		"jira-cloud-platform-openapi-v3",
		[]SecurityScheme{
			{
				Name: "OAuth2",
				Type: SecuritySchemeOAuth2,
				Flows: []OAuthFlow{{
					Type:             OAuthFlowAuthorizationCode,
					AuthorizationURL: "https://auth.atlassian.com/authorize",
					TokenURL:         "https://auth.atlassian.com/oauth/token",
					Scopes:           []string{"read:jira-work", "write:jira-work", "manage:jira-project"},
				}},
				Description: "Atlassian OAuth 2.0 authorization-code security metadata for Jira Cloud.",
			},
			{
				Name:        "basicAuth",
				Type:        SecuritySchemeHTTP,
				Scheme:      "basic",
				Description: "Jira Cloud basic authentication metadata from the official OpenAPI artifact.",
			},
		},
		[]SecurityRequirement{{Scheme: "OAuth2"}, {Scheme: "basicAuth"}},
		[]string{
			"https://dac-static.atlassian.com/cloud/jira/platform/swagger-v3.v3.json",
			"https://developer.atlassian.com/cloud/jira/platform/rest/v3/intro/",
			"https://developer.atlassian.com/cloud/jira/platform/oauth-2-3lo-apps/",
		},
		"Jira Cloud's official OpenAPI artifact is tracked as parseable-openapi-invalid by strict refresh validation, so OAuth2 and basic security metadata is carried as review-only overlay guidance.",
	),
	{
		ID:         "jenkins-remote-api-auth-overlay",
		ProviderID: "jenkins",
		SpecRefID:  "jenkins-api-docs",
		Status:     AuthStatusOverlayRequired,
		SecuritySchemes: []SecurityScheme{
			{
				Name:        "jenkinsBasic",
				Type:        SecuritySchemeHTTP,
				Scheme:      "basic",
				Description: "Jenkins username with API token or password using HTTP Basic authentication.",
			},
			{
				Name:          "jenkinsCrumb",
				Type:          SecuritySchemeAPIKey,
				In:            APIKeyInHeader,
				ParameterName: "Jenkins-Crumb",
				Description:   "Jenkins CSRF crumb header for POST requests when required by the instance.",
			},
		},
		RootSecurity: []SecurityRequirement{{Scheme: "jenkinsBasic"}},
		SourceRefs: []string{
			"https://www.jenkins.io/doc/book/using/remote-access-api/",
			"https://www.jenkins.io/blog/2018/07/02/new-api-token-system/",
		},
		SourceNote: "Jenkins has human Remote Access API docs in this catalog entry but no recorded official OpenAPI document; docs describe basic authentication with API tokens and instance-dependent CSRF crumbs, so OpenAPI imports need an advisory auth overlay.",
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
	parseableSpecAuthReviewOverlay(
		"notion-api-openapi-auth-review",
		"notion",
		"notion-api-openapi",
		[]SecurityScheme{
			{
				Name:        "bearerAuth",
				Type:        SecuritySchemeHTTP,
				Scheme:      "bearer",
				Description: "Notion integration token or OAuth access token carried as an Authorization bearer token.",
			},
			{
				Name:        "basicAuth",
				Type:        SecuritySchemeHTTP,
				Scheme:      "basic",
				Description: "Notion basic authentication metadata recorded in the official OpenAPI artifact.",
			},
		},
		[]SecurityRequirement{{Scheme: "bearerAuth"}},
		[]string{
			"https://developers.notion.com/openapi.json",
			"https://developers.notion.com/reference/authentication",
		},
		"Notion's official OpenAPI artifact is tracked as parseable-openapi-invalid by strict refresh validation, so bearer-token security metadata is carried as review-only overlay guidance.",
	),
	parseableSpecAuthReviewOverlay(
		"okta-management-minimal-openapi-auth-review",
		"okta",
		"okta-management-minimal-openapi",
		[]SecurityScheme{
			{
				Name:          "apiToken",
				Type:          SecuritySchemeAPIKey,
				In:            APIKeyInHeader,
				ParameterName: "Authorization",
				Description:   "Okta SSWS API token carried in the Authorization header.",
			},
			{
				Name:         "oktaOAuth2",
				Type:         SecuritySchemeHTTP,
				Scheme:       "bearer",
				BearerFormat: "OAuth 2.0 access token",
				Description:  "Okta scoped OAuth 2.0 access token carried as an Authorization bearer token.",
			},
		},
		[]SecurityRequirement{{Scheme: "apiToken"}, {Scheme: "oktaOAuth2"}},
		[]string{
			"https://raw.githubusercontent.com/okta/okta-management-openapi-spec/master/dist/current/management-minimal.yaml",
			"https://developer.okta.com/docs/api/openapi/okta-management/guides/overview/",
			"https://developer.okta.com/docs/guides/implement-oauth-for-okta/",
		},
		"Okta's official Management OpenAPI artifact is tracked as parseable-openapi-invalid by strict refresh validation, so SSWS token and scoped OAuth metadata is carried as review-only overlay guidance.",
	),
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
	parseableSpecAuthReviewOverlay(
		"pagerduty-rest-openapi-v3-auth-review",
		"pagerduty",
		"pagerduty-rest-openapi-v3",
		[]SecurityScheme{{
			Name:          "api_key",
			Type:          SecuritySchemeAPIKey,
			In:            APIKeyInHeader,
			ParameterName: "Authorization",
			Description:   "PagerDuty API token carried in the Authorization header.",
		}},
		[]SecurityRequirement{{Scheme: "api_key"}},
		[]string{
			"https://raw.githubusercontent.com/PagerDuty/api-schema/main/reference/REST/openapiv3.json",
			"https://developer.pagerduty.com/api-reference/",
		},
		"PagerDuty's official OpenAPI artifact is tracked as parseable-openapi-invalid by strict refresh validation, so API-token security metadata is carried as review-only overlay guidance.",
	),
	parseableSpecAuthReviewOverlay(
		"pipedrive-api-v2-openapi-auth-review",
		"pipedrive",
		"pipedrive-api-v2-openapi",
		[]SecurityScheme{
			{
				Name:        "basic_authentication",
				Type:        SecuritySchemeHTTP,
				Scheme:      "basic",
				Description: "Pipedrive basic authentication metadata from the official OpenAPI artifact.",
			},
			{
				Name:          "api_key",
				Type:          SecuritySchemeAPIKey,
				In:            APIKeyInHeader,
				ParameterName: "x-api-token",
				Description:   "Pipedrive API token carried in the x-api-token header.",
			},
			{
				Name: "oauth2",
				Type: SecuritySchemeOAuth2,
				Flows: []OAuthFlow{{
					Type:             OAuthFlowAuthorizationCode,
					AuthorizationURL: "https://oauth.pipedrive.com/oauth/authorize",
					TokenURL:         "https://oauth.pipedrive.com/oauth/token",
					RefreshURL:       "https://oauth.pipedrive.com/oauth/token",
					Scopes:           []string{"base", "deals:read", "deals:write"},
				}},
				Description: "Pipedrive OAuth 2.0 authorization-code security metadata from the official OpenAPI artifact.",
			},
		},
		[]SecurityRequirement{{Scheme: "api_key"}, {Scheme: "oauth2"}, {Scheme: "basic_authentication"}},
		[]string{
			"https://developers.pipedrive.com/docs/api/v1/openapi-v2.yaml",
			"https://developers.pipedrive.com/docs/api/v1",
		},
		"Pipedrive's official OpenAPI artifact is tracked as parseable-openapi-invalid by strict refresh validation, so API-token, OAuth2, and basic security metadata is carried as review-only overlay guidance.",
	),
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
		ID:         "sentry-api-auth-overlay",
		ProviderID: "sentry",
		SpecRefID:  "sentry-api-docs",
		Status:     AuthStatusOverlayRequired,
		SecuritySchemes: []SecurityScheme{
			{
				Name:        "sentryBearer",
				Type:        SecuritySchemeHTTP,
				Scheme:      "bearer",
				Description: "Sentry user, organization, or internal integration authentication token carried in the Authorization header.",
			},
			{
				Name:        "sentryBasicLegacy",
				Type:        SecuritySchemeHTTP,
				Scheme:      "basic",
				Description: "Legacy Sentry API key basic authentication for older accounts and limited endpoints.",
			},
		},
		RootSecurity: []SecurityRequirement{{Scheme: "sentryBearer"}},
		SourceRefs: []string{
			"https://docs.sentry.io/api/auth/",
			"https://docs.sentry.io/api/permissions/",
			"https://docs.sentry.io/api/",
		},
		SourceNote: "Sentry has human API docs in this catalog entry but no recorded official OpenAPI document; docs describe bearer auth tokens, limited DSN auth, and legacy API keys, so OpenAPI imports need an advisory auth overlay.",
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
		ID:         "splunk-rest-api-auth-overlay",
		ProviderID: "splunk",
		SpecRefID:  "splunk-api-docs",
		Status:     AuthStatusOverlayRequired,
		SecuritySchemes: []SecurityScheme{
			{
				Name:        "splunkBearer",
				Type:        SecuritySchemeHTTP,
				Scheme:      "bearer",
				Description: "Splunk authentication token or session key carried in the Authorization header.",
			},
			{
				Name:        "splunkBasic",
				Type:        SecuritySchemeHTTP,
				Scheme:      "basic",
				Description: "Splunk username/password basic authentication for login/session endpoints where enabled.",
			},
		},
		RootSecurity: []SecurityRequirement{{Scheme: "splunkBearer"}, {Scheme: "splunkBasic"}},
		SourceRefs: []string{
			"https://docs.splunk.com/Documentation/Splunk/latest/RESTUM/RESTusing",
			"https://docs.splunk.com/Documentation/Splunk/latest/RESTREF",
		},
		SourceNote: "Splunk has human REST API docs in this catalog entry but no recorded stable public OpenAPI document for the general Enterprise API; docs describe authentication tokens, session keys, and credentials, so OpenAPI imports need an advisory auth overlay.",
	},
	{
		ID:         "supabase-management-api-auth-review",
		ProviderID: "supabase",
		SpecRefID:  "supabase-management-api-openapi",
		Status:     AuthStatusPresentIncomplete,
		SecuritySchemes: []SecurityScheme{
			{
				Name:         "bearer",
				Type:         SecuritySchemeHTTP,
				Scheme:       "bearer",
				BearerFormat: "JWT",
				Description:  "Supabase Management API personal access token or OAuth access token carried in the Authorization header.",
			},
		},
		RootSecurity: []SecurityRequirement{{Scheme: "bearer"}},
		SourceRefs: []string{
			"https://api.supabase.com/api/v1-json",
			"https://supabase.com/docs/reference/api/introduction",
		},
		SourceNote: "Supabase's official Management API OpenAPI artifact is tracked as parseable-openapi-invalid by strict refresh validation and declares a bearer scheme but no root security requirement; official API docs state that all API requests require an Authorization bearer token.",
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
	parseableSpecAuthReviewOverlay(
		"trello-cloud-openapi-v3-auth-review",
		"trello",
		"trello-cloud-openapi-v3",
		[]SecurityScheme{
			{
				Name:          "APIKey",
				Type:          SecuritySchemeAPIKey,
				In:            APIKeyInQuery,
				ParameterName: "key",
				Description:   "Trello API key passed as the key query parameter.",
			},
			{
				Name:          "APIToken",
				Type:          SecuritySchemeAPIKey,
				In:            APIKeyInQuery,
				ParameterName: "token",
				Description:   "Trello API token passed as the token query parameter.",
			},
		},
		[]SecurityRequirement{{Scheme: "APIKey"}, {Scheme: "APIToken"}},
		[]string{
			"https://dac-static.atlassian.com/cloud/trello/swagger.v3.json",
			"https://developer.atlassian.com/cloud/trello/rest/",
			"https://developer.atlassian.com/cloud/trello/guides/rest-api/api-introduction/",
		},
		"Trello's official OpenAPI artifact is tracked as parseable-openapi-invalid by strict refresh validation, so key/token query security metadata is carried as review-only overlay guidance.",
	),
	parseableSpecAuthReviewOverlay(
		"zendesk-sunshine-auth-review",
		"zendesk",
		"zendesk-sunshine-conversations-openapi",
		[]SecurityScheme{
			{
				Name:         "bearerAuth",
				Type:         SecuritySchemeHTTP,
				Scheme:       "bearer",
				BearerFormat: "JWT",
				Description:  "Zendesk Sunshine Conversations bearer token metadata from the official OpenAPI artifact.",
			},
			{
				Name:        "basicAuth",
				Type:        SecuritySchemeHTTP,
				Scheme:      "basic",
				Description: "Zendesk Sunshine Conversations basic authentication metadata from the official OpenAPI artifact.",
			},
		},
		[]SecurityRequirement{{Scheme: "bearerAuth", Scopes: []string{"app", "account"}}, {Scheme: "basicAuth", Scopes: []string{"app", "account"}}},
		[]string{
			"https://raw.githubusercontent.com/zendesk/sunshine-conversations-api-spec/master/openapi.yaml",
			"https://developer.zendesk.com/documentation/conversations/references/openapi-specification/",
			"https://developer.zendesk.com/api-reference/introduction/security-and-auth/",
		},
		"Zendesk's official Sunshine Conversations OpenAPI artifact is tracked as parseable-openapi-invalid by strict refresh validation, so messaging API bearer/basic metadata is carried as review-only overlay guidance.",
	),
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
	{
		ID:         "activecampaign-api-v3-auth-overlay",
		ProviderID: "activecampaign",
		SpecRefID:  "activecampaign-auth-docs",
		Status:     AuthStatusOverlayRequired,
		SecuritySchemes: []SecurityScheme{{
			Name:          "activeCampaignAPIToken",
			Type:          SecuritySchemeAPIKey,
			In:            APIKeyInHeader,
			ParameterName: "Api-Token",
			Description:   "ActiveCampaign API token carried in the Api-Token header.",
		}},
		RootSecurity: []SecurityRequirement{{Scheme: "activeCampaignAPIToken"}},
		SourceRefs: []string{
			"https://developers.activecampaign.com/reference/authentication",
			"https://developers.activecampaign.com/reference/overview",
		},
		SourceNote: "ActiveCampaign has human API docs in this catalog entry but no recorded official OpenAPI document; official docs describe Api-Token header authentication, so OpenAPI imports need an advisory API-token overlay.",
	},
	{
		ID:         "bamboohr-api-auth-overlay",
		ProviderID: "bamboohr",
		SpecRefID:  "bamboohr-api-docs",
		Status:     AuthStatusOverlayRequired,
		SecuritySchemes: []SecurityScheme{{
			Name:        "bambooHRBasic",
			Type:        SecuritySchemeHTTP,
			Scheme:      "basic",
			Description: "BambooHR API key used as the HTTP Basic username with an arbitrary password.",
		}},
		RootSecurity: []SecurityRequirement{{Scheme: "bambooHRBasic"}},
		SourceRefs: []string{
			"https://documentation.bamboohr.com/docs",
			"https://documentation.bamboohr.com/docs/api-details",
		},
		SourceNote: "BambooHR has human API docs in this catalog entry but no recorded official OpenAPI document; docs describe API-key Basic authentication against company-subdomain API hosts.",
	},
	parseableSpecAuthReviewOverlay(
		"chargebee-api-v2-pc-v2-auth-review",
		"chargebee",
		"chargebee-api-v2-pc-v2-openapi",
		[]SecurityScheme{{
			Name:        "BasicAuth",
			Type:        SecuritySchemeHTTP,
			Scheme:      "basic",
			Description: "Chargebee API key carried with HTTP Basic authentication.",
		}},
		[]SecurityRequirement{{Scheme: "BasicAuth"}},
		[]string{
			"https://raw.githubusercontent.com/chargebee/openapi/main/spec/chargebee_api_v2_pc_v2_spec.json",
			"https://github.com/chargebee/openapi",
			"https://apidocs.chargebee.com/docs/api",
		},
		"Chargebee's official OpenAPI artifact declares BasicAuth but no root security requirement, so Basic auth metadata is carried as present-incomplete review guidance.",
	),
	{
		ID:         "contentful-api-auth-overlay",
		ProviderID: "contentful",
		SpecRefID:  "contentful-auth-docs",
		Status:     AuthStatusOverlayRequired,
		SecuritySchemes: []SecurityScheme{{
			Name:        "contentfulBearer",
			Type:        SecuritySchemeHTTP,
			Scheme:      "bearer",
			Description: "Contentful API access token carried in the Authorization header.",
		}},
		RootSecurity: []SecurityRequirement{{Scheme: "contentfulBearer"}},
		SourceRefs: []string{
			"https://www.contentful.com/developers/docs/references/authentication/",
			"https://www.contentful.com/developers/docs/references/content-management-api/",
		},
		SourceNote: "Contentful has human API docs in this catalog entry but no recorded official OpenAPI document; docs describe bearer-token authentication across Contentful API surfaces.",
	},
	{
		ID:         "customer-io-openapi-auth-review",
		ProviderID: "customer-io",
		SpecRefID:  "customer-io-journeys-app-openapi",
		Status:     AuthStatusPresentIncomplete,
		SecuritySchemes: []SecurityScheme{
			{
				Name:        "customerIoBearer",
				Type:        SecuritySchemeHTTP,
				Scheme:      "bearer",
				Description: "Customer.io App API key or service account token carried as a bearer token.",
			},
			{
				Name:        "customerIoBasic",
				Type:        SecuritySchemeHTTP,
				Scheme:      "basic",
				Description: "Customer.io Track or Pipelines credentials carried with HTTP Basic authentication.",
			},
		},
		RootSecurity: []SecurityRequirement{{Scheme: "customerIoBearer"}, {Scheme: "customerIoBasic"}},
		SourceRefs: []string{
			"https://docs.customer.io/files/journeys-app.json",
			"https://docs.customer.io/files/journeys-track.json",
			"https://docs.customer.io/files/pipelines.json",
			"https://docs.customer.io/integrations/api/customerio-apis/",
		},
		SourceNote: "Customer.io publishes separate official OpenAPI documents with different bearer and basic auth schemes, but root security is incomplete across the reviewed specs; keep this as present-incomplete auth review metadata.",
	},
	{
		ID:         "eventbrite-api-auth-overlay",
		ProviderID: "eventbrite",
		SpecRefID:  "eventbrite-api-auth-docs",
		Status:     AuthStatusOverlayRequired,
		SecuritySchemes: []SecurityScheme{{
			Name:        "eventbriteBearer",
			Type:        SecuritySchemeHTTP,
			Scheme:      "bearer",
			Description: "Eventbrite OAuth access token carried in the Authorization header.",
		}},
		RootSecurity: []SecurityRequirement{{Scheme: "eventbriteBearer"}},
		SourceRefs: []string{
			"https://www.eventbrite.com/platform/api",
			"https://www.eventbrite.com/platform/api#/introduction/authentication",
		},
		SourceNote: "Eventbrite has human API docs in this catalog entry but no recorded official OpenAPI document; docs describe OAuth bearer-token authentication.",
	},
	{
		ID:         "freshdesk-api-auth-overlay",
		ProviderID: "freshdesk",
		SpecRefID:  "freshdesk-api-docs",
		Status:     AuthStatusOverlayRequired,
		SecuritySchemes: []SecurityScheme{{
			Name:        "freshdeskBasic",
			Type:        SecuritySchemeHTTP,
			Scheme:      "basic",
			Description: "Freshdesk API key carried with HTTP Basic authentication.",
		}},
		RootSecurity: []SecurityRequirement{{Scheme: "freshdeskBasic"}},
		SourceRefs: []string{
			"https://developers.freshdesk.com/api",
			"https://support.freshdesk.com/en/support/solutions/articles/225441-is-there-any-documentation-for-the-apis-on-freshdesk-",
		},
		SourceNote: "Freshdesk has human API docs in this catalog entry but no recorded official OpenAPI document; docs describe API-key Basic authentication for account-domain API hosts.",
	},
	{
		ID:         "freshservice-api-auth-overlay",
		ProviderID: "freshservice",
		SpecRefID:  "freshservice-api-docs",
		Status:     AuthStatusOverlayRequired,
		SecuritySchemes: []SecurityScheme{{
			Name:        "freshserviceBasic",
			Type:        SecuritySchemeHTTP,
			Scheme:      "basic",
			Description: "Freshservice API key supplied as the HTTP Basic username with a dummy password.",
		}},
		RootSecurity: []SecurityRequirement{{Scheme: "freshserviceBasic"}},
		SourceRefs: []string{
			"https://api.freshservice.com/",
			"https://support.freshservice.com/support/solutions/articles/50000012704-working-with-apis-in-freshservice",
		},
		SourceNote: "Freshservice has human API v2 docs in this catalog entry but no recorded official OpenAPI document; docs describe API-key HTTP Basic authentication on account-domain API hosts.",
	},
	{
		ID:         "gong-api-auth-overlay",
		ProviderID: "gong",
		SpecRefID:  "gong-api-access-docs",
		Status:     AuthStatusOverlayRequired,
		SecuritySchemes: []SecurityScheme{
			{
				Name:        "gongBasic",
				Type:        SecuritySchemeHTTP,
				Scheme:      "basic",
				Description: "Gong access key and access key secret supplied with HTTP Basic authentication.",
			},
			{
				Name:        "gongBearer",
				Type:        SecuritySchemeHTTP,
				Scheme:      "bearer",
				Description: "Gong OAuth access token supplied as an Authorization bearer token for customer API base URLs.",
			},
		},
		RootSecurity: []SecurityRequirement{{Scheme: "gongBasic"}, {Scheme: "gongBearer"}},
		SourceRefs: []string{
			"https://help.gong.io/docs/receive-access-to-the-api",
			"https://help.gong.io/docs/create-an-app-for-gong",
			"https://help.gong.io/hc/en-us/articles/360046818511-Uploading-calls-from-a-non-integrated-telephony-system",
		},
		SourceNote: "Gong has human API docs in this catalog entry but no recorded official OpenAPI document; docs describe access key/secret Basic authentication and OAuth bearer-token authentication for customer API base URLs.",
	},
	parseableSpecAuthReviewOverlay(
		"mailgun-send-api-auth-review",
		"mailgun",
		"mailgun-send-api-openapi",
		[]SecurityScheme{{
			Name:        "basicAuth",
			Type:        SecuritySchemeHTTP,
			Scheme:      "basic",
			Description: "Mailgun HTTP Basic auth using api:YOUR_API_KEY.",
		}},
		[]SecurityRequirement{{Scheme: "basicAuth"}},
		[]string{
			"https://documentation.mailgun.com/_bundle/docs/mailgun/api-reference/send/mailgun.json",
			"https://documentation.mailgun.com/docs/mailgun/api-reference/send/mailgun/keys/api",
		},
		"Mailgun's official OpenAPI artifact declares HTTP Basic auth but no root security requirement, so Basic auth metadata is carried as present-incomplete review guidance.",
	),
	{
		ID:         "postmark-api-auth-overlay",
		ProviderID: "postmark",
		SpecRefID:  "postmark-api-docs",
		Status:     AuthStatusOverlayRequired,
		SecuritySchemes: []SecurityScheme{
			{
				Name:          "postmarkServerToken",
				Type:          SecuritySchemeAPIKey,
				In:            APIKeyInHeader,
				ParameterName: "X-Postmark-Server-Token",
				Description:   "Postmark server token carried in the X-Postmark-Server-Token header.",
			},
			{
				Name:          "postmarkAccountToken",
				Type:          SecuritySchemeAPIKey,
				In:            APIKeyInHeader,
				ParameterName: "X-Postmark-Account-Token",
				Description:   "Postmark account token carried in the X-Postmark-Account-Token header for account-level endpoints.",
			},
		},
		RootSecurity: []SecurityRequirement{{Scheme: "postmarkServerToken"}, {Scheme: "postmarkAccountToken"}},
		SourceRefs: []string{
			"https://postmarkapp.com/developer/api/overview",
			"https://postmarkapp.com/api-explorer",
		},
		SourceNote: "Postmark has human API docs in this catalog entry but no recorded official OpenAPI document; docs describe server and account API-token headers.",
	},
	{
		ID:         "todoist-rest-api-auth-overlay",
		ProviderID: "todoist",
		SpecRefID:  "todoist-auth-docs",
		Status:     AuthStatusOverlayRequired,
		SecuritySchemes: []SecurityScheme{{
			Name:        "todoistBearer",
			Type:        SecuritySchemeHTTP,
			Scheme:      "bearer",
			Description: "Todoist API token carried as an Authorization bearer token.",
		}},
		RootSecurity: []SecurityRequirement{{Scheme: "todoistBearer"}},
		SourceRefs: []string{
			"https://developer.todoist.com/rest/v2/",
			"https://developer.todoist.com/guides/#authentication",
		},
		SourceNote: "Todoist has human REST API docs in this catalog entry but no recorded official OpenAPI document; docs describe bearer-token authentication.",
	},
	{
		ID:         "webflow-data-api-auth-overlay",
		ProviderID: "webflow",
		SpecRefID:  "webflow-auth-docs",
		Status:     AuthStatusOverlayRequired,
		SecuritySchemes: []SecurityScheme{{
			Name:        "webflowBearer",
			Type:        SecuritySchemeHTTP,
			Scheme:      "bearer",
			Description: "Webflow API token carried as an Authorization bearer token.",
		}},
		RootSecurity: []SecurityRequirement{{Scheme: "webflowBearer"}},
		SourceRefs: []string{
			"https://developers.webflow.com/reference",
			"https://developers.webflow.com/data/reference/authorization",
		},
		SourceNote: "Webflow has human API docs in this catalog entry but no recorded stable public OpenAPI download; docs describe bearer-token authorization for Data API requests.",
	},
	{
		ID:         "acuity-scheduling-api-auth-overlay",
		ProviderID: "acuity-scheduling",
		SpecRefID:  "acuity-quick-start",
		Status:     AuthStatusOverlayRequired,
		SecuritySchemes: []SecurityScheme{{
			Name:        "acuityBasic",
			Type:        SecuritySchemeHTTP,
			Scheme:      "basic",
			Description: "Acuity Scheduling numeric user ID and API key carried with HTTP Basic authentication.",
		}},
		RootSecurity: []SecurityRequirement{{Scheme: "acuityBasic"}},
		SourceRefs: []string{
			"https://developers.acuityscheduling.com/docs/quick-start",
			"https://developers.acuityscheduling.com/reference",
		},
		SourceNote: "Acuity Scheduling has human API docs in this catalog entry but no recorded official OpenAPI document; docs describe Basic authentication with numeric user ID and API key.",
	},
	{
		ID:         "clockify-api-auth-overlay",
		ProviderID: "clockify",
		SpecRefID:  "clockify-api-docs",
		Status:     AuthStatusOverlayRequired,
		SecuritySchemes: []SecurityScheme{
			{
				Name:          "clockifyAPIKey",
				Type:          SecuritySchemeAPIKey,
				In:            APIKeyInHeader,
				ParameterName: "X-Api-Key",
				Description:   "Clockify API key carried in the X-Api-Key header.",
			},
			{
				Name:          "clockifyAddonToken",
				Type:          SecuritySchemeAPIKey,
				In:            APIKeyInHeader,
				ParameterName: "X-Addon-Token",
				Description:   "Clockify addon token carried in the X-Addon-Token header.",
			},
		},
		RootSecurity: []SecurityRequirement{{Scheme: "clockifyAPIKey"}, {Scheme: "clockifyAddonToken"}},
		SourceRefs: []string{
			"https://docs.clockify.me/",
			"https://clockify.me/help/getting-started/clockify-api-overview",
		},
		SourceNote: "Clockify has human API docs in this catalog entry but no recorded official OpenAPI document; docs describe X-Api-Key or X-Addon-Token header authentication.",
	},
	{
		ID:         "bannerbear-api-auth-overlay",
		ProviderID: "bannerbear",
		SpecRefID:  "bannerbear-api-v2-docs",
		Status:     AuthStatusOverlayRequired,
		SecuritySchemes: []SecurityScheme{{
			Name:        "bannerbearBearer",
			Type:        SecuritySchemeHTTP,
			Scheme:      "bearer",
			Description: "Bannerbear Project API Key or Master API Key carried as an Authorization bearer token.",
		}},
		RootSecurity: []SecurityRequirement{{Scheme: "bannerbearBearer"}},
		SourceRefs: []string{
			"https://developers.bannerbear.com/v2/",
			"https://www.bannerbear.com/help/api/",
		},
		SourceNote: "Bannerbear has human API v2 docs in this catalog entry but no recorded official OpenAPI document; docs describe Authorization: Bearer API_KEY authentication.",
	},
	{
		ID:         "ghost-api-auth-overlay",
		ProviderID: "ghost",
		SpecRefID:  "ghost-admin-api-docs",
		Status:     AuthStatusOverlayRequired,
		SecuritySchemes: []SecurityScheme{
			{
				Name:        "ghostAdminToken",
				Type:        SecuritySchemeHTTP,
				Scheme:      "bearer",
				Description: "Ghost Admin API token carried in the Authorization header after operator-side JWT generation.",
			},
			{
				Name:          "ghostContentAPIKey",
				Type:          SecuritySchemeAPIKey,
				In:            APIKeyInQuery,
				ParameterName: "key",
				Description:   "Ghost Content API key carried as the key query parameter.",
			},
		},
		RootSecurity: []SecurityRequirement{{Scheme: "ghostAdminToken"}, {Scheme: "ghostContentAPIKey"}},
		SourceRefs: []string{
			"https://docs.ghost.org/admin-api/",
			"https://docs.ghost.org/content-api/",
		},
		SourceNote: "Ghost has human Admin and Content API docs in this catalog entry but no recorded official OpenAPI document; Admin API JWT signing is an operator/runtime concern and is recorded here only as bearer-token metadata.",
	},
	{
		ID:         "harvest-api-v2-auth-overlay",
		ProviderID: "harvest",
		SpecRefID:  "harvest-auth-docs",
		Status:     AuthStatusOverlayRequired,
		SecuritySchemes: []SecurityScheme{{
			Name:        "harvestBearer",
			Type:        SecuritySchemeHTTP,
			Scheme:      "bearer",
			Description: "Harvest OAuth or personal access token carried as an Authorization bearer token.",
		}},
		RootSecurity: []SecurityRequirement{{Scheme: "harvestBearer"}},
		SourceRefs: []string{
			"https://help.getharvest.com/api-v2/",
			"https://help.getharvest.com/api-v2/authentication-api/authentication/authentication/",
		},
		SourceNote: "Harvest has human API v2 docs in this catalog entry but no recorded official OpenAPI document; docs describe bearer authorization and a Harvest-Account-ID header.",
	},
	{
		ID:         "grist-api-auth-overlay",
		ProviderID: "grist",
		SpecRefID:  "grist-rest-api-usage-docs",
		Status:     AuthStatusOverlayRequired,
		SecuritySchemes: []SecurityScheme{{
			Name:        "gristBearer",
			Type:        SecuritySchemeHTTP,
			Scheme:      "bearer",
			Description: "Grist API key carried as an Authorization bearer token.",
		}},
		RootSecurity: []SecurityRequirement{{Scheme: "gristBearer"}},
		SourceRefs: []string{
			"https://support.getgrist.com/rest-api/",
			"https://support.getgrist.com/api/",
		},
		SourceNote: "Grist has human REST API docs in this catalog entry but no recorded official OpenAPI document; docs describe API-key bearer authentication.",
	},
	{
		ID:         "help-scout-inbox-api-auth-overlay",
		ProviderID: "help-scout",
		SpecRefID:  "help-scout-mailbox-api-docs",
		Status:     AuthStatusOverlayRequired,
		SecuritySchemes: []SecurityScheme{
			{
				Name:        "helpScoutOAuth",
				Type:        SecuritySchemeHTTP,
				Scheme:      "bearer",
				Description: "Help Scout Inbox API OAuth 2 access token carried as an Authorization bearer token.",
			},
			{
				Name:        "helpScoutDocsBasic",
				Type:        SecuritySchemeHTTP,
				Scheme:      "basic",
				Description: "Help Scout Docs API key carried as the HTTP Basic username with a dummy password.",
			},
		},
		RootSecurity: []SecurityRequirement{{Scheme: "helpScoutOAuth"}, {Scheme: "helpScoutDocsBasic"}},
		SourceRefs: []string{
			"https://developer.helpscout.com/mailbox-api/",
			"https://developer.helpscout.com/docs-api/",
		},
		SourceNote: "Help Scout has human Inbox API and Docs API docs in this catalog entry but no recorded official OpenAPI document; Inbox API uses OAuth bearer tokens while Docs API uses Basic API-key authentication.",
	},
	{
		ID:         "iterable-api-auth-overlay",
		ProviderID: "iterable",
		SpecRefID:  "iterable-api-key-docs",
		Status:     AuthStatusOverlayRequired,
		SecuritySchemes: []SecurityScheme{{
			Name:          "iterableAPIKey",
			Type:          SecuritySchemeAPIKey,
			In:            APIKeyInHeader,
			ParameterName: "Api-Key",
			Description:   "Iterable API key carried in the Api-Key header.",
		}},
		RootSecurity: []SecurityRequirement{{Scheme: "iterableAPIKey"}},
		SourceRefs: []string{
			"https://api.iterable.com/api/docs",
			"https://support.iterable.com/hc/en-us/articles/360043464871-API-Keys",
		},
		SourceNote: "Iterable has human API docs in this catalog entry but no recorded official OpenAPI document; docs describe Api-Key header authentication.",
	},
	{
		ID:         "mailjet-api-auth-overlay",
		ProviderID: "mailjet",
		SpecRefID:  "mailjet-api-key-docs",
		Status:     AuthStatusOverlayRequired,
		SecuritySchemes: []SecurityScheme{{
			Name:        "mailjetBasic",
			Type:        SecuritySchemeHTTP,
			Scheme:      "basic",
			Description: "Mailjet API key and secret key carried as HTTP Basic username and password credentials.",
		}},
		RootSecurity: []SecurityRequirement{{Scheme: "mailjetBasic"}},
		SourceRefs: []string{
			"https://documentation.mailjet.com/hc/en-us/articles/360044088173-REST-API",
			"https://documentation.mailjet.com/hc/en-us/articles/360043225693-What-is-an-API-key",
		},
		SourceNote: "Mailjet has human REST API docs in this catalog entry but no recorded official OpenAPI document; docs describe API key and secret key credentials for API access.",
	},
	{
		ID:         "coingecko-api-auth-overlay",
		ProviderID: "coingecko",
		SpecRefID:  "coingecko-auth-docs",
		Status:     AuthStatusOverlayRequired,
		SecuritySchemes: []SecurityScheme{
			{
				Name:          "coingeckoDemoAPIKeyHeader",
				Type:          SecuritySchemeAPIKey,
				In:            APIKeyInHeader,
				ParameterName: "x-cg-demo-api-key",
				Description:   "CoinGecko Public/Demo API key carried in the x-cg-demo-api-key header.",
			},
			{
				Name:          "coingeckoDemoAPIKeyQuery",
				Type:          SecuritySchemeAPIKey,
				In:            APIKeyInQuery,
				ParameterName: "x_cg_demo_api_key",
				Description:   "CoinGecko Public/Demo API key carried in the x_cg_demo_api_key query parameter.",
			},
			{
				Name:          "coingeckoProAPIKeyHeader",
				Type:          SecuritySchemeAPIKey,
				In:            APIKeyInHeader,
				ParameterName: "x-cg-pro-api-key",
				Description:   "CoinGecko Pro API key carried in the x-cg-pro-api-key header.",
			},
			{
				Name:          "coingeckoProAPIKeyQuery",
				Type:          SecuritySchemeAPIKey,
				In:            APIKeyInQuery,
				ParameterName: "x_cg_pro_api_key",
				Description:   "CoinGecko Pro API key carried in the x_cg_pro_api_key query parameter.",
			},
		},
		RootSecurity: []SecurityRequirement{{Scheme: "coingeckoDemoAPIKeyHeader"}, {Scheme: "coingeckoDemoAPIKeyQuery"}, {Scheme: "coingeckoProAPIKeyHeader"}, {Scheme: "coingeckoProAPIKeyQuery"}},
		SourceRefs: []string{
			"https://docs.coingecko.com/reference/authentication",
			"https://docs.coingecko.com/v3.0.1/reference/authentication",
		},
		SourceNote: "CoinGecko has human API docs in this catalog entry but no recorded official OpenAPI document; docs describe separate Public/Demo and Pro API-key header and query-parameter names.",
	},
	{
		ID:         "nasa-open-apis-auth-overlay",
		ProviderID: "nasa",
		SpecRefID:  "nasa-api-docs",
		Status:     AuthStatusOverlayRequired,
		SecuritySchemes: []SecurityScheme{{
			Name:          "nasaAPIKey",
			Type:          SecuritySchemeAPIKey,
			In:            APIKeyInQuery,
			ParameterName: "api_key",
			Description:   "NASA Open APIs key carried in the api_key query parameter; official examples may use DEMO_KEY.",
		}},
		RootSecurity: []SecurityRequirement{{Scheme: "nasaAPIKey"}},
		SourceRefs: []string{
			"https://api.nasa.gov/",
		},
		SourceNote: "NASA Open APIs have human docs in this catalog entry but no recorded official OpenAPI document; docs describe api_key query authentication and DEMO_KEY examples.",
	},
	{
		ID:         "onesimpleapi-token-auth-overlay",
		ProviderID: "onesimpleapi",
		SpecRefID:  "onesimpleapi-docs",
		Status:     AuthStatusOverlayRequired,
		SecuritySchemes: []SecurityScheme{{
			Name:          "oneSimpleAPIToken",
			Type:          SecuritySchemeAPIKey,
			In:            APIKeyInQuery,
			ParameterName: "token",
			Description:   "OneSimpleApi access token carried in the token query parameter.",
		}},
		RootSecurity: []SecurityRequirement{{Scheme: "oneSimpleAPIToken"}},
		SourceRefs: []string{
			"https://onesimpleapi.com/docs",
			"https://onesimpleapi.com/user/api-tokens",
		},
		SourceNote: "OneSimpleApi has human docs in this catalog entry but no recorded official OpenAPI document; docs describe creating an access token and examples use the token query parameter.",
	},
	{
		ID:         "reddit-oauth-auth-overlay",
		ProviderID: "reddit",
		SpecRefID:  "reddit-api-docs",
		Status:     AuthStatusOverlayRequired,
		SecuritySchemes: []SecurityScheme{{
			Name:         "redditBearer",
			Type:         SecuritySchemeHTTP,
			Scheme:       "bearer",
			BearerFormat: "OAuth 2.0 access token",
			Description:  "Reddit OAuth access token carried in the Authorization bearer header for oauth.reddit.com API requests.",
		}},
		RootSecurity: []SecurityRequirement{{Scheme: "redditBearer"}},
		SourceRefs: []string{
			"https://www.reddit.com/dev/api/",
			"https://www.reddit.com/dev/api/oauth",
			"https://developers.reddit.com/docs/capabilities/server/reddit-api",
		},
		SourceNote: "Reddit has generated human API docs in this catalog entry but no recorded official OpenAPI document; OAuth API requests use bearer-token authorization metadata.",
	},
	{
		ID:         "autopilot-api-key-auth-overlay",
		ProviderID: "autopilot",
		SpecRefID:  "autopilot-api-docs",
		Status:     AuthStatusOverlayRequired,
		SecuritySchemes: []SecurityScheme{{
			Name:          "autopilotAPIKey",
			Type:          SecuritySchemeAPIKey,
			In:            APIKeyInHeader,
			ParameterName: "autopilotapikey",
			Description:   "Autopilot API key carried in the autopilotapikey header.",
		}},
		RootSecurity: []SecurityRequirement{{Scheme: "autopilotAPIKey"}},
		SourceRefs: []string{
			"https://autopilot.docs.apiary.io/",
			"https://help.ortto.com/a-376-autopilot-how-to-use-autopilots-api",
		},
		SourceNote: "Autopilot has human API docs in this catalog entry but no recorded official OpenAPI document; the legacy REST API uses an autopilotapikey header.",
	},
	{
		ID:         "drift-oauth-auth-overlay",
		ProviderID: "drift",
		SpecRefID:  "drift-auth-docs",
		Status:     AuthStatusOverlayRequired,
		SecuritySchemes: []SecurityScheme{
			{
				Name:         "driftBearer",
				Type:         SecuritySchemeHTTP,
				Scheme:       "bearer",
				BearerFormat: "OAuth 2.0 access token",
				Description:  "Drift API access token carried in the Authorization bearer header.",
			},
			{
				Name:        "driftOAuth2",
				Type:        SecuritySchemeOAuth2,
				Description: "Drift OAuth 2.0 authorization-code flow.",
				Flows: []OAuthFlow{{
					Type:             OAuthFlowAuthorizationCode,
					AuthorizationURL: "https://dev.drift.com/authorize",
					TokenURL:         "https://driftapi.com/oauth2/token",
				}},
			},
		},
		RootSecurity: []SecurityRequirement{{Scheme: "driftBearer"}, {Scheme: "driftOAuth2"}},
		SourceRefs: []string{
			"https://devdocs.drift.com/docs/authentication-and-scopes",
			"https://devdocs.drift.com/docs/platform-apis",
		},
		SourceNote: "Drift has human backend API docs in this catalog entry but no recorded official OpenAPI document; docs describe OAuth 2.0 access for platform APIs.",
	},
	{
		ID:         "freshworks-crm-token-auth-overlay",
		ProviderID: "freshworks-crm",
		SpecRefID:  "freshworks-crm-api-docs",
		Status:     AuthStatusOverlayRequired,
		SecuritySchemes: []SecurityScheme{{
			Name:          "freshworksCRMToken",
			Type:          SecuritySchemeAPIKey,
			In:            APIKeyInHeader,
			ParameterName: "Authorization",
			Description:   "Freshworks CRM API key carried in the Authorization header using Token token={api_key} syntax.",
		}},
		RootSecurity: []SecurityRequirement{{Scheme: "freshworksCRMToken"}},
		SourceRefs: []string{
			"https://developers.freshworks.com/crm/api/",
		},
		SourceNote: "Freshworks CRM has human API docs in this catalog entry but no recorded official OpenAPI document; docs describe API-key authentication through the Authorization header.",
	},
	{
		ID:         "getresponse-auth-overlay",
		ProviderID: "getresponse",
		SpecRefID:  "getresponse-auth-docs",
		Status:     AuthStatusOverlayRequired,
		SecuritySchemes: []SecurityScheme{
			{
				Name:          "getResponseAPIKey",
				Type:          SecuritySchemeAPIKey,
				In:            APIKeyInHeader,
				ParameterName: "X-Auth-Token",
				Description:   "GetResponse API key carried in the X-Auth-Token header as api-key {key}.",
			},
			{
				Name:         "getResponseOAuth2",
				Type:         SecuritySchemeHTTP,
				Scheme:       "bearer",
				BearerFormat: "OAuth 2.0 access token",
				Description:  "GetResponse OAuth 2.0 access token carried in the Authorization bearer header.",
			},
		},
		RootSecurity: []SecurityRequirement{{Scheme: "getResponseAPIKey"}, {Scheme: "getResponseOAuth2"}},
		SourceRefs: []string{
			"https://apidocs.getresponse.com/v3/authentication",
			"https://apidocs.getresponse.com/v3",
		},
		SourceNote: "GetResponse has human API docs in this catalog entry but no recorded official OpenAPI document; docs describe API-key and OAuth 2.0 authentication methods.",
	},
	{
		ID:         "keap-oauth-auth-overlay",
		ProviderID: "keap",
		SpecRefID:  "keap-rest-api-docs",
		Status:     AuthStatusOverlayRequired,
		SecuritySchemes: []SecurityScheme{{
			Name:         "keapOAuth2",
			Type:         SecuritySchemeHTTP,
			Scheme:       "bearer",
			BearerFormat: "OAuth 2.0 access token",
			Description:  "Keap REST API OAuth 2.0 access token carried in the Authorization bearer header.",
		}},
		RootSecurity: []SecurityRequirement{{Scheme: "keapOAuth2"}},
		SourceRefs: []string{
			"https://developer.keap.com/docs/rest/1000/",
		},
		SourceNote: "Keap has human REST API docs in this catalog entry but no recorded official OpenAPI document; docs describe OAuth 2.0 authentication for REST endpoints.",
	},
	{
		ID:         "mailerlite-auth-overlay",
		ProviderID: "mailerlite",
		SpecRefID:  "mailerlite-api-docs",
		Status:     AuthStatusOverlayRequired,
		SecuritySchemes: []SecurityScheme{
			{
				Name:         "mailerLiteBearer",
				Type:         SecuritySchemeHTTP,
				Scheme:       "bearer",
				BearerFormat: "API token",
				Description:  "Current MailerLite API token carried in the Authorization bearer header.",
			},
			{
				Name:          "mailerLiteClassicAPIKey",
				Type:          SecuritySchemeAPIKey,
				In:            APIKeyInHeader,
				ParameterName: "X-MailerLite-ApiKey",
				Description:   "MailerLite Classic API key carried in the X-MailerLite-ApiKey header.",
			},
		},
		RootSecurity: []SecurityRequirement{{Scheme: "mailerLiteBearer"}, {Scheme: "mailerLiteClassicAPIKey"}},
		SourceRefs: []string{
			"https://developers.mailerlite.com/docs/",
			"https://developers-classic.mailerlite.com/docs/authentication",
		},
		SourceNote: "MailerLite has human API docs in this catalog entry but no recorded official OpenAPI document; current API docs describe bearer tokens and Classic docs describe X-MailerLite-ApiKey.",
	},
	{
		ID:         "mautic-auth-overlay",
		ProviderID: "mautic",
		SpecRefID:  "mautic-api-docs",
		Status:     AuthStatusOverlayRequired,
		SecuritySchemes: []SecurityScheme{
			{
				Name:        "mauticBasic",
				Type:        SecuritySchemeHTTP,
				Scheme:      "basic",
				Description: "Mautic basic authentication for configured instances.",
			},
			{
				Name:         "mauticOAuth2",
				Type:         SecuritySchemeHTTP,
				Scheme:       "bearer",
				BearerFormat: "OAuth access token",
				Description:  "Mautic OAuth access token carried in the Authorization bearer header.",
			},
		},
		RootSecurity: []SecurityRequirement{{Scheme: "mauticBasic"}, {Scheme: "mauticOAuth2"}},
		SourceRefs: []string{
			"https://developer.mautic.org/",
			"https://kb.mautic.org/article/what-is-mautic-039%3Bs-api.html",
		},
		SourceNote: "Mautic has human API docs in this catalog entry but no recorded official OpenAPI document; docs describe OAuth or Basic authentication depending on instance configuration.",
	},
	{
		ID:         "monica-crm-bearer-auth-overlay",
		ProviderID: "monica-crm",
		SpecRefID:  "monica-crm-api-docs",
		Status:     AuthStatusOverlayRequired,
		SecuritySchemes: []SecurityScheme{{
			Name:         "monicaBearer",
			Type:         SecuritySchemeHTTP,
			Scheme:       "bearer",
			BearerFormat: "OAuth 2.0 access token",
			Description:  "Monica API OAuth 2.0 access token carried in the Authorization bearer header.",
		}},
		RootSecurity: []SecurityRequirement{{Scheme: "monicaBearer"}},
		SourceRefs: []string{
			"https://www.monicahq.com/api",
		},
		SourceNote: "Monica has human API docs in this catalog entry but no recorded official OpenAPI document; docs describe OAuth 2 token authentication sent in a header.",
	},
	{
		ID:         "salesmate-session-auth-overlay",
		ProviderID: "salesmate",
		SpecRefID:  "salesmate-api-auth-example",
		Status:     AuthStatusOverlayRequired,
		SecuritySchemes: []SecurityScheme{
			{
				Name:          "salesmateSessionToken",
				Type:          SecuritySchemeAPIKey,
				In:            APIKeyInHeader,
				ParameterName: "sessionToken",
				Description:   "Salesmate session token carried in the sessionToken header.",
			},
			{
				Name:          "salesmateLinkName",
				Type:          SecuritySchemeAPIKey,
				In:            APIKeyInHeader,
				ParameterName: "x-linkname",
				Description:   "Salesmate account link name carried in the x-linkname header.",
			},
		},
		RootSecurity: []SecurityRequirement{{Scheme: "salesmateSessionToken"}, {Scheme: "salesmateLinkName"}},
		RootSecuritySets: []SecurityRequirementSet{{
			Requirements: []SecurityRequirement{{Scheme: "salesmateSessionToken"}, {Scheme: "salesmateLinkName"}},
		}},
		SourceRefs: []string{
			"https://support.salesmate.io/hc/en-us/articles/12864176787609-What-Are-API-s-Usage-and-Working",
			"https://support.salesmate.io/hc/en-us/articles/360043653751-Auto-enroll-Contacts-to-sequence-whenever-a-Deal-stage-is-updated-via-Webhooks",
		},
		SourceNote: "Salesmate has human API docs in this catalog entry but no recorded official OpenAPI document; docs show sessionToken and x-linkname request headers for API calls.",
	},
	{
		ID:         "sendy-api-key-auth-overlay",
		ProviderID: "sendy",
		SpecRefID:  "sendy-api-docs",
		Status:     AuthStatusOverlayRequired,
		SecuritySchemes: []SecurityScheme{{
			Name:          "sendyAPIKey",
			Type:          SecuritySchemeAPIKey,
			In:            APIKeyInQuery,
			ParameterName: "api_key",
			Description:   "Sendy API key carried as the api_key form/query parameter on documented POST operations.",
		}},
		RootSecurity: []SecurityRequirement{{Scheme: "sendyAPIKey"}},
		SourceRefs: []string{
			"https://sendy.co/api",
		},
		SourceNote: "Sendy has human API docs in this catalog entry but no recorded official OpenAPI document; docs describe an api_key parameter for API operations.",
	},
	{
		ID:         "vero-auth-token-overlay",
		ProviderID: "vero",
		SpecRefID:  "vero-track-api-docs",
		Status:     AuthStatusOverlayRequired,
		SecuritySchemes: []SecurityScheme{{
			Name:          "veroAuthToken",
			Type:          SecuritySchemeAPIKey,
			In:            APIKeyInQuery,
			ParameterName: "auth_token",
			Description:   "Vero authentication token carried as the auth_token request parameter.",
		}},
		RootSecurity: []SecurityRequirement{{Scheme: "veroAuthToken"}},
		SourceRefs: []string{
			"https://help.getvero.com/api-reference/track/overview",
			"https://help.getvero.com/developer-docs/overview",
		},
		SourceNote: "Vero has human API docs in this catalog entry but no recorded official OpenAPI document; Track API docs describe auth_token request-parameter authentication.",
	},
	{
		ID:         "jotform-api-key-auth-overlay",
		ProviderID: "jotform",
		SpecRefID:  "jotform-api-docs",
		Status:     AuthStatusOverlayRequired,
		SecuritySchemes: []SecurityScheme{{
			Name:          "jotformAPIKey",
			Type:          SecuritySchemeAPIKey,
			In:            APIKeyInHeader,
			ParameterName: "APIKEY",
			Description:   "JotForm API key carried in the APIKEY header.",
		}},
		RootSecurity: []SecurityRequirement{{Scheme: "jotformAPIKey"}},
		SourceRefs: []string{
			"https://api.jotform.com/docs/",
			"https://api.jotform.com/",
		},
		SourceNote: "JotForm has human API docs in this catalog entry but no recorded official OpenAPI document; API examples and integrations use an APIKEY credential.",
	},
	{
		ID:         "formio-token-auth-overlay",
		ProviderID: "formio",
		SpecRefID:  "formio-auth-docs",
		Status:     AuthStatusOverlayRequired,
		SecuritySchemes: []SecurityScheme{{
			Name:          "formioJWTToken",
			Type:          SecuritySchemeAPIKey,
			In:            APIKeyInHeader,
			ParameterName: "x-jwt-token",
			Description:   "Form.io JWT token carried in the x-jwt-token header.",
		}},
		RootSecurity: []SecurityRequirement{{Scheme: "formioJWTToken"}},
		SourceRefs: []string{
			"https://help.form.io/developers/authentication",
			"https://help.form.io/developers/introduction/api-documentation",
		},
		SourceNote: "Form.io has human API docs in this catalog entry but no recorded official OpenAPI document; docs describe token-based authenticated API access.",
	},
	{
		ID:         "formstack-oauth-auth-overlay",
		ProviderID: "formstack",
		SpecRefID:  "formstack-auth-docs",
		Status:     AuthStatusOverlayRequired,
		SecuritySchemes: []SecurityScheme{
			{
				Name:         "formstackBearer",
				Type:         SecuritySchemeHTTP,
				Scheme:       "bearer",
				BearerFormat: "OAuth 2.0 access token",
				Description:  "Formstack OAuth access token carried in the Authorization bearer header.",
			},
			{
				Name:        "formstackOAuth2",
				Type:        SecuritySchemeOAuth2,
				Description: "Formstack OAuth 2.0 authorization-code flow.",
				Flows: []OAuthFlow{{
					Type:             OAuthFlowAuthorizationCode,
					AuthorizationURL: "https://www.formstack.com/api/v2/oauth2/authorize",
					TokenURL:         "https://www.formstack.com/api/v2/oauth2/token",
				}},
			},
		},
		RootSecurity: []SecurityRequirement{{Scheme: "formstackBearer"}, {Scheme: "formstackOAuth2"}},
		SourceRefs: []string{
			"https://developers.formstack.com/reference/authorization",
			"https://developers.formstack.com/v2.0/reference/api-overview",
		},
		SourceNote: "Formstack has human API docs in this catalog entry but no recorded official OpenAPI document; docs describe OAuth bearer access for API calls.",
	},
	{
		ID:         "surveymonkey-oauth-auth-overlay",
		ProviderID: "surveymonkey",
		SpecRefID:  "surveymonkey-api-docs",
		Status:     AuthStatusOverlayRequired,
		SecuritySchemes: []SecurityScheme{{
			Name:         "surveyMonkeyBearer",
			Type:         SecuritySchemeHTTP,
			Scheme:       "bearer",
			BearerFormat: "OAuth 2.0 access token",
			Description:  "SurveyMonkey OAuth access token carried in the Authorization bearer header.",
		}},
		RootSecurity: []SecurityRequirement{{Scheme: "surveyMonkeyBearer"}},
		SourceRefs: []string{
			"https://api.surveymonkey.com/v3/docs",
			"https://help.surveymonkey.com/en/surveymonkey/integrations/surveymonkey-api/",
		},
		SourceNote: "SurveyMonkey has human API docs in this catalog entry but no recorded official OpenAPI document; API v3 uses OAuth 2.0 bearer access.",
	},
	{
		ID:         "bubble-bearer-auth-overlay",
		ProviderID: "bubble",
		SpecRefID:  "bubble-api-docs",
		Status:     AuthStatusOverlayRequired,
		SecuritySchemes: []SecurityScheme{{
			Name:         "bubbleBearer",
			Type:         SecuritySchemeHTTP,
			Scheme:       "bearer",
			BearerFormat: "Bubble API token",
			Description:  "Bubble API token carried in the Authorization bearer header.",
		}},
		RootSecurity: []SecurityRequirement{{Scheme: "bubbleBearer"}},
		SourceRefs: []string{
			"https://manual.bubble.io/help-guides/integrations/api/the-bubble-api",
			"https://manual.bubble.io/core-resources/api/data-api",
		},
		SourceNote: "Bubble has human API docs and app-specific Swagger metadata but no stable provider-wide public official OpenAPI document; API access uses app API tokens.",
	},
	{
		ID:         "cal-api-v2-auth-review",
		ProviderID: "cal",
		SpecRefID:  "cal-api-v2-openapi",
		Status:     AuthStatusPresentIncomplete,
		SecuritySchemes: []SecurityScheme{{
			Name:         "calBearer",
			Type:         SecuritySchemeHTTP,
			Scheme:       "bearer",
			BearerFormat: "API key or OAuth access token",
			Description:  "Cal.com API v2 access token carried in the Authorization bearer header.",
		}},
		RootSecurity: []SecurityRequirement{{Scheme: "calBearer"}},
		SourceRefs: []string{
			"https://cal.com/docs",
			"https://cal.com/docs/api-reference/v2/openapi.json",
		},
		SourceNote: "Cal.com's official API v2 OpenAPI document is tracked as an official artifact but models Authorization as header parameters rather than reusable security schemes.",
	},
	{
		ID:         "cockpit-token-auth-overlay",
		ProviderID: "cockpit",
		SpecRefID:  "cockpit-api-docs",
		Status:     AuthStatusOverlayRequired,
		SecuritySchemes: []SecurityScheme{{
			Name:          "cockpitToken",
			Type:          SecuritySchemeAPIKey,
			In:            APIKeyInQuery,
			ParameterName: "token",
			Description:   "Cockpit API token carried as the token request parameter.",
		}},
		RootSecurity: []SecurityRequirement{{Scheme: "cockpitToken"}},
		SourceRefs: []string{
			"https://getcockpit.com/documentation/core/api",
			"https://getcockpit.com/documentation/core",
		},
		SourceNote: "Cockpit has human API docs in this catalog entry but no recorded official OpenAPI document; API access is token-based and instance-hosted.",
	},
	{
		ID:         "stackby-api-key-auth-overlay",
		ProviderID: "stackby",
		SpecRefID:  "stackby-api-key-docs",
		Status:     AuthStatusOverlayRequired,
		SecuritySchemes: []SecurityScheme{{
			Name:          "stackbyAPIKey",
			Type:          SecuritySchemeAPIKey,
			In:            APIKeyInHeader,
			ParameterName: "api-key",
			Description:   "Stackby API key carried in the api-key header.",
		}},
		RootSecurity: []SecurityRequirement{{Scheme: "stackbyAPIKey"}},
		SourceRefs: []string{
			"https://help.stackby.com/en/articles/29-developer-api",
			"https://help.stackby.com/en/articles/124-how-to-get-your-api-key-in-stackby",
		},
		SourceNote: "Stackby has human API docs in this catalog entry but no recorded official OpenAPI document; docs describe account API-key use for integrations.",
	},
	{
		ID:         "airtop-api-auth-review",
		ProviderID: "airtop",
		SpecRefID:  "airtop-api-openapi",
		Status:     AuthStatusPresentIncomplete,
		SecuritySchemes: []SecurityScheme{{
			Name:         "airtopBearer",
			Type:         SecuritySchemeHTTP,
			Scheme:       "bearer",
			BearerFormat: "API key",
			Description:  "Airtop API key carried in the Authorization bearer header.",
		}},
		RootSecurity: []SecurityRequirement{{Scheme: "airtopBearer"}},
		SourceRefs: []string{
			"https://docs.airtop.ai/openapi.json",
			"https://docs.airtop.ai/api-reference/airtop-api",
		},
		SourceNote: "Airtop's official OpenAPI document declares bearer auth metadata and also repeats Authorization as operation header parameters, so bearer metadata is carried as a present-incomplete review overlay.",
	},
	{
		ID:         "filemaker-data-api-auth-overlay",
		ProviderID: "filemaker",
		SpecRefID:  "filemaker-data-api-docs",
		Status:     AuthStatusOverlayRequired,
		SecuritySchemes: []SecurityScheme{
			{
				Name:        "fileMakerBasic",
				Type:        SecuritySchemeHTTP,
				Scheme:      "basic",
				Description: "FileMaker Data API session creation uses HTTP Basic credentials against a hosted database.",
			},
			{
				Name:         "fileMakerBearer",
				Type:         SecuritySchemeHTTP,
				Scheme:       "bearer",
				BearerFormat: "FileMaker Data API session token",
				Description:  "FileMaker Data API session token carried in the Authorization bearer header after session creation.",
			},
		},
		RootSecurity: []SecurityRequirement{{Scheme: "fileMakerBasic"}, {Scheme: "fileMakerBearer"}},
		SourceRefs: []string{
			"https://help.claris.com/en/data-api-guide/content/data-api-reference.html",
			"https://help.claris.com/en/data-api-guide/",
		},
		SourceNote: "FileMaker has human Data API docs in this catalog entry but no recorded stable public official OpenAPI document; auth is a hosted-database Basic login followed by bearer session tokens.",
	},
	{
		ID:         "facebook-graph-api-auth-overlay",
		ProviderID: "facebook",
		SpecRefID:  "facebook-graph-api-docs",
		Status:     AuthStatusOverlayRequired,
		SecuritySchemes: []SecurityScheme{{
			Name:         "facebookBearer",
			Type:         SecuritySchemeHTTP,
			Scheme:       "bearer",
			BearerFormat: "Meta Graph API access token",
			Description:  "Meta Graph API access token carried in the Authorization bearer header.",
		}},
		RootSecurity: []SecurityRequirement{{Scheme: "facebookBearer"}},
		SourceRefs: []string{
			"https://developers.facebook.com/docs/graph-api/",
			"https://developers.facebook.com/docs/pages-api/",
		},
		SourceNote: "Facebook has official Graph API human docs but no recorded stable public official OpenAPI document; bearer access-token metadata comes from advisory overlay notes.",
	},
	{
		ID:         "facebook-lead-ads-auth-overlay",
		ProviderID: "facebook-lead-ads",
		SpecRefID:  "facebook-lead-ads-retrieval-docs",
		Status:     AuthStatusOverlayRequired,
		SecuritySchemes: []SecurityScheme{{
			Name:         "facebookLeadAdsBearer",
			Type:         SecuritySchemeHTTP,
			Scheme:       "bearer",
			BearerFormat: "Meta Graph API access token",
			Description:  "Meta access token with Lead Ads permissions carried in the Authorization bearer header.",
		}},
		RootSecurity: []SecurityRequirement{{Scheme: "facebookLeadAdsBearer"}},
		SourceRefs: []string{
			"https://developers.facebook.com/docs/marketing-api/guides/lead-ads/retrieving/",
			"https://developers.facebook.com/docs/marketing-api/guides/lead-ads/webhooks/",
		},
		SourceNote: "Facebook Lead Ads has official human docs but no recorded stable public official OpenAPI document; access-token and permission requirements come from advisory overlay notes.",
	},
	{
		ID:         "linkedin-api-auth-overlay",
		ProviderID: "linkedin",
		SpecRefID:  "linkedin-marketing-api-docs",
		Status:     AuthStatusOverlayRequired,
		SecuritySchemes: []SecurityScheme{{
			Name:         "linkedinOAuth2",
			Type:         SecuritySchemeHTTP,
			Scheme:       "bearer",
			BearerFormat: "OAuth 2.0 access token",
			Description:  "LinkedIn OAuth 2.0 access token carried in the Authorization bearer header.",
		}},
		RootSecurity: []SecurityRequirement{{Scheme: "linkedinOAuth2"}},
		SourceRefs: []string{
			"https://learn.microsoft.com/en-us/linkedin/marketing/",
			"https://learn.microsoft.com/en-us/linkedin/shared/authentication/authorization-code-flow",
		},
		SourceNote: "LinkedIn has official human API docs but no recorded stable public official OpenAPI document; OAuth bearer metadata comes from advisory overlay notes.",
	},
	{
		ID:         "messagebird-api-auth-overlay",
		ProviderID: "messagebird",
		SpecRefID:  "messagebird-api-docs",
		Status:     AuthStatusOverlayRequired,
		SecuritySchemes: []SecurityScheme{{
			Name:          "messageBirdAccessKey",
			Type:          SecuritySchemeAPIKey,
			In:            APIKeyInHeader,
			ParameterName: "Authorization",
			Description:   "MessageBird API access key carried in the Authorization header with the AccessKey authentication scheme.",
		}},
		RootSecurity: []SecurityRequirement{{Scheme: "messageBirdAccessKey"}},
		SourceRefs: []string{
			"https://developers.messagebird.com/api/",
		},
		SourceNote: "MessageBird has official human API docs but no recorded stable public official OpenAPI document; docs describe Authorization: AccessKey authentication.",
	},
	{
		ID:         "rocket-chat-openapi-auth-review",
		ProviderID: "rocket-chat",
		SpecRefID:  "rocket-chat-messaging-openapi",
		Status:     AuthStatusPresentIncomplete,
		SecuritySchemes: []SecurityScheme{
			{
				Name:          "rocketChatAuthToken",
				Type:          SecuritySchemeAPIKey,
				In:            APIKeyInHeader,
				ParameterName: "X-Auth-Token",
				Description:   "Rocket.Chat user authentication token or personal access token carried in the X-Auth-Token header.",
			},
			{
				Name:          "rocketChatUserID",
				Type:          SecuritySchemeAPIKey,
				In:            APIKeyInHeader,
				ParameterName: "X-User-Id",
				Description:   "Rocket.Chat authenticated user ID carried in the X-User-Id header together with X-Auth-Token.",
			},
		},
		RootSecuritySets: []SecurityRequirementSet{{
			Requirements: []SecurityRequirement{{Scheme: "rocketChatAuthToken"}, {Scheme: "rocketChatUserID"}},
		}},
		SourceRefs: []string{
			"https://raw.githubusercontent.com/RocketChat/Rocket.Chat-Open-API/main/authentication.yaml",
			"https://raw.githubusercontent.com/RocketChat/Rocket.Chat-Open-API/main/messaging.yaml",
			"https://developer.rocket.chat/apidocs",
		},
		SourceNote: "Rocket.Chat publishes official OpenAPI modules that model x-Auth-Token and x-User-Id as header parameters instead of reusable security schemes; this review overlay preserves their combined requirement as metadata.",
	},
	{
		ID:         "twake-developers-api-auth-overlay",
		ProviderID: "twake",
		SpecRefID:  "twake-developers-api-docs",
		Status:     AuthStatusOverlayRequired,
		SecuritySchemes: []SecurityScheme{{
			Name:        "twakeBasic",
			Type:        SecuritySchemeHTTP,
			Scheme:      "basic",
			Description: "Twake application public_id and private_api_key carried using HTTP Basic authentication.",
		}},
		RootSecurity: []SecurityRequirement{{Scheme: "twakeBasic"}},
		SourceRefs: []string{
			"https://doc.twake.app/developers-api/api-reference/auth",
			"https://doc.twake.app/developers-api/api-reference/message/post-request",
		},
		SourceNote: "Twake has official human Developers API docs but no recorded stable public official OpenAPI document; Basic application authentication metadata comes from advisory overlay notes.",
	},
	{
		ID:         "twist-api-v3-auth-overlay",
		ProviderID: "twist",
		SpecRefID:  "twist-api-v3-docs",
		Status:     AuthStatusOverlayRequired,
		SecuritySchemes: []SecurityScheme{{
			Name:         "twistBearer",
			Type:         SecuritySchemeHTTP,
			Scheme:       "bearer",
			BearerFormat: "Twist OAuth 2.0 or test token",
			Description:  "Twist API token carried in the Authorization bearer header.",
		}},
		RootSecurity: []SecurityRequirement{{Scheme: "twistBearer"}},
		SourceRefs: []string{
			"https://developer.twist.com/v3/",
		},
		SourceNote: "Twist has official API v3 human docs but no recorded stable public official OpenAPI document; OAuth/test bearer token metadata comes from advisory overlay notes.",
	},
	{
		ID:         "whatsapp-cloud-api-auth-overlay",
		ProviderID: "whatsapp",
		SpecRefID:  "whatsapp-cloud-api-docs",
		Status:     AuthStatusOverlayRequired,
		SecuritySchemes: []SecurityScheme{{
			Name:         "whatsAppBearer",
			Type:         SecuritySchemeHTTP,
			Scheme:       "bearer",
			BearerFormat: "Meta Graph API access token",
			Description:  "WhatsApp Cloud API access token carried in the Authorization bearer header.",
		}},
		RootSecurity: []SecurityRequirement{{Scheme: "whatsAppBearer"}},
		SourceRefs: []string{
			"https://developers.facebook.com/docs/whatsapp/cloud-api/",
			"https://developers.facebook.com/docs/whatsapp/cloud-api/reference/messages",
		},
		SourceNote: "WhatsApp Cloud API has official human docs but no recorded stable public official OpenAPI document; bearer access-token metadata comes from advisory overlay notes.",
	},
	{
		ID:         "magento-rest-api-auth-overlay",
		ProviderID: "magento",
		SpecRefID:  "magento-rest-api-docs",
		Status:     AuthStatusOverlayRequired,
		SecuritySchemes: []SecurityScheme{{
			Name:         "magentoBearer",
			Type:         SecuritySchemeHTTP,
			Scheme:       "bearer",
			BearerFormat: "Adobe Commerce admin, customer, or integration token",
			Description:  "Adobe Commerce REST API token carried in the Authorization bearer header.",
		}},
		RootSecurity: []SecurityRequirement{{Scheme: "magentoBearer"}},
		SourceRefs: []string{
			"https://developer.adobe.com/commerce/webapi/rest/reference/",
			"https://developer.adobe.com/commerce/webapi/get-started/authentication/",
		},
		SourceNote: "Adobe Commerce has official human REST docs and instance-local Swagger generation but no recorded stable public official OpenAPI document; bearer token metadata comes from advisory overlay notes.",
	},
	{
		ID:         "gumroad-api-auth-overlay",
		ProviderID: "gumroad",
		SpecRefID:  "gumroad-api-docs",
		Status:     AuthStatusOverlayRequired,
		SecuritySchemes: []SecurityScheme{{
			Name:         "gumroadBearer",
			Type:         SecuritySchemeHTTP,
			Scheme:       "bearer",
			BearerFormat: "Gumroad access token",
			Description:  "Gumroad OAuth access token carried in the Authorization bearer header.",
		}},
		RootSecurity: []SecurityRequirement{{Scheme: "gumroadBearer"}},
		SourceRefs: []string{
			"https://gumroad.com/api",
			"https://gumroad.com/oauth/applications",
		},
		SourceNote: "Gumroad has official human API docs but no recorded stable public official OpenAPI document; OAuth bearer metadata comes from advisory overlay notes.",
	},
	{
		ID:         "invoice-ninja-api-auth-overlay",
		ProviderID: "invoice-ninja",
		SpecRefID:  "invoice-ninja-api-docs",
		Status:     AuthStatusOverlayRequired,
		SecuritySchemes: []SecurityScheme{{
			Name:          "invoiceNinjaAPIToken",
			Type:          SecuritySchemeAPIKey,
			In:            APIKeyInHeader,
			ParameterName: "X-API-TOKEN",
			Description:   "Invoice Ninja v5 API token carried in the X-API-TOKEN header.",
		}},
		RootSecurity: []SecurityRequirement{{Scheme: "invoiceNinjaAPIToken"}},
		SourceRefs: []string{
			"https://api-docs.invoicing.co/",
			"https://invoiceninja.github.io/docs/developer-guide",
		},
		SourceNote: "Invoice Ninja has official OpenAPI-rendered human API docs but no recorded stable standalone downloadable OpenAPI document; X-API-TOKEN metadata comes from advisory overlay notes.",
	},
	{
		ID:         "odoo-external-api-auth-overlay",
		ProviderID: "odoo",
		SpecRefID:  "odoo-external-api-docs",
		Status:     AuthStatusOverlayRequired,
		SecuritySchemes: []SecurityScheme{{
			Name:        "odooRPCLogin",
			Type:        SecuritySchemeHTTP,
			Scheme:      "basic",
			Description: "Odoo external API authentication uses database, username, and password or API key values in RPC login/authenticate calls; this scheme is metadata-only and not an HTTP Basic assertion.",
		}},
		RootSecurity: []SecurityRequirement{{Scheme: "odooRPCLogin"}},
		SourceRefs: []string{
			"https://www.odoo.com/documentation/18.0/developer/reference/external_api.html",
		},
		SourceNote: "Odoo has official external API human docs but no recorded stable public official OpenAPI document; API-key/password RPC authentication metadata comes from advisory overlay notes.",
	},
	{
		ID:         "erpnext-rest-api-auth-overlay",
		ProviderID: "erpnext",
		SpecRefID:  "erpnext-rest-api-docs",
		Status:     AuthStatusOverlayRequired,
		SecuritySchemes: []SecurityScheme{{
			Name:          "frappeToken",
			Type:          SecuritySchemeAPIKey,
			In:            APIKeyInHeader,
			ParameterName: "Authorization",
			Description:   "Frappe/ERPNext API key and secret carried in the Authorization header using the token scheme.",
		}},
		RootSecurity: []SecurityRequirement{{Scheme: "frappeToken"}},
		SourceRefs: []string{
			"https://docs.frappe.io/erpnext/rest-api",
			"https://docs.frappe.io/framework/user/en/api/rest#1-token-based-authentication",
		},
		SourceNote: "ERPNext/Frappe has official human REST API docs but no recorded stable public official OpenAPI document; token authentication metadata comes from advisory overlay notes.",
	},
	{
		ID:         "woocommerce-rest-api-auth-overlay",
		ProviderID: "woocommerce",
		SpecRefID:  "woocommerce-rest-api-v3-docs",
		Status:     AuthStatusOverlayRequired,
		SecuritySchemes: []SecurityScheme{
			{
				Name:        "wooCommerceBasic",
				Type:        SecuritySchemeHTTP,
				Scheme:      "basic",
				Description: "WooCommerce REST API consumer key and consumer secret carried using HTTP Basic authentication over HTTPS.",
			},
			{
				Name:          "wooCommerceConsumerKey",
				Type:          SecuritySchemeAPIKey,
				In:            APIKeyInQuery,
				ParameterName: "consumer_key",
				Description:   "WooCommerce REST API consumer key query parameter for environments where Basic auth is unavailable.",
			},
			{
				Name:          "wooCommerceConsumerSecret",
				Type:          SecuritySchemeAPIKey,
				In:            APIKeyInQuery,
				ParameterName: "consumer_secret",
				Description:   "WooCommerce REST API consumer secret query parameter for environments where Basic auth is unavailable.",
			},
		},
		RootSecuritySets: []SecurityRequirementSet{
			{Requirements: []SecurityRequirement{{Scheme: "wooCommerceBasic"}}},
			{Requirements: []SecurityRequirement{{Scheme: "wooCommerceConsumerKey"}, {Scheme: "wooCommerceConsumerSecret"}}},
		},
		SourceRefs: []string{
			"https://woocommerce.github.io/woocommerce-rest-api-docs/#authentication",
			"https://developer.woocommerce.com/docs/apis/rest-api/v3/",
		},
		SourceNote: "WooCommerce has official human REST API docs but no recorded stable public official OpenAPI document; Basic and query consumer-key auth metadata comes from advisory overlay notes.",
	},
	{
		ID:         "wise-platform-api-auth-overlay",
		ProviderID: "wise",
		SpecRefID:  "wise-platform-api-docs",
		Status:     AuthStatusOverlayRequired,
		SecuritySchemes: []SecurityScheme{{
			Name:         "wiseBearer",
			Type:         SecuritySchemeHTTP,
			Scheme:       "bearer",
			BearerFormat: "Wise OAuth 2.0 access token",
			Description:  "Wise Platform API access token carried in the Authorization bearer header.",
		}},
		RootSecurity: []SecurityRequirement{{Scheme: "wiseBearer"}},
		SourceRefs: []string{
			"https://docs.wise.com/api-reference",
			"https://docs.wise.com/api-docs/features",
		},
		SourceNote: "Wise has official human API docs but no recorded stable public official OpenAPI document; OAuth bearer metadata comes from advisory overlay notes.",
	},
	{
		ID:         "dhl-shipment-tracking-auth-overlay",
		ProviderID: "dhl",
		SpecRefID:  "dhl-shipment-tracking-docs",
		Status:     AuthStatusOverlayRequired,
		SecuritySchemes: []SecurityScheme{{
			Name:          "dhlAPIKey",
			Type:          SecuritySchemeAPIKey,
			In:            APIKeyInHeader,
			ParameterName: "DHL-API-Key",
			Description:   "DHL Shipment Tracking API subscription key carried in the DHL-API-Key header.",
		}},
		RootSecurity: []SecurityRequirement{{Scheme: "dhlAPIKey"}},
		SourceRefs: []string{
			"https://developer.dhl.com/api-reference/shipment-tracking?language_content_entity=en",
		},
		SourceNote: "DHL has official Shipment Tracking human API docs but no recorded stable public downloadable official OpenAPI document; subscription-key metadata comes from advisory overlay notes.",
	},
	{
		ID:         "onfleet-api-auth-overlay",
		ProviderID: "onfleet",
		SpecRefID:  "onfleet-api-reference",
		Status:     AuthStatusOverlayRequired,
		SecuritySchemes: []SecurityScheme{{
			Name:        "onfleetBasic",
			Type:        SecuritySchemeHTTP,
			Scheme:      "basic",
			Description: "Onfleet API key carried as the HTTP Basic username with an empty password.",
		}},
		RootSecurity: []SecurityRequirement{{Scheme: "onfleetBasic"}},
		SourceRefs: []string{
			"https://docs.onfleet.com/reference",
			"https://support.onfleet.com/hc/en-us/articles/360045763292-API",
		},
		SourceNote: "Onfleet has official human API docs but no recorded stable public official OpenAPI document; Basic API-key metadata comes from advisory overlay notes.",
	},
	{
		ID:         "unleashed-software-api-auth-overlay",
		ProviderID: "unleashed-software",
		SpecRefID:  "unleashed-api-docs",
		Status:     AuthStatusOverlayRequired,
		SecuritySchemes: []SecurityScheme{
			{
				Name:          "unleashedAPIID",
				Type:          SecuritySchemeAPIKey,
				In:            APIKeyInHeader,
				ParameterName: "api-auth-id",
				Description:   "Unleashed API ID carried in the api-auth-id header.",
			},
			{
				Name:          "unleashedSignature",
				Type:          SecuritySchemeAPIKey,
				In:            APIKeyInHeader,
				ParameterName: "api-auth-signature",
				Description:   "Unleashed request signature carried in the api-auth-signature header.",
			},
			{
				Name:          "unleashedClientType",
				Type:          SecuritySchemeAPIKey,
				In:            APIKeyInHeader,
				ParameterName: "client-type",
				Description:   "Unleashed client-type header identifying the integration for API support and usage stats.",
			},
		},
		RootSecuritySets: []SecurityRequirementSet{{
			Requirements: []SecurityRequirement{{Scheme: "unleashedAPIID"}, {Scheme: "unleashedSignature"}, {Scheme: "unleashedClientType"}},
		}},
		SourceRefs: []string{
			"https://apidocs.unleashedsoftware.com/",
			"https://apidocs.unleashedsoftware.com/Authentication",
		},
		SourceNote: "Unleashed has official human API docs but no recorded stable public official OpenAPI document; API ID, signature, and client-type metadata comes from advisory overlay notes. Apitools must not compute request signatures.",
	},
	{
		ID:         "workable-api-auth-overlay",
		ProviderID: "workable",
		SpecRefID:  "workable-api-reference",
		Status:     AuthStatusOverlayRequired,
		SecuritySchemes: []SecurityScheme{{
			Name:         "workableBearer",
			Type:         SecuritySchemeHTTP,
			Scheme:       "bearer",
			BearerFormat: "Workable access token",
			Description:  "Workable API access token carried in the Authorization bearer header.",
		}},
		RootSecurity: []SecurityRequirement{{Scheme: "workableBearer"}},
		SourceRefs: []string{
			"https://workable.readme.io/reference",
			"https://help.workable.com/hc/en-us/articles/115013356548-Workable-API-Documentation",
		},
		SourceNote: "Workable has official human API docs but no recorded stable public official OpenAPI document; bearer access-token metadata comes from advisory overlay notes.",
	},
	{
		ID:         "bitwarden-public-api-auth-overlay",
		ProviderID: "bitwarden",
		SpecRefID:  "bitwarden-public-api-docs",
		Status:     AuthStatusOverlayRequired,
		SecuritySchemes: []SecurityScheme{{
			Name:         "bitwardenBearer",
			Type:         SecuritySchemeHTTP,
			Scheme:       "bearer",
			BearerFormat: "Bitwarden Public API access token",
			Description:  "Bitwarden Public API access token carried in the Authorization bearer header after OAuth client-credentials token exchange.",
		}},
		RootSecurity: []SecurityRequirement{{Scheme: "bitwardenBearer"}},
		SourceRefs: []string{
			"https://bitwarden.com/help/api/",
		},
		SourceNote: "Bitwarden has official Public API docs but no recorded stable public downloadable official OpenAPI document; bearer access-token metadata comes from advisory overlay notes. Apitools must not request OAuth tokens.",
	},
	{
		ID:         "cisco-webex-api-auth-overlay",
		ProviderID: "cisco-webex",
		SpecRefID:  "cisco-webex-rooms-docs",
		Status:     AuthStatusOverlayRequired,
		SecuritySchemes: []SecurityScheme{{
			Name:         "webexOAuth2",
			Type:         SecuritySchemeHTTP,
			Scheme:       "bearer",
			BearerFormat: "Cisco Webex OAuth 2.0 access token",
			Description:  "Cisco Webex OAuth access token carried in the Authorization bearer header.",
		}},
		RootSecurity: []SecurityRequirement{{Scheme: "webexOAuth2"}},
		SourceRefs: []string{
			"https://developer.webex.com/docs/api/v1/rooms",
			"https://developer.webex.com/docs/integrations",
		},
		SourceNote: "Cisco Webex has official human API docs but no recorded stable public downloadable official OpenAPI document; OAuth bearer metadata comes from advisory overlay notes.",
	},
	{
		ID:         "cortex-api-auth-overlay",
		ProviderID: "cortex",
		SpecRefID:  "cortex-api-guide",
		Status:     AuthStatusOverlayRequired,
		SecuritySchemes: []SecurityScheme{{
			Name:         "cortexBearer",
			Type:         SecuritySchemeHTTP,
			Scheme:       "bearer",
			BearerFormat: "Cortex API key",
			Description:  "Cortex API key carried in the Authorization bearer header.",
		}},
		RootSecurity: []SecurityRequirement{{Scheme: "cortexBearer"}},
		SourceRefs: []string{
			"https://docs.strangebee.com/cortex/api/api-guide/",
		},
		SourceNote: "Cortex has official human API docs but no recorded stable public official OpenAPI document; bearer API-key metadata comes from advisory overlay notes.",
	},
	{
		ID:         "home-assistant-rest-api-auth-overlay",
		ProviderID: "home-assistant",
		SpecRefID:  "home-assistant-rest-api-docs",
		Status:     AuthStatusOverlayRequired,
		SecuritySchemes: []SecurityScheme{{
			Name:         "homeAssistantBearer",
			Type:         SecuritySchemeHTTP,
			Scheme:       "bearer",
			BearerFormat: "Home Assistant long-lived access token",
			Description:  "Home Assistant long-lived access token carried in the Authorization bearer header.",
		}},
		RootSecurity: []SecurityRequirement{{Scheme: "homeAssistantBearer"}},
		SourceRefs: []string{
			"https://developers.home-assistant.io/docs/api/rest/",
		},
		SourceNote: "Home Assistant has official REST API human docs but no recorded stable public official OpenAPI document; long-lived bearer access-token metadata comes from advisory overlay notes.",
	},
	{
		ID:         "netscaler-adc-nitro-api-auth-overlay",
		ProviderID: "netscaler",
		SpecRefID:  "netscaler-adc-nitro-api-docs",
		Status:     AuthStatusOverlayRequired,
		SecuritySchemes: []SecurityScheme{{
			Name:          "netscalerNitroAuthToken",
			Type:          SecuritySchemeAPIKey,
			In:            APIKeyInHeader,
			ParameterName: "Cookie",
			Description:   "NetScaler ADC NITRO session authentication carried as a NITRO_AUTH_TOKEN cookie after login.",
		}},
		RootSecurity: []SecurityRequirement{{Scheme: "netscalerNitroAuthToken"}},
		SourceRefs: []string{
			"https://developer-docs.netscaler.com/en-us/adc-nitro-api/current-release/api-reference.html",
		},
		SourceNote: "NetScaler ADC has official NITRO API human docs but no recorded stable public official OpenAPI document; NITRO session cookie metadata comes from advisory overlay notes. Apitools must not log in to appliances.",
	},
	{
		ID:         "venafi-api-auth-overlay",
		ProviderID: "venafi",
		SpecRefID:  "venafi-tls-protect-cloud-api-docs",
		Status:     AuthStatusOverlayRequired,
		SecuritySchemes: []SecurityScheme{
			{
				Name:          "venafiCloudAPIKey",
				Type:          SecuritySchemeAPIKey,
				In:            APIKeyInHeader,
				ParameterName: "tppl-api-key",
				Description:   "Venafi TLS Protect Cloud API key carried in the tppl-api-key header.",
			},
			{
				Name:         "venafiDatacenterBearer",
				Type:         SecuritySchemeHTTP,
				Scheme:       "bearer",
				BearerFormat: "Venafi TLS Protect Datacenter OAuth access token",
				Description:  "Venafi TLS Protect Datacenter WebSDK access token carried in the Authorization bearer header.",
			},
		},
		RootSecurity: []SecurityRequirement{{Scheme: "venafiCloudAPIKey"}, {Scheme: "venafiDatacenterBearer"}},
		SourceRefs: []string{
			"https://developer.venafi.com/tlsprotectcloud/reference",
			"https://developer.venafi.com/tlsprotectdatacenter/reference",
		},
		SourceNote: "Venafi has official human API docs for TLS Protect Cloud and Datacenter but no recorded stable public official OpenAPI document; API-key and bearer-token metadata comes from advisory overlay notes.",
	},
	{
		ID:         "wekan-rest-api-auth-overlay",
		ProviderID: "wekan",
		SpecRefID:  "wekan-rest-api-docs",
		Status:     AuthStatusOverlayRequired,
		SecuritySchemes: []SecurityScheme{{
			Name:         "wekanBearer",
			Type:         SecuritySchemeHTTP,
			Scheme:       "bearer",
			BearerFormat: "Wekan session token",
			Description:  "Wekan REST API session token carried in the Authorization bearer header after login.",
		}},
		RootSecurity: []SecurityRequirement{{Scheme: "wekanBearer"}},
		SourceRefs: []string{
			"https://github.com/wekan/wekan/wiki/REST-API",
			"https://github.com/wekan/wekan/wiki/REST-API-Authentication",
		},
		SourceNote: "Wekan has official REST API wiki docs and OpenAPI generation tooling but no recorded stable public generated OpenAPI artifact; bearer session-token metadata comes from advisory overlay notes.",
	},
	{
		ID:         "zammad-api-auth-overlay",
		ProviderID: "zammad",
		SpecRefID:  "zammad-api-intro-docs",
		Status:     AuthStatusOverlayRequired,
		SecuritySchemes: []SecurityScheme{
			{
				Name:        "zammadBasic",
				Type:        SecuritySchemeHTTP,
				Scheme:      "basic",
				Description: "Zammad username and password supplied with HTTP Basic authentication.",
			},
			{
				Name:          "zammadToken",
				Type:          SecuritySchemeAPIKey,
				In:            APIKeyInHeader,
				ParameterName: "Authorization",
				Description:   "Zammad access token carried in the Authorization header using the Token token=... syntax.",
			},
		},
		RootSecurity: []SecurityRequirement{{Scheme: "zammadBasic"}, {Scheme: "zammadToken"}},
		SourceRefs: []string{
			"https://docs.zammad.org/en/latest/api/intro.html",
		},
		SourceNote: "Zammad has official human API docs but no recorded stable public official OpenAPI document; Basic and token authentication metadata comes from advisory overlay notes.",
	},
}

func awsSigV4Overlay(id, providerID, specRefID, serviceName string, sourceRefs []string) SecurityOverlay {
	return SecurityOverlay{
		ID:         id,
		ProviderID: providerID,
		SpecRefID:  specRefID,
		Status:     AuthStatusOverlayRequired,
		SecuritySchemes: []SecurityScheme{
			{
				Name:          "awsSigV4",
				Type:          SecuritySchemeAPIKey,
				In:            APIKeyInHeader,
				ParameterName: "Authorization",
				Description:   serviceName + " requests use AWS Signature Version 4. The Authorization header carries the computed signature; signed requests also require date metadata and may require an x-amz-security-token for temporary credentials.",
			},
		},
		RootSecurity: []SecurityRequirement{{Scheme: "awsSigV4"}},
		SourceRefs:   sourceRefs,
		SourceNote:   serviceName + " uses AWS Signature Version 4. This overlay records signing requirements as metadata only; apitools must not calculate signatures, resolve credentials, or choose AWS accounts.",
	}
}

func parseableSpecAuthReviewOverlay(id, providerID, specRefID string, schemes []SecurityScheme, rootSecurity []SecurityRequirement, sourceRefs []string, sourceNote string) SecurityOverlay {
	return SecurityOverlay{
		ID:              id,
		ProviderID:      providerID,
		SpecRefID:       specRefID,
		Status:          AuthStatusPresentIncomplete,
		SecuritySchemes: schemes,
		RootSecurity:    rootSecurity,
		SourceRefs:      sourceRefs,
		SourceNote:      sourceNote,
	}
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
	overlay.RootSecuritySets = cloneSecurityRequirementSets(overlay.RootSecuritySets)
	overlay.OperationSecurity = cloneOperationSecurity(overlay.OperationSecurity)
	overlay.SourceRefs = append([]string(nil), overlay.SourceRefs...)
	return overlay
}
