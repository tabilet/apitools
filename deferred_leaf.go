package apitools

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// LeafOptions configures a review-only leaf adapter.
type LeafOptions struct {
	Name                    string                  `json:"name,omitempty"`
	Source                  string                  `json:"source,omitempty"`
	DeferredExecutionPolicy DeferredExecutionPolicy `json:"deferred_execution_policy,omitempty"`
}

// DeferredExecutionPolicy records the generic safety boundary for a leaf.
type DeferredExecutionPolicy struct {
	ReviewOnly            bool     `json:"review_only,omitempty"`
	RuntimeDeferred       bool     `json:"runtime_deferred,omitempty"`
	DirectExecutionDenied bool     `json:"direct_execution_denied,omitempty"`
	Notes                 []string `json:"notes,omitempty"`
}

// LeafAdapter is an embeddable concrete object for review-only,
// runtime-deferred leaf behavior.
type LeafAdapter struct {
	Name                    string                  `json:"name,omitempty"`
	Source                  string                  `json:"source,omitempty"`
	ArtifactSet             ArtifactSet             `json:"artifact_set,omitempty"`
	ReviewPackage           ReviewPackage           `json:"review_package,omitempty"`
	DeferredExecutionPolicy DeferredExecutionPolicy `json:"deferred_execution_policy,omitempty"`
}

// ReviewPackage is prompt-safe review metadata derived from an ArtifactSet.
type ReviewPackage struct {
	Name                    string                  `json:"name,omitempty"`
	Source                  string                  `json:"source,omitempty"`
	Artifacts               []ArtifactReview        `json:"artifacts,omitempty"`
	Diagnostics             []Diagnostic            `json:"diagnostics,omitempty"`
	ReadinessIssues         []ReadinessIssue        `json:"readiness_issues,omitempty"`
	SymbolicBindings        []SymbolicBinding       `json:"symbolic_bindings,omitempty"`
	BindingNames            []string                `json:"binding_names,omitempty"`
	Assumptions             []Assumption            `json:"assumptions,omitempty"`
	QuestionPlan            QuestionPlan            `json:"question_plan,omitempty"`
	TranscriptSummary       []string                `json:"transcript_summary,omitempty"`
	CredentialAudit         BindingAudit            `json:"credential_audit,omitempty"`
	RequiredReviewActions   []string                `json:"required_review_actions,omitempty"`
	DeferredExecutionPolicy DeferredExecutionPolicy `json:"deferred_execution_policy,omitempty"`
}

// ArtifactReview describes an artifact without duplicating its content.
type ArtifactReview struct {
	Path      string `json:"path"`
	MediaType string `json:"media_type,omitempty"`
	SizeBytes int    `json:"size_bytes,omitempty"`
}

// BindingAudit reports symbolic bindings and literal credential findings.
type BindingAudit struct {
	DeclaredSymbolicBindings     []string     `json:"declared_symbolic_bindings,omitempty"`
	LiteralCredentialDiagnostics []Diagnostic `json:"literal_credential_diagnostics,omitempty"`
}

// NewLeafAdapter returns an embeddable adapter with review metadata derived
// from the supplied artifact set. It does not bind credentials or execute APIs.
func NewLeafAdapter(set ArtifactSet, opts LeafOptions) LeafAdapter {
	policy := opts.DeferredExecutionPolicy
	if !policy.ReviewOnly && !policy.RuntimeDeferred && !policy.DirectExecutionDenied && len(policy.Notes) == 0 {
		policy = defaultDeferredExecutionPolicy()
	}
	leaf := LeafAdapter{
		Name:                    strings.TrimSpace(opts.Name),
		Source:                  strings.TrimSpace(opts.Source),
		ArtifactSet:             cloneArtifactSet(set),
		DeferredExecutionPolicy: policy,
	}
	leaf.ReviewPackage = leaf.MinimumReviewPackage()
	return leaf
}

