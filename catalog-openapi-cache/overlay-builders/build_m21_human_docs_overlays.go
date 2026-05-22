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
	Schemas      []string
	Paths        map[string]map[string]any
	OutputPath   string
}

func main() {
	for _, spec := range []overlaySpec{
		activeCampaignOverlay(),
		bambooHROverlay(),
		contentfulOverlay(),
		eventbriteOverlay(),
		freshdeskOverlay(),
		postmarkOverlay(),
		todoistOverlay(),
		webflowOverlay(),
	} {
		write(spec.OutputPath, build(spec))
	}
}

func activeCampaignOverlay() overlaySpec {
	security := apiKey("activeCampaignAPIToken", "header", "Api-Token", "ActiveCampaign API token carried in the Api-Token header.")
	return overlaySpec{
		ProviderID:   "activecampaign",
		OverlayID:    "activecampaign-api-v3-advisory-overlay",
		Title:        "ActiveCampaign API v3 Advisory Overlay",
		Description:  "Advisory OpenAPI overlay derived from official ActiveCampaign API v3 human documentation. This is not an official ActiveCampaign OpenAPI document.",
		ServerURL:    "https://{account}.api-us1.com/api/3",
		ServerVars:   map[string]map[string]any{"account": {"default": "example", "description": "Operator-supplied ActiveCampaign account name."}},
		Sources:      []string{"https://developers.activecampaign.com/reference/overview", "https://developers.activecampaign.com/reference/authentication", "https://developers.activecampaign.com/reference/contact", "https://developers.activecampaign.com/reference/list-all-contacts", "https://developers.activecampaign.com/reference/get-contact", "https://developers.activecampaign.com/reference/schema"},
		SourceNote:   "ActiveCampaign publishes REST-shaped API v3 human documentation with account-region hosts and Api-Token header authentication; this overlay covers a small contacts, lists, campaigns, and deals subset.",
		SecurityName: "activeCampaignAPIToken",
		Security:     security,
		Schemas:      []string{"ActiveCampaignObject", "ActiveCampaignCollection", "ActiveCampaignError"},
		OutputPath:   "catalog-openapi-cache/advisory-overlays/activecampaign-api-v3-overlay.json",
		Paths: map[string]map[string]any{
			"/contacts": {
				"get":  op("listActiveCampaignContacts", "List contacts", params(query("limit", "Maximum number of contacts to return."), query("offset", "Offset for pagination.")), "", "#/components/schemas/ActiveCampaignCollection", "activeCampaignAPIToken"),
				"post": op("createActiveCampaignContact", "Create a contact", nil, "#/components/schemas/ActiveCampaignObject", "#/components/schemas/ActiveCampaignObject", "activeCampaignAPIToken"),
			},
			"/contacts/{contact_id}": {
				"get":    op("getActiveCampaignContact", "Retrieve a contact", params(path("contact_id", "ActiveCampaign contact ID.")), "", "#/components/schemas/ActiveCampaignObject", "activeCampaignAPIToken"),
				"put":    op("updateActiveCampaignContact", "Update a contact", params(path("contact_id", "ActiveCampaign contact ID.")), "#/components/schemas/ActiveCampaignObject", "#/components/schemas/ActiveCampaignObject", "activeCampaignAPIToken"),
				"delete": op("deleteActiveCampaignContact", "Delete a contact", params(path("contact_id", "ActiveCampaign contact ID.")), "", "#/components/schemas/ActiveCampaignObject", "activeCampaignAPIToken"),
			},
			"/lists":     {"get": op("listActiveCampaignLists", "List lists", nil, "", "#/components/schemas/ActiveCampaignCollection", "activeCampaignAPIToken")},
			"/campaigns": {"get": op("listActiveCampaignCampaigns", "List campaigns", nil, "", "#/components/schemas/ActiveCampaignCollection", "activeCampaignAPIToken")},
			"/deals":     {"get": op("listActiveCampaignDeals", "List deals", params(query("limit", "Maximum number of deals to return."), query("offset", "Offset for pagination.")), "", "#/components/schemas/ActiveCampaignCollection", "activeCampaignAPIToken")},
		},
	}
}

