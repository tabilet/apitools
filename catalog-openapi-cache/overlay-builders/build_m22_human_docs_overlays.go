//go:build ignore

package main

import (
	"encoding/json"
	"os"
)

type overlaySpec struct {
	ProviderID  string
	OverlayID   string
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
		acuitySchedulingOverlay(),
		clockifyOverlay(),
		ghostOverlay(),
		harvestOverlay(),
		helpScoutOverlay(),
		iterableOverlay(),
		mailjetOverlay(),
	} {
		write(spec.OutputPath, build(spec))
	}
}

func acuitySchedulingOverlay() overlaySpec {
	security := map[string]map[string]any{"acuityBasic": {"type": "http", "scheme": "basic", "description": "Acuity Scheduling numeric user ID and API key carried with HTTP Basic authentication."}}
	return overlaySpec{
		ProviderID:  "acuity-scheduling",
		OverlayID:   "acuity-scheduling-api-v1-advisory-overlay",
		Title:       "Acuity Scheduling API v1 Advisory Overlay",
		Description: "Advisory OpenAPI overlay derived from official Acuity Scheduling API human documentation. This is not an official Acuity Scheduling OpenAPI document.",
		ServerURL:   "https://acuityscheduling.com/api/v1",
		Sources:     []string{"https://developers.acuityscheduling.com/reference", "https://developers.acuityscheduling.com/docs/quick-start", "https://developers.acuityscheduling.com/reference/get-appointments", "https://developers.acuityscheduling.com/reference/post-appointments", "https://developers.acuityscheduling.com/reference/get-appointments-id"},
		SourceNote:  "Acuity Scheduling publishes REST-shaped API docs with Basic auth; this overlay covers appointments, availability, appointment types, calendars, and clients.",
		Security:    security,
		Schemas:     []string{"AcuityObject", "AcuityCollection", "AcuityError"},
		OutputPath:  "catalog-openapi-cache/advisory-overlays/acuity-scheduling-api-v1-overlay.json",
		Paths: map[string]map[string]any{
			"/appointments": {
				"get":  op("listAcuityAppointments", "List appointments", params(query("max", "Maximum number of appointments to return."), query("minDate", "Minimum appointment date."), query("maxDate", "Maximum appointment date."), query("calendarID", "Calendar ID filter."), query("appointmentTypeID", "Appointment type ID filter.")), "", "#/components/schemas/AcuityCollection", "acuityBasic"),
				"post": op("createAcuityAppointment", "Create an appointment", nil, "#/components/schemas/AcuityObject", "#/components/schemas/AcuityObject", "acuityBasic"),
			},
			"/appointments/{appointment_id}": {
				"get":    op("getAcuityAppointment", "Get an appointment", params(path("appointment_id", "Acuity appointment ID.")), "", "#/components/schemas/AcuityObject", "acuityBasic"),
				"put":    op("updateAcuityAppointment", "Update an appointment", params(path("appointment_id", "Acuity appointment ID.")), "#/components/schemas/AcuityObject", "#/components/schemas/AcuityObject", "acuityBasic"),
				"delete": op("deleteAcuityAppointment", "Delete an appointment", params(path("appointment_id", "Acuity appointment ID.")), "", "#/components/schemas/AcuityObject", "acuityBasic"),
			},
			"/availability/times": {"get": op("listAcuityAvailabilityTimes", "List availability times", params(query("appointmentTypeID", "Appointment type ID."), query("date", "Availability date.")), "", "#/components/schemas/AcuityCollection", "acuityBasic")},
			"/appointment-types":  {"get": op("listAcuityAppointmentTypes", "List appointment types", nil, "", "#/components/schemas/AcuityCollection", "acuityBasic")},
			"/calendars":          {"get": op("listAcuityCalendars", "List calendars", nil, "", "#/components/schemas/AcuityCollection", "acuityBasic")},
			"/clients":            {"get": op("listAcuityClients", "List clients", params(query("search", "Client search term.")), "", "#/components/schemas/AcuityCollection", "acuityBasic")},
		},
	}
}

