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
	if got, want := len(catalog.ListProviders()), 302; got != want {
		t.Fatalf("len(ListProviders()) = %d, want %d", got, want)
	}
}

func TestBuiltInProviderIDsAreDeterministic(t *testing.T) {
	got := ProviderIDs(BuiltInProviders())
	want := []string{
		"action-network",
		"activecampaign",
		"acuity-scheduling",
		"acumatica",
		"adalo",
		"adobe-acrobat-sign",
		"adyen",
		"affinity",
		"aftership",
		"agile-crm",
		"aircall",
		"airtable",
		"airtop",
		"airwallex",
		"amplitude",
		"anthropic",
		"apitemplate-io",
		"apollo",
		"asana",
		"ashby",
		"attio",
		"auth0",
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
		"azure-cosmos-db",
		"azure-storage",
		"bamboohr",
		"bannerbear",
		"baserow",
		"beeminder",
		"bigcommerce",
		"bitbucket",
		"bitly",
		"bitwarden",
		"box",
		"brandfetch",
		"braze",
		"brevo",
		"bubble",
		"cal",
		"calendly",
		"canva",
		"chargebee",
		"checkr",
		"circleci",
		"cisco-meraki",
		"cisco-webex",
		"clearbit",
		"clickup",
		"clockify",
		"cloudflare",
		"cockpit",
		"coda",
		"coingecko",
		"confluence-cloud",
		"confluent-cloud",
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
		"docker-engine",
		"docker-hub",
		"docker-registry",
		"docusign",
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
		"fivetran",
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
		"google-contacts",
		"google-docs",
		"google-drive",
		"google-firebase-realtime-database",
		"google-firestore",
		"google-perspective",
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
		"klaviyo",
		"kobotoolbox",
		"kubernetes",
		"launchdarkly",
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
		"marketo",
		"marketstack",
		"matrix",
		"mattermost",
		"mautic",
		"medium",
		"messagebird",
		"metabase",
		"microsoft-entra",
		"microsoft-excel",
		"microsoft-graph",
		"microsoft-graph-security",
		"microsoft-onedrive",
		"microsoft-outlook",
		"microsoft-sharepoint",
		"microsoft-teams",
		"microsoft-todo",
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
		"netsuite",
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
		"oracle-fusion-cloud-applications",
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
		"postman",
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
		"sap-s4hana",
		"sap-successfactors",
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
		"workday",
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
		t.Fatalf("ProviderIDs() = %#v, want %#v", got, want)
	}
}