func bambooHROverlay() overlaySpec {
	security := map[string]any{"bambooHRBasic": map[string]any{"type": "http", "scheme": "basic", "description": "BambooHR API key used as the HTTP Basic username with an arbitrary password."}}
	return overlaySpec{
		ProviderID:   "bamboohr",
		OverlayID:    "bamboohr-api-v1-advisory-overlay",
		Title:        "BambooHR API v1 Advisory Overlay",
		Description:  "Advisory OpenAPI overlay derived from official BambooHR API human documentation. This is not an official BambooHR OpenAPI document.",
		ServerURL:    "https://{company_domain}.bamboohr.com/api",
		ServerVars:   map[string]map[string]any{"company_domain": {"default": "example", "description": "Operator-supplied BambooHR company subdomain."}},
		Sources:      []string{"https://documentation.bamboohr.com/docs", "https://documentation.bamboohr.com/docs/api-details", "https://documentation.bamboohr.com/reference/get-employees-list", "https://documentation.bamboohr.com/reference/get-employees-directory-1"},
		SourceNote:   "BambooHR publishes REST-shaped API documentation with customer-subdomain hosts and API-key Basic authentication; this overlay covers employee, directory, metadata, report, and time-off review endpoints.",
		SecurityName: "bambooHRBasic",
		Security:     security,
		Schemas:      []string{"BambooHRObject", "BambooHRCollection", "BambooHRError"},
		OutputPath:   "catalog-openapi-cache/advisory-overlays/bamboohr-api-v1-overlay.json",
		Paths: map[string]map[string]any{
			"/v1/employees": {
				"get": op("listBambooHREmployees", "List employees", params(query("page", "Page number."), query("size", "Page size.")), "", "#/components/schemas/BambooHRCollection", "bambooHRBasic"),
			},
			"/v1/employees/directory": {
				"get": op("getBambooHREmployeeDirectory", "Get employee directory", nil, "", "#/components/schemas/BambooHRCollection", "bambooHRBasic"),
			},
			"/v1/employees/{employee_id}": {
				"get":  op("getBambooHREmployee", "Get an employee", params(path("employee_id", "BambooHR employee ID."), query("fields", "Comma-separated employee fields to return.")), "", "#/components/schemas/BambooHRObject", "bambooHRBasic"),
				"post": op("updateBambooHREmployee", "Update an employee", params(path("employee_id", "BambooHR employee ID.")), "#/components/schemas/BambooHRObject", "#/components/schemas/BambooHRObject", "bambooHRBasic"),
			},
			"/v1/meta/fields":       {"get": op("listBambooHRFields", "List employee fields", nil, "", "#/components/schemas/BambooHRCollection", "bambooHRBasic")},
			"/v1/reports/custom":    {"post": op("createBambooHRCustomReport", "Create a custom report", nil, "#/components/schemas/BambooHRObject", "#/components/schemas/BambooHRCollection", "bambooHRBasic")},
			"/v1/time_off/requests": {"get": op("listBambooHRTimeOffRequests", "List time off requests", params(query("start", "Start date filter."), query("end", "End date filter.")), "", "#/components/schemas/BambooHRCollection", "bambooHRBasic")},
		},
	}
}

