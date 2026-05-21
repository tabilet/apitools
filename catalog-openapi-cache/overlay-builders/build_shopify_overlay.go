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
			"title":       "Shopify Admin REST API Advisory Overlay",
			"version":     "2026-05-18",
			"description": "Advisory OpenAPI overlay derived from official Shopify REST Admin API human documentation. This is not an official Shopify OpenAPI document.",
		},
		"servers": []map[string]any{{
			"url": "https://{shop}.myshopify.com/admin/api/{apiVersion}",
			"variables": map[string]any{
				"shop":       map[string]any{"default": "example"},
				"apiVersion": map[string]any{"default": "2026-01"},
			},
		}},
		"x-apitools-overlay": map[string]any{
			"provider_id":       "shopify",
			"overlay_id":        "shopify-admin-rest-advisory-overlay",
			"official_openapi":  false,
			"derived_from_docs": true,
			"source_refs": []string{
				"https://shopify.dev/docs/api/admin-rest",
				"https://shopify.dev/docs/api/admin-rest/usage/access-scopes",
				"https://shopify.dev/docs/api/admin-rest/usage/versioning",
			},
			"source_note": "Shopify publishes REST Admin API human documentation, but no official downloadable OpenAPI document is recorded in the apitools catalog.",
		},
		"components": map[string]any{
			"securitySchemes": map[string]any{
				"shopifyAccessToken": map[string]any{
					"type":        "apiKey",
					"in":          "header",
					"name":        "X-Shopify-Access-Token",
					"description": "Shopify Admin API access token.",
				},
			},
			"schemas": map[string]any{
				"Product":  objectSchema(),
				"Order":    objectSchema(),
				"Customer": objectSchema(),
				"Error":    objectSchema(),
			},
		},
		"security": []map[string]any{{"shopifyAccessToken": []string{}}},
		"paths": map[string]any{
			"/products.json": map[string]any{
				"get":  operation("listShopifyProducts", "List Shopify products", []map[string]any{queryParam("limit", "Maximum records to return."), queryParam("page_info", "Pagination cursor.")}, "", "#/components/schemas/Product"),
				"post": operation("createShopifyProduct", "Create a Shopify product", nil, "#/components/schemas/Product", "#/components/schemas/Product"),
			},
			"/products/{productId}.json": map[string]any{
				"get":    operation("getShopifyProduct", "Get a Shopify product", []map[string]any{pathParam("productId", "Product identifier.")}, "", "#/components/schemas/Product"),
				"put":    operation("updateShopifyProduct", "Update a Shopify product", []map[string]any{pathParam("productId", "Product identifier.")}, "#/components/schemas/Product", "#/components/schemas/Product"),
				"delete": operation("deleteShopifyProduct", "Delete a Shopify product", []map[string]any{pathParam("productId", "Product identifier.")}, "", "#/components/schemas/Product"),
			},
			"/orders.json": map[string]any{
				"get": operation("listShopifyOrders", "List Shopify orders", []map[string]any{queryParam("status", "Order status filter."), queryParam("limit", "Maximum records to return."), queryParam("page_info", "Pagination cursor.")}, "", "#/components/schemas/Order"),
			},
			"/orders/{orderId}.json": map[string]any{
				"get": operation("getShopifyOrder", "Get a Shopify order", []map[string]any{pathParam("orderId", "Order identifier.")}, "", "#/components/schemas/Order"),
			},
			"/customers.json": map[string]any{
				"get":  operation("listShopifyCustomers", "List Shopify customers", []map[string]any{queryParam("limit", "Maximum records to return."), queryParam("page_info", "Pagination cursor.")}, "", "#/components/schemas/Customer"),
				"post": operation("createShopifyCustomer", "Create a Shopify customer", nil, "#/components/schemas/Customer", "#/components/schemas/Customer"),
			},
			"/customers/{customerId}.json": map[string]any{
				"get":    operation("getShopifyCustomer", "Get a Shopify customer", []map[string]any{pathParam("customerId", "Customer identifier.")}, "", "#/components/schemas/Customer"),
				"put":    operation("updateShopifyCustomer", "Update a Shopify customer", []map[string]any{pathParam("customerId", "Customer identifier.")}, "#/components/schemas/Customer", "#/components/schemas/Customer"),
				"delete": operation("deleteShopifyCustomer", "Delete a Shopify customer", []map[string]any{pathParam("customerId", "Customer identifier.")}, "", "#/components/schemas/Customer"),
			},
		},
	}
	write("catalog-openapi-cache/advisory-overlays/shopify-admin-rest-overlay.json", doc)
}

func operation(operationID, summary string, parameters []map[string]any, requestSchema, responseSchema string) map[string]any {
	op := map[string]any{
		"operationId": operationID,
		"summary":     summary,
		"description": "Advisory operation derived from official Shopify REST Admin API documentation.",
		"security":    []map[string]any{{"shopifyAccessToken": []string{}}},
		"responses": map[string]any{
			"200": response("Shopify response.", responseSchema),
			"201": response("Shopify resource created.", responseSchema),
			"204": map[string]any{"description": "Shopify request completed without response content."},
			"default": map[string]any{
				"description": "Shopify error response.",
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
