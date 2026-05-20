//go:build ignore

package main

import (
	"encoding/json"
	"os"
	"sort"
)

type overlaySpec struct {
	ProviderID  string
	Title       string
	Description string
	ServerURL   string
	Sources     []string
	SourceNote  string
	Security    map[string]map[string]any
	Schemas     []string
	Paths       map[string]map[string]any
	OutputPath  string
}

func main() {
	for _, spec := range []overlaySpec{
		humanticAIOverlay(),
		hunterOverlay(),
		lingvaNexOverlay(),
		mistralAIOverlay(),
		mindeeOverlay(),
		phantombusterOverlay(),
		upleadOverlay(),
		dropcontactOverlay(),
	} {
		write(spec.OutputPath, build(spec))
	}
}

func humanticAIOverlay() overlaySpec {
	security := map[string]map[string]any{
		"humanticAIAPIKey": {"type": "apiKey", "in": "query", "name": "apikey", "description": "Humantic AI API key carried in the apikey query parameter."},
	}
	return overlaySpec{
		ProviderID:  "humantic-ai",
		Title:       "Humantic AI API Advisory Overlay",
		Description: "Advisory OpenAPI overlay derived from official Humantic AI human API documentation. This is not an official Humantic AI OpenAPI document.",
		ServerURL:   "https://api.humantic.ai/v1",
		Sources:     []string{"https://api.humantic.ai/"},
		SourceNote:  "Humantic AI publishes human API documentation but no recorded stable public official OpenAPI document; this overlay covers selected user-profile create, retrieve, and update endpoints.",
		Security:    security,
		Schemas:     []string{"HumanticAIObject", "HumanticAICollection", "HumanticAIError"},
		OutputPath:  "catalog-openapi-cache/advisory-overlays/humantic-ai-api-overlay.json",
		Paths: map[string]map[string]any{
			"/user-profile/create": {"get": op("createHumanticAIProfile", "Create profile from profile URL or email", params(query("userid", "LinkedIn URL, email, or unique profile identifier.")), "", "#/components/schemas/HumanticAIObject", "humanticAIAPIKey"), "post": op("createHumanticAIProfileWithResumeOrText", "Create profile from resume or text", params(query("userid", "LinkedIn URL, email, or unique profile identifier.")), "#/components/schemas/HumanticAIObject", "#/components/schemas/HumanticAIObject", "humanticAIAPIKey")},
			"/user-profile":        {"get": op("getHumanticAIProfile", "Get profile", params(query("userid", "Profile identifier."), query("persona", "Optional comma-delimited persona filter.")), "", "#/components/schemas/HumanticAIObject", "humanticAIAPIKey")},
		},
	}
}

func hunterOverlay() overlaySpec {
	security := map[string]map[string]any{
		"hunterAPIKey": {"type": "apiKey", "in": "query", "name": "api_key", "description": "Hunter API key carried in the api_key query parameter."},
	}
	return overlaySpec{
		ProviderID:  "hunter",
		Title:       "Hunter API Advisory Overlay",
		Description: "Advisory OpenAPI overlay derived from official Hunter human API documentation. This is not an official Hunter OpenAPI document.",
		ServerURL:   "https://api.hunter.io/v2",
		Sources:     []string{"https://hunter.io/api-documentation"},
		SourceNote:  "Hunter publishes human API documentation but no recorded stable public official OpenAPI document; this overlay covers selected domain search, email finder, email verifier, leads, campaigns, and account endpoints.",
		Security:    security,
		Schemas:     []string{"HunterObject", "HunterCollection", "HunterError"},
		OutputPath:  "catalog-openapi-cache/advisory-overlays/hunter-api-overlay.json",
		Paths: map[string]map[string]any{
			"/domain-search":  {"get": op("searchHunterDomain", "Search domain emails", params(query("domain", "Domain name."), query("type", "Email type filter."), query("seniority", "Seniority filter."), query("department", "Department filter."), query("limit", "Maximum results."), query("offset", "Pagination offset.")), "", "#/components/schemas/HunterCollection", "hunterAPIKey")},
			"/email-finder":   {"get": op("findHunterEmail", "Find email", params(query("first_name", "First name."), query("last_name", "Last name."), query("domain", "Domain name.")), "", "#/components/schemas/HunterObject", "hunterAPIKey")},
			"/email-verifier": {"get": op("verifyHunterEmail", "Verify email", params(query("email", "Email address.")), "", "#/components/schemas/HunterObject", "hunterAPIKey")},
			"/leads":          {"get": op("listHunterLeads", "List leads", params(query("limit", "Maximum results."), query("offset", "Pagination offset.")), "", "#/components/schemas/HunterCollection", "hunterAPIKey"), "post": op("createHunterLead", "Create lead", nil, "#/components/schemas/HunterObject", "#/components/schemas/HunterObject", "hunterAPIKey")},
			"/campaigns":      {"get": op("listHunterCampaigns", "List campaigns", nil, "", "#/components/schemas/HunterCollection", "hunterAPIKey")},
			"/account":        {"get": op("getHunterAccount", "Get account", nil, "", "#/components/schemas/HunterObject", "hunterAPIKey")},
		},
	}
}

