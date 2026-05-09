package apitools

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadOperationIndexDetectsDuplicatesAndLooksUpOperationID(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "openapi.yaml")
	if err := os.WriteFile(path, []byte(`openapi: 3.0.0
info:
  title: Index API
  version: 1.0.0
paths:
  /items:
    post:
      operationId: createItem
      responses:
        "200":
          description: ok
  /items/{id}:
    get:
      operationId: readItem
      responses:
        "200":
          description: ok
`), 0o644); err != nil {
		t.Fatal(err)
	}
	index, err := LoadOperationIndex(path)
	if err != nil {
		t.Fatal(err)
	}
	operation, ok := index.LookupOperationID("readItem")
	if !ok || operation.Method != "GET" || operation.Path != "/items/{id}" {
		t.Fatalf("operation = %#v ok=%v", operation, ok)
	}

	duplicate := filepath.Join(dir, "duplicate.yaml")
	if err := os.WriteFile(duplicate, []byte(`openapi: 3.0.0
info:
  title: Duplicate API
  version: 1.0.0
paths:
  /left:
    get:
      operationId: same
      responses:
        "200":
          description: ok
  /right:
    post:
      operationId: same
      responses:
        "200":
          description: ok
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadOperationIndex(duplicate); err == nil || !strings.Contains(err.Error(), "duplicated") {
		t.Fatalf("expected duplicate error, got %v", err)
	}
}

func TestLoadOperationIndexSurfacesReadFailuresAsDiagnostics(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target.yaml")
	if err := os.WriteFile(target, []byte(`openapi: 3.0.0
info:
  title: Index API
  version: 1.0.0
paths: {}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "openapi.yaml")
	symlinkOrSkip(t, target, link)

	_, err := LoadOperationIndex(link)
	var diagnosticErr DiagnosticError
	if !errors.As(err, &diagnosticErr) {
		t.Fatalf("expected DiagnosticError, got %T %v", err, err)
	}
	if len(diagnosticErr.Diagnostics) != 1 || diagnosticErr.Diagnostics[0].Code != "document.read" {
		t.Fatalf("diagnostics = %#v", diagnosticErr.Diagnostics)
	}
}
