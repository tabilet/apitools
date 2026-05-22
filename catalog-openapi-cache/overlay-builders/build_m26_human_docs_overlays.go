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
		jotFormOverlay(),
		formIOOverlay(),
		formstackOverlay(),
		surveyMonkeyOverlay(),
		bubbleOverlay(),
		cockpitOverlay(),
		stackbyOverlay(),
		fileMakerOverlay(),
	} {
		write(spec.OutputPath, build(spec))
	}
}

func jotFormOverlay() overlaySpec {
	security := map[string]map[string]any{
		"jotformAPIKey": {"type": "apiKey", "in": "header", "name": "APIKEY", "description": "JotForm API key carried in the APIKEY header."},
	}
	return overlaySpec{
		ProviderID:  "jotform",
		Title:       "JotForm API Advisory Overlay",
		Description: "Advisory OpenAPI overlay derived from official JotForm human documentation. This is not an official JotForm OpenAPI document.",
		ServerURL:   "https://api.jotform.com",
		Sources:     []string{"https://api.jotform.com/docs/", "https://api.jotform.com/"},
		SourceNote:  "JotForm publishes human API documentation but no recorded official OpenAPI document; this overlay covers selected form, submission, question, report, and folder endpoints.",
		Security:    security,
		Schemas:     []string{"JotFormObject", "JotFormCollection", "JotFormError"},
		OutputPath:  "catalog-openapi-cache/advisory-overlays/jotform-api-overlay.json",
		Paths: map[string]map[string]any{
			"/user":                       {"get": op("getJotFormUser", "Get user account details", nil, "", "#/components/schemas/JotFormObject", "jotformAPIKey")},
			"/user/forms":                 {"get": op("listJotFormForms", "List user forms", params(query("limit", "Maximum number of forms."), query("offset", "Pagination offset.")), "", "#/components/schemas/JotFormCollection", "jotformAPIKey")},
			"/form/{form_id}":             {"get": op("getJotFormForm", "Get form", params(path("form_id", "JotForm form ID.")), "", "#/components/schemas/JotFormObject", "jotformAPIKey")},
			"/form/{form_id}/submissions": {"get": op("listJotFormFormSubmissions", "List form submissions", params(path("form_id", "JotForm form ID."), query("limit", "Maximum number of submissions."), query("offset", "Pagination offset.")), "", "#/components/schemas/JotFormCollection", "jotformAPIKey")},
			"/form/{form_id}/questions":   {"get": op("listJotFormFormQuestions", "List form questions", params(path("form_id", "JotForm form ID.")), "", "#/components/schemas/JotFormCollection", "jotformAPIKey")},
			"/form/{form_id}/reports": {
				"get":  op("listJotFormFormReports", "List form reports", params(path("form_id", "JotForm form ID.")), "", "#/components/schemas/JotFormCollection", "jotformAPIKey"),
				"post": op("createJotFormFormReport", "Create form report", params(path("form_id", "JotForm form ID.")), "#/components/schemas/JotFormObject", "#/components/schemas/JotFormObject", "jotformAPIKey"),
			},
			"/form/{form_id}/webhooks":    {"get": op("listJotFormFormWebhooks", "List form webhooks", params(path("form_id", "JotForm form ID.")), "", "#/components/schemas/JotFormCollection", "jotformAPIKey"), "post": op("createJotFormFormWebhook", "Create form webhook", params(path("form_id", "JotForm form ID.")), "", "#/components/schemas/JotFormObject", "jotformAPIKey")},
			"/folder":                     {"post": op("createJotFormFolder", "Create folder", nil, "#/components/schemas/JotFormObject", "#/components/schemas/JotFormObject", "jotformAPIKey")},
			"/folder/{folder_id}":         {"get": op("getJotFormFolder", "Get folder", params(path("folder_id", "JotForm folder ID.")), "", "#/components/schemas/JotFormObject", "jotformAPIKey"), "put": op("updateJotFormFolder", "Update folder", params(path("folder_id", "JotForm folder ID.")), "#/components/schemas/JotFormObject", "#/components/schemas/JotFormObject", "jotformAPIKey"), "delete": op("deleteJotFormFolder", "Delete folder", params(path("folder_id", "JotForm folder ID.")), "", "#/components/schemas/JotFormObject", "jotformAPIKey")},
			"/report/{report_id}":         {"get": op("getJotFormReport", "Get report", params(path("report_id", "JotForm report ID.")), "", "#/components/schemas/JotFormObject", "jotformAPIKey"), "delete": op("deleteJotFormReport", "Delete report", params(path("report_id", "JotForm report ID.")), "", "#/components/schemas/JotFormObject", "jotformAPIKey")},
			"/submission/{submission_id}": {"get": op("getJotFormSubmission", "Get submission", params(path("submission_id", "Submission ID.")), "", "#/components/schemas/JotFormObject", "jotformAPIKey")},
			"/user/folders":               {"get": op("listJotFormFolders", "List user folders", params(query("limit", "Maximum number of folders."), query("offset", "Pagination offset.")), "", "#/components/schemas/JotFormCollection", "jotformAPIKey")},
			"/user/reports":               {"get": op("listJotFormReports", "List user reports", params(query("limit", "Maximum number of reports."), query("offset", "Pagination offset.")), "", "#/components/schemas/JotFormCollection", "jotformAPIKey")},
		},
	}
}

