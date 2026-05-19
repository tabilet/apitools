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
		autopilotOverlay(),
		driftOverlay(),
		freshworksCRMOverlay(),
		getResponseOverlay(),
		keapOverlay(),
		mailerLiteOverlay(),
		mauticOverlay(),
		monicaCRMOverlay(),
		salesmateOverlay(),
		sendyOverlay(),
		veroOverlay(),
	} {
		write(spec.OutputPath, build(spec))
	}
}

func autopilotOverlay() overlaySpec {
	security := map[string]map[string]any{
		"autopilotAPIKey": {"type": "apiKey", "in": "header", "name": "autopilotapikey", "description": "Autopilot API key carried in the autopilotapikey header."},
	}
	return overlaySpec{
		ProviderID:  "autopilot",
		Title:       "Autopilot API Advisory Overlay",
		Description: "Advisory OpenAPI overlay derived from official Autopilot human documentation. This is not an official Autopilot OpenAPI document.",
		ServerURL:   "https://api2.autopilothq.com/v1",
		Sources:     []string{"https://autopilot.docs.apiary.io/", "https://help.ortto.com/a-376-autopilot-how-to-use-autopilots-api"},
		SourceNote:  "Autopilot publishes API Blueprint-style and help documentation with an autopilotapikey header but no recorded official OpenAPI document; this overlay covers selected contact and list endpoints.",
		Security:    security,
		Schemas:     []string{"AutopilotContact", "AutopilotCollection", "AutopilotError"},
		OutputPath:  "catalog-openapi-cache/advisory-overlays/autopilot-api-overlay.json",
		Paths: map[string]map[string]any{
			"/contacts":                  {"get": op("listAutopilotContacts", "List contacts", params(query("page", "Page number."), query("limit", "Page size.")), "", "#/components/schemas/AutopilotCollection", "autopilotAPIKey"), "post": op("createAutopilotContact", "Create or add a contact", nil, "#/components/schemas/AutopilotContact", "#/components/schemas/AutopilotContact", "autopilotAPIKey")},
			"/contacts/{contact_id}":     {"get": op("getAutopilotContact", "Get contact", params(path("contact_id", "Autopilot contact ID.")), "", "#/components/schemas/AutopilotContact", "autopilotAPIKey"), "delete": op("deleteAutopilotContact", "Delete contact", params(path("contact_id", "Autopilot contact ID.")), "", "", "autopilotAPIKey")},
			"/contact/{contact_id}/list": {"post": op("addAutopilotContactToList", "Add contact to list", params(path("contact_id", "Autopilot contact ID.")), "#/components/schemas/AutopilotContact", "#/components/schemas/AutopilotContact", "autopilotAPIKey")},
			"/lists":                     {"get": op("listAutopilotLists", "List lists", nil, "", "#/components/schemas/AutopilotCollection", "autopilotAPIKey")},
		},
	}
}

