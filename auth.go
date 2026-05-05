package apitools

import (
	"fmt"
	"sort"
	"strings"
	"unicode"
)

// AuthRequirementSummary is a runtime-neutral summary of the credential and
// configuration contract implied by OpenAPI security requirements.
type AuthRequirementSummary struct {
	Kind                     string             `json:"kind"`
	Dialect                  string             `json:"dialect,omitempty"`
	Scheme                   string             `json:"scheme,omitempty"`
	Type                     string             `json:"type,omitempty"`
	In                       string             `json:"in,omitempty"`
	ParameterName            string             `json:"parameter_name,omitempty"`
	Flows                    []string           `json:"flows,omitempty"`
	OAuthFlows               []OAuthFlowSummary `json:"oauth_flows,omitempty"`
	AuthorizationURL         string             `json:"authorization_url,omitempty"`
	TokenURL                 string             `json:"token_url,omitempty"`
	RefreshURL               string             `json:"refresh_url,omitempty"`
	Scopes                   []string           `json:"scopes,omitempty"`
	Description              string             `json:"description,omitempty"`
	CredentialFields         []string           `json:"credential_fields,omitempty"`
	OptionalCredentialFields []string           `json:"optional_credential_fields,omitempty"`
	ConfigFields             []string           `json:"config_fields,omitempty"`
	OptionalConfigFields     []string           `json:"optional_config_fields,omitempty"`
}

// AuthRequirementsForOperation derives runtime-neutral auth metadata for one
// operation. It does not resolve credentials, acquire tokens, or sign requests.
func AuthRequirementsForOperation(provider string, operation OperationSummary) []AuthRequirementSummary {
	if len(operation.Security) == 0 {
		return nil
	}
	out := make([]AuthRequirementSummary, 0, len(operation.Security))
	for _, security := range operation.Security {
		requirement := authRequirementForSecurity(provider, security)
		if requirement.Kind == "" {
			continue
		}
		out = append(out, requirement)
	}
	return mergeAuthRequirements(out)
}

// AuthRequirementsForOperations derives merged runtime-neutral auth metadata
// for a set of operations. Scopes and flows are merged deterministically.
func AuthRequirementsForOperations(provider string, operations []OperationSummary) []AuthRequirementSummary {
	if len(operations) == 0 {
		return nil
	}
	operations = append([]OperationSummary(nil), operations...)
	sort.SliceStable(operations, func(i, j int) bool {
		if operations[i].OperationID != operations[j].OperationID {
			return operations[i].OperationID < operations[j].OperationID
		}
		if operations[i].Path != operations[j].Path {
			return operations[i].Path < operations[j].Path
		}
		return operations[i].Method < operations[j].Method
	})
	var summaries []AuthRequirementSummary
	for _, operation := range operations {
		summaries = append(summaries, AuthRequirementsForOperation(provider, operation)...)
	}
	return mergeAuthRequirements(summaries)
}

func authRequirementForSecurity(provider string, security SecuritySummary) AuthRequirementSummary {
	requirement := AuthRequirementSummary{
		Scheme:           strings.TrimSpace(security.Name),
		Type:             strings.TrimSpace(security.Type),
		In:               strings.TrimSpace(security.In),
		ParameterName:    strings.TrimSpace(security.ParameterName),
		Flows:            sortedUniqueStrings(security.Flows),
		OAuthFlows:       sortedOAuthFlows(security.OAuthFlows),
		AuthorizationURL: strings.TrimSpace(security.AuthorizationURL),
		TokenURL:         strings.TrimSpace(security.TokenURL),
		RefreshURL:       strings.TrimSpace(security.RefreshURL),
		Scopes:           sortedUniqueStrings(security.Scopes),
		Description:      strings.TrimSpace(security.Description),
	}
	if isAWSSignatureSecurity(provider, security) {
		requirement.Kind = "aws_signature"
		requirement.CredentialFields = []string{"aws_access_key_id", "aws_secret_access_key"}
		requirement.OptionalCredentialFields = []string{"aws_session_token"}
		requirement.ConfigFields = []string{"region", "service"}
		if requirement.Description == "" {
			requirement.Description = fmt.Sprintf("OpenAPI security scheme %q requires AWS Signature Version 4 metadata.", requirement.Scheme)
		}
		return requirement
	}
	switch strings.ToLower(strings.TrimSpace(security.Type)) {
	case "apikey":
		requirement.Kind = "api_key"
		field := sanitizedCredentialField(firstNonEmptyAuthString(security.ParameterName, security.Name, "api_key"))
		if field != "" {
			requirement.CredentialFields = []string{field}
		}
	case "http":
		switch strings.ToLower(strings.TrimSpace(security.Scheme)) {
		case "bearer":
			requirement.Kind = "bearer"
			requirement.CredentialFields = []string{"access_token"}
		case "basic":
			requirement.Kind = "basic"
			requirement.CredentialFields = []string{"username", "password"}
		default:
			requirement.Kind = "http"
		}
	case "oauth2":
		requirement.Kind = "oauth2"
		if isGCPOAuth2Security(provider, security) {
			requirement.Dialect = "gcp"
			requirement.CredentialFields = []string{"oauth_client_id", "oauth_client_secret"}
			requirement.ConfigFields = []string{"oauth_auth_uri", "oauth_token_uri", "oauth_redirect_uri"}
			requirement.OptionalConfigFields = []string{"oauth_project_id", "oauth_scopes"}
		}
	case "openidconnect":
		requirement.Kind = "openid_connect"
	default:
		requirement.Kind = firstNonEmptyAuthString(strings.ToLower(strings.TrimSpace(security.Type)), "credential")
	}
	if requirement.Description == "" {
		requirement.Description = genericAuthDescription(requirement)
	}
	return requirement
}

