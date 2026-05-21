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
	if got, want := len(overlays), 213; got != want {
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
		"acumatica",
		"adalo",
		"adyen",
		"affinity",
		"agile-crm",
		"airtable",
		"airtop",
		"apitemplate-io",
		"asana",
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
		"brevo",
		"bubble",
		"cal",
		"calendly",
		"chargebee",
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
		"kobotoolbox",
		"kubernetes",
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
	if !reflect.DeepEqual(gotIDs, wantIDs) {
		t.Fatalf("report provider ids = %#v, want %#v", gotIDs, wantIDs)
	}
	wantStatuses := map[string]AuthCompletenessStatus{
		"activecampaign":                    AuthStatusOverlayRequired,
		"action-network":                    AuthStatusOverlayRequired,
		"acumatica":                         AuthStatusPresentIncomplete,
		"adalo":                             AuthStatusOverlayRequired,
		"adyen":                             AuthStatusPresentIncomplete,
		"affinity":                          AuthStatusOverlayRequired,
		"agile-crm":                         AuthStatusOverlayRequired,
		"acuity-scheduling":                 AuthStatusOverlayRequired,
		"airtable":                          AuthStatusOverlayRequired,
		"airtop":                            AuthStatusPresentIncomplete,
		"asana":                             AuthStatusPresentIncomplete,
		"auth0":                             AuthStatusPresentIncomplete,
		"autopilot":                         AuthStatusOverlayRequired,
		"aws-acm":                           AuthStatusOverlayRequired,
		"aws-cognito":                       AuthStatusOverlayRequired,
		"aws-comprehend":                    AuthStatusOverlayRequired,
		"aws-dynamodb":                      AuthStatusOverlayRequired,
		"aws-elb":                           AuthStatusOverlayRequired,
		"aws-elbv2":                         AuthStatusOverlayRequired,
		"aws-iam":                           AuthStatusOverlayRequired,
		"aws-lambda":                        AuthStatusOverlayRequired,
		"aws-rekognition":                   AuthStatusOverlayRequired,
		"aws-s3":                            AuthStatusOverlayRequired,
		"aws-ses":                           AuthStatusOverlayRequired,
		"aws-sns":                           AuthStatusOverlayRequired,
		"aws-sqs":                           AuthStatusOverlayRequired,
		"aws-textract":                      AuthStatusOverlayRequired,
		"aws-transcribe":                    AuthStatusOverlayRequired,
		"azure-cosmos-db":                   AuthStatusOverlayRequired,
		"azure-storage":                     AuthStatusOverlayRequired,
		"bamboohr":                          AuthStatusOverlayRequired,
		"bannerbear":                        AuthStatusOverlayRequired,
		"baserow":                           AuthStatusComplete,
		"beeminder":                         AuthStatusOverlayRequired,
		"bigcommerce":                       AuthStatusPresentIncomplete,
		"bitbucket":                         AuthStatusPresentIncomplete,
		"bitly":                             AuthStatusComplete,
		"bitwarden":                         AuthStatusOverlayRequired,
		"box":                               AuthStatusPresentIncomplete,
		"brandfetch":                        AuthStatusComplete,
		"brevo":                             AuthStatusComplete,
		"bubble":                            AuthStatusOverlayRequired,
		"cal":                               AuthStatusPresentIncomplete,
		"calendly":                          AuthStatusOverlayRequired,
		"chargebee":                         AuthStatusPresentIncomplete,
		"circleci":                          AuthStatusPresentIncomplete,
		"cisco-meraki":                      AuthStatusPresentIncomplete,
		"cisco-webex":                       AuthStatusOverlayRequired,
		"clearbit":                          AuthStatusOverlayRequired,
		"clickup":                           AuthStatusPresentIncomplete,
		"clockify":                          AuthStatusOverlayRequired,
		"cloudflare":                        AuthStatusPresentIncomplete,
		"coda":                              AuthStatusComplete,
		"cockpit":                           AuthStatusOverlayRequired,
		"coingecko":                         AuthStatusOverlayRequired,
		"confluence-cloud":                  AuthStatusPresentIncomplete,
		"confluent-cloud":                   AuthStatusPresentIncomplete,
		"contentful":                        AuthStatusOverlayRequired,
		"convertkit":                        AuthStatusComplete,
		"copper":                            AuthStatusOverlayRequired,
		"cortex":                            AuthStatusOverlayRequired,
		"customer-io":                       AuthStatusPresentIncomplete,
		"databricks":                        AuthStatusOverlayRequired,
		"deepl":                             AuthStatusComplete,
		"dhl":                               AuthStatusOverlayRequired,
		"discord":                           AuthStatusComplete,
		"discourse":                         AuthStatusOverlayRequired,
		"docker-engine":                     AuthStatusPresentIncomplete,
		"docker-hub":                        AuthStatusComplete,
		"docker-registry":                   AuthStatusPresentIncomplete,
		"disqus":                            AuthStatusOverlayRequired,
		"docusign":                          AuthStatusPresentIncomplete,
		"drift":                             AuthStatusOverlayRequired,
		"dropbox":                           AuthStatusOverlayRequired,
		"dropcontact":                       AuthStatusOverlayRequired,
		"elastic":                           AuthStatusComplete,
		"emelia":                            AuthStatusOverlayRequired,
		"erpnext":                           AuthStatusOverlayRequired,
		"eventbrite":                        AuthStatusOverlayRequired,
		"facebook":                          AuthStatusOverlayRequired,
		"facebook-lead-ads":                 AuthStatusOverlayRequired,
		"figma":                             AuthStatusComplete,
		"filemaker":                         AuthStatusOverlayRequired,
		"formio":                            AuthStatusOverlayRequired,
		"formstack":                         AuthStatusOverlayRequired,
		"freshdesk":                         AuthStatusOverlayRequired,
		"freshservice":                      AuthStatusOverlayRequired,
		"freshworks-crm":                    AuthStatusOverlayRequired,
		"getresponse":                       AuthStatusOverlayRequired,
		"ghost":                             AuthStatusOverlayRequired,
		"github":                            AuthStatusOverlayRequired,
		"gitlab":                            AuthStatusPresentIncomplete,
		"gmail":                             AuthStatusOverlayRequired,
		"gong":                              AuthStatusOverlayRequired,
		"google-admin":                      AuthStatusOverlayRequired,
		"google-analytics":                  AuthStatusOverlayRequired,
		"google-bigquery":                   AuthStatusOverlayRequired,
		"google-books":                      AuthStatusOverlayRequired,
		"google-business-profile":           AuthStatusOverlayRequired,
		"google-calendar":                   AuthStatusOverlayRequired,
		"google-chat":                       AuthStatusOverlayRequired,
		"google-cloud-language":             AuthStatusOverlayRequired,
		"google-cloud-storage":              AuthStatusOverlayRequired,
		"google-contacts":                   AuthStatusOverlayRequired,
		"google-docs":                       AuthStatusOverlayRequired,
		"google-drive":                      AuthStatusOverlayRequired,
		"google-firebase-realtime-database": AuthStatusOverlayRequired,
		"google-firestore":                  AuthStatusOverlayRequired,
		"google-perspective":                AuthStatusOverlayRequired,
		"google-sheets":                     AuthStatusOverlayRequired,
		"google-slides":                     AuthStatusOverlayRequired,
		"google-tasks":                      AuthStatusOverlayRequired,
		"google-translate":                  AuthStatusOverlayRequired,
		"google-youtube":                    AuthStatusOverlayRequired,
		"gotify":                            AuthStatusComplete,
		"grafana":                           AuthStatusComplete,
		"grist":                             AuthStatusOverlayRequired,
		"gumroad":                           AuthStatusOverlayRequired,
		"hackernews":                        AuthStatusIntentionallyAnonymous,
		"harvest":                           AuthStatusOverlayRequired,
		"help-scout":                        AuthStatusOverlayRequired,
		"highlevel":                         AuthStatusComplete,
		"home-assistant":                    AuthStatusOverlayRequired,
		"hubspot":                           AuthStatusOverlayRequired,
		"humantic-ai":                       AuthStatusOverlayRequired,
		"hunter":                            AuthStatusOverlayRequired,
		"intercom":                          AuthStatusPresentIncomplete,
		"invoice-ninja":                     AuthStatusOverlayRequired,
		"iterable":                          AuthStatusOverlayRequired,
		"jenkins":                           AuthStatusOverlayRequired,
		"jina-ai":                           AuthStatusOverlayRequired,
		"jira-cloud":                        AuthStatusPresentIncomplete,
		"jotform":                           AuthStatusOverlayRequired,
		"keap":                              AuthStatusOverlayRequired,
		"kobotoolbox":                       AuthStatusComplete,
		"kubernetes":                        AuthStatusPresentIncomplete,
		"line":                              AuthStatusComplete,
		"linear":                            AuthStatusOverlayRequired,
		"lingvanex":                         AuthStatusOverlayRequired,
		"linkedin":                          AuthStatusOverlayRequired,
		"magento":                           AuthStatusOverlayRequired,
		"mailchimp":                         AuthStatusOverlayRequired,
		"mailerlite":                        AuthStatusOverlayRequired,
		"mailgun":                           AuthStatusPresentIncomplete,
		"mailjet":                           AuthStatusOverlayRequired,
		"marketstack":                       AuthStatusComplete,
		"matrix":                            AuthStatusComplete,
		"mattermost":                        AuthStatusComplete,
		"mautic":                            AuthStatusOverlayRequired,
		"medium":                            AuthStatusOverlayRequired,
		"messagebird":                       AuthStatusOverlayRequired,
		"microsoft-entra":                   AuthStatusOverlayRequired,
		"microsoft-excel":                   AuthStatusOverlayRequired,
		"microsoft-graph":                   AuthStatusOverlayRequired,
		"microsoft-graph-security":          AuthStatusOverlayRequired,
		"microsoft-onedrive":                AuthStatusOverlayRequired,
		"microsoft-outlook":                 AuthStatusOverlayRequired,
		"microsoft-sharepoint":              AuthStatusOverlayRequired,
		"microsoft-teams":                   AuthStatusOverlayRequired,
		"microsoft-todo":                    AuthStatusOverlayRequired,
		"mindee":                            AuthStatusOverlayRequired,
		"misp":                              AuthStatusComplete,
		"mistral-ai":                        AuthStatusOverlayRequired,
		"mocean":                            AuthStatusOverlayRequired,
		"monday-com":                        AuthStatusOverlayRequired,
		"monica-crm":                        AuthStatusOverlayRequired,
		"msg91":                             AuthStatusOverlayRequired,
		"nasa":                              AuthStatusOverlayRequired,
		"netlify":                           AuthStatusComplete,
		"netscaler":                         AuthStatusOverlayRequired,
		"netsuite":                          AuthStatusPresentIncomplete,
		"nocodb":                            AuthStatusPresentIncomplete,
		"notion":                            AuthStatusPresentIncomplete,
		"npm":                               AuthStatusOverlayRequired,
		"odoo":                              AuthStatusOverlayRequired,
		"okta":                              AuthStatusPresentIncomplete,
		"onesimpleapi":                      AuthStatusOverlayRequired,
		"onfleet":                           AuthStatusOverlayRequired,
		"openai":                            AuthStatusComplete,
		"openthesaurus":                     AuthStatusIntentionallyAnonymous,
		"openweathermap":                    AuthStatusOverlayRequired,
		"oracle-fusion-cloud-applications":  AuthStatusPresentIncomplete,
		"paddle":                            AuthStatusComplete,
		"pagerduty":                         AuthStatusPresentIncomplete,
		"paypal":                            AuthStatusComplete,
		"peekalink":                         AuthStatusComplete,
		"perplexity":                        AuthStatusComplete,
		"phantombuster":                     AuthStatusOverlayRequired,
		"pipedrive":                         AuthStatusPresentIncomplete,
		"plivo":                             AuthStatusOverlayRequired,
		"posthog":                           AuthStatusPresentIncomplete,
		"postmark":                          AuthStatusOverlayRequired,
		"pushbullet":                        AuthStatusOverlayRequired,
		"pushcut":                           AuthStatusOverlayRequired,
		"pushover":                          AuthStatusOverlayRequired,
		"quickbooks":                        AuthStatusOverlayRequired,
		"quickchart":                        AuthStatusIntentionallyAnonymous,
		"raindrop":                          AuthStatusOverlayRequired,
		"reddit":                            AuthStatusOverlayRequired,
		"rocket-chat":                       AuthStatusPresentIncomplete,
		"rundeck":                           AuthStatusComplete,
		"salesforce":                        AuthStatusOverlayRequired,
		"salesmate":                         AuthStatusOverlayRequired,
		"sap-s4hana":                        AuthStatusPresentIncomplete,
		"sap-successfactors":                AuthStatusPresentIncomplete,
		"seatable":                          AuthStatusComplete,
		"securityscorecard":                 AuthStatusComplete,
		"segment":                           AuthStatusComplete,
		"sendgrid":                          AuthStatusComplete,
		"sendy":                             AuthStatusOverlayRequired,
		"sentry":                            AuthStatusOverlayRequired,
		"servicenow":                        AuthStatusOverlayRequired,
		"shopify":                           AuthStatusOverlayRequired,
		"signl4":                            AuthStatusOverlayRequired,
		"slack":                             AuthStatusPresentIncomplete,
		"sms77":                             AuthStatusOverlayRequired,
		"snowflake":                         AuthStatusComplete,
		"splunk":                            AuthStatusOverlayRequired,
		"spotify":                           AuthStatusComplete,
		"stackby":                           AuthStatusOverlayRequired,
		"storyblok":                         AuthStatusOverlayRequired,
		"strapi":                            AuthStatusOverlayRequired,
		"strava":                            AuthStatusComplete,
		"stripe":                            AuthStatusComplete,
		"supabase":                          AuthStatusPresentIncomplete,
		"surveymonkey":                      AuthStatusOverlayRequired,
		"telegram":                          AuthStatusOverlayRequired,
		"todoist":                           AuthStatusOverlayRequired,
		"toggl":                             AuthStatusPresentIncomplete,
		"travis-ci":                         AuthStatusOverlayRequired,
		"flow":                              AuthStatusOverlayRequired,
		"gotowebinar":                       AuthStatusOverlayRequired,
		"halopsa":                           AuthStatusOverlayRequired,
		"lonescale":                         AuthStatusOverlayRequired,
		"lemlist":                           AuthStatusOverlayRequired,
		"orbit":                             AuthStatusOverlayRequired,
		"oura":                              AuthStatusOverlayRequired,
		"profitwell":                        AuthStatusOverlayRequired,
		"quickbase":                         AuthStatusOverlayRequired,
		"syncromsp":                         AuthStatusComplete,
		"taiga":                             AuthStatusOverlayRequired,
		"tapfiliate":                        AuthStatusOverlayRequired,
		"thehive":                           AuthStatusComplete,
		"thehive-project":                   AuthStatusComplete,
		"apitemplate-io":                    AuthStatusOverlayRequired,
		"currents":                          AuthStatusOverlayRequired,
		"demio":                             AuthStatusOverlayRequired,
		"egoi":                              AuthStatusComplete,
		"mailcheck":                         AuthStatusComplete,
		"mandrill":                          AuthStatusOverlayRequired,
		"metabase":                          AuthStatusOverlayRequired,
		"nextcloud":                         AuthStatusComplete,
		"nvidia-dsx-air":                    AuthStatusOverlayRequired,
		"philips-hue":                       AuthStatusOverlayRequired,
		"postbin":                           AuthStatusIntentionallyAnonymous,
		"trello":                            AuthStatusPresentIncomplete,
		"twake":                             AuthStatusOverlayRequired,
		"twilio":                            AuthStatusComplete,
		"twist":                             AuthStatusOverlayRequired,
		"twitter":                           AuthStatusComplete,
		"typeform":                          AuthStatusOverlayRequired,
		"unleashed-software":                AuthStatusOverlayRequired,
		"uplead":                            AuthStatusOverlayRequired,
		"uproc":                             AuthStatusPresentIncomplete,
		"uptimerobot":                       AuthStatusComplete,
		"urlscan":                           AuthStatusComplete,
		"venafi":                            AuthStatusOverlayRequired,
		"vero":                              AuthStatusOverlayRequired,
		"vonage":                            AuthStatusComplete,
		"webflow":                           AuthStatusOverlayRequired,
		"wekan":                             AuthStatusOverlayRequired,
		"whatsapp":                          AuthStatusOverlayRequired,
		"wise":                              AuthStatusOverlayRequired,
		"woocommerce":                       AuthStatusOverlayRequired,
		"wordpress":                         AuthStatusOverlayRequired,
		"workable":                          AuthStatusOverlayRequired,
		"workday":                           AuthStatusPresentIncomplete,
		"wufoo":                             AuthStatusOverlayRequired,
		"xero":                              AuthStatusComplete,
		"yourls":                            AuthStatusOverlayRequired,
		"zammad":                            AuthStatusOverlayRequired,
		"zendesk":                           AuthStatusPresentIncomplete,
		"zoho":                              AuthStatusComplete,
		"zoom":                              AuthStatusPresentIncomplete,
		"zulip":                             AuthStatusComplete,
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
