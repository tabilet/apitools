//go:build ignore

package main

import (
	"encoding/json"
	"os"
)

const provider = "ServiceNow"

func main() {
	doc := map[string]any{
		"openapi": "3.0.3",
		"info": map[string]any{
			"title":       "ServiceNow REST API Advisory Overlay",
			"version":     "2026-05-18",
			"description": "Advisory OpenAPI overlay derived from official ServiceNow REST API Explorer and Table API human documentation. This is not an official ServiceNow OpenAPI document.",
		},
		"servers": []map[string]any{{
			"url": "https://{instance}.service-now.com",
			"variables": map[string]any{
				"instance": map[string]any{"default": "example", "description": "ServiceNow instance host prefix."},
			},
		}},
		"x-apitools-overlay": overlayMeta("servicenow", "servicenow-rest-api-overlay", []string{
			"https://www.servicenow.com/docs/r/api-reference/rest-api-explorer/c_RESTAPI.html",
			"https://www.servicenow.com/docs/r/api-reference/rest-api-explorer/export-openapi-specification.html",
			"https://www.servicenow.com/docs/r/api-reference/rest-apis/c_ImportSetAPI.html",
		}, "ServiceNow documents REST API Explorer and per-instance OpenAPI export, but no stable public downloadable OpenAPI document is recorded in the apitools catalog."),
		"components": map[string]any{
			"securitySchemes": map[string]any{
				"servicenowBasic":  map[string]any{"type": "http", "scheme": "basic", "description": "ServiceNow REST API Basic authentication authorized by instance ACLs."},
				"servicenowOAuth2": map[string]any{"type": "http", "scheme": "bearer", "bearerFormat": "OAuth 2.0 access token", "description": "ServiceNow OAuth 2.0 access token carried in the Authorization header."},
			},
			"schemas": commonSchemas([]string{"ServiceNowObject", "ServiceNowCollection", "ServiceNowError"}),
		},
		"security": []map[string]any{{"servicenowBasic": []string{}}, {"servicenowOAuth2": []string{}}},
		"paths": map[string]any{
			"/api/now/table/{tableName}": map[string]any{
				"get":  operation("listServiceNowTableRecords", "List table records", append(tableParams(), listParams()...), "", "#/components/schemas/ServiceNowCollection", "servicenowBasic", "servicenowOAuth2"),
				"post": operation("createServiceNowTableRecord", "Create a table record", tableParams(), "#/components/schemas/ServiceNowObject", "#/components/schemas/ServiceNowObject", "servicenowBasic", "servicenowOAuth2"),
			},
			"/api/now/table/{tableName}/{sys_id}": map[string]any{
				"get":    operation("getServiceNowTableRecord", "Get a table record", recordParams(), "", "#/components/schemas/ServiceNowObject", "servicenowBasic", "servicenowOAuth2"),
				"patch":  operation("updateServiceNowTableRecord", "Update a table record", recordParams(), "#/components/schemas/ServiceNowObject", "#/components/schemas/ServiceNowObject", "servicenowBasic", "servicenowOAuth2"),
				"delete": operation("deleteServiceNowTableRecord", "Delete a table record", recordParams(), "", "#/components/schemas/ServiceNowObject", "servicenowBasic", "servicenowOAuth2"),
			},
			"/api/now/attachment/file": map[string]any{
				"post": operation("uploadServiceNowAttachment", "Upload an attachment", attachmentParams(), "", "#/components/schemas/ServiceNowObject", "servicenowBasic", "servicenowOAuth2"),
			},
			"/api/now/import/{stagingTableName}": map[string]any{
				"post": operation("createServiceNowImportSetRecord", "Insert an import set record and trigger transformation", importParams(), "#/components/schemas/ServiceNowObject", "#/components/schemas/ServiceNowObject", "servicenowBasic", "servicenowOAuth2"),
			},
			"/api/now/import/{stagingTableName}/{sys_id}": map[string]any{
				"get": operation("getServiceNowImportSetResult", "Get an import set staging record and transformation result", importRecordParams(), "", "#/components/schemas/ServiceNowObject", "servicenowBasic", "servicenowOAuth2"),
			},
			"/api/now/import/{stagingTableName}/insertMultiple": map[string]any{
				"post": operation("createServiceNowImportSetRecords", "Insert multiple import set records", importParams(), "#/components/schemas/ServiceNowObject", "#/components/schemas/ServiceNowObject", "servicenowBasic", "servicenowOAuth2"),
			},
		},
	}
	write("catalog-openapi-cache/advisory-overlays/servicenow-rest-api-overlay.json", doc)
}

func tableParams() []map[string]any {
	return []map[string]any{pathParam("tableName", "ServiceNow table name, such as incident.")}
}

func recordParams() []map[string]any {
	return []map[string]any{pathParam("tableName", "ServiceNow table name."), pathParam("sys_id", "ServiceNow record sys_id.")}
}

func listParams() []map[string]any {
	return []map[string]any{
		queryParam("sysparm_query", "Encoded query string."),
		queryParam("sysparm_fields", "Comma-separated field list."),
		queryParam("sysparm_limit", "Maximum number of records to return."),
		queryParam("sysparm_offset", "Pagination offset."),
	}
}

func attachmentParams() []map[string]any {
	return []map[string]any{
		queryParam("table_name", "Target table name."),
		queryParam("table_sys_id", "Target record sys_id."),
		queryParam("file_name", "Attachment file name."),
	}
}

func importParams() []map[string]any {
	return []map[string]any{
		pathParam("stagingTableName", "Import set staging table name."),
	}
}

func importRecordParams() []map[string]any {
	return []map[string]any{
		pathParam("stagingTableName", "Import set staging table name."),
		pathParam("sys_id", "Import staging record sys_id."),
	}
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
			"default": response(provider+" error response.", "#/components/schemas/ServiceNowError"),
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
