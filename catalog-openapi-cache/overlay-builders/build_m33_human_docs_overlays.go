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
		flowOverlay(),
		gotoWebinarOverlay(),
		haloPSAOverlay(),
		loneScaleOverlay(),
		lemlistOverlay(),
		profitWellOverlay(),
		quickbaseOverlay(),
		taigaOverlay(),
		tapfiliateOverlay(),
	} {
		write(spec.OutputPath, build(spec))
	}
}

func flowOverlay() overlaySpec {
	security := map[string]map[string]any{
		"flowBearer": {"type": "http", "scheme": "bearer", "bearerFormat": "Flow personal access token", "description": "Flow personal access token carried in the Authorization bearer header."},
	}
	return overlaySpec{
		ProviderID:  "flow",
		Title:       "Flow API Advisory Overlay",
		Description: "Advisory OpenAPI overlay derived from official Flow human API documentation. This is not an official Flow OpenAPI document.",
		ServerURL:   "https://api.getflow.com/v2",
		Sources:     []string{"https://developer.getflow.com/api", "https://developer.getflow.com/"},
		SourceNote:  "Flow publishes human API documentation but no recorded stable public official OpenAPI document; this overlay covers selected task, token, and integration webhook endpoints.",
		Security:    security,
		Schemas:     []string{"FlowObject", "FlowCollection", "FlowError"},
		OutputPath:  "catalog-openapi-cache/advisory-overlays/flow-api-overlay.json",
		Paths: map[string]map[string]any{
			"/access_tokens":                     {"get": op("listFlowAccessTokens", "List access tokens", params(query("organization_id", "Organization ID.")), "", "#/components/schemas/FlowCollection", "flowBearer")},
			"/integration_webhooks":              {"get": op("listFlowIntegrationWebhooks", "List integration webhooks", nil, "", "#/components/schemas/FlowCollection", "flowBearer"), "post": op("createFlowIntegrationWebhook", "Create integration webhook", nil, "#/components/schemas/FlowObject", "#/components/schemas/FlowObject", "flowBearer")},
			"/integration_webhooks/{webhook_id}": {"delete": op("deleteFlowIntegrationWebhook", "Delete integration webhook", params(path("webhook_id", "Webhook ID.")), "", "#/components/schemas/FlowObject", "flowBearer")},
			"/tasks":                             {"get": op("listFlowTasks", "List tasks", params(query("organization_id", "Organization ID."), query("workspace_id", "Workspace ID."), query("include", "Related resources to include.")), "", "#/components/schemas/FlowCollection", "flowBearer"), "post": op("createFlowTask", "Create task", nil, "#/components/schemas/FlowObject", "#/components/schemas/FlowObject", "flowBearer")},
			"/tasks/{task_id}":                   {"get": op("getFlowTask", "Get task", params(path("task_id", "Task ID.")), "", "#/components/schemas/FlowObject", "flowBearer"), "put": op("updateFlowTask", "Update task", params(path("task_id", "Task ID.")), "#/components/schemas/FlowObject", "#/components/schemas/FlowObject", "flowBearer")},
		},
	}
}

