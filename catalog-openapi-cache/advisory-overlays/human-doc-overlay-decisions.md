# Human-Docs Endpoint Overlay Decisions

These OpenAPI-shaped advisory overlays are derived from official documentation
or official machine-readable sources. They are not official provider OpenAPI
documents and must not be treated as provider truth.

## Built

| Provider | Overlay | Reason |
|---|---|---|
| Acuity Scheduling | `acuity-scheduling-api-v1-overlay.json` | REST-shaped API v1 docs support a focused appointments, availability, appointment-types, calendars, and clients subset. |
| ActiveCampaign | `activecampaign-api-v3-overlay.json` | REST-shaped API v3 docs support a focused contacts, lists, campaigns, and deals subset; account host remains operator supplied. |
| Airtable | `airtable-web-api-overlay.json` | REST-shaped official Web API docs support a focused table/record subset. |
| BambooHR | `bamboohr-api-v1-overlay.json` | REST-shaped API docs support a focused employees, directory, fields, reports, and time-off subset; customer subdomain remains operator supplied. |
| Calendly | `calendly-public-api-overlay.json` | REST-shaped official public API docs support event type, event, webhook, and user endpoints. |
| Clockify | `clockify-api-v1-overlay.json` | REST-shaped API v1 docs support a focused users, workspaces, projects, clients, tags, and time-entries subset. |
| Contentful | `contentful-management-api-overlay.json` | REST-shaped Content Management API docs support a focused spaces, environments, entries, assets, and content-types subset. |
| Databricks | `databricks-workspace-rest-overlay.json` | REST-shaped workspace API docs support a focused jobs, clusters, workspace, and DBFS subset; host remains operator supplied. |
| Eventbrite | `eventbrite-platform-api-v3-overlay.json` | REST-shaped Platform API docs support a focused users, organizations, events, attendees, venues, and ticket-classes subset. |
| Freshdesk | `freshdesk-api-v2-overlay.json` | REST-shaped API v2 docs support a focused tickets, contacts, companies, and agents subset; account domain remains operator supplied. |
| Ghost | `ghost-admin-api-overlay.json` | REST-shaped Admin API docs support a focused posts, pages, tags, users, and members subset; admin host remains operator supplied. |
| Harvest | `harvest-api-v2-overlay.json` | REST-shaped API v2 docs support a focused users, company, clients, projects, time-entries, and invoices subset. |
| Help Scout | `help-scout-inbox-api-v2-overlay.json` | REST-shaped Inbox API v2 docs support a focused conversations, customers, mailboxes, and threads subset. |
| Iterable | `iterable-api-overlay.json` | REST-shaped API docs support a focused users, events, campaigns, lists, and templates subset. |
| Jenkins | `jenkins-remote-api-overlay.json` | Official Remote Access API docs support a small generic controller/job/queue subset; instance and plugin coverage remains variable. |
| Mailchimp | `mailchimp-marketing-api-overlay.json` | REST-shaped Marketing API docs support a focused lists, campaigns, and reports subset. |
| Mailjet | `mailjet-rest-api-overlay.json` | REST-shaped REST API docs support a focused contacts, list-recipients, campaigns, messages, and send subset. |
| OpenWeatherMap | `openweathermap-one-call-3-overlay.json` | One Call 3.0 docs support a narrow weather endpoint overlay. |
| Postmark | `postmark-api-overlay.json` | REST-shaped API docs and API Explorer support a focused email, template, bounce, and server subset with separate server/account token headers. |
| QuickBooks | `quickbooks-online-accounting-api-overlay.json` | REST-shaped Accounting API docs support a focused customer, invoice, payment, and company-info subset. |
| Salesforce | `salesforce-rest-core-overlay.json` | REST API docs support a focused sObject/query/composite subset; instance and API version remain operator supplied. |
| Sentry | `sentry-rest-api-overlay.json` | REST-shaped API docs support a focused organization, project, issue, event, and release subset. |
| ServiceNow | `servicenow-rest-api-overlay.json` | REST API Explorer and Table API docs support a focused instance/table/import subset. |
| Shopify | `shopify-admin-rest-overlay.json` | Admin REST docs support a focused products, orders, customers, webhooks, and inventory subset. |
| Splunk | `splunk-enterprise-rest-overlay.json` | REST API docs support a focused server, search jobs, results, and saved-search subset; product version remains review-sensitive. |
| Telegram | `telegram-bot-api-overlay.json` | Bot API docs support common bot methods; bot token is modeled as a server variable because OpenAPI security schemes cannot represent Telegram path-token auth exactly. |
| Todoist | `todoist-rest-api-v2-overlay.json` | REST API v2 docs support a focused projects, sections, tasks, and comments subset; docs now point developers toward newer unified API docs. |
| Typeform | `typeform-rest-api-overlay.json` | REST-shaped Typeform docs support a focused forms, responses, webhooks, and workspaces subset. |
| Webflow | `webflow-data-api-v2-overlay.json` | REST-shaped Data API docs support a focused sites, pages, collections, and CMS items subset. |

## No Endpoint Overlay

| Provider | Decision |
|---|---|
| Linear | No OpenAPI-shaped endpoint overlay for now. Linear's public API is GraphQL with introspection; a single `POST /graphql` wrapper would hide operation semantics rather than improve OpenAPI-like metadata. |
| Monday.com | No OpenAPI-shaped endpoint overlay for now. Monday.com's platform API is GraphQL at a single endpoint; useful operation coverage should come from a GraphQL-aware classifier or schema/introspection workflow, not a REST-shaped overlay. |
