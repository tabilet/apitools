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
	}
	for _, test := range tests {
		if got := ClassifyAuthCompleteness(test.metadata); got != test.want {
			t.Fatalf("%s: ClassifyAuthCompleteness() = %q, want %q", test.name, got, test.want)
		}
	}
}

func TestBuiltInSecurityOverlaysValidate(t *testing.T) {
	overlays := BuiltInSecurityOverlays()
	if got, want := len(overlays), 21; got != want {
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
		"airtable",
		"asana",
		"box",
		"calendly",
		"clickup",
		"discord",
		"dropbox",
		"github",
		"gitlab",
		"gmail",
		"google-calendar",
		"google-drive",
		"google-sheets",
		"hubspot",
		"jira-cloud",
		"mailchimp",
		"microsoft-graph",
		"notion",
		"okta",
		"openweathermap",
		"pagerduty",
		"paypal",
		"quickbooks",
		"salesforce",
		"sendgrid",
		"servicenow",
		"shopify",
		"slack",
		"stripe",
		"trello",
		"twilio",
		"xero",
		"zendesk",
		"zoom",
	}
	if !reflect.DeepEqual(gotIDs, wantIDs) {
		t.Fatalf("report provider ids = %#v, want %#v", gotIDs, wantIDs)
	}
	wantStatuses := map[string]AuthCompletenessStatus{
		"airtable":        AuthStatusOverlayRequired,
		"asana":           AuthStatusComplete,
		"box":             AuthStatusComplete,
		"calendly":        AuthStatusOverlayRequired,
		"clickup":         AuthStatusPresentIncomplete,
		"discord":         AuthStatusComplete,
		"dropbox":         AuthStatusOverlayRequired,
		"github":          AuthStatusOverlayRequired,
		"gitlab":          AuthStatusPresentIncomplete,
		"gmail":           AuthStatusOverlayRequired,
		"google-calendar": AuthStatusOverlayRequired,
		"google-drive":    AuthStatusOverlayRequired,
		"google-sheets":   AuthStatusOverlayRequired,
		"hubspot":         AuthStatusOverlayRequired,
		"jira-cloud":      AuthStatusComplete,
		"mailchimp":       AuthStatusOverlayRequired,
		"microsoft-graph": AuthStatusOverlayRequired,
		"notion":          AuthStatusComplete,
		"okta":            AuthStatusComplete,
		"openweathermap":  AuthStatusOverlayRequired,
		"pagerduty":       AuthStatusComplete,
		"paypal":          AuthStatusComplete,
		"quickbooks":      AuthStatusOverlayRequired,
		"salesforce":      AuthStatusOverlayRequired,
		"sendgrid":        AuthStatusComplete,
		"servicenow":      AuthStatusOverlayRequired,
		"shopify":         AuthStatusOverlayRequired,
		"slack":           AuthStatusPresentIncomplete,
		"stripe":          AuthStatusComplete,
		"trello":          AuthStatusComplete,
		"twilio":          AuthStatusComplete,
		"xero":            AuthStatusComplete,
		"zendesk":         AuthStatusPresentIncomplete,
		"zoom":            AuthStatusPresentIncomplete,
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
