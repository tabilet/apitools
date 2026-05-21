//go:build ignore

package main

import (
	"encoding/json"
	"os"
	"sort"
)

type overlaySpec struct {
	ProviderID   string
	Title        string
	Description  string
	ServerURL    string
	Sources      []string
	SourceNote   string
	Security     map[string]map[string]any
	SecurityReqs []map[string][]string
	Schemas      []string
	Paths        map[string]map[string]any
	OutputPath   string
}

func main() {
	for _, spec := range []overlaySpec{
		marketoOverlay(),
		aircallOverlay(),
		checkrOverlay(),
		apolloOverlay(),
		aftershipOverlay(),
		acrobatSignOverlay(),
	} {
		write(spec.OutputPath, build(spec))
	}
}

func marketoOverlay() overlaySpec {
	security := map[string]map[string]any{
		"marketoBearer": {"type": "http", "scheme": "bearer", "bearerFormat": "Marketo access token", "description": "Marketo REST API access token carried in the Authorization bearer header."},
	}
	return overlaySpec{
		ProviderID:   "marketo",
		Title:        "Adobe Marketo Engage REST API Advisory Overlay",
		Description:  "Advisory OpenAPI overlay derived from official Adobe Marketo Engage REST API human documentation. This is not an official Adobe OpenAPI document.",
		ServerURL:    "https://{marketo_endpoint_host}",
		Sources:      []string{"https://experienceleague.adobe.com/en/docs/marketo-developer/marketo/rest/rest-api", "https://experienceleague.adobe.com/en/docs/marketo-developer/marketo/rest/lead-database/leads", "https://experienceleague.adobe.com/en/docs/marketo-developer/marketo/rest/lead-database/activities"},
		SourceNote:   "Adobe Marketo Engage publishes human REST documentation but no recorded stable public official OpenAPI document; this overlay covers selected lead metadata, lead retrieval, and activity retrieval endpoints.",
		Security:     security,
		SecurityReqs: securityReqs("marketoBearer"),
		Schemas:      []string{"MarketoObject", "MarketoCollection", "MarketoError"},
		OutputPath:   "catalog-openapi-cache/advisory-overlays/marketo-rest-api-overlay.json",
		Paths: map[string]map[string]any{
			"/rest/v1/activities.json":       {"get": op("listMarketoActivities", "Get lead activities", params(queryRequired("activityTypeIds", "Comma-separated activity type IDs."), queryRequired("nextPageToken", "Paging token returned by Marketo.")), "", "#/components/schemas/MarketoCollection", securityReqs("marketoBearer"))},
			"/rest/v1/activities/types.json": {"get": op("listMarketoActivityTypes", "Get activity types", nil, "", "#/components/schemas/MarketoCollection", securityReqs("marketoBearer"))},
			"/rest/v1/lead/{id}.json":        {"get": op("getMarketoLeadByID", "Get lead by ID", params(path("id", "Lead ID."), query("fields", "Comma-separated API field names.")), "", "#/components/schemas/MarketoObject", securityReqs("marketoBearer"))},
			"/rest/v1/leads.json":            {"get": op("listMarketoLeadsByFilter", "Get leads by filter type", params(queryRequired("filterType", "Lead field used for filtering."), queryRequired("filterValues", "Comma-separated filter values."), query("fields", "Comma-separated API field names.")), "", "#/components/schemas/MarketoCollection", securityReqs("marketoBearer"))},
			"/rest/v1/leads/describe.json":   {"get": op("describeMarketoLeads", "Describe lead fields", nil, "", "#/components/schemas/MarketoObject", securityReqs("marketoBearer"))},
		},
	}
}

