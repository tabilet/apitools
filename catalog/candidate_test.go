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
	if got, want := len(candidates), 256; got != want {
		t.Fatalf("len(BuiltInCandidates()) = %d, want %d", got, want)
	}
}

func TestBuiltInCandidateIDsAreDeterministic(t *testing.T) {
	got := CandidateIDs(BuiltInCandidates())
	want := []string{
		"action-network",
		"activecampaign",
		"acuity-scheduling",
		"adalo",
		"affinity",
		"agile-crm",
		"airtable",
		"airtop",
		"apitemplate-io",
		"asana",
		"autopilot",
		"aws-acm",
		"aws-cognito",
		"aws-comprehend",
		"aws-dynamodb",
		"aws-elb",
		"aws-elbv2",
		"aws-iam",
		"aws-lambda",
		"aws-rekognition",
		"aws-s3",
		"aws-ses",
		"aws-sns",
		"aws-sqs",
		"aws-textract",
		"aws-transcribe",
		"bamboohr",
		"bannerbear",
		"baserow",
		"beeminder",
		"bitbucket",
		"bitly",
		"bitwarden",
		"box",
		"brandfetch",
		"brevo",
		"bubble",
		"cal",
		"calendly",
		"chargebee",
		"circleci",
		"cisco-webex",
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
		"cortex",
		"currents",
		"customer-io",
		"databricks",
		"deepl",
		"demio",
		"dhl",
		"discord",
		"discourse",
		"disqus",
		"drift",
		"dropbox",
		"dropcontact",
		"egoi",
		"elastic",
		"emelia",
		"erpnext",
		"eventbrite",
		"facebook",
		"facebook-lead-ads",
		"figma",
		"filemaker",
		"flow",
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
		"google-admin",
		"google-analytics",
		"google-bigquery",
		"google-books",
		"google-business-profile",
		"google-calendar",
		"google-chat",
		"google-cloud-language",
		"google-cloud-storage",
		"google-docs",
		"google-drive",
		"google-firestore",
		"google-sheets",
		"google-slides",
		"google-tasks",
		"google-translate",
		"google-youtube",
		"gotify",
		"gotowebinar",
		"grafana",
		"grist",
		"gumroad",
		"hackernews",
		"halopsa",
		"harvest",
		"help-scout",
		"highlevel",
		"home-assistant",
		"hubspot",
		"humantic-ai",
		"hunter",
		"intercom",
		"invoice-ninja",
		"iterable",
		"jenkins",
		"jina-ai",
		"jira-cloud",
		"jotform",
		"keap",
		"kobotoolbox",
		"lemlist",
		"line",
		"linear",
		"lingvanex",
		"linkedin",
		"lonescale",
		"magento",
		"mailcheck",
		"mailchimp",
		"mailerlite",
		"mailgun",
		"mailjet",
		"mandrill",
		"marketstack",
		"matrix",
		"mattermost",
		"mautic",
		"medium",
		"messagebird",
		"metabase",
		"microsoft-graph",
		"mindee",
		"misp",
		"mistral-ai",
		"mocean",
		"monday-com",
		"monica-crm",
		"msg91",
		"nasa",
		"netlify",
		"netscaler",
		"nextcloud",
		"nocodb",
		"notion",
		"npm",
		"nvidia-dsx-air",
		"odoo",
		"okta",
		"onesimpleapi",
		"onfleet",
		"openai",
		"openthesaurus",
		"openweathermap",
		"orbit",
		"oura",
		"paddle",
		"pagerduty",
		"paypal",
		"peekalink",
		"perplexity",
		"phantombuster",
		"philips-hue",
		"pipedrive",
		"plivo",
		"postbin",
		"posthog",
		"postmark",
		"profitwell",
		"pushbullet",
		"pushcut",
		"pushover",
		"quickbase",
		"quickbooks",
		"quickchart",
		"raindrop",
		"reddit",
		"rocket-chat",
		"rundeck",
		"salesforce",
		"salesmate",
		"seatable",
		"securityscorecard",
		"segment",
		"sendgrid",
		"sendy",
		"sentry",
		"servicenow",
		"shopify",
		"signl4",
		"slack",
		"sms77",
		"snowflake",
		"splunk",
		"spotify",
		"stackby",
		"storyblok",
		"strapi",
		"strava",
		"stripe",
		"supabase",
		"surveymonkey",
		"syncromsp",
		"taiga",
		"tapfiliate",
		"telegram",
		"thehive",
		"thehive-project",
		"timesaved",
		"todoist",
		"toggl",
		"travis-ci",
		"trello",
		"twake",
		"twilio",
		"twist",
		"twitter",
		"typeform",
		"unleashed-software",
		"uplead",
		"uproc",
		"uptimerobot",
		"urlscan",
		"venafi",
		"vero",
		"vonage",
		"webflow",
		"wekan",
		"whatsapp",
		"wise",
		"woocommerce",
		"wordpress",
		"workable",
		"wufoo",
		"xero",
		"yourls",
		"zammad",
		"zendesk",
		"zoho",
		"zoom",
		"zulip",
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
		{key: "airtop browser", id: "airtop"},
		{key: "bubble data api", id: "bubble"},
		{key: "cal.com", id: "cal"},
		{key: "cockpit cms", id: "cockpit"},
		{key: "filemaker data api", id: "filemaker"},
		{key: "form io api", id: "formio"},
		{key: "formstack forms", id: "formstack"},
		{key: "high level crm", id: "highlevel"},
		{key: "freshworks crm", id: "freshworks-crm"},
		{key: "mailer lite", id: "mailerlite"},
		{key: "google drive api", id: "google-drive"},
		{key: "Open Weather Map", id: "openweathermap"},
		{key: "office 365", id: "microsoft-graph"},
		{key: "click up", id: "clickup"},
		{key: "active campaign", id: "activecampaign"},
		{key: "actionnetwork", id: "action-network"},
		{key: "adalo collections api", id: "adalo"},
		{key: "affinity crm api", id: "affinity"},
		{key: "agilecrm", id: "agile-crm"},
		{key: "acuity scheduling api", id: "acuity-scheduling"},
		{key: "acm", id: "aws-acm"},
		{key: "cognito-idp", id: "aws-cognito"},
		{key: "comprehend", id: "aws-comprehend"},
		{key: "dynamodb", id: "aws-dynamodb"},
		{key: "elb", id: "aws-elb"},
		{key: "elbv2", id: "aws-elbv2"},
		{key: "aws iam api", id: "aws-iam"},
		{key: "lambda", id: "aws-lambda"},
		{key: "rekognition", id: "aws-rekognition"},
		{key: "s3", id: "aws-s3"},
		{key: "ses", id: "aws-ses"},
		{key: "sns", id: "aws-sns"},
		{key: "sqs", id: "aws-sqs"},
		{key: "textract", id: "aws-textract"},
		{key: "transcribe", id: "aws-transcribe"},
		{key: "google admin sdk", id: "google-admin"},
		{key: "ga4 api", id: "google-analytics"},
		{key: "bigquery", id: "google-bigquery"},
		{key: "books api", id: "google-books"},
		{key: "google my business", id: "google-business-profile"},
		{key: "google chat api", id: "google-chat"},
		{key: "cloud natural language api", id: "google-cloud-language"},
		{key: "gcs", id: "google-cloud-storage"},
		{key: "google docs api", id: "google-docs"},
		{key: "firebase firestore", id: "google-firestore"},
		{key: "slides api", id: "google-slides"},
		{key: "tasks api", id: "google-tasks"},
		{key: "cloud translation api", id: "google-translate"},
		{key: "youtube data api", id: "google-youtube"},
		{key: "bamboo hr", id: "bamboohr"},
		{key: "bannerbear api", id: "bannerbear"},
		{key: "baserow database api", id: "baserow"},
		{key: "beeminder api", id: "beeminder"},
		{key: "bitbucket cloud", id: "bitbucket"},
		{key: "bitly api", id: "bitly"},
		{key: "brand api", id: "brandfetch"},
		{key: "sendinblue", id: "brevo"},
		{key: "chargebee api", id: "chargebee"},
		{key: "clearbit prospector", id: "clearbit"},
		{key: "clockify api", id: "clockify"},
		{key: "coda docs", id: "coda"},
		{key: "coin gecko", id: "coingecko"},
		{key: "contentful management api", id: "contentful"},
		{key: "kit email api", id: "convertkit"},
		{key: "copper crm", id: "copper"},
		{key: "customerio", id: "customer-io"},
		{key: "deepl translate", id: "deepl"},
		{key: "discourse forum api", id: "discourse"},
		{key: "circleci v2", id: "circleci"},
		{key: "cloudflare api", id: "cloudflare"},
		{key: "databricks rest api", id: "databricks"},
		{key: "dhl tracking api", id: "dhl"},
		{key: "elasticsearch api", id: "elastic"},
		{key: "erpnext rest api", id: "erpnext"},
		{key: "eventbrite api", id: "eventbrite"},
		{key: "facebook graph api", id: "facebook"},
		{key: "facebook leads api", id: "facebook-lead-ads"},
		{key: "freshdesk api", id: "freshdesk"},
		{key: "freshservice service desk api", id: "freshservice"},
		{key: "figma rest api", id: "figma"},
		{key: "ghost admin api", id: "ghost"},
		{key: "gong public api", id: "gong"},
		{key: "grafana http api", id: "grafana"},
		{key: "grist rest api", id: "grist"},
		{key: "gumroad api", id: "gumroad"},
		{key: "hn api", id: "hackernews"},
		{key: "harvestapp", id: "harvest"},
		{key: "helpscout api", id: "help-scout"},
		{key: "autopilot api", id: "autopilot"},
		{key: "intercom rest api", id: "intercom"},
		{key: "invoiceninja api", id: "invoice-ninja"},
		{key: "iterable api", id: "iterable"},
		{key: "jenkins remote api", id: "jenkins"},
		{key: "jot form", id: "jotform"},
		{key: "keap crm", id: "keap"},
		{key: "kobo toolbox", id: "kobotoolbox"},
		{key: "line messaging api", id: "line"},
		{key: "linkedin marketing api", id: "linkedin"},
		{key: "magento 2 api", id: "magento"},
		{key: "mailgun api", id: "mailgun"},
		{key: "mailjet api", id: "mailjet"},
		{key: "market stack", id: "marketstack"},
		{key: "matrix client server api", id: "matrix"},
		{key: "mattermost rest api", id: "mattermost"},
		{key: "mautic api", id: "mautic"},
		{key: "messagebird api", id: "messagebird"},
		{key: "monica crm api", id: "monica-crm"},
		{key: "api.nasa.gov", id: "nasa"},
		{key: "netlify api", id: "netlify"},
		{key: "nvidia air api", id: "nvidia-dsx-air"},
		{key: "nocodb api", id: "nocodb"},
		{key: "odoo xml-rpc api", id: "odoo"},
		{key: "onfleet api", id: "onfleet"},
		{key: "paddle billing api", id: "paddle"},
		{key: "post hog", id: "posthog"},
		{key: "postmark api", id: "postmark"},
		{key: "one simple api", id: "onesimpleapi"},
		{key: "open thesaurus", id: "openthesaurus"},
		{key: "quick chart", id: "quickchart"},
		{key: "reddit api", id: "reddit"},
		{key: "rocket.chat api", id: "rocket-chat"},
		{key: "salesmate crm", id: "salesmate"},
		{key: "sendy api", id: "sendy"},
		{key: "seatable api", id: "seatable"},
		{key: "sentry.io", id: "sentry"},
		{key: "snowflake sql api", id: "snowflake"},
		{key: "splunk rest api", id: "splunk"},
		{key: "stackby database", id: "stackby"},
		{key: "supabase management api", id: "supabase"},
		{key: "survey monkey", id: "surveymonkey"},
		{key: "track time saved", id: "timesaved"},
		{key: "todoist api", id: "todoist"},
		{key: "twake developers api", id: "twake"},
		{key: "twist v3 api", id: "twist"},
		{key: "x twitter api", id: "twitter"},
		{key: "unleashed software api", id: "unleashed-software"},
		{key: "uptimerobot api", id: "uptimerobot"},
		{key: "vero api", id: "vero"},
		{key: "webflow data api", id: "webflow"},
		{key: "whatsapp cloud api", id: "whatsapp"},
		{key: "wise platform api", id: "wise"},
		{key: "woocommerce rest api", id: "woocommerce"},
		{key: "workable recruiting api", id: "workable"},
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
		var foundOfficialDocsReview bool
		for _, evidence := range candidate.Evidence {
			switch evidence.Source {
			case EvidenceN8nNodeDirectory:
				foundN8n = true
				if evidence.Use != EvidenceUsePriority {
					t.Fatalf("%s n8n evidence use = %q, want %q", candidate.ID, evidence.Use, EvidenceUsePriority)
				}
			case EvidenceOfficialDocs:
				if evidence.Use == EvidenceUseReview {
					foundOfficialDocsReview = true
				}
			}
		}
		if candidate.ID == "nvidia-dsx-air" {
			if !foundOfficialDocsReview {
				t.Fatalf("%s missing official-docs review evidence", candidate.ID)
			}
			continue
		}
		if !foundN8n {
			t.Fatalf("%s missing n8n priority evidence", candidate.ID)
		}
	}
}

