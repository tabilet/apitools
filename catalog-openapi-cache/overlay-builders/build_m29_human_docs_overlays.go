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
		bitwardenOverlay(),
		webexOverlay(),
		cortexOverlay(),
		homeAssistantOverlay(),
		netscalerOverlay(),
		venafiOverlay(),
		wekanOverlay(),
		zammadOverlay(),
	} {
		write(spec.OutputPath, build(spec))
	}
}

func bitwardenOverlay() overlaySpec {
	security := map[string]map[string]any{
		"bitwardenBearer": {"type": "http", "scheme": "bearer", "bearerFormat": "Bitwarden Public API access token", "description": "Bitwarden Public API access token carried in the Authorization bearer header."},
	}
	return overlaySpec{
		ProviderID:  "bitwarden",
		Title:       "Bitwarden Public API Advisory Overlay",
		Description: "Advisory OpenAPI overlay derived from official Bitwarden Public API human documentation. This is not an official Bitwarden OpenAPI document.",
		ServerURL:   "https://api.bitwarden.com",
		Sources:     []string{"https://bitwarden.com/help/api/"},
		SourceNote:  "Bitwarden publishes Swagger-rendered Public API documentation but no recorded stable public downloadable OpenAPI document; this overlay covers selected organization groups, collections, members, and events endpoints.",
		Security:    security,
		Schemas:     []string{"BitwardenObject", "BitwardenCollection", "BitwardenError"},
		OutputPath:  "catalog-openapi-cache/advisory-overlays/bitwarden-public-api-overlay.json",
		Paths: map[string]map[string]any{
			"/public/groups":            {"get": op("listBitwardenGroups", "List groups", nil, "", "#/components/schemas/BitwardenCollection", "bitwardenBearer"), "post": op("createBitwardenGroup", "Create group", nil, "#/components/schemas/BitwardenObject", "#/components/schemas/BitwardenObject", "bitwardenBearer")},
			"/public/groups/{id}":       {"get": op("getBitwardenGroup", "Get group", params(path("id", "Group ID.")), "", "#/components/schemas/BitwardenObject", "bitwardenBearer"), "put": op("updateBitwardenGroup", "Update group", params(path("id", "Group ID.")), "#/components/schemas/BitwardenObject", "#/components/schemas/BitwardenObject", "bitwardenBearer"), "delete": op("deleteBitwardenGroup", "Delete group", params(path("id", "Group ID.")), "", "", "bitwardenBearer")},
			"/public/collections":       {"get": op("listBitwardenCollections", "List collections", nil, "", "#/components/schemas/BitwardenCollection", "bitwardenBearer")},
			"/public/collections/{id}":  {"get": op("getBitwardenCollection", "Get collection", params(path("id", "Collection ID.")), "", "#/components/schemas/BitwardenObject", "bitwardenBearer")},
			"/public/members":           {"get": op("listBitwardenMembers", "List members", nil, "", "#/components/schemas/BitwardenCollection", "bitwardenBearer")},
			"/public/members/{id}":      {"get": op("getBitwardenMember", "Get member", params(path("id", "Member ID.")), "", "#/components/schemas/BitwardenObject", "bitwardenBearer")},
			"/public/events":            {"get": op("listBitwardenEvents", "List events", params(query("start", "Start timestamp."), query("end", "End timestamp.")), "", "#/components/schemas/BitwardenCollection", "bitwardenBearer")},
			"/public/organization/info": {"get": op("getBitwardenOrganizationInfo", "Get organization information", nil, "", "#/components/schemas/BitwardenObject", "bitwardenBearer")},
		},
	}
}

