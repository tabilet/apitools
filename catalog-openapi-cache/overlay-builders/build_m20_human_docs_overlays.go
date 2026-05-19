//go:build ignore

package main

import (
	"encoding/json"
	"os"
)

type overlaySpec struct {
	ProviderID   string
	OverlayID    string
	Title        string
	Description  string
	ServerURL    string
	ServerVars   map[string]map[string]any
	Sources      []string
	SourceNote   string
	SecurityName string
	Security     map[string]any
	SecurityAlt  map[string]map[string]any
	TokenNote    string
	Schemas      []string
	Paths        map[string]map[string]any
	OutputPath   string
}

func main() {
	for _, spec := range []overlaySpec{
		databricksOverlay(),
		jenkinsOverlay(),
		sentryOverlay(),
		splunkOverlay(),
		telegramOverlay(),
	} {
		write(spec.OutputPath, build(spec))
	}
}

func databricksOverlay() overlaySpec {
	security := bearer("databricksBearer", "Databricks personal access token or OAuth bearer token carried in the Authorization header.")
	return overlaySpec{
		ProviderID:  "databricks",
		OverlayID:   "databricks-workspace-rest-advisory-overlay",
		Title:       "Databricks Workspace REST API Advisory Overlay",
		Description: "Advisory OpenAPI overlay derived from official Databricks REST API human documentation. This is not an official Databricks OpenAPI document.",
		ServerURL:   "https://{workspace_host}",
		ServerVars: map[string]map[string]any{
			"workspace_host": {"default": "dbc.example.cloud.databricks.com", "description": "Operator-supplied Databricks workspace host."},
		},
		Sources: []string{
			"https://docs.databricks.com/api/workspace/introduction",
			"https://docs.databricks.com/aws/en/reference/jobs-api-2-1-updates",
			"https://docs.databricks.com/aws/en/dev-tools/auth",
			"https://docs.databricks.com/aws/en/dev-tools/auth/pat",
		},
		SourceNote:   "Databricks publishes official human REST API documentation and auth docs; workspace/account host selection remains operator supplied.",
		SecurityName: "databricksBearer",
		Security:     security,
		Schemas:      []string{"DatabricksObject", "DatabricksCollection", "DatabricksError"},
		OutputPath:   "catalog-openapi-cache/advisory-overlays/databricks-workspace-rest-overlay.json",
		Paths: map[string]map[string]any{
			"/api/2.1/jobs/list": {
				"get": op("listDatabricksJobs", "List jobs", params(query("limit", "Maximum number of jobs to return."), query("offset", "Offset for pagination.")), "", "#/components/schemas/DatabricksCollection", "databricksBearer"),
			},
			"/api/2.1/jobs/get": {
				"get": op("getDatabricksJob", "Get a job", params(query("job_id", "Databricks job ID.")), "", "#/components/schemas/DatabricksObject", "databricksBearer"),
			},
			"/api/2.1/jobs/run-now": {
				"post": op("runDatabricksJobNow", "Trigger a job run", nil, "#/components/schemas/DatabricksObject", "#/components/schemas/DatabricksObject", "databricksBearer"),
			},
			"/api/2.0/clusters/list": {
				"get": op("listDatabricksClusters", "List clusters", nil, "", "#/components/schemas/DatabricksCollection", "databricksBearer"),
			},
			"/api/2.0/clusters/get": {
				"get": op("getDatabricksCluster", "Get a cluster", params(query("cluster_id", "Databricks cluster ID.")), "", "#/components/schemas/DatabricksObject", "databricksBearer"),
			},
			"/api/2.0/workspace/list": {
				"get": op("listDatabricksWorkspaceObjects", "List workspace objects", params(query("path", "Workspace path.")), "", "#/components/schemas/DatabricksCollection", "databricksBearer"),
			},
			"/api/2.0/dbfs/list": {
				"get": op("listDatabricksDBFS", "List DBFS files", params(query("path", "DBFS path.")), "", "#/components/schemas/DatabricksCollection", "databricksBearer"),
			},
		},
	}
}

