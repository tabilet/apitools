package catalog

import (
	"sort"
	"strings"
	"unicode"
)

// N8nNodeGapClassification records how a local n8n node root should be treated
// during catalog expansion review.
type N8nNodeGapClassification string

const (
	N8nNodeAlreadyCovered            N8nNodeGapClassification = "already-covered-by-alias-or-category"
	N8nNodeProviderAPICandidate      N8nNodeGapClassification = "provider-api-candidate"
	N8nNodeGenericProtocolExcluded   N8nNodeGapClassification = "generic-protocol-connector-excluded-m50"
	N8nNodeLocalWorkflowUtility      N8nNodeGapClassification = "local-workflow-utility-excluded"
	N8nNodeProtocolFamilyCandidate   N8nNodeGapClassification = "graphql-or-protocol-family-candidate"
	N8nNodeInternalDirectoryExcluded N8nNodeGapClassification = "internal-directory-excluded"
)

// N8nNodeGapReportOptions controls n8n node-root gap classification.
type N8nNodeGapReportOptions struct {
	NodeRoots []string
	Providers []Provider
}

// N8nNodeGapReport summarizes local n8n node roots against catalog provider
// lookup keys. It is metadata-only and does not inspect node implementation,
// execute n8n workflows, or probe provider APIs.
type N8nNodeGapReport struct {
	TotalNodes  int                         `json:"total_nodes"`
	Summary     []N8nNodeGapSummary         `json:"summary,omitempty"`
	Rows        []N8nNodeGapRow             `json:"rows,omitempty"`
	FrozenBatch []N8nFrozenSourceBatchEntry `json:"frozen_batch,omitempty"`
}

// N8nNodeGapSummary records a deterministic classification count.
type N8nNodeGapSummary struct {
	Classification N8nNodeGapClassification `json:"classification"`
	Count          int                      `json:"count"`
}

// N8nNodeGapRow records one n8n node-root classification.
type N8nNodeGapRow struct {
	NodeRoot           string                   `json:"node_root"`
	Classification     N8nNodeGapClassification `json:"classification"`
	MatchedProviderID  string                   `json:"matched_provider_id,omitempty"`
	MatchedDisplayName string                   `json:"matched_display_name,omitempty"`
	MatchedBy          string                   `json:"matched_by,omitempty"`
	Rationale          string                   `json:"rationale,omitempty"`
}

// N8nFrozenSourceBatchEntry records the next source-review batch selected from
// n8n-visible services with plausible provider-owned OpenAPI or strong
// docs-derived overlay evidence.
type N8nFrozenSourceBatchEntry struct {
	NodeRoot          string `json:"node_root"`
	ProviderID        string `json:"provider_id"`
	DisplayName       string `json:"display_name"`
	SourceExpectation string `json:"source_expectation"`
	Rationale         string `json:"rationale"`
}

// BuildN8nNodeGapReport compares local n8n node roots against provider catalog
// lookup keys and M50 exclusion rules.
func BuildN8nNodeGapReport(options N8nNodeGapReportOptions) N8nNodeGapReport {
	providers := options.Providers
	if providers == nil {
		providers = BuiltInProviders()
	} else {
		providers = cloneProviders(providers)
		sortProviders(providers)
	}
	matcher := newProviderMatcher(providers)
	nodes := sortedUniqueNodeRoots(options.NodeRoots)
	rows := make([]N8nNodeGapRow, 0, len(nodes))
	counts := map[N8nNodeGapClassification]int{}
	for _, node := range nodes {
		row := classifyN8nNodeRoot(node, matcher)
		rows = append(rows, row)
		counts[row.Classification]++
	}
	summaryOrder := []N8nNodeGapClassification{
		N8nNodeAlreadyCovered,
		N8nNodeProviderAPICandidate,
		N8nNodeGenericProtocolExcluded,
		N8nNodeLocalWorkflowUtility,
		N8nNodeProtocolFamilyCandidate,
		N8nNodeInternalDirectoryExcluded,
	}
	summary := make([]N8nNodeGapSummary, 0, len(summaryOrder))
	for _, classification := range summaryOrder {
		summary = append(summary, N8nNodeGapSummary{Classification: classification, Count: counts[classification]})
	}
	return N8nNodeGapReport{
		TotalNodes:  len(rows),
		Summary:     summary,
		Rows:        rows,
		FrozenBatch: frozenN8nSourceBatch(),
	}
}