func webexOverlay() overlaySpec {
	security := map[string]map[string]any{
		"webexOAuth2": {"type": "http", "scheme": "bearer", "bearerFormat": "Cisco Webex OAuth 2.0 access token", "description": "Cisco Webex OAuth access token carried in the Authorization bearer header."},
	}
	return overlaySpec{
		ProviderID:  "cisco-webex",
		Title:       "Cisco Webex API Advisory Overlay",
		Description: "Advisory OpenAPI overlay derived from official Cisco Webex API human documentation. This is not an official Cisco OpenAPI document.",
		ServerURL:   "https://webexapis.com",
		Sources:     []string{"https://developer.webex.com/docs/api/v1/rooms", "https://developer.webex.com/docs/api/v1/roomsrooms", "https://developer.webex.com/docs/api/v1/roomsmessages", "https://developer.webex.com/docs/integrations"},
		SourceNote:  "Cisco Webex publishes human API documentation but no recorded stable public downloadable OpenAPI document; this overlay covers selected rooms, messages, memberships, people, and webhooks endpoints.",
		Security:    security,
		Schemas:     []string{"WebexObject", "WebexCollection", "WebexError"},
		OutputPath:  "catalog-openapi-cache/advisory-overlays/cisco-webex-api-overlay.json",
		Paths: map[string]map[string]any{
			"/v1/rooms":                   {"get": op("listWebexRooms", "List rooms", params(query("type", "Room type filter.")), "", "#/components/schemas/WebexCollection", "webexOAuth2"), "post": op("createWebexRoom", "Create room", nil, "#/components/schemas/WebexObject", "#/components/schemas/WebexObject", "webexOAuth2")},
			"/v1/rooms/{roomId}":          {"get": op("getWebexRoom", "Get room", params(path("roomId", "Room ID.")), "", "#/components/schemas/WebexObject", "webexOAuth2"), "put": op("updateWebexRoom", "Update room", params(path("roomId", "Room ID.")), "#/components/schemas/WebexObject", "#/components/schemas/WebexObject", "webexOAuth2"), "delete": op("deleteWebexRoom", "Delete room", params(path("roomId", "Room ID.")), "", "", "webexOAuth2")},
			"/v1/messages":                {"get": op("listWebexMessages", "List messages", params(query("roomId", "Room ID.")), "", "#/components/schemas/WebexCollection", "webexOAuth2"), "post": op("createWebexMessage", "Create message", nil, "#/components/schemas/WebexObject", "#/components/schemas/WebexObject", "webexOAuth2")},
			"/v1/messages/{messageId}":    {"get": op("getWebexMessage", "Get message", params(path("messageId", "Message ID.")), "", "#/components/schemas/WebexObject", "webexOAuth2"), "delete": op("deleteWebexMessage", "Delete message", params(path("messageId", "Message ID.")), "", "", "webexOAuth2")},
			"/v1/memberships":             {"get": op("listWebexMemberships", "List memberships", params(query("roomId", "Room ID.")), "", "#/components/schemas/WebexCollection", "webexOAuth2"), "post": op("createWebexMembership", "Create membership", nil, "#/components/schemas/WebexObject", "#/components/schemas/WebexObject", "webexOAuth2")},
			"/v1/people/me":               {"get": op("getWebexMe", "Get current user", nil, "", "#/components/schemas/WebexObject", "webexOAuth2")},
			"/v1/webhooks":                {"get": op("listWebexWebhooks", "List webhooks", nil, "", "#/components/schemas/WebexCollection", "webexOAuth2"), "post": op("createWebexWebhook", "Create webhook", nil, "#/components/schemas/WebexObject", "#/components/schemas/WebexObject", "webexOAuth2")},
			"/v1/webhooks/{webhookId}":    {"get": op("getWebexWebhook", "Get webhook", params(path("webhookId", "Webhook ID.")), "", "#/components/schemas/WebexObject", "webexOAuth2"), "delete": op("deleteWebexWebhook", "Delete webhook", params(path("webhookId", "Webhook ID.")), "", "", "webexOAuth2")},
			"/v1/attachment/actions/{id}": {"get": op("getWebexAttachmentAction", "Get attachment action", params(path("id", "Attachment action ID.")), "", "#/components/schemas/WebexObject", "webexOAuth2")},
			"/v1/recordings":              {"get": op("listWebexRecordings", "List recordings", nil, "", "#/components/schemas/WebexCollection", "webexOAuth2")},
		},
	}
}