func clockifyOverlay() overlaySpec {
	security := map[string]map[string]any{
		"clockifyAPIKey":     {"type": "apiKey", "in": "header", "name": "X-Api-Key", "description": "Clockify API key carried in the X-Api-Key header."},
		"clockifyAddonToken": {"type": "apiKey", "in": "header", "name": "X-Addon-Token", "description": "Clockify addon token carried in the X-Addon-Token header."},
	}
	return overlaySpec{
		ProviderID:  "clockify",
		OverlayID:   "clockify-api-v1-advisory-overlay",
		Title:       "Clockify API v1 Advisory Overlay",
		Description: "Advisory OpenAPI overlay derived from official Clockify API human documentation. This is not an official Clockify OpenAPI document.",
		ServerURL:   "https://api.clockify.me/api",
		Sources:     []string{"https://docs.clockify.me/", "https://clockify.me/help/getting-started/clockify-api-overview"},
		SourceNote:  "Clockify publishes REST-shaped API v1 docs with X-Api-Key or X-Addon-Token authentication; this overlay covers users, workspaces, projects, clients, tags, and time entries.",
		Security:    security,
		Schemas:     []string{"ClockifyObject", "ClockifyCollection", "ClockifyError"},
		OutputPath:  "catalog-openapi-cache/advisory-overlays/clockify-api-v1-overlay.json",
		Paths: map[string]map[string]any{
			"/v1/user":       {"get": op("getClockifyCurrentUser", "Get current user", nil, "", "#/components/schemas/ClockifyObject", "clockifyAPIKey")},
			"/v1/workspaces": {"get": op("listClockifyWorkspaces", "List workspaces", nil, "", "#/components/schemas/ClockifyCollection", "clockifyAPIKey")},
			"/v1/workspaces/{workspace_id}/projects": {
				"get":  op("listClockifyProjects", "List projects", params(path("workspace_id", "Clockify workspace ID."), query("name", "Project name filter.")), "", "#/components/schemas/ClockifyCollection", "clockifyAPIKey"),
				"post": op("createClockifyProject", "Create a project", params(path("workspace_id", "Clockify workspace ID.")), "#/components/schemas/ClockifyObject", "#/components/schemas/ClockifyObject", "clockifyAPIKey"),
			},
			"/v1/workspaces/{workspace_id}/time-entries": {
				"get":  op("listClockifyTimeEntries", "List time entries", params(path("workspace_id", "Clockify workspace ID."), query("start", "Start date-time filter."), query("end", "End date-time filter.")), "", "#/components/schemas/ClockifyCollection", "clockifyAPIKey"),
				"post": op("createClockifyTimeEntry", "Create a time entry", params(path("workspace_id", "Clockify workspace ID.")), "#/components/schemas/ClockifyObject", "#/components/schemas/ClockifyObject", "clockifyAPIKey"),
			},
			"/v1/workspaces/{workspace_id}/time-entries/{time_entry_id}": {
				"get":    op("getClockifyTimeEntry", "Get a time entry", params(path("workspace_id", "Clockify workspace ID."), path("time_entry_id", "Clockify time entry ID.")), "", "#/components/schemas/ClockifyObject", "clockifyAPIKey"),
				"delete": op("deleteClockifyTimeEntry", "Delete a time entry", params(path("workspace_id", "Clockify workspace ID."), path("time_entry_id", "Clockify time entry ID.")), "", "#/components/schemas/ClockifyObject", "clockifyAPIKey"),
			},
			"/v1/workspaces/{workspace_id}/clients": {"get": op("listClockifyClients", "List clients", params(path("workspace_id", "Clockify workspace ID.")), "", "#/components/schemas/ClockifyCollection", "clockifyAPIKey")},
			"/v1/workspaces/{workspace_id}/tags":    {"get": op("listClockifyTags", "List tags", params(path("workspace_id", "Clockify workspace ID.")), "", "#/components/schemas/ClockifyCollection", "clockifyAPIKey")},
		},
	}
}