func gotoWebinarOverlay() overlaySpec {
	security := map[string]map[string]any{
		"gotoOAuth2": {"type": "oauth2", "description": "GoTo OAuth2 access token carried in the Authorization bearer header.", "flows": map[string]any{"authorizationCode": map[string]any{"authorizationUrl": "https://api.getgo.com/oauth/v2/authorize", "tokenUrl": "https://api.getgo.com/oauth/v2/token", "scopes": map[string]string{}}}},
	}
	return overlaySpec{
		ProviderID:  "gotowebinar",
		Title:       "GoTo Webinar API Advisory Overlay",
		Description: "Advisory OpenAPI overlay derived from official GoTo Webinar human API documentation. This is not an official GoTo OpenAPI document.",
		ServerURL:   "https://api.getgo.com/G2W/rest/v2",
		Sources:     []string{"https://developer.goto.com/GoToWebinarV2/", "https://developer.goto.com/guides/Authentication/"},
		SourceNote:  "GoTo publishes human GoTo Webinar API documentation but no recorded stable public official OpenAPI document; this overlay covers selected webinar, session, registrant, attendee, panelist, and co-organizer endpoints.",
		Security:    security,
		Schemas:     []string{"GoToWebinarObject", "GoToWebinarCollection", "GoToWebinarError"},
		OutputPath:  "catalog-openapi-cache/advisory-overlays/gotowebinar-api-overlay.json",
		Paths: map[string]map[string]any{
			"/accounts/{account_key}/webinars":                                                                               {"get": op("listGoToWebinarAccountWebinars", "List account webinars", params(path("account_key", "Account key.")), "", "#/components/schemas/GoToWebinarCollection", "gotoOAuth2")},
			"/organizers/{organizer_key}/sessions":                                                                           {"get": op("listGoToWebinarOrganizerSessions", "List organizer sessions", params(path("organizer_key", "Organizer key.")), "", "#/components/schemas/GoToWebinarCollection", "gotoOAuth2")},
			"/organizers/{organizer_key}/webinars":                                                                           {"post": op("createGoToWebinar", "Create webinar", params(path("organizer_key", "Organizer key.")), "#/components/schemas/GoToWebinarObject", "#/components/schemas/GoToWebinarObject", "gotoOAuth2")},
			"/organizers/{organizer_key}/webinars/{webinar_key}":                                                             {"get": op("getGoToWebinar", "Get webinar", params(path("organizer_key", "Organizer key."), path("webinar_key", "Webinar key.")), "", "#/components/schemas/GoToWebinarObject", "gotoOAuth2"), "put": op("updateGoToWebinar", "Update webinar", params(path("organizer_key", "Organizer key."), path("webinar_key", "Webinar key.")), "#/components/schemas/GoToWebinarObject", "#/components/schemas/GoToWebinarObject", "gotoOAuth2"), "delete": op("deleteGoToWebinar", "Delete webinar", params(path("organizer_key", "Organizer key."), path("webinar_key", "Webinar key.")), "", "#/components/schemas/GoToWebinarObject", "gotoOAuth2")},
			"/organizers/{organizer_key}/webinars/{webinar_key}/coorganizers":                                                {"get": op("listGoToWebinarCoorganizers", "List co-organizers", params(path("organizer_key", "Organizer key."), path("webinar_key", "Webinar key.")), "", "#/components/schemas/GoToWebinarCollection", "gotoOAuth2"), "post": op("createGoToWebinarCoorganizer", "Create co-organizer", params(path("organizer_key", "Organizer key."), path("webinar_key", "Webinar key.")), "#/components/schemas/GoToWebinarObject", "#/components/schemas/GoToWebinarObject", "gotoOAuth2")},
			"/organizers/{organizer_key}/webinars/{webinar_key}/panelists":                                                   {"get": op("listGoToWebinarPanelists", "List panelists", params(path("organizer_key", "Organizer key."), path("webinar_key", "Webinar key.")), "", "#/components/schemas/GoToWebinarCollection", "gotoOAuth2"), "post": op("createGoToWebinarPanelist", "Create panelist", params(path("organizer_key", "Organizer key."), path("webinar_key", "Webinar key.")), "#/components/schemas/GoToWebinarObject", "#/components/schemas/GoToWebinarObject", "gotoOAuth2")},
			"/organizers/{organizer_key}/webinars/{webinar_key}/registrants":                                                 {"get": op("listGoToWebinarRegistrants", "List registrants", params(path("organizer_key", "Organizer key."), path("webinar_key", "Webinar key.")), "", "#/components/schemas/GoToWebinarCollection", "gotoOAuth2"), "post": op("createGoToWebinarRegistrant", "Create registrant", params(path("organizer_key", "Organizer key."), path("webinar_key", "Webinar key.")), "#/components/schemas/GoToWebinarObject", "#/components/schemas/GoToWebinarObject", "gotoOAuth2")},
			"/organizers/{organizer_key}/webinars/{webinar_key}/sessions":                                                    {"get": op("listGoToWebinarSessions", "List webinar sessions", params(path("organizer_key", "Organizer key."), path("webinar_key", "Webinar key.")), "", "#/components/schemas/GoToWebinarCollection", "gotoOAuth2")},
			"/organizers/{organizer_key}/webinars/{webinar_key}/sessions/{session_key}/attendees":                            {"get": op("listGoToWebinarAttendees", "List attendees", params(path("organizer_key", "Organizer key."), path("webinar_key", "Webinar key."), path("session_key", "Session key.")), "", "#/components/schemas/GoToWebinarCollection", "gotoOAuth2")},
			"/organizers/{organizer_key}/webinars/{webinar_key}/sessions/{session_key}/attendees/{registrant_key}":           {"get": op("getGoToWebinarAttendee", "Get attendee", params(path("organizer_key", "Organizer key."), path("webinar_key", "Webinar key."), path("session_key", "Session key."), path("registrant_key", "Registrant key.")), "", "#/components/schemas/GoToWebinarObject", "gotoOAuth2")},
			"/organizers/{organizer_key}/webinars/{webinar_key}/sessions/{session_key}/attendees/{registrant_key}/{details}": {"get": op("getGoToWebinarAttendeeDetails", "Get attendee details", params(path("organizer_key", "Organizer key."), path("webinar_key", "Webinar key."), path("session_key", "Session key."), path("registrant_key", "Registrant key."), path("details", "Details segment.")), "", "#/components/schemas/GoToWebinarObject", "gotoOAuth2")},
			"/organizers/{organizer_key}/webinars/{webinar_key}/sessions/{session_key}/{details}":                            {"get": op("getGoToWebinarSessionDetails", "Get session details", params(path("organizer_key", "Organizer key."), path("webinar_key", "Webinar key."), path("session_key", "Session key."), path("details", "Details segment.")), "", "#/components/schemas/GoToWebinarObject", "gotoOAuth2")},
			"/organizers/{organizer_key}/webinars/{webinar_key}/registrants/{registrant_key}":                                {"get": op("getGoToWebinarRegistrant", "Get registrant", params(path("organizer_key", "Organizer key."), path("webinar_key", "Webinar key."), path("registrant_key", "Registrant key.")), "", "#/components/schemas/GoToWebinarObject", "gotoOAuth2"), "delete": op("deleteGoToWebinarRegistrant", "Delete registrant", params(path("organizer_key", "Organizer key."), path("webinar_key", "Webinar key."), path("registrant_key", "Registrant key.")), "", "#/components/schemas/GoToWebinarObject", "gotoOAuth2")},
		},
	}
}

