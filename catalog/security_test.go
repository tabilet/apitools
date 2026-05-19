package catalog

import (
	"reflect"
	"testing"
)

func TestClassifyAuthCompleteness(t *testing.T) {
	complete := SecurityMetadata{
		SecuritySchemes: []SecurityScheme{bearerScheme("bearerAuth")},
		RootSecurity:    []SecurityRequirement{{Scheme: "bearerAuth"}},
	}
	tests := []struct {
		name     string
		metadata SecurityMetadata
		want     AuthCompletenessStatus
	}{
		{
			name: "complete",
			metadata: SecurityMetadata{
				SecuritySchemes: []SecurityScheme{bearerScheme("bearerAuth")},
				OperationSecurity: []OperationSecurity{
					{
						Match:    OperationMatch{OperationID: "listThings"},
						Security: []SecurityRequirement{{Scheme: "bearerAuth"}},
					},
				},
			},
			want: AuthStatusComplete,
		},
		{
			name: "present incomplete without requirements",
			metadata: SecurityMetadata{
				SecuritySchemes: []SecurityScheme{bearerScheme("bearerAuth")},
			},
			want: AuthStatusPresentIncomplete,
		},
		{
			name: "present incomplete unknown scheme",
			metadata: SecurityMetadata{
				SecuritySchemes: []SecurityScheme{bearerScheme("bearerAuth")},
				RootSecurity:    []SecurityRequirement{{Scheme: "missingAuth"}},
			},
			want: AuthStatusPresentIncomplete,
		},
		{
			name: "present incomplete duplicate scheme",
			metadata: SecurityMetadata{
				SecuritySchemes: []SecurityScheme{bearerScheme("bearerAuth"), bearerScheme("bearerAuth")},
				RootSecurity:    []SecurityRequirement{{Scheme: "bearerAuth"}},
			},
			want: AuthStatusPresentIncomplete,
		},
		{
			name: "present incomplete invalid operation match",
			metadata: SecurityMetadata{
				SecuritySchemes: []SecurityScheme{bearerScheme("bearerAuth")},
				OperationSecurity: []OperationSecurity{
					{Security: []SecurityRequirement{{Scheme: "bearerAuth"}}},
				},
			},
			want: AuthStatusPresentIncomplete,
		},
		{
			name:     "absent",
			metadata: SecurityMetadata{},
			want:     AuthStatusAbsent,
		},
		{
			name:     "intentionally anonymous",
			metadata: SecurityMetadata{IntentionallyAnonymous: true},
			want:     AuthStatusIntentionallyAnonymous,
		},
		{
			name:     "complete with root",
			metadata: complete,
			want:     AuthStatusComplete,
		},
		{
			name: "complete with combined requirement set",
			metadata: SecurityMetadata{
				SecuritySchemes: []SecurityScheme{bearerScheme("tokenAuth"), bearerScheme("userAuth")},
				RootSecuritySets: []SecurityRequirementSet{{
					Requirements: []SecurityRequirement{{Scheme: "tokenAuth"}, {Scheme: "userAuth"}},
				}},
			},
			want: AuthStatusComplete,
		},
		{
			name: "present incomplete combined requirement set unknown scheme",
			metadata: SecurityMetadata{
				SecuritySchemes: []SecurityScheme{bearerScheme("tokenAuth")},
				RootSecuritySets: []SecurityRequirementSet{{
					Requirements: []SecurityRequirement{{Scheme: "tokenAuth"}, {Scheme: "userAuth"}},
				}},
			},
			want: AuthStatusPresentIncomplete,
		},
	}
	for _, test := range tests {
		if got := ClassifyAuthCompleteness(test.metadata); got != test.want {
			t.Fatalf("%s: ClassifyAuthCompleteness() = %q, want %q", test.name, got, test.want)
		}
	}
}

func TestBuiltInSecurityOverlaysValidate(t *testing.T) {
	overlays := BuiltInSecurityOverlays()
	if got, want := len(overlays), 120; got != want {
		t.Fatalf("len(BuiltInSecurityOverlays()) = %d, want %d", got, want)
	}
	if err := ValidateSecurityOverlays(overlays, BuiltInProviders()); err != nil {
		t.Fatalf("ValidateSecurityOverlays() error = %v", err)
	}
}