func driftOverlay() overlaySpec {
	security := map[string]map[string]any{
		"driftBearer": {"type": "http", "scheme": "bearer", "bearerFormat": "OAuth 2.0 access token", "description": "Drift OAuth access token carried in the Authorization bearer header."},
	}
	return overlaySpec{
		ProviderID:  "drift",
		Title:       "Drift Platform API Advisory Overlay",
		Description: "Advisory OpenAPI overlay derived from official Drift human documentation. This is not an official Drift OpenAPI document.",
		ServerURL:   "https://driftapi.com",
		Sources:     []string{"https://devdocs.drift.com/docs/platform-apis", "https://devdocs.drift.com/docs/authentication-and-scopes"},
		SourceNote:  "Drift publishes REST-shaped platform API docs and OAuth scope guidance but no recorded official OpenAPI document; this overlay covers selected contacts, conversations, and users endpoints.",
		Security:    security,
		Schemas:     []string{"DriftObject", "DriftCollection", "DriftError"},
		OutputPath:  "catalog-openapi-cache/advisory-overlays/drift-platform-api-overlay.json",
		Paths: map[string]map[string]any{
			"/contacts":                        {"get": op("listDriftContacts", "List contacts", params(query("email", "Contact email filter."), query("limit", "Page size.")), "", "#/components/schemas/DriftCollection", "driftBearer")},
			"/contacts/{contact_id}":           {"get": op("getDriftContact", "Get contact", params(path("contact_id", "Drift contact ID.")), "", "#/components/schemas/DriftObject", "driftBearer"), "patch": op("updateDriftContact", "Update contact", params(path("contact_id", "Drift contact ID.")), "#/components/schemas/DriftObject", "#/components/schemas/DriftObject", "driftBearer")},
			"/conversations/{conversation_id}": {"get": op("getDriftConversation", "Get conversation", params(path("conversation_id", "Drift conversation ID.")), "", "#/components/schemas/DriftObject", "driftBearer")},
			"/users":                           {"get": op("listDriftUsers", "List users", nil, "", "#/components/schemas/DriftCollection", "driftBearer")},
		},
	}
}

func freshworksCRMOverlay() overlaySpec {
	security := map[string]map[string]any{
		"freshworksCRMToken": {"type": "apiKey", "in": "header", "name": "Authorization", "description": "Freshworks CRM token carried in the Authorization header using documented token syntax."},
	}
	return overlaySpec{
		ProviderID:  "freshworks-crm",
		Title:       "Freshworks CRM API Advisory Overlay",
		Description: "Advisory OpenAPI overlay derived from official Freshworks CRM human documentation. This is not an official Freshworks CRM OpenAPI document.",
		ServerURL:   "https://{domain}.myfreshworks.com/crm/sales/api",
		Sources:     []string{"https://developers.freshworks.com/crm/api/"},
		SourceNote:  "Freshworks CRM publishes REST API docs with account-domain URLs and token authorization but no recorded official OpenAPI document; this overlay covers selected contacts, accounts, deals, and tasks endpoints.",
		Security:    security,
		Schemas:     []string{"FreshworksCRMObject", "FreshworksCRMCollection", "FreshworksCRMError"},
		OutputPath:  "catalog-openapi-cache/advisory-overlays/freshworks-crm-api-overlay.json",
		Paths: map[string]map[string]any{
			"/contacts":       {"get": op("listFreshworksCRMContacts", "List contacts", params(query("page", "Page number."), query("per_page", "Page size.")), "", "#/components/schemas/FreshworksCRMCollection", "freshworksCRMToken"), "post": op("createFreshworksCRMContact", "Create contact", nil, "#/components/schemas/FreshworksCRMObject", "#/components/schemas/FreshworksCRMObject", "freshworksCRMToken")},
			"/contacts/{id}":  {"get": op("getFreshworksCRMContact", "Get contact", params(path("id", "Contact ID.")), "", "#/components/schemas/FreshworksCRMObject", "freshworksCRMToken"), "put": op("updateFreshworksCRMContact", "Update contact", params(path("id", "Contact ID.")), "#/components/schemas/FreshworksCRMObject", "#/components/schemas/FreshworksCRMObject", "freshworksCRMToken")},
			"/sales_accounts": {"get": op("listFreshworksCRMAccounts", "List sales accounts", nil, "", "#/components/schemas/FreshworksCRMCollection", "freshworksCRMToken")},
			"/deals":          {"get": op("listFreshworksCRMDeals", "List deals", nil, "", "#/components/schemas/FreshworksCRMCollection", "freshworksCRMToken")},
			"/tasks":          {"get": op("listFreshworksCRMTasks", "List tasks", nil, "", "#/components/schemas/FreshworksCRMCollection", "freshworksCRMToken")},
		},
	}
}