func classifyN8nNodeRoot(node string, matcher providerMatcher) N8nNodeGapRow {
	key := compactKey(node)
	if _, ok := n8nInternalDirectories[key]; ok {
		return N8nNodeGapRow{
			NodeRoot:       node,
			Classification: N8nNodeInternalDirectoryExcluded,
			Rationale:      "Internal directory entry, not an n8n provider node root.",
		}
	}
	if note, ok := n8nGenericProtocolNodes[key]; ok {
		return N8nNodeGapRow{
			NodeRoot:       node,
			Classification: N8nNodeGenericProtocolExcluded,
			Rationale:      note,
		}
	}
	if note, ok := n8nProtocolFamilyNodes[key]; ok {
		return N8nNodeGapRow{
			NodeRoot:       node,
			Classification: N8nNodeProtocolFamilyCandidate,
			Rationale:      note,
		}
	}
	if note, ok := n8nLocalWorkflowUtilityNodes[key]; ok {
		return N8nNodeGapRow{
			NodeRoot:       node,
			Classification: N8nNodeLocalWorkflowUtility,
			Rationale:      note,
		}
	}
	if provider, matchedBy, ok := matcher.find(node); ok {
		return N8nNodeGapRow{
			NodeRoot:           node,
			Classification:     N8nNodeAlreadyCovered,
			MatchedProviderID:  provider.ID,
			MatchedDisplayName: provider.DisplayName,
			MatchedBy:          matchedBy,
			Rationale:          "Matched built-in provider catalog ID, display name, alias, or curated category coverage.",
		}
	}
	return N8nNodeGapRow{
		NodeRoot:       node,
		Classification: N8nNodeProviderAPICandidate,
		Rationale:      "Provider-shaped n8n node root with no built-in provider catalog match; requires source review before promotion.",
	}
}

type providerMatcher struct {
	byKey map[string]providerMatch
}

type providerMatch struct {
	provider Provider
	matched  string
}

func newProviderMatcher(providers []Provider) providerMatcher {
	matcher := providerMatcher{byKey: map[string]providerMatch{}}
	for _, provider := range providers {
		for _, value := range append([]string{provider.ID, provider.DisplayName}, provider.Aliases...) {
			for _, key := range nodeLookupVariants(value) {
				if _, exists := matcher.byKey[key]; exists {
					continue
				}
				matcher.byKey[key] = providerMatch{provider: provider, matched: "provider-lookup-key"}
			}
		}
	}
	for node, providerID := range n8nCategoryCoverage {
		provider, ok := findProviderByID(providers, providerID)
		if !ok {
			continue
		}
		matcher.byKey[node] = providerMatch{provider: provider, matched: "category-coverage"}
	}
	return matcher
}

func (m providerMatcher) find(node string) (Provider, string, bool) {
	for _, key := range nodeLookupVariants(node) {
		if match, ok := m.byKey[key]; ok {
			return match.provider, match.matched, true
		}
	}
	return Provider{}, "", false
}

func findProviderByID(providers []Provider, id string) (Provider, bool) {
	for _, provider := range providers {
		if provider.ID == id {
			return provider, true
		}
	}
	return Provider{}, false
}

func sortedUniqueNodeRoots(nodes []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(nodes))
	for _, node := range nodes {
		node = strings.TrimSpace(node)
		if node == "" {
			continue
		}
		if _, exists := seen[node]; exists {
			continue
		}
		seen[node] = struct{}{}
		out = append(out, node)
	}
	sort.Strings(out)
	return out
}

