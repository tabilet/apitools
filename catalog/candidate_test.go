package catalog

import (
	"reflect"
	"testing"
)

func TestBuiltInCandidatesValidate(t *testing.T) {
	candidates := BuiltInCandidates()
	if err := ValidateCandidates(candidates); err != nil {
		t.Fatalf("ValidateCandidates() error = %v", err)
	}
	if got, want := len(candidates), 41; got != want {
		t.Fatalf("len(BuiltInCandidates()) = %d, want %d", got, want)
	}
}

func TestBuiltInCandidateIDsAreDeterministic(t *testing.T) {
	got := CandidateIDs(BuiltInCandidates())
	want := []string{
		"airtable",
		"asana",
		"aws-lambda",
		"aws-s3",
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
		"linear",
		"mailchimp",
		"microsoft-graph",
		"monday-com",
		"notion",
		"okta",
		"openweathermap",
		"pagerduty",
		"paypal",
		"pipedrive",
		"quickbooks",
		"salesforce",
		"sendgrid",
		"servicenow",
		"shopify",
		"slack",
		"stripe",
		"telegram",
		"trello",
		"twilio",
		"typeform",
		"xero",
		"zendesk",
		"zoom",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("CandidateIDs() = %#v, want %#v", got, want)
	}
}

func TestFindBuiltInCandidateMatchesAliases(t *testing.T) {
	tests := []struct {
		key string
		id  string
	}{
		{key: "Jira", id: "jira-cloud"},
		{key: "google drive api", id: "google-drive"},
		{key: "Open Weather Map", id: "openweathermap"},
		{key: "office 365", id: "microsoft-graph"},
		{key: "click up", id: "clickup"},
		{key: "lambda", id: "aws-lambda"},
		{key: "s3", id: "aws-s3"},
	}
	for _, test := range tests {
		candidate, ok := FindBuiltInCandidate(test.key)
		if !ok {
			t.Fatalf("FindBuiltInCandidate(%q) did not match", test.key)
		}
		if candidate.ID != test.id {
			t.Fatalf("FindBuiltInCandidate(%q).ID = %q, want %q", test.key, candidate.ID, test.id)
		}
	}
}

func TestBuiltInCandidatesCaptureFixtureAndPriorityEvidence(t *testing.T) {
	for _, candidate := range BuiltInCandidates() {
		if candidate.LocalOpenAPIFixture != "" && !candidate.HasEvidence(EvidenceTryN8nLocalFixture) {
			t.Fatalf("%s has local fixture but missing try-n8n fixture evidence", candidate.ID)
		}
		var foundN8n bool
		for _, evidence := range candidate.Evidence {
			if evidence.Source != EvidenceN8nNodeDirectory {
				continue
			}
			foundN8n = true
			if evidence.Use != EvidenceUsePriority {
				t.Fatalf("%s n8n evidence use = %q, want %q", candidate.ID, evidence.Use, EvidenceUsePriority)
			}
		}
		if !foundN8n {
			t.Fatalf("%s missing n8n priority evidence", candidate.ID)
		}
	}
}

func TestM6CandidatesAreFixtureFreeUntilSourceReview(t *testing.T) {
	for _, id := range []string{
		"asana",
		"aws-lambda",
		"aws-s3",
		"box",
		"calendly",
		"clickup",
		"discord",
		"dropbox",
		"github",
		"gitlab",
		"linear",
		"mailchimp",
		"google-calendar",
		"google-sheets",
		"microsoft-graph",
		"monday-com",
		"notion",
		"okta",
		"quickbooks",
		"salesforce",
		"sendgrid",
		"servicenow",
		"shopify",
		"stripe",
		"telegram",
		"twilio",
		"typeform",
		"xero",
		"zendesk",
		"zoom",
	} {
		candidate, ok := FindBuiltInCandidate(id)
		if !ok {
			t.Fatalf("missing candidate %s", id)
		}
		if candidate.LocalOpenAPIFixture != "" || candidate.HasEvidence(EvidenceTryN8nLocalFixture) {
			t.Fatalf("%s should not claim local fixture evidence before source review: %#v", id, candidate)
		}
		if !candidate.HasEvidence(EvidenceN8nNodeDirectory) {
			t.Fatalf("%s missing priority-only n8n evidence", id)
		}
	}
}

