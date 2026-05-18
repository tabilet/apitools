package catalog

import (
	"fmt"
	"sort"
	"strings"
)

// AuthCompletenessStatus records whether security metadata is sufficient for
// downstream inspection. It does not imply credentials are available or valid.
type AuthCompletenessStatus string

const (
	AuthStatusComplete               AuthCompletenessStatus = "complete"
	AuthStatusPresentIncomplete      AuthCompletenessStatus = "present-incomplete"
	AuthStatusAbsent                 AuthCompletenessStatus = "absent"
	AuthStatusOverlayRequired        AuthCompletenessStatus = "overlay-required"
	AuthStatusIntentionallyAnonymous AuthCompletenessStatus = "intentionally-anonymous"
	AuthStatusUnknown                AuthCompletenessStatus = "unknown"
)

// SecuritySchemeType mirrors the OpenAPI security scheme type values used by
// catalog overlays.
type SecuritySchemeType string

const (
	SecuritySchemeAPIKey        SecuritySchemeType = "apiKey"
	SecuritySchemeHTTP          SecuritySchemeType = "http"
	SecuritySchemeOAuth2        SecuritySchemeType = "oauth2"
	SecuritySchemeOpenIDConnect SecuritySchemeType = "openIdConnect"
)

// APIKeyLocation records where an API key is carried.
type APIKeyLocation string

const (
	APIKeyInQuery  APIKeyLocation = "query"
	APIKeyInHeader APIKeyLocation = "header"
	APIKeyInCookie APIKeyLocation = "cookie"
)

// OAuthFlowType mirrors OpenAPI OAuth flow names.
type OAuthFlowType string

const (
	OAuthFlowImplicit          OAuthFlowType = "implicit"
	OAuthFlowPassword          OAuthFlowType = "password"
	OAuthFlowClientCredentials OAuthFlowType = "clientCredentials"
	OAuthFlowAuthorizationCode OAuthFlowType = "authorizationCode"
)

// SecurityScheme is a metadata-only security scheme description. It is safe to
// expose in reports because it never contains credential values.
type SecurityScheme struct {
	Name             string             `json:"name"`
	Type             SecuritySchemeType `json:"type"`
	Description      string             `json:"description,omitempty"`
	Scheme           string             `json:"scheme,omitempty"`
	BearerFormat     string             `json:"bearer_format,omitempty"`
	In               APIKeyLocation     `json:"in,omitempty"`
	ParameterName    string             `json:"parameter_name,omitempty"`
	OpenIDConnectURL string             `json:"open_id_connect_url,omitempty"`
	Flows            []OAuthFlow        `json:"flows,omitempty"`
	Scopes           []string           `json:"scopes,omitempty"`
}

// OAuthFlow records OAuth endpoint and scope metadata without any token values.
type OAuthFlow struct {
	Type             OAuthFlowType `json:"type"`
	AuthorizationURL string        `json:"authorization_url,omitempty"`
	TokenURL         string        `json:"token_url,omitempty"`
	RefreshURL       string        `json:"refresh_url,omitempty"`
	Scopes           []string      `json:"scopes,omitempty"`
}

// SecurityRequirement records an OpenAPI-style security requirement.
type SecurityRequirement struct {
	Scheme string   `json:"scheme"`
	Scopes []string `json:"scopes,omitempty"`
}