func haloPSAOverlay() overlaySpec {
	security := map[string]map[string]any{
		"halopsaOAuth2": {"type": "oauth2", "description": "HaloPSA OAuth2 client credentials access token carried in the Authorization bearer header.", "flows": map[string]any{"clientCredentials": map[string]any{"tokenUrl": "https://{halo_auth_host}/auth/token", "scopes": map[string]string{"admin": "Administrative API access.", "edit:tickets": "Edit tickets.", "edit:customers": "Edit customers."}}}},
	}
	return overlaySpec{
		ProviderID:  "halopsa",
		Title:       "HaloPSA API Advisory Overlay",
		Description: "Advisory OpenAPI overlay derived from official HaloPSA human API documentation. This is not an official HaloPSA OpenAPI document.",
		ServerURL:   "https://{halo_resource_host}/api",
		Sources:     []string{"https://apidoc.halopsa.com/", "https://support.halopsa.com/portal/kb/articles/haloapi"},
		SourceNote:  "HaloPSA publishes human API documentation and tenant-specific API hosts but no recorded stable provider-wide public OpenAPI document; this overlay covers selected tickets, clients, sites, and users endpoints.",
		Security:    security,
		Schemas:     []string{"HaloPSAObject", "HaloPSACollection", "HaloPSAError"},
		OutputPath:  "catalog-openapi-cache/advisory-overlays/halopsa-api-overlay.json",
		Paths: map[string]map[string]any{
			"/client":       {"get": op("listHaloPSAClients", "List clients", nil, "", "#/components/schemas/HaloPSACollection", "halopsaOAuth2"), "post": op("createHaloPSAClient", "Create client", nil, "#/components/schemas/HaloPSAObject", "#/components/schemas/HaloPSAObject", "halopsaOAuth2")},
			"/client/{id}":  {"get": op("getHaloPSAClient", "Get client", params(path("id", "Client ID.")), "", "#/components/schemas/HaloPSAObject", "halopsaOAuth2"), "put": op("updateHaloPSAClient", "Update client", params(path("id", "Client ID.")), "#/components/schemas/HaloPSAObject", "#/components/schemas/HaloPSAObject", "halopsaOAuth2")},
			"/site":         {"get": op("listHaloPSASites", "List sites", nil, "", "#/components/schemas/HaloPSACollection", "halopsaOAuth2")},
			"/site/{id}":    {"get": op("getHaloPSASite", "Get site", params(path("id", "Site ID.")), "", "#/components/schemas/HaloPSAObject", "halopsaOAuth2")},
			"/tickets":      {"get": op("listHaloPSATickets", "List tickets", nil, "", "#/components/schemas/HaloPSACollection", "halopsaOAuth2"), "post": op("createHaloPSATicket", "Create ticket", nil, "#/components/schemas/HaloPSAObject", "#/components/schemas/HaloPSAObject", "halopsaOAuth2")},
			"/tickets/{id}": {"get": op("getHaloPSATicket", "Get ticket", params(path("id", "Ticket ID.")), "", "#/components/schemas/HaloPSAObject", "halopsaOAuth2"), "put": op("updateHaloPSATicket", "Update ticket", params(path("id", "Ticket ID.")), "#/components/schemas/HaloPSAObject", "#/components/schemas/HaloPSAObject", "halopsaOAuth2")},
			"/users":        {"get": op("listHaloPSAUsers", "List users", nil, "", "#/components/schemas/HaloPSACollection", "halopsaOAuth2")},
			"/users/{id}":   {"get": op("getHaloPSAUser", "Get user", params(path("id", "User ID.")), "", "#/components/schemas/HaloPSAObject", "halopsaOAuth2")},
		},
	}
}

