package apitools

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ReviewArtifactInput is caller-bound artifact metadata used to assemble a
// runtime-neutral review package or handoff manifest.
type ReviewArtifactInput struct {
	Path      string `json:"path"`
	Kind      string `json:"kind,omitempty"`
	MediaType string `json:"media_type,omitempty"`
	Purpose   string `json:"purpose,omitempty"`
	SizeBytes int    `json:"size_bytes,omitempty"`
	Required  bool   `json:"required,omitempty"`
}

// ReviewPackageInput builds prompt-safe review metadata without artifact
// content.
type ReviewPackageInput struct {
	Name                    string
	Source                  string
	Artifacts               []ReviewArtifactInput
	Diagnostics             []Diagnostic
	ReadinessIssues         []ReadinessIssue
	SymbolicBindings        []SymbolicBinding
	BindingNames            []string
	Assumptions             []Assumption
	QuestionPlan            QuestionPlan
	Transcript              *Transcript
	BindingContract         BindingContract
	DeferredExecutionPolicy DeferredExecutionPolicy
}

// BuildReviewPackage builds review metadata from caller-supplied artifact and
// issue metadata.
func BuildReviewPackage(input ReviewPackageInput) ReviewPackage {
	artifacts := make([]ArtifactReview, 0, len(input.Artifacts))
	for _, artifact := range input.Artifacts {
		path := strings.TrimSpace(artifact.Path)
		if path == "" {
			continue
		}
		artifacts = append(artifacts, ArtifactReview{
			Path:      path,
			MediaType: strings.TrimSpace(artifact.MediaType),
			SizeBytes: artifact.SizeBytes,
		})
	}
	sort.SliceStable(artifacts, func(i, j int) bool { return artifacts[i].Path < artifacts[j].Path })
	contract := input.BindingContract
	if len(contract.BindingNames) == 0 && (len(input.SymbolicBindings) > 0 || len(input.BindingNames) > 0) {
		contract = BuildBindingContract(BindingContractOptions{
			SymbolicBindings: input.SymbolicBindings,
			BindingNames:     input.BindingNames,
		})
	}
	policy := input.DeferredExecutionPolicy
	if !policy.ReviewOnly && !policy.RuntimeDeferred && !policy.DirectExecutionDenied && len(policy.Notes) == 0 {
		policy = defaultDeferredExecutionPolicy()
	}
	pkg := ReviewPackage{
		Name:                    strings.TrimSpace(input.Name),
		Source:                  strings.TrimSpace(input.Source),
		Artifacts:               artifacts,
		Diagnostics:             append([]Diagnostic(nil), input.Diagnostics...),
		ReadinessIssues:         append([]ReadinessIssue(nil), input.ReadinessIssues...),
		SymbolicBindings:        append([]SymbolicBinding(nil), contract.SymbolicBindings...),
		BindingNames:            append([]string(nil), contract.BindingNames...),
		Assumptions:             append([]Assumption(nil), input.Assumptions...),
		QuestionPlan:            input.QuestionPlan,
		TranscriptSummary:       transcriptSummary(input.Transcript),
		CredentialAudit:         contract.BindingAudit(),
		DeferredExecutionPolicy: policy,
	}
	pkg.RequiredReviewActions = requiredReviewActionsForPackage(pkg)
	return pkg
}