func ghostOverlay() overlaySpec {
	security := map[string]map[string]any{"ghostAdminToken": {"type": "http", "scheme": "bearer", "description": "Ghost Admin API token carried in the Authorization header after operator-side JWT generation."}}
	return overlaySpec{
		ProviderID:  "ghost",
		OverlayID:   "ghost-admin-api-advisory-overlay",
		Title:       "Ghost Admin API Advisory Overlay",
		Description: "Advisory OpenAPI overlay derived from official Ghost Admin API human documentation. This is not an official Ghost OpenAPI document.",
		ServerURL:   "https://{admin_domain}/ghost/api/admin",
		ServerVars:  map[string]map[string]any{"admin_domain": {"default": "example.com", "description": "Operator-supplied Ghost Admin domain."}},
		Sources:     []string{"https://docs.ghost.org/admin-api/", "https://docs.ghost.org/content-api/"},
		SourceNote:  "Ghost publishes REST-shaped Admin API docs with JWT bearer authorization; this overlay covers posts, pages, tags, users, and members.",
		Security:    security,
		Schemas:     []string{"GhostObject", "GhostCollection", "GhostError"},
		OutputPath:  "catalog-openapi-cache/advisory-overlays/ghost-admin-api-overlay.json",
		Paths: map[string]map[string]any{
			"/posts/": {
				"get":  op("listGhostPosts", "List posts", params(query("limit", "Maximum number of posts to return."), query("filter", "Ghost filter expression.")), "", "#/components/schemas/GhostCollection", "ghostAdminToken"),
				"post": op("createGhostPost", "Create a post", nil, "#/components/schemas/GhostObject", "#/components/schemas/GhostObject", "ghostAdminToken"),
			},
			"/posts/{post_id}/": {
				"get":    op("getGhostPost", "Get a post", params(path("post_id", "Ghost post ID.")), "", "#/components/schemas/GhostObject", "ghostAdminToken"),
				"put":    op("updateGhostPost", "Update a post", params(path("post_id", "Ghost post ID.")), "#/components/schemas/GhostObject", "#/components/schemas/GhostObject", "ghostAdminToken"),
				"delete": op("deleteGhostPost", "Delete a post", params(path("post_id", "Ghost post ID.")), "", "#/components/schemas/GhostObject", "ghostAdminToken"),
			},
			"/pages/":   {"get": op("listGhostPages", "List pages", nil, "", "#/components/schemas/GhostCollection", "ghostAdminToken")},
			"/tags/":    {"get": op("listGhostTags", "List tags", nil, "", "#/components/schemas/GhostCollection", "ghostAdminToken")},
			"/users/":   {"get": op("listGhostUsers", "List users", nil, "", "#/components/schemas/GhostCollection", "ghostAdminToken")},
			"/members/": {"get": op("listGhostMembers", "List members", nil, "", "#/components/schemas/GhostCollection", "ghostAdminToken")},
		},
	}
}

func harvestOverlay() overlaySpec {
	security := map[string]map[string]any{"harvestBearer": {"type": "http", "scheme": "bearer", "description": "Harvest OAuth or personal access token carried as an Authorization bearer token."}}
	accountID := header("Harvest-Account-ID", "Operator-supplied Harvest account ID.")
	return overlaySpec{
		ProviderID:  "harvest",
		OverlayID:   "harvest-api-v2-advisory-overlay",
		Title:       "Harvest API v2 Advisory Overlay",
		Description: "Advisory OpenAPI overlay derived from official Harvest API v2 human documentation. This is not an official Harvest OpenAPI document.",
		ServerURL:   "https://api.harvestapp.com/v2",
		Sources:     []string{"https://help.getharvest.com/api-v2/", "https://help.getharvest.com/api-v2/authentication-api/authentication/authentication/"},
		SourceNote:  "Harvest publishes REST-shaped API v2 docs with bearer authorization and Harvest-Account-ID header metadata; this overlay covers users, company, clients, projects, time entries, and invoices.",
		Security:    security,
		Schemas:     []string{"HarvestObject", "HarvestCollection", "HarvestError"},
		OutputPath:  "catalog-openapi-cache/advisory-overlays/harvest-api-v2-overlay.json",
		Paths: map[string]map[string]any{
			"/users/me": {"get": op("getHarvestCurrentUser", "Get current user", params(accountID), "", "#/components/schemas/HarvestObject", "harvestBearer")},
			"/company":  {"get": op("getHarvestCompany", "Get company", params(accountID), "", "#/components/schemas/HarvestObject", "harvestBearer")},
			"/clients": {
				"get":  op("listHarvestClients", "List clients", params(accountID, query("page", "Page number.")), "", "#/components/schemas/HarvestCollection", "harvestBearer"),
				"post": op("createHarvestClient", "Create a client", params(accountID), "#/components/schemas/HarvestObject", "#/components/schemas/HarvestObject", "harvestBearer"),
			},
			"/projects":     {"get": op("listHarvestProjects", "List projects", params(accountID, query("client_id", "Client ID filter.")), "", "#/components/schemas/HarvestCollection", "harvestBearer")},
			"/time_entries": {"get": op("listHarvestTimeEntries", "List time entries", params(accountID, query("from", "Start date filter."), query("to", "End date filter.")), "", "#/components/schemas/HarvestCollection", "harvestBearer")},
			"/time_entries/{time_entry_id}": {
				"get":    op("getHarvestTimeEntry", "Get a time entry", params(accountID, path("time_entry_id", "Harvest time entry ID.")), "", "#/components/schemas/HarvestObject", "harvestBearer"),
				"delete": op("deleteHarvestTimeEntry", "Delete a time entry", params(accountID, path("time_entry_id", "Harvest time entry ID.")), "", "#/components/schemas/HarvestObject", "harvestBearer"),
			},
			"/invoices": {"get": op("listHarvestInvoices", "List invoices", params(accountID, query("client_id", "Client ID filter.")), "", "#/components/schemas/HarvestCollection", "harvestBearer")},
		},
	}
}