func cortexOverlay() overlaySpec {
	security := map[string]map[string]any{
		"cortexBearer": {"type": "http", "scheme": "bearer", "bearerFormat": "Cortex API key", "description": "Cortex API key carried in the Authorization bearer header."},
	}
	return overlaySpec{
		ProviderID:  "cortex",
		Title:       "Cortex API Advisory Overlay",
		Description: "Advisory OpenAPI overlay derived from official StrangeBee Cortex API human documentation. This is not an official Cortex OpenAPI document.",
		ServerURL:   "https://{cortex_host}/api",
		Sources:     []string{"https://docs.strangebee.com/cortex/api/api-guide/"},
		SourceNote:  "Cortex publishes human API documentation but no recorded stable public official OpenAPI document; this overlay covers selected analyzer, responder, job, report, and organization endpoints.",
		Security:    security,
		Schemas:     []string{"CortexObject", "CortexCollection", "CortexError"},
		OutputPath:  "catalog-openapi-cache/advisory-overlays/cortex-api-overlay.json",
		Paths: map[string]map[string]any{
			"/analyzer":        {"get": op("listCortexAnalyzers", "List analyzers", nil, "", "#/components/schemas/CortexCollection", "cortexBearer")},
			"/analyzer/{id}":   {"get": op("getCortexAnalyzer", "Get analyzer", params(path("id", "Analyzer ID.")), "", "#/components/schemas/CortexObject", "cortexBearer")},
			"/responder":       {"get": op("listCortexResponders", "List responders", nil, "", "#/components/schemas/CortexCollection", "cortexBearer")},
			"/responder/{id}":  {"get": op("getCortexResponder", "Get responder", params(path("id", "Responder ID.")), "", "#/components/schemas/CortexObject", "cortexBearer")},
			"/job":             {"get": op("listCortexJobs", "List jobs", nil, "", "#/components/schemas/CortexCollection", "cortexBearer"), "post": op("createCortexJob", "Create job", nil, "#/components/schemas/CortexObject", "#/components/schemas/CortexObject", "cortexBearer")},
			"/job/{id}":        {"get": op("getCortexJob", "Get job", params(path("id", "Job ID.")), "", "#/components/schemas/CortexObject", "cortexBearer")},
			"/job/{id}/report": {"get": op("getCortexJobReport", "Get job report", params(path("id", "Job ID.")), "", "#/components/schemas/CortexObject", "cortexBearer")},
			"/organization":    {"get": op("listCortexOrganizations", "List organizations", nil, "", "#/components/schemas/CortexCollection", "cortexBearer")},
		},
	}
}