func getResponseOverlay() overlaySpec {
	security := map[string]map[string]any{
		"getResponseAPIKey": {"type": "apiKey", "in": "header", "name": "X-Auth-Token", "description": "GetResponse API key carried in the X-Auth-Token header."},
	}
	return overlaySpec{
		ProviderID:  "getresponse",
		Title:       "GetResponse API v3 Advisory Overlay",
		Description: "Advisory OpenAPI overlay derived from official GetResponse human documentation. This is not an official GetResponse OpenAPI document.",
		ServerURL:   "https://api.getresponse.com/v3",
		Sources:     []string{"https://apidocs.getresponse.com/v3", "https://apidocs.getresponse.com/v3/authentication"},
		SourceNote:  "GetResponse publishes REST API v3 docs with API key and OAuth authentication guidance but no recorded official OpenAPI document; this overlay covers selected campaigns, contacts, newsletters, and tags endpoints.",
		Security:    security,
		Schemas:     []string{"GetResponseObject", "GetResponseCollection", "GetResponseError"},
		OutputPath:  "catalog-openapi-cache/advisory-overlays/getresponse-api-v3-overlay.json",
		Paths: map[string]map[string]any{
			"/campaigns":     {"get": op("listGetResponseCampaigns", "List campaigns", nil, "", "#/components/schemas/GetResponseCollection", "getResponseAPIKey")},
			"/contacts":      {"get": op("listGetResponseContacts", "List contacts", params(query("query[email]", "Contact email filter."), query("page", "Page number."), query("perPage", "Page size.")), "", "#/components/schemas/GetResponseCollection", "getResponseAPIKey"), "post": op("createGetResponseContact", "Create contact", nil, "#/components/schemas/GetResponseObject", "#/components/schemas/GetResponseObject", "getResponseAPIKey")},
			"/contacts/{id}": {"get": op("getGetResponseContact", "Get contact", params(path("id", "Contact ID.")), "", "#/components/schemas/GetResponseObject", "getResponseAPIKey"), "delete": op("deleteGetResponseContact", "Delete contact", params(path("id", "Contact ID.")), "", "", "getResponseAPIKey")},
			"/newsletters":   {"get": op("listGetResponseNewsletters", "List newsletters", nil, "", "#/components/schemas/GetResponseCollection", "getResponseAPIKey")},
			"/tags":          {"get": op("listGetResponseTags", "List tags", nil, "", "#/components/schemas/GetResponseCollection", "getResponseAPIKey")},
		},
	}
}

func keapOverlay() overlaySpec {
	security := map[string]map[string]any{
		"keapBearer": {"type": "http", "scheme": "bearer", "bearerFormat": "OAuth 2.0 access token", "description": "Keap OAuth access token carried in the Authorization bearer header."},
	}
	return overlaySpec{
		ProviderID:  "keap",
		Title:       "Keap REST API Advisory Overlay",
		Description: "Advisory OpenAPI overlay derived from official Keap human documentation. This is not an official Keap OpenAPI document.",
		ServerURL:   "https://api.infusionsoft.com/crm/rest/v1",
		Sources:     []string{"https://developer.keap.com/docs/rest/1000/"},
		SourceNote:  "Keap publishes REST docs with OAuth authentication guidance but no recorded official OpenAPI document; this overlay covers selected contacts, companies, orders, and tags endpoints.",
		Security:    security,
		Schemas:     []string{"KeapObject", "KeapCollection", "KeapError"},
		OutputPath:  "catalog-openapi-cache/advisory-overlays/keap-rest-api-overlay.json",
		Paths: map[string]map[string]any{
			"/contacts":      {"get": op("listKeapContacts", "List contacts", params(query("limit", "Page size."), query("offset", "Offset.")), "", "#/components/schemas/KeapCollection", "keapBearer"), "post": op("createKeapContact", "Create contact", nil, "#/components/schemas/KeapObject", "#/components/schemas/KeapObject", "keapBearer")},
			"/contacts/{id}": {"get": op("getKeapContact", "Get contact", params(path("id", "Contact ID.")), "", "#/components/schemas/KeapObject", "keapBearer"), "patch": op("updateKeapContact", "Update contact", params(path("id", "Contact ID.")), "#/components/schemas/KeapObject", "#/components/schemas/KeapObject", "keapBearer")},
			"/companies":     {"get": op("listKeapCompanies", "List companies", nil, "", "#/components/schemas/KeapCollection", "keapBearer")},
			"/orders":        {"get": op("listKeapOrders", "List orders", nil, "", "#/components/schemas/KeapCollection", "keapBearer")},
			"/tags":          {"get": op("listKeapTags", "List tags", nil, "", "#/components/schemas/KeapCollection", "keapBearer")},
		},
	}
}

