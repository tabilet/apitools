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
			"title":       "Dropbox Core API Advisory Overlay",
			"version":     "2026-05-20",
			"description": "Stone-derived advisory OpenAPI overlay for a reviewed Dropbox API v2 subset. This is not an official Dropbox OpenAPI document.",
		},
		"servers": []map[string]any{{"url": "https://api.dropboxapi.com"}},
		"x-apitools-overlay": map[string]any{
			"provider_id":                        "dropbox",
			"overlay_id":                         "dropbox-core-api-advisory-overlay",
			"official_openapi":                   false,
			"derived_from_docs":                  true,
			"derived_from_official_machine_spec": true,
			"source_protocol":                    "dropbox-stone",
			"source_refs": []string{
				"https://github.com/dropbox/dropbox-api-spec",
				"https://www.dropbox.com/developers/documentation/http/documentation",
				"https://developers.dropbox.com/oauth-guide",
			},
			"source_note": "Dropbox publishes an official API v2 Stone specification for SDK generation plus human HTTP and OAuth documentation, but no official OpenAPI document is recorded in the apitools catalog. This overlay is a reviewed advisory subset, not a general Stone conversion.",
		},
		"components": map[string]any{
			"securitySchemes": map[string]any{
				"dropboxOAuth2": map[string]any{
					"type": "oauth2",
					"flows": map[string]any{
						"authorizationCode": map[string]any{
							"authorizationUrl": "https://www.dropbox.com/oauth2/authorize",
							"tokenUrl":         "https://api.dropboxapi.com/oauth2/token",
							"scopes": map[string]any{
								"account_info.read":   "Read basic account information.",
								"files.content.read":  "Read file contents.",
								"files.content.write": "Write file contents.",
								"files.metadata.read": "Read file and folder metadata.",
								"sharing.read":        "Read shared-link metadata.",
								"sharing.write":       "Create or modify sharing links.",
							},
						},
					},
					"description": "Dropbox OAuth 2.0 bearer access token with scoped permissions.",
				},
			},
			"schemas": map[string]any{
				"DropboxRequest":  objectSchema(),
				"DropboxResponse": objectSchema(),
				"DropboxError":    objectSchema(),
				"ListFolderRequest": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"path":                                map[string]any{"type": "string"},
						"recursive":                           map[string]any{"type": "boolean"},
						"include_media_info":                  map[string]any{"type": "boolean"},
						"include_deleted":                     map[string]any{"type": "boolean"},
						"include_has_explicit_shared_members": map[string]any{"type": "boolean"},
						"include_mounted_folders":             map[string]any{"type": "boolean"},
						"limit":                               map[string]any{"type": "integer", "minimum": 1},
					},
					"additionalProperties": true,
				},
				"ContinueRequest": map[string]any{
					"type":                 "object",
					"required":             []string{"cursor"},
					"properties":           map[string]any{"cursor": map[string]any{"type": "string"}},
					"additionalProperties": true,
				},
				"PathRequest": map[string]any{
					"type":                 "object",
					"required":             []string{"path"},
					"properties":           map[string]any{"path": map[string]any{"type": "string"}},
					"additionalProperties": true,
				},
				"SharedLinkRequest": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"path":     map[string]any{"type": "string"},
						"settings": objectSchema(),
					},
					"additionalProperties": true,
				},
				"ListSharedLinksRequest": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"path":   map[string]any{"type": "string"},
						"cursor": map[string]any{"type": "string"},
						"direct_only": map[string]any{
							"type":        "boolean",
							"description": "When true, return only links directly attached to the supplied path.",
						},
					},
					"additionalProperties": true,
				},
			},
		},
		"security": []map[string]any{{"dropboxOAuth2": []string{"files.metadata.read"}}},
		"paths": map[string]any{
			"/2/users/get_current_account": map[string]any{
				"post": operation("getDropboxCurrentAccount", "Get current Dropbox account", nil, "", "#/components/schemas/DropboxResponse", []string{"account_info.read"}),
			},
			"/2/files/list_folder": map[string]any{
				"post": operation("listDropboxFolder", "List a Dropbox folder", nil, "#/components/schemas/ListFolderRequest", "#/components/schemas/DropboxResponse", []string{"files.metadata.read"}),
			},
			"/2/files/list_folder/continue": map[string]any{
				"post": operation("continueDropboxFolderListing", "Continue a Dropbox folder listing", nil, "#/components/schemas/ContinueRequest", "#/components/schemas/DropboxResponse", []string{"files.metadata.read"}),
			},
			"/2/files/get_metadata": map[string]any{
				"post": operation("getDropboxMetadata", "Get Dropbox file or folder metadata", nil, "#/components/schemas/PathRequest", "#/components/schemas/DropboxResponse", []string{"files.metadata.read"}),
			},
			"/2/files/upload": map[string]any{
				"post": contentOperation("uploadDropboxFile", "Upload a Dropbox file", "files.content.write"),
			},
			"/2/files/download": map[string]any{
				"post": downloadOperation("downloadDropboxFile", "Download a Dropbox file", "files.content.read"),
			},
			"/2/sharing/create_shared_link_with_settings": map[string]any{
				"post": operation("createDropboxSharedLink", "Create a Dropbox shared link", nil, "#/components/schemas/SharedLinkRequest", "#/components/schemas/DropboxResponse", []string{"sharing.write"}),
			},
			"/2/sharing/list_shared_links": map[string]any{
				"post": operation("listDropboxSharedLinks", "List Dropbox shared links", nil, "#/components/schemas/ListSharedLinksRequest", "#/components/schemas/DropboxResponse", []string{"sharing.read"}),
			},
		},
	}
	write("catalog-openapi-cache/advisory-overlays/dropbox-core-api-overlay.json", doc)
}

