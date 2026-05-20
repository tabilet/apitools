# Human-Docs Endpoint Overlay Decisions

These OpenAPI-shaped advisory overlays are derived from official documentation
or official machine-readable sources. They are not official provider OpenAPI
documents and must not be treated as provider truth.

## M46 High-Value SaaS Review

The 2026-05-20 high-value SaaS review retained the existing official
OpenAPI/Swagger/OpenAPI-index artifacts for Asana, Box, Discord, GitHub,
GitLab, HubSpot, Jira Cloud, Notion, OpenAI, PagerDuty, SendGrid, Slack,
Stripe, Trello, Twilio, Xero, Zendesk, and Zoho.

The same review retained docs-derived endpoint overlays for Airtable,
QuickBooks, Salesforce, ServiceNow, and Shopify because official docs support
focused REST-shaped subsets, while no stable public provider-wide OpenAPI
artifact was recorded. Linear remains an explicit no-endpoint-overlay case:
its public API is GraphQL with introspection, and a generic POST `/graphql`
overlay would hide native operation semantics.

## Built

| Provider | Overlay | Reason |
|---|---|---|
| Action Network | `action-network-api-v2-overlay.json` | OSDI/HAL+JSON REST API v2 docs support a focused API entry point, people, petitions, forms, events, tags, taggings, and webhooks subset. |
| Acuity Scheduling | `acuity-scheduling-api-v1-overlay.json` | REST-shaped API v1 docs support a focused appointments, availability, appointment-types, calendars, and clients subset. |
| ActiveCampaign | `activecampaign-api-v3-overlay.json` | REST-shaped API v3 docs support a focused contacts, lists, campaigns, and deals subset; account host remains operator supplied. |
| Adalo | `adalo-api-overlay.json` | API docs support a focused app collections, records, and push notification subset; app, collection, and field schemas remain operator supplied. |
| Affinity | `affinity-v1-api-overlay.json` | V1 API docs support a focused lists, fields, field values, people, organizations, opportunities, interactions, notes, files, and webhooks subset. |
| Agile CRM | `agile-crm-rest-api-overlay.json` | REST API docs support a focused contacts, companies, deals, tasks, events, tracks, campaigns, and help desk subset; account subdomain remains operator supplied. |
| APITemplate.io | `apitemplate-io-api-overlay.json` | API docs support a focused v1 PDF/image creation and template listing subset with X-API-KEY auth. |
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
| Currents | `currents-api-overlay.json` | API docs support a focused projects, runs, tests, spec files, actions, signatures, instances, and webhooks subset with bearer API-key auth. |
| Databricks | `databricks-workspace-rest-overlay.json` | REST-shaped workspace API docs support a focused jobs, clusters, workspace, and DBFS subset; host remains operator supplied. |
| Demio | `demio-api-overlay.json` | Public API docs support a focused events, registrations, and reports subset with paired Api-Key and Api-Secret headers. |
| DHL | `dhl-shipment-tracking-overlay.json` | Shipment Tracking - Unified docs support the documented shipment tracking endpoint with subscription-key auth. |
| Drift | `drift-platform-api-overlay.json` | REST-shaped Platform API docs support a focused contacts, conversations, and users subset with OAuth bearer auth. |
| Disqus | `disqus-api-overlay.json` | REST-shaped API docs support a focused forums, threads, and posts subset with access_token/api_key request-parameter auth. |
| ERPNext | `erpnext-rest-api-overlay.json` | ERPNext and Frappe REST docs support a focused generic DocType, common ERP resource, and whitelisted-method subset; site host and DocTypes remain operator supplied. |
| Eventbrite | `eventbrite-platform-api-v3-overlay.json` | REST-shaped Platform API docs support a focused users, organizations, events, attendees, venues, and ticket-classes subset. |
| Facebook | `facebook-graph-api-overlay.json` | Graph API and Pages API docs support a focused profile, page, feed, comments, likes, page subscriptions, photos, conversations, and app subscription subset. |
| Facebook Lead Ads | `facebook-lead-ads-overlay.json` | Lead Ads retrieval and webhook docs support a focused page form, lead retrieval, leadgen form, and webhook subscription subset. |
| FileMaker | `filemaker-data-api-overlay.json` | Data API reference docs support a focused sessions, layouts, records, scripts, and containers subset; host, database, layout, and script names remain operator supplied. |
| Flow | `flow-api-overlay.json` | REST-shaped API docs support a focused tasks, access tokens, and integration webhook subset with bearer personal-access-token auth. |
| Form.io | `formio-api-overlay.json` | API docs support a focused projects, forms, submissions, and roles subset; cloud/self-hosted base URL and project aliases remain operator supplied. |
| Formstack | `formstack-api-v2-overlay.json` | Forms API v2 docs support a focused forms, fields, submissions, folders, and webhooks subset. |
| Freshdesk | `freshdesk-api-v2-overlay.json` | REST-shaped API v2 docs support a focused tickets, contacts, companies, and agents subset; account domain remains operator supplied. |
| Freshservice | `freshservice-api-v2-overlay.json` | REST-shaped API v2 docs support a focused tickets, problems, changes, releases, requesters, agents, assets, groups, departments, and service catalog subset; account domain remains operator supplied. |
| Freshworks CRM | `freshworks-crm-api-overlay.json` | REST-shaped CRM docs support a focused contacts, sales accounts, deals, and tasks subset; account domain remains operator supplied. |
| GetResponse | `getresponse-api-v3-overlay.json` | REST-shaped API v3 docs support a focused campaigns, contacts, newsletters, and tags subset with API-key auth. |
| Ghost | `ghost-admin-api-overlay.json` | REST-shaped Admin API docs support a focused posts, pages, tags, users, and members subset; admin host remains operator supplied. |
| Gong | `gong-public-api-overlay.json` | Public help docs support a narrow users and call-upload subset; the full API reference remains account-session gated. |
| GoTo Webinar | `gotowebinar-api-overlay.json` | API v2 docs support a focused webinars, sessions, registrants, attendees, panelists, and co-organizers subset with OAuth2 bearer auth. |
| Grist | `grist-rest-api-overlay.json` | REST API usage and reference docs support a focused organizations, workspaces, documents, tables, records, SQL queries, and webhooks subset; site host remains operator supplied. |
| Gumroad | `gumroad-api-overlay.json` | API docs support a focused products, sales, offers, license verification, and resource-subscription subset. |
| Hacker News | `hackernews-firebase-api-overlay.json` | Official Firebase API docs support a focused public item, user, story-list, and updates subset. |
| HaloPSA | `halopsa-api-overlay.json` | API docs support a focused tickets, clients, sites, and users subset; auth and resource hosts remain tenant supplied. |
| Harvest | `harvest-api-v2-overlay.json` | REST-shaped API v2 docs support a focused users, company, clients, projects, time-entries, and invoices subset. |
| Help Scout | `help-scout-inbox-api-v2-overlay.json` | REST-shaped Inbox API v2 docs support a focused conversations, customers, mailboxes, and threads subset. |
| Iterable | `iterable-api-overlay.json` | REST-shaped API docs support a focused users, events, campaigns, lists, and templates subset. |
| Invoice Ninja | `invoice-ninja-api-overlay.json` | API docs support a focused clients, invoices, quotes, payments, products, vendors, projects, and webhook-endpoints subset; host remains operator supplied for self-hosted installs. |
| Jenkins | `jenkins-remote-api-overlay.json` | Official Remote Access API docs support a small generic controller/job/queue subset; instance and plugin coverage remains variable. |
| JotForm | `jotform-api-overlay.json` | API docs support a focused forms, submissions, questions, reports, folders, and webhooks subset. |
| Keap | `keap-rest-api-overlay.json` | REST-shaped docs support a focused contacts, companies, orders, and tags subset with OAuth bearer auth. |
| Lemlist | `lemlist-api-overlay.json` | API reference docs support a focused campaigns, leads, activities, enrichment, team, and unsubscribes subset with Basic auth using an API key password. |
| LinkedIn | `linkedin-api-overlay.json` | Marketing and shared API docs support a focused userinfo, organization, posts, ad accounts, and lead forms subset; Rest.li/version headers and product access remain review-sensitive. |
| LoneScale | `lonescale-api-overlay.json` | Public API docs support a focused lists, list items, user, and workflows subset with X-API-KEY auth. |
| Magento / Adobe Commerce | `magento-rest-api-overlay.json` | REST docs and local Swagger generation guidance support a focused products, categories, customers, orders, invoices, shipments, and token subset; Commerce host, store code, installed modules, and scopes remain operator supplied. |
| Mailchimp | `mailchimp-marketing-api-overlay.json` | REST-shaped Marketing API docs support a focused lists, campaigns, and reports subset. |
| MailerLite | `mailerlite-api-overlay.json` | REST-shaped current API docs support a focused subscribers, groups, campaigns, and forms subset; Classic auth remains security metadata. |
| Mailjet | `mailjet-rest-api-overlay.json` | REST-shaped REST API docs support a focused contacts, list-recipients, campaigns, messages, and send subset. |
| Mandrill | `mandrill-transactional-api-overlay.json` | Transactional API docs support a focused messages, templates, users, and webhooks subset; API key is modeled as advisory credential placement because it is sent in each request payload. |
| Mautic | `mautic-api-overlay.json` | Instance-hosted REST API docs support a focused contacts, companies, segments, campaigns, and emails subset; host remains operator supplied. |
| MessageBird | `messagebird-api-overlay.json` | REST-shaped API docs support a focused balance, messages, contacts, voice calls, and HLR subset with AccessKey auth. |
| Metabase | `metabase-api-overlay.json` | API docs and instance live-doc guidance support a focused session, current user, databases, questions, alerts, and metrics subset; host remains operator supplied. |
| Medium | `medium-api-overlay.json` | REST-shaped official API docs support a focused current-user, publication list, and post publishing subset with bearer auth. |
| Monica CRM | `monica-crm-api-overlay.json` | REST-shaped API docs support a focused contacts, activities, calls, reminders, and tags subset with bearer auth. |
| Mocean | `mocean-api-overlay.json` | REST-shaped API docs support a focused SMS, voice dial, and account balance subset with API key and secret request parameters. |
| MSG91 | `msg91-api-overlay.json` | REST-shaped API docs support a focused SMS, transactional SMS, OTP send, and OTP verify subset with authkey request-parameter auth. |
| NASA | `nasa-open-apis-overlay.json` | Open APIs docs support a focused APOD, Mars Rover Photos, NeoWs, EPIC, and DONKI subset. |
| npm | `npm-registry-api-overlay.json` | Registry docs support a focused package metadata, search, dist-tag, and whoami subset with bearer auth metadata. |
| Onfleet | `onfleet-api-overlay.json` | API docs support a focused auth, tasks, workers, teams, hubs, destinations, recipients, admins, and webhooks subset. |
| OneSimpleApi | `onesimpleapi-toolkit-overlay.json` | Official docs support a focused web metadata, page status, screenshot, PDF, and QR utility subset. |
| OpenThesaurus | `openthesaurus-api-overlay.json` | Webservice docs support the documented anonymous synonym search endpoint with JSON/XML and lookup options. |
| OpenWeatherMap | `openweathermap-one-call-3-overlay.json` | One Call 3.0 docs support a narrow weather endpoint overlay. |
| Postmark | `postmark-api-overlay.json` | REST-shaped API docs and API Explorer support a focused email, template, bounce, and server subset with separate server/account token headers. |
| ProfitWell | `profitwell-api-v2-overlay.json` | API v2 docs support a focused company settings and metrics subset with Authorization header token auth. |
| Quickbase | `quickbase-rest-api-overlay.json` | REST API docs support a focused fields, records, reports, and file subset with bearer user-token and realm-hostname headers. |
| QuickBooks | `quickbooks-online-accounting-api-overlay.json` | REST-shaped Accounting API docs support a focused customer, invoice, payment, and company-info subset. |
| QuickChart | `quickchart-api-overlay.json` | Chart and QR-code docs support a focused anonymous chart, QR, and Graphviz rendering subset. |
| Raindrop.io | `raindrop-api-overlay.json` | REST-shaped API docs support a focused raindrops, collections, tags, filters, highlights, and user subset with OAuth bearer auth. |
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
| Storyblok | `storyblok-api-overlay.json` | Content Delivery and Management API docs support a focused story read, list, publish, unpublish, and management subset with content-token and bearer auth. |
| Strapi | `strapi-rest-api-overlay.json` | REST API docs support a generic collection-entry and local-auth subset; host and content types remain operator supplied. |
| Taiga | `taiga-api-overlay.json` | REST API docs support a focused projects, epics, issues, tasks, user stories, users, and webhooks subset; cloud/self-hosted host remains operator selected. |
| Tapfiliate | `tapfiliate-api-overlay.json` | REST API docs support a focused affiliates, affiliate metadata, programs, and program-affiliate subset with Api-Key auth. |
| Telegram | `telegram-bot-api-overlay.json` | Bot API docs support common bot methods; bot token is modeled as a server variable because OpenAPI security schemes cannot represent Telegram path-token auth exactly. |
| Todoist | `todoist-rest-api-v2-overlay.json` | REST API v2 docs support a focused projects, sections, tasks, and comments subset; docs now point developers toward newer unified API docs. |
| Travis CI | `travis-ci-api-overlay.json` | API v3 docs support a focused repositories, builds, jobs, requests, branches, and environment-variable subset with token and API-version headers. |
| Twake | `twake-developers-api-overlay.json` | Developers API docs support the documented application-authenticated message save/remove subset; workspace and channel identifiers remain operator supplied. |
| Twist | `twist-api-v3-overlay.json` | API v3 docs support a focused users, workspaces, channels, threads, comments, messages, and hooks subset with bearer auth. |
| Typeform | `typeform-rest-api-overlay.json` | REST-shaped Typeform docs support a focused forms, responses, webhooks, and workspaces subset. |
| Unleashed Software | `unleashed-software-api-overlay.json` | API docs support a focused sales orders, stock on hand, products, customers, suppliers, purchase orders, and warehouses subset; signature calculation remains outside apitools. |
| Vero | `vero-track-api-overlay.json` | Track API docs support a focused users, events, tags, and unsubscribe subset with auth_token request parameters. |
| Webflow | `webflow-data-api-v2-overlay.json` | REST-shaped Data API docs support a focused sites, pages, collections, and CMS items subset. |
| WordPress | `wordpress-rest-api-overlay.json` | REST API docs support a focused posts, pages, media, users, categories, and tags subset; site host, plugins, and custom post types remain operator supplied. |
| Wufoo | `wufoo-api-v3-overlay.json` | REST-shaped API v3 docs support a focused forms, entries, reports, users, and webhooks subset with API-key Basic auth. |
| YOURLS | `yourls-api-overlay.json` | API docs support a focused shorturl, expand, stats, and URL stats action subset with signature query auth. |
| WhatsApp | `whatsapp-cloud-api-overlay.json` | Cloud API docs support a focused messages, media, message templates, phone-number metadata, and app subscription subset. |
| Wise | `wise-platform-api-overlay.json` | Platform API docs support a focused profiles, quotes, recipients, transfers, balances, statements, rates, and webhook subscriptions subset. |
| WooCommerce | `woocommerce-rest-api-overlay.json` | REST API docs support a focused products, orders, customers, coupons, refunds, reports, and webhooks subset; WordPress site host remains operator supplied. |
| Workable | `workable-api-overlay.json` | API docs support a focused jobs, candidates, members, recruiters, stages, scheduled events, and requisitions subset; account subdomain remains operator supplied. |
| Bitwarden | `bitwarden-public-api-overlay.json` | Public API docs support a focused organization groups, collections, members, events, and organization info subset; OAuth token exchange remains security metadata only. |
| Cisco Webex | `cisco-webex-api-overlay.json` | API docs support a focused rooms, messages, memberships, people, webhooks, attachment actions, and recordings subset. |
| Cortex | `cortex-api-overlay.json` | API guide docs support a focused analyzer, responder, job, report, and organization subset; instance host remains operator supplied. |
| Home Assistant | `home-assistant-rest-api-overlay.json` | REST API docs support a focused status, config, states, services, history, template, events, and camera proxy subset; local entity schemas remain instance-specific. |
| NetScaler ADC | `netscaler-adc-nitro-api-overlay.json` | NITRO API docs support a focused login, load-balancing, service, server, content-switching, and statistics subset; appliance host remains operator supplied. |
| Venafi | `venafi-api-overlay.json` | TLS Protect Cloud and Datacenter docs support a focused certificate, certificate request, application, preference, OAuth, request, and retrieve subset. |
| Wekan | `wekan-rest-api-overlay.json` | REST API wiki docs and OpenAPI generator docs support a focused login, user, board, list, card, and checklist subset; host and board schemas remain operator supplied. |
| Zammad | `zammad-api-overlay.json` | API docs support a focused ticket, user, group, organization, article, object attribute, and search subset; instance host remains operator supplied. |
| Humantic AI | `humantic-ai-api-overlay.json` | API docs support a focused user-profile create, retrieve, and update subset; profile analysis execution remains outside apitools. |
| Hunter | `hunter-api-overlay.json` | API docs support a focused domain search, email finder, email verifier, leads, campaigns, and account subset with query API-key auth. |
| LingvaNex | `lingvanex-api-overlay.json` | API docs and official example repository support a focused translation, detection, language, dictionary, and file-URL translation subset. |
| Mistral AI | `mistral-ai-api-overlay.json` | Rendered API specs support a focused chat, embeddings, OCR, files, batch jobs, agents, and models subset; model selection remains outside apitools. |
| Mindee | `mindee-api-overlay.json` | API docs support a focused receipt, invoice, and custom-model prediction subset across current and legacy token-header patterns. |
| Phantombuster | `phantombuster-api-overlay.json` | API reference docs and llms index support a focused agents, containers, scripts, and organization subset; agent execution remains outside apitools. |
| Plivo | `plivo-api-overlay.json` | REST-shaped Messaging and Voice API docs support a focused messages and calls subset with HTTP Basic auth. |
| UpLead | `uplead-api-overlay.json` | API docs support a focused person search, company search, and credits subset with Authorization header API-key auth. |
| Dropcontact | `dropcontact-api-overlay.json` | Developer docs support a focused batch enrichment and result retrieval subset with X-Access-Token auth. |
| Pushbullet | `pushbullet-api-overlay.json` | REST-shaped API docs support a focused pushes, upload request, devices, contacts, channels, and subscriptions subset with OAuth bearer auth. |
| Pushcut | `pushcut-api-overlay.json` | REST-shaped integration docs support a focused notifications, devices, and subscriptions subset with API-Key header auth. |
| Pushover | `pushover-api-overlay.json` | REST-shaped API docs support a focused messages, receipts, user validation, and sounds subset with token and user request parameters. |
| sms77 | `sms77-api-overlay.json` | REST-shaped seven.io API docs support a focused SMS, voice, balance, pricing, status, and contacts subset with X-Api-Key header auth. |

