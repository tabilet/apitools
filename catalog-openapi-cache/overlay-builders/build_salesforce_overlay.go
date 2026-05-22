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
			"title":       "Salesforce REST API Advisory Overlay",
			"version":     "2026-05-18",
			"description": "Advisory OpenAPI overlay derived from official Salesforce REST API human documentation. This is not an official Salesforce OpenAPI document.",
		},
		"servers": []map[string]any{{
			"url":         "https://{instanceHost}/services/data/{apiVersion}",
			"description": "Salesforce instance REST API base URL.",
			"variables": map[string]any{
				"instanceHost": map[string]any{"default": "your-domain.my.salesforce.com"},
				"apiVersion":   map[string]any{"default": "v61.0"},
			},
		}},
		"x-apitools-overlay": map[string]any{
			"provider_id":       "salesforce",
			"overlay_id":        "salesforce-rest-core-advisory-overlay",
			"official_openapi":  false,
			"derived_from_docs": true,
			"source_refs": []string{
				"https://developer.salesforce.com/docs/atlas.en-us.api_rest.meta/api_rest/intro_rest.htm",
				"https://developer.salesforce.com/docs/atlas.en-us.api_rest.meta/api_rest/resources_composite.htm",
				"https://developer.salesforce.com/docs/atlas.en-us.api_rest.meta/api_rest/resources_composite_batch.htm",
				"https://developer.salesforce.com/docs/atlas.en-us.api_rest.meta/api_rest/resources_composite_sobject_tree.htm",
				"https://developer.salesforce.com/docs/atlas.en-us.api_rest.meta/api_rest/quickstart_oauth.htm",
				"https://help.salesforce.com/s/articleView?id=release-notes.rn_api_rest.htm&language=en_US&release=236&type=5",
			},
			"source_note": "Salesforce documents REST API resources, composite resources, and org-side OpenAPI generation, but no stable public downloadable OpenAPI document is recorded in the apitools catalog.",
		},
		"components": map[string]any{
			"securitySchemes": map[string]any{
				"salesforceBearer": map[string]any{
					"type":        "http",
					"scheme":      "bearer",
					"description": "Salesforce OAuth 2.0 access token.",
				},
			},
			"schemas": map[string]any{
				"SObjectRecord":    objectSchema(),
				"SObjectList":      objectSchema(),
				"QueryResult":      objectSchema(),
				"SearchResult":     objectSchema(),
				"CompositeRequest": objectSchema(),
				"CompositeResult":  objectSchema(),
				"Error":            objectSchema(),
			},
		},
		"security": []map[string]any{{"salesforceBearer": []string{}}},
		"paths": map[string]any{
			"/": map[string]any{
				"get": operation("listSalesforceResources", "List Salesforce REST API resources", nil, "", "#/components/schemas/SObjectList"),
			},
			"/sobjects": map[string]any{
				"get": operation("listSalesforceObjects", "List Salesforce object metadata", nil, "", "#/components/schemas/SObjectList"),
			},
			"/sobjects/{objectApiName}": map[string]any{
				"get":  operation("describeSalesforceObject", "Describe a Salesforce object", []map[string]any{pathParam("objectApiName", "Salesforce object API name.")}, "", "#/components/schemas/SObjectRecord"),
				"post": operation("createSalesforceRecord", "Create a Salesforce record", []map[string]any{pathParam("objectApiName", "Salesforce object API name.")}, "#/components/schemas/SObjectRecord", "#/components/schemas/SObjectRecord"),
			},
			"/sobjects/{objectApiName}/{recordId}": map[string]any{
				"get":    operation("getSalesforceRecord", "Get a Salesforce record", []map[string]any{pathParam("objectApiName", "Salesforce object API name."), pathParam("recordId", "Salesforce record identifier.")}, "", "#/components/schemas/SObjectRecord"),
				"patch":  operation("updateSalesforceRecord", "Update a Salesforce record", []map[string]any{pathParam("objectApiName", "Salesforce object API name."), pathParam("recordId", "Salesforce record identifier.")}, "#/components/schemas/SObjectRecord", "#/components/schemas/SObjectRecord"),
				"delete": operation("deleteSalesforceRecord", "Delete a Salesforce record", []map[string]any{pathParam("objectApiName", "Salesforce object API name."), pathParam("recordId", "Salesforce record identifier.")}, "", "#/components/schemas/SObjectRecord"),
			},
			"/query": map[string]any{
				"get": operation("querySalesforce", "Run a Salesforce SOQL query", []map[string]any{queryParam("q", "SOQL query string.")}, "", "#/components/schemas/QueryResult"),
			},
			"/search": map[string]any{
				"get": operation("searchSalesforce", "Run a Salesforce SOSL search", []map[string]any{queryParam("q", "SOSL search string.")}, "", "#/components/schemas/SearchResult"),
			},
			"/composite": map[string]any{
				"post": operation("runSalesforceCompositeRequest", "Run a Salesforce composite request", nil, "#/components/schemas/CompositeRequest", "#/components/schemas/CompositeResult"),
			},
			"/composite/batch": map[string]any{
				"post": operation("runSalesforceCompositeBatchRequest", "Run a Salesforce composite batch request", nil, "#/components/schemas/CompositeRequest", "#/components/schemas/CompositeResult"),
			},
			"/composite/tree/{objectApiName}": map[string]any{
				"post": operation("createSalesforceCompositeTree", "Create Salesforce records with a composite tree request", []map[string]any{pathParam("objectApiName", "Root Salesforce object API name.")}, "#/components/schemas/CompositeRequest", "#/components/schemas/CompositeResult"),
			},
		},
	}
	write("catalog-openapi-cache/advisory-overlays/salesforce-rest-core-overlay.json", doc)
}

func operation(operationID, summary string, parameters []map[string]any, requestSchema, responseSchema string) map[string]any {
	op := map[string]any{
		"operationId": operationID,
		"summary":     summary,
		"description": "Advisory operation derived from official Salesforce REST API documentation.",
		"security":    []map[string]any{{"salesforceBearer": []string{}}},
		"responses": map[string]any{
			"200": response("Salesforce response.", responseSchema),
			"201": response("Salesforce resource created.", responseSchema),
			"204": map[string]any{"description": "Salesforce request completed without response content."},
			"default": map[string]any{
				"description": "Salesforce error response.",
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
		"required":    true,
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