func isGCPOAuth2Security(provider string, security SecuritySummary) bool {
	if !strings.EqualFold(strings.TrimSpace(security.Type), "oauth2") {
		return false
	}
	provider = strings.ToLower(strings.TrimSpace(provider))
	if root, _, ok := strings.Cut(provider, "."); ok {
		provider = root
	}
	switch provider {
	case "gcp", "google", "googleapis", "gmail":
		return true
	}
	values := []string{security.AuthorizationURL, security.TokenURL}
	for _, flow := range security.OAuthFlows {
		values = append(values, flow.AuthorizationURL, flow.TokenURL, flow.RefreshURL)
	}
	for _, value := range values {
		normalized := strings.ToLower(strings.TrimSpace(value))
		if strings.Contains(normalized, "google") || strings.Contains(normalized, "googleapis.com") {
			return true
		}
	}
	return false
}

func isAWSSignatureSecurity(provider string, security SecuritySummary) bool {
	provider = strings.ToLower(strings.TrimSpace(provider))
	if root, _, ok := strings.Cut(provider, "."); ok {
		provider = root
	}
	if provider == "aws" {
		if strings.EqualFold(security.Name, "hmac") {
			return true
		}
		if strings.EqualFold(security.Type, "apiKey") && strings.EqualFold(security.ParameterName, "Authorization") {
			return true
		}
	}
	for _, value := range []string{security.Name, security.Type, security.Scheme, security.Description} {
		normalized := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(value), " ", ""))
		if strings.Contains(normalized, "awssigv4") || strings.Contains(normalized, "sigv4") || strings.Contains(normalized, "signatureversion4") {
			return true
		}
	}
	for _, value := range security.Extensions {
		normalized := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(value), " ", ""))
		if strings.Contains(normalized, "aws") && (strings.Contains(normalized, "sigv4") || strings.Contains(normalized, "iam")) {
			return true
		}
	}
	return false
}

func mergeAuthRequirements(in []AuthRequirementSummary) []AuthRequirementSummary {
	seen := map[string]AuthRequirementSummary{}
	for _, requirement := range in {
		key := authRequirementKey(requirement)
		if key == "" {
			continue
		}
		if existing, ok := seen[key]; ok {
			existing.Flows = sortedUniqueStrings(append(existing.Flows, requirement.Flows...))
			existing.OAuthFlows = mergeOAuthFlows(existing.OAuthFlows, requirement.OAuthFlows)
			existing.Scopes = sortedUniqueStrings(append(existing.Scopes, requirement.Scopes...))
			if existing.Dialect == "" {
				existing.Dialect = requirement.Dialect
			}
			if existing.AuthorizationURL == "" {
				existing.AuthorizationURL = requirement.AuthorizationURL
			}
			if existing.TokenURL == "" {
				existing.TokenURL = requirement.TokenURL
			}
			if existing.RefreshURL == "" {
				existing.RefreshURL = requirement.RefreshURL
			}
			existing.CredentialFields = appendUniqueAuthStrings(existing.CredentialFields, requirement.CredentialFields...)
			existing.OptionalCredentialFields = appendUniqueAuthStrings(existing.OptionalCredentialFields, requirement.OptionalCredentialFields...)
			existing.ConfigFields = appendUniqueAuthStrings(existing.ConfigFields, requirement.ConfigFields...)
			existing.OptionalConfigFields = appendUniqueAuthStrings(existing.OptionalConfigFields, requirement.OptionalConfigFields...)
			if existing.Description == "" {
				existing.Description = requirement.Description
			}
			seen[key] = existing
			continue
		}
		requirement.Flows = sortedUniqueStrings(requirement.Flows)
		requirement.OAuthFlows = sortedOAuthFlows(requirement.OAuthFlows)
		requirement.Scopes = sortedUniqueStrings(requirement.Scopes)
		seen[key] = requirement
	}
	out := make([]AuthRequirementSummary, 0, len(seen))
	for _, requirement := range seen {
		out = append(out, requirement)
	}
	sort.Slice(out, func(i, j int) bool { return authRequirementKey(out[i]) < authRequirementKey(out[j]) })
	return out
}

