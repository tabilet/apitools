package apitools

import (
	"encoding/json"
	"testing"
)

func TestDefaultReviewStateMachine(t *testing.T) {
	states := DefaultReviewStateMachine()
	if !ReviewStateMachineHasRequiredStates(states) {
		t.Fatalf("missing required states: %#v", states)
	}
	if !ReviewStateCanTransition(states, string(ReviewStateGenerated), string(ReviewStateValidated)) {
		t.Fatalf("expected generated -> validated transition")
	}
	if ReviewStateCanTransition(states, string(ReviewStateGenerated), string(ReviewStateApprovedForProduction)) {
		t.Fatalf("unexpected generated -> approved_for_production transition")
	}
}

func TestNewReviewHandoffDefaultsAndRoundTrip(t *testing.T) {
	manifest := NewReviewHandoff(ReviewHandoffOptions{
		HandoffInputs: []ReviewHandoffInput{{Path: "workflow.hcl", Required: true}},
		OwnerSplit: ReviewOwnerSplit{
			"runtime": {"execute approved artifacts"},
		},
		ExecutionPolicy: DefaultReviewExecutionPolicy(true),
		CredentialBindings: ReviewCredentialBindings{
			Declared: []string{"api_token"},
		},
	})
	if manifest.Version != ReviewHandoffVersion {
		t.Fatalf("version = %q", manifest.Version)
	}
	if manifest.GeneratedState != string(ReviewStateGenerated) {
		t.Fatalf("generated state = %q", manifest.GeneratedState)
	}
	if diagnostics := ValidateReviewHandoff(manifest); len(diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %#v", diagnostics)
	}
	data, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	var decoded ReviewHandoff
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.OwnerSplit["runtime"][0] != "execute approved artifacts" {
		t.Fatalf("owner split did not round trip: %#v", decoded.OwnerSplit)
	}
}

func TestValidateReviewHandoffReportsUnsafePolicy(t *testing.T) {
	manifest := validReviewHandoff()
	manifest.ExecutionPolicy = ReviewExecutionPolicy{
		SideEffectful:             true,
		DirectProductionExecution: true,
	}
	manifest.CredentialBindings = ReviewCredentialBindings{
		ValuesAllowedInArtifacts: true,
	}
	diagnostics := ValidateReviewHandoff(manifest)
	codes := map[string]bool{}
	for _, diagnostic := range diagnostics {
		codes[diagnostic.Code] = true
	}
	for _, code := range []string{"review_handoff.credential_values", "review_handoff.direct_production", "review_handoff.execution_policy"} {
		if !codes[code] {
			t.Fatalf("missing diagnostic %s in %#v", code, diagnostics)
		}
	}
}

func TestValidateReviewHandoffReportsContractGaps(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*ReviewHandoff)
		code   string
	}{
		{
			name: "missing inputs",
			mutate: func(manifest *ReviewHandoff) {
				manifest.HandoffInputs = nil
			},
			code: "review_handoff.inputs",
		},
		{
			name: "empty input path",
			mutate: func(manifest *ReviewHandoff) {
				manifest.HandoffInputs[0].Path = " "
			},
			code: "review_handoff.input_path",
		},
		{
			name: "unsafe input path",
			mutate: func(manifest *ReviewHandoff) {
				manifest.HandoffInputs[0].Path = "../workflow.hcl"
			},
			code: "review_handoff.input_path",
		},
		{
			name: "duplicate input path",
			mutate: func(manifest *ReviewHandoff) {
				manifest.HandoffInputs = append(manifest.HandoffInputs, ReviewHandoffInput{Path: "./workflow.hcl", Required: true})
			},
			code: "review_handoff.input_path",
		},
		{
			name: "missing owner split",
			mutate: func(manifest *ReviewHandoff) {
				manifest.OwnerSplit = nil
			},
			code: "review_handoff.owner_split",
		},
		{
			name: "empty owner responsibility",
			mutate: func(manifest *ReviewHandoff) {
				manifest.OwnerSplit["runtime"] = []string{""}
			},
			code: "review_handoff.owner_split",
		},
		{
			name: "duplicate approval state",
			mutate: func(manifest *ReviewHandoff) {
				manifest.ApprovalStates = append(manifest.ApprovalStates, ReviewApprovalState{Name: string(ReviewStateGenerated)})
			},
			code: "review_handoff.approval_states",
		},
		{
			name: "unknown transition",
			mutate: func(manifest *ReviewHandoff) {
				manifest.ApprovalStates[0].AllowedNextStates = append(manifest.ApprovalStates[0].AllowedNextStates, "unknown")
			},
			code: "review_handoff.approval_states",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manifest := validReviewHandoff()
			tt.mutate(&manifest)
			diagnostics := ValidateReviewHandoff(manifest)
			if !diagnosticsContainCode(diagnostics, tt.code) {
				t.Fatalf("missing diagnostic %s in %#v", tt.code, diagnostics)
			}
		})
	}
}

func validReviewHandoff() ReviewHandoff {
	return NewReviewHandoff(ReviewHandoffOptions{
		HandoffInputs: []ReviewHandoffInput{{Path: "workflow.hcl", Required: true}},
		OwnerSplit: ReviewOwnerSplit{
			"runtime": {"execute approved artifacts"},
		},
		ExecutionPolicy: ReviewExecutionPolicy{
			SideEffectful:            true,
			RequiredNextState:        string(ReviewStateReviewRequired),
			SandboxProofRunState:     string(ReviewStateApprovedForSandbox),
			ProductionExecutionState: string(ReviewStateApprovedForProduction),
		},
	})
}

func diagnosticsContainCode(diagnostics []Diagnostic, code string) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == code {
			return true
		}
	}
	return false
}

func TestLeafAdapterReviewHandoffUsesArtifactsAndBindings(t *testing.T) {
	leaf := NewLeafAdapter(ArtifactSet{
		Artifacts: []Artifact{{Path: "intent.hcl", MediaType: "text/hcl", Content: []byte("workflow {}")}},
		SymbolicBindings: []SymbolicBinding{
			{Name: "api_token", Kind: "credential"},
		},
	}, LeafOptions{Name: "test"})
	manifest := leaf.ReviewHandoff(ReviewHandoffOptions{})
	if len(manifest.HandoffInputs) != 1 || manifest.HandoffInputs[0].Path != "intent.hcl" {
		t.Fatalf("handoff inputs = %#v", manifest.HandoffInputs)
	}
	if len(manifest.CredentialBindings.Declared) != 1 || manifest.CredentialBindings.Declared[0] != "api_token" {
		t.Fatalf("credential bindings = %#v", manifest.CredentialBindings)
	}
}