func jenkinsOverlay() overlaySpec {
	security := map[string]map[string]any{
		"jenkinsBasic": {"type": "http", "scheme": "basic", "description": "Jenkins user name with API token or password for HTTP Basic authentication."},
		"jenkinsCrumb": {"type": "apiKey", "in": "header", "name": "Jenkins-Crumb", "description": "Optional CSRF crumb header when required by the Jenkins controller."},
	}
	return overlaySpec{
		ProviderID:  "jenkins",
		OverlayID:   "jenkins-remote-api-advisory-overlay",
		Title:       "Jenkins Remote Access API Advisory Overlay",
		Description: "Advisory OpenAPI overlay derived from official Jenkins Remote Access API human documentation. This is not an official Jenkins OpenAPI document.",
		ServerURL:   "https://{jenkins_host}",
		ServerVars: map[string]map[string]any{
			"jenkins_host": {"default": "jenkins.example.com", "description": "Operator-supplied Jenkins controller host."},
		},
		Sources: []string{
			"https://www.jenkins.io/doc/book/using/remote-access-api/",
			"https://www.jenkins.io/blog/2018/07/02/new-api-token-system/",
		},
		SourceNote:  "Jenkins exposes REST-like JSON API endpoints under object-specific /api/ URLs; installed plugins and controller configuration affect the available surface.",
		SecurityAlt: security,
		Schemas:     []string{"JenkinsObject", "JenkinsCollection", "JenkinsError"},
		OutputPath:  "catalog-openapi-cache/advisory-overlays/jenkins-remote-api-overlay.json",
		Paths: map[string]map[string]any{
			"/api/json": {
				"get": op("getJenkinsRoot", "Get Jenkins root API data", params(query("depth", "Optional Jenkins tree depth.")), "", "#/components/schemas/JenkinsObject", "jenkinsBasic"),
			},
			"/job/{job_name}/api/json": {
				"get": op("getJenkinsJob", "Get job API data", params(path("job_name", "Jenkins job name."), query("depth", "Optional Jenkins tree depth.")), "", "#/components/schemas/JenkinsObject", "jenkinsBasic"),
			},
			"/job/{job_name}/build": {
				"post": op("buildJenkinsJob", "Trigger a job build", params(path("job_name", "Jenkins job name.")), "", "#/components/schemas/JenkinsObject", "jenkinsBasic", "jenkinsCrumb"),
			},
			"/job/{job_name}/buildWithParameters": {
				"post": op("buildJenkinsJobWithParameters", "Trigger a parameterized job build", params(path("job_name", "Jenkins job name.")), "#/components/schemas/JenkinsObject", "#/components/schemas/JenkinsObject", "jenkinsBasic", "jenkinsCrumb"),
			},
			"/queue/api/json": {
				"get": op("getJenkinsQueue", "Get build queue", nil, "", "#/components/schemas/JenkinsCollection", "jenkinsBasic"),
			},
			"/crumbIssuer/api/json": {
				"get": op("getJenkinsCrumb", "Get CSRF crumb", nil, "", "#/components/schemas/JenkinsObject", "jenkinsBasic"),
			},
		},
	}
}

