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
	Sources     []string
	SourceNote  string
	Security    map[string]map[string]any
	Schemas     []string
	Paths       map[string]map[string]any
	OutputPath  string
}

func main() {
	for _, spec := range []overlaySpec{
		facebookOverlay(),
		facebookLeadAdsOverlay(),
		linkedInOverlay(),
		messageBirdOverlay(),
		twakeOverlay(),
		twistOverlay(),
		whatsAppOverlay(),
	} {
		write(spec.OutputPath, build(spec))
	}
}

func facebookOverlay() overlaySpec {
	security := map[string]map[string]any{
		"facebookBearer": {"type": "http", "scheme": "bearer", "bearerFormat": "Meta Graph API access token", "description": "Meta Graph API access token carried in the Authorization bearer header."},
	}
	return overlaySpec{
		ProviderID:  "facebook",
		Title:       "Facebook Graph API Advisory Overlay",
		Description: "Advisory OpenAPI overlay derived from official Meta Graph API human documentation. This is not an official Meta OpenAPI document.",
		ServerURL:   "https://graph.facebook.com/{graph_api_version}",
		Sources:     []string{"https://developers.facebook.com/docs/graph-api/", "https://developers.facebook.com/docs/pages-api/", "https://developers.facebook.com/docs/graph-api/webhooks/"},
		SourceNote:  "Facebook publishes human Graph API and Pages API documentation but no recorded stable public official OpenAPI document; this overlay covers selected profile, page, feed, comment, and webhook-management entry points.",
		Security:    security,
		Schemas:     []string{"FacebookObject", "FacebookCollection", "FacebookError"},
		OutputPath:  "catalog-openapi-cache/advisory-overlays/facebook-graph-api-overlay.json",
		Paths: map[string]map[string]any{
			"/me":                         {"get": op("getFacebookMe", "Get current Graph API user or page object", params(query("fields", "Comma-separated Graph API fields.")), "", "#/components/schemas/FacebookObject", "facebookBearer")},
			"/{page_id}":                  {"get": op("getFacebookPage", "Get page", params(path("page_id", "Facebook page ID."), query("fields", "Comma-separated Graph API fields.")), "", "#/components/schemas/FacebookObject", "facebookBearer")},
			"/{page_id}/feed":             {"get": op("listFacebookPageFeed", "List page feed posts", params(path("page_id", "Facebook page ID."), query("fields", "Comma-separated Graph API fields."), query("limit", "Page size.")), "", "#/components/schemas/FacebookCollection", "facebookBearer"), "post": op("createFacebookPageFeedPost", "Create page feed post", params(path("page_id", "Facebook page ID.")), "#/components/schemas/FacebookObject", "#/components/schemas/FacebookObject", "facebookBearer")},
			"/{object_id}/comments":       {"get": op("listFacebookObjectComments", "List comments for an object", params(path("object_id", "Graph API object ID."), query("fields", "Comma-separated Graph API fields."), query("limit", "Page size.")), "", "#/components/schemas/FacebookCollection", "facebookBearer"), "post": op("createFacebookObjectComment", "Create comment on an object", params(path("object_id", "Graph API object ID.")), "#/components/schemas/FacebookObject", "#/components/schemas/FacebookObject", "facebookBearer")},
			"/{object_id}/likes":          {"get": op("listFacebookObjectLikes", "List likes for an object", params(path("object_id", "Graph API object ID."), query("limit", "Page size.")), "", "#/components/schemas/FacebookCollection", "facebookBearer")},
			"/{page_id}/subscribed_apps":  {"get": op("listFacebookPageSubscribedApps", "List page webhook subscriptions", params(path("page_id", "Facebook page ID.")), "", "#/components/schemas/FacebookCollection", "facebookBearer"), "post": op("subscribeFacebookPageApp", "Subscribe app to page webhooks", params(path("page_id", "Facebook page ID.")), "#/components/schemas/FacebookObject", "#/components/schemas/FacebookObject", "facebookBearer")},
			"/{page_id}/photos":           {"get": op("listFacebookPagePhotos", "List page photos", params(path("page_id", "Facebook page ID."), query("fields", "Comma-separated Graph API fields."), query("limit", "Page size.")), "", "#/components/schemas/FacebookCollection", "facebookBearer")},
			"/{page_id}/conversations":    {"get": op("listFacebookPageConversations", "List page conversations", params(path("page_id", "Facebook page ID."), query("fields", "Comma-separated Graph API fields."), query("limit", "Page size.")), "", "#/components/schemas/FacebookCollection", "facebookBearer")},
			"/{conversation_id}/messages": {"get": op("listFacebookConversationMessages", "List conversation messages", params(path("conversation_id", "Conversation ID."), query("fields", "Comma-separated Graph API fields."), query("limit", "Page size.")), "", "#/components/schemas/FacebookCollection", "facebookBearer")},
			"/{app_id}/subscriptions":     {"get": op("listFacebookAppSubscriptions", "List app webhook subscriptions", params(path("app_id", "Meta app ID.")), "", "#/components/schemas/FacebookCollection", "facebookBearer")},
		},
	}
}

