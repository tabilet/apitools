package catalog

import "testing"

func TestResolveProviderUsesBuiltInMetadata(t *testing.T) {
	resolved, err := ResolveProvider(ResolveProviderOptions{ProviderKey: "slack"})
	if err != nil {
		t.Fatalf("ResolveProvider() error = %v", err)
	}
	if resolved.OpenAPI.Source != ResolutionSourceBuiltInSpecReference {
		t.Fatalf("OpenAPI source = %q, want %q", resolved.OpenAPI.Source, ResolutionSourceBuiltInSpecReference)
	}
	if resolved.OpenAPI.SpecRefID != "slack-web-openapi-v2" {
		t.Fatalf("OpenAPI spec ref = %q", resolved.OpenAPI.SpecRefID)
	}
	if resolved.Security.Source != ResolutionSourceBuiltInSecurityOverlay {
		t.Fatalf("Security source = %q, want %q", resolved.Security.Source, ResolutionSourceBuiltInSecurityOverlay)
	}
	if resolved.Security.SourceNote == "" || resolved.Security.SourceNote != resolved.CatalogSecurityOverlays[0].SourceNote {
		t.Fatalf("Security source note = %q, want overlay note %q", resolved.Security.SourceNote, resolved.CatalogSecurityOverlays[0].SourceNote)
	}
	if resolved.SecurityStatus != AuthStatusPresentIncomplete {
		t.Fatalf("SecurityStatus = %q, want %q", resolved.SecurityStatus, AuthStatusPresentIncomplete)
	}
}

func TestResolveProviderPrecedence(t *testing.T) {
	tests := []struct {
		name         string
		options      ResolveProviderOptions
		wantOpenAPI  ResolutionSource
		wantSecurity ResolutionSource
		wantStatus   AuthCompletenessStatus
	}{
		{
			name: "user openapi wins over built in spec and overlay",
			options: ResolveProviderOptions{
				ProviderKey: "slack",
				UserOpenAPI: "./openapi/slack.yaml",
			},
			wantOpenAPI:  ResolutionSourceUserOpenAPI,
			wantSecurity: ResolutionSourceNone,
			wantStatus:   AuthStatusUnknown,
		},
		{
			name: "user security overlay wins over built in overlay",
			options: ResolveProviderOptions{
				ProviderKey:         "slack",
				UserSecurityOverlay: "./security/slack-overlay.json",
			},
			wantOpenAPI:  ResolutionSourceBuiltInSpecReference,
			wantSecurity: ResolutionSourceUserSecurityOverlay,
			wantStatus:   AuthStatusUnknown,
		},
		{
			name: "project local openapi wins over built in spec and overlay",
			options: ResolveProviderOptions{
				ProviderKey:         "slack",
				ProjectLocalOpenAPI: "./openapi/slack.yaml",
			},
			wantOpenAPI:  ResolutionSourceProjectLocalOpenAPI,
			wantSecurity: ResolutionSourceNone,
			wantStatus:   AuthStatusUnknown,
		},
		{
			name: "user openapi wins over project local openapi",
			options: ResolveProviderOptions{
				ProviderKey:         "slack",
				UserOpenAPI:         "https://example.com/slack.yaml",
				ProjectLocalOpenAPI: "./openapi/slack.yaml",
			},
			wantOpenAPI:  ResolutionSourceUserOpenAPI,
			wantSecurity: ResolutionSourceNone,
			wantStatus:   AuthStatusUnknown,
		},
	}
	for _, test := range tests {
		resolved, err := ResolveProvider(test.options)
		if err != nil {
			t.Fatalf("%s: ResolveProvider() error = %v", test.name, err)
		}
		if resolved.OpenAPI.Source != test.wantOpenAPI {
			t.Fatalf("%s: OpenAPI source = %q, want %q", test.name, resolved.OpenAPI.Source, test.wantOpenAPI)
		}
		if resolved.Security.Source != test.wantSecurity {
			t.Fatalf("%s: Security source = %q, want %q", test.name, resolved.Security.Source, test.wantSecurity)
		}
		if resolved.SecurityStatus != test.wantStatus {
			t.Fatalf("%s: SecurityStatus = %q, want %q", test.name, resolved.SecurityStatus, test.wantStatus)
		}
	}
}

func TestResolveProviderRejectsUnknownProvider(t *testing.T) {
	if _, err := ResolveProvider(ResolveProviderOptions{ProviderKey: "missing"}); err == nil {
		t.Fatalf("ResolveProvider() expected unknown provider error")
	}
}

func TestResolveProviderRejectsUnsupportedURLScheme(t *testing.T) {
	_, err := ResolveProvider(ResolveProviderOptions{
		ProviderKey: "slack",
		UserOpenAPI: "ftp://example.com/slack.yaml",
	})
	if err == nil {
		t.Fatalf("ResolveProvider() expected unsupported URL scheme error")
	}
}

func TestSecurityReportRowsReturnCopies(t *testing.T) {
	report, err := BuiltInSecurityReport()
	if err != nil {
		t.Fatal(err)
	}
	rows := SecurityReportRows(report)
	rows[0].OverlayIDs = append(rows[0].OverlayIDs, "mutated")
	fresh := SecurityReportRows(report)
	for _, id := range fresh[0].OverlayIDs {
		if id == "mutated" {
			t.Fatalf("SecurityReportRows leaked overlay ids slice")
		}
	}
}