func sentryOverlay() overlaySpec {
	security := bearer("sentryBearer", "Sentry authentication token carried in the Authorization header.")
	return overlaySpec{
		ProviderID:  "sentry",
		OverlayID:   "sentry-rest-api-advisory-overlay",
		Title:       "Sentry REST API Advisory Overlay",
		Description: "Advisory OpenAPI overlay derived from official Sentry API human documentation. This is not an official Sentry OpenAPI document.",
		ServerURL:   "https://sentry.io/api/0",
		Sources: []string{
			"https://docs.sentry.io/api/",
			"https://docs.sentry.io/api/auth/",
			"https://docs.sentry.io/api/events/",
			"https://docs.sentry.io/api/organizations/",
			"https://docs.sentry.io/api/projects/",
			"https://docs.sentry.io/api/releases/",
		},
		SourceNote:   "Sentry publishes official human API reference pages with REST paths and bearer token guidance; region-specific domains may be required by operators.",
		SecurityName: "sentryBearer",
		Security:     security,
		Schemas:      []string{"SentryObject", "SentryCollection", "SentryError"},
		OutputPath:   "catalog-openapi-cache/advisory-overlays/sentry-rest-api-overlay.json",
		Paths: map[string]map[string]any{
			"/organizations/{organization_slug}/projects/": {
				"get": op("listSentryOrganizationProjects", "List organization projects", params(path("organization_slug", "Sentry organization slug.")), "", "#/components/schemas/SentryCollection", "sentryBearer"),
			},
			"/organizations/{organization_slug}/issues/": {
				"get": op("listSentryOrganizationIssues", "List organization issues", params(path("organization_slug", "Sentry organization slug."), query("query", "Issue search query."), query("project", "Project filter.")), "", "#/components/schemas/SentryCollection", "sentryBearer"),
			},
			"/issues/{issue_id}/": {
				"get":    op("getSentryIssue", "Retrieve an issue", params(path("issue_id", "Sentry issue ID.")), "", "#/components/schemas/SentryObject", "sentryBearer"),
				"put":    op("updateSentryIssue", "Update an issue", params(path("issue_id", "Sentry issue ID.")), "#/components/schemas/SentryObject", "#/components/schemas/SentryObject", "sentryBearer"),
				"delete": op("deleteSentryIssue", "Delete an issue", params(path("issue_id", "Sentry issue ID.")), "", "#/components/schemas/SentryObject", "sentryBearer"),
			},
			"/issues/{issue_id}/events/": {
				"get": op("listSentryIssueEvents", "List an issue's events", params(path("issue_id", "Sentry issue ID.")), "", "#/components/schemas/SentryCollection", "sentryBearer"),
			},
			"/organizations/{organization_slug}/events/": {
				"get": op("listSentryOrganizationEvents", "List organization events", params(path("organization_slug", "Sentry organization slug."), query("project", "Project filter."), query("statsPeriod", "Relative time window.")), "", "#/components/schemas/SentryCollection", "sentryBearer"),
			},
			"/projects/{organization_slug}/{project_slug}/releases/": {
				"get": op("listSentryProjectReleases", "List project releases", params(path("organization_slug", "Sentry organization slug."), path("project_slug", "Sentry project slug.")), "", "#/components/schemas/SentryCollection", "sentryBearer"),
			},
		},
	}
}