func formIOOverlay() overlaySpec {
	security := map[string]map[string]any{
		"formioJWTToken": {"type": "apiKey", "in": "header", "name": "x-jwt-token", "description": "Form.io JWT token carried in the x-jwt-token header."},
	}
	return overlaySpec{
		ProviderID:  "formio",
		Title:       "Form.io API Advisory Overlay",
		Description: "Advisory OpenAPI overlay derived from official Form.io human documentation. This is not an official Form.io OpenAPI document.",
		ServerURL:   "https://api.form.io",
		Sources:     []string{"https://help.form.io/developers/introduction/api-documentation", "https://help.form.io/developers/authentication"},
		SourceNote:  "Form.io publishes human API documentation for cloud and self-hosted deployments but no stable public official OpenAPI document; this overlay covers selected projects, forms, submissions, and roles endpoints.",
		Security:    security,
		Schemas:     []string{"FormIOObject", "FormIOCollection", "FormIOError"},
		OutputPath:  "catalog-openapi-cache/advisory-overlays/formio-api-overlay.json",
		Paths: map[string]map[string]any{
			"/project":                                {"get": op("listFormIOProjects", "List projects", nil, "", "#/components/schemas/FormIOCollection", "formioJWTToken")},
			"/project/{project_id}":                   {"get": op("getFormIOProject", "Get project", params(path("project_id", "Project ID.")), "", "#/components/schemas/FormIOObject", "formioJWTToken")},
			"/{project_alias}/form":                   {"get": op("listFormIOForms", "List project forms", params(path("project_alias", "Project alias or path.")), "", "#/components/schemas/FormIOCollection", "formioJWTToken"), "post": op("createFormIOForm", "Create form", params(path("project_alias", "Project alias or path.")), "#/components/schemas/FormIOObject", "#/components/schemas/FormIOObject", "formioJWTToken")},
			"/{project_alias}/{form_path}":            {"get": op("getFormIOForm", "Get form", params(path("project_alias", "Project alias or path."), path("form_path", "Form path.")), "", "#/components/schemas/FormIOObject", "formioJWTToken")},
			"/{project_alias}/{form_path}/submission": {"get": op("listFormIOSubmissions", "List form submissions", params(path("project_alias", "Project alias or path."), path("form_path", "Form path."), query("limit", "Page size."), query("skip", "Pagination offset.")), "", "#/components/schemas/FormIOCollection", "formioJWTToken"), "post": op("createFormIOSubmission", "Create submission", params(path("project_alias", "Project alias or path."), path("form_path", "Form path.")), "#/components/schemas/FormIOObject", "#/components/schemas/FormIOObject", "formioJWTToken")},
			"/{project_alias}/role":                   {"get": op("listFormIORoles", "List project roles", params(path("project_alias", "Project alias or path.")), "", "#/components/schemas/FormIOCollection", "formioJWTToken")},
		},
	}
}

