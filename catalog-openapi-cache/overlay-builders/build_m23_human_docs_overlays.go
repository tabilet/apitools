//go:build ignore

package main

import (
	"encoding/json"
	"os"
	"sort"
)

type overlaySpec struct {
	ProviderID  string
	Title       string
	Description string
	ServerURL   string
	ServerVars  map[string]map[string]any
	Sources     []string
	SourceNote  string
	Security    map[string]map[string]any
	Schemas     []string
	Paths       map[string]map[string]any
	OutputPath  string
}

func main() {
	for _, spec := range []overlaySpec{
		bannerbearOverlay(),
		gristOverlay(),
	} {
		write(spec.OutputPath, build(spec))
	}
}

func bannerbearOverlay() overlaySpec {
	security := map[string]map[string]any{
		"bannerbearBearer": {"type": "http", "scheme": "bearer", "description": "Bannerbear Project API Key or Master API Key carried as an Authorization bearer token."},
	}
	return overlaySpec{
		ProviderID:  "bannerbear",
		Title:       "Bannerbear API v2 Advisory Overlay",
		Description: "Advisory OpenAPI overlay derived from official Bannerbear API v2 human documentation. This is not an official Bannerbear OpenAPI document.",
		ServerURL:   "https://api.bannerbear.com",
		Sources:     []string{"https://developers.bannerbear.com/v2/", "https://www.bannerbear.com/help/api/"},
		SourceNote:  "Bannerbear publishes REST-shaped API v2 docs with Authorization bearer API-key metadata; this overlay covers authentication, images, collections, videos, screenshots, templates, and projects.",
		Security:    security,
		Schemas:     []string{"BannerbearObject", "BannerbearCollection", "BannerbearError"},
		OutputPath:  "catalog-openapi-cache/advisory-overlays/bannerbear-api-v2-overlay.json",
		Paths: map[string]map[string]any{
			"/v2/auth": {"get": op("getBannerbearAuth", "Check authentication", nil, "", "#/components/schemas/BannerbearObject", "bannerbearBearer")},
			"/v2/images": {
				"get":  op("listBannerbearImages", "List images", params(query("page", "Page number.")), "", "#/components/schemas/BannerbearCollection", "bannerbearBearer"),
				"post": op("createBannerbearImage", "Create an image", nil, "#/components/schemas/BannerbearObject", "#/components/schemas/BannerbearObject", "bannerbearBearer"),
			},
			"/v2/images/{image_uid}": {"get": op("getBannerbearImage", "Get an image", params(path("image_uid", "Bannerbear image UID.")), "", "#/components/schemas/BannerbearObject", "bannerbearBearer")},
			"/v2/collections": {
				"get":  op("listBannerbearCollections", "List collections", params(query("page", "Page number.")), "", "#/components/schemas/BannerbearCollection", "bannerbearBearer"),
				"post": op("createBannerbearCollection", "Create a collection", nil, "#/components/schemas/BannerbearObject", "#/components/schemas/BannerbearObject", "bannerbearBearer"),
			},
			"/v2/collections/{collection_uid}": {"get": op("getBannerbearCollection", "Get a collection", params(path("collection_uid", "Bannerbear collection UID.")), "", "#/components/schemas/BannerbearObject", "bannerbearBearer")},
			"/v2/videos": {
				"get":  op("listBannerbearVideos", "List videos", params(query("page", "Page number.")), "", "#/components/schemas/BannerbearCollection", "bannerbearBearer"),
				"post": op("createBannerbearVideo", "Create a video", nil, "#/components/schemas/BannerbearObject", "#/components/schemas/BannerbearObject", "bannerbearBearer"),
			},
			"/v2/videos/{video_uid}": {"get": op("getBannerbearVideo", "Get a video", params(path("video_uid", "Bannerbear video UID.")), "", "#/components/schemas/BannerbearObject", "bannerbearBearer")},
			"/v2/screenshots":        {"post": op("createBannerbearScreenshot", "Create a screenshot", nil, "#/components/schemas/BannerbearObject", "#/components/schemas/BannerbearObject", "bannerbearBearer")},
			"/v2/screenshots/{screenshot_uid}": {
				"get": op("getBannerbearScreenshot", "Get a screenshot", params(path("screenshot_uid", "Bannerbear screenshot UID.")), "", "#/components/schemas/BannerbearObject", "bannerbearBearer"),
			},
			"/v2/templates": {
				"get":  op("listBannerbearTemplates", "List templates", params(query("page", "Page number.")), "", "#/components/schemas/BannerbearCollection", "bannerbearBearer"),
				"post": op("createBannerbearTemplate", "Create a template", nil, "#/components/schemas/BannerbearObject", "#/components/schemas/BannerbearObject", "bannerbearBearer"),
			},
			"/v2/templates/{template_uid}": {
				"get":    op("getBannerbearTemplate", "Get a template", params(path("template_uid", "Bannerbear template UID.")), "", "#/components/schemas/BannerbearObject", "bannerbearBearer"),
				"patch":  op("updateBannerbearTemplate", "Update a template", params(path("template_uid", "Bannerbear template UID.")), "#/components/schemas/BannerbearObject", "#/components/schemas/BannerbearObject", "bannerbearBearer"),
				"delete": op("deleteBannerbearTemplate", "Delete a template", params(path("template_uid", "Bannerbear template UID.")), "", "#/components/schemas/BannerbearObject", "bannerbearBearer"),
			},
			"/v2/projects": {
				"get":  op("listBannerbearProjects", "List projects", nil, "", "#/components/schemas/BannerbearCollection", "bannerbearBearer"),
				"post": op("createBannerbearProject", "Create a project", nil, "#/components/schemas/BannerbearObject", "#/components/schemas/BannerbearObject", "bannerbearBearer"),
			},
			"/v2/projects/{project_uid}": {"get": op("getBannerbearProject", "Get a project", params(path("project_uid", "Bannerbear project UID.")), "", "#/components/schemas/BannerbearObject", "bannerbearBearer")},
		},
	}
}