func nodeLookupVariants(value string) []string {
	variants := []string{
		normalizeKey(value),
		compactKey(value),
		compactKey(splitIdentifier(value)),
	}
	out := make([]string, 0, len(variants))
	seen := map[string]struct{}{}
	for _, variant := range variants {
		if variant == "" {
			continue
		}
		if _, exists := seen[variant]; exists {
			continue
		}
		seen[variant] = struct{}{}
		out = append(out, variant)
	}
	return out
}

func compactKey(value string) string {
	var builder strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(value)) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			builder.WriteRune(r)
		}
	}
	return builder.String()
}

func splitIdentifier(value string) string {
	var builder strings.Builder
	var previous rune
	for i, r := range strings.TrimSpace(value) {
		if i > 0 && shouldSplitIdentifier(previous, r) {
			builder.WriteRune(' ')
		}
		builder.WriteRune(r)
		previous = r
	}
	return builder.String()
}

func shouldSplitIdentifier(previous, current rune) bool {
	if previous == 0 {
		return false
	}
	if unicode.IsLower(previous) && unicode.IsUpper(current) {
		return true
	}
	if unicode.IsDigit(previous) && unicode.IsLetter(current) {
		return true
	}
	if unicode.IsLetter(previous) && unicode.IsDigit(current) {
		return true
	}
	return false
}

var n8nInternalDirectories = map[string]string{
	"nodes": "internal directory",
}

var n8nCategoryCoverage = map[string]string{
	"aws":       "aws-s3",
	"facebook":  "facebook",
	"google":    "google-drive",
	"microsoft": "microsoft-graph",
	"s3":        "aws-s3",
}

var n8nGenericProtocolNodes = map[string]string{
	"amqp":          "Generic AMQP broker connector excluded by the M50 protocol boundary.",
	"cratedb":       "Database protocol connector excluded by the M50 protocol boundary.",
	"emailreadimap": "Generic IMAP mail protocol connector excluded by the M50 protocol boundary.",
	"emailsend":     "Generic SMTP/mail-send connector excluded by the M50 protocol boundary.",
	"ftp":           "Generic FTP protocol connector excluded by the M50 protocol boundary.",
	"git":           "Generic Git client behavior is not provider API metadata.",
	"kafka":         "Generic Kafka broker connector excluded by the M50 protocol boundary.",
	"ldap":          "Generic LDAP directory connector excluded by the M50 protocol boundary.",
	"mongodb":       "Database protocol connector excluded by the M50 protocol boundary.",
	"mqtt":          "Generic MQTT broker connector excluded by the M50 protocol boundary.",
	"mysql":         "Database protocol connector excluded by the M50 protocol boundary.",
	"oracle":        "Database protocol connector excluded by the M50 protocol boundary.",
	"postgres":      "Database protocol connector excluded by the M50 protocol boundary.",
	"questdb":       "Database protocol connector excluded by the M50 protocol boundary.",
	"rabbitmq":      "Generic RabbitMQ broker connector excluded by the M50 protocol boundary.",
	"redis":         "Key-value/database protocol connector excluded by the M50 protocol boundary.",
	"ssh":           "Generic SSH protocol connector excluded by the M50 protocol boundary.",
	"timescaledb":   "Database protocol connector excluded by the M50 protocol boundary.",
}

var n8nProtocolFamilyNodes = map[string]string{
	"graphql":     "Future GraphQL support should start from schema artifacts, not generic node runtime behavior.",
	"icalendar":   "Calendar feed parsing may become a source-family review item, but it is not provider OpenAPI metadata.",
	"rssfeedread": "RSS/Atom feed parsing may become a source-family review item, but it is not provider OpenAPI metadata.",
}