func formstackOverlay() overlaySpec {
	security := map[string]map[string]any{
		"formstackBearer": {"type": "http", "scheme": "bearer", "bearerFormat": "OAuth 2.0 access token", "description": "Formstack OAuth access token carried in the Authorization bearer header."},
	}
	return overlaySpec{
		ProviderID:  "formstack",
		Title:       "Formstack API v2 Advisory Overlay",
		Description: "Advisory OpenAPI overlay derived from official Formstack human documentation. This is not an official Formstack OpenAPI document.",
		ServerURL:   "https://www.formstack.com/api/v2",
		Sources:     []string{"https://help.formstack.com/hc/en-us/articles/44593115536403", "https://developers.formstack.com/v2.0/reference/api-overview", "https://developers.formstack.com/reference/authorization"},
		SourceNote:  "Formstack publishes human API documentation but no recorded official OpenAPI document; this overlay covers selected forms, fields, submissions, folders, and webhooks endpoints.",
		Security:    security,
		Schemas:     []string{"FormstackObject", "FormstackCollection", "FormstackError"},
		OutputPath:  "catalog-openapi-cache/advisory-overlays/formstack-api-v2-overlay.json",
		Paths: map[string]map[string]any{
			"/form.json":                       {"get": op("listFormstackForms", "List forms", params(query("page", "Page number."), query("per_page", "Page size.")), "", "#/components/schemas/FormstackCollection", "formstackBearer")},
			"/form/{form_id}.json":             {"get": op("getFormstackForm", "Get form", params(path("form_id", "Form ID.")), "", "#/components/schemas/FormstackObject", "formstackBearer")},
			"/form/{form_id}/field.json":       {"get": op("listFormstackFormFields", "List form fields", params(path("form_id", "Form ID.")), "", "#/components/schemas/FormstackCollection", "formstackBearer")},
			"/form/{form_id}/submission.json":  {"get": op("listFormstackSubmissions", "List form submissions", params(path("form_id", "Form ID."), query("page", "Page number."), query("per_page", "Page size.")), "", "#/components/schemas/FormstackCollection", "formstackBearer")},
			"/form/{form_id}/webhook.json":     {"get": op("listFormstackWebhooks", "List form webhooks", params(path("form_id", "Form ID.")), "", "#/components/schemas/FormstackCollection", "formstackBearer"), "post": op("createFormstackWebhook", "Create form webhook", params(path("form_id", "Form ID.")), "#/components/schemas/FormstackObject", "#/components/schemas/FormstackObject", "formstackBearer")},
			"/submission/{submission_id}.json": {"get": op("getFormstackSubmission", "Get submission", params(path("submission_id", "Submission ID.")), "", "#/components/schemas/FormstackObject", "formstackBearer")},
		},
	}
}

func surveyMonkeyOverlay() overlaySpec {
	security := map[string]map[string]any{
		"surveyMonkeyBearer": {"type": "http", "scheme": "bearer", "bearerFormat": "OAuth 2.0 access token", "description": "SurveyMonkey OAuth access token carried in the Authorization bearer header."},
	}
	return overlaySpec{
		ProviderID:  "surveymonkey",
		Title:       "SurveyMonkey API v3 Advisory Overlay",
		Description: "Advisory OpenAPI overlay derived from official SurveyMonkey human documentation. This is not an official SurveyMonkey OpenAPI document.",
		ServerURL:   "https://api.surveymonkey.com/v3",
		Sources:     []string{"https://api.surveymonkey.com/v3/docs", "https://help.surveymonkey.com/en/surveymonkey/integrations/surveymonkey-api/"},
		SourceNote:  "SurveyMonkey publishes human API v3 documentation but no recorded official OpenAPI document; this overlay covers selected surveys, collectors, responses, questions, and webhooks endpoints.",
		Security:    security,
		Schemas:     []string{"SurveyMonkeyObject", "SurveyMonkeyCollection", "SurveyMonkeyError"},
		OutputPath:  "catalog-openapi-cache/advisory-overlays/surveymonkey-api-v3-overlay.json",
		Paths: map[string]map[string]any{
			"/surveys":                                       {"get": op("listSurveyMonkeySurveys", "List surveys", params(query("page", "Page number."), query("per_page", "Page size.")), "", "#/components/schemas/SurveyMonkeyCollection", "surveyMonkeyBearer")},
			"/surveys/{survey_id}":                           {"get": op("getSurveyMonkeySurvey", "Get survey", params(path("survey_id", "Survey ID.")), "", "#/components/schemas/SurveyMonkeyObject", "surveyMonkeyBearer")},
			"/surveys/{survey_id}/collectors":                {"get": op("listSurveyMonkeyCollectors", "List survey collectors", params(path("survey_id", "Survey ID.")), "", "#/components/schemas/SurveyMonkeyCollection", "surveyMonkeyBearer")},
			"/surveys/{survey_id}/responses/bulk":            {"get": op("listSurveyMonkeyResponsesBulk", "List survey responses in bulk", params(path("survey_id", "Survey ID."), query("page", "Page number."), query("per_page", "Page size.")), "", "#/components/schemas/SurveyMonkeyCollection", "surveyMonkeyBearer")},
			"/surveys/{survey_id}/pages/{page_id}/questions": {"get": op("listSurveyMonkeyPageQuestions", "List page questions", params(path("survey_id", "Survey ID."), path("page_id", "Page ID.")), "", "#/components/schemas/SurveyMonkeyCollection", "surveyMonkeyBearer")},
			"/webhooks":                                      {"get": op("listSurveyMonkeyWebhooks", "List webhooks", nil, "", "#/components/schemas/SurveyMonkeyCollection", "surveyMonkeyBearer"), "post": op("createSurveyMonkeyWebhook", "Create webhook", nil, "#/components/schemas/SurveyMonkeyObject", "#/components/schemas/SurveyMonkeyObject", "surveyMonkeyBearer")},
		},
	}
}