func mailerLiteOverlay() overlaySpec {
	security := map[string]map[string]any{
		"mailerLiteBearer": {"type": "http", "scheme": "bearer", "bearerFormat": "API token", "description": "MailerLite API token carried in the Authorization bearer header."},
	}
	return overlaySpec{
		ProviderID:  "mailerlite",
		Title:       "MailerLite API Advisory Overlay",
		Description: "Advisory OpenAPI overlay derived from official MailerLite human documentation. This is not an official MailerLite OpenAPI document.",
		ServerURL:   "https://connect.mailerlite.com/api",
		Sources:     []string{"https://developers.mailerlite.com/docs/", "https://developers-classic.mailerlite.com/docs/authentication", "https://www.mailerlite.com/help/where-to-find-the-mailerlite-api-key-groupid-and-documentation"},
		SourceNote:  "MailerLite publishes REST API docs plus Classic auth docs but no recorded official OpenAPI document; this overlay covers selected subscriber, group, campaign, and form endpoints using the current bearer-token API.",
		Security:    security,
		Schemas:     []string{"MailerLiteObject", "MailerLiteCollection", "MailerLiteError"},
		OutputPath:  "catalog-openapi-cache/advisory-overlays/mailerlite-api-overlay.json",
		Paths: map[string]map[string]any{
			"/subscribers":      {"get": op("listMailerLiteSubscribers", "List subscribers", params(query("filter[email]", "Subscriber email filter."), query("limit", "Page size.")), "", "#/components/schemas/MailerLiteCollection", "mailerLiteBearer"), "post": op("createMailerLiteSubscriber", "Create subscriber", nil, "#/components/schemas/MailerLiteObject", "#/components/schemas/MailerLiteObject", "mailerLiteBearer")},
			"/subscribers/{id}": {"get": op("getMailerLiteSubscriber", "Get subscriber", params(path("id", "Subscriber ID or email.")), "", "#/components/schemas/MailerLiteObject", "mailerLiteBearer")},
			"/groups":           {"get": op("listMailerLiteGroups", "List groups", nil, "", "#/components/schemas/MailerLiteCollection", "mailerLiteBearer")},
			"/campaigns":        {"get": op("listMailerLiteCampaigns", "List campaigns", nil, "", "#/components/schemas/MailerLiteCollection", "mailerLiteBearer")},
			"/forms":            {"get": op("listMailerLiteForms", "List forms", nil, "", "#/components/schemas/MailerLiteCollection", "mailerLiteBearer")},
		},
	}
}