func defaultDeferredExecutionPolicy() DeferredExecutionPolicy {
	return DeferredExecutionPolicy{
		ReviewOnly:            true,
		RuntimeDeferred:       true,
		DirectExecutionDenied: true,
		Notes: []string{
			"Artifacts are review-only until a caller-specific renderer validates them.",
			"Credential values must be supplied by a trusted runtime binding layer.",
			"apitools does not execute APIs or workflows.",
		},
	}
}

// ArtifactPaths returns stable artifact paths.
func (leaf LeafAdapter) ArtifactPaths() []string {
	paths := make([]string, 0, len(leaf.ArtifactSet.Artifacts))
	for _, artifact := range leaf.ArtifactSet.Artifacts {
		if strings.TrimSpace(artifact.Path) != "" {
			paths = append(paths, artifact.Path)
		}
	}
	sort.Strings(paths)
	return paths
}

// FindArtifact returns the first artifact matching path.
func (leaf LeafAdapter) FindArtifact(path string) (Artifact, bool) {
	path = strings.TrimSpace(path)
	for _, artifact := range leaf.ArtifactSet.Artifacts {
		if artifact.Path == path {
			return artifact, true
		}
	}
	return Artifact{}, false
}

// WithArtifact returns a copy of leaf with artifact added or replaced by path.
func (leaf LeafAdapter) WithArtifact(artifact Artifact) LeafAdapter {
	next := leaf
	next.ArtifactSet = cloneArtifactSet(leaf.ArtifactSet)
	replaced := false
	for i := range next.ArtifactSet.Artifacts {
		if next.ArtifactSet.Artifacts[i].Path == artifact.Path {
			next.ArtifactSet.Artifacts[i] = cloneArtifact(artifact)
			replaced = true
			break
		}
	}
	if !replaced {
		next.ArtifactSet.Artifacts = append(next.ArtifactSet.Artifacts, cloneArtifact(artifact))
	}
	SortArtifacts(next.ArtifactSet)
	next.ReviewPackage = next.MinimumReviewPackage()
	return next
}

// ReviewArtifact returns prompt-safe metadata for one artifact.
func (leaf LeafAdapter) ReviewArtifact(path string) (ArtifactReview, bool) {
	artifact, ok := leaf.FindArtifact(path)
	if !ok {
		return ArtifactReview{}, false
	}
	return artifactReview(artifact), true
}

// WithReviewArtifact returns a copy with review metadata added or replaced.
func (leaf LeafAdapter) WithReviewArtifact(review ArtifactReview) LeafAdapter {
	next := leaf
	next.ReviewPackage = cloneReviewPackage(leaf.MinimumReviewPackage())
	replaced := false
	for i := range next.ReviewPackage.Artifacts {
		if next.ReviewPackage.Artifacts[i].Path == review.Path {
			next.ReviewPackage.Artifacts[i] = review
			replaced = true
			break
		}
	}
	if !replaced {
		next.ReviewPackage.Artifacts = append(next.ReviewPackage.Artifacts, review)
	}
	sort.SliceStable(next.ReviewPackage.Artifacts, func(i, j int) bool {
		return next.ReviewPackage.Artifacts[i].Path < next.ReviewPackage.Artifacts[j].Path
	})
	return next
}

// ErrorDiagnostics returns error-severity diagnostics.
func (leaf LeafAdapter) ErrorDiagnostics() []Diagnostic {
	var out []Diagnostic
	for _, diagnostic := range leaf.ArtifactSet.Diagnostics {
		if strings.EqualFold(diagnostic.Severity, "error") {
			out = append(out, diagnostic)
		}
	}
	out = append(out, leaf.CredentialValueDiagnostics()...)
	return out
}

// BlockingReadinessIssues returns readiness issues that block rendering or use.
func (leaf LeafAdapter) BlockingReadinessIssues() []ReadinessIssue {
	var out []ReadinessIssue
	for _, issue := range leaf.ArtifactSet.ReadinessIssues {
		if blockingSeverity(issue.Severity) {
			out = append(out, issue)
		}
	}
	return out
}