func contentfulOverlay() overlaySpec {
	security := bearer("contentfulBearer", "Contentful API access token carried in the Authorization header.")
	return overlaySpec{
		ProviderID:   "contentful",
		OverlayID:    "contentful-management-api-advisory-overlay",
		Title:        "Contentful Content Management API Advisory Overlay",
		Description:  "Advisory OpenAPI overlay derived from official Contentful Content Management API human documentation. This is not an official Contentful OpenAPI document.",
		ServerURL:    "https://api.contentful.com",
		Sources:      []string{"https://www.contentful.com/developers/docs/references/content-management-api/", "https://www.contentful.com/developers/docs/references/authentication/", "https://www.contentful.com/developers/docs/references/content-delivery-api/"},
		SourceNote:   "Contentful publishes multiple REST API surfaces; this overlay focuses on the Content Management API for spaces, environments, entries, assets, and content types.",
		SecurityName: "contentfulBearer",
		Security:     security,
		Schemas:      []string{"ContentfulObject", "ContentfulCollection", "ContentfulError"},
		OutputPath:   "catalog-openapi-cache/advisory-overlays/contentful-management-api-overlay.json",
		Paths: map[string]map[string]any{
			"/spaces": {"get": op("listContentfulSpaces", "List spaces", nil, "", "#/components/schemas/ContentfulCollection", "contentfulBearer")},
			"/spaces/{space_id}": {
				"get": op("getContentfulSpace", "Get a space", params(path("space_id", "Contentful space ID.")), "", "#/components/schemas/ContentfulObject", "contentfulBearer"),
			},
			"/spaces/{space_id}/environments": {
				"get": op("listContentfulEnvironments", "List environments", params(path("space_id", "Contentful space ID.")), "", "#/components/schemas/ContentfulCollection", "contentfulBearer"),
			},
			"/spaces/{space_id}/environments/{environment_id}/entries": {
				"get":  op("listContentfulEntries", "List entries", params(path("space_id", "Contentful space ID."), path("environment_id", "Contentful environment ID."), query("content_type", "Content type filter.")), "", "#/components/schemas/ContentfulCollection", "contentfulBearer"),
				"post": op("createContentfulEntry", "Create an entry", params(path("space_id", "Contentful space ID."), path("environment_id", "Contentful environment ID.")), "#/components/schemas/ContentfulObject", "#/components/schemas/ContentfulObject", "contentfulBearer"),
			},
			"/spaces/{space_id}/environments/{environment_id}/entries/{entry_id}": {
				"get":    op("getContentfulEntry", "Get an entry", params(path("space_id", "Contentful space ID."), path("environment_id", "Contentful environment ID."), path("entry_id", "Contentful entry ID.")), "", "#/components/schemas/ContentfulObject", "contentfulBearer"),
				"put":    op("putContentfulEntry", "Create or replace an entry", params(path("space_id", "Contentful space ID."), path("environment_id", "Contentful environment ID."), path("entry_id", "Contentful entry ID.")), "#/components/schemas/ContentfulObject", "#/components/schemas/ContentfulObject", "contentfulBearer"),
				"delete": op("deleteContentfulEntry", "Delete an entry", params(path("space_id", "Contentful space ID."), path("environment_id", "Contentful environment ID."), path("entry_id", "Contentful entry ID.")), "", "#/components/schemas/ContentfulObject", "contentfulBearer"),
			},
			"/spaces/{space_id}/environments/{environment_id}/assets":        {"get": op("listContentfulAssets", "List assets", params(path("space_id", "Contentful space ID."), path("environment_id", "Contentful environment ID.")), "", "#/components/schemas/ContentfulCollection", "contentfulBearer")},
			"/spaces/{space_id}/environments/{environment_id}/content_types": {"get": op("listContentfulContentTypes", "List content types", params(path("space_id", "Contentful space ID."), path("environment_id", "Contentful environment ID.")), "", "#/components/schemas/ContentfulCollection", "contentfulBearer")},
		},
	}
}