func lingvaNexOverlay() overlaySpec {
	security := map[string]map[string]any{
		"lingvaNexBearer": {"type": "http", "scheme": "bearer", "bearerFormat": "LingvaNex API key", "description": "LingvaNex API key carried in the Authorization bearer header."},
	}
	return overlaySpec{
		ProviderID:  "lingvanex",
		Title:       "LingvaNex API Advisory Overlay",
		Description: "Advisory OpenAPI overlay derived from official LingvaNex human API documentation. This is not an official LingvaNex OpenAPI document.",
		ServerURL:   "https://api-b2b.backenster.com/b1/api/v3",
		Sources:     []string{"https://docs.lingvanex.com/reference/user-guide", "https://github.com/lingvanex-mt/python-translation-api"},
		SourceNote:  "LingvaNex publishes human API documentation but no recorded stable public official OpenAPI document; this overlay covers selected translation, language detection, and language-list endpoints.",
		Security:    security,
		Schemas:     []string{"LingvaNexObject", "LingvaNexCollection", "LingvaNexError"},
		OutputPath:  "catalog-openapi-cache/advisory-overlays/lingvanex-api-overlay.json",
		Paths: map[string]map[string]any{
			"/translate":          {"post": op("translateLingvaNexText", "Translate text", nil, "#/components/schemas/LingvaNexObject", "#/components/schemas/LingvaNexObject", "lingvaNexBearer")},
			"/detect":             {"post": op("detectLingvaNexLanguage", "Detect language", nil, "#/components/schemas/LingvaNexObject", "#/components/schemas/LingvaNexObject", "lingvaNexBearer")},
			"/getLanguages":       {"get": op("listLingvaNexLanguages", "List languages", nil, "", "#/components/schemas/LingvaNexCollection", "lingvaNexBearer")},
			"/getLanguageByCode":  {"get": op("getLingvaNexLanguageByCode", "Get language by code", params(query("code", "Language code.")), "", "#/components/schemas/LingvaNexObject", "lingvaNexBearer")},
			"/getDictionary":      {"post": op("getLingvaNexDictionary", "Get dictionary entries", nil, "#/components/schemas/LingvaNexObject", "#/components/schemas/LingvaNexObject", "lingvaNexBearer")},
			"/translate_file_url": {"post": op("translateLingvaNexFileURL", "Translate file by URL", nil, "#/components/schemas/LingvaNexObject", "#/components/schemas/LingvaNexObject", "lingvaNexBearer")},
		},
	}
}

