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
	ServerVars  map[string]map[string]any
	Sources     []string
	SourceNote  string
	Security    map[string]map[string]any
	Schemas     []string
	Paths       map[string]map[string]any
	OutputPath  string
}

func main() {
	for _, spec := range []overlaySpec{
		actionNetworkOverlay(),
		adaloOverlay(),
		affinityOverlay(),
		agileCRMOverlay(),
		bannerbearOverlay(),
		gristOverlay(),
	} {
		write(spec.OutputPath, build(spec))
	}
}

func actionNetworkOverlay() overlaySpec {
	security := map[string]map[string]any{
		"actionNetworkAPIToken": {"type": "apiKey", "in": "header", "name": "OSDI-API-Token", "description": "Action Network API key carried in the OSDI-API-Token header for most API v2 endpoints."},
	}
	return overlaySpec{
		ProviderID:  "action-network",
		Title:       "Action Network API v2 Advisory Overlay",
		Description: "Advisory OpenAPI overlay derived from official Action Network API v2 human documentation. This is not an official Action Network OpenAPI document.",
		ServerURL:   "https://actionnetwork.org/api/v2",
		Sources:     []string{"https://actionnetwork.org/docs/v2/", "https://actionnetwork.org/docs/v2/webhooks"},
		SourceNote:  "Action Network publishes OSDI/HAL+JSON REST API v2 docs with OSDI-API-Token header metadata; this overlay covers the API entry point, people, petitions, forms, events, tags, taggings, and webhooks.",
		Security:    security,
		Schemas:     []string{"ActionNetworkObject", "ActionNetworkCollection", "ActionNetworkError"},
		OutputPath:  "catalog-openapi-cache/advisory-overlays/action-network-api-v2-overlay.json",
		Paths: map[string]map[string]any{
			"/": {"get": op("getActionNetworkAPIEntryPoint", "Get API entry point", nil, "", "#/components/schemas/ActionNetworkObject", "actionNetworkAPIToken")},
			"/people": {
				"get":  op("listActionNetworkPeople", "List people", params(query("page", "Page number."), query("filter", "OData filter expression.")), "", "#/components/schemas/ActionNetworkCollection", "actionNetworkAPIToken"),
				"post": op("createActionNetworkPerson", "Create a person", nil, "#/components/schemas/ActionNetworkObject", "#/components/schemas/ActionNetworkObject", "actionNetworkAPIToken"),
			},
			"/people/{person_id}": {
				"get": op("getActionNetworkPerson", "Get a person", params(path("person_id", "Action Network person ID.")), "", "#/components/schemas/ActionNetworkObject", "actionNetworkAPIToken"),
				"put": op("updateActionNetworkPerson", "Update a person", params(path("person_id", "Action Network person ID.")), "#/components/schemas/ActionNetworkObject", "#/components/schemas/ActionNetworkObject", "actionNetworkAPIToken"),
			},
			"/petitions": {
				"get":  op("listActionNetworkPetitions", "List petitions", params(query("page", "Page number."), query("filter", "OData filter expression.")), "", "#/components/schemas/ActionNetworkCollection", "actionNetworkAPIToken"),
				"post": op("createActionNetworkPetition", "Create a petition", nil, "#/components/schemas/ActionNetworkObject", "#/components/schemas/ActionNetworkObject", "actionNetworkAPIToken"),
			},
			"/petitions/{petition_id}": {"get": op("getActionNetworkPetition", "Get a petition", params(path("petition_id", "Action Network petition ID.")), "", "#/components/schemas/ActionNetworkObject", "actionNetworkAPIToken")},
			"/forms": {
				"get":  op("listActionNetworkForms", "List forms", params(query("page", "Page number."), query("filter", "OData filter expression.")), "", "#/components/schemas/ActionNetworkCollection", "actionNetworkAPIToken"),
				"post": op("createActionNetworkForm", "Create a form", nil, "#/components/schemas/ActionNetworkObject", "#/components/schemas/ActionNetworkObject", "actionNetworkAPIToken"),
			},
			"/forms/{form_id}": {"get": op("getActionNetworkForm", "Get a form", params(path("form_id", "Action Network form ID.")), "", "#/components/schemas/ActionNetworkObject", "actionNetworkAPIToken")},
			"/events": {
				"get":  op("listActionNetworkEvents", "List events", params(query("page", "Page number."), query("filter", "OData filter expression.")), "", "#/components/schemas/ActionNetworkCollection", "actionNetworkAPIToken"),
				"post": op("createActionNetworkEvent", "Create an event", nil, "#/components/schemas/ActionNetworkObject", "#/components/schemas/ActionNetworkObject", "actionNetworkAPIToken"),
			},
			"/events/{event_id}": {"get": op("getActionNetworkEvent", "Get an event", params(path("event_id", "Action Network event ID.")), "", "#/components/schemas/ActionNetworkObject", "actionNetworkAPIToken")},
			"/tags":              {"get": op("listActionNetworkTags", "List tags", params(query("page", "Page number."), query("filter", "OData filter expression.")), "", "#/components/schemas/ActionNetworkCollection", "actionNetworkAPIToken")},
			"/taggings": {
				"get":  op("listActionNetworkTaggings", "List taggings", params(query("page", "Page number."), query("filter", "OData filter expression.")), "", "#/components/schemas/ActionNetworkCollection", "actionNetworkAPIToken"),
				"post": op("createActionNetworkTagging", "Create a tagging", nil, "#/components/schemas/ActionNetworkObject", "#/components/schemas/ActionNetworkObject", "actionNetworkAPIToken"),
			},
			"/webhooks": {"get": op("listActionNetworkWebhooks", "List webhooks", nil, "", "#/components/schemas/ActionNetworkCollection", "actionNetworkAPIToken")},
		},
	}
}