func TestSecurityOverlayValidationRejectsBadRecords(t *testing.T) {
	base := SecurityOverlay{
		ID:         "example-auth-overlay",
		ProviderID: "airtable",
		SpecRefID:  "airtable-web-api-docs",
		Status:     AuthStatusOverlayRequired,
		SecuritySchemes: []SecurityScheme{
			bearerScheme("exampleAuth"),
		},
		RootSecurity: []SecurityRequirement{{Scheme: "exampleAuth"}},
		SourceRefs:   []string{"https://example.com/auth"},
		SourceNote:   "Source-backed test overlay.",
	}
	tests := []struct {
		name   string
		mutate func(SecurityOverlay) SecurityOverlay
	}{
		{
			name: "unknown provider",
			mutate: func(overlay SecurityOverlay) SecurityOverlay {
				overlay.ProviderID = "missing"
				return overlay
			},
		},
		{
			name: "unknown spec ref",
			mutate: func(overlay SecurityOverlay) SecurityOverlay {
				overlay.SpecRefID = "missing-docs"
				return overlay
			},
		},
		{
			name: "duplicate scheme",
			mutate: func(overlay SecurityOverlay) SecurityOverlay {
				overlay.SecuritySchemes = append(overlay.SecuritySchemes, bearerScheme("exampleAuth"))
				return overlay
			},
		},
		{
			name: "unknown security requirement",
			mutate: func(overlay SecurityOverlay) SecurityOverlay {
				overlay.RootSecurity = []SecurityRequirement{{Scheme: "missingAuth"}}
				return overlay
			},
		},
		{
			name: "duplicate requirement in set",
			mutate: func(overlay SecurityOverlay) SecurityOverlay {
				overlay.RootSecuritySets = []SecurityRequirementSet{{
					Requirements: []SecurityRequirement{{Scheme: "exampleAuth"}, {Scheme: "exampleAuth"}},
				}}
				return overlay
			},
		},
		{
			name: "bad operation match",
			mutate: func(overlay SecurityOverlay) SecurityOverlay {
				overlay.OperationSecurity = []OperationSecurity{{Security: []SecurityRequirement{{Scheme: "exampleAuth"}}}}
				return overlay
			},
		},
		{
			name: "insecure source ref",
			mutate: func(overlay SecurityOverlay) SecurityOverlay {
				overlay.SourceRefs = []string{"http://example.com/auth"}
				return overlay
			},
		},
	}
	for _, test := range tests {
		if err := ValidateSecurityOverlays([]SecurityOverlay{test.mutate(base)}, BuiltInProviders()); err == nil {
			t.Fatalf("%s: ValidateSecurityOverlays() expected error", test.name)
		}
	}
}

