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
		moceanOverlay(),
		msg91Overlay(),
		plivoOverlay(),
		pushbulletOverlay(),
		pushcutOverlay(),
		pushoverOverlay(),
		sms77Overlay(),
	} {
		write(spec.OutputPath, build(spec))
	}
}

func moceanOverlay() overlaySpec {
	security := map[string]map[string]any{
		"moceanAPIKey":    {"type": "apiKey", "in": "query", "name": "mocean-api-key", "description": "Mocean API key carried in the mocean-api-key request parameter."},
		"moceanAPISecret": {"type": "apiKey", "in": "query", "name": "mocean-api-secret", "description": "Mocean API secret carried in the mocean-api-secret request parameter."},
	}
	return overlaySpec{
		ProviderID:  "mocean",
		Title:       "Mocean API Advisory Overlay",
		Description: "Advisory OpenAPI overlay derived from official Mocean human API documentation. This is not an official Mocean OpenAPI document.",
		ServerURL:   "https://rest.moceanapi.com",
		Sources:     []string{"https://moceanapi.com/docs"},
		SourceNote:  "Mocean publishes human API documentation but no recorded stable public official OpenAPI document; this overlay covers selected SMS, voice, and account endpoints.",
		Security:    security,
		Schemas:     []string{"MoceanObject", "MoceanCollection", "MoceanError"},
		OutputPath:  "catalog-openapi-cache/advisory-overlays/mocean-api-overlay.json",
		Paths: map[string]map[string]any{
			"/rest/2/account/balance": {"get": op("getMoceanBalance", "Get account balance", nil, "", "#/components/schemas/MoceanObject", "moceanAPIKey", "moceanAPISecret")},
			"/rest/2/sms":             {"post": op("sendMoceanSMS", "Send SMS", nil, "#/components/schemas/MoceanObject", "#/components/schemas/MoceanObject", "moceanAPIKey", "moceanAPISecret")},
			"/rest/2/voice/dial":      {"post": op("sendMoceanVoice", "Start voice call", nil, "#/components/schemas/MoceanObject", "#/components/schemas/MoceanObject", "moceanAPIKey", "moceanAPISecret")},
		},
	}
}

func msg91Overlay() overlaySpec {
	security := map[string]map[string]any{
		"msg91AuthKey": {"type": "apiKey", "in": "query", "name": "authkey", "description": "MSG91 auth key carried in the authkey request parameter."},
	}
	return overlaySpec{
		ProviderID:  "msg91",
		Title:       "MSG91 API Advisory Overlay",
		Description: "Advisory OpenAPI overlay derived from official MSG91 human API documentation. This is not an official MSG91 OpenAPI document.",
		ServerURL:   "https://api.msg91.com/api",
		Sources:     []string{"https://docs.msg91.com/sms", "https://docs.msg91.com/overview"},
		SourceNote:  "MSG91 publishes human API documentation but no recorded stable public official OpenAPI document; this overlay covers selected SMS and OTP endpoints.",
		Security:    security,
		Schemas:     []string{"MSG91Object", "MSG91Collection", "MSG91Error"},
		OutputPath:  "catalog-openapi-cache/advisory-overlays/msg91-api-overlay.json",
		Paths: map[string]map[string]any{
			"/sendhttp.php":  {"get": op("sendMSG91SMS", "Send SMS", nil, "", "#/components/schemas/MSG91Object", "msg91AuthKey")},
			"/v2/sendsms":    {"post": op("sendMSG91TransactionalSMS", "Send transactional SMS", nil, "#/components/schemas/MSG91Object", "#/components/schemas/MSG91Object", "msg91AuthKey")},
			"/v5/otp":        {"post": op("sendMSG91OTP", "Send OTP", nil, "#/components/schemas/MSG91Object", "#/components/schemas/MSG91Object", "msg91AuthKey")},
			"/v5/otp/verify": {"get": op("verifyMSG91OTP", "Verify OTP", params(query("otp", "OTP value."), query("mobile", "Recipient mobile number.")), "", "#/components/schemas/MSG91Object", "msg91AuthKey"), "post": op("verifyMSG91OTPWithBody", "Verify OTP with request body", nil, "#/components/schemas/MSG91Object", "#/components/schemas/MSG91Object", "msg91AuthKey")},
		},
	}
}