func facebookLeadAdsOverlay() overlaySpec {
	security := map[string]map[string]any{
		"facebookLeadAdsBearer": {"type": "http", "scheme": "bearer", "bearerFormat": "Meta Graph API access token", "description": "Meta access token with Lead Ads permissions carried in the Authorization bearer header."},
	}
	return overlaySpec{
		ProviderID:  "facebook-lead-ads",
		Title:       "Facebook Lead Ads API Advisory Overlay",
		Description: "Advisory OpenAPI overlay derived from official Meta Lead Ads human documentation. This is not an official Meta OpenAPI document.",
		ServerURL:   "https://graph.facebook.com/{graph_api_version}",
		Sources:     []string{"https://developers.facebook.com/docs/marketing-api/guides/lead-ads/retrieving/", "https://developers.facebook.com/docs/marketing-api/guides/lead-ads/webhooks/", "https://developers.facebook.com/docs/marketing-apis/"},
		SourceNote:  "Facebook Lead Ads uses Meta Graph API human documentation but no recorded stable public official OpenAPI document; this overlay covers selected page form, lead retrieval, and leadgen webhook endpoints.",
		Security:    security,
		Schemas:     []string{"FacebookLeadAdsObject", "FacebookLeadAdsCollection", "FacebookLeadAdsError"},
		OutputPath:  "catalog-openapi-cache/advisory-overlays/facebook-lead-ads-overlay.json",
		Paths: map[string]map[string]any{
			"/{page_id}/leadgen_forms":       {"get": op("listFacebookLeadgenForms", "List lead generation forms for a page", params(path("page_id", "Facebook page ID."), query("fields", "Comma-separated Graph API fields."), query("limit", "Page size.")), "", "#/components/schemas/FacebookLeadAdsCollection", "facebookLeadAdsBearer")},
			"/{form_id}":                     {"get": op("getFacebookLeadgenForm", "Get lead generation form", params(path("form_id", "Lead generation form ID."), query("fields", "Comma-separated Graph API fields.")), "", "#/components/schemas/FacebookLeadAdsObject", "facebookLeadAdsBearer")},
			"/{form_id}/leads":               {"get": op("listFacebookLeadgenFormLeads", "List leads for a form", params(path("form_id", "Lead generation form ID."), query("fields", "Comma-separated Graph API fields."), query("limit", "Page size.")), "", "#/components/schemas/FacebookLeadAdsCollection", "facebookLeadAdsBearer")},
			"/{leadgen_id}":                  {"get": op("getFacebookLead", "Get lead details", params(path("leadgen_id", "Leadgen ID from a webhook or form lead list."), query("fields", "Comma-separated Graph API fields.")), "", "#/components/schemas/FacebookLeadAdsObject", "facebookLeadAdsBearer")},
			"/{page_id}/subscribed_apps":     {"get": op("listFacebookLeadAdsPageSubscriptions", "List page app subscriptions", params(path("page_id", "Facebook page ID.")), "", "#/components/schemas/FacebookLeadAdsCollection", "facebookLeadAdsBearer"), "post": op("subscribeFacebookLeadAdsPage", "Subscribe page to leadgen webhooks", params(path("page_id", "Facebook page ID.")), "#/components/schemas/FacebookLeadAdsObject", "#/components/schemas/FacebookLeadAdsObject", "facebookLeadAdsBearer")},
			"/{app_id}/subscriptions":        {"get": op("listFacebookLeadAdsAppSubscriptions", "List app webhook subscriptions", params(path("app_id", "Meta app ID.")), "", "#/components/schemas/FacebookLeadAdsCollection", "facebookLeadAdsBearer")},
			"/{ad_account_id}/leadgen_forms": {"get": op("listFacebookAdAccountLeadgenForms", "List ad account lead generation forms", params(path("ad_account_id", "Ad account ID."), query("fields", "Comma-separated Graph API fields."), query("limit", "Page size.")), "", "#/components/schemas/FacebookLeadAdsCollection", "facebookLeadAdsBearer")},
		},
	}
}