var n8nLocalWorkflowUtilityNodes = map[string]string{
	"aitransform":                  "Local n8n AI data-transform utility, not provider API metadata.",
	"code":                         "Local n8n code execution utility, not provider API metadata.",
	"comparedatasets":              "Local n8n data utility, not provider API metadata.",
	"compression":                  "Local n8n data utility, not provider API metadata.",
	"cron":                         "Local scheduling trigger, not provider API metadata.",
	"crypto":                       "Local n8n crypto utility, not provider API metadata.",
	"datatable":                    "Local n8n data-table utility, not provider API metadata.",
	"datetime":                     "Local n8n date/time utility, not provider API metadata.",
	"debughelper":                  "Local n8n debugging utility, not provider API metadata.",
	"dynamiccredentialcheck":       "Local n8n credential-check utility, not provider API metadata.",
	"e2etest":                      "Local n8n test utility, not provider API metadata.",
	"editimage":                    "Local n8n image-editing utility, not provider API metadata.",
	"errortrigger":                 "Local workflow trigger, not provider API metadata.",
	"evaluation":                   "Local n8n evaluation utility, not provider API metadata.",
	"executecommand":               "Local command execution utility, not provider API metadata.",
	"executeworkflow":              "Local n8n workflow execution utility, not provider API metadata.",
	"executiondata":                "Local n8n execution-data utility, not provider API metadata.",
	"files":                        "Local file utility, not provider API metadata.",
	"filter":                       "Local n8n item-filter utility, not provider API metadata.",
	"flow":                         "Local n8n control-flow utility, not provider API metadata.",
	"form":                         "Local n8n form trigger/utility, not provider API metadata.",
	"function":                     "Local n8n code utility, not provider API metadata.",
	"functionitem":                 "Local n8n code utility, not provider API metadata.",
	"html":                         "Local HTML utility, not provider API metadata.",
	"htmlextract":                  "Local HTML extraction utility, not provider API metadata.",
	"httprequest":                  "Generic HTTP utility node, not provider API metadata.",
	"if":                           "Local n8n control-flow utility, not provider API metadata.",
	"interval":                     "Local interval trigger, not provider API metadata.",
	"itemlists":                    "Local n8n item-list utility, not provider API metadata.",
	"jwt":                          "Local JWT utility, not provider API metadata.",
	"localfiletrigger":             "Local file trigger, not provider API metadata.",
	"manualtrigger":                "Local manual workflow trigger, not provider API metadata.",
	"markdown":                     "Local Markdown utility, not provider API metadata.",
	"merge":                        "Local n8n data merge utility, not provider API metadata.",
	"messageanagent":               "Local n8n agent-message utility, not provider API metadata.",
	"movebinarydata":               "Local binary-data utility, not provider API metadata.",
	"n8ntrainingcustomerdatastore": "n8n training/demo node, not provider API metadata.",
	"n8ntrainingcustomermessenger": "n8n training/demo node, not provider API metadata.",
	"n8ntrigger":                   "Local n8n trigger, not provider API metadata.",
	"noop":                         "Local no-op workflow utility, not provider API metadata.",
	"readbinaryfile":               "Local file utility, not provider API metadata.",
	"readbinaryfiles":              "Local file utility, not provider API metadata.",
	"readpdf":                      "Local PDF utility, not provider API metadata.",
	"renamekeys":                   "Local n8n item-transform utility, not provider API metadata.",
	"respondtowebhook":             "Local webhook response utility, not provider API metadata.",
	"schedule":                     "Local scheduling trigger, not provider API metadata.",
	"set":                          "Local n8n item-transform utility, not provider API metadata.",
	"simulate":                     "Local n8n simulation utility, not provider API metadata.",
	"splitinbatches":               "Local n8n control-flow utility, not provider API metadata.",
	"spreadsheetfile":              "Local spreadsheet-file utility, not provider API metadata.",
	"ssetrigger":                   "Generic SSE trigger, not provider API metadata.",
	"stickynote":                   "Local workflow note utility, not provider API metadata.",
	"stopanderror":                 "Local n8n control-flow utility, not provider API metadata.",
	"switch":                       "Local n8n control-flow utility, not provider API metadata.",
	"totp":                         "Local TOTP utility, not provider API metadata.",
	"transform":                    "Local n8n transform utility, not provider API metadata.",
	"wait":                         "Local workflow timing utility, not provider API metadata.",
	"webhook":                      "Generic webhook trigger, not provider API metadata.",
	"workflowtrigger":              "Local workflow trigger, not provider API metadata.",
	"writebinaryfile":              "Local file utility, not provider API metadata.",
	"xml":                          "Local XML utility, not provider API metadata.",
}

