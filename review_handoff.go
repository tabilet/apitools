package apitools

import (
	"fmt"
	"path"
	"sort"
	"strings"
)

// ReviewHandoffVersion is the public review handoff schema version.
const ReviewHandoffVersion = "apitools.review-handoff.v1"

// ReviewState names a public review lifecycle state.
type ReviewState string

const (
	ReviewStateGenerated             ReviewState = "generated"
	ReviewStateValidated             ReviewState = "validated"
	ReviewStateReviewRequired        ReviewState = "review_required"
	ReviewStateApprovedForSandbox    ReviewState = "approved_for_sandbox"
	ReviewStateApprovedForProduction ReviewState = "approved_for_production"
	ReviewStateRejected              ReviewState = "rejected"
)

// ReviewHandoff is the public, runtime-neutral handoff manifest. Downstream
// runtimes provide concrete inputs, owners, credentials, and trusted runners.
type ReviewHandoff struct {
	Version            string                   `json:"version"`
	GeneratedState     string                   `json:"generated_state"`
	HandoffInputs      []ReviewHandoffInput     `json:"handoff_inputs"`
	ApprovalStates     []ReviewApprovalState    `json:"approval_states"`
	OwnerSplit         ReviewOwnerSplit         `json:"owner_split"`
	ExecutionPolicy    ReviewExecutionPolicy    `json:"execution_policy"`
	CredentialBindings ReviewCredentialBindings `json:"credential_bindings"`
	TrustedRunner      ReviewTrustedRunner      `json:"trusted_runner"`
}

// ReviewHandoffInput is one artifact or evidence file in a handoff package.
type ReviewHandoffInput struct {
	Path     string `json:"path"`
	Purpose  string `json:"purpose"`
	Required bool   `json:"required"`
}

// ReviewApprovalState declares one state and its allowed next states.
type ReviewApprovalState struct {
	Name              string   `json:"name"`
	Meaning           string   `json:"meaning"`
	AllowedNextStates []string `json:"allowed_next_states,omitempty"`
}

// ReviewOwnerSplit maps downstream owner names to their responsibilities.
type ReviewOwnerSplit map[string][]string

// ReviewExecutionPolicy records the public execution safety contract.
type ReviewExecutionPolicy struct {
	SideEffectful             bool   `json:"side_effectful"`
	RequiredNextState         string `json:"required_next_state,omitempty"`
	SandboxProofRunState      string `json:"sandbox_proof_run_state,omitempty"`
	ProductionExecutionState  string `json:"production_execution_state,omitempty"`
	DirectProductionExecution bool   `json:"direct_production_execution"`
}

// ReviewCredentialBindings records symbolic credential expectations.
type ReviewCredentialBindings struct {
	Declared                 []string `json:"declared,omitempty"`
	ExpectedFromPlan         []string `json:"expected_from_plan,omitempty"`
	ValuesAllowedInArtifacts bool     `json:"values_allowed_in_artifacts"`
}

// ReviewTrustedRunner records a downstream trusted execution entrypoint.
type ReviewTrustedRunner struct {
	Command     string `json:"command,omitempty"`
	SandboxOnly bool   `json:"sandbox_only"`
}

// ReviewHandoffOptions configures a generated review handoff.
type ReviewHandoffOptions struct {
	Version            string
	GeneratedState     string
	HandoffInputs      []ReviewHandoffInput
	ApprovalStates     []ReviewApprovalState
	OwnerSplit         ReviewOwnerSplit
	ExecutionPolicy    ReviewExecutionPolicy
	CredentialBindings ReviewCredentialBindings
	TrustedRunner      ReviewTrustedRunner
}

// ReviewHandoffValidationOptions configures handoff validation.
type ReviewHandoffValidationOptions struct {
	AllowedVersions []string
}

// NewReviewHandoff returns a handoff manifest with public defaults.
func NewReviewHandoff(opts ReviewHandoffOptions) ReviewHandoff {
	version := strings.TrimSpace(opts.Version)
	if version == "" {
		version = ReviewHandoffVersion
	}
	generatedState := strings.TrimSpace(opts.GeneratedState)
	if generatedState == "" {
		generatedState = string(ReviewStateGenerated)
	}
	approvalStates := cloneReviewApprovalStates(opts.ApprovalStates)
	if len(approvalStates) == 0 {
		approvalStates = DefaultReviewStateMachine()
	}
	return ReviewHandoff{
		Version:            version,
		GeneratedState:     generatedState,
		HandoffInputs:      cloneReviewHandoffInputs(opts.HandoffInputs),
		ApprovalStates:     approvalStates,
		OwnerSplit:         cloneReviewOwnerSplit(opts.OwnerSplit),
		ExecutionPolicy:    opts.ExecutionPolicy,
		CredentialBindings: cloneReviewCredentialBindings(opts.CredentialBindings),
		TrustedRunner:      opts.TrustedRunner,
	}
}

