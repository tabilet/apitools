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
	if got, want := len(candidates), 83; got != want {
		t.Fatalf("len(BuiltInCandidates()) = %d, want %d", got, want)
	}
}

func TestBuiltInCandidateIDsAreDeterministic(t *testing.T) {
	got := CandidateIDs(BuiltInCandidates())
	want := []string{
		"activecampaign",
		"acuity-scheduling",
		"airtable",
		"asana",
		"aws-lambda",
		"aws-s3",
		"aws-sns",
		"bamboohr",
		"bannerbear",
		"baserow",
		"bitbucket",
		"bitly",
		"box",
		"brandfetch",
		"brevo",
		"calendly",
		"chargebee",
		"circleci",
		"clickup",
		"clockify",
		"cloudflare",
		"coda",
		"contentful",
		"convertkit",
		"customer-io",
		"databricks",
		"deepl",
		"discord",
		"dropbox",
		"elastic",
		"eventbrite",
		"figma",
		"freshdesk",
		"ghost",
		"github",
		"gitlab",
		"gmail",
		"google-calendar",
		"google-drive",
		"google-sheets",
		"grafana",
		"grist",
		"harvest",
		"help-scout",
		"hubspot",
		"intercom",
		"iterable",
		"jenkins",
		"jira-cloud",
		"linear",
		"mailchimp",
		"mailgun",
		"mailjet",
		"microsoft-graph",
		"monday-com",
		"netlify",
		"notion",
		"okta",
		"openweathermap",
		"pagerduty",
		"paypal",
		"pipedrive",
		"postmark",
		"quickbooks",
		"salesforce",
		"sendgrid",
		"sentry",
		"servicenow",
		"shopify",
		"slack",
		"snowflake",
		"splunk",
		"stripe",
		"supabase",
		"telegram",
		"todoist",
		"trello",
		"twilio",
		"typeform",
		"webflow",
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
		{key: "active campaign", id: "activecampaign"},
		{key: "acuity scheduling api", id: "acuity-scheduling"},
		{key: "lambda", id: "aws-lambda"},
		{key: "s3", id: "aws-s3"},
		{key: "sns", id: "aws-sns"},
		{key: "bamboo hr", id: "bamboohr"},
		{key: "bannerbear api", id: "bannerbear"},
		{key: "baserow database api", id: "baserow"},
		{key: "bitbucket cloud", id: "bitbucket"},
		{key: "bitly api", id: "bitly"},
		{key: "brand api", id: "brandfetch"},
		{key: "sendinblue", id: "brevo"},
		{key: "chargebee api", id: "chargebee"},
		{key: "clockify api", id: "clockify"},
		{key: "coda docs", id: "coda"},
		{key: "contentful management api", id: "contentful"},
		{key: "kit email api", id: "convertkit"},
		{key: "customerio", id: "customer-io"},
		{key: "deepl translate", id: "deepl"},
		{key: "circleci v2", id: "circleci"},
		{key: "cloudflare api", id: "cloudflare"},
		{key: "databricks rest api", id: "databricks"},
		{key: "elasticsearch api", id: "elastic"},
		{key: "eventbrite api", id: "eventbrite"},
		{key: "freshdesk api", id: "freshdesk"},
		{key: "figma rest api", id: "figma"},
		{key: "ghost admin api", id: "ghost"},
		{key: "grafana http api", id: "grafana"},
		{key: "grist rest api", id: "grist"},
		{key: "harvestapp", id: "harvest"},
		{key: "helpscout api", id: "help-scout"},
		{key: "intercom rest api", id: "intercom"},
		{key: "iterable api", id: "iterable"},
		{key: "jenkins remote api", id: "jenkins"},
		{key: "mailgun api", id: "mailgun"},
		{key: "mailjet api", id: "mailjet"},
		{key: "netlify api", id: "netlify"},
		{key: "postmark api", id: "postmark"},
		{key: "sentry.io", id: "sentry"},
		{key: "snowflake sql api", id: "snowflake"},
		{key: "splunk rest api", id: "splunk"},
		{key: "supabase management api", id: "supabase"},
		{key: "todoist api", id: "todoist"},
		{key: "webflow data api", id: "webflow"},
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
		"aws-sns",
		"acuity-scheduling",
		"bitbucket",
		"bitly",
		"box",
		"bannerbear",
		"baserow",
		"brandfetch",
		"brevo",
		"calendly",
		"circleci",
		"clickup",
		"clockify",
		"cloudflare",
		"coda",
		"convertkit",
		"databricks",
		"deepl",
		"discord",
		"dropbox",
		"elastic",
		"figma",
		"ghost",
		"github",
		"gitlab",
		"harvest",
		"help-scout",
		"iterable",
		"jenkins",
		"linear",
		"mailchimp",
		"mailjet",
		"google-calendar",
		"google-sheets",
		"grafana",
		"grist",
		"microsoft-graph",
		"monday-com",
		"netlify",
		"notion",
		"okta",
		"quickbooks",
		"salesforce",
		"sendgrid",
		"sentry",
		"servicenow",
		"shopify",
		"stripe",
		"snowflake",
		"splunk",
		"supabase",
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
	for _, id := range []string{"aws-lambda", "aws-s3", "aws-sns", "gmail", "google-calendar", "google-drive", "google-sheets"} {
		candidate, ok := FindBuiltInCandidate(id)
		if !ok {
			t.Fatalf("missing %s", id)
		}
		if candidate.OfficialMachineSpecStatus != SpecStatusNeedsVerification {
			t.Fatalf("%s machine spec status = %q, want %q", id, candidate.OfficialMachineSpecStatus, SpecStatusNeedsVerification)
		}
		wantKind := "google-discovery"
		if id == "aws-lambda" || id == "aws-s3" || id == "aws-sns" {
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