// HasBlockingIssues reports whether diagnostics, readiness, or credentials
// require review before caller-specific rendering.
func (leaf LeafAdapter) HasBlockingIssues() bool {
	return len(leaf.ErrorDiagnostics()) > 0 || len(leaf.BlockingReadinessIssues()) > 0
}

// FormatDiagnostics renders diagnostics and readiness issues for plain-text
// review surfaces.
func (leaf LeafAdapter) FormatDiagnostics() string {
	var b strings.Builder
	for _, diagnostic := range leaf.ArtifactSet.Diagnostics {
		fmt.Fprintf(&b, "- `%s` %s", firstNonEmpty(diagnostic.Severity, "info"), diagnostic.Message)
		if diagnostic.Code != "" {
			fmt.Fprintf(&b, " (`%s`)", diagnostic.Code)
		}
		if diagnostic.Path != "" {
			fmt.Fprintf(&b, " at `%s`", diagnostic.Path)
		}
		if diagnostic.Remediation != "" {
			fmt.Fprintf(&b, " %s", diagnostic.Remediation)
		}
		b.WriteByte('\n')
	}
	for _, issue := range leaf.ArtifactSet.ReadinessIssues {
		fmt.Fprintf(&b, "- `%s` %s", firstNonEmpty(issue.Severity, "info"), issue.Message)
		if issue.Code != "" {
			fmt.Fprintf(&b, " (`%s`)", issue.Code)
		}
		if issue.Path != "" {
			fmt.Fprintf(&b, " at `%s`", issue.Path)
		}
		if issue.Remediation != "" {
			fmt.Fprintf(&b, " %s", issue.Remediation)
		}
		b.WriteByte('\n')
	}
	return strings.TrimRight(b.String(), "\n")
}

