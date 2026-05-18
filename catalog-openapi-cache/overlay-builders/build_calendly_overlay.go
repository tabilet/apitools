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
			"title":       "Calendly Public API Advisory Overlay",
			"version":     "2026-05-18",
			"description": "Advisory OpenAPI overlay derived from official Calendly human documentation. This is not an official Calendly OpenAPI document.",
		},
		"servers": []map[string]any{{"url": "https://api.calendly.com"}},
		"x-apitools-overlay": map[string]any{
			"provider_id":       "calendly",
			"overlay_id":        "calendly-public-api-advisory-overlay",
			"official_openapi":  false,
			"derived_from_docs": true,
			"source_refs": []string{
				"https://developer.calendly.com/api-docs",
				"https://developer.calendly.com/getting-started",
				"https://developer.calendly.com/authentication",
				"https://developer.calendly.com/creating-an-oauth-app",
				"https://calendly.stoplight.io/docs/api-docs",
			},
			"source_note": "Calendly publishes official API v2 human documentation and Stoplight-hosted reference pages, but no official downloadable OpenAPI document is recorded in the apitools catalog.",
		},
		"components": map[string]any{
			"securitySchemes": map[string]any{
				"calendlyBearer": map[string]any{
					"type":        "http",
					"scheme":      "bearer",
					"description": "Calendly personal access token or OAuth 2.1 access token carried in the Authorization header.",
				},
			},
			"schemas": map[string]any{
				"CalendlyCollection": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"collection": map[string]any{"type": "array", "items": map[string]any{"type": "object", "additionalProperties": true}},
						"pagination": map[string]any{"type": "object", "additionalProperties": true},
					},
					"additionalProperties": true,
				},
				"CalendlyResource": map[string]any{
					"type":                 "object",
					"additionalProperties": true,
				},
				"SchedulingLinkRequest": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"owner":           map[string]any{"type": "string", "description": "Calendly event type URI."},
						"max_event_count": map[string]any{"type": "integer", "minimum": 1},
					},
					"additionalProperties": true,
				},
				"WebhookSubscriptionRequest": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"url":             map[string]any{"type": "string", "format": "uri"},
						"events":          map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
						"organization":    map[string]any{"type": "string"},
						"user":            map[string]any{"type": "string"},
						"scope":           map[string]any{"type": "string"},
						"signing_key":     map[string]any{"type": "string"},
						"routing_form":    map[string]any{"type": "string"},
						"subscription_id": map[string]any{"type": "string"},
					},
					"additionalProperties": true,
				},
				"CalendlyError": map[string]any{
					"type":                 "object",
					"additionalProperties": true,
				},
			},
		},
		"security": []map[string]any{{"calendlyBearer": []string{}}},
		"paths": map[string]any{
			"/users/me": map[string]any{
				"get": operation("getCalendlyCurrentUser", "Get current Calendly user", nil, "", "#/components/schemas/CalendlyResource"),
			},
			"/event_types": map[string]any{
				"get": operation("listCalendlyEventTypes", "List Calendly event types", []map[string]any{
					queryParam("user", "Calendly user URI."),
					queryParam("organization", "Calendly organization URI."),
					queryParam("count", "Maximum number of records to return."),
					queryParam("page_token", "Pagination token from a previous response."),
				}, "", "#/components/schemas/CalendlyCollection"),
			},
			"/scheduled_events": map[string]any{
				"get": operation("listCalendlyScheduledEvents", "List Calendly scheduled events", []map[string]any{
					queryParam("user", "Calendly user URI."),
					queryParam("organization", "Calendly organization URI."),
					queryParam("count", "Maximum number of records to return."),
					queryParam("page_token", "Pagination token from a previous response."),
					queryParam("min_start_time", "Lower scheduled-event start time bound."),
					queryParam("max_start_time", "Upper scheduled-event start time bound."),
					queryParam("status", "Scheduled event status."),
				}, "", "#/components/schemas/CalendlyCollection"),
			},
			"/scheduled_events/{event_uuid}": map[string]any{
				"get": operation("getCalendlyScheduledEvent", "Get a Calendly scheduled event", []map[string]any{pathParam("event_uuid", "Scheduled event UUID.")}, "", "#/components/schemas/CalendlyResource"),
			},
			"/scheduled_events/{event_uuid}/invitees": map[string]any{
				"get": operation("listCalendlyEventInvitees", "List Calendly scheduled event invitees", append([]map[string]any{pathParam("event_uuid", "Scheduled event UUID.")}, queryParam("count", "Maximum number of records to return."), queryParam("page_token", "Pagination token from a previous response.")), "", "#/components/schemas/CalendlyCollection"),
			},
			"/scheduling_links": map[string]any{
				"post": operation("createCalendlySchedulingLink", "Create a Calendly scheduling link", nil, "#/components/schemas/SchedulingLinkRequest", "#/components/schemas/CalendlyResource"),
			},
			"/webhook_subscriptions": map[string]any{
				"get": operation("listCalendlyWebhookSubscriptions", "List Calendly webhook subscriptions", []map[string]any{
					queryParam("organization", "Calendly organization URI."),
					queryParam("user", "Calendly user URI."),
					queryParam("scope", "Webhook subscription scope."),
					queryParam("count", "Maximum number of records to return."),
					queryParam("page_token", "Pagination token from a previous response."),
				}, "", "#/components/schemas/CalendlyCollection"),
				"post": operation("createCalendlyWebhookSubscription", "Create a Calendly webhook subscription", nil, "#/components/schemas/WebhookSubscriptionRequest", "#/components/schemas/CalendlyResource"),
			},
			"/webhook_subscriptions/{webhook_uuid}": map[string]any{
				"get":    operation("getCalendlyWebhookSubscription", "Get a Calendly webhook subscription", []map[string]any{pathParam("webhook_uuid", "Webhook subscription UUID.")}, "", "#/components/schemas/CalendlyResource"),
				"delete": operation("deleteCalendlyWebhookSubscription", "Delete a Calendly webhook subscription", []map[string]any{pathParam("webhook_uuid", "Webhook subscription UUID.")}, "", "#/components/schemas/CalendlyResource"),
			},
		},
	}
	write("catalog-openapi-cache/advisory-overlays/calendly-public-api-overlay.json", doc)
}

func operation(operationID, summary string, parameters []map[string]any, requestSchema, responseSchema string) map[string]any {
	op := map[string]any{
		"operationId": operationID,
		"summary":     summary,
		"description": "Advisory operation derived from official Calendly human documentation.",
		"security":    []map[string]any{{"calendlyBearer": []string{}}},
		"responses": map[string]any{
			"200": response("Calendly response.", responseSchema),
			"201": response("Calendly resource created.", responseSchema),
			"default": map[string]any{
				"description": "Calendly error response.",
				"content":     map[string]any{"application/json": map[string]any{"schema": map[string]any{"$ref": "#/components/schemas/CalendlyError"}}},
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