func mauticOverlay() overlaySpec {
	security := map[string]map[string]any{
		"mauticBearer": {"type": "http", "scheme": "bearer", "bearerFormat": "OAuth 2.0 access token", "description": "Mautic OAuth access token carried in the Authorization bearer header."},
	}
	return overlaySpec{
		ProviderID:  "mautic",
		Title:       "Mautic API Advisory Overlay",
		Description: "Advisory OpenAPI overlay derived from official Mautic human documentation. This is not an official Mautic OpenAPI document.",
		ServerURL:   "https://{mautic_host}/api",
		Sources:     []string{"https://developer.mautic.org/", "https://kb.mautic.org/article/what-is-mautic-039%3Bs-api.html"},
		SourceNote:  "Mautic publishes API docs for instance-hosted REST endpoints with OAuth and Basic authentication options but no recorded official OpenAPI document; this overlay covers selected contacts, companies, segments, campaigns, and emails endpoints.",
		Security:    security,
		Schemas:     []string{"MauticObject", "MauticCollection", "MauticError"},
		OutputPath:  "catalog-openapi-cache/advisory-overlays/mautic-api-overlay.json",
		Paths: map[string]map[string]any{
			"/contacts":      {"get": op("listMauticContacts", "List contacts", params(query("search", "Search query."), query("limit", "Page size."), query("start", "Offset.")), "", "#/components/schemas/MauticCollection", "mauticBearer"), "post": op("createMauticContact", "Create contact", nil, "#/components/schemas/MauticObject", "#/components/schemas/MauticObject", "mauticBearer")},
			"/contacts/{id}": {"get": op("getMauticContact", "Get contact", params(path("id", "Contact ID.")), "", "#/components/schemas/MauticObject", "mauticBearer"), "patch": op("updateMauticContact", "Update contact", params(path("id", "Contact ID.")), "#/components/schemas/MauticObject", "#/components/schemas/MauticObject", "mauticBearer")},
			"/companies":     {"get": op("listMauticCompanies", "List companies", nil, "", "#/components/schemas/MauticCollection", "mauticBearer")},
			"/segments":      {"get": op("listMauticSegments", "List segments", nil, "", "#/components/schemas/MauticCollection", "mauticBearer")},
			"/campaigns":     {"get": op("listMauticCampaigns", "List campaigns", nil, "", "#/components/schemas/MauticCollection", "mauticBearer")},
			"/emails":        {"get": op("listMauticEmails", "List emails", nil, "", "#/components/schemas/MauticCollection", "mauticBearer")},
		},
	}
}

func monicaCRMOverlay() overlaySpec {
	security := map[string]map[string]any{
		"monicaBearer": {"type": "http", "scheme": "bearer", "bearerFormat": "Personal access token", "description": "Monica API token carried in the Authorization bearer header."},
	}
	return overlaySpec{
		ProviderID:  "monica-crm",
		Title:       "Monica CRM API Advisory Overlay",
		Description: "Advisory OpenAPI overlay derived from official Monica human documentation. This is not an official Monica OpenAPI document.",
		ServerURL:   "https://app.monicahq.com/api",
		Sources:     []string{"https://www.monicahq.com/api"},
		SourceNote:  "Monica publishes REST API docs with bearer token guidance but no recorded official OpenAPI document; this overlay covers selected contacts, activities, calls, reminders, and tags endpoints.",
		Security:    security,
		Schemas:     []string{"MonicaCRMObject", "MonicaCRMCollection", "MonicaCRMError"},
		OutputPath:  "catalog-openapi-cache/advisory-overlays/monica-crm-api-overlay.json",
		Paths: map[string]map[string]any{
			"/contacts":      {"get": op("listMonicaCRMContacts", "List contacts", params(query("page", "Page number.")), "", "#/components/schemas/MonicaCRMCollection", "monicaBearer"), "post": op("createMonicaCRMContact", "Create contact", nil, "#/components/schemas/MonicaCRMObject", "#/components/schemas/MonicaCRMObject", "monicaBearer")},
			"/contacts/{id}": {"get": op("getMonicaCRMContact", "Get contact", params(path("id", "Contact ID.")), "", "#/components/schemas/MonicaCRMObject", "monicaBearer"), "put": op("updateMonicaCRMContact", "Update contact", params(path("id", "Contact ID.")), "#/components/schemas/MonicaCRMObject", "#/components/schemas/MonicaCRMObject", "monicaBearer")},
			"/activities":    {"get": op("listMonicaCRMActivities", "List activities", nil, "", "#/components/schemas/MonicaCRMCollection", "monicaBearer")},
			"/calls":         {"get": op("listMonicaCRMCalls", "List calls", nil, "", "#/components/schemas/MonicaCRMCollection", "monicaBearer")},
			"/reminders":     {"get": op("listMonicaCRMReminders", "List reminders", nil, "", "#/components/schemas/MonicaCRMCollection", "monicaBearer")},
			"/tags":          {"get": op("listMonicaCRMTags", "List tags", nil, "", "#/components/schemas/MonicaCRMCollection", "monicaBearer")},
		},
	}
}