func loneScaleOverlay() overlaySpec {
	security := map[string]map[string]any{
		"lonescaleAPIKey": {"type": "apiKey", "in": "header", "name": "X-API-KEY", "description": "LoneScale API key carried in the X-API-KEY request header."},
	}
	return overlaySpec{
		ProviderID:  "lonescale",
		Title:       "LoneScale API Advisory Overlay",
		Description: "Advisory OpenAPI overlay derived from official LoneScale human API documentation. This is not an official LoneScale OpenAPI document.",
		ServerURL:   "https://public-api.lonescale.com",
		Sources:     []string{"https://help-center.lonescale.com/en/articles/6454360-lonescale-public-api", "https://public-api.lonescale.com/api"},
		SourceNote:  "LoneScale publishes public API help documentation but no recorded stable public official OpenAPI document; this overlay covers selected list, item, and workflow endpoints.",
		Security:    security,
		Schemas:     []string{"LoneScaleObject", "LoneScaleCollection", "LoneScaleError"},
		OutputPath:  "catalog-openapi-cache/advisory-overlays/lonescale-api-overlay.json",
		Paths: map[string]map[string]any{
			"/lists":                {"get": op("listLoneScaleLists", "List lists", params(query("entity", "Entity type.")), "", "#/components/schemas/LoneScaleCollection", "lonescaleAPIKey"), "post": op("createLoneScaleList", "Create list", nil, "#/components/schemas/LoneScaleObject", "#/components/schemas/LoneScaleObject", "lonescaleAPIKey")},
			"/lists/{list_id}":      {"get": op("getLoneScaleList", "Get list", params(path("list_id", "List ID.")), "", "#/components/schemas/LoneScaleObject", "lonescaleAPIKey")},
			"/lists/{list_id}/item": {"post": op("addLoneScaleListItem", "Add list item", params(path("list_id", "List ID.")), "#/components/schemas/LoneScaleObject", "#/components/schemas/LoneScaleObject", "lonescaleAPIKey")},
			"/users":                {"get": op("getLoneScaleCurrentUser", "Get current user", nil, "", "#/components/schemas/LoneScaleObject", "lonescaleAPIKey")},
			"/workflows":            {"get": op("listLoneScaleWorkflows", "List workflows", nil, "", "#/components/schemas/LoneScaleCollection", "lonescaleAPIKey")},
		},
	}
}