func helpScoutOverlay() overlaySpec {
	security := map[string]map[string]any{"helpScoutOAuth": {"type": "http", "scheme": "bearer", "description": "Help Scout Inbox API OAuth 2 access token carried as an Authorization bearer token."}}
	return overlaySpec{
		ProviderID:  "help-scout",
		OverlayID:   "help-scout-inbox-api-v2-advisory-overlay",
		Title:       "Help Scout Inbox API v2 Advisory Overlay",
		Description: "Advisory OpenAPI overlay derived from official Help Scout Inbox API human documentation. This is not an official Help Scout OpenAPI document.",
		ServerURL:   "https://api.helpscout.net/v2",
		Sources:     []string{"https://developer.helpscout.com/", "https://developer.helpscout.com/mailbox-api/", "https://developer.helpscout.com/docs-api/"},
		SourceNote:  "Help Scout publishes REST-shaped Inbox API v2 docs with OAuth bearer authentication; this overlay covers conversations, customers, mailboxes, and threads.",
		Security:    security,
		Schemas:     []string{"HelpScoutObject", "HelpScoutCollection", "HelpScoutError"},
		OutputPath:  "catalog-openapi-cache/advisory-overlays/help-scout-inbox-api-v2-overlay.json",
		Paths: map[string]map[string]any{
			"/conversations": {
				"get":  op("listHelpScoutConversations", "List conversations", params(query("mailbox", "Mailbox ID filter."), query("status", "Conversation status filter.")), "", "#/components/schemas/HelpScoutCollection", "helpScoutOAuth"),
				"post": op("createHelpScoutConversation", "Create a conversation", nil, "#/components/schemas/HelpScoutObject", "#/components/schemas/HelpScoutObject", "helpScoutOAuth"),
			},
			"/conversations/{conversation_id}": {
				"get": op("getHelpScoutConversation", "Get a conversation", params(path("conversation_id", "Help Scout conversation ID.")), "", "#/components/schemas/HelpScoutObject", "helpScoutOAuth"),
			},
			"/conversations/{conversation_id}/threads": {"post": op("createHelpScoutThread", "Create a conversation thread", params(path("conversation_id", "Help Scout conversation ID.")), "#/components/schemas/HelpScoutObject", "#/components/schemas/HelpScoutObject", "helpScoutOAuth")},
			"/customers":               {"get": op("listHelpScoutCustomers", "List customers", params(query("query", "Customer search query.")), "", "#/components/schemas/HelpScoutCollection", "helpScoutOAuth")},
			"/customers/{customer_id}": {"get": op("getHelpScoutCustomer", "Get a customer", params(path("customer_id", "Help Scout customer ID.")), "", "#/components/schemas/HelpScoutObject", "helpScoutOAuth")},
			"/mailboxes":               {"get": op("listHelpScoutMailboxes", "List mailboxes", nil, "", "#/components/schemas/HelpScoutCollection", "helpScoutOAuth")},
		},
	}
}

func iterableOverlay() overlaySpec {
	security := map[string]map[string]any{"iterableAPIKey": {"type": "apiKey", "in": "header", "name": "Api-Key", "description": "Iterable API key carried in the Api-Key header."}}
	return overlaySpec{
		ProviderID:  "iterable",
		OverlayID:   "iterable-api-advisory-overlay",
		Title:       "Iterable API Advisory Overlay",
		Description: "Advisory OpenAPI overlay derived from official Iterable API human documentation. This is not an official Iterable OpenAPI document.",
		ServerURL:   "https://api.iterable.com/api",
		Sources:     []string{"https://api.iterable.com/api/docs", "https://support.iterable.com/hc/en-us/articles/41044692130196-Getting-Started-with-Iterable-s-API", "https://support.iterable.com/hc/en-us/articles/360043464871-API-Keys"},
		SourceNote:  "Iterable publishes REST-shaped API docs with Api-Key header authentication; this overlay covers users, events, campaigns, lists, and templates.",
		Security:    security,
		Schemas:     []string{"IterableObject", "IterableCollection", "IterableError"},
		OutputPath:  "catalog-openapi-cache/advisory-overlays/iterable-api-overlay.json",
		Paths: map[string]map[string]any{
			"/users/update":    {"post": op("updateIterableUser", "Update a user", nil, "#/components/schemas/IterableObject", "#/components/schemas/IterableObject", "iterableAPIKey")},
			"/users/{email}":   {"get": op("getIterableUserByEmail", "Get a user by email", params(path("email", "Iterable user email.")), "", "#/components/schemas/IterableObject", "iterableAPIKey")},
			"/events/track":    {"post": op("trackIterableEvent", "Track an event", nil, "#/components/schemas/IterableObject", "#/components/schemas/IterableObject", "iterableAPIKey")},
			"/events/commerce": {"post": op("trackIterableCommerceEvent", "Track a commerce event", nil, "#/components/schemas/IterableObject", "#/components/schemas/IterableObject", "iterableAPIKey")},
			"/campaigns":       {"get": op("listIterableCampaigns", "List campaigns", nil, "", "#/components/schemas/IterableCollection", "iterableAPIKey")},
			"/lists":           {"get": op("listIterableLists", "List lists", nil, "", "#/components/schemas/IterableCollection", "iterableAPIKey")},
			"/templates/email": {"get": op("listIterableEmailTemplates", "List email templates", nil, "", "#/components/schemas/IterableCollection", "iterableAPIKey")},
		},
	}
}