func TestBuiltInCandidateClassificationValues(t *testing.T) {
	for _, candidate := range BuiltInCandidates() {
		if candidate.OfficialOpenAPIStatus != SpecStatusNeedsVerification {
			t.Fatalf("%s official OpenAPI status = %q, want %q", candidate.ID, candidate.OfficialOpenAPIStatus, SpecStatusNeedsVerification)
		}
		if candidate.AuthSecurityReview != AuthSecurityNotReviewed {
			t.Fatalf("%s auth review = %q, want %q", candidate.ID, candidate.AuthSecurityReview, AuthSecurityNotReviewed)
		}
	}
	for _, id := range []string{"aws-lambda", "aws-s3", "gmail", "google-calendar", "google-drive", "google-sheets"} {
		candidate, ok := FindBuiltInCandidate(id)
		if !ok {
			t.Fatalf("missing %s", id)
		}
		if candidate.OfficialMachineSpecStatus != SpecStatusNeedsVerification {
			t.Fatalf("%s machine spec status = %q, want %q", id, candidate.OfficialMachineSpecStatus, SpecStatusNeedsVerification)
		}
		wantKind := "google-discovery"
		if id == "aws-lambda" || id == "aws-s3" {
			wantKind = string(SpecKindSmithyJSON)
		}
		if candidate.OfficialMachineSpecKind != wantKind {
			t.Fatalf("%s machine spec kind = %q, want %q", id, candidate.OfficialMachineSpecKind, wantKind)
		}
		if candidate.UserOpenAPINeed != UserOpenAPINeedLikely {
			t.Fatalf("%s user OpenAPI need = %q, want %q", id, candidate.UserOpenAPINeed, UserOpenAPINeedLikely)
		}
	}
}

func TestBuiltInCandidatesReturnCopies(t *testing.T) {
	candidates := BuiltInCandidates()
	candidates[0].Aliases[0] = "mutated"
	candidates[0].Evidence[0].Ref = "mutated"

	fresh := BuiltInCandidates()
	if fresh[0].Aliases[0] == "mutated" {
		t.Fatalf("BuiltInCandidates leaked alias slice")
	}
	if fresh[0].Evidence[0].Ref == "mutated" {
		t.Fatalf("BuiltInCandidates leaked evidence slice")
	}
}

func TestValidateCandidatesRejectsDuplicateLookupKeys(t *testing.T) {
	candidates := []Candidate{
		{
			ID:                        "one",
			DisplayName:               "One",
			Aliases:                   []string{"shared"},
			OfficialOpenAPIStatus:     SpecStatusUnknown,
			OfficialMachineSpecStatus: SpecStatusUnknown,
			UserOpenAPINeed:           UserOpenAPINeedUnknown,
			AuthSecurityReview:        AuthSecurityUnknown,
		},
		{
			ID:                        "two",
			DisplayName:               "Two",
			Aliases:                   []string{"shared"},
			OfficialOpenAPIStatus:     SpecStatusUnknown,
			OfficialMachineSpecStatus: SpecStatusUnknown,
			UserOpenAPINeed:           UserOpenAPINeedUnknown,
			AuthSecurityReview:        AuthSecurityUnknown,
		},
	}
	if err := ValidateCandidates(candidates); err == nil {
		t.Fatalf("ValidateCandidates() expected duplicate alias error")
	}
}

func TestValidateCandidatesRejectsMismatchedFixtureEvidence(t *testing.T) {
	candidate := BuiltInCandidates()[0]
	candidate.LocalOpenAPIFixture = "../try-n8n/reducibility/specs/other.json"
	if err := ValidateCandidates([]Candidate{candidate}); err == nil {
		t.Fatalf("ValidateCandidates() expected mismatched fixture evidence error")
	}
}