func lemlistOverlay() overlaySpec {
	security := map[string]map[string]any{
		"lemlistBasic": {"type": "http", "scheme": "basic", "description": "lemlist API key carried as the HTTP Basic password with an empty username."},
	}
	return overlaySpec{
		ProviderID:  "lemlist",
		Title:       "lemlist API Advisory Overlay",
		Description: "Advisory OpenAPI overlay derived from official lemlist human API documentation. This is not an official lemlist OpenAPI document.",
		ServerURL:   "https://api.lemlist.com/api",
		Sources:     []string{"https://developer.lemlist.com/api-reference/getting-started/overview", "https://developer.lemlist.com/api-reference/getting-started/authentication", "https://developer.lemlist.com/llms.txt"},
		SourceNote:  "lemlist publishes human API documentation but no recorded stable public official OpenAPI document; this overlay covers selected campaigns, leads, activities, enrichment, team, and unsubscribe endpoints.",
		Security:    security,
		Schemas:     []string{"LemlistObject", "LemlistCollection", "LemlistError"},
		OutputPath:  "catalog-openapi-cache/advisory-overlays/lemlist-api-overlay.json",
		Paths: map[string]map[string]any{
			"/activities":                            {"get": op("listLemlistActivities", "List activities", params(query("campaignId", "Campaign ID."), query("limit", "Maximum results.")), "", "#/components/schemas/LemlistCollection", "lemlistBasic")},
			"/campaigns":                             {"get": op("listLemlistCampaigns", "List campaigns", nil, "", "#/components/schemas/LemlistCollection", "lemlistBasic"), "post": op("createLemlistCampaign", "Create campaign", nil, "#/components/schemas/LemlistObject", "#/components/schemas/LemlistObject", "lemlistBasic")},
			"/campaigns/{campaign_id}":               {"get": op("getLemlistCampaign", "Get campaign", params(path("campaign_id", "Campaign ID.")), "", "#/components/schemas/LemlistObject", "lemlistBasic"), "patch": op("updateLemlistCampaign", "Update campaign", params(path("campaign_id", "Campaign ID.")), "#/components/schemas/LemlistObject", "#/components/schemas/LemlistObject", "lemlistBasic")},
			"/campaigns/{campaign_id}/leads":         {"get": op("listLemlistCampaignLeads", "List campaign leads", params(path("campaign_id", "Campaign ID.")), "", "#/components/schemas/LemlistCollection", "lemlistBasic")},
			"/campaigns/{campaign_id}/leads/{email}": {"get": op("getLemlistLeadByEmail", "Get lead by email", params(path("campaign_id", "Campaign ID."), path("email", "Lead email.")), "", "#/components/schemas/LemlistObject", "lemlistBasic"), "post": op("createLemlistCampaignLead", "Create campaign lead", params(path("campaign_id", "Campaign ID."), path("email", "Lead email.")), "#/components/schemas/LemlistObject", "#/components/schemas/LemlistObject", "lemlistBasic"), "delete": op("deleteLemlistCampaignLead", "Delete campaign lead", params(path("campaign_id", "Campaign ID."), path("email", "Lead email.")), "", "#/components/schemas/LemlistObject", "lemlistBasic")},
			"/enrich":                                {"post": op("enrichLemlistData", "Enrich data", nil, "#/components/schemas/LemlistObject", "#/components/schemas/LemlistObject", "lemlistBasic")},
			"/leads/{lead_id}/enrich":                {"post": op("enrichLemlistLead", "Enrich lead", params(path("lead_id", "Lead ID.")), "#/components/schemas/LemlistObject", "#/components/schemas/LemlistObject", "lemlistBasic")},
			"/team":                                  {"get": op("getLemlistTeam", "Get team", nil, "", "#/components/schemas/LemlistObject", "lemlistBasic")},
			"/unsubscribes":                          {"get": op("listLemlistUnsubscribes", "List unsubscribes", nil, "", "#/components/schemas/LemlistCollection", "lemlistBasic"), "post": op("addLemlistUnsubscribe", "Add unsubscribe", nil, "#/components/schemas/LemlistObject", "#/components/schemas/LemlistObject", "lemlistBasic")},
			"/unsubscribes/{email}":                  {"delete": op("deleteLemlistUnsubscribe", "Delete unsubscribe", params(path("email", "Email address.")), "", "#/components/schemas/LemlistObject", "lemlistBasic")},
		},
	}
}

func profitWellOverlay() overlaySpec {
	security := map[string]map[string]any{
		"profitwellAuthorization": {"type": "apiKey", "in": "header", "name": "Authorization", "description": "ProfitWell private or public API token carried directly in the Authorization header."},
	}
	return overlaySpec{
		ProviderID:  "profitwell",
		Title:       "ProfitWell API Advisory Overlay",
		Description: "Advisory OpenAPI overlay derived from official ProfitWell human API documentation. This is not an official ProfitWell OpenAPI document.",
		ServerURL:   "https://api.profitwell.com/v2",
		Sources:     []string{"https://profitwellapiv2.docs.apiary.io/"},
		SourceNote:  "ProfitWell publishes human API documentation but no recorded stable public official OpenAPI document; this overlay covers selected company and metrics endpoints.",
		Security:    security,
		Schemas:     []string{"ProfitWellObject", "ProfitWellCollection", "ProfitWellError"},
		OutputPath:  "catalog-openapi-cache/advisory-overlays/profitwell-api-v2-overlay.json",
		Paths: map[string]map[string]any{
			"/company/settings": {"get": op("getProfitWellCompanySettings", "Get company settings", nil, "", "#/components/schemas/ProfitWellObject", "profitwellAuthorization")},
			"/metrics/daily":    {"get": op("getProfitWellDailyMetrics", "Get daily metrics", params(query("month", "Month."), query("metrics", "Comma-separated metrics.")), "", "#/components/schemas/ProfitWellObject", "profitwellAuthorization")},
			"/metrics/monthly":  {"get": op("getProfitWellMonthlyMetrics", "Get monthly metrics", params(query("metrics", "Comma-separated metrics.")), "", "#/components/schemas/ProfitWellObject", "profitwellAuthorization")},
			"/metrics/plans":    {"get": op("listProfitWellPlans", "List metric plans", nil, "", "#/components/schemas/ProfitWellCollection", "profitwellAuthorization")},
		},
	}
}