func plivoOverlay() overlaySpec {
	security := map[string]map[string]any{
		"plivoBasic": {"type": "http", "scheme": "basic", "description": "Plivo Auth ID and Auth Token supplied using HTTP Basic authentication."},
	}
	return overlaySpec{
		ProviderID:  "plivo",
		Title:       "Plivo API Advisory Overlay",
		Description: "Advisory OpenAPI overlay derived from official Plivo human API documentation. This is not an official Plivo OpenAPI document.",
		ServerURL:   "https://api.plivo.com/v1/Account/{auth_id}",
		Sources:     []string{"https://www.plivo.com/docs/messaging/api/message/retrieve-a-message", "https://www.plivo.com/docs/voice/api/call/the-call-object/"},
		SourceNote:  "Plivo publishes human Messaging and Voice API documentation but no recorded stable public official OpenAPI document; this overlay covers selected message and call endpoints.",
		Security:    security,
		Schemas:     []string{"PlivoObject", "PlivoCollection", "PlivoError"},
		OutputPath:  "catalog-openapi-cache/advisory-overlays/plivo-api-overlay.json",
		Paths: map[string]map[string]any{
			"/Call/":                   {"get": op("listPlivoCalls", "List calls", nil, "", "#/components/schemas/PlivoCollection", "plivoBasic"), "post": op("makePlivoCall", "Make call", nil, "#/components/schemas/PlivoObject", "#/components/schemas/PlivoObject", "plivoBasic")},
			"/Call/{call_uuid}/":       {"get": op("getPlivoCall", "Get call", params(path("call_uuid", "Call UUID.")), "", "#/components/schemas/PlivoObject", "plivoBasic")},
			"/Message/":                {"get": op("listPlivoMessages", "List messages", nil, "", "#/components/schemas/PlivoCollection", "plivoBasic"), "post": op("sendPlivoMessage", "Send message", nil, "#/components/schemas/PlivoObject", "#/components/schemas/PlivoObject", "plivoBasic")},
			"/Message/{message_uuid}/": {"get": op("getPlivoMessage", "Get message", params(path("message_uuid", "Message UUID.")), "", "#/components/schemas/PlivoObject", "plivoBasic")},
		},
	}
}

func pushbulletOverlay() overlaySpec {
	security := map[string]map[string]any{
		"pushbulletBearer": {"type": "http", "scheme": "bearer", "bearerFormat": "Pushbullet OAuth token", "description": "Pushbullet OAuth access token carried in the Authorization bearer header."},
	}
	return overlaySpec{
		ProviderID:  "pushbullet",
		Title:       "Pushbullet API Advisory Overlay",
		Description: "Advisory OpenAPI overlay derived from official Pushbullet human API documentation. This is not an official Pushbullet OpenAPI document.",
		ServerURL:   "https://api.pushbullet.com/v2",
		Sources:     []string{"https://docs.pushbullet.com/"},
		SourceNote:  "Pushbullet publishes human API documentation but no recorded stable public official OpenAPI document; this overlay covers selected pushes, upload, devices, contacts, channels, and subscriptions endpoints.",
		Security:    security,
		Schemas:     []string{"PushbulletObject", "PushbulletCollection", "PushbulletError"},
		OutputPath:  "catalog-openapi-cache/advisory-overlays/pushbullet-api-overlay.json",
		Paths: map[string]map[string]any{
			"/channels":           {"get": op("listPushbulletChannels", "List channels", nil, "", "#/components/schemas/PushbulletCollection", "pushbulletBearer")},
			"/contacts":           {"get": op("listPushbulletContacts", "List contacts", nil, "", "#/components/schemas/PushbulletCollection", "pushbulletBearer")},
			"/devices":            {"get": op("listPushbulletDevices", "List devices", nil, "", "#/components/schemas/PushbulletCollection", "pushbulletBearer")},
			"/pushes":             {"get": op("listPushbulletPushes", "List pushes", params(query("modified_after", "Only return pushes modified after this timestamp.")), "", "#/components/schemas/PushbulletCollection", "pushbulletBearer"), "post": op("createPushbulletPush", "Create push", nil, "#/components/schemas/PushbulletObject", "#/components/schemas/PushbulletObject", "pushbulletBearer")},
			"/pushes/{push_iden}": {"delete": op("deletePushbulletPush", "Delete push", params(path("push_iden", "Push identifier.")), "", "#/components/schemas/PushbulletObject", "pushbulletBearer"), "post": op("updatePushbulletPush", "Update push", params(path("push_iden", "Push identifier.")), "#/components/schemas/PushbulletObject", "#/components/schemas/PushbulletObject", "pushbulletBearer")},
			"/subscriptions":      {"get": op("listPushbulletSubscriptions", "List subscriptions", nil, "", "#/components/schemas/PushbulletCollection", "pushbulletBearer")},
			"/upload-request":     {"post": op("createPushbulletUploadRequest", "Create upload request", nil, "#/components/schemas/PushbulletObject", "#/components/schemas/PushbulletObject", "pushbulletBearer")},
		},
	}
}