// OperationMatch identifies operation-level overlay targets without binding to
// a live endpoint call.
type OperationMatch struct {
	OperationID string   `json:"operation_id,omitempty"`
	Method      string   `json:"method,omitempty"`
	Path        string   `json:"path,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

// OperationSecurity records operation-level security metadata for matched
// operations.
type OperationSecurity struct {
	Match    OperationMatch        `json:"match"`
	Security []SecurityRequirement `json:"security,omitempty"`
}

// SecurityMetadata is the normalized input accepted by the auth classifier.
type SecurityMetadata struct {
	SecuritySchemes        []SecurityScheme      `json:"security_schemes,omitempty"`
	RootSecurity           []SecurityRequirement `json:"root_security,omitempty"`
	OperationSecurity      []OperationSecurity   `json:"operation_security,omitempty"`
	IntentionallyAnonymous bool                  `json:"intentionally_anonymous,omitempty"`
}

// ClassifyAuthCompleteness classifies OpenAPI-style security metadata without
// resolving credentials, choosing accounts, or calling provider APIs.
func ClassifyAuthCompleteness(metadata SecurityMetadata) AuthCompletenessStatus {
	hasSchemes := len(metadata.SecuritySchemes) > 0
	hasRequirements := len(metadata.RootSecurity) > 0
	for _, operation := range metadata.OperationSecurity {
		if len(operation.Security) > 0 {
			hasRequirements = true
			break
		}
	}

	if metadata.IntentionallyAnonymous && !hasSchemes && !hasRequirements {
		return AuthStatusIntentionallyAnonymous
	}
	if !hasSchemes && !hasRequirements {
		return AuthStatusAbsent
	}
	if !hasSchemes || !hasRequirements {
		return AuthStatusPresentIncomplete
	}

	schemes := map[string]struct{}{}
	for _, scheme := range metadata.SecuritySchemes {
		if err := validateSecurityScheme(scheme); err != nil {
			return AuthStatusPresentIncomplete
		}
		if _, exists := schemes[scheme.Name]; exists {
			return AuthStatusPresentIncomplete
		}
		schemes[scheme.Name] = struct{}{}
	}
	for _, requirement := range metadata.RootSecurity {
		if err := validateSecurityRequirement(requirement, schemes); err != nil {
			return AuthStatusPresentIncomplete
		}
	}
	for _, operation := range metadata.OperationSecurity {
		if len(operation.Security) == 0 {
			continue
		}
		if err := validateOperationSecurity(operation, schemes); err != nil {
			return AuthStatusPresentIncomplete
		}
	}
	return AuthStatusComplete
}

func validAuthCompletenessStatus(value AuthCompletenessStatus) bool {
	switch value {
	case AuthStatusComplete, AuthStatusPresentIncomplete, AuthStatusAbsent, AuthStatusOverlayRequired, AuthStatusIntentionallyAnonymous, AuthStatusUnknown:
		return true
	default:
		return false
	}
}

func validateSecurityScheme(scheme SecurityScheme) error {
	if !validSchemeName(scheme.Name) {
		return fmt.Errorf("invalid security scheme name %q", scheme.Name)
	}
	switch scheme.Type {
	case SecuritySchemeAPIKey:
		if !validAPIKeyLocation(scheme.In) {
			return fmt.Errorf("security scheme %q: invalid api key location %q", scheme.Name, scheme.In)
		}
		if strings.TrimSpace(scheme.ParameterName) == "" {
			return fmt.Errorf("security scheme %q: api key requires parameter name", scheme.Name)
		}
	case SecuritySchemeHTTP:
		if strings.TrimSpace(scheme.Scheme) == "" {
			return fmt.Errorf("security scheme %q: http requires scheme", scheme.Name)
		}
	case SecuritySchemeOAuth2:
		if len(scheme.Flows) == 0 {
			return fmt.Errorf("security scheme %q: oauth2 requires at least one flow", scheme.Name)
		}
		for i, flow := range scheme.Flows {
			if err := validateOAuthFlow(flow); err != nil {
				return fmt.Errorf("security scheme %q flow[%d]: %w", scheme.Name, i, err)
			}
		}
	case SecuritySchemeOpenIDConnect:
		if !validHTTPSURL(scheme.OpenIDConnectURL) {
			return fmt.Errorf("security scheme %q: openid connect url must be https", scheme.Name)
		}
	default:
		return fmt.Errorf("security scheme %q: invalid type %q", scheme.Name, scheme.Type)
	}
	if err := validateUniqueStrings("scope", scheme.Scopes); err != nil {
		return fmt.Errorf("security scheme %q: %w", scheme.Name, err)
	}
	return nil
}

func validateOAuthFlow(flow OAuthFlow) error {
	switch flow.Type {
	case OAuthFlowImplicit:
		if !validHTTPSURL(flow.AuthorizationURL) {
			return fmt.Errorf("implicit flow requires https authorization url")
		}
	case OAuthFlowPassword, OAuthFlowClientCredentials:
		if !validHTTPSURL(flow.TokenURL) {
			return fmt.Errorf("%s flow requires https token url", flow.Type)
		}
	case OAuthFlowAuthorizationCode:
		if !validHTTPSURL(flow.AuthorizationURL) {
			return fmt.Errorf("authorizationCode flow requires https authorization url")
		}
		if !validHTTPSURL(flow.TokenURL) {
			return fmt.Errorf("authorizationCode flow requires https token url")
		}
	default:
		return fmt.Errorf("invalid oauth flow type %q", flow.Type)
	}
	if strings.TrimSpace(flow.RefreshURL) != "" && !validHTTPSURL(flow.RefreshURL) {
		return fmt.Errorf("refresh url must be https")
	}
	return validateUniqueStrings("scope", flow.Scopes)
}

func validateSecurityRequirement(requirement SecurityRequirement, schemes map[string]struct{}) error {
	if !validSchemeName(requirement.Scheme) {
		return fmt.Errorf("invalid security requirement scheme %q", requirement.Scheme)
	}
	if _, ok := schemes[requirement.Scheme]; !ok {
		return fmt.Errorf("security requirement references unknown scheme %q", requirement.Scheme)
	}
	return validateUniqueStrings("scope", requirement.Scopes)
}

func validateOperationSecurity(operation OperationSecurity, schemes map[string]struct{}) error {
	if err := validateOperationMatch(operation.Match); err != nil {
		return err
	}
	for i, requirement := range operation.Security {
		if err := validateSecurityRequirement(requirement, schemes); err != nil {
			return fmt.Errorf("security[%d]: %w", i, err)
		}
	}
	return nil
}

func validateOperationMatch(match OperationMatch) error {
	if strings.TrimSpace(match.OperationID) == "" && strings.TrimSpace(match.Method) == "" && strings.TrimSpace(match.Path) == "" && len(match.Tags) == 0 {
		return fmt.Errorf("operation match requires operation id, method/path, or tag")
	}
	method := strings.TrimSpace(match.Method)
	path := strings.TrimSpace(match.Path)
	if method != "" || path != "" {
		if !validHTTPMethod(method) {
			return fmt.Errorf("invalid operation method %q", match.Method)
		}
		if !strings.HasPrefix(path, "/") {
			return fmt.Errorf("operation path must start with /")
		}
	}
	return validateUniqueStrings("tag", match.Tags)
}

func validSchemeName(name string) bool {
	if strings.TrimSpace(name) != name || name == "" || len(name) > 128 {
		return false
	}
	for _, r := range name {
		if r >= 'a' && r <= 'z' {
			continue
		}
		if r >= 'A' && r <= 'Z' {
			continue
		}
		if r >= '0' && r <= '9' {
			continue
		}
		if r == '.' || r == '_' || r == '-' {
			continue
		}
		return false
	}
	return true
}

func validAPIKeyLocation(value APIKeyLocation) bool {
	switch value {
	case APIKeyInQuery, APIKeyInHeader, APIKeyInCookie:
		return true
	default:
		return false
	}
}

func validHTTPMethod(method string) bool {
	switch strings.ToUpper(strings.TrimSpace(method)) {
	case "GET", "PUT", "POST", "DELETE", "OPTIONS", "HEAD", "PATCH", "TRACE":
		return true
	default:
		return false
	}
}

func validHTTPSURL(value string) bool {
	return strings.HasPrefix(strings.TrimSpace(value), "https://")
}

func validateUniqueStrings(label string, values []string) error {
	seen := map[string]struct{}{}
	for i, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			return fmt.Errorf("%s[%d]: empty value", label, i)
		}
		if _, exists := seen[trimmed]; exists {
			return fmt.Errorf("duplicate %s %q", label, trimmed)
		}
		seen[trimmed] = struct{}{}
	}
	return nil
}

func cloneSecuritySchemes(in []SecurityScheme) []SecurityScheme {
	out := make([]SecurityScheme, len(in))
	for i, scheme := range in {
		out[i] = scheme
		out[i].Flows = cloneOAuthFlows(scheme.Flows)
		out[i].Scopes = append([]string(nil), scheme.Scopes...)
	}
	return out
}

func cloneOAuthFlows(in []OAuthFlow) []OAuthFlow {
	out := make([]OAuthFlow, len(in))
	for i, flow := range in {
		out[i] = flow
		out[i].Scopes = append([]string(nil), flow.Scopes...)
	}
	return out
}

func cloneSecurityRequirements(in []SecurityRequirement) []SecurityRequirement {
	out := make([]SecurityRequirement, len(in))
	for i, requirement := range in {
		out[i] = requirement
		out[i].Scopes = append([]string(nil), requirement.Scopes...)
	}
	return out
}

func cloneOperationSecurity(in []OperationSecurity) []OperationSecurity {
	out := make([]OperationSecurity, len(in))
	for i, operation := range in {
		out[i] = operation
		out[i].Match.Tags = append([]string(nil), operation.Match.Tags...)
		out[i].Security = cloneSecurityRequirements(operation.Security)
	}
	return out
}

func sortedStrings(values []string) []string {
	out := append([]string(nil), values...)
	sort.Strings(out)
	return out
}
