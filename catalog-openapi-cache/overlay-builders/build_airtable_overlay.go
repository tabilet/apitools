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
			"title":       "Airtable Web API Advisory Overlay",
			"version":     "2026-05-18",
			"description": "Advisory OpenAPI overlay derived from official Airtable Web API human documentation. This is not an official Airtable OpenAPI document.",
		},
		"servers": []map[string]any{{"url": "https://api.airtable.com"}},
		"x-apitools-overlay": map[string]any{
			"provider_id":       "airtable",
			"overlay_id":        "airtable-web-api-advisory-overlay",
			"official_openapi":  false,
			"derived_from_docs": true,
			"source_refs": []string{
				"https://airtable.com/developers/web/api/introduction",
				"https://airtable.com/developers/web/api/oauth-integration",
				"https://airtable.com/developers/web/api/webhooks-overview",
				"https://airtable.com/developers/web/api/create-a-webhook",
				"https://airtable.com/developers/web/api/list-webhooks",
				"https://airtable.com/developers/web/api/list-webhook-payloads",
				"https://airtable.com/developers/web/api/delete-a-webhook",
				"https://airtable.com/developers/web/api/refresh-a-webhook",
				"https://support.airtable.com/docs/getting-started-with-airtables-web-api",
				"https://support.airtable.com/docs/airtable-webhooks-api-overview",
			},
			"source_note": "Airtable publishes official Web API and Webhooks API human documentation and client libraries, but no official OpenAPI document is recorded in the apitools catalog.",
		},
		"components": map[string]any{
			"securitySchemes": map[string]any{
				"airtableBearer": map[string]any{
					"type":        "http",
					"scheme":      "bearer",
					"description": "Airtable Personal Access Token or OAuth access token carried in the Authorization header.",
				},
			},
			"schemas": map[string]any{
				"AirtableRecord": map[string]any{
					"type":     "object",
					"required": []string{"id"},
					"properties": map[string]any{
						"id":          map[string]any{"type": "string"},
						"createdTime": map[string]any{"type": "string", "format": "date-time"},
						"fields":      map[string]any{"type": "object", "additionalProperties": true},
					},
				},
				"AirtableRecordsRequest": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"records":  map[string]any{"type": "array", "items": map[string]any{"type": "object", "properties": map[string]any{"fields": map[string]any{"type": "object", "additionalProperties": true}}}},
						"typecast": map[string]any{"type": "boolean"},
					},
				},
				"AirtableRecordsResponse": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"records": map[string]any{"type": "array", "items": map[string]any{"$ref": "#/components/schemas/AirtableRecord"}},
						"offset":  map[string]any{"type": "string"},
					},
				},
				"AirtableBaseList": map[string]any{
					"type":                 "object",
					"additionalProperties": true,
				},
				"AirtableTableList": map[string]any{
					"type":                 "object",
					"additionalProperties": true,
				},
				"AirtableWebhook": map[string]any{
					"type":        "object",
					"description": "Advisory schema for an Airtable webhook registration.",
					"properties": map[string]any{
						"id":                      map[string]any{"type": "string"},
						"areNotificationsEnabled": map[string]any{"type": "boolean"},
						"cursorForNextPayload":    map[string]any{"type": "integer"},
						"expirationTime":          map[string]any{"type": "string", "format": "date-time"},
						"specification":           map[string]any{"type": "object", "additionalProperties": true},
					},
				},
				"AirtableWebhookList": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"webhooks": map[string]any{"type": "array", "items": map[string]any{"$ref": "#/components/schemas/AirtableWebhook"}},
					},
				},
				"AirtableWebhookRequest": map[string]any{
					"type":        "object",
					"description": "Advisory schema for creating an Airtable webhook. The notification URL and specification options are workflow-specific.",
					"properties": map[string]any{
						"notificationUrl": map[string]any{"type": "string", "format": "uri"},
						"specification":   map[string]any{"type": "object", "additionalProperties": true},
					},
				},
				"AirtableWebhookPayloads": map[string]any{
					"type":        "object",
					"description": "Advisory schema for Airtable webhook payload retrieval.",
					"properties": map[string]any{
						"payloads":      map[string]any{"type": "array", "items": map[string]any{"type": "object", "additionalProperties": true}},
						"cursor":        map[string]any{"type": "integer"},
						"mightHaveMore": map[string]any{"type": "boolean"},
					},
				},
				"AirtableError": map[string]any{
					"type":                 "object",
					"additionalProperties": true,
				},
			},
		},
		"security": []map[string]any{{"airtableBearer": []string{}}},
		"paths": map[string]any{
			"/v0/meta/bases": map[string]any{
				"get": operation("listAirtableBases", "List Airtable bases", nil, "", "#/components/schemas/AirtableBaseList"),
			},
			"/v0/meta/bases/{baseId}/tables": map[string]any{
				"get": operation("listAirtableBaseTables", "List Airtable base tables", []map[string]any{pathParam("baseId", "Airtable base ID.")}, "", "#/components/schemas/AirtableTableList"),
			},
			"/v0/{baseId}/{tableIdOrName}": map[string]any{
				"get":   operation("listAirtableRecords", "List Airtable records", append(recordPathParams(), listRecordQueryParams()...), "", "#/components/schemas/AirtableRecordsResponse"),
				"post":  operation("createAirtableRecords", "Create Airtable records", recordPathParams(), "#/components/schemas/AirtableRecordsRequest", "#/components/schemas/AirtableRecordsResponse"),
				"patch": operation("updateAirtableRecords", "Update Airtable records", recordPathParams(), "#/components/schemas/AirtableRecordsRequest", "#/components/schemas/AirtableRecordsResponse"),
			},
			"/v0/{baseId}/{tableIdOrName}/{recordId}": map[string]any{
				"get":    operation("getAirtableRecord", "Get an Airtable record", append(recordPathParams(), pathParam("recordId", "Airtable record ID.")), "", "#/components/schemas/AirtableRecord"),
				"delete": operation("deleteAirtableRecord", "Delete an Airtable record", append(recordPathParams(), pathParam("recordId", "Airtable record ID.")), "", "#/components/schemas/AirtableRecord"),
			},
			"/v0/bases/{baseId}/webhooks": map[string]any{
				"get":  operation("listAirtableWebhooks", "List Airtable webhooks", []map[string]any{pathParam("baseId", "Airtable base ID.")}, "", "#/components/schemas/AirtableWebhookList"),
				"post": operation("createAirtableWebhook", "Create an Airtable webhook", []map[string]any{pathParam("baseId", "Airtable base ID.")}, "#/components/schemas/AirtableWebhookRequest", "#/components/schemas/AirtableWebhook"),
			},
			"/v0/bases/{baseId}/webhooks/{webhookId}": map[string]any{
				"delete": operation("deleteAirtableWebhook", "Delete an Airtable webhook", []map[string]any{pathParam("baseId", "Airtable base ID."), pathParam("webhookId", "Airtable webhook ID.")}, "", "#/components/schemas/AirtableWebhook"),
			},
			"/v0/bases/{baseId}/webhooks/{webhookId}/payloads": map[string]any{
				"get": operation("listAirtableWebhookPayloads", "List Airtable webhook payloads", []map[string]any{pathParam("baseId", "Airtable base ID."), pathParam("webhookId", "Airtable webhook ID."), queryParam("cursor", "Cursor for retrieving webhook payloads after a previous response.")}, "", "#/components/schemas/AirtableWebhookPayloads"),
			},
			"/v0/bases/{baseId}/webhooks/{webhookId}/refresh": map[string]any{
				"post": operation("refreshAirtableWebhook", "Refresh an Airtable webhook", []map[string]any{pathParam("baseId", "Airtable base ID."), pathParam("webhookId", "Airtable webhook ID.")}, "", "#/components/schemas/AirtableWebhook"),
			},
		},
	}
	write("catalog-openapi-cache/advisory-overlays/airtable-web-api-overlay.json", doc)
}