func pushcutOverlay() overlaySpec {
	security := map[string]map[string]any{
		"pushcutAPIKey": {"type": "apiKey", "in": "header", "name": "API-Key", "description": "Pushcut API key carried in the API-Key header."},
	}
	return overlaySpec{
		ProviderID:  "pushcut",
		Title:       "Pushcut API Advisory Overlay",
		Description: "Advisory OpenAPI overlay derived from official Pushcut human API documentation. This is not an official Pushcut OpenAPI document.",
		ServerURL:   "https://api.pushcut.io/v1",
		Sources:     []string{"https://www.pushcut.io/support/integrations", "https://www.pushcut.io/support/notifications"},
		SourceNote:  "Pushcut publishes human API documentation but no recorded stable public official OpenAPI document; this overlay covers selected notifications, devices, and subscriptions endpoints.",
		Security:    security,
		Schemas:     []string{"PushcutObject", "PushcutCollection", "PushcutError"},
		OutputPath:  "catalog-openapi-cache/advisory-overlays/pushcut-api-overlay.json",
		Paths: map[string]map[string]any{
			"/devices":                           {"get": op("listPushcutDevices", "List devices", nil, "", "#/components/schemas/PushcutCollection", "pushcutAPIKey")},
			"/notifications":                     {"get": op("listPushcutNotifications", "List notifications", nil, "", "#/components/schemas/PushcutCollection", "pushcutAPIKey")},
			"/notifications/{notification_name}": {"post": op("sendPushcutNotification", "Send notification", params(path("notification_name", "Notification name.")), "#/components/schemas/PushcutObject", "#/components/schemas/PushcutObject", "pushcutAPIKey")},
			"/subscriptions":                     {"get": op("listPushcutSubscriptions", "List subscriptions", nil, "", "#/components/schemas/PushcutCollection", "pushcutAPIKey"), "post": op("createPushcutSubscription", "Create subscription", nil, "#/components/schemas/PushcutObject", "#/components/schemas/PushcutObject", "pushcutAPIKey")},
			"/subscriptions/{subscription_id}":   {"delete": op("deletePushcutSubscription", "Delete subscription", params(path("subscription_id", "Subscription ID.")), "", "#/components/schemas/PushcutObject", "pushcutAPIKey")},
		},
	}
}