func mistralAIOverlay() overlaySpec {
	security := map[string]map[string]any{
		"mistralBearer": {"type": "http", "scheme": "bearer", "bearerFormat": "Mistral AI API key", "description": "Mistral AI API key carried in the Authorization bearer header."},
	}
	return overlaySpec{
		ProviderID:  "mistral-ai",
		Title:       "Mistral AI API Advisory Overlay",
		Description: "Advisory OpenAPI overlay derived from official Mistral AI API specification documentation. This is not an official Mistral AI OpenAPI document.",
		ServerURL:   "https://api.mistral.ai",
		Sources:     []string{"https://docs.mistral.ai/api", "https://docs.mistral.ai/admin/security-access/api-keys"},
		SourceNote:  "Mistral AI publishes rendered API specifications but no recorded stable public downloadable OpenAPI document; this overlay covers selected chat, embeddings, OCR, files, batch jobs, agents, and models endpoints.",
		Security:    security,
		Schemas:     []string{"MistralAIObject", "MistralAICollection", "MistralAIError"},
		OutputPath:  "catalog-openapi-cache/advisory-overlays/mistral-ai-api-overlay.json",
		Paths: map[string]map[string]any{
			"/v1/chat/completions": {"post": op("createMistralAIChatCompletion", "Create chat completion", nil, "#/components/schemas/MistralAIObject", "#/components/schemas/MistralAIObject", "mistralBearer")},
			"/v1/embeddings":       {"post": op("createMistralAIEmbeddings", "Create embeddings", nil, "#/components/schemas/MistralAIObject", "#/components/schemas/MistralAIObject", "mistralBearer")},
			"/v1/ocr":              {"post": op("createMistralAIOCR", "Run OCR", nil, "#/components/schemas/MistralAIObject", "#/components/schemas/MistralAIObject", "mistralBearer")},
			"/v1/files":            {"get": op("listMistralAIFiles", "List files", nil, "", "#/components/schemas/MistralAICollection", "mistralBearer"), "post": op("uploadMistralAIFile", "Upload file", nil, "#/components/schemas/MistralAIObject", "#/components/schemas/MistralAIObject", "mistralBearer")},
			"/v1/files/{file_id}":  {"get": op("getMistralAIFile", "Get file", params(path("file_id", "File ID.")), "", "#/components/schemas/MistralAIObject", "mistralBearer"), "delete": op("deleteMistralAIFile", "Delete file", params(path("file_id", "File ID.")), "", "#/components/schemas/MistralAIObject", "mistralBearer")},
			"/v1/batch/jobs":       {"get": op("listMistralAIBatchJobs", "List batch jobs", nil, "", "#/components/schemas/MistralAICollection", "mistralBearer"), "post": op("createMistralAIBatchJob", "Create batch job", nil, "#/components/schemas/MistralAIObject", "#/components/schemas/MistralAIObject", "mistralBearer")},
			"/v1/models":           {"get": op("listMistralAIModels", "List models", nil, "", "#/components/schemas/MistralAICollection", "mistralBearer")},
			"/v1/agents":           {"get": op("listMistralAIAgents", "List agents", nil, "", "#/components/schemas/MistralAICollection", "mistralBearer")},
		},
	}
}

func mindeeOverlay() overlaySpec {
	security := map[string]map[string]any{
		"mindeeV1Token":        {"type": "apiKey", "in": "header", "name": "Authorization", "description": "Mindee v1 product API key carried in the Authorization header using Token syntax."},
		"mindeeInferuserToken": {"type": "apiKey", "in": "header", "name": "X-Inferuser-Token", "description": "Mindee legacy product API key carried in the X-Inferuser-Token header."},
	}
	return overlaySpec{
		ProviderID:  "mindee",
		Title:       "Mindee API Advisory Overlay",
		Description: "Advisory OpenAPI overlay derived from official Mindee human API documentation. This is not an official Mindee OpenAPI document.",
		ServerURL:   "https://api.mindee.net",
		Sources:     []string{"https://docs.mindee.com/integrations/api-overview", "https://docs.mindee.com/integrations/api-keys"},
		SourceNote:  "Mindee publishes human API documentation but no recorded stable public official OpenAPI document; this overlay covers selected receipt, invoice, and model inference endpoints.",
		Security:    security,
		Schemas:     []string{"MindeeObject", "MindeeCollection", "MindeeError"},
		OutputPath:  "catalog-openapi-cache/advisory-overlays/mindee-api-overlay.json",
		Paths: map[string]map[string]any{
			"/v1/products/mindee/expense_receipts/v4/predict": {"post": op("predictMindeeReceiptV4", "Predict receipt", nil, "#/components/schemas/MindeeObject", "#/components/schemas/MindeeObject", "mindeeV1Token")},
			"/v1/products/mindee/invoices/v4/predict":         {"post": op("predictMindeeInvoiceV4", "Predict invoice", nil, "#/components/schemas/MindeeObject", "#/components/schemas/MindeeObject", "mindeeV1Token")},
			"/products/expense_receipts/v2/predict":           {"post": op("predictMindeeLegacyReceipt", "Predict legacy receipt", nil, "#/components/schemas/MindeeObject", "#/components/schemas/MindeeObject", "mindeeInferuserToken")},
			"/products/invoices/v2/predict":                   {"post": op("predictMindeeLegacyInvoice", "Predict legacy invoice", nil, "#/components/schemas/MindeeObject", "#/components/schemas/MindeeObject", "mindeeInferuserToken")},
			"/v1/products/{owner}/{model}/v1/predict":         {"post": op("predictMindeeCustomModel", "Predict custom model", params(path("owner", "Model owner."), path("model", "Model identifier.")), "#/components/schemas/MindeeObject", "#/components/schemas/MindeeObject", "mindeeV1Token")},
		},
	}
}

