package catalog

import (
	"reflect"
	"testing"
)

func TestBuiltInCatalogValidates(t *testing.T) {
	catalog := BuiltInCatalog()
	if err := catalog.Validate(); err != nil {
		t.Fatalf("BuiltInCatalog().Validate() error = %v", err)
	}
	if got, want := len(catalog.ListProviders()), 29; got != want {
		t.Fatalf("len(ListProviders()) = %d, want %d", got, want)
	}
}

func TestBuiltInProviderIDsAreDeterministic(t *testing.T) {
	got := ProviderIDs(BuiltInProviders())
	want := []string{
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
		"openweathermap",
		"pagerduty",
		"salesforce",
		"sendgrid",
		"shopify",
		"slack",
		"stripe",
		"trello",
		"twilio",
		"zendesk",
		"zoom",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ProviderIDs() = %#v, want %#v", got, want)
	}
}

func TestFindBuiltInProviderMatchesAliases(t *testing.T) {
	tests := []struct {
		key string
		id  string
	}{
		{key: "Jira", id: "jira-cloud"},
		{key: "google drive api", id: "google-drive"},
		{key: "Open Weather Map", id: "openweathermap"},
		{key: "ms graph", id: "microsoft-graph"},
		{key: "asana api", id: "asana"},
		{key: "box api", id: "box"},
		{key: "calendly api", id: "calendly"},
		{key: "clickup api", id: "clickup"},
		{key: "discord api", id: "discord"},
		{key: "dropbox api", id: "dropbox"},
		{key: "github rest api", id: "github"},
		{key: "gitlab rest api", id: "gitlab"},
		{key: "google calendar api", id: "google-calendar"},
		{key: "google sheets api", id: "google-sheets"},
		{key: "mailchimp marketing", id: "mailchimp"},
		{key: "notion api", id: "notion"},
		{key: "salesforce rest api", id: "salesforce"},
		{key: "twilio sendgrid", id: "sendgrid"},
		{key: "shopify admin api", id: "shopify"},
		{key: "stripe api", id: "stripe"},
		{key: "twilio sms", id: "twilio"},
		{key: "zoom meetings", id: "zoom"},
		{key: "zendesk support", id: "zendesk"},
	}
	for _, test := range tests {
		provider, ok := FindBuiltInProvider(test.key)
		if !ok {
			t.Fatalf("FindBuiltInProvider(%q) did not match", test.key)
		}
		if provider.ID != test.id {
			t.Fatalf("FindBuiltInProvider(%q).ID = %q, want %q", test.key, provider.ID, test.id)
		}
	}
}

func TestBuiltInProvidersArePromotedFromCandidates(t *testing.T) {
	candidateIDs := map[string]struct{}{}
	for _, candidate := range BuiltInCandidates() {
		candidateIDs[candidate.ID] = struct{}{}
	}
	for _, provider := range BuiltInProviders() {
		if provider.ReviewState != ProviderReviewedCatalogEntry {
			t.Fatalf("%s review state = %q, want %q", provider.ID, provider.ReviewState, ProviderReviewedCatalogEntry)
		}
		if provider.CandidateID != provider.ID {
			t.Fatalf("%s candidate id = %q, want provider id", provider.ID, provider.CandidateID)
		}
		if _, ok := candidateIDs[provider.CandidateID]; !ok {
			t.Fatalf("%s references missing candidate %q", provider.ID, provider.CandidateID)
		}
	}
}

func TestProviderSpecAvailabilityClassifications(t *testing.T) {
	tests := []struct {
		id              string
		openAPI         SpecAvailability
		machine         SpecAvailability
		userOpenAPINeed UserOpenAPINeed
		specKind        SpecKind
	}{
		{id: "asana", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "box", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "calendly", openAPI: SpecAvailabilityUnknown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "clickup", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "discord", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedPossible, specKind: SpecKindOpenAPI},
		{id: "dropbox", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityKnown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindDropboxStone},
		{id: "github", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "gitlab", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedPossible, specKind: SpecKindOpenAPI},
		{id: "gmail", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityKnown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindGoogleDiscovery},
		{id: "google-calendar", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityKnown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindGoogleDiscovery},
		{id: "google-drive", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityKnown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindGoogleDiscovery},
		{id: "google-sheets", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityKnown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindGoogleDiscovery},
		{id: "hubspot", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPIIndex},
		{id: "jira-cloud", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "mailchimp", openAPI: SpecAvailabilityNeedsVerification, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "microsoft-graph", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "notion", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "salesforce", openAPI: SpecAvailabilityNeedsVerification, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "sendgrid", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedPossible, specKind: SpecKindOpenAPI},
		{id: "shopify", openAPI: SpecAvailabilityUnknown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "pagerduty", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "slack", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedPossible, specKind: SpecKindOpenAPI},
		{id: "stripe", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "trello", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "twilio", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "zendesk", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedPossible, specKind: SpecKindOpenAPI},
		{id: "zoom", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedPossible, specKind: SpecKindOpenAPI},
	}
	for _, test := range tests {
		provider, ok := FindBuiltInProvider(test.id)
		if !ok {
			t.Fatalf("missing provider %s", test.id)
		}
		if provider.OfficialOpenAPIAvailability != test.openAPI {
			t.Fatalf("%s OpenAPI availability = %q, want %q", test.id, provider.OfficialOpenAPIAvailability, test.openAPI)
		}
		if provider.OfficialMachineSpecAvailability != test.machine {
			t.Fatalf("%s machine availability = %q, want %q", test.id, provider.OfficialMachineSpecAvailability, test.machine)
		}
		if provider.UserOpenAPINeed != test.userOpenAPINeed {
			t.Fatalf("%s user OpenAPI need = %q, want %q", test.id, provider.UserOpenAPINeed, test.userOpenAPINeed)
		}
		if refs := provider.SpecReferencesByKind(test.specKind); len(refs) == 0 {
			t.Fatalf("%s missing spec reference kind %q", test.id, test.specKind)
		}
	}
}