func splunkOverlay() overlaySpec {
	security := map[string]map[string]any{
		"splunkBearer": {"type": "http", "scheme": "bearer", "bearerFormat": "Splunk session key or token", "description": "Splunk bearer token or session key carried in the Authorization header."},
		"splunkBasic":  {"type": "http", "scheme": "basic", "description": "Splunk username and password authentication for deployments that allow Basic auth."},
	}
	return overlaySpec{
		ProviderID:  "splunk",
		OverlayID:   "splunk-enterprise-rest-advisory-overlay",
		Title:       "Splunk Enterprise REST API Advisory Overlay",
		Description: "Advisory OpenAPI overlay derived from official Splunk REST API human documentation. This is not an official Splunk OpenAPI document.",
		ServerURL:   "https://{splunk_host}:8089",
		ServerVars: map[string]map[string]any{
			"splunk_host": {"default": "splunk.example.com", "description": "Operator-supplied Splunk management host."},
		},
		Sources: []string{
			"https://docs.splunk.com/Documentation/Splunk/latest/RESTREF",
			"https://docs.splunk.com/Documentation/Splunk/latest/RESTUM/RESTusing",
			"https://docs.splunk.com/Documentation/Splunk/9.4.2/RESTREF/RESTsearch",
			"https://docs.splunk.com/Documentation/Splunk/9.4.2/RESTTUT/RESTsearches",
		},
		SourceNote:  "Splunk Enterprise REST documentation is broad and version-specific; this overlay covers a small reviewed search/admin subset.",
		SecurityAlt: security,
		Schemas:     []string{"SplunkObject", "SplunkCollection", "SplunkError"},
		OutputPath:  "catalog-openapi-cache/advisory-overlays/splunk-enterprise-rest-overlay.json",
		Paths: map[string]map[string]any{
			"/services/server/info": {
				"get": op("getSplunkServerInfo", "Get server information", nil, "", "#/components/schemas/SplunkObject", "splunkBearer"),
			},
			"/services/search/jobs": {
				"get":  op("listSplunkSearchJobs", "List search jobs", nil, "", "#/components/schemas/SplunkCollection", "splunkBearer"),
				"post": op("createSplunkSearchJob", "Create a search job", nil, "#/components/schemas/SplunkObject", "#/components/schemas/SplunkObject", "splunkBearer"),
			},
			"/services/search/jobs/{sid}": {
				"get":    op("getSplunkSearchJob", "Get a search job", params(path("sid", "Splunk search job ID.")), "", "#/components/schemas/SplunkObject", "splunkBearer"),
				"delete": op("deleteSplunkSearchJob", "Delete a search job", params(path("sid", "Splunk search job ID.")), "", "#/components/schemas/SplunkObject", "splunkBearer"),
			},
			"/services/search/jobs/{sid}/results": {
				"get": op("getSplunkSearchJobResults", "Get search job results", params(path("sid", "Splunk search job ID."), query("output_mode", "Response output mode.")), "", "#/components/schemas/SplunkCollection", "splunkBearer"),
			},
			"/servicesNS/{owner}/{app}/saved/searches": {
				"get":  op("listSplunkSavedSearches", "List saved searches", params(path("owner", "Splunk namespace owner."), path("app", "Splunk app namespace.")), "", "#/components/schemas/SplunkCollection", "splunkBearer"),
				"post": op("createSplunkSavedSearch", "Create a saved search", params(path("owner", "Splunk namespace owner."), path("app", "Splunk app namespace.")), "#/components/schemas/SplunkObject", "#/components/schemas/SplunkObject", "splunkBearer"),
			},
		},
	}
}

func telegramOverlay() overlaySpec {
	return overlaySpec{
		ProviderID:  "telegram",
		OverlayID:   "telegram-bot-api-advisory-overlay",
		Title:       "Telegram Bot API Advisory Overlay",
		Description: "Advisory OpenAPI overlay derived from official Telegram Bot API human documentation. This is not an official Telegram OpenAPI document.",
		ServerURL:   "https://api.telegram.org/bot{token}",
		ServerVars: map[string]map[string]any{
			"token": {"default": "000000:bot_token", "description": "Telegram bot token. OpenAPI security schemes cannot model this path token exactly."},
		},
		Sources: []string{
			"https://core.telegram.org/bots/api",
		},
		SourceNote: "Telegram Bot API docs define bot methods under https://api.telegram.org/bot<token>/METHOD_NAME and allow GET or POST; this overlay uses POST for common methods.",
		TokenNote:  "Bot token is modeled as a server variable for advisory endpoint shape only; credential handling remains downstream.",
		Schemas:    []string{"TelegramObject", "TelegramCollection", "TelegramError"},
		OutputPath: "catalog-openapi-cache/advisory-overlays/telegram-bot-api-overlay.json",
		Paths: map[string]map[string]any{
			"/getMe": {
				"post": op("getTelegramBotMe", "Get bot information", nil, "", "#/components/schemas/TelegramObject"),
			},
			"/getUpdates": {
				"post": op("getTelegramUpdates", "Get incoming updates", nil, "#/components/schemas/TelegramObject", "#/components/schemas/TelegramCollection"),
			},
			"/sendMessage": {
				"post": op("sendTelegramMessage", "Send a text message", nil, "#/components/schemas/TelegramObject", "#/components/schemas/TelegramObject"),
			},
			"/editMessageText": {
				"post": op("editTelegramMessageText", "Edit a text message", nil, "#/components/schemas/TelegramObject", "#/components/schemas/TelegramObject"),
			},
			"/answerCallbackQuery": {
				"post": op("answerTelegramCallbackQuery", "Answer a callback query", nil, "#/components/schemas/TelegramObject", "#/components/schemas/TelegramObject"),
			},
			"/sendDocument": {
				"post": op("sendTelegramDocument", "Send a document", nil, "#/components/schemas/TelegramObject", "#/components/schemas/TelegramObject"),
			},
		},
	}
}