func salesmateOverlay() overlaySpec {
	security := map[string]map[string]any{
		"salesmateSessionToken": {"type": "apiKey", "in": "header", "name": "sessionToken", "description": "Salesmate session token carried in the sessionToken header."},
		"salesmateLinkName":     {"type": "apiKey", "in": "header", "name": "x-linkname", "description": "Salesmate account link name carried in the x-linkname header."},
	}
	return overlaySpec{
		ProviderID:  "salesmate",
		Title:       "Salesmate API Advisory Overlay",
		Description: "Advisory OpenAPI overlay derived from official Salesmate human documentation. This is not an official Salesmate OpenAPI document.",
		ServerURL:   "https://apis.salesmate.io/v4",
		Sources:     []string{"https://support.salesmate.io/hc/en-us/articles/12864176787609-What-Are-API-s-Usage-and-Working", "https://support.salesmate.io/hc/en-us/articles/360043653751-Auto-enroll-Contacts-to-sequence-whenever-a-Deal-stage-is-updated-via-Webhooks", "https://apidocs.salesmate.io/"},
		SourceNote:  "Salesmate publishes API usage/support docs and API docs requiring sessionToken plus x-linkname headers but no recorded official OpenAPI document; this overlay covers selected contacts, companies, deals, activities, and users endpoints.",
		Security:    security,
		Schemas:     []string{"SalesmateObject", "SalesmateCollection", "SalesmateError"},
		OutputPath:  "catalog-openapi-cache/advisory-overlays/salesmate-api-overlay.json",
		Paths: map[string]map[string]any{
			"/contact":      {"get": op("listSalesmateContacts", "List contacts", params(query("rows", "Page size."), query("from", "Offset.")), "", "#/components/schemas/SalesmateCollection", "salesmateSessionToken", "salesmateLinkName"), "post": op("createSalesmateContact", "Create contact", nil, "#/components/schemas/SalesmateObject", "#/components/schemas/SalesmateObject", "salesmateSessionToken", "salesmateLinkName")},
			"/contact/{id}": {"get": op("getSalesmateContact", "Get contact", params(path("id", "Contact ID.")), "", "#/components/schemas/SalesmateObject", "salesmateSessionToken", "salesmateLinkName")},
			"/company":      {"get": op("listSalesmateCompanies", "List companies", nil, "", "#/components/schemas/SalesmateCollection", "salesmateSessionToken", "salesmateLinkName")},
			"/deal":         {"get": op("listSalesmateDeals", "List deals", nil, "", "#/components/schemas/SalesmateCollection", "salesmateSessionToken", "salesmateLinkName")},
			"/activity":     {"get": op("listSalesmateActivities", "List activities", nil, "", "#/components/schemas/SalesmateCollection", "salesmateSessionToken", "salesmateLinkName")},
			"/users":        {"get": op("listSalesmateUsers", "List users", nil, "", "#/components/schemas/SalesmateCollection", "salesmateSessionToken", "salesmateLinkName")},
		},
	}
}