func adaloOverlay() overlaySpec {
	security := map[string]map[string]any{
		"adaloBearer": {"type": "http", "scheme": "bearer", "description": "Adalo app API key carried as an Authorization bearer token."},
	}
	return overlaySpec{
		ProviderID:  "adalo",
		Title:       "Adalo API Advisory Overlay",
		Description: "Advisory OpenAPI overlay derived from official Adalo API human documentation. This is not an official Adalo OpenAPI document.",
		ServerURL:   "https://api.adalo.com",
		Sources:     []string{"https://help.adalo.com/integrations/the-adalo-api", "https://help.adalo.com/integrations/the-adalo-api/collections", "https://help.adalo.com/integrations/the-adalo-api/push-notifications"},
		SourceNote:  "Adalo publishes human API docs for app-specific Collections API endpoints and push notifications with bearer API-key metadata; this overlay covers variable app, collection, record, and notification endpoints.",
		Security:    security,
		Schemas:     []string{"AdaloRecord", "AdaloCollection", "AdaloNotificationRequest", "AdaloObject", "AdaloError"},
		OutputPath:  "catalog-openapi-cache/advisory-overlays/adalo-api-overlay.json",
		Paths: map[string]map[string]any{
			"/v0/apps/{app_id}/collections/{collection_id}/records": {
				"get":  op("listAdaloCollectionRecords", "List collection records", params(path("app_id", "Adalo app ID."), path("collection_id", "Adalo collection ID."), query("filterKey", "Collection field key for simple filtering."), query("filterValue", "Filter value.")), "", "#/components/schemas/AdaloCollection", "adaloBearer"),
				"post": op("createAdaloCollectionRecord", "Create a collection record", params(path("app_id", "Adalo app ID."), path("collection_id", "Adalo collection ID.")), "#/components/schemas/AdaloRecord", "#/components/schemas/AdaloRecord", "adaloBearer"),
			},
			"/v0/apps/{app_id}/collections/{collection_id}/records/{record_id}": {
				"get":    op("getAdaloCollectionRecord", "Get a collection record", params(path("app_id", "Adalo app ID."), path("collection_id", "Adalo collection ID."), path("record_id", "Adalo record ID.")), "", "#/components/schemas/AdaloRecord", "adaloBearer"),
				"put":    op("replaceAdaloCollectionRecord", "Replace a collection record", params(path("app_id", "Adalo app ID."), path("collection_id", "Adalo collection ID."), path("record_id", "Adalo record ID.")), "#/components/schemas/AdaloRecord", "#/components/schemas/AdaloRecord", "adaloBearer"),
				"patch":  op("updateAdaloCollectionRecord", "Update a collection record", params(path("app_id", "Adalo app ID."), path("collection_id", "Adalo collection ID."), path("record_id", "Adalo record ID.")), "#/components/schemas/AdaloRecord", "#/components/schemas/AdaloRecord", "adaloBearer"),
				"delete": op("deleteAdaloCollectionRecord", "Delete a collection record", params(path("app_id", "Adalo app ID."), path("collection_id", "Adalo collection ID."), path("record_id", "Adalo record ID.")), "", "#/components/schemas/AdaloObject", "adaloBearer"),
			},
			"/notifications": {"post": op("sendAdaloPushNotification", "Send a push notification", nil, "#/components/schemas/AdaloNotificationRequest", "#/components/schemas/AdaloObject", "adaloBearer")},
		},
	}
}