func linkedInOverlay() overlaySpec {
	security := map[string]map[string]any{
		"linkedinOAuth2": {"type": "http", "scheme": "bearer", "bearerFormat": "OAuth 2.0 access token", "description": "LinkedIn OAuth 2.0 access token carried in the Authorization bearer header."},
	}
	return overlaySpec{
		ProviderID:  "linkedin",
		Title:       "LinkedIn API Advisory Overlay",
		Description: "Advisory OpenAPI overlay derived from official LinkedIn human documentation. This is not an official LinkedIn OpenAPI document.",
		ServerURL:   "https://api.linkedin.com",
		Sources:     []string{"https://learn.microsoft.com/en-us/linkedin/marketing/", "https://learn.microsoft.com/en-us/linkedin/shared/authentication/authorization-code-flow", "https://learn.microsoft.com/en-us/linkedin/shared/api-guide/concepts/versioning"},
		SourceNote:  "LinkedIn publishes human REST API documentation but no recorded stable public official OpenAPI document; this overlay covers selected identity, organization, post, ad account, and lead-form endpoints.",
		Security:    security,
		Schemas:     []string{"LinkedInObject", "LinkedInCollection", "LinkedInError"},
		OutputPath:  "catalog-openapi-cache/advisory-overlays/linkedin-api-overlay.json",
		Paths: map[string]map[string]any{
			"/v2/userinfo":                 {"get": op("getLinkedInUserInfo", "Get OpenID Connect user info", nil, "", "#/components/schemas/LinkedInObject", "linkedinOAuth2")},
			"/rest/organizationAcls":       {"get": op("listLinkedInOrganizationAcls", "List organization ACLs", params(query("q", "LinkedIn finder such as roleAssignee."), query("role", "Organization role filter.")), "", "#/components/schemas/LinkedInCollection", "linkedinOAuth2")},
			"/rest/organizations/{org_id}": {"get": op("getLinkedInOrganization", "Get organization", params(path("org_id", "LinkedIn organization ID.")), "", "#/components/schemas/LinkedInObject", "linkedinOAuth2")},
			"/rest/posts":                  {"get": op("listLinkedInPosts", "Find posts", params(query("q", "LinkedIn finder for posts."), query("author", "Author URN filter."), query("count", "Page size.")), "", "#/components/schemas/LinkedInCollection", "linkedinOAuth2"), "post": op("createLinkedInPost", "Create post", nil, "#/components/schemas/LinkedInObject", "#/components/schemas/LinkedInObject", "linkedinOAuth2")},
			"/rest/adAccounts":             {"get": op("listLinkedInAdAccounts", "Search ad accounts", params(query("q", "LinkedIn finder for ad accounts."), query("search", "Search criteria.")), "", "#/components/schemas/LinkedInCollection", "linkedinOAuth2")},
			"/rest/leadForms":              {"get": op("listLinkedInLeadForms", "Find lead forms", params(query("q", "LinkedIn finder for lead forms."), query("owner", "Owner organization URN.")), "", "#/components/schemas/LinkedInCollection", "linkedinOAuth2")},
		},
	}
}