func build(spec overlaySpec) map[string]any {
	doc := map[string]any{
		"openapi": "3.0.3",
		"info": map[string]any{
			"title":       spec.Title,
			"version":     "2026-05-19",
			"description": spec.Description,
		},
		"servers": []map[string]any{server(spec.ServerURL, spec.ServerVars)},
		"x-apitools-overlay": map[string]any{
			"provider_id":       spec.ProviderID,
			"overlay_id":        spec.OverlayID,
			"official_openapi":  false,
			"derived_from_docs": true,
			"source_refs":       spec.Sources,
			"source_note":       spec.SourceNote,
		},
		"components": map[string]any{
			"securitySchemes": securitySchemes(spec),
			"schemas":         schemas(spec.Schemas),
		},
		"paths": spec.Paths,
	}
	if spec.SecurityName != "" {
		doc["security"] = []map[string]any{{spec.SecurityName: []string{}}}
	}
	if spec.TokenNote != "" {
		doc["x-apitools-token-placement"] = spec.TokenNote
	}
	return doc
}

func server(url string, variables map[string]map[string]any) map[string]any {
	out := map[string]any{"url": url}
	if len(variables) > 0 {
		out["variables"] = variables
	}
	return out
}

func securitySchemes(spec overlaySpec) map[string]any {
	if len(spec.SecurityAlt) > 0 {
		out := map[string]any{}
		for name, value := range spec.SecurityAlt {
			out[name] = value
		}
		return out
	}
	if spec.SecurityName == "" {
		return map[string]any{}
	}
	return map[string]any{spec.SecurityName: spec.Security[spec.SecurityName]}
}

func bearer(name, description string) map[string]any {
	return map[string]any{name: map[string]any{"type": "http", "scheme": "bearer", "description": description}}
}

func schemas(names []string) map[string]any {
	out := map[string]any{}
	for _, name := range names {
		out[name] = map[string]any{"type": "object", "additionalProperties": true}
	}
	return out
}

func op(operationID, summary string, parameters []map[string]any, requestSchema, responseSchema string, securityNames ...string) map[string]any {
	value := map[string]any{
		"operationId": operationID,
		"summary":     summary,
		"description": "Advisory operation derived from official human API documentation.",
		"responses": map[string]any{
			"200":     response("Successful response.", responseSchema),
			"default": response("Provider error response.", ""),
		},
	}
	if len(securityNames) > 0 {
		value["security"] = security(securityNames...)
	}
	if len(parameters) > 0 {
		value["parameters"] = parameters
	}
	if requestSchema != "" {
		value["requestBody"] = requestBody(requestSchema)
	}
	return value
}

func security(names ...string) []map[string]any {
	out := make([]map[string]any, 0, len(names))
	req := map[string]any{}
	for _, name := range names {
		req[name] = []string{}
	}
	if len(req) > 0 {
		out = append(out, req)
	}
	return out
}

func response(description, schema string) map[string]any {
	out := map[string]any{"description": description}
	if schema != "" {
		out["content"] = map[string]any{"application/json": map[string]any{"schema": map[string]any{"$ref": schema}}}
	}
	return out
}

func requestBody(schema string) map[string]any {
	return map[string]any{"required": true, "content": map[string]any{"application/json": map[string]any{"schema": map[string]any{"$ref": schema}}}}
}

func params(values ...map[string]any) []map[string]any {
	return values
}

func path(name, description string) map[string]any {
	return map[string]any{"name": name, "in": "path", "required": true, "schema": map[string]any{"type": "string"}, "description": description}
}

func query(name, description string) map[string]any {
	return map[string]any{"name": name, "in": "query", "required": false, "schema": map[string]any{"type": "string"}, "description": description}
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