func operation(operationID, summary string, parameters []map[string]any, requestSchema, responseSchema string, scopes []string) map[string]any {
	op := map[string]any{
		"operationId": operationID,
		"summary":     summary,
		"description": "Advisory operation derived from official Dropbox documentation and Stone spec names.",
		"security":    []map[string]any{{"dropboxOAuth2": scopes}},
		"responses": map[string]any{
			"200": response("Dropbox response.", responseSchema),
			"default": map[string]any{
				"description": "Dropbox error response.",
				"content":     map[string]any{"application/json": map[string]any{"schema": map[string]any{"$ref": "#/components/schemas/DropboxError"}}},
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

func contentOperation(operationID, summary, scope string) map[string]any {
	op := operation(operationID, summary, []map[string]any{dropboxAPIArgParam()}, "", "#/components/schemas/DropboxResponse", []string{scope})
	op["servers"] = []map[string]any{{"url": "https://content.dropboxapi.com"}}
	op["requestBody"] = map[string]any{
		"required": true,
		"content": map[string]any{
			"application/octet-stream": map[string]any{
				"schema": map[string]any{"type": "string", "format": "binary"},
			},
		},
	}
	return op
}

func downloadOperation(operationID, summary, scope string) map[string]any {
	op := operation(operationID, summary, []map[string]any{dropboxAPIArgParam()}, "", "#/components/schemas/DropboxResponse", []string{scope})
	op["servers"] = []map[string]any{{"url": "https://content.dropboxapi.com"}}
	op["responses"] = map[string]any{
		"200": map[string]any{
			"description": "Dropbox file content response. Metadata is returned in Dropbox-API-Result.",
			"headers": map[string]any{
				"Dropbox-API-Result": map[string]any{
					"description": "JSON-encoded Dropbox file metadata.",
					"schema":      map[string]any{"type": "string"},
				},
			},
			"content": map[string]any{
				"application/octet-stream": map[string]any{
					"schema": map[string]any{"type": "string", "format": "binary"},
				},
			},
		},
		"default": map[string]any{
			"description": "Dropbox error response.",
			"content":     map[string]any{"application/json": map[string]any{"schema": map[string]any{"$ref": "#/components/schemas/DropboxError"}}},
		},
	}
	return op
}

func dropboxAPIArgParam() map[string]any {
	return map[string]any{
		"name":        "Dropbox-API-Arg",
		"in":          "header",
		"required":    true,
		"schema":      map[string]any{"type": "string"},
		"description": "JSON-encoded Dropbox endpoint argument header.",
	}
}

func response(description, schema string) map[string]any {
	return map[string]any{
		"description": description,
		"content":     map[string]any{"application/json": map[string]any{"schema": map[string]any{"$ref": schema}}},
	}
}

func objectSchema() map[string]any {
	return map[string]any{
		"type":                 "object",
		"additionalProperties": true,
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