// ReviewHandoffInputsFromArtifacts converts artifact metadata to stable,
// deduplicated handoff inputs and appends stable extra inputs.
func ReviewHandoffInputsFromArtifacts(artifacts []ReviewArtifactInput, extra ...ReviewHandoffInput) []ReviewHandoffInput {
	seen := map[string]struct{}{}
	var inputs []ReviewHandoffInput
	add := func(input ReviewHandoffInput) {
		input.Path = strings.TrimSpace(input.Path)
		input.Purpose = strings.TrimSpace(input.Purpose)
		if input.Path == "" {
			return
		}
		clean, ok := cleanReviewHandoffInputPath(input.Path)
		if !ok {
			input.Path = strings.TrimSpace(input.Path)
		} else {
			input.Path = clean
		}
		if _, ok := seen[input.Path]; ok {
			return
		}
		seen[input.Path] = struct{}{}
		inputs = append(inputs, input)
	}
	for _, artifact := range artifacts {
		required := artifact.Required
		if !required {
			required = true
		}
		add(ReviewHandoffInput{
			Path:     artifact.Path,
			Purpose:  firstNonEmpty(artifact.Purpose, "Reviewable artifact."),
			Required: required,
		})
	}
	for _, input := range extra {
		add(input)
	}
	sort.SliceStable(inputs, func(i, j int) bool { return inputs[i].Path < inputs[j].Path })
	return inputs
}

// ReviewHandoffDigestOptions configures a deterministic digest over required
// handoff input files.
type ReviewHandoffDigestOptions struct {
	Root    string
	Scope   string
	Version string
	Inputs  []ReviewHandoffInput
}

type reviewHandoffDigestFile struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

type reviewHandoffDigest struct {
	Version string                    `json:"version"`
	Scope   string                    `json:"scope"`
	Files   []reviewHandoffDigestFile `json:"files"`
}

// ComputeReviewHandoffDigest hashes the required files referenced by a handoff
// manifest. The digest includes file paths and file SHA-256s, not file content.
func ComputeReviewHandoffDigest(opts ReviewHandoffDigestOptions) (string, error) {
	root := strings.TrimSpace(opts.Root)
	if root == "" {
		root = "."
	}
	scope := strings.Trim(strings.TrimSpace(filepath.ToSlash(opts.Scope)), "/")
	version := strings.TrimSpace(opts.Version)
	if version == "" {
		version = "apitools.review-handoff-digest.v1"
	}
	fileSet := map[string]struct{}{}
	for _, input := range opts.Inputs {
		if !input.Required {
			continue
		}
		clean, ok := cleanReviewHandoffInputPath(input.Path)
		if !ok {
			return "", fmt.Errorf("handoff input path must be safe relative path: %q", input.Path)
		}
		fileSet[clean] = struct{}{}
	}
	files := make([]string, 0, len(fileSet))
	for path := range fileSet {
		files = append(files, path)
	}
	sort.Strings(files)
	digest := reviewHandoffDigest{
		Version: version,
		Scope:   scope,
	}
	for _, path := range files {
		full := filepath.Join(root, filepath.FromSlash(path))
		data, err := os.ReadFile(full)
		if err != nil {
			return "", fmt.Errorf("read handoff input %s: %w", path, err)
		}
		sum := sha256.Sum256(data)
		reportPath := path
		if scope != "" {
			reportPath = filepath.ToSlash(filepath.Join(scope, path))
		}
		digest.Files = append(digest.Files, reviewHandoffDigestFile{
			Path:   reportPath,
			SHA256: hex.EncodeToString(sum[:]),
		})
	}
	canonical, err := json.Marshal(digest)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(canonical)
	return hex.EncodeToString(sum[:]), nil
}

func requiredReviewActionsForPackage(pkg ReviewPackage) []string {
	actions := []string{
		"Review all artifacts before caller-specific rendering.",
		"Validate artifacts with the downstream renderer and policy checks.",
		"Keep credential values out of prompts, artifacts, logs, and committed files.",
	}
	if len(pkg.BindingNames) > 0 {
		actions = append(actions, "Map symbolic binding names to trusted runtime bindings outside apitools.")
	}
	if len(pkg.ReadinessIssues) > 0 || len(pkg.Diagnostics) > 0 || HasBlockingIssues(IssueSetFromFindings(pkg.Diagnostics, pkg.ReadinessIssues).Issues) {
		actions = append(actions, "Resolve blocking diagnostics and readiness issues before execution-capable handoff.")
	}
	if len(pkg.QuestionPlan.Questions) > 0 {
		actions = append(actions, "Answer clarification questions before approving downstream artifacts.")
	}
	return actions
}
