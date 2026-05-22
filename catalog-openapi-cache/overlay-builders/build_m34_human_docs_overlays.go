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
		apiTemplateOverlay(),
		currentsOverlay(),
		demioOverlay(),
		mandrillOverlay(),
		metabaseOverlay(),
	} {
		write(spec.OutputPath, build(spec))
	}
}

func apiTemplateOverlay() overlaySpec {
	security := map[string]map[string]any{
		"apitemplateAPIKey": {"type": "apiKey", "in": "header", "name": "X-API-KEY", "description": "APITemplate.io API key carried in the X-API-KEY request header."},
	}
	return overlaySpec{
		ProviderID:  "apitemplate-io",
		Title:       "APITemplate.io API Advisory Overlay",
		Description: "Advisory OpenAPI overlay derived from official APITemplate.io human API documentation. This is not an official APITemplate.io OpenAPI document.",
		ServerURL:   "https://api.apitemplate.io/v1",
		Sources:     []string{"https://docs.apitemplate.io/api/index.html"},
		SourceNote:  "APITemplate.io publishes human API documentation but no recorded stable public official OpenAPI document; this overlay covers selected v1 template, PDF, and image generation endpoints.",
		Security:    security,
		Schemas:     []string{"APITemplateObject", "APITemplateCollection", "APITemplateError"},
		OutputPath:  "catalog-openapi-cache/advisory-overlays/apitemplate-io-api-overlay.json",
		Paths: map[string]map[string]any{
			"/create":         {"post": op("createAPITemplatePDF", "Create PDF or image from template", params(query("template_id", "Template ID.")), "#/components/schemas/APITemplateObject", "#/components/schemas/APITemplateObject", "apitemplateAPIKey")},
			"/create-image":   {"post": op("createAPITemplateImage", "Create image from template", params(query("template_id", "Template ID.")), "#/components/schemas/APITemplateObject", "#/components/schemas/APITemplateObject", "apitemplateAPIKey")},
			"/create-pdf":     {"post": op("createAPITemplatePDFDocument", "Create PDF document from template", params(query("template_id", "Template ID.")), "#/components/schemas/APITemplateObject", "#/components/schemas/APITemplateObject", "apitemplateAPIKey")},
			"/list-templates": {"get": op("listAPITemplateTemplates", "List templates", nil, "", "#/components/schemas/APITemplateCollection", "apitemplateAPIKey")},
		},
	}
}

