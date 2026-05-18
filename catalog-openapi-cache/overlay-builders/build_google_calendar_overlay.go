//go:build ignore

package main

import (
	"encoding/json"
	"os"
)

func main() {
	doc := map[string]any{
		"openapi": "3.0.3",
		"info": map[string]any{
			"title":       "Google Calendar API v3 Advisory Overlay",
			"version":     "2026-05-18",
			"description": "Advisory OpenAPI overlay derived from the official Google Calendar Discovery document and REST documentation. This is not an official Google OpenAPI document.",
		},
		"servers": []map[string]any{{"url": "https://www.googleapis.com/calendar/v3"}},
		"x-apitools-overlay": map[string]any{
			"provider_id":       "google-calendar",
			"overlay_id":        "google-calendar-v3-advisory-overlay",
			"official_openapi":  false,
			"derived_from_docs": true,
			"source_refs": []string{
				"https://www.googleapis.com/discovery/v1/apis/calendar/v3/rest",
				"https://developers.google.com/workspace/calendar/api/v3/reference",
				"https://developers.google.com/workspace/calendar/api/auth",
			},
			"source_note": "Google Calendar publishes an official Discovery document, but no official OpenAPI document is recorded in the apitools catalog.",
		},
		"components": map[string]any{
			"securitySchemes": map[string]any{
				"googleOAuth2": googleOAuthScheme(map[string]string{
					"https://www.googleapis.com/auth/calendar.calendarlist.readonly": "Read calendar list entries.",
					"https://www.googleapis.com/auth/calendar.calendars":             "Create and manage calendars.",
					"https://www.googleapis.com/auth/calendar.events":                "Read and write calendar events.",
					"https://www.googleapis.com/auth/calendar.events.readonly":       "Read calendar events.",
				}),
			},
			"schemas": map[string]any{
				"CalendarList": objectSchema(),
				"Calendar":     objectSchema(),
				"Event":        objectSchema(),
				"EventRequest": objectSchema(),
				"Error":        objectSchema(),
			},
		},
		"security": []map[string]any{{"googleOAuth2": []string{"https://www.googleapis.com/auth/calendar.events.readonly"}}},
		"paths": map[string]any{
			"/users/me/calendarList": map[string]any{
				"get": operation("listGoogleCalendarList", "List Google Calendar calendar-list entries", nil, "", "#/components/schemas/CalendarList", []string{"https://www.googleapis.com/auth/calendar.calendarlist.readonly"}),
			},
			"/calendars": map[string]any{
				"post": operation("insertGoogleCalendar", "Create a secondary Google Calendar", nil, "#/components/schemas/Calendar", "#/components/schemas/Calendar", []string{"https://www.googleapis.com/auth/calendar.calendars"}),
			},
			"/calendars/{calendarId}": map[string]any{
				"get":    operation("getGoogleCalendar", "Get Google Calendar metadata", []map[string]any{pathParam("calendarId", "Calendar identifier.")}, "", "#/components/schemas/Calendar", []string{"https://www.googleapis.com/auth/calendar.events.readonly"}),
				"patch":  operation("patchGoogleCalendar", "Patch Google Calendar metadata", []map[string]any{pathParam("calendarId", "Calendar identifier.")}, "#/components/schemas/Calendar", "#/components/schemas/Calendar", []string{"https://www.googleapis.com/auth/calendar.calendars"}),
				"delete": operation("deleteGoogleCalendar", "Delete a secondary Google Calendar", []map[string]any{pathParam("calendarId", "Calendar identifier.")}, "", "#/components/schemas/Calendar", []string{"https://www.googleapis.com/auth/calendar.calendars"}),
			},
			"/calendars/{calendarId}/events": map[string]any{
				"get": operation("listGoogleCalendarEvents", "List Google Calendar events", []map[string]any{
					pathParam("calendarId", "Calendar identifier."),
					queryParam("timeMin", "Lower bound for event start time."),
					queryParam("timeMax", "Upper bound for event start time."),
					queryParam("q", "Free text search terms."),
					queryParam("pageToken", "Pagination token."),
				}, "", "#/components/schemas/CalendarList", []string{"https://www.googleapis.com/auth/calendar.events.readonly"}),
				"post": operation("insertGoogleCalendarEvent", "Create a Google Calendar event", []map[string]any{pathParam("calendarId", "Calendar identifier.")}, "#/components/schemas/EventRequest", "#/components/schemas/Event", []string{"https://www.googleapis.com/auth/calendar.events"}),
			},
			"/calendars/{calendarId}/events/{eventId}": map[string]any{
				"get":    operation("getGoogleCalendarEvent", "Get a Google Calendar event", []map[string]any{pathParam("calendarId", "Calendar identifier."), pathParam("eventId", "Event identifier.")}, "", "#/components/schemas/Event", []string{"https://www.googleapis.com/auth/calendar.events.readonly"}),
				"patch":  operation("patchGoogleCalendarEvent", "Patch a Google Calendar event", []map[string]any{pathParam("calendarId", "Calendar identifier."), pathParam("eventId", "Event identifier.")}, "#/components/schemas/EventRequest", "#/components/schemas/Event", []string{"https://www.googleapis.com/auth/calendar.events"}),
				"delete": operation("deleteGoogleCalendarEvent", "Delete a Google Calendar event", []map[string]any{pathParam("calendarId", "Calendar identifier."), pathParam("eventId", "Event identifier.")}, "", "#/components/schemas/Event", []string{"https://www.googleapis.com/auth/calendar.events"}),
			},
		},
	}
	write("catalog-openapi-cache/advisory-overlays/google-calendar-v3-overlay.json", doc)
}