func homeAssistantOverlay() overlaySpec {
	security := map[string]map[string]any{
		"homeAssistantBearer": {"type": "http", "scheme": "bearer", "bearerFormat": "Home Assistant long-lived access token", "description": "Home Assistant long-lived access token carried in the Authorization bearer header."},
	}
	return overlaySpec{
		ProviderID:  "home-assistant",
		Title:       "Home Assistant REST API Advisory Overlay",
		Description: "Advisory OpenAPI overlay derived from official Home Assistant REST API human documentation. This is not an official Home Assistant OpenAPI document.",
		ServerURL:   "https://{home_assistant_host}/api",
		Sources:     []string{"https://developers.home-assistant.io/docs/api/rest/"},
		SourceNote:  "Home Assistant publishes human REST API documentation but no recorded stable public official OpenAPI document; this overlay covers selected config, state, service, event, history, template, and camera endpoints.",
		Security:    security,
		Schemas:     []string{"HomeAssistantObject", "HomeAssistantCollection", "HomeAssistantError"},
		OutputPath:  "catalog-openapi-cache/advisory-overlays/home-assistant-rest-api-overlay.json",
		Paths: map[string]map[string]any{
			"/":                            {"get": op("getHomeAssistantAPIStatus", "Get API status", nil, "", "#/components/schemas/HomeAssistantObject", "homeAssistantBearer")},
			"/config":                      {"get": op("getHomeAssistantConfig", "Get configuration", nil, "", "#/components/schemas/HomeAssistantObject", "homeAssistantBearer")},
			"/states":                      {"get": op("listHomeAssistantStates", "List states", nil, "", "#/components/schemas/HomeAssistantCollection", "homeAssistantBearer")},
			"/states/{entity_id}":          {"get": op("getHomeAssistantState", "Get entity state", params(path("entity_id", "Entity ID.")), "", "#/components/schemas/HomeAssistantObject", "homeAssistantBearer"), "post": op("setHomeAssistantState", "Set entity state", params(path("entity_id", "Entity ID.")), "#/components/schemas/HomeAssistantObject", "#/components/schemas/HomeAssistantObject", "homeAssistantBearer")},
			"/services":                    {"get": op("listHomeAssistantServices", "List services", nil, "", "#/components/schemas/HomeAssistantCollection", "homeAssistantBearer")},
			"/services/{domain}/{service}": {"post": op("callHomeAssistantService", "Call service", params(path("domain", "Service domain."), path("service", "Service name.")), "#/components/schemas/HomeAssistantObject", "#/components/schemas/HomeAssistantObject", "homeAssistantBearer")},
			"/history/period":              {"get": op("getHomeAssistantHistory", "Get history", params(query("filter_entity_id", "Entity filter.")), "", "#/components/schemas/HomeAssistantCollection", "homeAssistantBearer")},
			"/template":                    {"post": op("renderHomeAssistantTemplate", "Render template", nil, "#/components/schemas/HomeAssistantObject", "#/components/schemas/HomeAssistantObject", "homeAssistantBearer")},
			"/events/{event_type}":         {"post": op("fireHomeAssistantEvent", "Fire event", params(path("event_type", "Event type.")), "#/components/schemas/HomeAssistantObject", "#/components/schemas/HomeAssistantObject", "homeAssistantBearer")},
			"/camera_proxy/{entity_id}":    {"get": op("getHomeAssistantCameraProxy", "Get camera proxy image", params(path("entity_id", "Camera entity ID.")), "", "#/components/schemas/HomeAssistantObject", "homeAssistantBearer")},
		},
	}
}

func netscalerOverlay() overlaySpec {
	security := map[string]map[string]any{
		"netscalerNitroAuthToken": {"type": "apiKey", "in": "header", "name": "Cookie", "description": "NetScaler ADC NITRO session cookie carrying NITRO_AUTH_TOKEN after login."},
	}
	return overlaySpec{
		ProviderID:  "netscaler",
		Title:       "NetScaler ADC NITRO API Advisory Overlay",
		Description: "Advisory OpenAPI overlay derived from official NetScaler ADC NITRO API human documentation. This is not an official NetScaler OpenAPI document.",
		ServerURL:   "https://{netscaler_host}",
		Sources:     []string{"https://developer-docs.netscaler.com/en-us/adc-nitro-api/current-release/api-reference.html"},
		SourceNote:  "NetScaler ADC publishes human NITRO API reference documentation but no recorded stable public official OpenAPI document; this overlay covers selected configuration, statistics, and login endpoints.",
		Security:    security,
		Schemas:     []string{"NetScalerObject", "NetScalerCollection", "NetScalerError"},
		OutputPath:  "catalog-openapi-cache/advisory-overlays/netscaler-adc-nitro-api-overlay.json",
		Paths: map[string]map[string]any{
			"/nitro/v1/config/login":     {"post": op("loginNetScalerNitro", "Create NITRO session", nil, "#/components/schemas/NetScalerObject", "#/components/schemas/NetScalerObject")},
			"/nitro/v1/config/lbvserver": {"get": op("listNetScalerLBVServers", "List load-balancing virtual servers", nil, "", "#/components/schemas/NetScalerCollection", "netscalerNitroAuthToken"), "post": op("createNetScalerLBVServer", "Create load-balancing virtual server", nil, "#/components/schemas/NetScalerObject", "#/components/schemas/NetScalerObject", "netscalerNitroAuthToken")},
			"/nitro/v1/config/service":   {"get": op("listNetScalerServices", "List services", nil, "", "#/components/schemas/NetScalerCollection", "netscalerNitroAuthToken")},
			"/nitro/v1/config/server":    {"get": op("listNetScalerServers", "List servers", nil, "", "#/components/schemas/NetScalerCollection", "netscalerNitroAuthToken")},
			"/nitro/v1/config/csvserver": {"get": op("listNetScalerCSVServers", "List content switching virtual servers", nil, "", "#/components/schemas/NetScalerCollection", "netscalerNitroAuthToken")},
			"/nitro/v1/stat/lbvserver":   {"get": op("listNetScalerLBVServerStats", "List load-balancing virtual server statistics", nil, "", "#/components/schemas/NetScalerCollection", "netscalerNitroAuthToken")},
		},
	}
}

