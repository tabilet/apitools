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
	if got, want := len(catalog.ListProviders()), 117; got != want {
		t.Fatalf("len(ListProviders()) = %d, want %d", got, want)
	}
}

func TestBuiltInProviderIDsAreDeterministic(t *testing.T) {
	got := ProviderIDs(BuiltInProviders())
	want := []string{
		"action-network",
		"activecampaign",
		"acuity-scheduling",
		"adalo",
		"affinity",
		"agile-crm",
		"airtable",
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
		"calendly",
		"chargebee",
		"circleci",
		"clearbit",
		"clickup",
		"clockify",
		"cloudflare",
		"coda",
		"coingecko",
		"contentful",
		"convertkit",
		"copper",
		"customer-io",
		"databricks",
		"deepl",
		"discord",
		"discourse",
		"drift",
		"dropbox",
		"elastic",
		"eventbrite",
		"figma",
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
		"hackernews",
		"harvest",
		"help-scout",
		"highlevel",
		"hubspot",
		"intercom",
		"iterable",
		"jenkins",
		"jira-cloud",
		"keap",
		"linear",
		"mailchimp",
		"mailerlite",
		"mailgun",
		"mailjet",
		"marketstack",
		"mautic",
		"microsoft-graph",
		"monday-com",
		"monica-crm",
		"nasa",
		"netlify",
		"nocodb",
		"notion",
		"okta",
		"onesimpleapi",
		"openthesaurus",
		"openweathermap",
		"pagerduty",
		"paypal",
		"pipedrive",
		"posthog",
		"postmark",
		"quickbooks",
		"quickchart",
		"reddit",
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
		"stripe",
		"supabase",
		"telegram",
		"todoist",
		"trello",
		"twilio",
		"typeform",
		"uptimerobot",
		"vero",
		"webflow",
		"xero",
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
		{key: "highlevel api", id: "highlevel"},
		{key: "freshworks crm api", id: "freshworks-crm"},
		{key: "mailer lite", id: "mailerlite"},
		{key: "google drive api", id: "google-drive"},
		{key: "Open Weather Map", id: "openweathermap"},
		{key: "ms graph", id: "microsoft-graph"},
		{key: "activecampaign api", id: "activecampaign"},
		{key: "action network api", id: "action-network"},
		{key: "adalo collections api", id: "adalo"},
		{key: "affinity crm api", id: "affinity"},
		{key: "agile crm", id: "agile-crm"},
		{key: "acuity scheduling api", id: "acuity-scheduling"},
		{key: "asana api", id: "asana"},
		{key: "box api", id: "box"},
		{key: "bamboohr api", id: "bamboohr"},
		{key: "bannerbear image api", id: "bannerbear"},
		{key: "baserow database api", id: "baserow"},
		{key: "beeminder api", id: "beeminder"},
		{key: "amazon lambda", id: "aws-lambda"},
		{key: "amazon s3", id: "aws-s3"},
		{key: "amazon sns", id: "aws-sns"},
		{key: "bitbucket cloud", id: "bitbucket"},
		{key: "bitly api", id: "bitly"},
		{key: "brandfetch brand api", id: "brandfetch"},
		{key: "calendly api", id: "calendly"},
		{key: "sendinblue", id: "brevo"},
		{key: "chargebee api", id: "chargebee"},
		{key: "circleci api", id: "circleci"},
		{key: "clearbit prospector", id: "clearbit"},
		{key: "cloudflare api", id: "cloudflare"},
		{key: "clickup api", id: "clickup"},
		{key: "clockify api", id: "clockify"},
		{key: "coda docs", id: "coda"},
		{key: "coingecko market data", id: "coingecko"},
		{key: "contentful api", id: "contentful"},
		{key: "convertkit api", id: "convertkit"},
		{key: "copper crm", id: "copper"},
		{key: "customer.io api", id: "customer-io"},
		{key: "databricks api", id: "databricks"},
		{key: "deepl translate", id: "deepl"},
		{key: "discord api", id: "discord"},
		{key: "discourse forum api", id: "discourse"},
		{key: "dropbox api", id: "dropbox"},
		{key: "elasticsearch", id: "elastic"},
		{key: "eventbrite api", id: "eventbrite"},
		{key: "figma rest api", id: "figma"},
		{key: "freshdesk api", id: "freshdesk"},
		{key: "freshservice service desk api", id: "freshservice"},
		{key: "ghost admin api", id: "ghost"},
		{key: "github rest api", id: "github"},
		{key: "gitlab rest api", id: "gitlab"},
		{key: "gong public api", id: "gong"},
		{key: "google calendar api", id: "google-calendar"},
		{key: "google sheets api", id: "google-sheets"},
		{key: "grafana api", id: "grafana"},
		{key: "getgrist", id: "grist"},
		{key: "hacker news", id: "hackernews"},
		{key: "harvestapp", id: "harvest"},
		{key: "helpscout api", id: "help-scout"},
		{key: "autopilot api", id: "autopilot"},
		{key: "intercom api", id: "intercom"},
		{key: "iterable api", id: "iterable"},
		{key: "jenkins api", id: "jenkins"},
		{key: "keap api", id: "keap"},
		{key: "linear graphql", id: "linear"},
		{key: "mailchimp marketing", id: "mailchimp"},
		{key: "mailgun api", id: "mailgun"},
		{key: "mailjet api", id: "mailjet"},
		{key: "marketstack api", id: "marketstack"},
		{key: "mautic marketing", id: "mautic"},
		{key: "monday graphql", id: "monday-com"},
		{key: "monica crm", id: "monica-crm"},
		{key: "nasa api", id: "nasa"},
		{key: "netlify api", id: "netlify"},
		{key: "noco db", id: "nocodb"},
		{key: "notion api", id: "notion"},
		{key: "okta management", id: "okta"},
		{key: "onesimpleapi api", id: "onesimpleapi"},
		{key: "openthesaurus api", id: "openthesaurus"},
		{key: "paypal checkout", id: "paypal"},
		{key: "pipedrive crm", id: "pipedrive"},
		{key: "posthog api", id: "posthog"},
		{key: "postmark api", id: "postmark"},
		{key: "intuit quickbooks", id: "quickbooks"},
		{key: "quickchart api", id: "quickchart"},
		{key: "reddit api", id: "reddit"},
		{key: "salesforce rest api", id: "salesforce"},
		{key: "salesmate crm", id: "salesmate"},
		{key: "sea table", id: "seatable"},
		{key: "twilio sendgrid", id: "sendgrid"},
		{key: "sendy api", id: "sendy"},
		{key: "sentry api", id: "sentry"},
		{key: "servicenow rest", id: "servicenow"},
		{key: "shopify admin api", id: "shopify"},
		{key: "stripe api", id: "stripe"},
		{key: "snowflake api", id: "snowflake"},
		{key: "splunk api", id: "splunk"},
		{key: "supabase api", id: "supabase"},
		{key: "telegram bot api", id: "telegram"},
		{key: "todoist api", id: "todoist"},
		{key: "twilio sms", id: "twilio"},
		{key: "typeform api", id: "typeform"},
		{key: "uptime robot api", id: "uptimerobot"},
		{key: "vero track", id: "vero"},
		{key: "webflow api", id: "webflow"},
		{key: "xero accounting", id: "xero"},
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
		{id: "action-network", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "adalo", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "affinity", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "agile-crm", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "acuity-scheduling", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "activecampaign", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "autopilot", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "aws-lambda", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityKnown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindSmithyJSON},
		{id: "aws-s3", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityKnown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindSmithyJSON},
		{id: "aws-sns", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityKnown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindSmithyJSON},
		{id: "bitbucket", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "bitly", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "box", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "brevo", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "airtable", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "bamboohr", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "bannerbear", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "baserow", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "beeminder", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "brandfetch", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "calendly", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "chargebee", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "circleci", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "clearbit", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "cloudflare", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "clickup", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "clockify", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "coda", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "coingecko", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "contentful", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "convertkit", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "copper", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "customer-io", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "databricks", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "deepl", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "discord", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedPossible, specKind: SpecKindOpenAPI},
		{id: "discourse", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "drift", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "dropbox", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityKnown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindDropboxStone},
		{id: "elastic", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "eventbrite", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "figma", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "freshdesk", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "freshservice", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "freshworks-crm", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "getresponse", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "ghost", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "github", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "gitlab", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedPossible, specKind: SpecKindOpenAPI},
		{id: "gmail", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityKnown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindGoogleDiscovery},
		{id: "gong", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "google-calendar", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityKnown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindGoogleDiscovery},
		{id: "google-drive", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityKnown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindGoogleDiscovery},
		{id: "google-sheets", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityKnown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindGoogleDiscovery},
		{id: "grafana", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "grist", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "hackernews", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "harvest", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "help-scout", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "highlevel", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "hubspot", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPIIndex},
		{id: "intercom", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "iterable", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "jenkins", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "jira-cloud", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "keap", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "linear", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "mailchimp", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "mailerlite", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "mailgun", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "mailjet", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "marketstack", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "mautic", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "microsoft-graph", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "monday-com", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "monica-crm", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "nasa", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "netlify", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "nocodb", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "notion", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "okta", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "onesimpleapi", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "openthesaurus", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "openweathermap", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "paypal", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedPossible, specKind: SpecKindOpenAPI},
		{id: "pipedrive", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "posthog", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "postmark", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "quickbooks", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "quickchart", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "reddit", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "salesforce", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "salesmate", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "seatable", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "sendgrid", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedPossible, specKind: SpecKindOpenAPI},
		{id: "sendy", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "sentry", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "servicenow", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "shopify", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "pagerduty", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "slack", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedPossible, specKind: SpecKindOpenAPI},
		{id: "snowflake", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "splunk", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "stripe", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "supabase", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "telegram", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "todoist", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "trello", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "twilio", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "typeform", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "uptimerobot", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "vero", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "webflow", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "xero", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedPossible, specKind: SpecKindOpenAPI},
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

func TestSpecReferenceProtocolClassification(t *testing.T) {
	tests := []struct {
		name    string
		ref     SpecReference
		want    SpecProtocol
		version string
	}{
		{name: "openapi31", ref: SpecReference{Kind: SpecKindOpenAPI, Version: "3.1.0"}, want: SpecProtocolOpenAPI, version: "3.1"},
		{name: "openapi30", ref: SpecReference{Kind: SpecKindOpenAPI, SourceNote: "Official API OpenAPI v3.0 document."}, want: SpecProtocolOpenAPI, version: "3.0"},
		{name: "openapi3 family", ref: SpecReference{Kind: SpecKindOpenAPI, SourceNote: "Official API OpenAPI v3 specification."}, want: SpecProtocolOpenAPI, version: "3"},
		{name: "swagger", ref: SpecReference{ID: "gitlab-openapi-v2", Kind: SpecKindOpenAPI, Version: "Swagger 2.0 / GitLab API v4"}, want: SpecProtocolSwagger, version: "2.0"},
		{name: "product version not openapi version", ref: SpecReference{Kind: SpecKindOpenAPI, Version: "10.3.0"}, want: SpecProtocolOpenAPI},
		{name: "openapi id with product v2", ref: SpecReference{ID: "clickup-api-v2-openapi", Kind: SpecKindOpenAPI, Version: "2.0"}, want: SpecProtocolOpenAPI},
		{name: "smithy", ref: SpecReference{Kind: SpecKindSmithyJSON}, want: SpecProtocolSmithy},
		{name: "discovery", ref: SpecReference{Kind: SpecKindGoogleDiscovery}, want: SpecProtocolGoogleDiscovery},
		{name: "stone", ref: SpecReference{Kind: SpecKindDropboxStone}, want: SpecProtocolDropboxStone},
		{name: "human docs", ref: SpecReference{Kind: SpecKindHumanDocs}, want: SpecProtocolHumanDocs},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := test.ref.ProtocolClassification()
			if got.Protocol != test.want || got.Version != test.version {
				t.Fatalf("ProtocolClassification() = %#v, want protocol=%q version=%q", got, test.want, test.version)
			}
		})
	}
}