func googleOAuthScheme(scopes map[string]string) map[string]any {
	return map[string]any{
		"type": "oauth2",
		"flows": map[string]any{
			"authorizationCode": map[string]any{
				"authorizationUrl": "https://accounts.google.com/o/oauth2/v2/auth",
				"tokenUrl":         "https://oauth2.googleapis.com/token",
				"scopes":           scopes,
			},
		},
		"description": "Google OAuth 2.0 authorization code flow with Calendar API scopes.",
	}
}

func operation(operationID, summary string, parameters []map[string]any, requestSchema, responseSchema string, scopes []string) map[string]any {
	op := map[string]any{
		"operationId": operationID,
		"summary":     summary,
		"description": "Advisory operation derived from official Google Calendar Discovery and REST documentation.",
		"security":    []map[string]any{{"googleOAuth2": scopes}},
		"responses": map[string]any{
			"200": response("Google Calendar response.", responseSchema),
			"default": map[string]any{
				"description": "Google API error response.",
				"content":     map[string]any{"application/json": map[string]any{"schema": map[string]any{"$ref": "#/components/schemas/Error"}}},
			},
		},
	}
	if len(parameters) > 0 {
		op["parameters"] = parameters
	}
	if requestSchema != "" {
		op["requestBody"] = map[string]any{
			"required": true,
			"content":  map[string]any{"application/json": map[string]any{"schema": map[string]any{"$ref": requestSchema}}},
		}
	}
	return op
}

func response(description, schema string) map[string]any {
	return map[string]any{
		"description": description,
		"content":     map[string]any{"application/json": map[string]any{"schema": map[string]any{"$ref": schema}}},
	}
}

func pathParam(name, description string) map[string]any {
	return map[string]any{
		"name":        name,
		"in":          "path",
		"required":    true,
		"schema":      map[string]any{"type": "string"},
		"description": description,
	}
}

func queryParam(name, description string) map[string]any {
	return map[string]any{
		"name":        name,
		"in":          "query",
		"required":    false,
		"schema":      map[string]any{"type": "string"},
		"description": description,
	}
}

func objectSchema() map[string]any {
	return map[string]any{"type": "object", "additionalProperties": true}
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
