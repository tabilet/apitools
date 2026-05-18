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
			"title":       "Google Sheets API v4 Advisory Overlay",
			"version":     "2026-05-18",
			"description": "Advisory OpenAPI overlay derived from the official Google Sheets Discovery document and REST documentation. This is not an official Google OpenAPI document.",
		},
		"servers": []map[string]any{{"url": "https://sheets.googleapis.com"}},
		"x-apitools-overlay": map[string]any{
			"provider_id":       "google-sheets",
			"overlay_id":        "google-sheets-v4-advisory-overlay",
			"official_openapi":  false,
			"derived_from_docs": true,
			"source_refs": []string{
				"https://sheets.googleapis.com/$discovery/rest?version=v4",
				"https://developers.google.com/workspace/sheets/api/reference/rest",
				"https://developers.google.com/workspace/sheets/api/scopes",
			},
			"source_note": "Google Sheets publishes an official Discovery document, but no official OpenAPI document is recorded in the apitools catalog.",
		},
		"components": map[string]any{
			"securitySchemes": map[string]any{
				"googleOAuth2": googleOAuthScheme(map[string]string{
					"https://www.googleapis.com/auth/drive.file":            "Read and write only files opened or created by the app.",
					"https://www.googleapis.com/auth/spreadsheets":          "Read and write Google Sheets spreadsheets.",
					"https://www.googleapis.com/auth/spreadsheets.readonly": "Read Google Sheets spreadsheets.",
				}),
			},
			"schemas": map[string]any{
				"Spreadsheet":        objectSchema(),
				"SpreadsheetRequest": objectSchema(),
				"ValueRange":         objectSchema(),
				"BatchUpdateRequest": objectSchema(),
				"Error":              objectSchema(),
			},
		},
		"security": []map[string]any{{"googleOAuth2": []string{"https://www.googleapis.com/auth/spreadsheets.readonly"}}},
		"paths": map[string]any{
			"/v4/spreadsheets": map[string]any{
				"post": operation("createGoogleSpreadsheet", "Create a Google spreadsheet", nil, "#/components/schemas/SpreadsheetRequest", "#/components/schemas/Spreadsheet", []string{"https://www.googleapis.com/auth/spreadsheets"}),
			},
			"/v4/spreadsheets/{spreadsheetId}": map[string]any{
				"get": operation("getGoogleSpreadsheet", "Get a Google spreadsheet", []map[string]any{
					pathParam("spreadsheetId", "Spreadsheet identifier."),
					queryParam("ranges", "A1 notation ranges to retrieve."),
					queryParam("includeGridData", "Whether to include grid data."),
				}, "", "#/components/schemas/Spreadsheet", []string{"https://www.googleapis.com/auth/spreadsheets.readonly"}),
			},
			"/v4/spreadsheets/{spreadsheetId}:batchUpdate": map[string]any{
				"post": operation("batchUpdateGoogleSpreadsheet", "Batch update a Google spreadsheet", []map[string]any{pathParam("spreadsheetId", "Spreadsheet identifier.")}, "#/components/schemas/BatchUpdateRequest", "#/components/schemas/Spreadsheet", []string{"https://www.googleapis.com/auth/spreadsheets"}),
			},
			"/v4/spreadsheets/{spreadsheetId}/values/{range}": map[string]any{
				"get": operation("getGoogleSheetValues", "Get Google Sheets values", []map[string]any{
					pathParam("spreadsheetId", "Spreadsheet identifier."),
					pathParam("range", "A1 notation range."),
					queryParam("majorDimension", "Major dimension for values."),
					queryParam("valueRenderOption", "How values should be represented."),
				}, "", "#/components/schemas/ValueRange", []string{"https://www.googleapis.com/auth/spreadsheets.readonly"}),
				"put": operation("updateGoogleSheetValues", "Update Google Sheets values", []map[string]any{
					pathParam("spreadsheetId", "Spreadsheet identifier."),
					pathParam("range", "A1 notation range."),
					queryParam("valueInputOption", "How input data should be interpreted."),
				}, "#/components/schemas/ValueRange", "#/components/schemas/ValueRange", []string{"https://www.googleapis.com/auth/spreadsheets"}),
			},
			"/v4/spreadsheets/{spreadsheetId}/values/{range}:append": map[string]any{
				"post": operation("appendGoogleSheetValues", "Append Google Sheets values", []map[string]any{
					pathParam("spreadsheetId", "Spreadsheet identifier."),
					pathParam("range", "A1 notation range."),
					queryParam("valueInputOption", "How input data should be interpreted."),
					queryParam("insertDataOption", "How inserted data should be handled."),
				}, "#/components/schemas/ValueRange", "#/components/schemas/ValueRange", []string{"https://www.googleapis.com/auth/spreadsheets"}),
			},
			"/v4/spreadsheets/{spreadsheetId}/values:batchUpdate": map[string]any{
				"post": operation("batchUpdateGoogleSheetValues", "Batch update Google Sheets values", []map[string]any{pathParam("spreadsheetId", "Spreadsheet identifier.")}, "#/components/schemas/BatchUpdateRequest", "#/components/schemas/ValueRange", []string{"https://www.googleapis.com/auth/spreadsheets"}),
			},
		},
	}
	write("catalog-openapi-cache/advisory-overlays/google-sheets-v4-overlay.json", doc)
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
		"description": "Google OAuth 2.0 authorization code flow with Sheets API scopes.",
	}
}

func operation(operationID, summary string, parameters []map[string]any, requestSchema, responseSchema string, scopes []string) map[string]any {
	op := map[string]any{
		"operationId": operationID,
		"summary":     summary,
		"description": "Advisory operation derived from official Google Sheets Discovery and REST documentation.",
		"security":    []map[string]any{{"googleOAuth2": scopes}},
		"responses": map[string]any{
			"200": response("Google Sheets response.", responseSchema),
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