func quickbaseOverlay() overlaySpec {
	security := map[string]map[string]any{
		"quickbaseBearer":        {"type": "http", "scheme": "bearer", "bearerFormat": "Quickbase user token", "description": "Quickbase user token carried in the Authorization bearer header."},
		"quickbaseRealmHostname": {"type": "apiKey", "in": "header", "name": "QB-Realm-Hostname", "description": "Quickbase realm hostname carried in the QB-Realm-Hostname request header."},
	}
	return overlaySpec{
		ProviderID:  "quickbase",
		Title:       "Quickbase REST API Advisory Overlay",
		Description: "Advisory OpenAPI overlay derived from official Quickbase human API documentation. This is not an official Quickbase OpenAPI document.",
		ServerURL:   "https://api.quickbase.com/v1",
		Sources:     []string{"https://developer.quickbase.com/", "https://help.quickbase.com/docs/authentication-and-secure-access", "https://help.quickbase.com/docs/create-and-use-user-tokens"},
		SourceNote:  "Quickbase publishes REST API human documentation but no directly downloadable stable official OpenAPI document during review; this overlay covers selected fields, records, reports, and file endpoints.",
		Security:    security,
		Schemas:     []string{"QuickbaseObject", "QuickbaseCollection", "QuickbaseError"},
		OutputPath:  "catalog-openapi-cache/advisory-overlays/quickbase-rest-api-overlay.json",
		Paths: map[string]map[string]any{
			"/files/{file_id}":         {"get": op("downloadQuickbaseFile", "Download file", params(path("file_id", "File ID.")), "", "#/components/schemas/QuickbaseObject", "quickbaseBearer", "quickbaseRealmHostname"), "delete": op("deleteQuickbaseFile", "Delete file", params(path("file_id", "File ID.")), "", "#/components/schemas/QuickbaseObject", "quickbaseBearer", "quickbaseRealmHostname")},
			"/fields":                  {"get": op("listQuickbaseFields", "List fields", params(query("tableId", "Table ID.")), "", "#/components/schemas/QuickbaseCollection", "quickbaseBearer", "quickbaseRealmHostname")},
			"/records":                 {"post": op("upsertQuickbaseRecords", "Create or update records", nil, "#/components/schemas/QuickbaseObject", "#/components/schemas/QuickbaseObject", "quickbaseBearer", "quickbaseRealmHostname"), "delete": op("deleteQuickbaseRecords", "Delete records", nil, "#/components/schemas/QuickbaseObject", "#/components/schemas/QuickbaseObject", "quickbaseBearer", "quickbaseRealmHostname")},
			"/records/query":           {"post": op("queryQuickbaseRecords", "Query records", nil, "#/components/schemas/QuickbaseObject", "#/components/schemas/QuickbaseCollection", "quickbaseBearer", "quickbaseRealmHostname")},
			"/reports/{report_id}/run": {"post": op("runQuickbaseReport", "Run report", params(path("report_id", "Report ID.")), "#/components/schemas/QuickbaseObject", "#/components/schemas/QuickbaseCollection", "quickbaseBearer", "quickbaseRealmHostname")},
			"/reports/{report_id}":     {"get": op("getQuickbaseReport", "Get report", params(path("report_id", "Report ID.")), "", "#/components/schemas/QuickbaseObject", "quickbaseBearer", "quickbaseRealmHostname")},
		},
	}
}