func venafiOverlay() overlaySpec {
	security := map[string]map[string]any{
		"venafiCloudAPIKey":      {"type": "apiKey", "in": "header", "name": "tppl-api-key", "description": "Venafi TLS Protect Cloud API key carried in the tppl-api-key header."},
		"venafiDatacenterBearer": {"type": "http", "scheme": "bearer", "bearerFormat": "Venafi Datacenter WebSDK access token", "description": "Venafi TLS Protect Datacenter access token carried in the Authorization bearer header."},
	}
	return overlaySpec{
		ProviderID:  "venafi",
		Title:       "Venafi API Advisory Overlay",
		Description: "Advisory OpenAPI overlay derived from official Venafi TLS Protect Cloud and Datacenter human documentation. This is not an official Venafi OpenAPI document.",
		ServerURL:   "https://{venafi_host}",
		Sources:     []string{"https://developer.venafi.com/tlsprotectcloud/reference", "https://developer.venafi.com/tlsprotectdatacenter/reference"},
		SourceNote:  "Venafi publishes human API documentation but no recorded stable public official OpenAPI document; this overlay covers selected TLS Protect Cloud certificate/application endpoints and Datacenter WebSDK certificate/token endpoints.",
		Security:    security,
		Schemas:     []string{"VenafiObject", "VenafiCollection", "VenafiError"},
		OutputPath:  "catalog-openapi-cache/advisory-overlays/venafi-api-overlay.json",
		Paths: map[string]map[string]any{
			"/v1/certificates":              {"get": op("listVenafiCloudCertificates", "List TLS Protect Cloud certificates", nil, "", "#/components/schemas/VenafiCollection", "venafiCloudAPIKey")},
			"/v1/certificaterequests":       {"get": op("listVenafiCloudCertificateRequests", "List TLS Protect Cloud certificate requests", nil, "", "#/components/schemas/VenafiCollection", "venafiCloudAPIKey"), "post": op("createVenafiCloudCertificateRequest", "Create TLS Protect Cloud certificate request", nil, "#/components/schemas/VenafiObject", "#/components/schemas/VenafiObject", "venafiCloudAPIKey")},
			"/v1/applications":              {"get": op("listVenafiCloudApplications", "List TLS Protect Cloud applications", nil, "", "#/components/schemas/VenafiCollection", "venafiCloudAPIKey")},
			"/v1/preferences":               {"get": op("getVenafiCloudPreferences", "Get TLS Protect Cloud preferences", nil, "", "#/components/schemas/VenafiObject", "venafiCloudAPIKey")},
			"/vedauth/authorize/oauth":      {"post": op("authorizeVenafiDatacenter", "Create Datacenter WebSDK access token", nil, "#/components/schemas/VenafiObject", "#/components/schemas/VenafiObject")},
			"/vedsdk/certificates/request":  {"post": op("requestVenafiDatacenterCertificate", "Request Datacenter certificate", nil, "#/components/schemas/VenafiObject", "#/components/schemas/VenafiObject", "venafiDatacenterBearer")},
			"/vedsdk/certificates/retrieve": {"post": op("retrieveVenafiDatacenterCertificate", "Retrieve Datacenter certificate", nil, "#/components/schemas/VenafiObject", "#/components/schemas/VenafiObject", "venafiDatacenterBearer")},
		},
	}
}