func aircallOverlay() overlaySpec {
	security := map[string]map[string]any{
		"aircallBasic": {"type": "http", "scheme": "basic", "description": "Aircall api_id and api_token supplied with HTTP Basic authentication."},
		"aircallOAuth2": {"type": "oauth2", "description": "Aircall partner OAuth 2.0 authorization-code flow for public_api access.", "flows": map[string]any{
			"authorizationCode": map[string]any{
				"authorizationUrl": "https://dashboard.aircall.io/oauth/authorize",
				"tokenUrl":         "https://api.aircall.io/v1/oauth/token",
				"scopes":           map[string]string{"public_api": "Access Aircall Public API resources."},
			},
		}},
	}
	reqs := []map[string][]string{{"aircallBasic": []string{}}, {"aircallOAuth2": []string{"public_api"}}}
	return overlaySpec{
		ProviderID:   "aircall",
		Title:        "Aircall Public API Advisory Overlay",
		Description:  "Advisory OpenAPI overlay derived from official Aircall Public API human documentation. This is not an official Aircall OpenAPI document.",
		ServerURL:    "https://api.aircall.io/v1",
		Sources:      []string{"https://developer.aircall.io/api-references/"},
		SourceNote:   "Aircall publishes human API documentation but no recorded stable public official OpenAPI document; this overlay covers selected read-oriented users, availability, calls, and contacts endpoints.",
		Security:     security,
		SecurityReqs: reqs,
		Schemas:      []string{"AircallObject", "AircallCollection", "AircallError"},
		OutputPath:   "catalog-openapi-cache/advisory-overlays/aircall-public-api-overlay.json",
		Paths: map[string]map[string]any{
			"/calls":                {"get": op("listAircallCalls", "List calls", params(query("page", "Page number."), query("per_page", "Items per page.")), "", "#/components/schemas/AircallCollection", reqs)},
			"/contacts":             {"get": op("listAircallContacts", "List contacts", params(query("page", "Page number."), query("per_page", "Items per page.")), "", "#/components/schemas/AircallCollection", reqs)},
			"/users":                {"get": op("listAircallUsers", "List users", nil, "", "#/components/schemas/AircallCollection", reqs)},
			"/users/{id}":           {"get": op("getAircallUser", "Retrieve user", params(path("id", "User ID or email.")), "", "#/components/schemas/AircallObject", reqs)},
			"/users/availabilities": {"get": op("listAircallUserAvailabilities", "Retrieve users availability", nil, "", "#/components/schemas/AircallCollection", reqs)},
		},
	}
}

func checkrOverlay() overlaySpec {
	security := map[string]map[string]any{
		"checkrBasic": {"type": "http", "scheme": "basic", "description": "Checkr API key supplied with HTTP Basic authentication."},
	}
	return overlaySpec{
		ProviderID:   "checkr",
		Title:        "Checkr API Advisory Overlay",
		Description:  "Advisory OpenAPI overlay derived from official Checkr human API documentation. This is not an official Checkr OpenAPI document.",
		ServerURL:    "https://api.checkr.com",
		Sources:      []string{"https://docs.checkr.com/"},
		SourceNote:   "Checkr publishes Redoc-rendered human API documentation but no recorded stable public official OpenAPI JSON URL; this overlay covers a conservative read-oriented account, candidate, package, and report subset.",
		Security:     security,
		SecurityReqs: securityReqs("checkrBasic"),
		Schemas:      []string{"CheckrObject", "CheckrCollection", "CheckrError"},
		OutputPath:   "catalog-openapi-cache/advisory-overlays/checkr-api-overlay.json",
		Paths: map[string]map[string]any{
			"/v1/account":         {"get": op("getCheckrAccount", "Get account", nil, "", "#/components/schemas/CheckrObject", securityReqs("checkrBasic"))},
			"/v1/candidates/{id}": {"get": op("getCheckrCandidate", "Get candidate", params(path("id", "Candidate ID.")), "", "#/components/schemas/CheckrObject", securityReqs("checkrBasic"))},
			"/v1/packages":        {"get": op("listCheckrPackages", "List packages", params(query("page", "Page number."), query("per_page", "Items per page.")), "", "#/components/schemas/CheckrCollection", securityReqs("checkrBasic"))},
			"/v1/reports/{id}":    {"get": op("getCheckrReport", "Get report", params(path("id", "Report ID.")), "", "#/components/schemas/CheckrObject", securityReqs("checkrBasic"))},
		},
	}
}

func apolloOverlay() overlaySpec {
	security := map[string]map[string]any{
		"apolloAPIKey": {"type": "apiKey", "in": "header", "name": "x-api-key", "description": "Apollo API key carried in the x-api-key request header."},
		"apolloBearer": {"type": "http", "scheme": "bearer", "bearerFormat": "OAuth access token", "description": "Apollo OAuth access token carried in the Authorization bearer header."},
	}
	reqs := []map[string][]string{{"apolloAPIKey": []string{}}, {"apolloBearer": []string{}}}
	return overlaySpec{
		ProviderID:   "apollo",
		Title:        "Apollo.io API Advisory Overlay",
		Description:  "Advisory OpenAPI overlay derived from official Apollo human API documentation and endpoint-level OpenAPI fragments. This is not a provider-wide official Apollo OpenAPI document.",
		ServerURL:    "https://api.apollo.io/api/v1",
		Sources:      []string{"https://docs.apollo.io/llms.txt", "https://docs.apollo.io/reference/authentication.md", "https://docs.apollo.io/reference/people-api-search.md", "https://docs.apollo.io/reference/organization-search.md"},
		SourceNote:   "Apollo publishes endpoint-level OpenAPI fragments in official docs but no recorded stable provider-wide OpenAPI artifact; this overlay covers selected search endpoints.",
		Security:     security,
		SecurityReqs: reqs,
		Schemas:      []string{"ApolloObject", "ApolloCollection", "ApolloError"},
		OutputPath:   "catalog-openapi-cache/advisory-overlays/apollo-api-overlay.json",
		Paths: map[string]map[string]any{
			"/mixed_companies/search": {"post": op("searchApolloOrganizations", "Organization search", params(
				query("q_organization_domains_list[]", "Organization domains to include."),
				query("organization_locations[]", "Organization headquarters locations."),
				query("q_organization_name", "Organization name filter."),
				query("organization_ids[]", "Apollo organization IDs to include."),
				query("page", "Page number."),
				query("per_page", "Results per page."),
			), "", "#/components/schemas/ApolloCollection", reqs)},
			"/mixed_people/api_search": {"post": op("searchApolloPeople", "People API search", params(
				query("person_titles[]", "Job titles to include."),
				query("q_keywords", "Keyword filter."),
				query("person_locations[]", "Person locations to include."),
				query("person_seniorities[]", "Person seniorities to include."),
				query("organization_locations[]", "Employer headquarters locations."),
				query("q_organization_domains_list[]", "Employer domains to include."),
				query("page", "Page number."),
				query("per_page", "Results per page."),
			), "", "#/components/schemas/ApolloCollection", reqs)},
		},
	}
}

