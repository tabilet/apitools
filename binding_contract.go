package apitools

import (
	"sort"
	"strings"
)

// BindingContract is the review-time contract for runtime-provided bindings.
// It records names and auth metadata only; callers resolve concrete values at
// execution time.
type BindingContract struct {
	SymbolicBindings             []SymbolicBinding        `json:"symbolic_bindings,omitempty"`
	BindingNames                 []string                 `json:"binding_names,omitempty"`
	ExpectedBindingNames         []string                 `json:"expected_binding_names,omitempty"`
	AuthRequirements             []AuthRequirementSummary `json:"auth_requirements,omitempty"`
	CredentialFields             []BindingField           `json:"credential_fields,omitempty"`
	OptionalCredentialFields     []BindingField           `json:"optional_credential_fields,omitempty"`
	ConfigFields                 []BindingField           `json:"config_fields,omitempty"`
	OptionalConfigFields         []BindingField           `json:"optional_config_fields,omitempty"`
	LiteralCredentialDiagnostics []Diagnostic             `json:"literal_credential_diagnostics,omitempty"`
	ValuesAllowedInArtifacts     bool                     `json:"values_allowed_in_artifacts"`
}

// BindingField describes a named field required by an auth requirement.
type BindingField struct {
	Name     string `json:"name"`
	Kind     string `json:"kind,omitempty"`
	Scheme   string `json:"scheme,omitempty"`
	Binding  string `json:"binding,omitempty"`
	Required bool   `json:"required,omitempty"`
}

// BindingContractOptions configures binding contract construction.
type BindingContractOptions struct {
	SymbolicBindings         []SymbolicBinding
	BindingNames             []string
	ExpectedBindingNames     []string
	AuthRequirements         []AuthRequirementSummary
	Artifacts                []Artifact
	ValuesAllowedInArtifacts bool
}

// BuildBindingContract merges symbolic binding names, OpenAPI auth summaries,
// and optional artifact credential scans into a stable runtime-bound contract.
func BuildBindingContract(opts BindingContractOptions) BindingContract {
	byName := map[string]SymbolicBinding{}
	for _, binding := range opts.SymbolicBindings {
		addSymbolicBinding(byName, binding)
	}
	for _, name := range opts.BindingNames {
		addSymbolicBinding(byName, SymbolicBinding{Name: name})
	}
	bindings := make([]SymbolicBinding, 0, len(byName))
	for _, binding := range byName {
		bindings = append(bindings, binding)
	}
	sort.SliceStable(bindings, func(i, j int) bool { return bindings[i].Name < bindings[j].Name })
	names := make([]string, 0, len(bindings))
	for _, binding := range bindings {
		names = append(names, binding.Name)
	}

	auth := mergeAuthRequirements(opts.AuthRequirements)
	contract := BindingContract{
		SymbolicBindings:         bindings,
		BindingNames:             names,
		ExpectedBindingNames:     sortedUniqueStrings(opts.ExpectedBindingNames),
		AuthRequirements:         auth,
		ValuesAllowedInArtifacts: opts.ValuesAllowedInArtifacts,
	}
	for _, requirement := range auth {
		bindingName := bindingNameForAuthRequirement(requirement)
		contract.CredentialFields = appendBindingFields(contract.CredentialFields, requirement, requirement.CredentialFields, bindingName, true)
		contract.OptionalCredentialFields = appendBindingFields(contract.OptionalCredentialFields, requirement, requirement.OptionalCredentialFields, bindingName, false)
		contract.ConfigFields = appendBindingFields(contract.ConfigFields, requirement, requirement.ConfigFields, bindingName, true)
		contract.OptionalConfigFields = appendBindingFields(contract.OptionalConfigFields, requirement, requirement.OptionalConfigFields, bindingName, false)
	}
	if !opts.ValuesAllowedInArtifacts {
		contract.LiteralCredentialDiagnostics = ScanCredentialValues(opts.Artifacts)
	}
	return contract
}

// ReviewCredentialBindings converts a binding contract into the generic handoff
// credential section.
func (contract BindingContract) ReviewCredentialBindings() ReviewCredentialBindings {
	expected := contract.ExpectedBindingNames
	if len(expected) == 0 {
		expected = contract.BindingNames
	}
	return ReviewCredentialBindings{
		Declared:                 sortedUniqueStrings(contract.BindingNames),
		ExpectedFromPlan:         sortedUniqueStrings(expected),
		ValuesAllowedInArtifacts: contract.ValuesAllowedInArtifacts,
	}
}

// BindingAudit converts a binding contract into the legacy audit shape.
func (contract BindingContract) BindingAudit() BindingAudit {
	return BindingAudit{
		DeclaredSymbolicBindings:     sortedUniqueStrings(contract.BindingNames),
		LiteralCredentialDiagnostics: append([]Diagnostic(nil), contract.LiteralCredentialDiagnostics...),
	}
}

func addSymbolicBinding(out map[string]SymbolicBinding, binding SymbolicBinding) {
	name := strings.TrimSpace(binding.Name)
	if name == "" {
		return
	}
	binding.Name = name
	if existing, ok := out[name]; ok {
		if existing.Kind == "" {
			existing.Kind = strings.TrimSpace(binding.Kind)
		}
		if existing.Source == "" {
			existing.Source = strings.TrimSpace(binding.Source)
		}
		if existing.Description == "" {
			existing.Description = strings.TrimSpace(binding.Description)
		}
		out[name] = existing
		return
	}
	binding.Kind = strings.TrimSpace(binding.Kind)
	binding.Source = strings.TrimSpace(binding.Source)
	binding.Description = strings.TrimSpace(binding.Description)
	out[name] = binding
}

func appendBindingFields(out []BindingField, requirement AuthRequirementSummary, fields []string, binding string, required bool) []BindingField {
	seen := map[string]struct{}{}
	for _, field := range out {
		seen[bindingFieldKey(field)] = struct{}{}
	}
	for _, name := range fields {
		field := BindingField{
			Name:     strings.TrimSpace(name),
			Kind:     strings.TrimSpace(requirement.Kind),
			Scheme:   strings.TrimSpace(requirement.Scheme),
			Binding:  binding,
			Required: required,
		}
		if field.Name == "" {
			continue
		}
		key := bindingFieldKey(field)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, field)
	}
	sort.SliceStable(out, func(i, j int) bool { return bindingFieldKey(out[i]) < bindingFieldKey(out[j]) })
	return out
}

func bindingFieldKey(field BindingField) string {
	return strings.Join([]string{field.Binding, field.Kind, field.Scheme, field.Name}, "\x00")
}

func bindingNameForAuthRequirement(requirement AuthRequirementSummary) string {
	scheme := strings.TrimSpace(requirement.Scheme)
	if scheme == "" {
		scheme = strings.TrimSpace(requirement.Kind)
	}
	return scheme
}