func affinityOverlay() overlaySpec {
	security := map[string]map[string]any{
		"affinityBasic":  {"type": "http", "scheme": "basic", "description": "Affinity API key supplied as the HTTP Basic password with no username."},
		"affinityBearer": {"type": "http", "scheme": "bearer", "description": "Affinity API key supplied as an Authorization bearer token."},
	}
	return overlaySpec{
		ProviderID:  "affinity",
		Title:       "Affinity V1 API Advisory Overlay",
		Description: "Advisory OpenAPI overlay derived from official Affinity V1 API human documentation. This is not an official Affinity OpenAPI document.",
		ServerURL:   "https://api.affinity.co",
		Sources:     []string{"https://api-docs.affinity.co/", "https://support.affinity.co/s/article/How-to-Create-and-Manage-API-Keys"},
		SourceNote:  "Affinity publishes human V1 API docs with HTTP Basic and bearer API-key metadata; this overlay covers authentication, rate limits, lists, fields, field values, people, organizations, opportunities, interactions, notes, files, and webhooks.",
		Security:    security,
		Schemas:     []string{"AffinityObject", "AffinityCollection", "AffinityError"},
		OutputPath:  "catalog-openapi-cache/advisory-overlays/affinity-v1-api-overlay.json",
		Paths: map[string]map[string]any{
			"/auth/whoami":                  {"get": op("getAffinityWhoami", "Get authentication metadata", nil, "", "#/components/schemas/AffinityObject", "affinityBearer")},
			"/rate-limit":                   {"get": op("getAffinityRateLimit", "Get rate limit usage", nil, "", "#/components/schemas/AffinityObject", "affinityBearer")},
			"/lists":                        {"get": op("listAffinityLists", "List lists", nil, "", "#/components/schemas/AffinityCollection", "affinityBearer"), "post": op("createAffinityList", "Create a list", nil, "#/components/schemas/AffinityObject", "#/components/schemas/AffinityObject", "affinityBearer")},
			"/lists/{list_id}":              {"get": op("getAffinityList", "Get a list", params(path("list_id", "Affinity list ID.")), "", "#/components/schemas/AffinityObject", "affinityBearer")},
			"/lists/{list_id}/list-entries": {"get": op("listAffinityListEntries", "List list entries", params(path("list_id", "Affinity list ID."), query("page_token", "Pagination token.")), "", "#/components/schemas/AffinityCollection", "affinityBearer"), "post": op("createAffinityListEntry", "Create a list entry", params(path("list_id", "Affinity list ID.")), "#/components/schemas/AffinityObject", "#/components/schemas/AffinityObject", "affinityBearer")},
			"/fields":                       {"get": op("listAffinityFields", "List fields", nil, "", "#/components/schemas/AffinityCollection", "affinityBearer"), "post": op("createAffinityField", "Create a field", nil, "#/components/schemas/AffinityObject", "#/components/schemas/AffinityObject", "affinityBearer")},
			"/field-values":                 {"get": op("listAffinityFieldValues", "List field values", nil, "", "#/components/schemas/AffinityCollection", "affinityBearer"), "post": op("createAffinityFieldValue", "Create a field value", nil, "#/components/schemas/AffinityObject", "#/components/schemas/AffinityObject", "affinityBearer")},
			"/persons":                      {"get": op("listAffinityPersons", "List persons", params(query("term", "Search term."), query("page_token", "Pagination token.")), "", "#/components/schemas/AffinityCollection", "affinityBearer"), "post": op("createAffinityPerson", "Create a person", nil, "#/components/schemas/AffinityObject", "#/components/schemas/AffinityObject", "affinityBearer")},
			"/persons/{person_id}":          {"get": op("getAffinityPerson", "Get a person", params(path("person_id", "Affinity person ID.")), "", "#/components/schemas/AffinityObject", "affinityBearer"), "put": op("updateAffinityPerson", "Update a person", params(path("person_id", "Affinity person ID.")), "#/components/schemas/AffinityObject", "#/components/schemas/AffinityObject", "affinityBearer"), "delete": op("deleteAffinityPerson", "Delete a person", params(path("person_id", "Affinity person ID.")), "", "#/components/schemas/AffinityObject", "affinityBearer")},
			"/organizations":                {"get": op("listAffinityOrganizations", "List organizations", params(query("term", "Search term."), query("page_token", "Pagination token.")), "", "#/components/schemas/AffinityCollection", "affinityBearer"), "post": op("createAffinityOrganization", "Create an organization", nil, "#/components/schemas/AffinityObject", "#/components/schemas/AffinityObject", "affinityBearer")},
			"/opportunities":                {"get": op("listAffinityOpportunities", "List opportunities", params(query("term", "Search term."), query("page_token", "Pagination token.")), "", "#/components/schemas/AffinityCollection", "affinityBearer"), "post": op("createAffinityOpportunity", "Create an opportunity", nil, "#/components/schemas/AffinityObject", "#/components/schemas/AffinityObject", "affinityBearer")},
			"/interactions":                 {"get": op("listAffinityInteractions", "List interactions", nil, "", "#/components/schemas/AffinityCollection", "affinityBearer"), "post": op("createAffinityInteraction", "Create an interaction", nil, "#/components/schemas/AffinityObject", "#/components/schemas/AffinityObject", "affinityBearer")},
			"/notes":                        {"get": op("listAffinityNotes", "List notes", nil, "", "#/components/schemas/AffinityCollection", "affinityBearer"), "post": op("createAffinityNote", "Create a note", nil, "#/components/schemas/AffinityObject", "#/components/schemas/AffinityObject", "affinityBearer")},
			"/entity-files":                 {"get": op("listAffinityEntityFiles", "List entity files", params(query("page_token", "Pagination token.")), "", "#/components/schemas/AffinityCollection", "affinityBearer"), "post": op("createAffinityEntityFile", "Create an entity file", nil, "#/components/schemas/AffinityObject", "#/components/schemas/AffinityObject", "affinityBearer")},
			"/webhook":                      {"get": op("listAffinityWebhookSubscriptions", "List webhook subscriptions", nil, "", "#/components/schemas/AffinityCollection", "affinityBearer")},
			"/webhook/subscribe":            {"post": op("createAffinityWebhookSubscription", "Create a webhook subscription", nil, "#/components/schemas/AffinityObject", "#/components/schemas/AffinityObject", "affinityBearer")},
		},
	}
}