func phantombusterOverlay() overlaySpec {
	security := map[string]map[string]any{
		"phantombusterAPIKey": {"type": "apiKey", "in": "header", "name": "X-Phantombuster-Key", "description": "Phantombuster API key carried in the X-Phantombuster-Key header."},
	}
	return overlaySpec{
		ProviderID:  "phantombuster",
		Title:       "Phantombuster API Advisory Overlay",
		Description: "Advisory OpenAPI overlay derived from official Phantombuster human API documentation. This is not an official Phantombuster OpenAPI document.",
		ServerURL:   "https://api.phantombuster.com/api/v2",
		Sources:     []string{"https://hub.phantombuster.com/docs/api", "https://hub.phantombuster.com/reference", "https://hub.phantombuster.com/llms.txt"},
		SourceNote:  "Phantombuster publishes human API reference documentation but no recorded stable public official OpenAPI document; this overlay covers selected agents, containers, scripts, and organization endpoints.",
		Security:    security,
		Schemas:     []string{"PhantombusterObject", "PhantombusterCollection", "PhantombusterError"},
		OutputPath:  "catalog-openapi-cache/advisory-overlays/phantombuster-api-overlay.json",
		Paths: map[string]map[string]any{
			"/agents/fetch-all":     {"get": op("listPhantombusterAgents", "List agents", nil, "", "#/components/schemas/PhantombusterCollection", "phantombusterAPIKey")},
			"/agents/fetch":         {"get": op("getPhantombusterAgent", "Get agent", params(query("id", "Agent ID.")), "", "#/components/schemas/PhantombusterObject", "phantombusterAPIKey")},
			"/agents/launch":        {"post": op("launchPhantombusterAgent", "Launch agent", nil, "#/components/schemas/PhantombusterObject", "#/components/schemas/PhantombusterObject", "phantombusterAPIKey")},
			"/agents/fetch-output":  {"get": op("getPhantombusterAgentOutput", "Get agent output", params(query("id", "Agent ID.")), "", "#/components/schemas/PhantombusterObject", "phantombusterAPIKey")},
			"/agents/delete":        {"post": op("deletePhantombusterAgent", "Delete agent", nil, "#/components/schemas/PhantombusterObject", "#/components/schemas/PhantombusterObject", "phantombusterAPIKey")},
			"/containers/fetch-all": {"get": op("listPhantombusterContainers", "List containers", params(query("agentId", "Agent ID.")), "", "#/components/schemas/PhantombusterCollection", "phantombusterAPIKey")},
			"/containers/fetch":     {"get": op("getPhantombusterContainer", "Get container", params(query("id", "Container ID.")), "", "#/components/schemas/PhantombusterObject", "phantombusterAPIKey")},
			"/scripts/fetch-all":    {"get": op("listPhantombusterScripts", "List scripts", nil, "", "#/components/schemas/PhantombusterCollection", "phantombusterAPIKey")},
			"/orgs/fetch":           {"get": op("getPhantombusterOrganization", "Get organization", nil, "", "#/components/schemas/PhantombusterObject", "phantombusterAPIKey")},
		},
	}
}

