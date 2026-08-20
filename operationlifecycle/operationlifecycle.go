package operationlifecycle

import (
	"path"
	"slices"
	"strings"
	"unicode"

	"github.com/OpenUdon/apitools"
)

const (
	// MinimumSiblingScore is the minimum score required to select a lifecycle
	// sibling.
	MinimumSiblingScore = 60
	// HTTPVerbScore rewards an HTTP method that directly represents the role.
	HTTPVerbScore = 45
	// OperationNameScore rewards an operation name that names the role.
	OperationNameScore = 35
	// PatchMethodPreferenceScore prefers PATCH among otherwise valid updates.
	PatchMethodPreferenceScore = 8
	// PatchNamePreferenceScore rewards update operations explicitly named patch.
	PatchNamePreferenceScore = 6
	// FamilyMatchScore rewards operation-family agreement with the seed.
	FamilyMatchScore = 30
	// PathMatchScore rewards a matching collection/item path pair.
	PathMatchScore = 25
	// HighConfidenceScore is the minimum score reported as high confidence.
	HighConfidenceScore = 90
	// MediumConfidenceScore is the minimum score reported as medium confidence.
	MediumConfidenceScore = 70
)

// Options configures lifecycle expansion.
type Options struct {
	Goal         string
	DesiredState bool
}

// Expansion is one conservative lifecycle-role expansion.
type Expansion struct {
	SeedOperationID string          `json:"seed_operation_id,omitempty"`
	FamilyKey       string          `json:"family_key,omitempty"`
	Roles           []RoleCandidate `json:"roles,omitempty"`
	Diagnostics     []Diagnostic    `json:"diagnostics,omitempty"`
}

// RoleCandidate associates one API operation with a lifecycle role.
type RoleCandidate struct {
	Role       string                    `json:"role"`
	Operation  apitools.OperationSummary `json:"operation"`
	Confidence string                    `json:"confidence,omitempty"`
	Reason     string                    `json:"reason,omitempty"`
}

// Diagnostic explains why a possible lifecycle role was omitted.
type Diagnostic struct {
	Code     string `json:"code,omitempty"`
	Severity string `json:"severity,omitempty"`
	Message  string `json:"message,omitempty"`
}

type candidateScore struct {
	operation apitools.OperationSummary
	score     int
	reason    string
}

// Expand returns a conservative lifecycle expansion for seed. It only selects
// sibling operations when one same-source candidate is clearly stronger for a
// role; ambiguous role matches are diagnosed and omitted.
func Expand(operations []apitools.OperationSummary, seed apitools.OperationSummary, opts Options) Expansion {
	seed = normalizeSeed(operations, seed)
	seedID := operationID(seed)
	out := Expansion{SeedOperationID: seedID, FamilyKey: familyKey(seed)}
	if seedID == "" {
		out.Diagnostics = append(out.Diagnostics, Diagnostic{Code: "operation_lifecycle.seed_missing", Severity: "warning", Message: "seed operation is empty"})
		return out
	}
	siblings := map[string]candidateScore{}
	for _, role := range []string{"read", "update", "delete"} {
		if role == "update" && !goalWantsUpdate(opts.Goal) {
			continue
		}
		match, ok, diag := bestSibling(operations, seed, role)
		if diag.Code != "" {
			out.Diagnostics = append(out.Diagnostics, diag)
		}
		if ok {
			siblings[role] = match
		}
	}
	primaryRole := primaryRole(seed, opts, len(siblings) > 0)
	out.Roles = append(out.Roles, RoleCandidate{Role: primaryRole, Operation: seed, Confidence: "high", Reason: "selected seed operation"})
	if primaryRole == "read" {
		return out
	}
	for _, role := range []string{"read", "update", "delete"} {
		if role == primaryRole {
			continue
		}
		match, ok := siblings[role]
		if !ok {
			continue
		}
		out.Roles = append(out.Roles, RoleCandidate{Role: role, Operation: match.operation, Confidence: confidence(match.score), Reason: match.reason})
	}
	return out
}

func normalizeSeed(operations []apitools.OperationSummary, seed apitools.OperationSummary) apitools.OperationSummary {
	if operationID(seed) == "" {
		return seed
	}
	for _, operation := range operations {
		if sameOperation(operation, seed) {
			return operation
		}
	}
	return seed
}

func primaryRole(seed apitools.OperationSummary, opts Options, expanded bool) string {
	if opts.DesiredState && expanded {
		switch strings.ToUpper(strings.TrimSpace(seed.Method)) {
		case "POST":
			return "create"
		case "PUT":
			if operationHasAny(seed, "create", "createorupdate", "insert") {
				return "create"
			}
			return "update"
		case "PATCH":
			return "update"
		case "GET", "HEAD":
			return "read"
		case "DELETE":
			return "delete"
		}
	}
	return methodRole(seed)
}