func agileCRMOverlay() overlaySpec {
	security := map[string]map[string]any{
		"agileCRMBasic": {"type": "http", "scheme": "basic", "description": "Agile CRM user email and REST API key supplied with HTTP Basic authentication."},
	}
	return overlaySpec{
		ProviderID:  "agile-crm",
		Title:       "Agile CRM REST API Advisory Overlay",
		Description: "Advisory OpenAPI overlay derived from official Agile CRM REST API human and GitHub documentation. This is not an official Agile CRM OpenAPI document.",
		ServerURL:   "https://{domain}.agilecrm.com/dev",
		ServerVars:  map[string]map[string]any{"domain": {"default": "example", "description": "Operator-supplied Agile CRM account subdomain."}},
		Sources:     []string{"https://www.agilecrm.com/api", "https://github.com/agilecrm/rest-api"},
		SourceNote:  "Agile CRM publishes REST API docs with account-subdomain base URLs and HTTP Basic email/API-key metadata; this overlay covers contacts, companies, deals, tasks, events, tracks, campaigns, documents, and help desk records.",
		Security:    security,
		Schemas:     []string{"AgileCRMObject", "AgileCRMCollection", "AgileCRMError"},
		OutputPath:  "catalog-openapi-cache/advisory-overlays/agile-crm-rest-api-overlay.json",
		Paths: map[string]map[string]any{
			"/api/contacts":                      {"get": op("listAgileCRMContacts", "List contacts", params(query("page_size", "Page size."), query("cursor", "Pagination cursor.")), "", "#/components/schemas/AgileCRMCollection", "agileCRMBasic"), "post": op("createAgileCRMContact", "Create a contact or company", nil, "#/components/schemas/AgileCRMObject", "#/components/schemas/AgileCRMObject", "agileCRMBasic")},
			"/api/contacts/{id}":                 {"get": op("getAgileCRMContact", "Get a contact or company", params(path("id", "Agile CRM contact or company ID.")), "", "#/components/schemas/AgileCRMObject", "agileCRMBasic"), "delete": op("deleteAgileCRMContact", "Delete a contact or company", params(path("id", "Agile CRM contact or company ID.")), "", "#/components/schemas/AgileCRMObject", "agileCRMBasic")},
			"/api/contacts/edit-properties":      {"put": op("updateAgileCRMContactProperties", "Update contact or company properties", nil, "#/components/schemas/AgileCRMObject", "#/components/schemas/AgileCRMObject", "agileCRMBasic")},
			"/api/contacts/search/email/{email}": {"get": op("searchAgileCRMContactByEmail", "Search contact by email", params(path("email", "Contact email address.")), "", "#/components/schemas/AgileCRMObject", "agileCRMBasic")},
			"/api/search":                        {"get": op("searchAgileCRMContactsCompanies", "Search contacts and companies", params(query("q", "Search query."), query("page_size", "Page size."), query("type", "Contact type.")), "", "#/components/schemas/AgileCRMCollection", "agileCRMBasic")},
			"/api/opportunity":                   {"get": op("listAgileCRMDeals", "List deals", params(query("page_size", "Page size."), query("cursor", "Pagination cursor.")), "", "#/components/schemas/AgileCRMCollection", "agileCRMBasic"), "post": op("createAgileCRMDeal", "Create a deal", nil, "#/components/schemas/AgileCRMObject", "#/components/schemas/AgileCRMObject", "agileCRMBasic")},
			"/api/opportunity/{id}":              {"get": op("getAgileCRMDeal", "Get a deal", params(path("id", "Agile CRM deal ID.")), "", "#/components/schemas/AgileCRMObject", "agileCRMBasic"), "delete": op("deleteAgileCRMDeal", "Delete a deal", params(path("id", "Agile CRM deal ID.")), "", "#/components/schemas/AgileCRMObject", "agileCRMBasic")},
			"/api/opportunity/partial-update":    {"put": op("updateAgileCRMDeal", "Update a deal", nil, "#/components/schemas/AgileCRMObject", "#/components/schemas/AgileCRMObject", "agileCRMBasic")},
			"/api/tasks":                         {"get": op("listAgileCRMTasks", "List tasks", nil, "", "#/components/schemas/AgileCRMCollection", "agileCRMBasic"), "post": op("createAgileCRMTask", "Create a task", nil, "#/components/schemas/AgileCRMObject", "#/components/schemas/AgileCRMObject", "agileCRMBasic")},
			"/api/tasks/{id}":                    {"get": op("getAgileCRMTask", "Get a task", params(path("id", "Agile CRM task ID.")), "", "#/components/schemas/AgileCRMObject", "agileCRMBasic"), "delete": op("deleteAgileCRMTask", "Delete a task", params(path("id", "Agile CRM task ID.")), "", "#/components/schemas/AgileCRMObject", "agileCRMBasic")},
			"/api/events":                        {"get": op("listAgileCRMEvents", "List events", nil, "", "#/components/schemas/AgileCRMCollection", "agileCRMBasic"), "post": op("createAgileCRMEvent", "Create an event", nil, "#/components/schemas/AgileCRMObject", "#/components/schemas/AgileCRMObject", "agileCRMBasic")},
			"/api/events/{id}":                   {"delete": op("deleteAgileCRMEvent", "Delete an event", params(path("id", "Agile CRM event ID.")), "", "#/components/schemas/AgileCRMObject", "agileCRMBasic")},
			"/api/tracks":                        {"get": op("listAgileCRMTracks", "List tracks", nil, "", "#/components/schemas/AgileCRMCollection", "agileCRMBasic"), "post": op("createAgileCRMTrack", "Create a track", nil, "#/components/schemas/AgileCRMObject", "#/components/schemas/AgileCRMObject", "agileCRMBasic")},
			"/api/campaigns":                     {"get": op("listAgileCRMCampaigns", "List campaigns", nil, "", "#/components/schemas/AgileCRMCollection", "agileCRMBasic")},
			"/api/tickets":                       {"get": op("listAgileCRMHelpDeskTickets", "List help desk tickets", nil, "", "#/components/schemas/AgileCRMCollection", "agileCRMBasic"), "post": op("createAgileCRMHelpDeskTicket", "Create a help desk ticket", nil, "#/components/schemas/AgileCRMObject", "#/components/schemas/AgileCRMObject", "agileCRMBasic")},
		},
	}
}