func gristOverlay() overlaySpec {
	security := map[string]map[string]any{
		"gristBearer": {"type": "http", "scheme": "bearer", "description": "Grist API key carried as an Authorization bearer token."},
	}
	return overlaySpec{
		ProviderID:  "grist",
		Title:       "Grist REST API Advisory Overlay",
		Description: "Advisory OpenAPI overlay derived from official Grist REST API human documentation. This is not an official Grist OpenAPI document.",
		ServerURL:   "https://{site_domain}/api",
		ServerVars:  map[string]map[string]any{"site_domain": {"default": "docs.getgrist.com", "description": "Operator-supplied Grist personal, team, or self-hosted domain."}},
		Sources:     []string{"https://support.getgrist.com/rest-api/", "https://support.getgrist.com/api/"},
		SourceNote:  "Grist publishes REST API usage and reference docs with bearer API-key metadata; this overlay covers organizations, workspaces, documents, tables, records, SQL queries, and webhooks.",
		Security:    security,
		Schemas:     []string{"GristObject", "GristCollection", "GristError"},
		OutputPath:  "catalog-openapi-cache/advisory-overlays/grist-rest-api-overlay.json",
		Paths: map[string]map[string]any{
			"/orgs":                                    {"get": op("listGristOrgs", "List organizations", nil, "", "#/components/schemas/GristCollection", "gristBearer")},
			"/orgs/{org_id}":                           {"get": op("getGristOrg", "Get an organization", params(path("org_id", "Grist organization ID.")), "", "#/components/schemas/GristObject", "gristBearer")},
			"/orgs/{org_id}/workspaces":                {"get": op("listGristOrgWorkspaces", "List organization workspaces", params(path("org_id", "Grist organization ID.")), "", "#/components/schemas/GristCollection", "gristBearer")},
			"/workspaces/{workspace_id}":               {"get": op("getGristWorkspace", "Get a workspace", params(path("workspace_id", "Grist workspace ID.")), "", "#/components/schemas/GristObject", "gristBearer"), "patch": op("updateGristWorkspace", "Update a workspace", params(path("workspace_id", "Grist workspace ID.")), "#/components/schemas/GristObject", "#/components/schemas/GristObject", "gristBearer"), "delete": op("deleteGristWorkspace", "Delete a workspace", params(path("workspace_id", "Grist workspace ID.")), "", "#/components/schemas/GristObject", "gristBearer")},
			"/workspaces/{workspace_id}/docs":          {"get": op("listGristWorkspaceDocs", "List workspace documents", params(path("workspace_id", "Grist workspace ID.")), "", "#/components/schemas/GristCollection", "gristBearer"), "post": op("createGristWorkspaceDoc", "Create a workspace document", params(path("workspace_id", "Grist workspace ID.")), "#/components/schemas/GristObject", "#/components/schemas/GristObject", "gristBearer")},
			"/docs/{doc_id}":                           {"get": op("getGristDoc", "Get a document", params(path("doc_id", "Grist document ID.")), "", "#/components/schemas/GristObject", "gristBearer"), "patch": op("updateGristDoc", "Update a document", params(path("doc_id", "Grist document ID.")), "#/components/schemas/GristObject", "#/components/schemas/GristObject", "gristBearer"), "delete": op("deleteGristDoc", "Delete a document", params(path("doc_id", "Grist document ID.")), "", "#/components/schemas/GristObject", "gristBearer")},
			"/docs/{doc_id}/tables":                    {"get": op("listGristDocTables", "List document tables", params(path("doc_id", "Grist document ID.")), "", "#/components/schemas/GristCollection", "gristBearer")},
			"/docs/{doc_id}/tables/{table_id}/records": {"get": op("listGristTableRecords", "List table records", params(path("doc_id", "Grist document ID."), path("table_id", "Grist table ID."), query("sort", "Sort expression."), query("filter", "Filter expression.")), "", "#/components/schemas/GristCollection", "gristBearer"), "post": op("addGristTableRecords", "Add table records", params(path("doc_id", "Grist document ID."), path("table_id", "Grist table ID.")), "#/components/schemas/GristObject", "#/components/schemas/GristObject", "gristBearer"), "patch": op("updateGristTableRecords", "Update table records", params(path("doc_id", "Grist document ID."), path("table_id", "Grist table ID.")), "#/components/schemas/GristObject", "#/components/schemas/GristObject", "gristBearer"), "put": op("replaceGristTableRecords", "Replace table records", params(path("doc_id", "Grist document ID."), path("table_id", "Grist table ID.")), "#/components/schemas/GristObject", "#/components/schemas/GristObject", "gristBearer")},
			"/docs/{doc_id}/sql":                       {"get": op("runGristSQLQuery", "Run a SQL query", params(path("doc_id", "Grist document ID."), query("q", "SQL query text.")), "", "#/components/schemas/GristObject", "gristBearer")},
			"/docs/{doc_id}/webhooks":                  {"get": op("listGristWebhooks", "List webhooks", params(path("doc_id", "Grist document ID.")), "", "#/components/schemas/GristCollection", "gristBearer"), "post": op("createGristWebhook", "Create a webhook", params(path("doc_id", "Grist document ID.")), "#/components/schemas/GristObject", "#/components/schemas/GristObject", "gristBearer")},
		},
	}
}