// RequiredBindings returns declared symbolic runtime bindings only.
func (leaf LeafAdapter) RequiredBindings() []SymbolicBinding {
	byName := map[string]SymbolicBinding{}
	for _, binding := range leaf.ArtifactSet.SymbolicBindings {
		name := strings.TrimSpace(binding.Name)
		if name != "" {
			binding.Name = name
			byName[name] = binding
		}
	}
	for _, slot := range leaf.ArtifactSet.Slots {
		addBindingRef(byName, slot.BindingRef, slot.Source)
	}
	for _, assumption := range leaf.ArtifactSet.Assumptions {
		addBindingRef(byName, assumption.Binding, assumption.Source)
	}
	out := make([]SymbolicBinding, 0, len(byName))
	for _, binding := range byName {
		out = append(out, binding)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// BindingNames returns declared symbolic binding names.
func (leaf LeafAdapter) BindingNames() []string {
	bindings := leaf.RequiredBindings()
	names := make([]string, 0, len(bindings))
	for _, binding := range bindings {
		names = append(names, binding.Name)
	}
	sort.Strings(names)
	return names
}

// BindingAudit returns a symbolic-only binding audit.
func (leaf LeafAdapter) BindingAudit() BindingAudit {
	return BindingAudit{
		DeclaredSymbolicBindings:     leaf.BindingNames(),
		LiteralCredentialDiagnostics: leaf.CredentialValueDiagnostics(),
	}
}

// CredentialValueDiagnostics flags likely literal credential values in artifact
// content without resolving or testing credentials.
func (leaf LeafAdapter) CredentialValueDiagnostics() []Diagnostic {
	return ScanCredentialValues(leaf.ArtifactSet.Artifacts)
}

// ScanCredentialValues flags likely literal credential values in artifact
// content without resolving or testing credentials.
func ScanCredentialValues(artifacts []Artifact) []Diagnostic {
	var diagnostics []Diagnostic
	for _, artifact := range artifacts {
		if len(artifact.Content) == 0 {
			continue
		}
		if ContainsLikelyCredentialValue(artifact.Content) {
			diagnostics = append(diagnostics, Diagnostic{
				Severity:    "error",
				Code:        "leaf.literal_credential",
				Message:     "artifact content appears to contain a literal credential value",
				Path:        artifact.Path,
				Remediation: "Replace literal values with symbolic binding names and provide secrets only through the trusted runtime.",
			})
		}
	}
	return diagnostics
}

// ReviewMarkdown renders the generic review package. It intentionally avoids
// execution instructions.
func (leaf LeafAdapter) ReviewMarkdown() string {
	pkg := leaf.MinimumReviewPackage()
	var b strings.Builder
	title := firstNonEmpty(pkg.Name, "Deferred Leaf Review")
	fmt.Fprintf(&b, "# %s\n\n", title)
	if pkg.Source != "" {
		fmt.Fprintf(&b, "- Source: `%s`\n", pkg.Source)
	}
	b.WriteString("- Execution boundary: review-only; runtime binding and execution are deferred to the caller.\n")
	b.WriteString("- Credential boundary: symbolic binding names only; literal credential values are prohibited.\n")
	b.WriteString("\n## Artifacts\n\n")
	if len(pkg.Artifacts) == 0 {
		b.WriteString("- No artifacts were provided.\n")
	} else {
		for _, artifact := range pkg.Artifacts {
			fmt.Fprintf(&b, "- `%s`", artifact.Path)
			if artifact.MediaType != "" {
				fmt.Fprintf(&b, " `%s`", artifact.MediaType)
			}
			fmt.Fprintf(&b, " %d bytes\n", artifact.SizeBytes)
		}
	}
	b.WriteString("\n## Required Bindings\n\n")
	if len(pkg.BindingNames) == 0 {
		b.WriteString("- No symbolic runtime bindings declared.\n")
	} else {
		for _, name := range pkg.BindingNames {
			fmt.Fprintf(&b, "- `%s`\n", name)
		}
	}
	b.WriteString("\n## Readiness Issues\n\n")
	if len(pkg.ReadinessIssues) == 0 {
		b.WriteString("- No readiness issues reported.\n")
	} else {
		for _, issue := range pkg.ReadinessIssues {
			fmt.Fprintf(&b, "- `%s`: %s\n", firstNonEmpty(issue.Code, issue.Severity), issue.Message)
		}
	}
	b.WriteString("\n## Diagnostics\n\n")
	formatted := leaf.FormatDiagnostics()
	if formatted == "" {
		b.WriteString("- No diagnostics reported.\n")
	} else {
		b.WriteString(formatted)
		b.WriteByte('\n')
	}
	b.WriteString("\n## Assumptions\n\n")
	if len(pkg.Assumptions) == 0 {
		b.WriteString("- No assumptions recorded.\n")
	} else {
		for _, assumption := range pkg.Assumptions {
			fmt.Fprintf(&b, "- %s\n", assumption.Text)
		}
	}
	b.WriteString("\n## Questions\n\n")
	if len(pkg.QuestionPlan.Questions) == 0 {
		b.WriteString("- No clarification questions recorded.\n")
	} else {
		for _, question := range pkg.QuestionPlan.Questions {
			fmt.Fprintf(&b, "- %s\n", question.Prompt)
		}
	}
	b.WriteString("\n## Required Review Actions\n\n")
	for _, action := range pkg.RequiredReviewActions {
		fmt.Fprintf(&b, "- %s\n", action)
	}
	return b.String()
}

// MinimumReviewPackage builds prompt-safe review metadata from the leaf.
func (leaf LeafAdapter) MinimumReviewPackage() ReviewPackage {
	artifacts := make([]ArtifactReview, 0, len(leaf.ArtifactSet.Artifacts))
	for _, artifact := range leaf.ArtifactSet.Artifacts {
		artifacts = append(artifacts, artifactReview(artifact))
	}
	seenArtifacts := make(map[string]bool, len(artifacts))
	for _, artifact := range artifacts {
		seenArtifacts[artifact.Path] = true
	}
	for _, artifact := range leaf.ReviewPackage.Artifacts {
		if !seenArtifacts[artifact.Path] {
			artifacts = append(artifacts, artifact)
			seenArtifacts[artifact.Path] = true
		}
	}
	sort.SliceStable(artifacts, func(i, j int) bool { return artifacts[i].Path < artifacts[j].Path })
	pkg := ReviewPackage{
		Name:                    leaf.Name,
		Source:                  leaf.Source,
		Artifacts:               artifacts,
		Diagnostics:             append([]Diagnostic(nil), leaf.ArtifactSet.Diagnostics...),
		ReadinessIssues:         append([]ReadinessIssue(nil), leaf.ArtifactSet.ReadinessIssues...),
		SymbolicBindings:        leaf.RequiredBindings(),
		BindingNames:            leaf.BindingNames(),
		Assumptions:             append([]Assumption(nil), leaf.ArtifactSet.Assumptions...),
		QuestionPlan:            leaf.ArtifactSet.QuestionPlan,
		TranscriptSummary:       transcriptSummary(leaf.ArtifactSet.Transcript),
		CredentialAudit:         leaf.BindingAudit(),
		DeferredExecutionPolicy: leaf.DeferredExecutionPolicy,
	}
	pkg.RequiredReviewActions = requiredReviewActions(leaf, pkg)
	return pkg
}

// RequiredReviewActions returns generic actions needed before downstream use.
func (leaf LeafAdapter) RequiredReviewActions() []string {
	return leaf.MinimumReviewPackage().RequiredReviewActions
}

// ArtifactOutputPath returns a filesystem path for artifactPath under root.
func ArtifactOutputPath(root, artifactPath string) (string, error) {
	artifactPath = strings.TrimSpace(artifactPath)
	if artifactPath == "" {
		return "", fmt.Errorf("artifact path is required")
	}
	rel := filepath.Clean(filepath.FromSlash(artifactPath))
	if filepath.IsAbs(rel) || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("artifact path %q must be relative to output root", artifactPath)
	}
	return filepath.Join(root, rel), nil
}

// WriteArtifactSet writes artifacts under root with path traversal protection.
func WriteArtifactSet(ctx context.Context, root string, set ArtifactSet) ([]string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	var written []string
	for _, artifact := range set.Artifacts {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		path, err := ArtifactOutputPath(root, artifact.Path)
		if err != nil {
			return nil, err
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return nil, err
		}
		if err := os.WriteFile(path, artifact.Content, 0o644); err != nil {
			return nil, err
		}
		written = append(written, path)
	}
	sort.Strings(written)
	return written, nil
}

func artifactReview(artifact Artifact) ArtifactReview {
	return ArtifactReview{
		Path:      artifact.Path,
		MediaType: artifact.MediaType,
		SizeBytes: len(artifact.Content),
	}
}

func addBindingRef(out map[string]SymbolicBinding, ref BindingRef, source string) {
	name := strings.TrimSpace(ref.Name)
	if name == "" {
		return
	}
	if _, ok := out[name]; ok {
		return
	}
	out[name] = SymbolicBinding{
		Name:        name,
		Kind:        ref.Kind,
		Source:      source,
		Description: ref.Description,
	}
}

func blockingSeverity(severity string) bool {
	switch strings.ToLower(strings.TrimSpace(severity)) {
	case "error", "critical", "blocking", "fatal":
		return true
	default:
		return false
	}
}

func requiredReviewActions(leaf LeafAdapter, pkg ReviewPackage) []string {
	actions := []string{
		"Review all artifacts before caller-specific rendering.",
		"Validate artifacts with the downstream renderer and policy checks.",
		"Keep credential values out of prompts, artifacts, logs, and committed files.",
	}
	if len(pkg.BindingNames) > 0 {
		actions = append(actions, "Map symbolic binding names to trusted runtime bindings outside apitools.")
	}
	if len(pkg.ReadinessIssues) > 0 || len(pkg.Diagnostics) > 0 || leaf.HasBlockingIssues() {
		actions = append(actions, "Resolve blocking diagnostics and readiness issues before execution-capable handoff.")
	}
	if len(pkg.QuestionPlan.Questions) > 0 {
		actions = append(actions, "Answer clarification questions before approving downstream artifacts.")
	}
	return actions
}

func transcriptSummary(transcript *Transcript) []string {
	if transcript == nil {
		return nil
	}
	out := make([]string, 0, len(transcript.Turns))
	for _, turn := range transcript.Turns {
		content := strings.TrimSpace(turn.Content)
		if content == "" {
			continue
		}
		if len(content) > 240 {
			content = content[:240] + "..."
		}
		if turn.Source != "" {
			content = turn.Source + ": " + content
		}
		out = append(out, content)
	}
	return out
}

func cloneArtifactSet(set ArtifactSet) ArtifactSet {
	out := set
	out.Artifacts = append([]Artifact(nil), set.Artifacts...)
	for i := range out.Artifacts {
		out.Artifacts[i] = cloneArtifact(out.Artifacts[i])
	}
	out.Diagnostics = append([]Diagnostic(nil), set.Diagnostics...)
	out.SymbolicBindings = append([]SymbolicBinding(nil), set.SymbolicBindings...)
	out.ReadinessIssues = append([]ReadinessIssue(nil), set.ReadinessIssues...)
	out.Slots = append([]Slot(nil), set.Slots...)
	out.Assumptions = append([]Assumption(nil), set.Assumptions...)
	if set.Transcript != nil {
		transcript := *set.Transcript
		transcript.Turns = append([]TranscriptTurn(nil), set.Transcript.Turns...)
		out.Transcript = &transcript
	}
	return out
}

func cloneArtifact(artifact Artifact) Artifact {
	out := artifact
	out.Content = append([]byte(nil), artifact.Content...)
	return out
}

func cloneReviewPackage(pkg ReviewPackage) ReviewPackage {
	out := pkg
	out.Artifacts = append([]ArtifactReview(nil), pkg.Artifacts...)
	out.Diagnostics = append([]Diagnostic(nil), pkg.Diagnostics...)
	out.ReadinessIssues = append([]ReadinessIssue(nil), pkg.ReadinessIssues...)
	out.SymbolicBindings = append([]SymbolicBinding(nil), pkg.SymbolicBindings...)
	out.BindingNames = append([]string(nil), pkg.BindingNames...)
	out.Assumptions = append([]Assumption(nil), pkg.Assumptions...)
	out.TranscriptSummary = append([]string(nil), pkg.TranscriptSummary...)
	out.CredentialAudit.DeclaredSymbolicBindings = append([]string(nil), pkg.CredentialAudit.DeclaredSymbolicBindings...)
	out.CredentialAudit.LiteralCredentialDiagnostics = append([]Diagnostic(nil), pkg.CredentialAudit.LiteralCredentialDiagnostics...)
	out.RequiredReviewActions = append([]string(nil), pkg.RequiredReviewActions...)
	out.DeferredExecutionPolicy.Notes = append([]string(nil), pkg.DeferredExecutionPolicy.Notes...)
	return out
}

var (
	providerCredentialPatterns = []*regexp.Regexp{
		regexp.MustCompile(`AIza[0-9A-Za-z_-]{20,}`),
		regexp.MustCompile(`sk-ant-api[0-9A-Za-z_-]*-[0-9A-Za-z_-]{20,}`),
		regexp.MustCompile(`sk-(?:proj-)?[0-9A-Za-z_-]{20,}`),
		regexp.MustCompile(`ghp_[0-9A-Za-z]{36,}`),
		regexp.MustCompile(`github_pat_[0-9A-Za-z_]{20,}`),
		regexp.MustCompile(`(?:AKIA|ASIA)[0-9A-Z]{16}`),
	}
	jwtValuePattern              = regexp.MustCompile(`[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+`)
	sensitiveAssignmentRegexp    = regexp.MustCompile(`(?i)\b([A-Za-z0-9_.-]*(?:api[_-]?key|apikey|app[_-]?id|appid|token|secret|password|authorization|credential)[A-Za-z0-9_.-]*)\s*[:=]\s*["']([^"'\r\n]+)["']`)
	bearerCredentialRegexp       = regexp.MustCompile(`(?i)\bBearer\s+[A-Za-z0-9._~+/-]{16,}`)
	tokenShapedCredentialPattern = regexp.MustCompile(`^[A-Za-z0-9_+/=-]+$`)
	tokenSourceAssignmentSuffix  = regexp.MustCompile(`(?i)(?:^|[_\-.])from$`)
)

// ContainsLikelyCredentialValue reports whether data contains a likely concrete
// credential value. Symbolic workflow references and binding names are allowed.
func ContainsLikelyCredentialValue(data []byte) bool {
	for _, pattern := range providerCredentialPatterns {
		if pattern.Match(data) {
			return true
		}
	}
	if bearerCredentialRegexp.Match(data) {
		return true
	}
	for _, candidate := range jwtValuePattern.FindAll(data, -1) {
		if isLeafJWT(string(candidate)) {
			return true
		}
	}
	for _, match := range sensitiveAssignmentRegexp.FindAllSubmatch(data, -1) {
		if len(match) < 3 {
			continue
		}
		if isSensitiveSourceAssignment(string(match[1])) {
			continue
		}
		if isLikelySecretLiteral(string(match[2])) {
			return true
		}
	}
	return false
}

func isSensitiveSourceAssignment(key string) bool {
	normalized := strings.ToLower(strings.TrimSpace(key))
	return normalized == "token_from" || tokenSourceAssignmentSuffix.MatchString(normalized)
}

func isLikelySecretLiteral(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" || isSymbolicReference(value) {
		return false
	}
	for _, pattern := range providerCredentialPatterns {
		if pattern.MatchString(value) {
			return true
		}
	}
	if isLeafJWT(value) {
		return true
	}
	if len(value) < 16 {
		return false
	}
	if !tokenShapedCredentialPattern.MatchString(value) {
		return false
	}
	var hasLetter, hasDigit bool
	for _, r := range value {
		if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') {
			hasLetter = true
		}
		if r >= '0' && r <= '9' {
			hasDigit = true
		}
	}
	return hasLetter && hasDigit
}

func isSymbolicReference(value string) bool {
	if strings.ContainsAny(value, " \t\r\n\"'") {
		return false
	}
	if strings.Contains(value, ".") || strings.Contains(value, "_") || strings.Contains(value, "-") {
		return true
	}
	return false
}

func isLeafJWT(candidate string) bool {
	parts := strings.Split(candidate, ".")
	if len(parts) != 3 {
		return false
	}
	header := map[string]any{}
	payload := map[string]any{}
	return decodeLeafBase64JSON(parts[0], &header) && decodeLeafBase64JSON(parts[1], &payload)
}

func decodeLeafBase64JSON(segment string, out any) bool {
	decoded, err := base64.RawURLEncoding.DecodeString(segment)
	if err != nil {
		decoded, err = base64.URLEncoding.DecodeString(segment)
		if err != nil {
			return false
		}
	}
	decoder := json.NewDecoder(bytes.NewReader(decoded))
	decoder.UseNumber()
	if err := decoder.Decode(out); err != nil {
		return false
	}
	if object, ok := out.(*map[string]any); ok {
		return len(*object) > 0
	}
	return true
}