func wekanOverlay() overlaySpec {
	security := map[string]map[string]any{
		"wekanBearer": {"type": "http", "scheme": "bearer", "bearerFormat": "Wekan session token", "description": "Wekan session token carried in the Authorization bearer header."},
	}
	return overlaySpec{
		ProviderID:  "wekan",
		Title:       "Wekan REST API Advisory Overlay",
		Description: "Advisory OpenAPI overlay derived from official Wekan REST API wiki documentation. This is not an official Wekan OpenAPI document.",
		ServerURL:   "https://{wekan_host}",
		Sources:     []string{"https://github.com/wekan/wekan/wiki/REST-API", "https://github.com/wekan/wekan/wiki/REST-API-Authentication", "https://github.com/wekan/wekan/tree/main/openapi"},
		SourceNote:  "Wekan publishes REST API wiki documentation and OpenAPI generation tooling but no recorded stable public generated OpenAPI artifact; this overlay covers selected login, user, board, list, card, and checklist endpoints.",
		Security:    security,
		Schemas:     []string{"WekanObject", "WekanCollection", "WekanError"},
		OutputPath:  "catalog-openapi-cache/advisory-overlays/wekan-rest-api-overlay.json",
		Paths: map[string]map[string]any{
			"/users/login":                 {"post": op("loginWekan", "Create Wekan session", nil, "#/components/schemas/WekanObject", "#/components/schemas/WekanObject")},
			"/api/user":                    {"get": op("getWekanCurrentUser", "Get current user", nil, "", "#/components/schemas/WekanObject", "wekanBearer")},
			"/api/boards":                  {"get": op("listWekanBoards", "List boards", nil, "", "#/components/schemas/WekanCollection", "wekanBearer"), "post": op("createWekanBoard", "Create board", nil, "#/components/schemas/WekanObject", "#/components/schemas/WekanObject", "wekanBearer")},
			"/api/boards/{board_id}":       {"get": op("getWekanBoard", "Get board", params(path("board_id", "Board ID.")), "", "#/components/schemas/WekanObject", "wekanBearer")},
			"/api/boards/{board_id}/lists": {"get": op("listWekanLists", "List board lists", params(path("board_id", "Board ID.")), "", "#/components/schemas/WekanCollection", "wekanBearer"), "post": op("createWekanList", "Create list", params(path("board_id", "Board ID.")), "#/components/schemas/WekanObject", "#/components/schemas/WekanObject", "wekanBearer")},
			"/api/boards/{board_id}/lists/{list_id}/cards": {"get": op("listWekanCards", "List cards", params(path("board_id", "Board ID."), path("list_id", "List ID.")), "", "#/components/schemas/WekanCollection", "wekanBearer"), "post": op("createWekanCard", "Create card", params(path("board_id", "Board ID."), path("list_id", "List ID.")), "#/components/schemas/WekanObject", "#/components/schemas/WekanObject", "wekanBearer")},
			"/api/boards/{board_id}/cards/{card_id}":       {"get": op("getWekanCard", "Get card", params(path("board_id", "Board ID."), path("card_id", "Card ID.")), "", "#/components/schemas/WekanObject", "wekanBearer"), "put": op("updateWekanCard", "Update card", params(path("board_id", "Board ID."), path("card_id", "Card ID.")), "#/components/schemas/WekanObject", "#/components/schemas/WekanObject", "wekanBearer")},
			"/api/checklists/{checklist_id}":               {"get": op("getWekanChecklist", "Get checklist", params(path("checklist_id", "Checklist ID.")), "", "#/components/schemas/WekanObject", "wekanBearer")},
		},
	}
}