func mailjetOverlay() overlaySpec {
	security := map[string]map[string]any{"mailjetBasic": {"type": "http", "scheme": "basic", "description": "Mailjet API key and secret key carried as HTTP Basic username and password credentials."}}
	return overlaySpec{
		ProviderID:  "mailjet",
		OverlayID:   "mailjet-rest-api-advisory-overlay",
		Title:       "Mailjet REST API Advisory Overlay",
		Description: "Advisory OpenAPI overlay derived from official Mailjet API human documentation. This is not an official Mailjet OpenAPI document.",
		ServerURL:   "https://api.mailjet.com",
		Sources:     []string{"https://documentation.mailjet.com/hc/en-us/articles/360044088173-REST-API", "https://documentation.mailjet.com/hc/en-us/articles/360043225693-What-is-an-API-key", "https://documentation.mailjet.com/hc/en-us/articles/360043230093-What-are-the-endpoints-available-for-the-API", "https://github.com/mailjet/api-documentation"},
		SourceNote:  "Mailjet publishes REST API docs and official documentation sources; this overlay covers contacts, list recipients, campaigns, messages, templates, and send API review endpoints.",
		Security:    security,
		Schemas:     []string{"MailjetObject", "MailjetCollection", "MailjetError"},
		OutputPath:  "catalog-openapi-cache/advisory-overlays/mailjet-rest-api-overlay.json",
		Paths: map[string]map[string]any{
			"/v3/REST/contact": {
				"get":  op("listMailjetContacts", "List contacts", params(query("Limit", "Maximum number of contacts to return."), query("Offset", "Offset for pagination.")), "", "#/components/schemas/MailjetCollection", "mailjetBasic"),
				"post": op("createMailjetContact", "Create a contact", nil, "#/components/schemas/MailjetObject", "#/components/schemas/MailjetObject", "mailjetBasic"),
			},
			"/v3/REST/contact/{contact_id}": {
				"get": op("getMailjetContact", "Get a contact", params(path("contact_id", "Mailjet contact ID.")), "", "#/components/schemas/MailjetObject", "mailjetBasic"),
			},
			"/v3/REST/listrecipient": {"get": op("listMailjetListRecipients", "List recipients", nil, "", "#/components/schemas/MailjetCollection", "mailjetBasic")},
			"/v3/REST/campaign":      {"get": op("listMailjetCampaigns", "List campaigns", nil, "", "#/components/schemas/MailjetCollection", "mailjetBasic")},
			"/v3/REST/message":       {"get": op("listMailjetMessages", "List messages", nil, "", "#/components/schemas/MailjetCollection", "mailjetBasic")},
			"/v3.1/send":             {"post": op("sendMailjetEmail", "Send email", nil, "#/components/schemas/MailjetObject", "#/components/schemas/MailjetObject", "mailjetBasic")},
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
			"securitySchemes": spec.Security,
			"schemas":         schemas(spec.Schemas),
		},
		"paths":    spec.Paths,
		"security": rootSecurity(spec.Security),
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

func rootSecurity(schemes map[string]map[string]any) []map[string]any {
	req := map[string]any{}
	for name := range schemes {
		req[name] = []string{}
	}
	if len(req) == 0 {
		return nil
	}
	return []map[string]any{req}
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

func header(name, description string) map[string]any {
	return map[string]any{"name": name, "in": "header", "required": true, "schema": map[string]any{"type": "string"}, "description": description}
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