func eventbriteOverlay() overlaySpec {
	security := bearer("eventbriteBearer", "Eventbrite OAuth access token carried in the Authorization header.")
	return overlaySpec{
		ProviderID:   "eventbrite",
		OverlayID:    "eventbrite-platform-api-v3-advisory-overlay",
		Title:        "Eventbrite Platform API v3 Advisory Overlay",
		Description:  "Advisory OpenAPI overlay derived from official Eventbrite Platform API human documentation. This is not an official Eventbrite OpenAPI document.",
		ServerURL:    "https://www.eventbriteapi.com/v3",
		Sources:      []string{"https://www.eventbrite.com/platform/api", "https://www.eventbrite.com/platform/api#/introduction/authentication"},
		SourceNote:   "Eventbrite publishes REST-shaped Platform API v3 documentation through an interactive reference; this overlay covers users, organizations, events, attendees, venues, and ticket classes.",
		SecurityName: "eventbriteBearer",
		Security:     security,
		Schemas:      []string{"EventbriteObject", "EventbriteCollection", "EventbriteError"},
		OutputPath:   "catalog-openapi-cache/advisory-overlays/eventbrite-platform-api-v3-overlay.json",
		Paths: map[string]map[string]any{
			"/users/me/":                               {"get": op("getEventbriteCurrentUser", "Get current user", nil, "", "#/components/schemas/EventbriteObject", "eventbriteBearer")},
			"/users/me/organizations/":                 {"get": op("listEventbriteCurrentUserOrganizations", "List current user's organizations", nil, "", "#/components/schemas/EventbriteCollection", "eventbriteBearer")},
			"/organizations/{organization_id}/events/": {"get": op("listEventbriteOrganizationEvents", "List organization events", params(path("organization_id", "Eventbrite organization ID."), query("status", "Event status filter.")), "", "#/components/schemas/EventbriteCollection", "eventbriteBearer")},
			"/events/{event_id}/": {
				"get":  op("getEventbriteEvent", "Get an event", params(path("event_id", "Eventbrite event ID.")), "", "#/components/schemas/EventbriteObject", "eventbriteBearer"),
				"post": op("updateEventbriteEvent", "Update an event", params(path("event_id", "Eventbrite event ID.")), "#/components/schemas/EventbriteObject", "#/components/schemas/EventbriteObject", "eventbriteBearer"),
			},
			"/events/{event_id}/attendees/":      {"get": op("listEventbriteEventAttendees", "List event attendees", params(path("event_id", "Eventbrite event ID.")), "", "#/components/schemas/EventbriteCollection", "eventbriteBearer")},
			"/events/{event_id}/ticket_classes/": {"get": op("listEventbriteTicketClasses", "List event ticket classes", params(path("event_id", "Eventbrite event ID.")), "", "#/components/schemas/EventbriteCollection", "eventbriteBearer")},
			"/venues/{venue_id}/":                {"get": op("getEventbriteVenue", "Get a venue", params(path("venue_id", "Eventbrite venue ID.")), "", "#/components/schemas/EventbriteObject", "eventbriteBearer")},
		},
	}
}