func currentsOverlay() overlaySpec {
	security := map[string]map[string]any{
		"currentsBearer": {"type": "http", "scheme": "bearer", "bearerFormat": "Currents API key", "description": "Currents API key carried in the Authorization bearer header."},
	}
	return overlaySpec{
		ProviderID:  "currents",
		Title:       "Currents API Advisory Overlay",
		Description: "Advisory OpenAPI overlay derived from official Currents human API documentation. This is not an official Currents OpenAPI document.",
		ServerURL:   "https://api.currents.dev/v1",
		Sources:     []string{"https://docs.currents.dev/api", "https://docs.currents.dev/resources/api/authentication", "https://docs.currents.dev/api/resources"},
		SourceNote:  "Currents publishes human API documentation but no recorded stable public official OpenAPI document; this overlay covers selected projects, runs, tests, spec files, actions, signatures, instances, and webhooks endpoints.",
		Security:    security,
		Schemas:     []string{"CurrentsObject", "CurrentsCollection", "CurrentsError"},
		OutputPath:  "catalog-openapi-cache/advisory-overlays/currents-api-overlay.json",
		Paths: map[string]map[string]any{
			"/actions":                        {"get": op("listCurrentsActions", "List actions", nil, "", "#/components/schemas/CurrentsCollection", "currentsBearer"), "post": op("createCurrentsAction", "Create action", nil, "#/components/schemas/CurrentsObject", "#/components/schemas/CurrentsObject", "currentsBearer")},
			"/actions/{action_id}":            {"get": op("getCurrentsAction", "Get action", params(path("action_id", "Action ID.")), "", "#/components/schemas/CurrentsObject", "currentsBearer"), "put": op("updateCurrentsAction", "Update action", params(path("action_id", "Action ID.")), "#/components/schemas/CurrentsObject", "#/components/schemas/CurrentsObject", "currentsBearer"), "delete": op("deleteCurrentsAction", "Delete action", params(path("action_id", "Action ID.")), "", "#/components/schemas/CurrentsObject", "currentsBearer")},
			"/instances/{instance_id}":        {"get": op("getCurrentsInstance", "Get instance", params(path("instance_id", "Instance ID.")), "", "#/components/schemas/CurrentsObject", "currentsBearer")},
			"/projects":                       {"get": op("listCurrentsProjects", "List projects", nil, "", "#/components/schemas/CurrentsCollection", "currentsBearer")},
			"/projects/{project_id}":          {"get": op("getCurrentsProject", "Get project", params(path("project_id", "Project ID.")), "", "#/components/schemas/CurrentsObject", "currentsBearer")},
			"/projects/{project_id}/insights": {"get": op("getCurrentsProjectInsights", "Get project insights", params(path("project_id", "Project ID.")), "", "#/components/schemas/CurrentsObject", "currentsBearer")},
			"/projects/{project_id}/runs":     {"get": op("listCurrentsProjectRuns", "List project runs", params(path("project_id", "Project ID.")), "", "#/components/schemas/CurrentsCollection", "currentsBearer")},
			"/runs/{run_id}":                  {"get": op("getCurrentsRun", "Get run", params(path("run_id", "Run ID.")), "", "#/components/schemas/CurrentsObject", "currentsBearer"), "delete": op("deleteCurrentsRun", "Delete run", params(path("run_id", "Run ID.")), "", "#/components/schemas/CurrentsObject", "currentsBearer")},
			"/runs/{run_id}/cancel":           {"put": op("cancelCurrentsRun", "Cancel run", params(path("run_id", "Run ID.")), "#/components/schemas/CurrentsObject", "#/components/schemas/CurrentsObject", "currentsBearer")},
			"/signature/test":                 {"post": op("createCurrentsTestSignature", "Create test signature", nil, "#/components/schemas/CurrentsObject", "#/components/schemas/CurrentsObject", "currentsBearer")},
			"/spec-files/{project_id}":        {"get": op("listCurrentsSpecFiles", "List spec files", params(path("project_id", "Project ID.")), "", "#/components/schemas/CurrentsCollection", "currentsBearer")},
			"/test-results/{signature}":       {"get": op("listCurrentsTestResults", "List test results", params(path("signature", "Test signature.")), "", "#/components/schemas/CurrentsCollection", "currentsBearer")},
			"/tests/{project_id}":             {"get": op("listCurrentsTests", "List tests", params(path("project_id", "Project ID.")), "", "#/components/schemas/CurrentsCollection", "currentsBearer")},
			"/webhooks":                       {"get": op("listCurrentsWebhooks", "List webhooks", nil, "", "#/components/schemas/CurrentsCollection", "currentsBearer"), "post": op("createCurrentsWebhook", "Create webhook", nil, "#/components/schemas/CurrentsObject", "#/components/schemas/CurrentsObject", "currentsBearer")},
			"/webhooks/{webhook_id}":          {"get": op("getCurrentsWebhook", "Get webhook", params(path("webhook_id", "Webhook ID.")), "", "#/components/schemas/CurrentsObject", "currentsBearer"), "put": op("updateCurrentsWebhook", "Update webhook", params(path("webhook_id", "Webhook ID.")), "#/components/schemas/CurrentsObject", "#/components/schemas/CurrentsObject", "currentsBearer"), "delete": op("deleteCurrentsWebhook", "Delete webhook", params(path("webhook_id", "Webhook ID.")), "", "#/components/schemas/CurrentsObject", "currentsBearer")},
		},
	}
}

func demioOverlay() overlaySpec {
	security := map[string]map[string]any{
		"demioAPIKey":    {"type": "apiKey", "in": "header", "name": "Api-Key", "description": "Demio API key carried in the Api-Key request header."},
		"demioAPISecret": {"type": "apiKey", "in": "header", "name": "Api-Secret", "description": "Demio API secret carried in the Api-Secret request header."},
	}
	return overlaySpec{
		ProviderID:  "demio",
		Title:       "Demio API Advisory Overlay",
		Description: "Advisory OpenAPI overlay derived from official Demio human API documentation. This is not an official Demio OpenAPI document.",
		ServerURL:   "https://my.demio.com/api/v1",
		Sources:     []string{"https://publicdemioapi.docs.apiary.io/", "https://help.demio.com/en/articles/4544025-api-limitations"},
		SourceNote:  "Demio publishes human API documentation but no recorded stable public official OpenAPI document; this overlay covers selected event, registration, and report endpoints.",
		Security:    security,
		Schemas:     []string{"DemioObject", "DemioCollection", "DemioError"},
		OutputPath:  "catalog-openapi-cache/advisory-overlays/demio-api-overlay.json",
		Paths: map[string]map[string]any{
			"/event/{event_id}":        {"get": op("getDemioEvent", "Get event", params(path("event_id", "Event ID.")), "", "#/components/schemas/DemioObject", "demioAPIKey", "demioAPISecret")},
			"/event/register":          {"put": op("registerDemioParticipant", "Register participant", nil, "#/components/schemas/DemioObject", "#/components/schemas/DemioObject", "demioAPIKey", "demioAPISecret")},
			"/event/{event_id}/report": {"get": op("getDemioEventReport", "Get event report", params(path("event_id", "Event ID.")), "", "#/components/schemas/DemioObject", "demioAPIKey", "demioAPISecret")},
			"/events":                  {"get": op("listDemioEvents", "List events", params(query("type", "Event type filter.")), "", "#/components/schemas/DemioCollection", "demioAPIKey", "demioAPISecret")},
		},
	}
}