// ReviewHandoff returns a public handoff seeded from the leaf review package.
func (leaf LeafAdapter) ReviewHandoff(opts ReviewHandoffOptions) ReviewHandoff {
	if len(opts.HandoffInputs) == 0 {
		for _, artifact := range leaf.MinimumReviewPackage().Artifacts {
			opts.HandoffInputs = append(opts.HandoffInputs, ReviewHandoffInput{
				Path:     artifact.Path,
				Purpose:  "Reviewable downstream artifact.",
				Required: true,
			})
		}
	}
	if len(opts.CredentialBindings.Declared) == 0 {
		opts.CredentialBindings.Declared = leaf.BindingNames()
	}
	return NewReviewHandoff(opts)
}

// DefaultReviewStateMachine returns the public v1 review lifecycle.
func DefaultReviewStateMachine() []ReviewApprovalState {
	return []ReviewApprovalState{
		{Name: string(ReviewStateGenerated), Meaning: "Artifacts were emitted; no approval is implied.", AllowedNextStates: []string{string(ReviewStateValidated), string(ReviewStateRejected)}},
		{Name: string(ReviewStateValidated), Meaning: "Required validators and quality gates passed, or known warnings are attached.", AllowedNextStates: []string{string(ReviewStateReviewRequired), string(ReviewStateRejected)}},
		{Name: string(ReviewStateReviewRequired), Meaning: "Human review is required before side-effectful execution.", AllowedNextStates: []string{string(ReviewStateApprovedForSandbox), string(ReviewStateApprovedForProduction), string(ReviewStateRejected)}},
		{Name: string(ReviewStateApprovedForSandbox), Meaning: "A reviewer approved sandbox or test-endpoint execution only.", AllowedNextStates: []string{string(ReviewStateReviewRequired), string(ReviewStateApprovedForProduction), string(ReviewStateRejected)}},
		{Name: string(ReviewStateApprovedForProduction), Meaning: "A reviewer approved production execution through a trusted runner and approved credentials.", AllowedNextStates: []string{string(ReviewStateRejected)}},
		{Name: string(ReviewStateRejected), Meaning: "A reviewer rejected the artifact or requested regeneration.", AllowedNextStates: []string{string(ReviewStateGenerated)}},
	}
}

// DefaultReviewExecutionPolicy returns the public execution policy defaults.
func DefaultReviewExecutionPolicy(sideEffectful bool) ReviewExecutionPolicy {
	if !sideEffectful {
		return ReviewExecutionPolicy{}
	}
	return ReviewExecutionPolicy{
		SideEffectful:             true,
		RequiredNextState:         string(ReviewStateReviewRequired),
		SandboxProofRunState:      string(ReviewStateApprovedForSandbox),
		ProductionExecutionState:  string(ReviewStateApprovedForProduction),
		DirectProductionExecution: false,
	}
}

// ReviewStateNames returns the required public review state names.
func ReviewStateNames() []string {
	return []string{
		string(ReviewStateGenerated),
		string(ReviewStateValidated),
		string(ReviewStateReviewRequired),
		string(ReviewStateApprovedForSandbox),
		string(ReviewStateApprovedForProduction),
		string(ReviewStateRejected),
	}
}

// ReviewStateMachineHasRequiredStates reports whether all public states exist.
func ReviewStateMachineHasRequiredStates(states []ReviewApprovalState) bool {
	found := map[string]bool{}
	for _, state := range states {
		found[strings.TrimSpace(state.Name)] = true
	}
	for _, name := range ReviewStateNames() {
		if !found[name] {
			return false
		}
	}
	return true
}

// ReviewStateCanTransition reports whether from can move to to.
func ReviewStateCanTransition(states []ReviewApprovalState, from, to string) bool {
	from = strings.TrimSpace(from)
	to = strings.TrimSpace(to)
	for _, state := range states {
		if strings.TrimSpace(state.Name) != from {
			continue
		}
		for _, next := range state.AllowedNextStates {
			if strings.TrimSpace(next) == to {
				return true
			}
		}
		return false
	}
	return false
}