func freshdeskOverlay() overlaySpec {
	security := map[string]any{"freshdeskBasic": map[string]any{"type": "http", "scheme": "basic", "description": "Freshdesk API key carried with HTTP Basic authentication."}}
	return overlaySpec{
		ProviderID:   "freshdesk",
		OverlayID:    "freshdesk-api-v2-advisory-overlay",
		Title:        "Freshdesk API v2 Advisory Overlay",
		Description:  "Advisory OpenAPI overlay derived from official Freshdesk API v2 human documentation. This is not an official Freshdesk OpenAPI document.",
		ServerURL:    "https://{domain}.freshdesk.com/api/v2",
		ServerVars:   map[string]map[string]any{"domain": {"default": "example", "description": "Operator-supplied Freshdesk account domain."}},
		Sources:      []string{"https://developers.freshdesk.com/api", "https://support.freshdesk.com/en/support/solutions/articles/225441-is-there-any-documentation-for-the-apis-on-freshdesk-"},
		SourceNote:   "Freshdesk publishes REST-shaped v2 API documentation with account-domain hosts and API-key Basic authentication; this overlay covers a focused tickets, contacts, companies, and agents subset.",
		SecurityName: "freshdeskBasic",
		Security:     security,
		Schemas:      []string{"FreshdeskObject", "FreshdeskCollection", "FreshdeskError"},
		OutputPath:   "catalog-openapi-cache/advisory-overlays/freshdesk-api-v2-overlay.json",
		Paths: map[string]map[string]any{
			"/tickets": {
				"get":  op("listFreshdeskTickets", "List tickets", params(query("page", "Page number."), query("per_page", "Page size.")), "", "#/components/schemas/FreshdeskCollection", "freshdeskBasic"),
				"post": op("createFreshdeskTicket", "Create a ticket", nil, "#/components/schemas/FreshdeskObject", "#/components/schemas/FreshdeskObject", "freshdeskBasic"),
			},
			"/tickets/{ticket_id}": {
				"get":    op("getFreshdeskTicket", "Get a ticket", params(path("ticket_id", "Freshdesk ticket ID.")), "", "#/components/schemas/FreshdeskObject", "freshdeskBasic"),
				"put":    op("updateFreshdeskTicket", "Update a ticket", params(path("ticket_id", "Freshdesk ticket ID.")), "#/components/schemas/FreshdeskObject", "#/components/schemas/FreshdeskObject", "freshdeskBasic"),
				"delete": op("deleteFreshdeskTicket", "Delete a ticket", params(path("ticket_id", "Freshdesk ticket ID.")), "", "#/components/schemas/FreshdeskObject", "freshdeskBasic"),
			},
			"/contacts": {
				"get":  op("listFreshdeskContacts", "List contacts", nil, "", "#/components/schemas/FreshdeskCollection", "freshdeskBasic"),
				"post": op("createFreshdeskContact", "Create a contact", nil, "#/components/schemas/FreshdeskObject", "#/components/schemas/FreshdeskObject", "freshdeskBasic"),
			},
			"/contacts/{contact_id}": {
				"get":    op("getFreshdeskContact", "Get a contact", params(path("contact_id", "Freshdesk contact ID.")), "", "#/components/schemas/FreshdeskObject", "freshdeskBasic"),
				"put":    op("updateFreshdeskContact", "Update a contact", params(path("contact_id", "Freshdesk contact ID.")), "#/components/schemas/FreshdeskObject", "#/components/schemas/FreshdeskObject", "freshdeskBasic"),
				"delete": op("deleteFreshdeskContact", "Delete a contact", params(path("contact_id", "Freshdesk contact ID.")), "", "#/components/schemas/FreshdeskObject", "freshdeskBasic"),
			},
			"/companies": {"get": op("listFreshdeskCompanies", "List companies", nil, "", "#/components/schemas/FreshdeskCollection", "freshdeskBasic")},
			"/agents":    {"get": op("listFreshdeskAgents", "List agents", nil, "", "#/components/schemas/FreshdeskCollection", "freshdeskBasic")},
		},
	}
}