func taigaOverlay() overlaySpec {
	security := map[string]map[string]any{
		"taigaBearer": {"type": "http", "scheme": "bearer", "bearerFormat": "Taiga auth token", "description": "Taiga authentication token carried in the Authorization bearer header."},
	}
	return overlaySpec{
		ProviderID:  "taiga",
		Title:       "Taiga API Advisory Overlay",
		Description: "Advisory OpenAPI overlay derived from official Taiga human API documentation. This is not an official Taiga OpenAPI document.",
		ServerURL:   "https://api.taiga.io/api/v1",
		Sources:     []string{"https://docs.taiga.io/api.html", "https://docs.taiga.io/api.html#_authentication"},
		SourceNote:  "Taiga publishes human REST API documentation but no recorded stable public official OpenAPI document; this overlay covers selected projects, epics, issues, tasks, user stories, users, and webhooks endpoints.",
		Security:    security,
		Schemas:     []string{"TaigaObject", "TaigaCollection", "TaigaError"},
		OutputPath:  "catalog-openapi-cache/advisory-overlays/taiga-api-overlay.json",
		Paths: map[string]map[string]any{
			"/auth":                  {"post": op("createTaigaAuthToken", "Create auth token", nil, "#/components/schemas/TaigaObject", "#/components/schemas/TaigaObject")},
			"/epics":                 {"get": op("listTaigaEpics", "List epics", params(query("project", "Project ID.")), "", "#/components/schemas/TaigaCollection", "taigaBearer"), "post": op("createTaigaEpic", "Create epic", nil, "#/components/schemas/TaigaObject", "#/components/schemas/TaigaObject", "taigaBearer")},
			"/epics/{id}":            {"get": op("getTaigaEpic", "Get epic", params(path("id", "Epic ID.")), "", "#/components/schemas/TaigaObject", "taigaBearer"), "patch": op("updateTaigaEpic", "Update epic", params(path("id", "Epic ID.")), "#/components/schemas/TaigaObject", "#/components/schemas/TaigaObject", "taigaBearer"), "delete": op("deleteTaigaEpic", "Delete epic", params(path("id", "Epic ID.")), "", "#/components/schemas/TaigaObject", "taigaBearer")},
			"/issues":                {"get": op("listTaigaIssues", "List issues", params(query("project", "Project ID.")), "", "#/components/schemas/TaigaCollection", "taigaBearer"), "post": op("createTaigaIssue", "Create issue", nil, "#/components/schemas/TaigaObject", "#/components/schemas/TaigaObject", "taigaBearer")},
			"/issues/{id}":           {"get": op("getTaigaIssue", "Get issue", params(path("id", "Issue ID.")), "", "#/components/schemas/TaigaObject", "taigaBearer"), "patch": op("updateTaigaIssue", "Update issue", params(path("id", "Issue ID.")), "#/components/schemas/TaigaObject", "#/components/schemas/TaigaObject", "taigaBearer"), "delete": op("deleteTaigaIssue", "Delete issue", params(path("id", "Issue ID.")), "", "#/components/schemas/TaigaObject", "taigaBearer")},
			"/projects":              {"get": op("listTaigaProjects", "List projects", nil, "", "#/components/schemas/TaigaCollection", "taigaBearer")},
			"/tasks":                 {"get": op("listTaigaTasks", "List tasks", params(query("project", "Project ID.")), "", "#/components/schemas/TaigaCollection", "taigaBearer"), "post": op("createTaigaTask", "Create task", nil, "#/components/schemas/TaigaObject", "#/components/schemas/TaigaObject", "taigaBearer")},
			"/tasks/{id}":            {"get": op("getTaigaTask", "Get task", params(path("id", "Task ID.")), "", "#/components/schemas/TaigaObject", "taigaBearer"), "patch": op("updateTaigaTask", "Update task", params(path("id", "Task ID.")), "#/components/schemas/TaigaObject", "#/components/schemas/TaigaObject", "taigaBearer"), "delete": op("deleteTaigaTask", "Delete task", params(path("id", "Task ID.")), "", "#/components/schemas/TaigaObject", "taigaBearer")},
			"/userstories":           {"get": op("listTaigaUserStories", "List user stories", params(query("project", "Project ID.")), "", "#/components/schemas/TaigaCollection", "taigaBearer"), "post": op("createTaigaUserStory", "Create user story", nil, "#/components/schemas/TaigaObject", "#/components/schemas/TaigaObject", "taigaBearer")},
			"/userstories/{id}":      {"get": op("getTaigaUserStory", "Get user story", params(path("id", "User story ID.")), "", "#/components/schemas/TaigaObject", "taigaBearer"), "patch": op("updateTaigaUserStory", "Update user story", params(path("id", "User story ID.")), "#/components/schemas/TaigaObject", "#/components/schemas/TaigaObject", "taigaBearer"), "delete": op("deleteTaigaUserStory", "Delete user story", params(path("id", "User story ID.")), "", "#/components/schemas/TaigaObject", "taigaBearer")},
			"/users/me":              {"get": op("getTaigaCurrentUser", "Get current user", nil, "", "#/components/schemas/TaigaObject", "taigaBearer")},
			"/webhooks":              {"get": op("listTaigaWebhooks", "List webhooks", nil, "", "#/components/schemas/TaigaCollection", "taigaBearer"), "post": op("createTaigaWebhook", "Create webhook", nil, "#/components/schemas/TaigaObject", "#/components/schemas/TaigaObject", "taigaBearer")},
			"/webhooks/{webhook_id}": {"delete": op("deleteTaigaWebhook", "Delete webhook", params(path("webhook_id", "Webhook ID.")), "", "#/components/schemas/TaigaObject", "taigaBearer")},
		},
	}
}