func frozenN8nSourceBatch() []N8nFrozenSourceBatchEntry {
	return []N8nFrozenSourceBatchEntry{
		{
			NodeRoot:          "Chargebee",
			ProviderID:        "chargebee",
			DisplayName:       "Chargebee",
			SourceExpectation: "provider-owned OpenAPI",
			Rationale:         "Official Chargebee OpenAPI repository is available; review and register current API/Product Catalog variants.",
		},
		{
			NodeRoot:          "Mailgun",
			ProviderID:        "mailgun",
			DisplayName:       "Mailgun",
			SourceExpectation: "provider-owned OpenAPI",
			Rationale:         "Official Mailgun API reference exposes OpenAPI/OAS documentation that should be reviewed for a stable downloadable artifact.",
		},
		{
			NodeRoot:          "Mattermost",
			ProviderID:        "mattermost",
			DisplayName:       "Mattermost",
			SourceExpectation: "provider-owned OpenAPI",
			Rationale:         "Mattermost documents OpenAPI-backed REST API reference material in the server repository/API documentation path.",
		},
		{
			NodeRoot:          "Paddle",
			ProviderID:        "paddle",
			DisplayName:       "Paddle",
			SourceExpectation: "strong docs-derived overlay",
			Rationale:         "Official Paddle API reference is resource-oriented and suitable for a reviewed advisory overlay if no stable OpenAPI download is found.",
		},
		{
			NodeRoot:          "Plivo",
			ProviderID:        "plivo",
			DisplayName:       "Plivo",
			SourceExpectation: "strong docs-derived overlay",
			Rationale:         "Official Plivo API docs expose stable REST resources and auth guidance suitable for overlay review.",
		},
		{
			NodeRoot:          "PostHog",
			ProviderID:        "posthog",
			DisplayName:       "PostHog",
			SourceExpectation: "provider-owned OpenAPI",
			Rationale:         "Official PostHog docs repository references an OpenAPI schema endpoint; review cloud/self-hosted host handling before registration.",
		},
		{
			NodeRoot:          "Postmark",
			ProviderID:        "postmark",
			DisplayName:       "Postmark",
			SourceExpectation: "strong docs-derived overlay",
			Rationale:         "Official Postmark developer API documentation is stable and resource-oriented enough for a reviewed advisory overlay.",
		},
		{
			NodeRoot:          "Rocketchat",
			ProviderID:        "rocket-chat",
			DisplayName:       "Rocket.Chat",
			SourceExpectation: "provider-owned OpenAPI or strong docs-derived overlay",
			Rationale:         "Official Rocket.Chat API docs and release notes indicate OpenAPI documentation coverage; review current downloadable source state.",
		},
		{
			NodeRoot:          "Vonage",
			ProviderID:        "vonage",
			DisplayName:       "Vonage",
			SourceExpectation: "provider-owned OpenAPI",
			Rationale:         "Vonage developer docs state APIs have OpenAPI descriptions; review product-specific spec URLs and registration shape.",
		},
		{
			NodeRoot:          "WooCommerce",
			ProviderID:        "woocommerce",
			DisplayName:       "WooCommerce",
			SourceExpectation: "strong docs-derived overlay",
			Rationale:         "Official WooCommerce REST API documentation is stable and WordPress-site scoped; review overlay boundaries for instance-specific hosts.",
		},
	}
}