func postmarkOverlay() overlaySpec {
	security := map[string]map[string]any{
		"postmarkServerToken":  {"type": "apiKey", "in": "header", "name": "X-Postmark-Server-Token", "description": "Postmark server token carried in the X-Postmark-Server-Token header."},
		"postmarkAccountToken": {"type": "apiKey", "in": "header", "name": "X-Postmark-Account-Token", "description": "Postmark account token carried in the X-Postmark-Account-Token header for account-level endpoints."},
	}
	return overlaySpec{
		ProviderID:  "postmark",
		OverlayID:   "postmark-api-advisory-overlay",
		Title:       "Postmark API Advisory Overlay",
		Description: "Advisory OpenAPI overlay derived from official Postmark API human documentation. This is not an official Postmark OpenAPI document.",
		ServerURL:   "https://api.postmarkapp.com",
		Sources:     []string{"https://postmarkapp.com/developer/api/overview", "https://postmarkapp.com/api-explorer", "https://postmarkapp.com/developer/api/email-api", "https://postmarkapp.com/developer/api/templates-api", "https://postmarkapp.com/developer/api/bounce-api", "https://postmarkapp.com/developer/api/server-api", "https://postmarkapp.com/developer/api/message-streams-api"},
		SourceNote:  "Postmark publishes REST-shaped API docs and an API Explorer; this overlay models server-token email/template/bounce/message-stream endpoints separately from account-token server endpoints.",
		SecurityAlt: security,
		Schemas:     []string{"PostmarkObject", "PostmarkCollection", "PostmarkError"},
		OutputPath:  "catalog-openapi-cache/advisory-overlays/postmark-api-overlay.json",
		Paths: map[string]map[string]any{
			"/email":       {"post": op("sendPostmarkEmail", "Send an email", nil, "#/components/schemas/PostmarkObject", "#/components/schemas/PostmarkObject", "postmarkServerToken")},
			"/email/batch": {"post": op("sendPostmarkEmailBatch", "Send a batch of emails", nil, "#/components/schemas/PostmarkCollection", "#/components/schemas/PostmarkCollection", "postmarkServerToken")},
			"/templates": {
				"get":  op("listPostmarkTemplates", "List templates", params(query("Count", "Maximum number of templates to return."), query("Offset", "Offset for pagination.")), "", "#/components/schemas/PostmarkCollection", "postmarkServerToken"),
				"post": op("createPostmarkTemplate", "Create a template", nil, "#/components/schemas/PostmarkObject", "#/components/schemas/PostmarkObject", "postmarkServerToken"),
			},
			"/templates/{template_id}": {
				"get":    op("getPostmarkTemplate", "Get a template", params(path("template_id", "Postmark template ID.")), "", "#/components/schemas/PostmarkObject", "postmarkServerToken"),
				"put":    op("updatePostmarkTemplate", "Update a template", params(path("template_id", "Postmark template ID.")), "#/components/schemas/PostmarkObject", "#/components/schemas/PostmarkObject", "postmarkServerToken"),
				"delete": op("deletePostmarkTemplate", "Delete a template", params(path("template_id", "Postmark template ID.")), "", "#/components/schemas/PostmarkObject", "postmarkServerToken"),
			},
			"/bounces":             {"get": op("listPostmarkBounces", "List bounces", nil, "", "#/components/schemas/PostmarkCollection", "postmarkServerToken")},
			"/bounces/{bounce_id}": {"get": op("getPostmarkBounce", "Get a bounce", params(path("bounce_id", "Postmark bounce ID.")), "", "#/components/schemas/PostmarkObject", "postmarkServerToken")},
			"/message-streams": {
				"get":  op("listPostmarkMessageStreams", "List message streams", params(query("MessageStreamType", "Filter by message stream type."), query("IncludeArchivedStreams", "Whether archived streams should be included.")), "", "#/components/schemas/PostmarkCollection", "postmarkServerToken"),
				"post": op("createPostmarkMessageStream", "Create a message stream", nil, "#/components/schemas/PostmarkObject", "#/components/schemas/PostmarkObject", "postmarkServerToken"),
			},
			"/message-streams/{stream_id}": {
				"get":    op("getPostmarkMessageStream", "Get a message stream", params(path("stream_id", "Postmark message stream ID.")), "", "#/components/schemas/PostmarkObject", "postmarkServerToken"),
				"patch":  op("updatePostmarkMessageStream", "Update a message stream", params(path("stream_id", "Postmark message stream ID.")), "#/components/schemas/PostmarkObject", "#/components/schemas/PostmarkObject", "postmarkServerToken"),
				"delete": op("archivePostmarkMessageStream", "Archive a message stream", params(path("stream_id", "Postmark message stream ID.")), "", "#/components/schemas/PostmarkObject", "postmarkServerToken"),
			},
			"/message-streams/{stream_id}/unarchive": {
				"patch": op("unarchivePostmarkMessageStream", "Unarchive a message stream", params(path("stream_id", "Postmark message stream ID.")), "", "#/components/schemas/PostmarkObject", "postmarkServerToken"),
			},
			"/servers": {"get": op("listPostmarkServers", "List servers", nil, "", "#/components/schemas/PostmarkCollection", "postmarkAccountToken")},
		},
	}
}