func bubbleOverlay() overlaySpec {
	security := map[string]map[string]any{
		"bubbleBearer": {"type": "http", "scheme": "bearer", "bearerFormat": "Bubble API token", "description": "Bubble API token carried in the Authorization bearer header."},
	}
	return overlaySpec{
		ProviderID:  "bubble",
		Title:       "Bubble API Advisory Overlay",
		Description: "Advisory OpenAPI overlay derived from official Bubble human documentation. This is not an official Bubble OpenAPI document.",
		ServerURL:   "https://{bubble_app_host}/api/1.1",
		Sources:     []string{"https://manual.bubble.io/help-guides/integrations/api/the-bubble-api", "https://manual.bubble.io/core-resources/api/data-api", "https://manual.bubble.io/core-resources/api/workflow-api"},
		SourceNote:  "Bubble publishes human API docs and app-specific Swagger metadata but no stable provider-wide OpenAPI document; this overlay covers generic Data API and Workflow API entry points.",
		Security:    security,
		Schemas:     []string{"BubbleObject", "BubbleCollection", "BubbleError"},
		OutputPath:  "catalog-openapi-cache/advisory-overlays/bubble-api-overlay.json",
		Paths: map[string]map[string]any{
			"/obj/{type_name}":             {"get": op("listBubbleObjects", "List objects of a type", params(path("type_name", "Bubble data type API name."), query("limit", "Page size."), query("cursor", "Pagination cursor."), query("constraints", "Stringified Bubble search constraints.")), "", "#/components/schemas/BubbleCollection", "bubbleBearer"), "post": op("createBubbleObject", "Create object", params(path("type_name", "Bubble data type API name.")), "#/components/schemas/BubbleObject", "#/components/schemas/BubbleObject", "bubbleBearer")},
			"/obj/{type_name}/{object_id}": {"get": op("getBubbleObject", "Get object", params(path("type_name", "Bubble data type API name."), path("object_id", "Bubble object ID.")), "", "#/components/schemas/BubbleObject", "bubbleBearer"), "patch": op("updateBubbleObject", "Update object", params(path("type_name", "Bubble data type API name."), path("object_id", "Bubble object ID.")), "#/components/schemas/BubbleObject", "#/components/schemas/BubbleObject", "bubbleBearer"), "delete": op("deleteBubbleObject", "Delete object", params(path("type_name", "Bubble data type API name."), path("object_id", "Bubble object ID.")), "", "", "bubbleBearer")},
			"/wf/{workflow_name}":          {"post": op("runBubbleWorkflow", "Run backend workflow", params(path("workflow_name", "Bubble backend workflow API name.")), "#/components/schemas/BubbleObject", "#/components/schemas/BubbleObject", "bubbleBearer")},
		},
	}
}

