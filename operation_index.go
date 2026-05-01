package apitools

import (
	"context"
	"fmt"
	"strings"
)

// OperationIndex provides stable lookup by OpenAPI operationId over an
// OperationInventory.
type OperationIndex struct {
	Inventory    OperationInventory          `json:"inventory"`
	OperationIDs map[string]OperationSummary `json:"operation_ids"`
}

// LoadOperationIndex loads one OpenAPI document and indexes operations by
// operationId. Duplicate operationIds are rejected.
func LoadOperationIndex(path string) (OperationIndex, error) {
	inventory, err := BuildOperationInventory(context.Background(), InventoryOptions{
		Documents: []InventoryDocument{{Path: path}},
	})
	if err != nil {
		return OperationIndex{}, err
	}
	return NewOperationIndex(inventory)
}

// NewOperationIndex builds an operationId index over an existing inventory.
func NewOperationIndex(inventory OperationInventory) (OperationIndex, error) {
	index := OperationIndex{
		Inventory:    inventory,
		OperationIDs: map[string]OperationSummary{},
	}
	for _, diagnostic := range inventory.Diagnostics {
		if diagnostic.Severity == "error" {
			return OperationIndex{}, DiagnosticError{Diagnostics: inventory.Diagnostics}
		}
	}
	for _, operation := range inventory.Operations {
		operationID := strings.TrimSpace(operation.OperationID)
		if operationID == "" {
			continue
		}
		if existing, ok := index.OperationIDs[operationID]; ok {
			return OperationIndex{}, fmt.Errorf("OpenAPI operationId %q is duplicated at %s %s and %s %s", operationID, existing.Method, existing.Path, operation.Method, operation.Path)
		}
		index.OperationIDs[operationID] = operation
	}
	if len(index.OperationIDs) == 0 {
		return OperationIndex{}, fmt.Errorf("OpenAPI document must define at least one operationId")
	}
	return index, nil
}

// LookupOperationID returns the operation summary for operationID.
func (index OperationIndex) LookupOperationID(operationID string) (OperationSummary, bool) {
	operation, ok := index.OperationIDs[strings.TrimSpace(operationID)]
	return operation, ok
}