func bannerbearOverlay() overlaySpec {
	security := map[string]map[string]any{
		"bannerbearBearer": {"type": "http", "scheme": "bearer", "description": "Bannerbear Project API Key or Master API Key carried as an Authorization bearer token."},
	}
	return overlaySpec{
		ProviderID:  "bannerbear",
		Title:       "Bannerbear API v2 Advisory Overlay",
		Description: "Advisory OpenAPI overlay derived from official Bannerbear API v2 human documentation. This is not an official Bannerbear OpenAPI document.",
		ServerURL:   "https://api.bannerbear.com",
		Sources:     []string{"https://developers.bannerbear.com/v2/", "https://www.bannerbear.com/help/api/"},
		SourceNote:  "Bannerbear publishes REST-shaped API v2 docs with Authorization bearer API-key metadata; this overlay covers authentication, images, collections, videos, screenshots, templates, and projects.",
		Security:    security,
		Schemas:     []string{"BannerbearObject", "BannerbearCollection", "BannerbearError"},
		OutputPath:  "catalog-openapi-cache/advisory-overlays/bannerbear-api-v2-overlay.json",
		Paths: map[string]map[string]any{
			"/v2/auth": {"get": op("getBannerbearAuth", "Check authentication", nil, "", "#/components/schemas/BannerbearObject", "bannerbearBearer")},
			"/v2/images": {
				"get":  op("listBannerbearImages", "List images", params(query("page", "Page number.")), "", "#/components/schemas/BannerbearCollection", "bannerbearBearer"),
				"post": op("createBannerbearImage", "Create an image", nil, "#/components/schemas/BannerbearObject", "#/components/schemas/BannerbearObject", "bannerbearBearer"),
			},
			"/v2/images/{image_uid}": {"get": op("getBannerbearImage", "Get an image", params(path("image_uid", "Bannerbear image UID.")), "", "#/components/schemas/BannerbearObject", "bannerbearBearer")},
			"/v2/collections": {
				"get":  op("listBannerbearCollections", "List collections", params(query("page", "Page number.")), "", "#/components/schemas/BannerbearCollection", "bannerbearBearer"),
				"post": op("createBannerbearCollection", "Create a collection", nil, "#/components/schemas/BannerbearObject", "#/components/schemas/BannerbearObject", "bannerbearBearer"),
			},
			"/v2/collections/{collection_uid}": {"get": op("getBannerbearCollection", "Get a collection", params(path("collection_uid", "Bannerbear collection UID.")), "", "#/components/schemas/BannerbearObject", "bannerbearBearer")},
			"/v2/videos": {
				"get":  op("listBannerbearVideos", "List videos", params(query("page", "Page number.")), "", "#/components/schemas/BannerbearCollection", "bannerbearBearer"),
				"post": op("createBannerbearVideo", "Create a video", nil, "#/components/schemas/BannerbearObject", "#/components/schemas/BannerbearObject", "bannerbearBearer"),
			},
			"/v2/videos/{video_uid}": {"get": op("getBannerbearVideo", "Get a video", params(path("video_uid", "Bannerbear video UID.")), "", "#/components/schemas/BannerbearObject", "bannerbearBearer")},
			"/v2/screenshots":        {"post": op("createBannerbearScreenshot", "Create a screenshot", nil, "#/components/schemas/BannerbearObject", "#/components/schemas/BannerbearObject", "bannerbearBearer")},
			"/v2/screenshots/{screenshot_uid}": {
				"get": op("getBannerbearScreenshot", "Get a screenshot", params(path("screenshot_uid", "Bannerbear screenshot UID.")), "", "#/components/schemas/BannerbearObject", "bannerbearBearer"),
			},
			"/v2/templates": {
				"get":  op("listBannerbearTemplates", "List templates", params(query("page", "Page number.")), "", "#/components/schemas/BannerbearCollection", "bannerbearBearer"),
				"post": op("createBannerbearTemplate", "Create a template", nil, "#/components/schemas/BannerbearObject", "#/components/schemas/BannerbearObject", "bannerbearBearer"),
			},
			"/v2/templates/{template_uid}": {
				"get":    op("getBannerbearTemplate", "Get a template", params(path("template_uid", "Bannerbear template UID.")), "", "#/components/schemas/BannerbearObject", "bannerbearBearer"),
				"patch":  op("updateBannerbearTemplate", "Update a template", params(path("template_uid", "Bannerbear template UID.")), "#/components/schemas/BannerbearObject", "#/components/schemas/BannerbearObject", "bannerbearBearer"),
				"delete": op("deleteBannerbearTemplate", "Delete a template", params(path("template_uid", "Bannerbear template UID.")), "", "#/components/schemas/BannerbearObject", "bannerbearBearer"),
			},
			"/v2/projects": {
				"get":  op("listBannerbearProjects", "List projects", nil, "", "#/components/schemas/BannerbearCollection", "bannerbearBearer"),
				"post": op("createBannerbearProject", "Create a project", nil, "#/components/schemas/BannerbearObject", "#/components/schemas/BannerbearObject", "bannerbearBearer"),
			},
			"/v2/projects/{project_uid}": {"get": op("getBannerbearProject", "Get a project", params(path("project_uid", "Bannerbear project UID.")), "", "#/components/schemas/BannerbearObject", "bannerbearBearer")},
		},
	}
}