func aftershipOverlay() overlaySpec {
	security := map[string]map[string]any{
		"aftershipAPIKey": {"type": "apiKey", "in": "header", "name": "as-api-key", "description": "AfterShip Tracking API key carried in the as-api-key request header."},
	}
	return overlaySpec{
		ProviderID:   "aftership",
		Title:        "AfterShip Tracking API Advisory Overlay",
		Description:  "Advisory OpenAPI overlay derived from official AfterShip Tracking API human documentation. This is not an official AfterShip OpenAPI document.",
		ServerURL:    "https://api.aftership.com/tracking/2026-01",
		Sources:      []string{"https://www.aftership.com/docs/tracking", "https://www.aftership.com/docs/tracking/fcd9acb5f448a-api-overview", "https://www.aftership.com/docs/tracking/jh865r66gc6hi-get-trackings"},
		SourceNote:   "AfterShip Tracking docs advertise an OAS export, but M57 direct artifact probes were not usable as a durable unauthenticated catalog fetch; this overlay covers selected read-oriented tracking and courier endpoints.",
		Security:     security,
		SecurityReqs: securityReqs("aftershipAPIKey"),
		Schemas:      []string{"AfterShipObject", "AfterShipCollection", "AfterShipError"},
		OutputPath:   "catalog-openapi-cache/advisory-overlays/aftership-tracking-api-overlay.json",
		Paths: map[string]map[string]any{
			"/couriers":       {"get": op("listAfterShipCouriers", "Get couriers", nil, "", "#/components/schemas/AfterShipCollection", securityReqs("aftershipAPIKey"))},
			"/trackings":      {"get": op("listAfterShipTrackings", "Get trackings", params(query("limit", "Number of trackings per page."), query("cursor", "Pagination cursor."), query("tracking_numbers", "Comma-separated tracking numbers.")), "", "#/components/schemas/AfterShipCollection", securityReqs("aftershipAPIKey"))},
			"/trackings/{id}": {"get": op("getAfterShipTracking", "Get tracking by ID", params(path("id", "Tracking ID.")), "", "#/components/schemas/AfterShipObject", securityReqs("aftershipAPIKey"))},
		},
	}
}

