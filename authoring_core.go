package apitools

import (
	"context"
	"fmt"
	"strings"
)

// Brief is the caller's natural-language authoring request.
type Brief struct {
	Text        string `json:"text,omitempty"`
	ProjectName string `json:"project_name,omitempty"`
}

// OpenAPIDoc is one OpenAPI or Swagger document supplied to the authoring core.
type OpenAPIDoc = InventoryDocument

// OperationContext is prompt-safe OpenAPI context built from a brief and one or
// more OpenAPI documents.
type OperationContext struct {
	Brief      Brief              `json:"brief,omitempty"`
	Documents  []OpenAPIDoc       `json:"documents,omitempty"`
	Inventory  OperationInventory `json:"inventory,omitempty"`
	Transcript Transcript         `json:"transcript,omitempty"`
}

// DraftInput configures the neutral authoring draft step.
type DraftInput struct {
	Brief                Brief              `json:"brief,omitempty"`
	Context              OperationContext   `json:"context,omitempty"`
	Inventory            OperationInventory `json:"inventory,omitempty"`
	SelectedOperationIDs []string           `json:"selected_operation_ids,omitempty"`
	Transcript           *Transcript        `json:"transcript,omitempty"`
}

// AuthoringCore builds prompt-safe operation context and neutral draft
// artifacts. It does not render caller-specific leaf artifacts or execute APIs.
type AuthoringCore interface {
	BuildContext(ctx context.Context, brief Brief, docs []OpenAPIDoc) (OperationContext, error)
	Draft(ctx context.Context, input DraftInput) (ArtifactSet, error)
}

// NeutralAuthoringCore is the default domain-neutral implementation of
// AuthoringCore.
type NeutralAuthoringCore struct {
	InventoryLimit int `json:"inventory_limit,omitempty"`
}

// NewAuthoringCore returns the default domain-neutral authoring core.
func NewAuthoringCore() NeutralAuthoringCore {
	return NeutralAuthoringCore{}
}

// BuildContext converts OpenAPI documents into prompt-safe operation context.
func (core NeutralAuthoringCore) BuildContext(ctx context.Context, brief Brief, docs []OpenAPIDoc) (OperationContext, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	inventory, err := BuildOperationInventory(ctx, InventoryOptions{
		Documents: docs,
		Query:     brief.Text,
		Limit:     core.InventoryLimit,
	})
	if err != nil {
		return OperationContext{}, err
	}
	transcript := buildContextTranscript(brief, docs, inventory)
	return OperationContext{
		Brief:      brief,
		Documents:  append([]OpenAPIDoc(nil), docs...),
		Inventory:  inventory,
		Transcript: transcript,
	}, nil
}

// Draft renders neutral project.md and intent.hcl draft artifacts from
// operation context or an explicit inventory.
func (core NeutralAuthoringCore) Draft(ctx context.Context, input DraftInput) (ArtifactSet, error) {
	_ = core
	brief := input.Brief
	if brief.Text == "" && brief.ProjectName == "" {
		brief = input.Context.Brief
	}
	inventory := input.Inventory
	if !hasOperationInventory(inventory) {
		inventory = input.Context.Inventory
	}
	transcript := input.Transcript
	if transcript == nil && len(input.Context.Transcript.Turns) > 0 {
		copy := input.Context.Transcript
		transcript = &copy
	}
	return DraftArtifacts(ctx, DraftOptions{
		Brief:                brief.Text,
		ProjectName:          brief.ProjectName,
		Inventory:            inventory,
		SelectedOperationIDs: input.SelectedOperationIDs,
		Transcript:           transcript,
	})
}

// DraftFromOpenAPI runs the full neutral authoring pipeline: build prompt-safe
// operation context, then render neutral draft artifacts.
func DraftFromOpenAPI(ctx context.Context, core AuthoringCore, brief Brief, docs []OpenAPIDoc, selectedOperationIDs []string) (OperationContext, ArtifactSet, error) {
	if core == nil {
		return OperationContext{}, ArtifactSet{}, fmt.Errorf("authoring core is required")
	}
	opContext, err := core.BuildContext(ctx, brief, docs)
	if err != nil {
		return OperationContext{}, ArtifactSet{}, err
	}
	artifacts, err := core.Draft(ctx, DraftInput{
		Context:              opContext,
		SelectedOperationIDs: selectedOperationIDs,
	})
	if err != nil {
		return opContext, ArtifactSet{}, err
	}
	return opContext, artifacts, nil
}

// LeafRenderer renders a review-only leaf into caller-owned artifacts. The
// renderer owns validation and any domain-specific output.
type LeafRenderer interface {
	RenderLeaf(ctx context.Context, leaf LeafAdapter) (ArtifactSet, []Diagnostic, error)
}

// DraftAndRender renders neutral draft artifacts through a caller-provided leaf
// renderer. The returned leaf contains prompt-safe review metadata only.
func DraftAndRender(ctx context.Context, core AuthoringCore, renderer LeafRenderer, input DraftInput) (LeafAdapter, ArtifactSet, []Diagnostic, error) {
	if core == nil {
		return LeafAdapter{}, ArtifactSet{}, nil, fmt.Errorf("authoring core is required")
	}
	if renderer == nil {
		return LeafAdapter{}, ArtifactSet{}, nil, fmt.Errorf("leaf renderer is required")
	}
	artifacts, err := core.Draft(ctx, input)
	if err != nil {
		return LeafAdapter{}, ArtifactSet{}, nil, err
	}
	leaf := NewLeafAdapter(artifacts, LeafOptions{
		Name:   firstNonEmpty(input.Brief.ProjectName, input.Context.Brief.ProjectName),
		Source: "apitools.draft",
	})
	rendered, diagnostics, err := renderer.RenderLeaf(ctx, leaf)
	return leaf, rendered, diagnostics, err
}

func hasOperationInventory(inventory OperationInventory) bool {
	return len(inventory.Documents) > 0 ||
		len(inventory.Operations) > 0 ||
		len(inventory.Diagnostics) > 0 ||
		len(inventory.ReadinessIssues) > 0
}

func buildContextTranscript(brief Brief, docs []OpenAPIDoc, inventory OperationInventory) Transcript {
	var transcript Transcript
	if strings.TrimSpace(brief.Text) != "" {
		transcript.Turns = append(transcript.Turns, TranscriptTurn{
			Role:    "user",
			Content: strings.TrimSpace(brief.Text),
			Source:  "brief",
		})
	}
	for _, doc := range docs {
		transcript.Turns = append(transcript.Turns, TranscriptTurn{
			Role:    "tool",
			Content: fmt.Sprintf("Loaded OpenAPI document %q from %s.", openAPIDocLabel(doc), openAPIDocSource(doc)),
			Source:  "openapi.document",
		})
	}
	transcript.Turns = append(transcript.Turns, TranscriptTurn{
		Role:    "tool",
		Content: fmt.Sprintf("Built prompt-safe operation inventory with %d document(s), %d operation(s), and %d readiness issue(s).", len(inventory.Documents), len(inventory.Operations), len(inventory.ReadinessIssues)),
		Source:  "openapi.inventory",
	})
	return transcript
}

func openAPIDocLabel(doc OpenAPIDoc) string {
	return firstNonEmpty(doc.Name, doc.Path, doc.URL, "inline document")
}

func openAPIDocSource(doc OpenAPIDoc) string {
	return firstNonEmpty(doc.Path, doc.URL, doc.Name, "inline content")
}
