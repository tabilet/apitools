//go:build ignore

package main

import (
	"encoding/json"
	"os"
)

const provider = "Typeform"

func main() {
	doc := map[string]any{
		"openapi": "3.0.3",
		"info": map[string]any{
			"title":       "Typeform REST API Advisory Overlay",
			"version":     "2026-05-18",
			"description": "Advisory OpenAPI overlay derived from official Typeform human documentation. This is not an official Typeform OpenAPI document.",
		},
		"servers": []map[string]any{{"url": "https://api.typeform.com"}},
		"x-apitools-overlay": overlayMeta("typeform", "typeform-rest-api-overlay", []string{
			"https://www.typeform.com/developers/get-started/",
			"https://www.typeform.com/developers/create/",
			"https://www.typeform.com/developers/get-started/personal-access-token/",
			"https://www.typeform.com/developers/get-started/applications/",
		}, "Typeform publishes official REST API human documentation, but no stable public downloadable OpenAPI document is recorded in the apitools catalog."),
		"components": map[string]any{
			"securitySchemes": map[string]any{
				"typeformBearer": map[string]any{"type": "http", "scheme": "bearer", "bearerFormat": "Personal access token or OAuth 2.0 access token", "description": "Typeform personal access token or OAuth access token carried in the Authorization header."},
			},
			"schemas": commonSchemas([]string{"TypeformObject", "TypeformCollection", "TypeformError"}),
		},
		"security": []map[string]any{{"typeformBearer": []string{}}},
		"paths": map[string]any{
			"/forms": map[string]any{
				"get":  operation("listTypeformForms", "List forms", nil, "", "#/components/schemas/TypeformCollection", "typeformBearer"),
				"post": operation("createTypeformForm", "Create a form", nil, "#/components/schemas/TypeformObject", "#/components/schemas/TypeformObject", "typeformBearer"),
			},
			"/forms/{form_id}": map[string]any{
				"get":    operation("getTypeformForm", "Get a form", formParams(), "", "#/components/schemas/TypeformObject", "typeformBearer"),
				"put":    operation("updateTypeformForm", "Update a form", formParams(), "#/components/schemas/TypeformObject", "#/components/schemas/TypeformObject", "typeformBearer"),
				"delete": operation("deleteTypeformForm", "Delete a form", formParams(), "", "#/components/schemas/TypeformObject", "typeformBearer"),
			},
			"/forms/{form_id}/responses": map[string]any{
				"get": operation("listTypeformResponses", "List form responses", append(formParams(), queryParam("page_size", "Maximum number of responses to return."), queryParam("since", "Lower bound response submitted timestamp."), queryParam("until", "Upper bound response submitted timestamp.")), "", "#/components/schemas/TypeformCollection", "typeformBearer"),
			},
			"/forms/{form_id}/webhooks/{tag}": map[string]any{
				"get":    operation("getTypeformWebhook", "Get a webhook", webhookParams(), "", "#/components/schemas/TypeformObject", "typeformBearer"),
				"put":    operation("upsertTypeformWebhook", "Create or update a webhook", webhookParams(), "#/components/schemas/TypeformObject", "#/components/schemas/TypeformObject", "typeformBearer"),
				"delete": operation("deleteTypeformWebhook", "Delete a webhook", webhookParams(), "", "#/components/schemas/TypeformObject", "typeformBearer"),
			},
			"/workspaces": map[string]any{
				"get": operation("listTypeformWorkspaces", "List workspaces", nil, "", "#/components/schemas/TypeformCollection", "typeformBearer"),
			},
		},
	}
	write("catalog-openapi-cache/advisory-overlays/typeform-rest-api-overlay.json", doc)
}

func formParams() []map[string]any {
	return []map[string]any{pathParam("form_id", "Typeform form ID.")}
}

func webhookParams() []map[string]any {
	return []map[string]any{pathParam("form_id", "Typeform form ID."), pathParam("tag", "Webhook tag.")}
}

func overlayMeta(providerID, overlayID string, sources []string, note string) map[string]any {
	return map[string]any{"provider_id": providerID, "overlay_id": overlayID, "official_openapi": false, "derived_from_docs": true, "source_refs": sources, "source_note": note}
}

func commonSchemas(names []string) map[string]any {
	schemas := map[string]any{}
	for _, name := range names {
		schemas[name] = map[string]any{"type": "object", "additionalProperties": true}
	}
	return schemas
}

func operation(operationID, summary string, parameters []map[string]any, requestSchema, responseSchema string, securitySchemes ...string) map[string]any {
	op := map[string]any{
		"operationId": operationID,
		"summary":     summary,
		"description": "Advisory operation derived from official " + provider + " human documentation.",
		"security":    security(securitySchemes...),
		"responses": map[string]any{
			"200":     response("Successful response.", responseSchema),
			"default": response(provider+" error response.", "#/components/schemas/TypeformError"),
		},
	}
	if len(parameters) > 0 {
		op["parameters"] = parameters
	}
	if requestSchema != "" {
		op["requestBody"] = requestBody(requestSchema)
	}
	return op
}

func security(names ...string) []map[string]any {
	out := make([]map[string]any, 0, len(names))
	for _, name := range names {
		out = append(out, map[string]any{name: []string{}})
	}
	return out
}

func response(description, schema string) map[string]any {
	return map[string]any{"description": description, "content": map[string]any{"application/json": map[string]any{"schema": map[string]any{"$ref": schema}}}}
}

func requestBody(schema string) map[string]any {
	return map[string]any{"required": true, "content": map[string]any{"application/json": map[string]any{"schema": map[string]any{"$ref": schema}}}}
}

func pathParam(name, description string) map[string]any {
	return map[string]any{"name": name, "in": "path", "required": true, "schema": map[string]any{"type": "string"}, "description": description}
}

func queryParam(name, description string) map[string]any {
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