func todoistOverlay() overlaySpec {
	security := bearer("todoistBearer", "Todoist API token carried as an Authorization bearer token.")
	return overlaySpec{
		ProviderID:   "todoist",
		OverlayID:    "todoist-rest-api-v2-advisory-overlay",
		Title:        "Todoist REST API v2 Advisory Overlay",
		Description:  "Advisory OpenAPI overlay derived from official Todoist REST API v2 human documentation. This is not an official Todoist OpenAPI document.",
		ServerURL:    "https://api.todoist.com/rest/v2",
		Sources:      []string{"https://developer.todoist.com/rest/v2/", "https://developer.todoist.com/guides/#authentication"},
		SourceNote:   "Todoist publishes REST API v2 human documentation with bearer-token authentication; the official page now points developers toward newer unified API docs, so this overlay stays limited to the reviewed REST v2 core resource subset.",
		SecurityName: "todoistBearer",
		Security:     security,
		Schemas:      []string{"TodoistObject", "TodoistCollection", "TodoistError"},
		OutputPath:   "catalog-openapi-cache/advisory-overlays/todoist-rest-api-v2-overlay.json",
		Paths: map[string]map[string]any{
			"/projects": {
				"get":  op("listTodoistProjects", "List projects", nil, "", "#/components/schemas/TodoistCollection", "todoistBearer"),
				"post": op("createTodoistProject", "Create a project", nil, "#/components/schemas/TodoistObject", "#/components/schemas/TodoistObject", "todoistBearer"),
			},
			"/projects/{project_id}": {
				"get":    op("getTodoistProject", "Get a project", params(path("project_id", "Todoist project ID.")), "", "#/components/schemas/TodoistObject", "todoistBearer"),
				"post":   op("updateTodoistProject", "Update a project", params(path("project_id", "Todoist project ID.")), "#/components/schemas/TodoistObject", "#/components/schemas/TodoistObject", "todoistBearer"),
				"delete": op("deleteTodoistProject", "Delete a project", params(path("project_id", "Todoist project ID.")), "", "#/components/schemas/TodoistObject", "todoistBearer"),
			},
			"/sections": {
				"get":  op("listTodoistSections", "List sections", params(query("project_id", "Project ID filter.")), "", "#/components/schemas/TodoistCollection", "todoistBearer"),
				"post": op("createTodoistSection", "Create a section", nil, "#/components/schemas/TodoistObject", "#/components/schemas/TodoistObject", "todoistBearer"),
			},
			"/tasks": {
				"get":  op("listTodoistTasks", "List active tasks", params(query("project_id", "Project ID filter."), query("section_id", "Section ID filter.")), "", "#/components/schemas/TodoistCollection", "todoistBearer"),
				"post": op("createTodoistTask", "Create a task", nil, "#/components/schemas/TodoistObject", "#/components/schemas/TodoistObject", "todoistBearer"),
			},
			"/tasks/{task_id}": {
				"get":    op("getTodoistTask", "Get a task", params(path("task_id", "Todoist task ID.")), "", "#/components/schemas/TodoistObject", "todoistBearer"),
				"post":   op("updateTodoistTask", "Update a task", params(path("task_id", "Todoist task ID.")), "#/components/schemas/TodoistObject", "#/components/schemas/TodoistObject", "todoistBearer"),
				"delete": op("deleteTodoistTask", "Delete a task", params(path("task_id", "Todoist task ID.")), "", "#/components/schemas/TodoistObject", "todoistBearer"),
			},
			"/comments": {"get": op("listTodoistComments", "List comments", params(query("task_id", "Task ID filter."), query("project_id", "Project ID filter.")), "", "#/components/schemas/TodoistCollection", "todoistBearer")},
		},
	}
}

