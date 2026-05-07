package apitools

import "testing"

func TestBuildBindingContractMergesNamesAuthAndCredentialDiagnostics(t *testing.T) {
	contract := BuildBindingContract(BindingContractOptions{
		SymbolicBindings: []SymbolicBinding{{Name: "runtime.api", Kind: "apiKey"}, {Name: "runtime.api", Description: "duplicate"}},
		BindingNames:     []string{"runtime.db"},
		AuthRequirements: []AuthRequirementSummary{{
			Kind:                     "api_key",
			Scheme:                   "ApiKeyAuth",
			CredentialFields:         []string{"api_key"},
			OptionalCredentialFields: []string{"session_token"},
			ConfigFields:             []string{"endpoint"},
		}},
		Artifacts: []Artifact{{Path: "secret.hcl", Content: []byte(`api_key = "sk-proj-abcdefghijklmnopqrst1234567890"`)}},
	})
	if len(contract.BindingNames) != 2 || contract.BindingNames[0] != "runtime.api" || contract.BindingNames[1] != "runtime.db" {
		t.Fatalf("binding names = %#v", contract.BindingNames)
	}
	if len(contract.CredentialFields) != 1 || contract.CredentialFields[0].Name != "api_key" || contract.CredentialFields[0].Binding != "ApiKeyAuth" {
		t.Fatalf("credential fields = %#v", contract.CredentialFields)
	}
	if len(contract.OptionalCredentialFields) != 1 || contract.OptionalCredentialFields[0].Required {
		t.Fatalf("optional credential fields = %#v", contract.OptionalCredentialFields)
	}
	if len(contract.LiteralCredentialDiagnostics) != 1 {
		t.Fatalf("credential diagnostics = %#v", contract.LiteralCredentialDiagnostics)
	}
	handoff := contract.ReviewCredentialBindings()
	if len(handoff.Declared) != 2 || len(handoff.ExpectedFromPlan) != 2 || handoff.ValuesAllowedInArtifacts {
		t.Fatalf("handoff bindings = %#v", handoff)
	}
}