// ValidateReviewHandoff returns diagnostics for public handoff contract issues.
func ValidateReviewHandoff(manifest ReviewHandoff, opts ...ReviewHandoffValidationOptions) []Diagnostic {
	allowedVersions := []string{ReviewHandoffVersion}
	if len(opts) > 0 && len(opts[0].AllowedVersions) > 0 {
		allowedVersions = append([]string(nil), opts[0].AllowedVersions...)
	}
	var diagnostics []Diagnostic
	if !stringInSet(manifest.Version, allowedVersions) {
		diagnostics = append(diagnostics, Diagnostic{
			Severity: "error",
			Code:     "review_handoff.version",
			Message:  fmt.Sprintf("review handoff version %q is not allowed", manifest.Version),
		})
	}
	if strings.TrimSpace(manifest.GeneratedState) != string(ReviewStateGenerated) {
		diagnostics = append(diagnostics, Diagnostic{
			Severity: "error",
			Code:     "review_handoff.generated_state",
			Message:  "review handoff generated_state must be generated",
		})
	}
	diagnostics = append(diagnostics, validateReviewHandoffInputs(manifest.HandoffInputs)...)
	diagnostics = append(diagnostics, validateReviewOwnerSplit(manifest.OwnerSplit)...)
	diagnostics = append(diagnostics, validateReviewApprovalStates(manifest.ApprovalStates)...)
	if !ReviewStateMachineHasRequiredStates(manifest.ApprovalStates) {
		diagnostics = append(diagnostics, Diagnostic{
			Severity: "error",
			Code:     "review_handoff.approval_states",
			Message:  "review handoff must include all required public approval states",
		})
	}
	if manifest.CredentialBindings.ValuesAllowedInArtifacts {
		diagnostics = append(diagnostics, Diagnostic{
			Severity: "error",
			Code:     "review_handoff.credential_values",
			Message:  "review handoff must not allow credential values in artifacts",
		})
	}
	if manifest.ExecutionPolicy.DirectProductionExecution {
		diagnostics = append(diagnostics, Diagnostic{
			Severity: "error",
			Code:     "review_handoff.direct_production",
			Message:  "review handoff must not allow direct production execution",
		})
	}
	if manifest.ExecutionPolicy.SideEffectful {
		required := map[string]string{
			"required_next_state":        manifest.ExecutionPolicy.RequiredNextState,
			"sandbox_proof_run_state":    manifest.ExecutionPolicy.SandboxProofRunState,
			"production_execution_state": manifest.ExecutionPolicy.ProductionExecutionState,
		}
		expected := map[string]string{
			"required_next_state":        string(ReviewStateReviewRequired),
			"sandbox_proof_run_state":    string(ReviewStateApprovedForSandbox),
			"production_execution_state": string(ReviewStateApprovedForProduction),
		}
		for field, value := range required {
			if strings.TrimSpace(value) != expected[field] {
				diagnostics = append(diagnostics, Diagnostic{
					Severity: "error",
					Code:     "review_handoff.execution_policy",
					Message:  fmt.Sprintf("side-effectful review handoff %s must be %q", field, expected[field]),
				})
			}
		}
	}
	return diagnostics
}

func validateReviewHandoffInputs(inputs []ReviewHandoffInput) []Diagnostic {
	if len(inputs) == 0 {
		return []Diagnostic{{
			Severity: "error",
			Code:     "review_handoff.inputs",
			Message:  "review handoff must include at least one handoff input",
		}}
	}
	var diagnostics []Diagnostic
	seen := map[string]bool{}
	for _, input := range inputs {
		clean, ok := cleanReviewHandoffInputPath(input.Path)
		if !ok {
			diagnostics = append(diagnostics, Diagnostic{
				Severity: "error",
				Code:     "review_handoff.input_path",
				Message:  fmt.Sprintf("review handoff input path %q must be a safe relative path", input.Path),
			})
			continue
		}
		if seen[clean] {
			diagnostics = append(diagnostics, Diagnostic{
				Severity: "error",
				Code:     "review_handoff.input_path",
				Message:  fmt.Sprintf("review handoff input path %q is duplicated", clean),
			})
			continue
		}
		seen[clean] = true
	}
	return diagnostics
}