func methodRole(op apitools.OperationSummary) string {
	switch strings.ToUpper(strings.TrimSpace(op.Method)) {
	case "GET", "HEAD":
		return "read"
	case "DELETE":
		return "delete"
	case "POST":
		return "post"
	case "PUT":
		return "put"
	case "PATCH":
		return "patch"
	default:
		return "post"
	}
}

func bestSibling(operations []apitools.OperationSummary, seed apitools.OperationSummary, role string) (candidateScore, bool, Diagnostic) {
	var matches []candidateScore
	for _, operation := range operations {
		if sameOperation(operation, seed) || !sameSource(operation, seed) {
			continue
		}
		score, reason := siblingScore(operation, seed, role)
		if score < MinimumSiblingScore {
			continue
		}
		matches = append(matches, candidateScore{operation: operation, score: score, reason: reason})
	}
	if len(matches) == 0 {
		return candidateScore{}, false, Diagnostic{}
	}
	slices.SortStableFunc(matches, func(a, b candidateScore) int {
		if a.score != b.score {
			return b.score - a.score
		}
		return strings.Compare(operationID(a.operation), operationID(b.operation))
	})
	if len(matches) > 1 && matches[0].score == matches[1].score {
		return candidateScore{}, false, Diagnostic{Code: "operation_lifecycle.ambiguous_" + role, Severity: "warning", Message: "multiple same-source operations match lifecycle role " + role}
	}
	return matches[0], true, Diagnostic{}
}

func siblingScore(op, seed apitools.OperationSummary, role string) (int, string) {
	if !lifecyclePathsMatch(seed, op, role) {
		return 0, ""
	}
	if role == "read" && operationHasAny(op, "list") {
		return 0, ""
	}
	if (role == "update" || role == "delete") && operationHasAny(op, "collection") {
		return 0, ""
	}
	score := 0
	var reasons []string
	if verbMatchesRole(op.Method, role) {
		score += HTTPVerbScore
		reasons = append(reasons, "HTTP verb matches "+role)
	}
	if operationNameMatchesRole(op, role) {
		score += OperationNameScore
		reasons = append(reasons, "operation id/name matches "+role)
	}
	if role == "update" && strings.EqualFold(op.Method, "PATCH") {
		score += PatchMethodPreferenceScore
		reasons = append(reasons, "PATCH update preferred")
	}
	if role == "update" && operationHasAny(op, "patch") {
		score += PatchNamePreferenceScore
		reasons = append(reasons, "patch operation preferred")
	}
	if sameFamily(seed, op) {
		score += FamilyMatchScore
		reasons = append(reasons, "operation family matches seed")
	}
	score += PathMatchScore
	reasons = append(reasons, "collection/item path matches seed")
	return score, strings.Join(reasons, "; ")
}

func verbMatchesRole(method, role string) bool {
	switch role {
	case "read":
		return strings.EqualFold(method, "GET") || strings.EqualFold(method, "HEAD")
	case "update":
		return strings.EqualFold(method, "PUT") || strings.EqualFold(method, "PATCH")
	case "delete":
		return strings.EqualFold(method, "DELETE")
	default:
		return false
	}
}

func operationNameMatchesRole(op apitools.OperationSummary, role string) bool {
	switch role {
	case "read":
		return operationHasAny(op, "get", "read", "show", "describe")
	case "update":
		return operationHasAny(op, "update", "patch", "replace", "createorupdate", "put")
	case "delete":
		return operationHasAny(op, "delete", "remove")
	default:
		return false
	}
}

func operationHasAny(op apitools.OperationSummary, terms ...string) bool {
	tokens := operationTokens(op)
	for _, term := range terms {
		if tokens[strings.ToLower(term)] {
			return true
		}
	}
	return false
}

func sameSource(a, b apitools.OperationSummary) bool {
	aSource, bSource := sourceID(a), sourceID(b)
	return aSource != "" && aSource == bSource
}

func sourceID(op apitools.OperationSummary) string {
	return strings.TrimSpace(firstNonEmpty(op.DocumentRelativePath, op.DocumentPath, op.DocumentURL, op.DocumentName))
}

func sameOperation(a, b apitools.OperationSummary) bool {
	aID, bID := operationID(a), operationID(b)
	return aID != "" && aID == bID && sourceID(a) == sourceID(b)
}

func operationID(op apitools.OperationSummary) string {
	return strings.TrimSpace(firstNonEmpty(op.OperationID, op.ID))
}

func familyKey(op apitools.OperationSummary) string { return strings.Join(familyTokens(op), ".") }

