# Human-Docs Endpoint Overlay Decisions

These OpenAPI-shaped advisory overlays are derived from official documentation
or official machine-readable sources. They are not official provider OpenAPI
documents and must not be treated as provider truth.

## Built

| Provider | Overlay | Reason |
|---|---|---|
| Action Network | `action-network-api-v2-overlay.json` | OSDI/HAL+JSON REST API v2 docs support a focused API entry point, people, petitions, forms, events, tags, taggings, and webhooks subset. |
| Acuity Scheduling | `acuity-scheduling-api-v1-overlay.json` | REST-shaped API v1 docs support a focused appointments, availability, appointment-types, calendars, and clients subset. |
| ActiveCampaign | `activecampaign-api-v3-overlay.json` | REST-shaped API v3 docs support a focused contacts, lists, campaigns, and deals subset; account host remains operator supplied. |
| Adalo | `adalo-api-overlay.json` | API docs support a focused app collections, records, and push notification subset; app, collection, and field schemas remain operator supplied. |
| Affinity | `affinity-v1-api-overlay.json` | V1 API docs support a focused lists, fields, field values, people, organizations, opportunities, interactions, notes, files, and webhooks subset. |
| Agile CRM | `agile-crm-rest-api-overlay.json` | REST API docs support a focused contacts, companies, deals, tasks, events, tracks, campaigns, and help desk subset; account subdomain remains operator supplied. |
| Airtable | `airtable-web-api-overlay.json` | REST-shaped official Web API docs support a focused table/record subset. |
| Autopilot | `autopilot-api-overlay.json` | API Blueprint-style docs and help docs support a focused contacts and lists subset with the documented API-key header. |
| BambooHR | `bamboohr-api-v1-overlay.json` | REST-shaped API docs support a focused employees, directory, fields, reports, and time-off subset; customer subdomain remains operator supplied. |
| Bannerbear | `bannerbear-api-v2-overlay.json` | REST-shaped API v2 docs support a focused auth, images, collections, videos, screenshots, templates, and projects subset. |
| Beeminder | `beeminder-api-v1-overlay.json` | REST-shaped API v1 docs support a focused users, goals, archived goals, datapoints, charges, and selected goal action subset. |
| Bubble | `bubble-api-overlay.json` | API docs and app-specific Swagger metadata support a generic Data API and Workflow API subset; app host and object schemas remain operator supplied. |
| Calendly | `calendly-public-api-overlay.json` | REST-shaped official public API docs support event type, event, webhook, and user endpoints. |
| Clearbit | `clearbit-api-overlay.json` | API support docs support a narrow Prospector people search and Person Combined enrichment subset; broader legacy APIs and account access remain review-sensitive. |
| Clockify | `clockify-api-v1-overlay.json` | REST-shaped API v1 docs support a focused users, workspaces, projects, clients, tags, and time-entries subset. |
| Cockpit | `cockpit-api-overlay.json` | Self-hosted API docs support a focused collections, singletons, assets, and content subset; host and content model names remain operator supplied. |
| CoinGecko | `coingecko-api-v3-overlay.json` | API reference docs support a focused public v3 price, coin, market, and exchange subset; Demo and Pro auth variants remain catalog security metadata. |
| Contentful | `contentful-management-api-overlay.json` | REST-shaped Content Management API docs support a focused spaces, environments, entries, assets, and content-types subset. |
| Copper | `copper-developer-api-overlay.json` | Developer API docs support a focused users, leads, people, companies, opportunities, projects, tasks, activities, and webhooks subset. |
| Databricks | `databricks-workspace-rest-overlay.json` | REST-shaped workspace API docs support a focused jobs, clusters, workspace, and DBFS subset; host remains operator supplied. |
| Drift | `drift-platform-api-overlay.json` | REST-shaped Platform API docs support a focused contacts, conversations, and users subset with OAuth bearer auth. |
| Eventbrite | `eventbrite-platform-api-v3-overlay.json` | REST-shaped Platform API docs support a focused users, organizations, events, attendees, venues, and ticket-classes subset. |
| FileMaker | `filemaker-data-api-overlay.json` | Data API reference docs support a focused sessions, layouts, records, scripts, and containers subset; host, database, layout, and script names remain operator supplied. |
| Form.io | `formio-api-overlay.json` | API docs support a focused projects, forms, submissions, and roles subset; cloud/self-hosted base URL and project aliases remain operator supplied. |
| Formstack | `formstack-api-v2-overlay.json` | Forms API v2 docs support a focused forms, fields, submissions, folders, and webhooks subset. |
| Freshdesk | `freshdesk-api-v2-overlay.json` | REST-shaped API v2 docs support a focused tickets, contacts, companies, and agents subset; account domain remains operator supplied. |
| Freshservice | `freshservice-api-v2-overlay.json` | REST-shaped API v2 docs support a focused tickets, problems, changes, releases, requesters, agents, assets, groups, departments, and service catalog subset; account domain remains operator supplied. |
| Freshworks CRM | `freshworks-crm-api-overlay.json` | REST-shaped CRM docs support a focused contacts, sales accounts, deals, and tasks subset; account domain remains operator supplied. |
| GetResponse | `getresponse-api-v3-overlay.json` | REST-shaped API v3 docs support a focused campaigns, contacts, newsletters, and tags subset with API-key auth. |
| Ghost | `ghost-admin-api-overlay.json` | REST-shaped Admin API docs support a focused posts, pages, tags, users, and members subset; admin host remains operator supplied. |
| Gong | `gong-public-api-overlay.json` | Public help docs support a narrow users and call-upload subset; the full API reference remains account-session gated. |
| Grist | `grist-rest-api-overlay.json` | REST API usage and reference docs support a focused organizations, workspaces, documents, tables, records, SQL queries, and webhooks subset; site host remains operator supplied. |
| Hacker News | `hackernews-firebase-api-overlay.json` | Official Firebase API docs support a focused public item, user, story-list, and updates subset. |
| Harvest | `harvest-api-v2-overlay.json` | REST-shaped API v2 docs support a focused users, company, clients, projects, time-entries, and invoices subset. |
| Help Scout | `help-scout-inbox-api-v2-overlay.json` | REST-shaped Inbox API v2 docs support a focused conversations, customers, mailboxes, and threads subset. |
| Iterable | `iterable-api-overlay.json` | REST-shaped API docs support a focused users, events, campaigns, lists, and templates subset. |
| Jenkins | `jenkins-remote-api-overlay.json` | Official Remote Access API docs support a small generic controller/job/queue subset; instance and plugin coverage remains variable. |
| JotForm | `jotform-api-overlay.json` | API docs support a focused forms, submissions, questions, reports, folders, and webhooks subset. |
| Keap | `keap-rest-api-overlay.json` | REST-shaped docs support a focused contacts, companies, orders, and tags subset with OAuth bearer auth. |
| Mailchimp | `mailchimp-marketing-api-overlay.json` | REST-shaped Marketing API docs support a focused lists, campaigns, and reports subset. |
| MailerLite | `mailerlite-api-overlay.json` | REST-shaped current API docs support a focused subscribers, groups, campaigns, and forms subset; Classic auth remains security metadata. |
| Mailjet | `mailjet-rest-api-overlay.json` | REST-shaped REST API docs support a focused contacts, list-recipients, campaigns, messages, and send subset. |
| Mautic | `mautic-api-overlay.json` | Instance-hosted REST API docs support a focused contacts, companies, segments, campaigns, and emails subset; host remains operator supplied. |
| Monica CRM | `monica-crm-api-overlay.json` | REST-shaped API docs support a focused contacts, activities, calls, reminders, and tags subset with bearer auth. |
| NASA | `nasa-open-apis-overlay.json` | Open APIs docs support a focused APOD, Mars Rover Photos, NeoWs, EPIC, and DONKI subset. |
| OneSimpleApi | `onesimpleapi-toolkit-overlay.json` | Official docs support a focused web metadata, page status, screenshot, PDF, and QR utility subset. |
| OpenThesaurus | `openthesaurus-api-overlay.json` | Webservice docs support the documented anonymous synonym search endpoint with JSON/XML and lookup options. |
| OpenWeatherMap | `openweathermap-one-call-3-overlay.json` | One Call 3.0 docs support a narrow weather endpoint overlay. |
| Postmark | `postmark-api-overlay.json` | REST-shaped API docs and API Explorer support a focused email, template, bounce, and server subset with separate server/account token headers. |
| QuickBooks | `quickbooks-online-accounting-api-overlay.json` | REST-shaped Accounting API docs support a focused customer, invoice, payment, and company-info subset. |
| QuickChart | `quickchart-api-overlay.json` | Chart and QR-code docs support a focused anonymous chart, QR, and Graphviz rendering subset. |
| Reddit | `reddit-api-overlay.json` | Generated API docs support a focused OAuth API subset for identity, user/subreddit lookup, listings, and submit operations. |
| Salesforce | `salesforce-rest-core-overlay.json` | REST API docs support a focused sObject/query/composite subset; instance and API version remain operator supplied. |
| Salesmate | `salesmate-api-overlay.json` | API docs and support docs support a focused contacts, companies, deals, activities, and users subset with sessionToken and x-linkname headers. |
| Sentry | `sentry-rest-api-overlay.json` | REST-shaped API docs support a focused organization, project, issue, event, and release subset. |
| ServiceNow | `servicenow-rest-api-overlay.json` | REST API Explorer and Table API docs support a focused instance/table/import subset. |
| Sendy | `sendy-api-overlay.json` | Installation-hosted API docs support a focused subscribe, unsubscribe, list, campaign, and subscriber-count subset; host remains operator supplied. |
| Shopify | `shopify-admin-rest-overlay.json` | Admin REST docs support a focused products, orders, customers, webhooks, and inventory subset. |
| Stackby | `stackby-api-overlay.json` | Developer API docs and API-key help support a focused stacks, tables, and rows subset; stack/table identifiers remain operator supplied. |
| SurveyMonkey | `surveymonkey-api-v3-overlay.json` | API v3 docs support a focused surveys, collectors, responses, questions, and webhooks subset. |
| Splunk | `splunk-enterprise-rest-overlay.json` | REST API docs support a focused server, search jobs, results, and saved-search subset; product version remains review-sensitive. |
| Telegram | `telegram-bot-api-overlay.json` | Bot API docs support common bot methods; bot token is modeled as a server variable because OpenAPI security schemes cannot represent Telegram path-token auth exactly. |
| Todoist | `todoist-rest-api-v2-overlay.json` | REST API v2 docs support a focused projects, sections, tasks, and comments subset; docs now point developers toward newer unified API docs. |
| Typeform | `typeform-rest-api-overlay.json` | REST-shaped Typeform docs support a focused forms, responses, webhooks, and workspaces subset. |
| Vero | `vero-track-api-overlay.json` | Track API docs support a focused users, events, tags, and unsubscribe subset with auth_token request parameters. |
| Webflow | `webflow-data-api-v2-overlay.json` | REST-shaped Data API docs support a focused sites, pages, collections, and CMS items subset. |

## No Endpoint Overlay

| Provider | Decision |
|---|---|
| Linear | No OpenAPI-shaped endpoint overlay for now. Linear's public API is GraphQL with introspection; a single `POST /graphql` wrapper would hide operation semantics rather than improve OpenAPI-like metadata. |
| Monday.com | No OpenAPI-shaped endpoint overlay for now. Monday.com's platform API is GraphQL at a single endpoint; useful operation coverage should come from a GraphQL-aware classifier or schema/introspection workflow, not a REST-shaped overlay. |
| TimeSaved | No OpenAPI-shaped endpoint overlay. The frozen M26 item is an n8n workflow metadata helper, not an external provider API with HTTP endpoints. |