func gristOverlay() overlaySpec {
	security := map[string]map[string]any{
		"gristBearer": {"type": "http", "scheme": "bearer", "description": "Grist API key carried as an Authorization bearer token."},
	}
	return overlaySpec{
		ProviderID:  "grist",
		Title:       "Grist REST API Advisory Overlay",
		Description: "Advisory OpenAPI overlay derived from official Grist REST API human documentation. This is not an official Grist OpenAPI document.",
		ServerURL:   "https://{site_domain}/api",
		ServerVars:  map[string]map[string]any{"site_domain": {"default": "docs.getgrist.com", "description": "Operator-supplied Grist personal, team, or self-hosted domain."}},
		Sources:     []string{"https://support.getgrist.com/rest-api/", "https://support.getgrist.com/api/"},
		SourceNote:  "Grist publishes REST API usage and reference docs with bearer API-key metadata; this overlay covers organizations, workspaces, documents, tables, records, SQL queries, and webhooks.",
		Security:    security,
		Schemas:     []string{"GristObject", "GristCollection", "GristError"},
		OutputPath:  "catalog-openapi-cache/advisory-overlays/grist-rest-api-overlay.json",
		Paths: map[string]map[string]any{
			"/orgs":                                    {"get": op("listGristOrgs", "List organizations", nil, "", "#/components/schemas/GristCollection", "gristBearer")},
			"/orgs/{org_id}":                           {"get": op("getGristOrg", "Get an organization", params(path("org_id", "Grist organization ID.")), "", "#/components/schemas/GristObject", "gristBearer")},
			"/orgs/{org_id}/workspaces":                {"get": op("listGristOrgWorkspaces", "List organization workspaces", params(path("org_id", "Grist organization ID.")), "", "#/components/schemas/GristCollection", "gristBearer")},
			"/workspaces/{workspace_id}":               {"get": op("getGristWorkspace", "Get a workspace", params(path("workspace_id", "Grist workspace ID.")), "", "#/components/schemas/GristObject", "gristBearer"), "patch": op("updateGristWorkspace", "Update a workspace", params(path("workspace_id", "Grist workspace ID.")), "#/components/schemas/GristObject", "#/components/schemas/GristObject", "gristBearer"), "delete": op("deleteGristWorkspace", "Delete a workspace", params(path("workspace_id", "Grist workspace ID.")), "", "#/components/schemas/GristObject", "gristBearer")},
			"/workspaces/{workspace_id}/docs":          {"get": op("listGristWorkspaceDocs", "List workspace documents", params(path("workspace_id", "Grist workspace ID.")), "", "#/components/schemas/GristCollection", "gristBearer"), "post": op("createGristWorkspaceDoc", "Create a workspace document", params(path("workspace_id", "Grist workspace ID.")), "#/components/schemas/GristObject", "#/components/schemas/GristObject", "gristBearer")},
			"/docs/{doc_id}":                           {"get": op("getGristDoc", "Get a document", params(path("doc_id", "Grist document ID.")), "", "#/components/schemas/GristObject", "gristBearer"), "patch": op("updateGristDoc", "Update a document", params(path("doc_id", "Grist document ID.")), "#/components/schemas/GristObject", "#/components/schemas/GristObject", "gristBearer"), "delete": op("deleteGristDoc", "Delete a document", params(path("doc_id", "Grist document ID.")), "", "#/components/schemas/GristObject", "gristBearer")},
			"/docs/{doc_id}/tables":                    {"get": op("listGristDocTables", "List document tables", params(path("doc_id", "Grist document ID.")), "", "#/components/schemas/GristCollection", "gristBearer")},
			"/docs/{doc_id}/tables/{table_id}/records": {"get": op("listGristTableRecords", "List table records", params(path("doc_id", "Grist document ID."), path("table_id", "Grist table ID."), query("sort", "Sort expression."), query("filter", "Filter expression.")), "", "#/components/schemas/GristCollection", "gristBearer"), "post": op("addGristTableRecords", "Add table records", params(path("doc_id", "Grist document ID."), path("table_id", "Grist table ID.")), "#/components/schemas/GristObject", "#/components/schemas/GristObject", "gristBearer"), "patch": op("updateGristTableRecords", "Update table records", params(path("doc_id", "Grist document ID."), path("table_id", "Grist table ID.")), "#/components/schemas/GristObject", "#/components/schemas/GristObject", "gristBearer"), "put": op("replaceGristTableRecords", "Replace table records", params(path("doc_id", "Grist document ID."), path("table_id", "Grist table ID.")), "#/components/schemas/GristObject", "#/components/schemas/GristObject", "gristBearer")},
			"/docs/{doc_id}/sql":                       {"get": op("runGristSQLQuery", "Run a SQL query", params(path("doc_id", "Grist document ID."), query("q", "SQL query text.")), "", "#/components/schemas/GristObject", "gristBearer")},
			"/docs/{doc_id}/webhooks":                  {"get": op("listGristWebhooks", "List webhooks", params(path("doc_id", "Grist document ID.")), "", "#/components/schemas/GristCollection", "gristBearer"), "post": op("createGristWebhook", "Create a webhook", params(path("doc_id", "Grist document ID.")), "#/components/schemas/GristObject", "#/components/schemas/GristObject", "gristBearer")},
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
	if len(spec.ServerVars) > 0 {
		doc["servers"] = []map[string]any{{"url": spec.ServerURL, "variables": orderedMap(spec.ServerVars)}}
	}
	return doc
}

func op(operationID, summary string, parameters []map[string]any, requestRef, responseRef, securityName string) map[string]any {
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
		"security": []map[string][]string{{securityName: []string{}}},
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
	out := make(map[string]V, len(in))
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