func acrobatSignOverlay() overlaySpec {
	security := map[string]map[string]any{
		"adobeAcrobatSignOAuth2": {"type": "oauth2", "description": "Adobe Acrobat Sign OAuth 2.0 access token with operation-specific scopes.", "flows": map[string]any{
			"authorizationCode": map[string]any{
				"authorizationUrl": "https://{acrobat_sign_web_access_point}/public/oauth",
				"tokenUrl":         "https://{acrobat_sign_api_access_point}/oauth/token",
				"scopes": map[string]string{
					"agreement_read": "Read agreement metadata.",
				},
			},
		}},
	}
	agreementRead := securityReqsWithScopes("adobeAcrobatSignOAuth2", "agreement_read")
	return overlaySpec{
		ProviderID:   "adobe-acrobat-sign",
		Title:        "Adobe Acrobat Sign REST API Advisory Overlay",
		Description:  "Advisory OpenAPI overlay derived from official Adobe Acrobat Sign SDK JSON and REST documentation. This is not an official Adobe OpenAPI 2/3 document.",
		ServerURL:    "https://{acrobat_sign_api_host}/api/rest/v6",
		Sources:      []string{"https://developer.adobe.com/acrobat-sign/docs/overview/sdks/openapi", "https://github.com/adobe/acrobat-sign/tree/main/sdks/AcrobatSign_OpenAPI_SDK", "https://raw.githubusercontent.com/adobe/acrobat-sign/main/sdks/AcrobatSign_OpenAPI_SDK/json/agreements.json", "https://raw.githubusercontent.com/adobe/acrobat-sign/main/sdks/AcrobatSign_OpenAPI_SDK/json/baseUris.json"},
		SourceNote:   "Adobe Acrobat Sign publishes official Swagger 1.2-style SDK JSON files rather than directly importable OpenAPI 2/3; this overlay covers selected base URI and agreement read endpoints.",
		Security:     security,
		SecurityReqs: securityReqs("adobeAcrobatSignOAuth2"),
		Schemas:      []string{"AcrobatSignObject", "AcrobatSignCollection", "AcrobatSignError"},
		OutputPath:   "catalog-openapi-cache/advisory-overlays/adobe-acrobat-sign-api-overlay.json",
		Paths: map[string]map[string]any{
			"/agreements":               {"get": op("listAcrobatSignAgreements", "Get agreements", params(query("userId", "Optional user ID.")), "", "#/components/schemas/AcrobatSignCollection", agreementRead)},
			"/agreements/{agreementId}": {"get": op("getAcrobatSignAgreement", "Get agreement", params(path("agreementId", "Agreement ID.")), "", "#/components/schemas/AcrobatSignObject", agreementRead)},
			"/baseUris":                 {"get": op("getAcrobatSignBaseUris", "Get base URI", nil, "", "#/components/schemas/AcrobatSignObject", securityReqs("adobeAcrobatSignOAuth2"))},
		},
	}
}

func build(spec overlaySpec) map[string]any {
	return map[string]any{
		"openapi": "3.1.0",
		"info": map[string]any{
			"title":       spec.Title,
			"version":     "2026-05-21",
			"description": spec.Description,
		},
		"servers":  []map[string]any{{"url": spec.ServerURL}},
		"security": spec.SecurityReqs,
		"paths":    orderedMap(spec.Paths),
		"components": map[string]any{
			"securitySchemes": orderedMap(spec.Security),
			"schemas":         schemas(spec.Schemas),
		},
		"x-apitools-overlay": map[string]any{
			"provider_id":       spec.ProviderID,
			"official_openapi":  false,
			"derived_from_docs": true,
			"source_refs":       spec.Sources,
			"source_note":       spec.SourceNote,
		},
	}
}

func op(operationID, summary string, parameters []map[string]any, requestRef, responseRef string, security []map[string][]string) map[string]any {
	out := map[string]any{
		"operationId": operationID,
		"summary":     summary,
		"description": "Advisory operation derived from official human API documentation.",
		"security":    security,
		"responses": map[string]any{
			"200":     response(responseRef),
			"default": map[string]any{"description": "Provider error response."},
		},
	}
	if len(parameters) > 0 {
		out["parameters"] = parameters
	}
	if requestRef != "" {
		out["requestBody"] = map[string]any{
			"required": true,
			"content":  map[string]any{"application/json": map[string]any{"schema": map[string]any{"$ref": requestRef}}},
		}
	}
	return out
}

func response(ref string) map[string]any {
	out := map[string]any{"description": "Successful response."}
	if ref != "" {
		out["content"] = map[string]any{"application/json": map[string]any{"schema": map[string]any{"$ref": ref}}}
	}
	return out
}

func securityReqs(names ...string) []map[string][]string {
	requirement := map[string][]string{}
	for _, name := range names {
		requirement[name] = []string{}
	}
	return []map[string][]string{requirement}
}

func securityReqsWithScopes(name string, scopes ...string) []map[string][]string {
	return []map[string][]string{{name: scopes}}
}

func params(items ...map[string]any) []map[string]any { return items }

func path(name, description string) map[string]any { return parameter(name, "path", description, true) }

func query(name, description string) map[string]any {
	return parameter(name, "query", description, false)
}

func queryRequired(name, description string) map[string]any {
	return parameter(name, "query", description, true)
}

func parameter(name, in, description string, required bool) map[string]any {
	return map[string]any{
		"name":        name,
		"in":          in,
		"required":    required,
		"description": description,
		"schema":      map[string]any{"type": "string"},
	}
}

func schemas(names []string) map[string]map[string]any {
	sort.Strings(names)
	out := map[string]map[string]any{}
	for _, name := range names {
		out[name] = map[string]any{"type": "object", "additionalProperties": true}
	}
	return out
}

func orderedMap[V any](in map[string]V) map[string]V {
	keys := make([]string, 0, len(in))
	for key := range in {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := map[string]V{}
	for _, key := range keys {
		out[key] = in[key]
	}
	return out
}

func write(path string, doc map[string]any) {
	content, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		panic(err)
	}
	content = append(content, '\n')
	if err := os.WriteFile(path, content, 0o644); err != nil {
		panic(err)
	}
}