func operation(operationID, summary string, parameters []map[string]any, requestSchema, responseSchema string) map[string]any {
	op := map[string]any{
		"operationId": operationID,
		"summary":     summary,
		"description": "Advisory operation derived from official Airtable human documentation.",
		"security":    []map[string]any{{"airtableBearer": []string{}}},
		"responses": map[string]any{
			"200": map[string]any{
				"description": "Airtable response.",
				"content":     map[string]any{"application/json": map[string]any{"schema": map[string]any{"$ref": responseSchema}}},
			},
			"default": map[string]any{
				"description": "Airtable error response.",
				"content":     map[string]any{"application/json": map[string]any{"schema": map[string]any{"$ref": "#/components/schemas/AirtableError"}}},
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

func recordPathParams() []map[string]any {
	return []map[string]any{
		pathParam("baseId", "Airtable base ID."),
		pathParam("tableIdOrName", "Airtable table ID or table name."),
	}
}

func listRecordQueryParams() []map[string]any {
	return []map[string]any{
		{"name": "pageSize", "in": "query", "required": false, "schema": map[string]any{"type": "integer", "minimum": 1, "maximum": 100}, "description": "Maximum number of records per page. Airtable documents 100 as the maximum."},
		queryParam("offset", "Pagination offset from a previous response."),
		queryParam("view", "View name or ID."),
		queryParam("filterByFormula", "Airtable formula used to filter records."),
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

func pathParam(name, description string) map[string]any {
	return map[string]any{
		"name":        name,
		"in":          "path",
		"required":    true,
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