func cleanReviewHandoffInputPath(inputPath string) (string, bool) {
	inputPath = strings.TrimSpace(inputPath)
	if inputPath == "" || strings.Contains(inputPath, `\`) || strings.HasPrefix(inputPath, "/") || hasWindowsVolumeName(inputPath) || hasDotDotPathSegment(inputPath) {
		return "", false
	}
	clean := path.Clean(inputPath)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return "", false
	}
	return clean, true
}

func hasDotDotPathSegment(inputPath string) bool {
	for _, segment := range strings.Split(inputPath, "/") {
		if segment == ".." {
			return true
		}
	}
	return false
}

func hasWindowsVolumeName(inputPath string) bool {
	if len(inputPath) < 2 || inputPath[1] != ':' {
		return false
	}
	letter := inputPath[0]
	return (letter >= 'A' && letter <= 'Z') || (letter >= 'a' && letter <= 'z')
}

func validateReviewOwnerSplit(split ReviewOwnerSplit) []Diagnostic {
	if len(split) == 0 {
		return []Diagnostic{{
			Severity: "error",
			Code:     "review_handoff.owner_split",
			Message:  "review handoff owner_split must include at least one owner",
		}}
	}
	var diagnostics []Diagnostic
	seen := map[string]bool{}
	for owner, responsibilities := range split {
		owner = strings.TrimSpace(owner)
		if owner == "" {
			diagnostics = append(diagnostics, Diagnostic{
				Severity: "error",
				Code:     "review_handoff.owner_split",
				Message:  "review handoff owner_split owner names must be non-empty",
			})
			continue
		}
		if seen[owner] {
			diagnostics = append(diagnostics, Diagnostic{
				Severity: "error",
				Code:     "review_handoff.owner_split",
				Message:  fmt.Sprintf("review handoff owner_split owner %q is duplicated", owner),
			})
		}
		seen[owner] = true
		if len(responsibilities) == 0 {
			diagnostics = append(diagnostics, Diagnostic{
				Severity: "error",
				Code:     "review_handoff.owner_split",
				Message:  fmt.Sprintf("review handoff owner_split owner %q must include at least one responsibility", owner),
			})
			continue
		}
		for _, responsibility := range responsibilities {
			if strings.TrimSpace(responsibility) == "" {
				diagnostics = append(diagnostics, Diagnostic{
					Severity: "error",
					Code:     "review_handoff.owner_split",
					Message:  fmt.Sprintf("review handoff owner_split owner %q responsibilities must be non-empty", owner),
				})
				break
			}
		}
	}
	return diagnostics
}

func validateReviewApprovalStates(states []ReviewApprovalState) []Diagnostic {
	var diagnostics []Diagnostic
	declared := map[string]bool{}
	for _, state := range states {
		name := strings.TrimSpace(state.Name)
		if name == "" {
			diagnostics = append(diagnostics, Diagnostic{
				Severity: "error",
				Code:     "review_handoff.approval_states",
				Message:  "review handoff approval state names must be non-empty",
			})
			continue
		}
		if declared[name] {
			diagnostics = append(diagnostics, Diagnostic{
				Severity: "error",
				Code:     "review_handoff.approval_states",
				Message:  fmt.Sprintf("review handoff approval state %q is duplicated", name),
			})
			continue
		}
		declared[name] = true
	}
	for _, state := range states {
		name := strings.TrimSpace(state.Name)
		for _, next := range state.AllowedNextStates {
			next = strings.TrimSpace(next)
			if next == "" || !declared[next] {
				diagnostics = append(diagnostics, Diagnostic{
					Severity: "error",
					Code:     "review_handoff.approval_states",
					Message:  fmt.Sprintf("review handoff approval state %q references undeclared next state %q", name, next),
				})
			}
		}
	}
	return diagnostics
}

func stringInSet(value string, allowed []string) bool {
	value = strings.TrimSpace(value)
	for _, candidate := range allowed {
		if value == strings.TrimSpace(candidate) {
			return true
		}
	}
	return false
}

func cloneReviewHandoffInputs(inputs []ReviewHandoffInput) []ReviewHandoffInput {
	out := append([]ReviewHandoffInput(nil), inputs...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}

func cloneReviewApprovalStates(states []ReviewApprovalState) []ReviewApprovalState {
	out := append([]ReviewApprovalState(nil), states...)
	for i := range out {
		out[i].AllowedNextStates = append([]string(nil), out[i].AllowedNextStates...)
	}
	return out
}

func cloneReviewOwnerSplit(split ReviewOwnerSplit) ReviewOwnerSplit {
	if len(split) == 0 {
		return nil
	}
	out := make(ReviewOwnerSplit, len(split))
	for owner, responsibilities := range split {
		out[owner] = append([]string(nil), responsibilities...)
	}
	return out
}

func cloneReviewCredentialBindings(bindings ReviewCredentialBindings) ReviewCredentialBindings {
	out := bindings
	out.Declared = append([]string(nil), bindings.Declared...)
	out.ExpectedFromPlan = append([]string(nil), bindings.ExpectedFromPlan...)
	sort.Strings(out.Declared)
	sort.Strings(out.ExpectedFromPlan)
	return out
}
