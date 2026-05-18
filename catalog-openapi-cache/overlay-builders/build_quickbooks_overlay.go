//go:build ignore

package main

import (
	"encoding/json"
	"os"
)

const provider = "QuickBooks"

func main() {
	doc := map[string]any{
		"openapi": "3.0.3",
		"info": map[string]any{
			"title":       "QuickBooks Online Accounting API Advisory Overlay",
			"version":     "2026-05-18",
			"description": "Advisory OpenAPI overlay derived from official Intuit QuickBooks Online API human documentation. This is not an official Intuit OpenAPI document.",
		},
		"servers": []map[string]any{{"url": "https://quickbooks.api.intuit.com"}},
		"x-apitools-overlay": overlayMeta("quickbooks", "quickbooks-online-accounting-api-overlay", []string{
			"https://developer.intuit.com/app/developer/qbo/docs/learn/explore-the-quickbooks-online-api",
			"https://developer.intuit.com/app/developer/qbo/docs/develop/authentication-and-authorization/oauth-2.0",
		}, "Intuit publishes official QuickBooks Online REST API human documentation and OAuth docs, but no stable public downloadable OpenAPI document is recorded in the apitools catalog."),
		"components": map[string]any{
			"securitySchemes": map[string]any{
				"quickbooksOAuth2": map[string]any{"type": "http", "scheme": "bearer", "bearerFormat": "OAuth 2.0 access token", "description": "QuickBooks Online OAuth 2.0 access token carried in the Authorization header."},
			},
			"schemas": commonSchemas([]string{"QuickBooksObject", "QuickBooksQueryResponse", "QuickBooksError"}),
		},
		"security": []map[string]any{{"quickbooksOAuth2": []string{}}},
		"paths": map[string]any{
			"/v3/company/{realmId}/companyinfo/{realmId}": map[string]any{
				"get": operation("getQuickBooksCompanyInfo", "Get company information", realmParams(), "", "#/components/schemas/QuickBooksObject", "quickbooksOAuth2"),
			},
			"/v3/company/{realmId}/query": map[string]any{
				"get": operation("queryQuickBooks", "Query QuickBooks entities", append(realmParams(), queryParam("query", "SQL-like QuickBooks query statement.")), "", "#/components/schemas/QuickBooksQueryResponse", "quickbooksOAuth2"),
			},
			"/v3/company/{realmId}/customer": map[string]any{
				"post": operation("createQuickBooksCustomer", "Create a customer", realmParams(), "#/components/schemas/QuickBooksObject", "#/components/schemas/QuickBooksObject", "quickbooksOAuth2"),
			},
			"/v3/company/{realmId}/customer/{customerId}": map[string]any{
				"get": operation("getQuickBooksCustomer", "Get a customer", append(realmParams(), pathParam("customerId", "QuickBooks customer ID.")), "", "#/components/schemas/QuickBooksObject", "quickbooksOAuth2"),
			},
			"/v3/company/{realmId}/invoice": map[string]any{
				"post": operation("createQuickBooksInvoice", "Create an invoice", realmParams(), "#/components/schemas/QuickBooksObject", "#/components/schemas/QuickBooksObject", "quickbooksOAuth2"),
			},
			"/v3/company/{realmId}/invoice/{invoiceId}": map[string]any{
				"get": operation("getQuickBooksInvoice", "Get an invoice", append(realmParams(), pathParam("invoiceId", "QuickBooks invoice ID.")), "", "#/components/schemas/QuickBooksObject", "quickbooksOAuth2"),
			},
			"/v3/company/{realmId}/payment": map[string]any{
				"post": operation("createQuickBooksPayment", "Create a payment", realmParams(), "#/components/schemas/QuickBooksObject", "#/components/schemas/QuickBooksObject", "quickbooksOAuth2"),
			},
		},
	}
	write("catalog-openapi-cache/advisory-overlays/quickbooks-online-accounting-api-overlay.json", doc)
}

func realmParams() []map[string]any {
	return []map[string]any{pathParam("realmId", "QuickBooks company realm ID.")}
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
			"default": response(provider+" error response.", "#/components/schemas/QuickBooksError"),
		},
	}
	if len(parameters) > 0 {
		op["parameters"] = append(parameters, queryParam("minorversion", "Optional QuickBooks minor version."))
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
