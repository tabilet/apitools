# Advisory Overlay Linked-Docs Review

This ledger records M61 decisions made while reviewing official links cited by
docs-derived advisory overlays. Advisory overlays remain non-official OpenAPI
metadata and should not be treated as provider truth.

## M61 Batch A

| Provider | Overlay | Decision | Notes |
|---|---|---|---|
| Airtable | `airtable-web-api-overlay.json` | Expanded | The support Webhooks API overview links to official Webhooks API developer docs and describes real-time base-change notifications. Added conservative webhook management and payload retrieval operations alongside the existing table/record and base-schema operations. |
| Contentful | `contentful-management-api-overlay.json` | No add | The linked Content Delivery API is official and workflow-relevant, but it uses a different host, read-only semantics, and delivery-token behavior over paths that overlap the Content Management API. Adding it to the same Management API overlay would conflate API surfaces; keep the source ref for provenance and defer a separate delivery-focused overlay if needed. |