func messageBirdOverlay() overlaySpec {
	security := map[string]map[string]any{
		"messageBirdAccessKey": {"type": "apiKey", "in": "header", "name": "Authorization", "description": "MessageBird access key carried in the Authorization header using the AccessKey scheme."},
	}
	return overlaySpec{
		ProviderID:  "messagebird",
		Title:       "MessageBird API Advisory Overlay",
		Description: "Advisory OpenAPI overlay derived from official MessageBird human documentation. This is not an official MessageBird OpenAPI document.",
		ServerURL:   "https://rest.messagebird.com",
		Sources:     []string{"https://developers.messagebird.com/api/", "https://developers.messagebird.com/api/sms-messaging/", "https://developers.messagebird.com/api/signing-requests/"},
		SourceNote:  "MessageBird publishes human API documentation but no recorded stable public official OpenAPI document; this overlay covers selected balance, messages, contacts, voice calls, and HLR endpoints.",
		Security:    security,
		Schemas:     []string{"MessageBirdObject", "MessageBirdCollection", "MessageBirdError"},
		OutputPath:  "catalog-openapi-cache/advisory-overlays/messagebird-api-overlay.json",
		Paths: map[string]map[string]any{
			"/balance":               {"get": op("getMessageBirdBalance", "Get account balance", nil, "", "#/components/schemas/MessageBirdObject", "messageBirdAccessKey")},
			"/messages":              {"get": op("listMessageBirdMessages", "List SMS messages", params(query("limit", "Page size."), query("offset", "Pagination offset.")), "", "#/components/schemas/MessageBirdCollection", "messageBirdAccessKey"), "post": op("createMessageBirdMessage", "Send SMS message", nil, "#/components/schemas/MessageBirdObject", "#/components/schemas/MessageBirdObject", "messageBirdAccessKey")},
			"/messages/{message_id}": {"get": op("getMessageBirdMessage", "Get SMS message", params(path("message_id", "MessageBird message ID.")), "", "#/components/schemas/MessageBirdObject", "messageBirdAccessKey"), "delete": op("deleteMessageBirdMessage", "Delete SMS message", params(path("message_id", "MessageBird message ID.")), "", "", "messageBirdAccessKey")},
			"/contacts":              {"get": op("listMessageBirdContacts", "List contacts", params(query("limit", "Page size."), query("offset", "Pagination offset.")), "", "#/components/schemas/MessageBirdCollection", "messageBirdAccessKey"), "post": op("createMessageBirdContact", "Create contact", nil, "#/components/schemas/MessageBirdObject", "#/components/schemas/MessageBirdObject", "messageBirdAccessKey")},
			"/contacts/{contact_id}": {"get": op("getMessageBirdContact", "Get contact", params(path("contact_id", "MessageBird contact ID.")), "", "#/components/schemas/MessageBirdObject", "messageBirdAccessKey"), "patch": op("updateMessageBirdContact", "Update contact", params(path("contact_id", "MessageBird contact ID.")), "#/components/schemas/MessageBirdObject", "#/components/schemas/MessageBirdObject", "messageBirdAccessKey")},
			"/voice_calls":           {"get": op("listMessageBirdVoiceCalls", "List voice calls", params(query("limit", "Page size."), query("offset", "Pagination offset.")), "", "#/components/schemas/MessageBirdCollection", "messageBirdAccessKey"), "post": op("createMessageBirdVoiceCall", "Create voice call", nil, "#/components/schemas/MessageBirdObject", "#/components/schemas/MessageBirdObject", "messageBirdAccessKey")},
			"/hlr":                   {"post": op("createMessageBirdHLR", "Request HLR lookup", nil, "#/components/schemas/MessageBirdObject", "#/components/schemas/MessageBirdObject", "messageBirdAccessKey")},
			"/hlr/{hlr_id}":          {"get": op("getMessageBirdHLR", "Get HLR lookup", params(path("hlr_id", "MessageBird HLR ID.")), "", "#/components/schemas/MessageBirdObject", "messageBirdAccessKey")},
		},
	}
}