func TestBuiltInRefreshableSpecReferences(t *testing.T) {
	rows := BuiltInRefreshableSpecReferences([]CatalogSpecArtifact{
		{ProviderID: "slack", SpecRefID: "slack-web-openapi-v2", Path: "openapi/slack-web-openapi-v2.json"},
	})
	if len(rows) == 0 {
		t.Fatal("BuiltInRefreshableSpecReferences() returned no rows")
	}
	var foundSlack bool
	var foundAWSS3 bool
	var foundDropbox bool
	var foundGmail bool
	var foundHumanDocs bool
	for _, row := range rows {
		if row.Kind == SpecKindHumanDocs {
			foundHumanDocs = true
		}
		if row.ProviderID == "slack" && row.SpecRefID == "slack-web-openapi-v2" {
			foundSlack = true
			if row.RegisteredArtifactPath != "openapi/slack-web-openapi-v2.json" {
				t.Fatalf("slack registered path = %q", row.RegisteredArtifactPath)
			}
			if row.Protocol != SpecProtocolSwagger || row.ProtocolVersion != "2.0" {
				t.Fatalf("slack protocol = %q %q, want swagger 2.0", row.Protocol, row.ProtocolVersion)
			}
		}
		if row.ProviderID == "aws-s3" && row.SpecRefID == "aws-s3-smithy-model" {
			foundAWSS3 = true
			if row.Protocol != SpecProtocolSmithy {
				t.Fatalf("aws-s3 protocol = %q, want smithy", row.Protocol)
			}
		}
		if row.ProviderID == "dropbox" {
			foundDropbox = true
			if row.Protocol != SpecProtocolDropboxStone {
				t.Fatalf("dropbox protocol = %q, want dropbox-stone", row.Protocol)
			}
		}
		if row.ProviderID == "gmail" {
			foundGmail = true
			if row.Protocol != SpecProtocolGoogleDiscovery {
				t.Fatalf("gmail protocol = %q, want google-discovery", row.Protocol)
			}
		}
	}
	if foundHumanDocs {
		t.Fatal("refreshable rows included human docs")
	}
	if !foundSlack {
		t.Fatal("refreshable rows missing slack OpenAPI reference")
	}
	if !foundAWSS3 || !foundDropbox || !foundGmail {
		t.Fatalf("refreshable rows missing protocol examples: aws=%t dropbox=%t gmail=%t", foundAWSS3, foundDropbox, foundGmail)
	}
	for i := 1; i < len(rows); i++ {
		if rows[i-1].ProviderID > rows[i].ProviderID || rows[i-1].ProviderID == rows[i].ProviderID && rows[i-1].SpecRefID > rows[i].SpecRefID {
			t.Fatalf("rows are not deterministic at %d: %#v before %#v", i, rows[i-1], rows[i])
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