func sendyOverlay() overlaySpec {
	security := map[string]map[string]any{
		"sendyAPIKey": {"type": "apiKey", "in": "query", "name": "api_key", "description": "Sendy API key carried in documented request parameters."},
	}
	return overlaySpec{
		ProviderID:  "sendy",
		Title:       "Sendy API Advisory Overlay",
		Description: "Advisory OpenAPI overlay derived from official Sendy human documentation. This is not an official Sendy OpenAPI document.",
		ServerURL:   "https://{sendy_host}",
		Sources:     []string{"https://sendy.co/api"},
		SourceNote:  "Sendy publishes API docs for installation-hosted endpoints with api_key request parameters but no recorded official OpenAPI document; this overlay covers selected subscriber, list, campaign, and brand endpoints.",
		Security:    security,
		Schemas:     []string{"SendyObject", "SendyCollection", "SendyError"},
		OutputPath:  "catalog-openapi-cache/advisory-overlays/sendy-api-overlay.json",
		Paths: map[string]map[string]any{
			"/subscribe":   {"post": op("subscribeSendySubscriber", "Subscribe user", params(query("list", "List ID."), query("email", "Email address.")), "", "#/components/schemas/SendyObject", "sendyAPIKey")},
			"/unsubscribe": {"post": op("unsubscribeSendySubscriber", "Unsubscribe user", params(query("list", "List ID."), query("email", "Email address.")), "", "#/components/schemas/SendyObject", "sendyAPIKey")},
			"/api/subscribers/active-subscriber-count.php": {"post": op("getSendyActiveSubscriberCount", "Get active subscriber count", params(query("list_id", "List ID.")), "", "#/components/schemas/SendyObject", "sendyAPIKey")},
			"/api/lists/get-lists.php":                     {"post": op("listSendyLists", "List lists", params(query("brand_id", "Brand ID.")), "", "#/components/schemas/SendyCollection", "sendyAPIKey")},
			"/api/campaigns/create.php":                    {"post": op("createSendyCampaign", "Create campaign", nil, "#/components/schemas/SendyObject", "#/components/schemas/SendyObject", "sendyAPIKey")},
		},
	}
}

func veroOverlay() overlaySpec {
	security := map[string]map[string]any{
		"veroAuthToken": {"type": "apiKey", "in": "query", "name": "auth_token", "description": "Vero auth token carried in documented request parameters."},
	}
	return overlaySpec{
		ProviderID:  "vero",
		Title:       "Vero Track API Advisory Overlay",
		Description: "Advisory OpenAPI overlay derived from official Vero human documentation. This is not an official Vero OpenAPI document.",
		ServerURL:   "https://api.getvero.com/api/v2",
		Sources:     []string{"https://help.getvero.com/developer-docs/overview", "https://help.getvero.com/api-reference/track/overview", "https://help.getvero.com/api-reference/events/track"},
		SourceNote:  "Vero publishes Track API reference docs with auth_token request parameters but no recorded official OpenAPI document; this overlay covers selected users, events, tags, and unsubscribe endpoints.",
		Security:    security,
		Schemas:     []string{"VeroObject", "VeroError"},
		OutputPath:  "catalog-openapi-cache/advisory-overlays/vero-track-api-overlay.json",
		Paths: map[string]map[string]any{
			"/users/track":       {"post": op("trackVeroUser", "Track or update user", nil, "#/components/schemas/VeroObject", "#/components/schemas/VeroObject", "veroAuthToken")},
			"/events/track":      {"post": op("trackVeroEvent", "Track event", nil, "#/components/schemas/VeroObject", "#/components/schemas/VeroObject", "veroAuthToken")},
			"/users/tags/edit":   {"post": op("editVeroUserTags", "Edit user tags", nil, "#/components/schemas/VeroObject", "#/components/schemas/VeroObject", "veroAuthToken")},
			"/users/unsubscribe": {"post": op("unsubscribeVeroUser", "Unsubscribe user", nil, "#/components/schemas/VeroObject", "#/components/schemas/VeroObject", "veroAuthToken")},
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
			"200": response(responseRef),
			"default": map[string]any{
				"description": "Provider error response.",
			},
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

func params(items ...map[string]any) []map[string]any {
	return items
}

func path(name, description string) map[string]any {
	return parameter(name, "path", description, true)
}

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