func TestM6CandidatesAreFixtureFreeUntilSourceReview(t *testing.T) {
	for _, id := range []string{
		"asana",
		"autopilot",
		"action-network",
		"adalo",
		"affinity",
		"agile-crm",
		"aws-acm",
		"aws-cognito",
		"aws-comprehend",
		"aws-dynamodb",
		"aws-elb",
		"aws-elbv2",
		"aws-iam",
		"aws-lambda",
		"aws-rekognition",
		"aws-s3",
		"aws-ses",
		"aws-sns",
		"aws-sqs",
		"aws-textract",
		"aws-transcribe",
		"acuity-scheduling",
		"bitbucket",
		"bitly",
		"box",
		"bannerbear",
		"baserow",
		"beeminder",
		"brandfetch",
		"brevo",
		"bubble",
		"cal",
		"calendly",
		"circleci",
		"clearbit",
		"clickup",
		"clockify",
		"cloudflare",
		"cockpit",
		"coda",
		"coingecko",
		"convertkit",
		"copper",
		"databricks",
		"deepl",
		"discord",
		"discourse",
		"drift",
		"dropbox",
		"elastic",
		"figma",
		"filemaker",
		"formio",
		"formstack",
		"freshservice",
		"freshworks-crm",
		"getresponse",
		"ghost",
		"github",
		"gitlab",
		"harvest",
		"help-scout",
		"highlevel",
		"iterable",
		"jenkins",
		"jotform",
		"keap",
		"kobotoolbox",
		"linear",
		"mailchimp",
		"mailerlite",
		"mailjet",
		"google-admin",
		"google-analytics",
		"google-bigquery",
		"google-books",
		"google-business-profile",
		"google-calendar",
		"google-chat",
		"google-cloud-language",
		"google-cloud-storage",
		"google-docs",
		"google-firestore",
		"google-sheets",
		"google-slides",
		"google-tasks",
		"google-translate",
		"google-youtube",
		"gong",
		"grafana",
		"grist",
		"hackernews",
		"microsoft-graph",
		"marketstack",
		"mautic",
		"monday-com",
		"monica-crm",
		"nasa",
		"netlify",
		"nocodb",
		"notion",
		"okta",
		"onesimpleapi",
		"openthesaurus",
		"posthog",
		"quickbooks",
		"quickchart",
		"reddit",
		"salesmate",
		"salesforce",
		"seatable",
		"sendgrid",
		"sendy",
		"sentry",
		"servicenow",
		"shopify",
		"stripe",
		"snowflake",
		"splunk",
		"stackby",
		"supabase",
		"surveymonkey",
		"telegram",
		"timesaved",
		"twilio",
		"typeform",
		"uptimerobot",
		"vero",
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
	for _, id := range []string{"aws-acm", "aws-cognito", "aws-comprehend", "aws-dynamodb", "aws-elb", "aws-elbv2", "aws-iam", "aws-lambda", "aws-rekognition", "aws-s3", "aws-ses", "aws-sns", "aws-sqs", "aws-textract", "aws-transcribe", "gmail", "google-admin", "google-analytics", "google-bigquery", "google-books", "google-business-profile", "google-calendar", "google-chat", "google-cloud-language", "google-cloud-storage", "google-docs", "google-drive", "google-firestore", "google-sheets", "google-slides", "google-tasks", "google-translate", "google-youtube"} {
		candidate, ok := FindBuiltInCandidate(id)
		if !ok {
			t.Fatalf("missing %s", id)
		}
		if candidate.OfficialMachineSpecStatus != SpecStatusNeedsVerification {
			t.Fatalf("%s machine spec status = %q, want %q", id, candidate.OfficialMachineSpecStatus, SpecStatusNeedsVerification)
		}
		wantKind := "google-discovery"
		if id == "aws-acm" || id == "aws-cognito" || id == "aws-comprehend" || id == "aws-dynamodb" || id == "aws-elb" || id == "aws-elbv2" || id == "aws-iam" || id == "aws-lambda" || id == "aws-rekognition" || id == "aws-s3" || id == "aws-ses" || id == "aws-sns" || id == "aws-sqs" || id == "aws-textract" || id == "aws-transcribe" {
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