func TestProvidersWithUnknownOpenAPIHaveDocsReferences(t *testing.T) {
	for _, id := range []string{"airtable", "calendly", "openweathermap"} {
		provider, ok := FindBuiltInProvider(id)
		if !ok {
			t.Fatalf("missing provider %s", id)
		}
		if provider.OfficialOpenAPIAvailability != SpecAvailabilityUnknown {
			t.Fatalf("%s OpenAPI availability = %q, want %q", id, provider.OfficialOpenAPIAvailability, SpecAvailabilityUnknown)
		}
		if refs := provider.SpecReferencesByKind(SpecKindHumanDocs); len(refs) == 0 {
			t.Fatalf("%s missing human docs reference", id)
		}
	}
}

func TestBuiltInProvidersReturnCopies(t *testing.T) {
	providers := BuiltInProviders()
	providers[0].Aliases[0] = "mutated"
	providers[0].SpecReferences[0].URL = "https://example.com/mutated"
	providers[0].SourceHints[0] = "mutated"

	fresh := BuiltInProviders()
	if fresh[0].Aliases[0] == "mutated" {
		t.Fatalf("BuiltInProviders leaked alias slice")
	}
	if fresh[0].SpecReferences[0].URL == "https://example.com/mutated" {
		t.Fatalf("BuiltInProviders leaked spec reference slice")
	}
	if fresh[0].SourceHints[0] == "mutated" {
		t.Fatalf("BuiltInProviders leaked source hints slice")
	}
}

func TestValidateProvidersRejectsDuplicateLookupKeys(t *testing.T) {
	providers := []Provider{
		minimalProvider("one", "One", []string{"shared"}),
		minimalProvider("two", "Two", []string{"shared"}),
	}
	if err := ValidateProviders(providers); err == nil {
		t.Fatalf("ValidateProviders() expected duplicate alias error")
	}
}

func TestValidateProvidersRequiresOpenAPIRefForKnownOpenAPI(t *testing.T) {
	provider := minimalProvider("one", "One", nil)
	provider.OfficialOpenAPIAvailability = SpecAvailabilityKnown
	provider.SpecReferences = nil
	if err := ValidateProviders([]Provider{provider}); err == nil {
		t.Fatalf("ValidateProviders() expected missing OpenAPI ref error")
	}
}

func TestValidateProvidersRequiresMachineRefForKnownMachineSpec(t *testing.T) {
	provider := minimalProvider("one", "One", nil)
	provider.OfficialMachineSpecAvailability = SpecAvailabilityKnown
	provider.SpecReferences = []SpecReference{humanDocsRef("one-docs", "https://example.com/docs", "docs")}
	if err := ValidateProviders([]Provider{provider}); err == nil {
		t.Fatalf("ValidateProviders() expected missing machine spec ref error")
	}
}

func minimalProvider(id, displayName string, aliases []string) Provider {
	return Provider{
		ID:                              id,
		DisplayName:                     displayName,
		Aliases:                         aliases,
		ReviewState:                     ProviderReviewedCatalogEntry,
		CandidateID:                     id,
		OfficialOpenAPIAvailability:     SpecAvailabilityUnknown,
		OfficialMachineSpecAvailability: SpecAvailabilityUnknown,
		UserOpenAPINeed:                 UserOpenAPINeedUnknown,
		SpecReferences:                  []SpecReference{humanDocsRef(id+"-docs", "https://example.com/docs", "docs")},
	}
}