func authRequirementKey(requirement AuthRequirementSummary) string {
	return strings.ToLower(strings.Join(nonEmptyStrings([]string{
		requirement.Kind,
		requirement.Scheme,
		requirement.Type,
		requirement.In,
		requirement.ParameterName,
	}...), "\x00"))
}

func genericAuthDescription(requirement AuthRequirementSummary) string {
	location := ""
	if requirement.In != "" || requirement.ParameterName != "" {
		location = " at " + strings.TrimSpace(strings.Join(nonEmptyStrings(requirement.In, requirement.ParameterName), " "))
	}
	return fmt.Sprintf("OpenAPI security scheme %q requires %s authentication%s.", requirement.Scheme, firstNonEmptyAuthString(requirement.Type, requirement.Kind, "security"), location)
}

func sortedUniqueStrings(values []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func mergeOAuthFlows(base []OAuthFlowSummary, values []OAuthFlowSummary) []OAuthFlowSummary {
	if len(base) == 0 {
		return sortedOAuthFlows(values)
	}
	out := append([]OAuthFlowSummary(nil), base...)
	seen := map[string]int{}
	for i, flow := range out {
		seen[oauthFlowKey(flow)] = i
	}
	for _, flow := range values {
		flow.Name = strings.TrimSpace(flow.Name)
		if flow.Name == "" {
			continue
		}
		flow.AuthorizationURL = strings.TrimSpace(flow.AuthorizationURL)
		flow.TokenURL = strings.TrimSpace(flow.TokenURL)
		flow.RefreshURL = strings.TrimSpace(flow.RefreshURL)
		flow.Scopes = sortedUniqueStrings(flow.Scopes)
		key := oauthFlowKey(flow)
		if i, ok := seen[key]; ok {
			out[i].Scopes = sortedUniqueStrings(append(out[i].Scopes, flow.Scopes...))
			continue
		}
		seen[key] = len(out)
		out = append(out, flow)
	}
	return sortedOAuthFlows(out)
}

func sortedOAuthFlows(values []OAuthFlowSummary) []OAuthFlowSummary {
	out := make([]OAuthFlowSummary, 0, len(values))
	for _, flow := range values {
		flow.Name = strings.TrimSpace(flow.Name)
		if flow.Name == "" {
			continue
		}
		flow.AuthorizationURL = strings.TrimSpace(flow.AuthorizationURL)
		flow.TokenURL = strings.TrimSpace(flow.TokenURL)
		flow.RefreshURL = strings.TrimSpace(flow.RefreshURL)
		flow.Scopes = sortedUniqueStrings(flow.Scopes)
		out = append(out, flow)
	}
	sort.Slice(out, func(i, j int) bool {
		return oauthFlowKey(out[i]) < oauthFlowKey(out[j])
	})
	return out
}

func oauthFlowKey(flow OAuthFlowSummary) string {
	return strings.Join([]string{
		strings.ToLower(strings.TrimSpace(flow.Name)),
		strings.TrimSpace(flow.AuthorizationURL),
		strings.TrimSpace(flow.TokenURL),
		strings.TrimSpace(flow.RefreshURL),
	}, "\x00")
}

func appendUniqueAuthStrings(base []string, values ...string) []string {
	seen := map[string]struct{}{}
	for _, value := range base {
		seen[value] = struct{}{}
	}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		base = append(base, value)
	}
	return base
}

func sanitizedCredentialField(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	var b strings.Builder
	lastUnderscore := false
	for _, r := range value {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(unicode.ToLower(r))
			lastUnderscore = false
			continue
		}
		if !lastUnderscore && b.Len() > 0 {
			b.WriteByte('_')
			lastUnderscore = true
		}
	}
	out := strings.Trim(b.String(), "_")
	if out == "" {
		return ""
	}
	if unicode.IsDigit([]rune(out)[0]) {
		out = "field_" + out
	}
	return out
}

func firstNonEmptyAuthString(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}