func sameFamily(a, b apitools.OperationSummary) bool {
	aTokens, bTokens := familyTokens(a), familyTokens(b)
	if len(aTokens) == 0 || len(bTokens) == 0 {
		return false
	}
	bSet := map[string]bool{}
	for _, token := range bTokens {
		bSet[token] = true
	}
	common := 0
	for _, token := range aTokens {
		if bSet[token] {
			common++
		}
	}
	return common >= min(2, min(len(aTokens), len(bTokens)))
}

func familyTokens(op apitools.OperationSummary) []string {
	raw := operationTokens(op)
	var out []string
	for token := range raw {
		if lifecycleWord(token) || token == "v1" || token == "v2" || token == "v3" || token == "api" {
			continue
		}
		out = append(out, token)
	}
	slices.Sort(out)
	return out
}

func operationTokens(op apitools.OperationSummary) map[string]bool {
	text := operationID(op) + " " + op.Summary + " " + strings.Join(op.Tags, " ")
	parts := strings.FieldsFunc(text, func(r rune) bool {
		return r == '_' || r == '-' || r == '.' || r == '/' || r == ':' || unicode.IsSpace(r)
	})
	out := map[string]bool{}
	for _, part := range parts {
		for _, token := range splitCamel(part) {
			token = strings.ToLower(strings.TrimSpace(token))
			if token != "" {
				out[token] = true
			}
		}
	}
	joined := strings.ToLower(strings.NewReplacer("_", "", "-", "", ".", "").Replace(operationID(op)))
	if strings.Contains(joined, "createorupdate") {
		out["createorupdate"], out["create"], out["update"] = true, true, true
	}
	return out
}

func splitCamel(value string) []string {
	var out []string
	start := 0
	runes := []rune(value)
	for i := 1; i < len(runes); i++ {
		if unicode.IsUpper(runes[i]) && (unicode.IsLower(runes[i-1]) || (i+1 < len(runes) && unicode.IsLower(runes[i+1]))) {
			out = append(out, string(runes[start:i]))
			start = i
		}
	}
	return append(out, string(runes[start:]))
}

func lifecycleWord(token string) bool {
	switch token {
	case "create", "insert", "post", "put", "get", "read", "show", "describe", "list", "update", "patch", "replace", "delete", "remove", "createorupdate":
		return true
	default:
		return false
	}
}

func lifecyclePathsMatch(seed, operation apitools.OperationSummary, role string) bool {
	seedPath := normalizePath(seed.Path, isGoogleDiscovery(seed))
	opPath := normalizePath(operation.Path, isGoogleDiscovery(operation))
	if seedPath == "" || opPath == "" {
		return false
	}
	if seedPath == opPath {
		return role != "read" && role != "update" && role != "delete" || hasPathParameter(opPath)
	}
	seedBase, opBase := collectionPath(seedPath), collectionPath(opPath)
	if seedBase == "" || opBase == "" || seedBase != opBase {
		return false
	}
	return role != "read" && role != "update" && role != "delete" || hasPathParameter(opPath)
}

func isGoogleDiscovery(op apitools.OperationSummary) bool {
	return strings.EqualFold(strings.TrimSpace(op.Extensions["x-uws-source-kind"]), apitools.APISourceKindGoogleDiscovery)
}

func normalizePath(value string, stripDiscoveryUpload bool) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	value = "/" + strings.Trim(path.Clean("/"+value), "/")
	if stripDiscoveryUpload {
		if value == "/upload" {
			return "/"
		}
		value = strings.TrimPrefix(value, "/upload/")
		if !strings.HasPrefix(value, "/") {
			value = "/" + value
		}
	}
	return value
}

func collectionPath(value string) string {
	parts := strings.Split(strings.Trim(value, "/"), "/")
	var out []string
	for _, part := range parts {
		if !strings.HasPrefix(part, "{") || !strings.HasSuffix(part, "}") {
			out = append(out, part)
		}
	}
	if len(out) == 0 {
		return "/"
	}
	return "/" + strings.Join(out, "/")
}

func hasPathParameter(value string) bool {
	for _, part := range strings.Split(value, "/") {
		if strings.HasPrefix(part, "{") && strings.HasSuffix(part, "}") {
			return true
		}
	}
	return false
}

func confidence(score int) string {
	switch {
	case score >= HighConfidenceScore:
		return "high"
	case score >= MediumConfidenceScore:
		return "medium"
	default:
		return "low"
	}
}

func goalWantsUpdate(goal string) bool {
	goal = strings.ToLower(goal)
	for _, word := range []string{"update", "updates", "updated", "patch", "patches", "modify", "modifies", "replace", "supports update"} {
		if strings.Contains(goal, word) {
			return true
		}
	}
	return false
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}