func tapfiliateOverlay() overlaySpec {
	security := map[string]map[string]any{
		"tapfiliateAPIKey": {"type": "apiKey", "in": "header", "name": "Api-Key", "description": "Tapfiliate API key carried in the Api-Key request header."},
	}
	return overlaySpec{
		ProviderID:  "tapfiliate",
		Title:       "Tapfiliate REST API Advisory Overlay",
		Description: "Advisory OpenAPI overlay derived from official Tapfiliate human API documentation. This is not an official Tapfiliate OpenAPI document.",
		ServerURL:   "https://api.tapfiliate.com/1.6",
		Sources:     []string{"https://tapfiliate.com/docs/rest/", "https://tapfiliate.com/docs/rest/#authentication"},
		SourceNote:  "Tapfiliate publishes human REST API documentation but no recorded stable public official OpenAPI document; this overlay covers selected affiliate, metadata, and program affiliate endpoints.",
		Security:    security,
		Schemas:     []string{"TapfiliateObject", "TapfiliateCollection", "TapfiliateError"},
		OutputPath:  "catalog-openapi-cache/advisory-overlays/tapfiliate-api-overlay.json",
		Paths: map[string]map[string]any{
			"/affiliates":                {"get": op("listTapfiliateAffiliates", "List affiliates", nil, "", "#/components/schemas/TapfiliateCollection", "tapfiliateAPIKey"), "post": op("createTapfiliateAffiliate", "Create affiliate", nil, "#/components/schemas/TapfiliateObject", "#/components/schemas/TapfiliateObject", "tapfiliateAPIKey")},
			"/affiliates/{affiliate_id}": {"get": op("getTapfiliateAffiliate", "Get affiliate", params(path("affiliate_id", "Affiliate ID.")), "", "#/components/schemas/TapfiliateObject", "tapfiliateAPIKey"), "delete": op("deleteTapfiliateAffiliate", "Delete affiliate", params(path("affiliate_id", "Affiliate ID.")), "", "#/components/schemas/TapfiliateObject", "tapfiliateAPIKey")},
			"/affiliates/{affiliate_id}/metadata/{metadata_key}":        {"put": op("setTapfiliateAffiliateMetadata", "Set affiliate metadata", params(path("affiliate_id", "Affiliate ID."), path("metadata_key", "Metadata key.")), "#/components/schemas/TapfiliateObject", "#/components/schemas/TapfiliateObject", "tapfiliateAPIKey"), "delete": op("deleteTapfiliateAffiliateMetadata", "Delete affiliate metadata", params(path("affiliate_id", "Affiliate ID."), path("metadata_key", "Metadata key.")), "", "#/components/schemas/TapfiliateObject", "tapfiliateAPIKey")},
			"/affiliates/{affiliate_id}/notes":                          {"get": op("listTapfiliateAffiliateNotes", "List affiliate notes", params(path("affiliate_id", "Affiliate ID.")), "", "#/components/schemas/TapfiliateCollection", "tapfiliateAPIKey")},
			"/programs":                                                 {"get": op("listTapfiliatePrograms", "List programs", nil, "", "#/components/schemas/TapfiliateCollection", "tapfiliateAPIKey")},
			"/programs/{program_id}/affiliates":                         {"get": op("listTapfiliateProgramAffiliates", "List program affiliates", params(path("program_id", "Program ID.")), "", "#/components/schemas/TapfiliateCollection", "tapfiliateAPIKey"), "post": op("addTapfiliateProgramAffiliate", "Add program affiliate", params(path("program_id", "Program ID.")), "#/components/schemas/TapfiliateObject", "#/components/schemas/TapfiliateObject", "tapfiliateAPIKey")},
			"/programs/{program_id}/affiliates/{affiliate_id}":          {"get": op("getTapfiliateProgramAffiliate", "Get program affiliate", params(path("program_id", "Program ID."), path("affiliate_id", "Affiliate ID.")), "", "#/components/schemas/TapfiliateObject", "tapfiliateAPIKey")},
			"/programs/{program_id}/affiliates/{affiliate_id}/approved": {"put": op("approveTapfiliateProgramAffiliate", "Approve program affiliate", params(path("program_id", "Program ID."), path("affiliate_id", "Affiliate ID.")), "#/components/schemas/TapfiliateObject", "#/components/schemas/TapfiliateObject", "tapfiliateAPIKey"), "delete": op("disapproveTapfiliateProgramAffiliate", "Disapprove program affiliate", params(path("program_id", "Program ID."), path("affiliate_id", "Affiliate ID.")), "", "#/components/schemas/TapfiliateObject", "tapfiliateAPIKey")},
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
