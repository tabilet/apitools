//go:build ignore

package main

import (
	"encoding/json"
	"os"
)

const provider = "Mailchimp"

func main() {
	doc := map[string]any{
		"openapi": "3.0.3",
		"info": map[string]any{
			"title":       "Mailchimp Marketing API Advisory Overlay",
			"version":     "2026-05-18",
			"description": "Advisory OpenAPI overlay derived from official Mailchimp Marketing API human documentation. This is not an official Mailchimp OpenAPI document.",
		},
		"servers": []map[string]any{{
			"url": "https://{dc}.api.mailchimp.com/3.0",
			"variables": map[string]any{
				"dc": map[string]any{"default": "us1", "description": "Mailchimp data center prefix from the account API key or account metadata."},
			},
		}},
		"x-apitools-overlay": overlayMeta("mailchimp", "mailchimp-marketing-api-overlay", []string{
			"https://mailchimp.com/developer/marketing/api/",
			"https://mailchimp.com/developer/marketing/docs/fundamentals/",
			"https://mailchimp.com/developer/marketing/guides/quick-start/",
		}, "Mailchimp publishes official Marketing API human documentation and notes OpenAPI-backed endpoint pages, but no stable public downloadable OpenAPI document is recorded in the apitools catalog."),
		"components": map[string]any{
			"securitySchemes": map[string]any{
				"mailchimpBasic":  map[string]any{"type": "http", "scheme": "basic", "description": "Mailchimp Marketing API key carried with HTTP Basic authentication as anystring:api_key."},
				"mailchimpOAuth2": map[string]any{"type": "http", "scheme": "bearer", "description": "Mailchimp OAuth 2 access token carried in the Authorization header."},
			},
			"schemas": commonSchemas([]string{"MailchimpObject", "MailchimpCollection", "MailchimpError"}),
		},
		"security": []map[string]any{{"mailchimpBasic": []string{}}, {"mailchimpOAuth2": []string{}}},
		"paths": map[string]any{
			"/": map[string]any{
				"get": operation("getMailchimpAccountRoot", "Get account root information", nil, "", "#/components/schemas/MailchimpObject", "mailchimpBasic", "mailchimpOAuth2"),
			},
			"/lists": map[string]any{
				"get":  operation("listMailchimpAudiences", "List audiences", nil, "", "#/components/schemas/MailchimpCollection", "mailchimpBasic", "mailchimpOAuth2"),
				"post": operation("createMailchimpAudience", "Create an audience", nil, "#/components/schemas/MailchimpObject", "#/components/schemas/MailchimpObject", "mailchimpBasic", "mailchimpOAuth2"),
			},
			"/lists/{list_id}/members": map[string]any{
				"get":  operation("listMailchimpAudienceMembers", "List audience members", []map[string]any{pathParam("list_id", "Mailchimp audience ID.")}, "", "#/components/schemas/MailchimpCollection", "mailchimpBasic", "mailchimpOAuth2"),
				"post": operation("addMailchimpAudienceMember", "Add an audience member", []map[string]any{pathParam("list_id", "Mailchimp audience ID.")}, "#/components/schemas/MailchimpObject", "#/components/schemas/MailchimpObject", "mailchimpBasic", "mailchimpOAuth2"),
			},
			"/lists/{list_id}/members/{subscriber_hash}": map[string]any{
				"get":    operation("getMailchimpAudienceMember", "Get an audience member", memberParams(), "", "#/components/schemas/MailchimpObject", "mailchimpBasic", "mailchimpOAuth2"),
				"put":    operation("upsertMailchimpAudienceMember", "Add or update an audience member", memberParams(), "#/components/schemas/MailchimpObject", "#/components/schemas/MailchimpObject", "mailchimpBasic", "mailchimpOAuth2"),
				"patch":  operation("updateMailchimpAudienceMember", "Update an audience member", memberParams(), "#/components/schemas/MailchimpObject", "#/components/schemas/MailchimpObject", "mailchimpBasic", "mailchimpOAuth2"),
				"delete": operation("deleteMailchimpAudienceMember", "Delete an audience member", memberParams(), "", "#/components/schemas/MailchimpObject", "mailchimpBasic", "mailchimpOAuth2"),
			},
			"/campaigns": map[string]any{
				"get":  operation("listMailchimpCampaigns", "List campaigns", nil, "", "#/components/schemas/MailchimpCollection", "mailchimpBasic", "mailchimpOAuth2"),
				"post": operation("createMailchimpCampaign", "Create a campaign", nil, "#/components/schemas/MailchimpObject", "#/components/schemas/MailchimpObject", "mailchimpBasic", "mailchimpOAuth2"),
			},
			"/campaigns/{campaign_id}/actions/send": map[string]any{
				"post": operation("sendMailchimpCampaign", "Send a campaign", []map[string]any{pathParam("campaign_id", "Mailchimp campaign ID.")}, "", "#/components/schemas/MailchimpObject", "mailchimpBasic", "mailchimpOAuth2"),
			},
			"/reports/{campaign_id}": map[string]any{
				"get": operation("getMailchimpCampaignReport", "Get campaign report", []map[string]any{pathParam("campaign_id", "Mailchimp campaign ID.")}, "", "#/components/schemas/MailchimpObject", "mailchimpBasic", "mailchimpOAuth2"),
			},
		},
	}
	write("catalog-openapi-cache/advisory-overlays/mailchimp-marketing-api-overlay.json", doc)
}

func memberParams() []map[string]any {
	return []map[string]any{
		pathParam("list_id", "Mailchimp audience ID."),
		pathParam("subscriber_hash", "MD5 hash of the lowercase subscriber email address."),
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
			"default": response(provider+" error response.", "#/components/schemas/MailchimpError"),
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