func mandrillOverlay() overlaySpec {
	security := map[string]map[string]any{
		"mandrillAPIKey": {"type": "apiKey", "in": "query", "name": "key", "description": "Mailchimp Transactional API key carried as the key field in each JSON or form request payload; query placement is advisory only because OpenAPI security schemes cannot model JSON-body credentials."},
	}
	return overlaySpec{
		ProviderID:  "mandrill",
		Title:       "Mailchimp Transactional API Advisory Overlay",
		Description: "Advisory OpenAPI overlay derived from official Mailchimp Transactional human API documentation. This is not an official Mailchimp OpenAPI document.",
		ServerURL:   "https://mandrillapp.com/api/1.0",
		Sources:     []string{"https://mailchimp.com/developer/transactional/api/", "https://mailchimp.com/developer/transactional/api/templates/update-template/", "https://mailchimp.com/developer/transactional/api/webhooks/list-webhooks/", "https://mailchimp.com/developer/transactional/docs/fundamentals/"},
		SourceNote:  "Mailchimp publishes Transactional API human documentation but no recorded stable public official OpenAPI document; this overlay covers selected messages, templates, users, and webhooks endpoints.",
		Security:    security,
		Schemas:     []string{"MandrillObject", "MandrillCollection", "MandrillError"},
		OutputPath:  "catalog-openapi-cache/advisory-overlays/mandrill-transactional-api-overlay.json",
		Paths: map[string]map[string]any{
			"/messages/send.json":          {"post": op("sendMandrillMessage", "Send message", nil, "#/components/schemas/MandrillObject", "#/components/schemas/MandrillCollection", "mandrillAPIKey")},
			"/messages/send-template.json": {"post": op("sendMandrillTemplateMessage", "Send template message", nil, "#/components/schemas/MandrillObject", "#/components/schemas/MandrillCollection", "mandrillAPIKey")},
			"/templates/add.json":          {"post": op("addMandrillTemplate", "Add template", nil, "#/components/schemas/MandrillObject", "#/components/schemas/MandrillObject", "mandrillAPIKey")},
			"/templates/delete.json":       {"post": op("deleteMandrillTemplate", "Delete template", nil, "#/components/schemas/MandrillObject", "#/components/schemas/MandrillObject", "mandrillAPIKey")},
			"/templates/info.json":         {"post": op("getMandrillTemplate", "Get template", nil, "#/components/schemas/MandrillObject", "#/components/schemas/MandrillObject", "mandrillAPIKey")},
			"/templates/list.json":         {"post": op("listMandrillTemplates", "List templates", nil, "#/components/schemas/MandrillObject", "#/components/schemas/MandrillCollection", "mandrillAPIKey")},
			"/templates/publish.json":      {"post": op("publishMandrillTemplate", "Publish template content", nil, "#/components/schemas/MandrillObject", "#/components/schemas/MandrillObject", "mandrillAPIKey")},
			"/templates/render.json":       {"post": op("renderMandrillTemplate", "Render HTML template", nil, "#/components/schemas/MandrillObject", "#/components/schemas/MandrillObject", "mandrillAPIKey")},
			"/templates/update.json":       {"post": op("updateMandrillTemplate", "Update template", nil, "#/components/schemas/MandrillObject", "#/components/schemas/MandrillObject", "mandrillAPIKey")},
			"/users/info.json":             {"post": op("getMandrillUserInfo", "Get user info", nil, "#/components/schemas/MandrillObject", "#/components/schemas/MandrillObject", "mandrillAPIKey")},
			"/users/ping.json":             {"post": op("pingMandrill", "Ping API", nil, "#/components/schemas/MandrillObject", "#/components/schemas/MandrillObject", "mandrillAPIKey")},
			"/webhooks/add.json":           {"post": op("addMandrillWebhook", "Add webhook", nil, "#/components/schemas/MandrillObject", "#/components/schemas/MandrillObject", "mandrillAPIKey")},
			"/webhooks/delete.json":        {"post": op("deleteMandrillWebhook", "Delete webhook", nil, "#/components/schemas/MandrillObject", "#/components/schemas/MandrillObject", "mandrillAPIKey")},
			"/webhooks/info.json":          {"post": op("getMandrillWebhook", "Get webhook info", nil, "#/components/schemas/MandrillObject", "#/components/schemas/MandrillObject", "mandrillAPIKey")},
			"/webhooks/list.json":          {"post": op("listMandrillWebhooks", "List webhooks", nil, "#/components/schemas/MandrillObject", "#/components/schemas/MandrillCollection", "mandrillAPIKey")},
			"/webhooks/update.json":        {"post": op("updateMandrillWebhook", "Update webhook", nil, "#/components/schemas/MandrillObject", "#/components/schemas/MandrillObject", "mandrillAPIKey")},
		},
	}
}