func pushoverOverlay() overlaySpec {
	security := map[string]map[string]any{
		"pushoverAppToken": {"type": "apiKey", "in": "query", "name": "token", "description": "Pushover application API token carried in the token request parameter."},
		"pushoverUserKey":  {"type": "apiKey", "in": "query", "name": "user", "description": "Pushover user or group key carried in the user request parameter."},
	}
	return overlaySpec{
		ProviderID:  "pushover",
		Title:       "Pushover API Advisory Overlay",
		Description: "Advisory OpenAPI overlay derived from official Pushover human API documentation. This is not an official Pushover OpenAPI document.",
		ServerURL:   "https://api.pushover.net/1",
		Sources:     []string{"https://pushover.net/api", "https://pushover.net/api/client"},
		SourceNote:  "Pushover publishes human API documentation but no recorded stable public official OpenAPI document; this overlay covers selected message, receipt, user validation, and sound endpoints.",
		Security:    security,
		Schemas:     []string{"PushoverObject", "PushoverCollection", "PushoverError"},
		OutputPath:  "catalog-openapi-cache/advisory-overlays/pushover-api-overlay.json",
		Paths: map[string]map[string]any{
			"/messages.json":           {"post": op("sendPushoverMessage", "Send message", nil, "#/components/schemas/PushoverObject", "#/components/schemas/PushoverObject", "pushoverAppToken", "pushoverUserKey")},
			"/receipts/{receipt}.json": {"get": op("getPushoverReceipt", "Get receipt", params(path("receipt", "Receipt token.")), "", "#/components/schemas/PushoverObject", "pushoverAppToken", "pushoverUserKey")},
			"/sounds.json":             {"get": op("listPushoverSounds", "List sounds", nil, "", "#/components/schemas/PushoverCollection", "pushoverAppToken", "pushoverUserKey")},
			"/users/validate.json":     {"post": op("validatePushoverUser", "Validate user or group", nil, "#/components/schemas/PushoverObject", "#/components/schemas/PushoverObject", "pushoverAppToken", "pushoverUserKey")},
		},
	}
}

func sms77Overlay() overlaySpec {
	security := map[string]map[string]any{
		"sevenAPIKey": {"type": "apiKey", "in": "header", "name": "X-Api-Key", "description": "seven.io API key carried in the X-Api-Key header."},
	}
	return overlaySpec{
		ProviderID:  "sms77",
		Title:       "seven.io sms77 API Advisory Overlay",
		Description: "Advisory OpenAPI overlay derived from official seven.io human API documentation. This is not an official seven.io OpenAPI document.",
		ServerURL:   "https://gateway.seven.io/api",
		Sources:     []string{"https://docs.seven.io/en/rest-api/endpoints/sms", "https://docs.seven.io/en/rest-api/authentication"},
		SourceNote:  "seven.io publishes human API documentation but no recorded stable public official OpenAPI document; this overlay covers selected SMS, voice, balance, pricing, status, and contacts endpoints.",
		Security:    security,
		Schemas:     []string{"SevenObject", "SevenCollection", "SevenError"},
		OutputPath:  "catalog-openapi-cache/advisory-overlays/sms77-api-overlay.json",
		Paths: map[string]map[string]any{
			"/balance":  {"get": op("getSevenBalance", "Get account balance", nil, "", "#/components/schemas/SevenObject", "sevenAPIKey")},
			"/contacts": {"get": op("listSevenContacts", "List contacts", nil, "", "#/components/schemas/SevenCollection", "sevenAPIKey"), "post": op("createSevenContact", "Create contact", nil, "#/components/schemas/SevenObject", "#/components/schemas/SevenObject", "sevenAPIKey")},
			"/pricing":  {"get": op("getSevenPricing", "Get pricing", nil, "", "#/components/schemas/SevenObject", "sevenAPIKey")},
			"/sms":      {"post": op("sendSevenSMS", "Send SMS", nil, "#/components/schemas/SevenObject", "#/components/schemas/SevenObject", "sevenAPIKey")},
			"/status":   {"get": op("getSevenMessageStatus", "Get message status", params(query("msg_id", "Message ID.")), "", "#/components/schemas/SevenObject", "sevenAPIKey")},
			"/voice":    {"post": op("sendSevenVoice", "Send voice message", nil, "#/components/schemas/SevenObject", "#/components/schemas/SevenObject", "sevenAPIKey")},
		},
	}
}

func build(spec overlaySpec) map[string]any {
	return map[string]any{
		"openapi": "3.0.3",
		"info": map[string]any{
			"title":       spec.Title,
			"version":     "2026-05-20",
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
