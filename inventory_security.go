package apitools

import (
	"sort"
	"strings"
)

func securitySchemes(root map[string]any) map[string]SecuritySummary {
	out := make(map[string]SecuritySummary)
	components := mapValue(root["components"])
	for name, value := range mapValue(components["securitySchemes"]) {
		scheme := mapValue(value)
		out[name] = SecuritySummary{
			Name:             name,
			Type:             stringValue(scheme["type"]),
			Scheme:           stringValue(scheme["scheme"]),
			In:               stringValue(scheme["in"]),
			ParameterName:    stringValue(scheme["name"]),
			Flows:            securitySchemeFlows(scheme),
			OAuthFlows:       securitySchemeOAuthFlows(scheme),
			AuthorizationURL: securitySchemeOAuthURL(scheme, "authorizationUrl"),
			TokenURL:         securitySchemeOAuthURL(scheme, "tokenUrl"),
			RefreshURL:       securitySchemeOAuthURL(scheme, "refreshUrl"),
			Description:      stringValue(scheme["description"]),
			Extensions:       securitySchemeExtensions(scheme),
		}
	}
	for name, value := range mapValue(root["securityDefinitions"]) {
		scheme := mapValue(value)
		out[name] = SecuritySummary{
			Name:             name,
			Type:             stringValue(scheme["type"]),
			Scheme:           stringValue(scheme["scheme"]),
			In:               stringValue(scheme["in"]),
			ParameterName:    stringValue(scheme["name"]),
			Flows:            securitySchemeFlows(scheme),
			OAuthFlows:       securitySchemeOAuthFlows(scheme),
			AuthorizationURL: securitySchemeOAuthURL(scheme, "authorizationUrl"),
			TokenURL:         securitySchemeOAuthURL(scheme, "tokenUrl"),
			RefreshURL:       securitySchemeOAuthURL(scheme, "refreshUrl"),
			Description:      stringValue(scheme["description"]),
			Extensions:       securitySchemeExtensions(scheme),
		}
	}
	return out
}

func securitySchemeFlows(scheme map[string]any) []string {
	flows := sortedMapKeys(mapValue(scheme["flows"]))
	if len(flows) > 0 {
		return flows
	}
	flow := strings.TrimSpace(stringValue(scheme["flow"]))
	if flow == "" {
		return nil
	}
	return []string{flow}
}

func securitySchemeOAuthFlows(scheme map[string]any) []OAuthFlowSummary {
	flows := mapValue(scheme["flows"])
	if len(flows) > 0 {
		out := make([]OAuthFlowSummary, 0, len(flows))
		for _, name := range sortedMapKeys(flows) {
			flow := mapValue(flows[name])
			out = append(out, OAuthFlowSummary{
				Name:             name,
				AuthorizationURL: strings.TrimSpace(stringValue(flow["authorizationUrl"])),
				TokenURL:         strings.TrimSpace(stringValue(flow["tokenUrl"])),
				RefreshURL:       strings.TrimSpace(stringValue(flow["refreshUrl"])),
				Scopes:           sortedMapKeys(mapValue(flow["scopes"])),
			})
		}
		return out
	}
	name := strings.TrimSpace(stringValue(scheme["flow"]))
	if name == "" {
		return nil
	}
	return []OAuthFlowSummary{{
		Name:             name,
		AuthorizationURL: strings.TrimSpace(stringValue(scheme["authorizationUrl"])),
		TokenURL:         strings.TrimSpace(stringValue(scheme["tokenUrl"])),
		RefreshURL:       strings.TrimSpace(stringValue(scheme["refreshUrl"])),
		Scopes:           sortedMapKeys(mapValue(scheme["scopes"])),
	}}
}

func securitySchemeOAuthURL(scheme map[string]any, key string) string {
	if value := strings.TrimSpace(stringValue(scheme[key])); value != "" {
		return value
	}
	var values []string
	for _, flowName := range sortedMapKeys(mapValue(scheme["flows"])) {
		value := strings.TrimSpace(stringValue(mapValue(mapValue(scheme["flows"])[flowName])[key]))
		if value != "" {
			values = append(values, value)
		}
	}
	if len(values) == 0 {
		return ""
	}
	sort.Strings(values)
	return values[0]
}

func securitySchemeExtensions(scheme map[string]any) map[string]string {
	allow := map[string]struct{}{
		"x-amazon-apigateway-authtype": {},
		"x-aws-auth-type":              {},
		"x-aws-signature-version":      {},
	}
	out := map[string]string{}
	for key := range allow {
		value := strings.TrimSpace(stringValue(scheme[key]))
		if value != "" {
			out[key] = value
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func securityRequirementSets(value any, schemes map[string]SecuritySummary) []SecurityRequirementSetSummary {
	values, ok := value.([]any)
	if !ok {
		return nil
	}
	if len(values) == 0 {
		return []SecurityRequirementSetSummary{{}}
	}
	out := make([]SecurityRequirementSetSummary, 0, len(values))
	for _, requirementValue := range values {
		requirement := mapValue(requirementValue)
		set := SecurityRequirementSetSummary{}
		for _, name := range sortedMapKeys(requirement) {
			summary := schemes[name]
			if summary.Name == "" {
				summary.Name = name
			}
			summary.Scopes = stringSlice(requirement[name])
			set.Requirements = append(set.Requirements, summary)
		}
		out = append(out, set)
	}
	return out
}