func zammadOverlay() overlaySpec {
	security := map[string]map[string]any{
		"zammadBasic": {"type": "http", "scheme": "basic", "description": "Zammad username and password supplied with HTTP Basic authentication."},
		"zammadToken": {"type": "apiKey", "in": "header", "name": "Authorization", "description": "Zammad access token carried in the Authorization header using Token token=... syntax."},
	}
	return overlaySpec{
		ProviderID:  "zammad",
		Title:       "Zammad API Advisory Overlay",
		Description: "Advisory OpenAPI overlay derived from official Zammad API human documentation. This is not an official Zammad OpenAPI document.",
		ServerURL:   "https://{zammad_host}",
		Sources:     []string{"https://docs.zammad.org/en/latest/api/intro.html", "https://docs.zammad.org/en/latest/api/ticket.html"},
		SourceNote:  "Zammad publishes human API documentation but no recorded stable public official OpenAPI document; this overlay covers selected tickets, users, groups, organizations, articles, attributes, and search endpoints.",
		Security:    security,
		Schemas:     []string{"ZammadObject", "ZammadCollection", "ZammadError"},
		OutputPath:  "catalog-openapi-cache/advisory-overlays/zammad-api-overlay.json",
		Paths: map[string]map[string]any{
			"/api/v1/tickets":                   {"get": op("listZammadTickets", "List tickets", params(query("page", "Page number."), query("per_page", "Page size.")), "", "#/components/schemas/ZammadCollection", "zammadToken"), "post": op("createZammadTicket", "Create ticket", nil, "#/components/schemas/ZammadObject", "#/components/schemas/ZammadObject", "zammadToken")},
			"/api/v1/tickets/{ticket_id}":       {"get": op("getZammadTicket", "Get ticket", params(path("ticket_id", "Ticket ID.")), "", "#/components/schemas/ZammadObject", "zammadToken"), "put": op("updateZammadTicket", "Update ticket", params(path("ticket_id", "Ticket ID.")), "#/components/schemas/ZammadObject", "#/components/schemas/ZammadObject", "zammadToken")},
			"/api/v1/users":                     {"get": op("listZammadUsers", "List users", nil, "", "#/components/schemas/ZammadCollection", "zammadToken"), "post": op("createZammadUser", "Create user", nil, "#/components/schemas/ZammadObject", "#/components/schemas/ZammadObject", "zammadToken")},
			"/api/v1/users/{user_id}":           {"get": op("getZammadUser", "Get user", params(path("user_id", "User ID.")), "", "#/components/schemas/ZammadObject", "zammadToken")},
			"/api/v1/groups":                    {"get": op("listZammadGroups", "List groups", nil, "", "#/components/schemas/ZammadCollection", "zammadToken")},
			"/api/v1/organizations":             {"get": op("listZammadOrganizations", "List organizations", nil, "", "#/components/schemas/ZammadCollection", "zammadToken")},
			"/api/v1/ticket_articles":           {"get": op("listZammadTicketArticles", "List ticket articles", nil, "", "#/components/schemas/ZammadCollection", "zammadToken"), "post": op("createZammadTicketArticle", "Create ticket article", nil, "#/components/schemas/ZammadObject", "#/components/schemas/ZammadObject", "zammadToken")},
			"/api/v1/object_manager_attributes": {"get": op("listZammadObjectManagerAttributes", "List object attributes", nil, "", "#/components/schemas/ZammadCollection", "zammadToken")},
			"/api/v1/search":                    {"get": op("searchZammad", "Search Zammad", params(query("query", "Search query.")), "", "#/components/schemas/ZammadCollection", "zammadToken")},
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