func TestBuiltInSecurityReportDeterministic(t *testing.T) {
	report, err := BuiltInSecurityReport()
	if err != nil {
		t.Fatalf("BuiltInSecurityReport() error = %v", err)
	}
	gotIDs := make([]string, 0, len(report.Providers))
	statusByID := map[string]AuthCompletenessStatus{}
	for _, provider := range report.Providers {
		gotIDs = append(gotIDs, provider.ProviderID)
		statusByID[provider.ProviderID] = provider.Status
	}
	wantIDs := []string{
		"action-network",
		"activecampaign",
		"acuity-scheduling",
		"adalo",
		"affinity",
		"agile-crm",
		"airtable",
		"airtop",
		"asana",
		"autopilot",
		"aws-lambda",
		"aws-s3",
		"aws-sns",
		"bamboohr",
		"bannerbear",
		"baserow",
		"beeminder",
		"bitbucket",
		"bitly",
		"box",
		"brandfetch",
		"brevo",
		"bubble",
		"cal",
		"calendly",
		"chargebee",
		"circleci",
		"clearbit",
		"clickup",
		"clockify",
		"cloudflare",
		"cockpit",
		"coda",
		"coingecko",
		"contentful",
		"convertkit",
		"copper",
		"customer-io",
		"databricks",
		"deepl",
		"dhl",
		"discord",
		"discourse",
		"drift",
		"dropbox",
		"elastic",
		"erpnext",
		"eventbrite",
		"facebook",
		"facebook-lead-ads",
		"figma",
		"filemaker",
		"formio",
		"formstack",
		"freshdesk",
		"freshservice",
		"freshworks-crm",
		"getresponse",
		"ghost",
		"github",
		"gitlab",
		"gmail",
		"gong",
		"google-calendar",
		"google-drive",
		"google-sheets",
		"grafana",
		"grist",
		"gumroad",
		"hackernews",
		"harvest",
		"help-scout",
		"highlevel",
		"hubspot",
		"intercom",
		"invoice-ninja",
		"iterable",
		"jenkins",
		"jira-cloud",
		"jotform",
		"keap",
		"kobotoolbox",
		"line",
		"linear",
		"linkedin",
		"magento",
		"mailchimp",
		"mailerlite",
		"mailgun",
		"mailjet",
		"marketstack",
		"matrix",
		"mattermost",
		"mautic",
		"messagebird",
		"microsoft-graph",
		"monday-com",
		"monica-crm",
		"nasa",
		"netlify",
		"nocodb",
		"notion",
		"odoo",
		"okta",
		"onesimpleapi",
		"onfleet",
		"openthesaurus",
		"openweathermap",
		"paddle",
		"pagerduty",
		"paypal",
		"pipedrive",
		"posthog",
		"postmark",
		"quickbooks",
		"quickchart",
		"reddit",
		"rocket-chat",
		"salesforce",
		"salesmate",
		"seatable",
		"sendgrid",
		"sendy",
		"sentry",
		"servicenow",
		"shopify",
		"slack",
		"snowflake",
		"splunk",
		"stackby",
		"stripe",
		"supabase",
		"surveymonkey",
		"telegram",
		"timesaved",
		"todoist",
		"trello",
		"twake",
		"twilio",
		"twist",
		"twitter",
		"typeform",
		"unleashed-software",
		"uptimerobot",
		"vero",
		"webflow",
		"whatsapp",
		"wise",
		"woocommerce",
		"workable",
		"xero",
		"zendesk",
		"zoom",
	}
	if !reflect.DeepEqual(gotIDs, wantIDs) {
		t.Fatalf("report provider ids = %#v, want %#v", gotIDs, wantIDs)
	}
	wantStatuses := map[string]AuthCompletenessStatus{
		"activecampaign":     AuthStatusOverlayRequired,
		"action-network":     AuthStatusOverlayRequired,
		"adalo":              AuthStatusOverlayRequired,
		"affinity":           AuthStatusOverlayRequired,
		"agile-crm":          AuthStatusOverlayRequired,
		"acuity-scheduling":  AuthStatusOverlayRequired,
		"airtable":           AuthStatusOverlayRequired,
		"airtop":             AuthStatusPresentIncomplete,
		"asana":              AuthStatusPresentIncomplete,
		"autopilot":          AuthStatusOverlayRequired,
		"aws-lambda":         AuthStatusOverlayRequired,
		"aws-s3":             AuthStatusOverlayRequired,
		"aws-sns":            AuthStatusOverlayRequired,
		"bamboohr":           AuthStatusOverlayRequired,
		"bannerbear":         AuthStatusOverlayRequired,
		"baserow":            AuthStatusComplete,
		"beeminder":          AuthStatusOverlayRequired,
		"bitbucket":          AuthStatusPresentIncomplete,
		"bitly":              AuthStatusComplete,
		"box":                AuthStatusPresentIncomplete,
		"brandfetch":         AuthStatusComplete,
		"brevo":              AuthStatusComplete,
		"bubble":             AuthStatusOverlayRequired,
		"cal":                AuthStatusPresentIncomplete,
		"calendly":           AuthStatusOverlayRequired,
		"chargebee":          AuthStatusPresentIncomplete,
		"circleci":           AuthStatusPresentIncomplete,
		"clearbit":           AuthStatusOverlayRequired,
		"clickup":            AuthStatusPresentIncomplete,
		"clockify":           AuthStatusOverlayRequired,
		"cloudflare":         AuthStatusPresentIncomplete,
		"coda":               AuthStatusComplete,
		"cockpit":            AuthStatusOverlayRequired,
		"coingecko":          AuthStatusOverlayRequired,
		"contentful":         AuthStatusOverlayRequired,
		"convertkit":         AuthStatusComplete,
		"copper":             AuthStatusOverlayRequired,
		"customer-io":        AuthStatusPresentIncomplete,
		"databricks":         AuthStatusOverlayRequired,
		"deepl":              AuthStatusComplete,
		"dhl":                AuthStatusOverlayRequired,
		"discord":            AuthStatusComplete,
		"discourse":          AuthStatusOverlayRequired,
		"drift":              AuthStatusOverlayRequired,
		"dropbox":            AuthStatusOverlayRequired,
		"elastic":            AuthStatusComplete,
		"erpnext":            AuthStatusOverlayRequired,
		"eventbrite":         AuthStatusOverlayRequired,
		"facebook":           AuthStatusOverlayRequired,
		"facebook-lead-ads":  AuthStatusOverlayRequired,
		"figma":              AuthStatusComplete,
		"filemaker":          AuthStatusOverlayRequired,
		"formio":             AuthStatusOverlayRequired,
		"formstack":          AuthStatusOverlayRequired,
		"freshdesk":          AuthStatusOverlayRequired,
		"freshservice":       AuthStatusOverlayRequired,
		"freshworks-crm":     AuthStatusOverlayRequired,
		"getresponse":        AuthStatusOverlayRequired,
		"ghost":              AuthStatusOverlayRequired,
		"github":             AuthStatusOverlayRequired,
		"gitlab":             AuthStatusPresentIncomplete,
		"gmail":              AuthStatusOverlayRequired,
		"gong":               AuthStatusOverlayRequired,
		"google-calendar":    AuthStatusOverlayRequired,
		"google-drive":       AuthStatusOverlayRequired,
		"google-sheets":      AuthStatusOverlayRequired,
		"grafana":            AuthStatusComplete,
		"grist":              AuthStatusOverlayRequired,
		"gumroad":            AuthStatusOverlayRequired,
		"hackernews":         AuthStatusIntentionallyAnonymous,
		"harvest":            AuthStatusOverlayRequired,
		"help-scout":         AuthStatusOverlayRequired,
		"highlevel":          AuthStatusComplete,
		"hubspot":            AuthStatusOverlayRequired,
		"intercom":           AuthStatusPresentIncomplete,
		"invoice-ninja":      AuthStatusOverlayRequired,
		"iterable":           AuthStatusOverlayRequired,
		"jenkins":            AuthStatusOverlayRequired,
		"jira-cloud":         AuthStatusPresentIncomplete,
		"jotform":            AuthStatusOverlayRequired,
		"keap":               AuthStatusOverlayRequired,
		"kobotoolbox":        AuthStatusComplete,
		"line":               AuthStatusComplete,
		"linear":             AuthStatusOverlayRequired,
		"linkedin":           AuthStatusOverlayRequired,
		"magento":            AuthStatusOverlayRequired,
		"mailchimp":          AuthStatusOverlayRequired,
		"mailerlite":         AuthStatusOverlayRequired,
		"mailgun":            AuthStatusPresentIncomplete,
		"mailjet":            AuthStatusOverlayRequired,
		"marketstack":        AuthStatusComplete,
		"matrix":             AuthStatusComplete,
		"mattermost":         AuthStatusComplete,
		"mautic":             AuthStatusOverlayRequired,
		"messagebird":        AuthStatusOverlayRequired,
		"microsoft-graph":    AuthStatusOverlayRequired,
		"monday-com":         AuthStatusOverlayRequired,
		"monica-crm":         AuthStatusOverlayRequired,
		"nasa":               AuthStatusOverlayRequired,
		"netlify":            AuthStatusComplete,
		"nocodb":             AuthStatusPresentIncomplete,
		"notion":             AuthStatusPresentIncomplete,
		"odoo":               AuthStatusOverlayRequired,
		"okta":               AuthStatusPresentIncomplete,
		"onesimpleapi":       AuthStatusOverlayRequired,
		"onfleet":            AuthStatusOverlayRequired,
		"openthesaurus":      AuthStatusIntentionallyAnonymous,
		"openweathermap":     AuthStatusOverlayRequired,
		"paddle":             AuthStatusComplete,
		"pagerduty":          AuthStatusPresentIncomplete,
		"paypal":             AuthStatusComplete,
		"pipedrive":          AuthStatusPresentIncomplete,
		"posthog":            AuthStatusPresentIncomplete,
		"postmark":           AuthStatusOverlayRequired,
		"quickbooks":         AuthStatusOverlayRequired,
		"quickchart":         AuthStatusIntentionallyAnonymous,
		"reddit":             AuthStatusOverlayRequired,
		"rocket-chat":        AuthStatusPresentIncomplete,
		"salesforce":         AuthStatusOverlayRequired,
		"salesmate":          AuthStatusOverlayRequired,
		"seatable":           AuthStatusComplete,
		"sendgrid":           AuthStatusComplete,
		"sendy":              AuthStatusOverlayRequired,
		"sentry":             AuthStatusOverlayRequired,
		"servicenow":         AuthStatusOverlayRequired,
		"shopify":            AuthStatusOverlayRequired,
		"slack":              AuthStatusPresentIncomplete,
		"snowflake":          AuthStatusComplete,
		"splunk":             AuthStatusOverlayRequired,
		"stackby":            AuthStatusOverlayRequired,
		"stripe":             AuthStatusComplete,
		"supabase":           AuthStatusPresentIncomplete,
		"surveymonkey":       AuthStatusOverlayRequired,
		"telegram":           AuthStatusOverlayRequired,
		"timesaved":          AuthStatusIntentionallyAnonymous,
		"todoist":            AuthStatusOverlayRequired,
		"trello":             AuthStatusPresentIncomplete,
		"twake":              AuthStatusOverlayRequired,
		"twilio":             AuthStatusComplete,
		"twist":              AuthStatusOverlayRequired,
		"twitter":            AuthStatusComplete,
		"typeform":           AuthStatusOverlayRequired,
		"unleashed-software": AuthStatusOverlayRequired,
		"uptimerobot":        AuthStatusComplete,
		"vero":               AuthStatusOverlayRequired,
		"webflow":            AuthStatusOverlayRequired,
		"whatsapp":           AuthStatusOverlayRequired,
		"wise":               AuthStatusOverlayRequired,
		"woocommerce":        AuthStatusOverlayRequired,
		"workable":           AuthStatusOverlayRequired,
		"xero":               AuthStatusComplete,
		"zendesk":            AuthStatusPresentIncomplete,
		"zoom":               AuthStatusPresentIncomplete,
	}
	if !reflect.DeepEqual(statusByID, wantStatuses) {
		t.Fatalf("report statuses = %#v, want %#v", statusByID, wantStatuses)
	}

	slack, ok := report.FindProvider("Slack")
	if !ok {
		t.Fatalf("FindProvider(Slack) did not match")
	}
	if !reflect.DeepEqual(slack.OverlayIDs, []string{"slack-web-api-auth-review"}) {
		t.Fatalf("slack overlay ids = %#v", slack.OverlayIDs)
	}

	parseableInvalidOverlays := map[string]string{
		"asana":      "asana-openapi-v1-auth-review",
		"bitbucket":  "bitbucket-cloud-swagger-v2-auth-review",
		"box":        "box-platform-openapi-v3-auth-review",
		"chargebee":  "chargebee-api-v2-pc-v2-auth-review",
		"circleci":   "circleci-api-v2-auth-review",
		"cloudflare": "cloudflare-api-auth-review",
		"intercom":   "intercom-api-v2-15-auth-review",
		"jira-cloud": "jira-cloud-platform-openapi-v3-auth-review",
		"mailgun":    "mailgun-send-api-auth-review",
		"notion":     "notion-api-openapi-auth-review",
		"okta":       "okta-management-minimal-openapi-auth-review",
		"pagerduty":  "pagerduty-rest-openapi-v3-auth-review",
		"pipedrive":  "pipedrive-api-v2-openapi-auth-review",
		"trello":     "trello-cloud-openapi-v3-auth-review",
		"zendesk":    "zendesk-sunshine-auth-review",
	}
	for providerID, overlayID := range parseableInvalidOverlays {
		row, ok := report.FindProvider(providerID)
		if !ok {
			t.Fatalf("FindProvider(%s) did not match", providerID)
		}
		if !containsString(row.OverlayIDs, overlayID) {
			t.Fatalf("%s overlay ids = %#v, want %q", providerID, row.OverlayIDs, overlayID)
		}
	}
}