func build(spec overlaySpec) map[string]any {
	doc := map[string]any{
		"openapi": "3.0.3",
		"info": map[string]any{
			"title":       spec.Title,
			"version":     "2026-05-19",
			"description": spec.Description,
		},
		"servers": []map[string]any{{"url": spec.ServerURL}},
		"paths":   orderedMap(spec.Paths),
		"components": map[string]any{
			"securitySchemes": orderedMap(spec.Security),
			"schemas":         schemas(spec.Schemas),
		},
		"x-apitools-overlay": map[string]any{
			"provider_id":       spec.ProviderID,
			"official_openapi":  false,
			"derived_from_docs": true,
			"source_refs":       spec.Sources,
			"source_note":       spec.SourceNote,
		},
	}
	if len(spec.ServerVars) > 0 {
		doc["servers"] = []map[string]any{{"url": spec.ServerURL, "variables": orderedMap(spec.ServerVars)}}
	}
	return doc
}

func op(operationID, summary string, parameters []map[string]any, requestRef, responseRef, securityName string) map[string]any {
	out := map[string]any{
		"operationId": operationID,
		"summary":     summary,
		"description": "Advisory operation derived from official human API documentation.",
		"responses": map[string]any{
			"200": response(responseRef),
			"default": map[string]any{
				"description": "Provider error response.",
			},
		},
		"security": []map[string][]string{{securityName: []string{}}},
	}
	if len(parameters) > 0 {
		out["parameters"] = parameters
	}
	if requestRef != "" {
		out["requestBody"] = map[string]any{
			"required": true,
			"content":  map[string]any{"application/json": map[string]any{"schema": map[string]any{"$ref": requestRef}}},
		}
	}
	return out
}

func response(ref string) map[string]any {
	out := map[string]any{"description": "Successful response."}
	if ref != "" {
		out["content"] = map[string]any{"application/json": map[string]any{"schema": map[string]any{"$ref": ref}}}
	}
	return out
}

func params(items ...map[string]any) []map[string]any {
	return items
}

func path(name, description string) map[string]any {
	return parameter(name, "path", description, true)
}

func query(name, description string) map[string]any {
	return parameter(name, "query", description, false)
}

func parameter(name, in, description string, required bool) map[string]any {
	return map[string]any{
		"name":        name,
		"in":          in,
		"required":    required,
		"description": description,
		"schema":      map[string]any{"type": "string"},
	}
}

func schemas(names []string) map[string]map[string]any {
	sort.Strings(names)
	out := map[string]map[string]any{}
	for _, name := range names {
		out[name] = map[string]any{"type": "object", "additionalProperties": true}
	}
	return out
}

func orderedMap[V any](in map[string]V) map[string]V {
	keys := make([]string, 0, len(in))
	for key := range in {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make(map[string]V, len(in))
	for _, key := range keys {
		out[key] = in[key]
	}
	return out
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