func TestFindBuiltInProviderMatchesAliases(t *testing.T) {
	tests := []struct {
		key string
		id  string
	}{
		{key: "Jira", id: "jira-cloud"},
		{key: "airtop browser", id: "airtop"},
		{key: "bubble api", id: "bubble"},
		{key: "cal.com", id: "cal"},
		{key: "cockpit cms", id: "cockpit"},
		{key: "filemaker api", id: "filemaker"},
		{key: "form.io", id: "formio"},
		{key: "formstack forms", id: "formstack"},
		{key: "highlevel api", id: "highlevel"},
		{key: "freshworks crm api", id: "freshworks-crm"},
		{key: "mailer lite", id: "mailerlite"},
		{key: "google drive api", id: "google-drive"},
		{key: "Open Weather Map", id: "openweathermap"},
		{key: "ms graph", id: "microsoft-graph"},
		{key: "azure blob storage", id: "azure-storage"},
		{key: "cosmos db api", id: "azure-cosmos-db"},
		{key: "entra id api", id: "microsoft-entra"},
		{key: "excel graph api", id: "microsoft-excel"},
		{key: "graph security api", id: "microsoft-graph-security"},
		{key: "one drive api", id: "microsoft-onedrive"},
		{key: "outlook mail api", id: "microsoft-outlook"},
		{key: "sharepoint sites api", id: "microsoft-sharepoint"},
		{key: "microsoft teams api", id: "microsoft-teams"},
		{key: "microsoft todo api", id: "microsoft-todo"},
		{key: "activecampaign api", id: "activecampaign"},
		{key: "action network api", id: "action-network"},
		{key: "adalo collections api", id: "adalo"},
		{key: "affinity crm api", id: "affinity"},
		{key: "agile crm", id: "agile-crm"},
		{key: "acuity scheduling api", id: "acuity-scheduling"},
		{key: "acumatica swagger", id: "acumatica"},
		{key: "adyen checkout api", id: "adyen"},
		{key: "asana api", id: "asana"},
		{key: "auth0 management api", id: "auth0"},
		{key: "box api", id: "box"},
		{key: "bamboohr api", id: "bamboohr"},
		{key: "bannerbear image api", id: "bannerbear"},
		{key: "baserow database api", id: "baserow"},
		{key: "beeminder api", id: "beeminder"},
		{key: "bigcommerce catalog api", id: "bigcommerce"},
		{key: "aws acm api", id: "aws-acm"},
		{key: "amazon cognito", id: "aws-cognito"},
		{key: "amazon comprehend", id: "aws-comprehend"},
		{key: "dynamodb", id: "aws-dynamodb"},
		{key: "classic load balancer", id: "aws-elb"},
		{key: "application load balancer", id: "aws-elbv2"},
		{key: "iam", id: "aws-iam"},
		{key: "amazon lambda", id: "aws-lambda"},
		{key: "amazon rekognition", id: "aws-rekognition"},
		{key: "amazon s3", id: "aws-s3"},
		{key: "amazon ses", id: "aws-ses"},
		{key: "amazon sns", id: "aws-sns"},
		{key: "amazon sqs", id: "aws-sqs"},
		{key: "amazon textract", id: "aws-textract"},
		{key: "amazon transcribe", id: "aws-transcribe"},
		{key: "bitbucket cloud", id: "bitbucket"},
		{key: "bitly api", id: "bitly"},
		{key: "brandfetch brand api", id: "brandfetch"},
		{key: "calendly api", id: "calendly"},
		{key: "sendinblue", id: "brevo"},
		{key: "chargebee api", id: "chargebee"},
		{key: "circleci api", id: "circleci"},
		{key: "meraki dashboard api", id: "cisco-meraki"},
		{key: "clearbit prospector", id: "clearbit"},
		{key: "cloudflare api", id: "cloudflare"},
		{key: "clickup api", id: "clickup"},
		{key: "clockify api", id: "clockify"},
		{key: "coda docs", id: "coda"},
		{key: "coingecko market data", id: "coingecko"},
		{key: "confluence cloud api", id: "confluence-cloud"},
		{key: "confluent cloud api", id: "confluent-cloud"},
		{key: "contentful api", id: "contentful"},
		{key: "convertkit api", id: "convertkit"},
		{key: "copper crm", id: "copper"},
		{key: "customer.io api", id: "customer-io"},
		{key: "databricks api", id: "databricks"},
		{key: "deepl translate", id: "deepl"},
		{key: "dhl shipment tracking api", id: "dhl"},
		{key: "discord api", id: "discord"},
		{key: "discourse forum api", id: "discourse"},
		{key: "docusign esignature", id: "docusign"},
		{key: "dropbox api", id: "dropbox"},
		{key: "elasticsearch", id: "elastic"},
		{key: "frappe api", id: "erpnext"},
		{key: "eventbrite api", id: "eventbrite"},
		{key: "facebook api", id: "facebook"},
		{key: "meta lead ads", id: "facebook-lead-ads"},
		{key: "figma rest api", id: "figma"},
		{key: "freshdesk api", id: "freshdesk"},
		{key: "freshservice service desk api", id: "freshservice"},
		{key: "ghost admin api", id: "ghost"},
		{key: "github rest api", id: "github"},
		{key: "gitlab rest api", id: "gitlab"},
		{key: "gong public api", id: "gong"},
		{key: "google admin sdk", id: "google-admin"},
		{key: "ga4 api", id: "google-analytics"},
		{key: "bigquery", id: "google-bigquery"},
		{key: "books api", id: "google-books"},
		{key: "google my business", id: "google-business-profile"},
		{key: "google calendar api", id: "google-calendar"},
		{key: "google chat api", id: "google-chat"},
		{key: "cloud natural language api", id: "google-cloud-language"},
		{key: "gcs", id: "google-cloud-storage"},
		{key: "people api", id: "google-contacts"},
		{key: "google docs api", id: "google-docs"},
		{key: "realtime database api", id: "google-firebase-realtime-database"},
		{key: "firebase firestore", id: "google-firestore"},
		{key: "perspective api", id: "google-perspective"},
		{key: "google sheets api", id: "google-sheets"},
		{key: "slides api", id: "google-slides"},
		{key: "tasks api", id: "google-tasks"},
		{key: "cloud translation api", id: "google-translate"},
		{key: "youtube data api", id: "google-youtube"},
		{key: "grafana api", id: "grafana"},
		{key: "getgrist", id: "grist"},
		{key: "gumroad api", id: "gumroad"},
		{key: "hacker news", id: "hackernews"},
		{key: "harvestapp", id: "harvest"},
		{key: "helpscout api", id: "help-scout"},
		{key: "autopilot api", id: "autopilot"},
		{key: "intercom api", id: "intercom"},
		{key: "invoice ninja api", id: "invoice-ninja"},
		{key: "iterable api", id: "iterable"},
		{key: "jenkins api", id: "jenkins"},
		{key: "jotform api", id: "jotform"},
		{key: "keap api", id: "keap"},
		{key: "kobotoolbox api", id: "kobotoolbox"},
		{key: "line bot api", id: "line"},
		{key: "linear graphql", id: "linear"},
		{key: "linkedin api", id: "linkedin"},
		{key: "adobe commerce api", id: "magento"},
		{key: "mailchimp marketing", id: "mailchimp"},
		{key: "mailgun api", id: "mailgun"},
		{key: "mailjet api", id: "mailjet"},
		{key: "marketstack api", id: "marketstack"},
		{key: "matrix api", id: "matrix"},
		{key: "mattermost api", id: "mattermost"},
		{key: "mautic marketing", id: "mautic"},
		{key: "bird messaging api", id: "messagebird"},
		{key: "monday graphql", id: "monday-com"},
		{key: "monica crm", id: "monica-crm"},
		{key: "nasa api", id: "nasa"},
		{key: "netlify api", id: "netlify"},
		{key: "suitetalk rest", id: "netsuite"},
		{key: "nvidia dsx air api", id: "nvidia-dsx-air"},
		{key: "noco db", id: "nocodb"},
		{key: "notion api", id: "notion"},
		{key: "odoo external api", id: "odoo"},
		{key: "okta management", id: "okta"},
		{key: "onesimpleapi api", id: "onesimpleapi"},
		{key: "onfleet api", id: "onfleet"},
		{key: "openthesaurus api", id: "openthesaurus"},
		{key: "oracle fusion rest api", id: "oracle-fusion-cloud-applications"},
		{key: "paddle api", id: "paddle"},
		{key: "paypal checkout", id: "paypal"},
		{key: "pipedrive crm", id: "pipedrive"},
		{key: "posthog api", id: "posthog"},
		{key: "postmark api", id: "postmark"},
		{key: "intuit quickbooks", id: "quickbooks"},
		{key: "quickchart api", id: "quickchart"},
		{key: "reddit api", id: "reddit"},
		{key: "rocketchat", id: "rocket-chat"},
		{key: "salesforce rest api", id: "salesforce"},
		{key: "salesmate crm", id: "salesmate"},
		{key: "sap s/4hana api", id: "sap-s4hana"},
		{key: "successfactors odata", id: "sap-successfactors"},
		{key: "sea table", id: "seatable"},
		{key: "twilio sendgrid", id: "sendgrid"},
		{key: "twilio segment", id: "segment"},
		{key: "sendy api", id: "sendy"},
		{key: "sentry api", id: "sentry"},
		{key: "servicenow rest", id: "servicenow"},
		{key: "shopify admin api", id: "shopify"},
		{key: "stripe api", id: "stripe"},
		{key: "strava v3 api", id: "strava"},
		{key: "snowflake api", id: "snowflake"},
		{key: "splunk api", id: "splunk"},
		{key: "stackby api", id: "stackby"},
		{key: "supabase api", id: "supabase"},
		{key: "surveymonkey api", id: "surveymonkey"},
		{key: "telegram bot api", id: "telegram"},
		{key: "todoist api", id: "todoist"},
		{key: "twake api", id: "twake"},
		{key: "twilio sms", id: "twilio"},
		{key: "twist api", id: "twist"},
		{key: "twitter api", id: "twitter"},
		{key: "typeform api", id: "typeform"},
		{key: "unleashed api", id: "unleashed-software"},
		{key: "uproc.io", id: "uproc"},
		{key: "uptime robot api", id: "uptimerobot"},
		{key: "vero track", id: "vero"},
		{key: "webflow api", id: "webflow"},
		{key: "whatsapp business api", id: "whatsapp"},
		{key: "wise api", id: "wise"},
		{key: "wc rest api", id: "woocommerce"},
		{key: "workable api", id: "workable"},
		{key: "workday wws", id: "workday"},
		{key: "xero accounting", id: "xero"},
		{key: "zoom meetings", id: "zoom"},
		{key: "zendesk support", id: "zendesk"},
		{key: "zohocrm", id: "zoho"},
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
		{id: "airtop", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "nvidia-dsx-air", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "autopilot", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "aws-acm", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityKnown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindSmithyJSON},
		{id: "aws-cognito", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityKnown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindSmithyJSON},
		{id: "aws-comprehend", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityKnown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindSmithyJSON},
		{id: "aws-dynamodb", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityKnown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindSmithyJSON},
		{id: "aws-elb", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityKnown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindSmithyJSON},
		{id: "aws-elbv2", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityKnown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindSmithyJSON},
		{id: "aws-iam", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityKnown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindSmithyJSON},
		{id: "aws-lambda", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityKnown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindSmithyJSON},
		{id: "aws-rekognition", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityKnown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindSmithyJSON},
		{id: "aws-s3", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityKnown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindSmithyJSON},
		{id: "aws-ses", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityKnown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindSmithyJSON},
		{id: "aws-sns", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityKnown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindSmithyJSON},
		{id: "aws-sqs", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityKnown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindSmithyJSON},
		{id: "aws-textract", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityKnown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindSmithyJSON},
		{id: "aws-transcribe", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityKnown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindSmithyJSON},
		{id: "azure-cosmos-db", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedPossible, specKind: SpecKindOpenAPI},
		{id: "azure-storage", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "bitbucket", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "bitly", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "box", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "brevo", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "bubble", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "cal", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
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
		{id: "cockpit", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "coda", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "coingecko", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "contentful", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "convertkit", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "copper", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "customer-io", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "databricks", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "deepl", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "dhl", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "discord", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedPossible, specKind: SpecKindOpenAPI},
		{id: "discourse", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "docker-engine", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "docker-hub", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "docker-registry", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedPossible, specKind: SpecKindOpenAPI},
		{id: "disqus", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "drift", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "dropbox", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityKnown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindDropboxStone},
		{id: "dropcontact", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "elastic", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "emelia", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "erpnext", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "eventbrite", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "facebook", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "facebook-lead-ads", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "figma", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "filemaker", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "formio", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "formstack", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "freshdesk", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "freshservice", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "freshworks-crm", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "getresponse", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "ghost", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "github", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "gitlab", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedPossible, specKind: SpecKindOpenAPI},
		{id: "gmail", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityKnown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindGoogleDiscovery},
		{id: "gong", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "google-admin", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityKnown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindGoogleDiscovery},
		{id: "google-analytics", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityKnown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindGoogleDiscovery},
		{id: "google-bigquery", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityKnown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindGoogleDiscovery},
		{id: "google-books", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityKnown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindGoogleDiscovery},
		{id: "google-business-profile", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityKnown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindGoogleDiscovery},
		{id: "google-calendar", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityKnown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindGoogleDiscovery},
		{id: "google-chat", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityKnown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindGoogleDiscovery},
		{id: "google-cloud-language", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityKnown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindGoogleDiscovery},
		{id: "google-cloud-storage", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityKnown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindGoogleDiscovery},
		{id: "google-contacts", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityKnown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindGoogleDiscovery},
		{id: "google-docs", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityKnown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindGoogleDiscovery},
		{id: "google-drive", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityKnown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindGoogleDiscovery},
		{id: "google-firebase-realtime-database", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityKnown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindGoogleDiscovery},
		{id: "google-firestore", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityKnown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindGoogleDiscovery},
		{id: "google-perspective", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityKnown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindGoogleDiscovery},
		{id: "google-sheets", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityKnown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindGoogleDiscovery},
		{id: "google-slides", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityKnown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindGoogleDiscovery},
		{id: "google-tasks", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityKnown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindGoogleDiscovery},
		{id: "google-translate", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityKnown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindGoogleDiscovery},
		{id: "google-youtube", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityKnown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindGoogleDiscovery},
		{id: "gotify", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "grafana", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "grist", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "gumroad", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "hackernews", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "harvest", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "help-scout", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "highlevel", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "hubspot", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPIIndex},
		{id: "humantic-ai", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "hunter", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "intercom", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "invoice-ninja", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "iterable", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "jenkins", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "jina-ai", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "jira-cloud", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "jotform", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "keap", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "kobotoolbox", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "kubernetes", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "line", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "linear", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "lingvanex", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "linkedin", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "magento", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "mailchimp", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "mailerlite", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "mailgun", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "mailjet", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "marketstack", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "matrix", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "mattermost", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "mautic", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "medium", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "messagebird", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "microsoft-entra", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "microsoft-excel", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "microsoft-graph", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "microsoft-graph-security", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "microsoft-onedrive", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "microsoft-outlook", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "microsoft-sharepoint", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "microsoft-teams", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "microsoft-todo", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "mindee", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "mocean", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "monday-com", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "monica-crm", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "mistral-ai", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "msg91", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "nasa", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "netlify", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "netsuite", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "nocodb", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "notion", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "npm", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "odoo", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "okta", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "onesimpleapi", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "onfleet", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "openai", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "openthesaurus", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "openweathermap", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "oracle-fusion-cloud-applications", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "paddle", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "paypal", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedPossible, specKind: SpecKindOpenAPI},
		{id: "peekalink", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "perplexity", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "phantombuster", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "pipedrive", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "plivo", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "posthog", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "postmark", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "pushbullet", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "pushcut", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "pushover", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "quickbooks", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "quickchart", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "raindrop", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "reddit", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "rocket-chat", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "salesforce", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "salesmate", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "sap-s4hana", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "sap-successfactors", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "seatable", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "segment", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedPossible, specKind: SpecKindHumanDocs},
		{id: "sendgrid", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedPossible, specKind: SpecKindOpenAPI},
		{id: "sendy", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "sentry", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "servicenow", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "shopify", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "pagerduty", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "signl4", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "slack", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedPossible, specKind: SpecKindOpenAPI},
		{id: "sms77", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "snowflake", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "splunk", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "spotify", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "stackby", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "storyblok", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "strapi", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "strava", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "stripe", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "supabase", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "surveymonkey", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "telegram", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "todoist", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "toggl", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "travis-ci", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "trello", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "twake", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "twilio", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "twist", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "twitter", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "typeform", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "unleashed-software", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "uplead", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "uproc", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "uptimerobot", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "vero", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "vonage", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "webflow", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "whatsapp", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "wise", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "woocommerce", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "wordpress", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "workable", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "workday", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "wufoo", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "xero", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedPossible, specKind: SpecKindOpenAPI},
		{id: "yourls", openAPI: SpecAvailabilityUnavailable, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedLikely, specKind: SpecKindHumanDocs},
		{id: "zendesk", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedPossible, specKind: SpecKindOpenAPI},
		{id: "zoho", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
		{id: "zoom", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedPossible, specKind: SpecKindOpenAPI},
		{id: "zulip", openAPI: SpecAvailabilityKnown, machine: SpecAvailabilityUnknown, userOpenAPINeed: UserOpenAPINeedNotExpected, specKind: SpecKindOpenAPI},
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
