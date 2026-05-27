package apitools

import "testing"

func TestParseODataOperationSummaries(t *testing.T) {
	ops, err := ParseODataOperationSummaries([]byte(`
<Schema Namespace="Demo" xmlns="http://docs.oasis-open.org/odata/ns/edm">
  <EntityType Name="Product">
    <Property Name="ID" Type="Edm.String" Nullable="false"/>
  </EntityType>
  <Function Name="TopProducts">
    <Parameter Name="count" Type="Edm.Int32" Nullable="false"/>
    <ReturnType Type="Collection(Demo.Product)"/>
  </Function>
  <EntityContainer Name="Container">
    <EntitySet Name="Products" EntityType="Demo.Product"/>
  </EntityContainer>
</Schema>`), "odata/metadata.xml")
	if err != nil {
		t.Fatalf("ParseODataOperationSummaries() error = %v", err)
	}
	byID := map[string]OperationSummary{}
	for _, op := range ops {
		byID[op.ID] = op
	}
	entitySet := byID["entityset.Products"]
	if entitySet.Method != "odata" || entitySet.Path != "#/entitySets/Products" || entitySet.Provenance != "odata" {
		t.Fatalf("entity set operation = %#v", entitySet)
	}
	if entitySet.Extensions["odata_collection"] != "true" || entitySet.Extensions["source_operation_ref"] != entitySet.Path {
		t.Fatalf("entity set extensions = %#v", entitySet.Extensions)
	}
	function := byID["function.Demo.TopProducts"]
	if function.Extensions["odata_return_type"] != "Demo.Product" {
		t.Fatalf("function extensions = %#v", function.Extensions)
	}
	if len(function.Parameters) == 0 || function.Parameters[0].In != "odata-parameter" || function.Parameters[0].Name != "count" {
		t.Fatalf("function parameters = %#v", function.Parameters)
	}
}