func webflowOverlay() overlaySpec {
	security := bearer("webflowBearer", "Webflow API token carried as an Authorization bearer token.")
	return overlaySpec{
		ProviderID:   "webflow",
		OverlayID:    "webflow-data-api-v2-advisory-overlay",
		Title:        "Webflow Data API v2 Advisory Overlay",
		Description:  "Advisory OpenAPI overlay derived from official Webflow Data API human documentation. This is not an official Webflow OpenAPI document.",
		ServerURL:    "https://api.webflow.com/v2",
		Sources:      []string{"https://developers.webflow.com/reference", "https://developers.webflow.com/data/reference/rest-introduction", "https://developers.webflow.com/data/reference", "https://developers.webflow.com/data/reference/authorization", "https://developers.webflow.com/data/reference/pages-and-components/pages/list"},
		SourceNote:   "Webflow publishes REST-shaped Data API documentation with bearer-token authorization; this overlay covers sites, pages, collections, and CMS item workflows.",
		SecurityName: "webflowBearer",
		Security:     security,
		Schemas:      []string{"WebflowObject", "WebflowCollection", "WebflowError"},
		OutputPath:   "catalog-openapi-cache/advisory-overlays/webflow-data-api-v2-overlay.json",
		Paths: map[string]map[string]any{
			"/sites": {"get": op("listWebflowSites", "List sites", nil, "", "#/components/schemas/WebflowCollection", "webflowBearer")},
			"/sites/{site_id}": {
				"get": op("getWebflowSite", "Get a site", params(path("site_id", "Webflow site ID.")), "", "#/components/schemas/WebflowObject", "webflowBearer"),
			},
			"/sites/{site_id}/pages": {
				"get": op("listWebflowPages", "List pages", params(path("site_id", "Webflow site ID.")), "", "#/components/schemas/WebflowCollection", "webflowBearer"),
			},
			"/sites/{site_id}/collections": {
				"get":  op("listWebflowCollections", "List collections", params(path("site_id", "Webflow site ID.")), "", "#/components/schemas/WebflowCollection", "webflowBearer"),
				"post": op("createWebflowCollection", "Create a collection", params(path("site_id", "Webflow site ID.")), "#/components/schemas/WebflowObject", "#/components/schemas/WebflowObject", "webflowBearer"),
			},
			"/collections/{collection_id}": {
				"get": op("getWebflowCollection", "Get a collection", params(path("collection_id", "Webflow collection ID.")), "", "#/components/schemas/WebflowObject", "webflowBearer"),
			},
			"/collections/{collection_id}/items": {
				"get":  op("listWebflowCollectionItems", "List collection items", params(path("collection_id", "Webflow collection ID."), query("limit", "Maximum number of items to return."), query("offset", "Offset for pagination.")), "", "#/components/schemas/WebflowCollection", "webflowBearer"),
				"post": op("createWebflowCollectionItem", "Create a collection item", params(path("collection_id", "Webflow collection ID.")), "#/components/schemas/WebflowObject", "#/components/schemas/WebflowObject", "webflowBearer"),
			},
			"/collections/{collection_id}/items/{item_id}": {
				"get":    op("getWebflowCollectionItem", "Get a collection item", params(path("collection_id", "Webflow collection ID."), path("item_id", "Webflow collection item ID.")), "", "#/components/schemas/WebflowObject", "webflowBearer"),
				"patch":  op("updateWebflowCollectionItem", "Update a collection item", params(path("collection_id", "Webflow collection ID."), path("item_id", "Webflow collection item ID.")), "#/components/schemas/WebflowObject", "#/components/schemas/WebflowObject", "webflowBearer"),
				"delete": op("deleteWebflowCollectionItem", "Delete a collection item", params(path("collection_id", "Webflow collection ID."), path("item_id", "Webflow collection item ID.")), "", "#/components/schemas/WebflowObject", "webflowBearer"),
			},
			"/collections/{collection_id}/items/publish": {"post": op("publishWebflowCollectionItems", "Publish collection items", params(path("collection_id", "Webflow collection ID.")), "#/components/schemas/WebflowObject", "#/components/schemas/WebflowObject", "webflowBearer")},
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

func apiKey(name, in, parameterName, description string) map[string]any {
	return map[string]any{name: map[string]any{"type": "apiKey", "in": in, "name": parameterName, "description": description}}
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
	req := map[string]any{}
	for _, name := range names {
		req[name] = []string{}
	}
	if len(req) == 0 {
		return nil
	}
	return []map[string]any{req}
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