func twakeOverlay() overlaySpec {
	security := map[string]map[string]any{
		"twakeBasic": {"type": "http", "scheme": "basic", "description": "Twake application public_id and private_api_key carried using HTTP Basic authentication."},
	}
	return overlaySpec{
		ProviderID:  "twake",
		Title:       "Twake Developers API Advisory Overlay",
		Description: "Advisory OpenAPI overlay derived from official Twake human documentation. This is not an official Twake OpenAPI document.",
		ServerURL:   "https://api.twake.app",
		Sources:     []string{"https://doc.twake.app/developers-api/get-started", "https://doc.twake.app/developers-api/api-reference/auth", "https://doc.twake.app/developers-api/api-reference/message/post-request", "https://doc.twake.app/developers-api/api-reference/message/delete-request"},
		SourceNote:  "Twake publishes human Developers API documentation but no recorded stable public official OpenAPI document; this overlay covers the documented application-authenticated message save/remove subset.",
		Security:    security,
		Schemas:     []string{"TwakeObject", "TwakeMessageRequest", "TwakeError"},
		OutputPath:  "catalog-openapi-cache/advisory-overlays/twake-developers-api-overlay.json",
		Paths: map[string]map[string]any{
			"/api/v1/messages/save":   {"post": op("saveTwakeMessage", "Send message to a channel", nil, "#/components/schemas/TwakeMessageRequest", "#/components/schemas/TwakeObject", "twakeBasic")},
			"/api/v1/messages/remove": {"post": op("removeTwakeMessage", "Delete message from a channel", nil, "#/components/schemas/TwakeMessageRequest", "#/components/schemas/TwakeObject", "twakeBasic")},
		},
	}
}

func twistOverlay() overlaySpec {
	security := map[string]map[string]any{
		"twistBearer": {"type": "http", "scheme": "bearer", "bearerFormat": "Twist OAuth 2.0 or test token", "description": "Twist API token carried in the Authorization bearer header."},
	}
	return overlaySpec{
		ProviderID:  "twist",
		Title:       "Twist API v3 Advisory Overlay",
		Description: "Advisory OpenAPI overlay derived from official Twist API v3 human documentation. This is not an official Twist OpenAPI document.",
		ServerURL:   "https://api.twist.com/api/v3",
		Sources:     []string{"https://developer.twist.com/v3/"},
		SourceNote:  "Twist publishes human API v3 documentation but no recorded stable public official OpenAPI document; this overlay covers selected user, workspace, channel, thread, comment, message, and webhook endpoints.",
		Security:    security,
		Schemas:     []string{"TwistObject", "TwistCollection", "TwistError"},
		OutputPath:  "catalog-openapi-cache/advisory-overlays/twist-api-v3-overlay.json",
		Paths: map[string]map[string]any{
			"/users/get_current":         {"get": op("getTwistCurrentUser", "Get current user", nil, "", "#/components/schemas/TwistObject", "twistBearer")},
			"/workspaces/get":            {"get": op("getTwistWorkspace", "Get workspace", params(query("id", "Workspace ID.")), "", "#/components/schemas/TwistObject", "twistBearer")},
			"/workspaces/get_all":        {"get": op("listTwistWorkspaces", "Get all workspaces", nil, "", "#/components/schemas/TwistCollection", "twistBearer")},
			"/channels/get":              {"get": op("getTwistChannel", "Get channel", params(query("id", "Channel ID.")), "", "#/components/schemas/TwistObject", "twistBearer")},
			"/channels/get_all":          {"get": op("listTwistChannels", "Get all channels", params(query("workspace_id", "Workspace ID.")), "", "#/components/schemas/TwistCollection", "twistBearer")},
			"/threads/get":               {"get": op("getTwistThread", "Get thread", params(query("id", "Thread ID.")), "", "#/components/schemas/TwistObject", "twistBearer")},
			"/threads/add":               {"post": op("createTwistThread", "Add thread", nil, "#/components/schemas/TwistObject", "#/components/schemas/TwistObject", "twistBearer")},
			"/comments/add":              {"post": op("createTwistComment", "Add comment", nil, "#/components/schemas/TwistObject", "#/components/schemas/TwistObject", "twistBearer")},
			"/conversation_messages/add": {"post": op("createTwistConversationMessage", "Add conversation message", nil, "#/components/schemas/TwistObject", "#/components/schemas/TwistObject", "twistBearer")},
			"/hooks/subscribe":           {"post": op("subscribeTwistHook", "Subscribe webhook", nil, "#/components/schemas/TwistObject", "#/components/schemas/TwistObject", "twistBearer")},
			"/hooks/unsubscribe":         {"post": op("unsubscribeTwistHook", "Unsubscribe webhook", nil, "#/components/schemas/TwistObject", "#/components/schemas/TwistObject", "twistBearer")},
		},
	}
}