func TestSecurityMetadataReturnsCopies(t *testing.T) {
	overlays := BuiltInSecurityOverlays()
	overlays[0].SecuritySchemes[0].Name = "mutated"
	overlays[0].RootSecurity[0].Scheme = "mutated"
	overlays[0].SourceRefs[0] = "https://example.com/mutated"

	fresh := BuiltInSecurityOverlays()
	if fresh[0].SecuritySchemes[0].Name == "mutated" {
		t.Fatalf("BuiltInSecurityOverlays leaked security scheme slice")
	}
	if fresh[0].RootSecurity[0].Scheme == "mutated" {
		t.Fatalf("BuiltInSecurityOverlays leaked root security slice")
	}
	if fresh[0].SourceRefs[0] == "https://example.com/mutated" {
		t.Fatalf("BuiltInSecurityOverlays leaked source refs slice")
	}

	classifications := BuiltInSecurityClassifications()
	classifications[0].SourceRefs[0] = "https://example.com/mutated"
	if BuiltInSecurityClassifications()[0].SourceRefs[0] == "https://example.com/mutated" {
		t.Fatalf("BuiltInSecurityClassifications leaked source refs slice")
	}
}

func bearerScheme(name string) SecurityScheme {
	return SecurityScheme{
		Name:   name,
		Type:   SecuritySchemeHTTP,
		Scheme: "bearer",
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