func upleadOverlay() overlaySpec {
	security := map[string]map[string]any{
		"upleadAPIKey": {"type": "apiKey", "in": "header", "name": "Authorization", "description": "UpLead API key carried in the Authorization header."},
	}
	return overlaySpec{
		ProviderID:  "uplead",
		Title:       "UpLead API Advisory Overlay",
		Description: "Advisory OpenAPI overlay derived from official UpLead human API documentation. This is not an official UpLead OpenAPI document.",
		ServerURL:   "https://api.uplead.com/v2",
		Sources:     []string{"https://docs.uplead.com/"},
		SourceNote:  "UpLead publishes human API documentation but no recorded stable public official OpenAPI document; this overlay covers selected person and company enrichment endpoints.",
		Security:    security,
		Schemas:     []string{"UpLeadObject", "UpLeadCollection", "UpLeadError"},
		OutputPath:  "catalog-openapi-cache/advisory-overlays/uplead-api-overlay.json",
		Paths: map[string]map[string]any{
			"/person-search":  {"get": op("searchUpLeadPerson", "Search person", params(query("email", "Email address."), query("first_name", "First name."), query("last_name", "Last name."), query("domain", "Domain name.")), "", "#/components/schemas/UpLeadObject", "upleadAPIKey")},
			"/company-search": {"get": op("searchUpLeadCompany", "Search company", params(query("domain", "Company domain."), query("company", "Company name.")), "", "#/components/schemas/UpLeadObject", "upleadAPIKey")},
			"/credits":        {"get": op("getUpLeadCredits", "Get credits", nil, "", "#/components/schemas/UpLeadObject", "upleadAPIKey")},
		},
	}
}

func dropcontactOverlay() overlaySpec {
	security := map[string]map[string]any{
		"dropcontactAccessToken": {"type": "apiKey", "in": "header", "name": "X-Access-Token", "description": "Dropcontact API key carried in the X-Access-Token header."},
	}
	return overlaySpec{
		ProviderID:  "dropcontact",
		Title:       "Dropcontact API Advisory Overlay",
		Description: "Advisory OpenAPI overlay derived from official Dropcontact human API documentation. This is not an official Dropcontact OpenAPI document.",
		ServerURL:   "https://api.dropcontact.io",
		Sources:     []string{"https://developer.dropcontact.com/"},
		SourceNote:  "Dropcontact publishes human API documentation but no recorded stable public official OpenAPI document; this overlay covers selected batch enrichment and request retrieval endpoints.",
		Security:    security,
		Schemas:     []string{"DropcontactObject", "DropcontactCollection", "DropcontactError"},
		OutputPath:  "catalog-openapi-cache/advisory-overlays/dropcontact-api-overlay.json",
		Paths: map[string]map[string]any{
			"/batch":              {"post": op("createDropcontactBatch", "Create enrichment batch", nil, "#/components/schemas/DropcontactObject", "#/components/schemas/DropcontactObject", "dropcontactAccessToken")},
			"/batch/{request_id}": {"get": op("getDropcontactBatch", "Get enrichment batch", params(path("request_id", "Batch request ID.")), "", "#/components/schemas/DropcontactObject", "dropcontactAccessToken")},
		},
	}
}

func build(spec overlaySpec) map[string]any {
	return map[string]any{
		"openapi": "3.0.3",
		"info": map[string]any{
			"title":       spec.Title,
			"version":     "2026-05-19",
			"description": spec.Description,
		},
		"servers": []map[string]any{{"url": spec.ServerURL}},
		"paths":   orderedMap(spec.Paths),
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

func op(operationID, summary string, parameters []map[string]any, requestRef, responseRef string, securityNames ...string) map[string]any {
	out := map[string]any{
		"operationId": operationID,
		"summary":     summary,
		"description": "Advisory operation derived from official human API documentation.",
		"responses": map[string]any{
			"200":     response(responseRef),
			"default": map[string]any{"description": "Provider error response."},
		},
	}
	if len(securityNames) > 0 {
		requirement := map[string][]string{}
		for _, name := range securityNames {
			if name != "" {
				requirement[name] = []string{}
			}
		}
		if len(requirement) > 0 {
			out["security"] = []map[string][]string{requirement}
		}
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

func params(items ...map[string]any) []map[string]any { return items }

func path(name, description string) map[string]any { return parameter(name, "path", description, true) }

func query(name, description string) map[string]any {
	return parameter(name, "query", description, false)
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