func whatsAppOverlay() overlaySpec {
	security := map[string]map[string]any{
		"whatsAppBearer": {"type": "http", "scheme": "bearer", "bearerFormat": "Meta Graph API access token", "description": "WhatsApp Cloud API access token carried in the Authorization bearer header."},
	}
	return overlaySpec{
		ProviderID:  "whatsapp",
		Title:       "WhatsApp Cloud API Advisory Overlay",
		Description: "Advisory OpenAPI overlay derived from official Meta WhatsApp Cloud API human documentation. This is not an official Meta OpenAPI document.",
		ServerURL:   "https://graph.facebook.com/{graph_api_version}",
		Sources:     []string{"https://developers.facebook.com/docs/whatsapp/cloud-api/", "https://developers.facebook.com/docs/whatsapp/cloud-api/reference/messages", "https://developers.facebook.com/docs/whatsapp/cloud-api/reference/media"},
		SourceNote:  "WhatsApp Cloud API uses Meta Graph API human documentation but no recorded stable public official OpenAPI document; this overlay covers selected message, media, template, phone-number, and webhook endpoints.",
		Security:    security,
		Schemas:     []string{"WhatsAppObject", "WhatsAppCollection", "WhatsAppError"},
		OutputPath:  "catalog-openapi-cache/advisory-overlays/whatsapp-cloud-api-overlay.json",
		Paths: map[string]map[string]any{
			"/{phone_number_id}/messages":              {"post": op("sendWhatsAppMessage", "Send WhatsApp message", params(path("phone_number_id", "WhatsApp phone number ID.")), "#/components/schemas/WhatsAppObject", "#/components/schemas/WhatsAppObject", "whatsAppBearer")},
			"/{phone_number_id}/media":                 {"post": op("uploadWhatsAppMedia", "Upload media", params(path("phone_number_id", "WhatsApp phone number ID.")), "#/components/schemas/WhatsAppObject", "#/components/schemas/WhatsAppObject", "whatsAppBearer")},
			"/{media_id}":                              {"get": op("getWhatsAppMedia", "Get media metadata", params(path("media_id", "WhatsApp media ID.")), "", "#/components/schemas/WhatsAppObject", "whatsAppBearer"), "delete": op("deleteWhatsAppMedia", "Delete media", params(path("media_id", "WhatsApp media ID.")), "", "", "whatsAppBearer")},
			"/{business_account_id}/message_templates": {"get": op("listWhatsAppMessageTemplates", "List message templates", params(path("business_account_id", "WhatsApp Business Account ID."), query("limit", "Page size.")), "", "#/components/schemas/WhatsAppCollection", "whatsAppBearer"), "post": op("createWhatsAppMessageTemplate", "Create message template", params(path("business_account_id", "WhatsApp Business Account ID.")), "#/components/schemas/WhatsAppObject", "#/components/schemas/WhatsAppObject", "whatsAppBearer")},
			"/{phone_number_id}":                       {"get": op("getWhatsAppPhoneNumber", "Get phone number metadata", params(path("phone_number_id", "WhatsApp phone number ID."), query("fields", "Comma-separated Graph API fields.")), "", "#/components/schemas/WhatsAppObject", "whatsAppBearer")},
			"/{app_id}/subscriptions":                  {"get": op("listWhatsAppAppSubscriptions", "List app webhook subscriptions", params(path("app_id", "Meta app ID.")), "", "#/components/schemas/WhatsAppCollection", "whatsAppBearer")},
		},
	}
}

func build(spec overlaySpec) map[string]any {
	return map[string]any{
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
}

func op(operationID, summary string, parameters []map[string]any, requestRef, responseRef string, securityNames ...string) map[string]any {
	out := map[string]any{
		"operationId": operationID,
		"summary":     summary,
		"description": "Advisory operation derived from official human API documentation.",
		"responses": map[string]any{
			"200":     response(responseRef),
			"default": map[string]any{"description": "Provider error response."},
		},
	}
	if len(securityNames) > 0 {
		requirement := map[string][]string{}
		for _, name := range securityNames {
			if name != "" {
				requirement[name] = []string{}
			}
		}
		if len(requirement) > 0 {
			out["security"] = []map[string][]string{requirement}
		}
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

func params(items ...map[string]any) []map[string]any { return items }

func path(name, description string) map[string]any { return parameter(name, "path", description, true) }

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
	out := map[string]V{}
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
