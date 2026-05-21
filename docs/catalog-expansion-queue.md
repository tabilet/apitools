# Catalog Expansion Queue

Future catalog expansion should remain source-first and batch-frozen before
implementation begins.

## Candidate Inputs

- Current OpenUdon workflow needs.
- n8n node presence as a priority signal only, not runtime compatibility
  evidence.
- Provider-owned OpenAPI, Swagger, OpenAPI index, Smithy, Discovery, Stone, or
  other machine-readable sources.
- Official human docs only when they expose enough endpoint instructions for a
  reviewed advisory overlay.

## Next Batch Shape

Prefer a small batch of 8-12 services with at least one of these properties:

- official OpenAPI/Swagger can be downloaded and validated or classified as
  parseable-invalid review evidence;
- official non-OpenAPI machine specs are available and fit the protocol
  classifications already tracked by the catalog;
- official human docs expose a small REST-shaped subset suitable for an
  advisory endpoint overlay.

Avoid broad platform surfaces until there is a clear source and validation
strategy. GraphQL-first services should wait for GraphQL protocol support
instead of being forced into REST-shaped overlays.

## M51 Frozen Source-Review Batch

`go run ./cmd/apitools catalog n8n-gap-report --nodes-dir
../n8n/packages/nodes-base/nodes` freezes the next source-review batch while
keeping n8n node presence as priority-only evidence. The current batch is
limited to n8n-visible services where provider-owned OpenAPI evidence or strong
official docs-derived overlay evidence is plausible enough to justify review:

| n8n node | Provider | Expected source path | Review focus |
|---|---|---|---|
| `Chargebee` | `chargebee` | Provider-owned OpenAPI | Review and register current Chargebee OpenAPI API/Product Catalog variants. |
| `Mailgun` | `mailgun` | Provider-owned OpenAPI | Review Mailgun's OpenAPI/OAS documentation for a stable downloadable artifact. |
| `Mattermost` | `mattermost` | Provider-owned OpenAPI | Review Mattermost server/API documentation OpenAPI source and versioning. |
| `Paddle` | `paddle` | Strong docs-derived overlay | Review official resource-oriented API docs and use an advisory overlay if no stable OpenAPI download is available. |
| `Plivo` | `plivo` | Strong docs-derived overlay | Review official REST docs and auth guidance for a useful endpoint overlay subset. |
| `PostHog` | `posthog` | Provider-owned OpenAPI | Review the official docs/schema endpoint and cloud/self-hosted host handling. |
| `Postmark` | `postmark` | Strong docs-derived overlay | Review official developer API docs for endpoint and token-header coverage. |
| `Rocketchat` | `rocket-chat` | Provider-owned OpenAPI or strong docs-derived overlay | Review current Rocket.Chat API/OpenAPI documentation coverage and downloadable source state. |
| `Vonage` | `vonage` | Provider-owned OpenAPI | Review product-specific Vonage OpenAPI descriptions and registration shape. |
| `WooCommerce` | `woocommerce` | Strong docs-derived overlay | Review official WordPress-site-scoped REST docs and instance-host boundaries. |

The gap report currently finds no uncovered provider-shaped n8n node roots
after alias/category matching. Remaining non-covered roots are M50 protocol
connector exclusions, local workflow utilities, or future protocol-family
signals such as GraphQL/RSS/iCalendar.

## Required Work Per Service

- Add or update candidate metadata.
- Add durable provider metadata with source refs, availability, user OpenAPI
  need, quirks, and verification date.
- Register downloadable official review artifacts when refresh succeeds or is
  parseable-invalid.
- Add auth/security classification or advisory security overlay metadata.
- Add a docs-derived endpoint overlay only when official docs support a useful
  reviewed subset and the overlay would not misrepresent the API model.
- Run catalog stats, refresh-report, catalog quality, Go tests, vet, and diff
  checks before committing the batch.