## No Endpoint Overlay

| Provider | Decision |
|---|---|
| Jina AI | Official OpenAPI documents are recorded for Search Foundation, Reader, and Search surfaces. DeepSearch OpenAPI probes require authorization, so M30 records only auth overlay metadata rather than a docs-derived endpoint overlay. |
| OpenAI | Official documented OpenAPI is recorded, so no docs-derived endpoint overlay is needed. |
| Perplexity | Official OpenAPI is recorded, so no docs-derived endpoint overlay is needed. |
| Peekalink | Official OpenAPI is recorded, so no docs-derived endpoint overlay is needed. |
| Linear | No OpenAPI-shaped endpoint overlay for now. Linear's public API is GraphQL with introspection; a single `POST /graphql` wrapper would hide operation semantics rather than improve OpenAPI-like metadata. |
| Monday.com | No OpenAPI-shaped endpoint overlay for now. Monday.com's platform API is GraphQL at a single endpoint; useful operation coverage should come from a GraphQL-aware classifier or schema/introspection workflow, not a REST-shaped overlay. |
| Odoo | No OpenAPI-shaped endpoint overlay for now. Odoo's official external API is model-oriented XML-RPC/JSON-RPC; a generic RPC wrapper would hide model semantics rather than improve REST-shaped metadata. |
| TimeSaved | No OpenAPI-shaped endpoint overlay. The frozen M26 item is an n8n workflow metadata helper, not an external provider API with HTTP endpoints. |
| SIGNL4 | Official OpenAPI is recorded for SIGNL4 API V1, so no docs-derived endpoint overlay is needed; auth overlay metadata records the reviewed X-S4-Api-Key requirement. |
| Vonage | Official Messages API OpenAPI is recorded, so no docs-derived endpoint overlay is needed. The n8n legacy SMS endpoint is not used as provider truth. |
| Zulip | Official REST API OpenAPI is recorded from the Zulip repository, so no docs-derived endpoint overlay is needed. |
| Gotify | Official Swagger/OpenAPI is recorded from the Gotify server repository, so no docs-derived endpoint overlay is needed. |
| Emelia | No OpenAPI-shaped endpoint overlay for now. Emelia's public API is GraphQL at a single endpoint; a generic `POST /graphql` wrapper would hide operation semantics rather than improve OpenAPI-like metadata. |
| Spotify | Official Web API OpenAPI is recorded, so no docs-derived endpoint overlay is needed. |
| Toggl Track | Official Track API Swagger/OpenAPI is recorded, so no docs-derived endpoint overlay is needed. |
| Orbit | No endpoint overlay for now. Orbit has shut down and the official docs endpoint was not reliably reachable during review; historical auth metadata is retained only for catalog inspection. |
| Oura | Official API v2 OpenAPI is recorded, so no docs-derived endpoint overlay is needed. |
| SyncroMSP | Official OpenAPI is recorded from the SyncroMSP API documentation, so no docs-derived endpoint overlay is needed. |
| TheHive | Official TheHive 4 OpenAPI is recorded, so no docs-derived endpoint overlay is needed. |
| TheHive Project | Official TheHive 5 OpenAPI is recorded, so no docs-derived endpoint overlay is needed. |
| E-goi | Official Marketing API v3 OpenAPI is recorded, so no docs-derived endpoint overlay is needed. |
| Mailcheck | Official OpenAPI is recorded from app.mailcheck.co, so no docs-derived endpoint overlay is needed. |
| Nextcloud | Official server OpenAPI is recorded from the developer manual, so no docs-derived endpoint overlay is needed; WebDAV remains protocol-specific metadata. |
| Philips Hue | No endpoint overlay for now. The official full API reference is login-gated and the available public Hue pages do not expose enough endpoint detail for a reviewed subset; unofficial OpenHue specs are not provider truth. |
| PostBin | No endpoint overlay for now. M34 review did not find stable official endpoint documentation or an official OpenAPI document for PostBin's public request-bin API. |
| Segment | No endpoint overlay for now. Official Segment Public API docs are rendered from an embedded OpenAPI 3.0.3 definition and expose a browser Download OpenAPI action, but M35 review did not find a stable standalone URL to register as a refreshable artifact. |
| Strava | Official Swagger/OpenAPI is recorded from developers.strava.com, so no docs-derived endpoint overlay is needed. |
| Zoho CRM | Official Zoho CRM v8.0 OpenAPI documents are recorded from the zoho/crm-oas repository, so no docs-derived endpoint overlay is needed. |
| uProc | No endpoint overlay for now. Official uProc tool pages describe API use through generated examples and images, but M35 review did not find stable official text endpoint documentation or an official OpenAPI document for a reviewed REST subset. |