func cockpitOverlay() overlaySpec {
	security := map[string]map[string]any{
		"cockpitToken": {"type": "apiKey", "in": "query", "name": "token", "description": "Cockpit API token carried as the token request parameter."},
	}
	return overlaySpec{
		ProviderID:  "cockpit",
		Title:       "Cockpit API Advisory Overlay",
		Description: "Advisory OpenAPI overlay derived from official Cockpit human documentation. This is not an official Cockpit OpenAPI document.",
		ServerURL:   "https://{cockpit_host}/api",
		Sources:     []string{"https://getcockpit.com/documentation/core/api", "https://getcockpit.com/documentation/core"},
		SourceNote:  "Cockpit publishes human API docs for self-hosted instances but no recorded official OpenAPI document; this overlay covers selected collections, singletons, assets, and content endpoints.",
		Security:    security,
		Schemas:     []string{"CockpitObject", "CockpitCollection", "CockpitError"},
		OutputPath:  "catalog-openapi-cache/advisory-overlays/cockpit-api-overlay.json",
		Paths: map[string]map[string]any{
			"/content/items/{model}":      {"get": op("listCockpitContentItems", "List content items", params(path("model", "Collection model name."), query("limit", "Page size."), query("skip", "Pagination offset.")), "", "#/components/schemas/CockpitCollection", "cockpitToken")},
			"/content/item/{model}":       {"get": op("getCockpitContentItem", "Get one content item", params(path("model", "Collection model name."), query("filter", "Filter expression.")), "", "#/components/schemas/CockpitObject", "cockpitToken"), "post": op("saveCockpitContentItem", "Save content item", params(path("model", "Collection model name.")), "#/components/schemas/CockpitObject", "#/components/schemas/CockpitObject", "cockpitToken")},
			"/content/tree/{model}":       {"get": op("getCockpitContentTree", "Get content tree", params(path("model", "Tree model name.")), "", "#/components/schemas/CockpitCollection", "cockpitToken")},
			"/assets":                     {"get": op("listCockpitAssets", "List assets", nil, "", "#/components/schemas/CockpitCollection", "cockpitToken")},
			"/singletons/get/{singleton}": {"get": op("getCockpitSingleton", "Get singleton", params(path("singleton", "Singleton name.")), "", "#/components/schemas/CockpitObject", "cockpitToken")},
		},
	}
}

func stackbyOverlay() overlaySpec {
	security := map[string]map[string]any{
		"stackbyAPIKey": {"type": "apiKey", "in": "header", "name": "api-key", "description": "Stackby API key carried in the api-key header."},
	}
	return overlaySpec{
		ProviderID:  "stackby",
		Title:       "Stackby API Advisory Overlay",
		Description: "Advisory OpenAPI overlay derived from official Stackby human documentation. This is not an official Stackby OpenAPI document.",
		ServerURL:   "https://stackby.com/api/betav1",
		Sources:     []string{"https://help.stackby.com/en/articles/29-developer-api", "https://help.stackby.com/en/articles/124-how-to-get-your-api-key-in-stackby", "https://help.stackby.com/en/collections/26-api"},
		SourceNote:  "Stackby publishes human API documentation and API-key help but no recorded official OpenAPI document; this overlay covers selected stack, table, and row endpoints.",
		Security:    security,
		Schemas:     []string{"StackbyObject", "StackbyCollection", "StackbyError"},
		OutputPath:  "catalog-openapi-cache/advisory-overlays/stackby-api-overlay.json",
		Paths: map[string]map[string]any{
			"/stacklist":                       {"get": op("listStackbyStacks", "List stacks", nil, "", "#/components/schemas/StackbyCollection", "stackbyAPIKey")},
			"/tablelist/{stack_id}":            {"get": op("listStackbyTables", "List tables", params(path("stack_id", "Stack ID.")), "", "#/components/schemas/StackbyCollection", "stackbyAPIKey")},
			"/rowlist/{stack_id}/{table_name}": {"get": op("listStackbyRows", "List rows", params(path("stack_id", "Stack ID."), path("table_name", "Table name."), query("limit", "Page size."), query("offset", "Pagination offset.")), "", "#/components/schemas/StackbyCollection", "stackbyAPIKey")},
			"/rowcreate/{record_key}":          {"post": op("createStackbyRow", "Create row", params(path("record_key", "Record key.")), "#/components/schemas/StackbyObject", "#/components/schemas/StackbyObject", "stackbyAPIKey")},
			"/rowupdate/{record_key}":          {"patch": op("updateStackbyRow", "Update row", params(path("record_key", "Record key.")), "#/components/schemas/StackbyObject", "#/components/schemas/StackbyObject", "stackbyAPIKey")},
			"/rowdelete/{record_key}":          {"delete": op("deleteStackbyRow", "Delete row", params(path("record_key", "Record key.")), "", "", "stackbyAPIKey")},
		},
	}
}