func metabaseOverlay() overlaySpec {
	security := map[string]map[string]any{
		"metabaseSession": {"type": "apiKey", "in": "header", "name": "X-Metabase-Session", "description": "Metabase session token carried in the X-Metabase-Session request header after a metadata-only /api/session exchange."},
	}
	return overlaySpec{
		ProviderID:  "metabase",
		Title:       "Metabase API Advisory Overlay",
		Description: "Advisory OpenAPI overlay derived from official Metabase human API documentation. This is not an official Metabase OpenAPI document.",
		ServerURL:   "https://{metabase_host}",
		Sources:     []string{"https://www.metabase.com/docs/latest/api", "https://www.metabase.com/learn/metabase-basics/administration/administration-and-operation/metabase-api"},
		SourceNote:  "Metabase publishes human API documentation and instance-specific live OpenAPI docs but no recorded stable provider-hosted OpenAPI artifact; this overlay covers selected questions, databases, alerts, metrics, and session endpoints.",
		Security:    security,
		Schemas:     []string{"MetabaseObject", "MetabaseCollection", "MetabaseError"},
		OutputPath:  "catalog-openapi-cache/advisory-overlays/metabase-api-overlay.json",
		Paths: map[string]map[string]any{
			"/api/alert":            {"get": op("listMetabaseAlerts", "List alerts", nil, "", "#/components/schemas/MetabaseCollection", "metabaseSession")},
			"/api/card":             {"get": op("listMetabaseQuestions", "List questions", nil, "", "#/components/schemas/MetabaseCollection", "metabaseSession"), "post": op("createMetabaseQuestion", "Create question", nil, "#/components/schemas/MetabaseObject", "#/components/schemas/MetabaseObject", "metabaseSession")},
			"/api/card/{card_id}":   {"get": op("getMetabaseQuestion", "Get question", params(path("card_id", "Question/card ID.")), "", "#/components/schemas/MetabaseObject", "metabaseSession"), "put": op("updateMetabaseQuestion", "Update question", params(path("card_id", "Question/card ID.")), "#/components/schemas/MetabaseObject", "#/components/schemas/MetabaseObject", "metabaseSession")},
			"/api/database":         {"get": op("listMetabaseDatabases", "List databases", nil, "", "#/components/schemas/MetabaseCollection", "metabaseSession")},
			"/api/database/{db_id}": {"get": op("getMetabaseDatabase", "Get database", params(path("db_id", "Database ID.")), "", "#/components/schemas/MetabaseObject", "metabaseSession")},
			"/api/metric":           {"get": op("listMetabaseMetrics", "List metrics", nil, "", "#/components/schemas/MetabaseCollection", "metabaseSession")},
			"/api/session":          {"post": op("createMetabaseSession", "Create session", nil, "#/components/schemas/MetabaseObject", "#/components/schemas/MetabaseObject")},
			"/api/user/current":     {"get": op("getMetabaseCurrentUser", "Get current user", nil, "", "#/components/schemas/MetabaseObject", "metabaseSession")},
		},
	}
}

func build(spec overlaySpec) map[string]any {
	return map[string]any{
		"openapi": "3.1.0",
		"info": map[string]any{
			"title":       spec.Title,
			"version":     "2026-05-20",
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