func fileMakerOverlay() overlaySpec {
	security := map[string]map[string]any{
		"fileMakerBasic":  {"type": "http", "scheme": "basic", "description": "FileMaker Data API credentials used to create a database session."},
		"fileMakerBearer": {"type": "http", "scheme": "bearer", "bearerFormat": "FileMaker Data API session token", "description": "FileMaker Data API session token carried in the Authorization bearer header."},
	}
	return overlaySpec{
		ProviderID:  "filemaker",
		Title:       "FileMaker Data API Advisory Overlay",
		Description: "Advisory OpenAPI overlay derived from official Claris FileMaker human documentation. This is not an official FileMaker OpenAPI document.",
		ServerURL:   "https://{filemaker_host}/fmi/data/v1/databases/{database}",
		Sources:     []string{"https://help.claris.com/en/data-api-guide/content/data-api-reference.html", "https://help.claris.com/en/data-api-guide/"},
		SourceNote:  "Claris publishes human Data API reference docs and host-local reference files but no recorded stable public official OpenAPI document; this overlay covers selected session, layout, record, script, and container endpoints.",
		Security:    security,
		Schemas:     []string{"FileMakerObject", "FileMakerCollection", "FileMakerError"},
		OutputPath:  "catalog-openapi-cache/advisory-overlays/filemaker-data-api-overlay.json",
		Paths: map[string]map[string]any{
			"/sessions":                              {"post": op("createFileMakerSession", "Create session", nil, "", "#/components/schemas/FileMakerObject", "fileMakerBasic"), "delete": op("deleteFileMakerSession", "Delete session", nil, "", "", "fileMakerBearer")},
			"/layouts":                               {"get": op("listFileMakerLayouts", "List layouts", nil, "", "#/components/schemas/FileMakerCollection", "fileMakerBearer")},
			"/layouts/{layout}/records":              {"get": op("listFileMakerRecords", "List records", params(path("layout", "Layout name."), query("_limit", "Page size."), query("_offset", "Pagination offset.")), "", "#/components/schemas/FileMakerCollection", "fileMakerBearer"), "post": op("createFileMakerRecord", "Create record", params(path("layout", "Layout name.")), "#/components/schemas/FileMakerObject", "#/components/schemas/FileMakerObject", "fileMakerBearer")},
			"/layouts/{layout}/records/{record_id}":  {"get": op("getFileMakerRecord", "Get record", params(path("layout", "Layout name."), path("record_id", "Record ID.")), "", "#/components/schemas/FileMakerObject", "fileMakerBearer"), "patch": op("updateFileMakerRecord", "Update record", params(path("layout", "Layout name."), path("record_id", "Record ID.")), "#/components/schemas/FileMakerObject", "#/components/schemas/FileMakerObject", "fileMakerBearer"), "delete": op("deleteFileMakerRecord", "Delete record", params(path("layout", "Layout name."), path("record_id", "Record ID.")), "", "", "fileMakerBearer")},
			"/layouts/{layout}/script/{script_name}": {"get": op("runFileMakerScript", "Run script", params(path("layout", "Layout name."), path("script_name", "Script name."), query("script.param", "Script parameter.")), "", "#/components/schemas/FileMakerObject", "fileMakerBearer")},
			"/layouts/{layout}/records/{record_id}/containers/{field_name}/{repetition}": {"get": op("getFileMakerContainerData", "Get container data", params(path("layout", "Layout name."), path("record_id", "Record ID."), path("field_name", "Container field name."), path("repetition", "Container repetition.")), "", "", "fileMakerBearer")},
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
