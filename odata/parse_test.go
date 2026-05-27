package odata

import "testing"

func TestParseCSDLXMLPreservesResourcesOperationsAndSelectors(t *testing.T) {
	model, err := Parse([]byte(`<?xml version="1.0" encoding="utf-8"?>
<edmx:Edmx Version="4.0" xmlns:edmx="http://docs.oasis-open.org/odata/ns/edmx">
  <edmx:DataServices>
    <Schema Namespace="Demo" xmlns="http://docs.oasis-open.org/odata/ns/edm">
      <EntityType Name="Product">
        <Key><PropertyRef Name="ID"/></Key>
        <Property Name="ID" Type="Edm.String" Nullable="false"/>
        <Property Name="Name" Type="Edm.String"/>
        <NavigationProperty Name="Category" Type="Demo.Category"/>
      </EntityType>
      <EntityType Name="Category">
        <Property Name="ID" Type="Edm.String"/>
      </EntityType>
      <Action Name="RateProduct" IsBound="true">
        <Parameter Name="bindingParameter" Type="Demo.Product"/>
        <Parameter Name="Rating" Type="Edm.Int32" Nullable="false"/>
      </Action>
      <Function Name="TopProducts" IsComposable="true">
        <Parameter Name="count" Type="Edm.Int32" Nullable="false"/>
        <ReturnType Type="Collection(Demo.Product)"/>
      </Function>
      <EntityContainer Name="Container">
        <EntitySet Name="Products" EntityType="Demo.Product"/>
        <Singleton Name="Me" Type="Demo.Product"/>
        <FunctionImport Name="TopProducts" Function="Demo.TopProducts" EntitySet="Products"/>
      </EntityContainer>
    </Schema>
  </edmx:DataServices>
</edmx:Edmx>`))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if model.SourceKind != ArtifactKindCSDLXML || model.Version != "4.0" {
		t.Fatalf("model kind/version = %q/%q", model.SourceKind, model.Version)
	}
	summaries := model.OperationSummaries()
	byID := map[string]OperationSummary{}
	for _, summary := range summaries {
		byID[summary.ID] = summary
	}
	if byID["entityset.Products"].Selector != "#/entitySets/Products" || !byID["entityset.Products"].Collection {
		t.Fatalf("entity set summary = %#v", byID["entityset.Products"])
	}
	if byID["singleton.Me"].EntityType != "Demo.Product" {
		t.Fatalf("singleton summary = %#v", byID["singleton.Me"])
	}
	action := byID["action.Demo.RateProduct"]
	if !action.Bound || len(action.Parameters) != 2 || action.Parameters[1].Name != "Rating" {
		t.Fatalf("action summary = %#v", action)
	}
	function := byID["function.Demo.TopProducts"]
	if function.ReturnType != "Demo.Product" || len(function.QueryOptions) == 0 {
		t.Fatalf("function summary = %#v", function)
	}
	nav := byID["navigation.Demo.Product.Category"]
	if nav.Selector != "#/entityTypes/Demo.Product/navigation/Category" || nav.EntityType != "Demo.Category" {
		t.Fatalf("navigation summary = %#v", nav)
	}
	target, err := model.ResolveSelector("function.Demo.TopProducts")
	if err != nil {
		t.Fatalf("ResolveSelector() error = %v", err)
	}
	if target.Kind != SelectorKindOperation || target.Summary.ID != "function.Demo.TopProducts" {
		t.Fatalf("selector target = %#v", target)
	}
}

func TestParseJSONCSDL(t *testing.T) {
	model, err := Parse([]byte(`{
  "$Version": "4.01",
  "$EntityContainer": "Demo.Container",
  "Demo": {
    "Product": {
      "$Kind": "EntityType",
      "$Key": ["ID"],
      "ID": {"$Type": "Edm.String", "$Nullable": false},
      "Category": {"$Kind": "NavigationProperty", "$Type": "Demo.Category"}
    },
    "RateProduct": {
      "$Kind": "Action",
      "$IsBound": true,
      "$Parameter": [
        {"$Name": "bindingParameter", "$Type": "Demo.Product"},
        {"$Name": "Rating", "$Type": "Edm.Int32", "$Nullable": false}
      ]
    },
    "Container": {
      "$Kind": "EntityContainer",
      "Products": {"$Kind": "EntitySet", "$Type": "Demo.Product"},
      "RateProduct": {"$Kind": "ActionImport", "$Action": "Demo.RateProduct"}
    }
  }
}`))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if model.SourceKind != ArtifactKindCSDLJSON || model.Version != "4.01" {
		t.Fatalf("model kind/version = %q/%q", model.SourceKind, model.Version)
	}
	if len(model.OperationSummaries()) != 4 {
		t.Fatalf("summaries = %#v", model.OperationSummaries())
	}
}

func TestDetectLocalArtifact(t *testing.T) {
	detection, ok := DetectLocalArtifact([]byte(`<Schema Namespace="Demo" xmlns="http://docs.oasis-open.org/odata/ns/edm"><EntityContainer Name="C"><EntitySet Name="Products" EntityType="Demo.Product"/></EntityContainer></Schema>`), "$metadata.xml")
	if !ok {
		t.Fatalf("DetectLocalArtifact() did not detect CSDL XML")
	}
	if detection.Kind != ArtifactKindCSDLXML || detection.Format != "xml" {
		t.Fatalf("detection = %#v", detection)
	}
	if _, ok := DetectLocalArtifact([]byte(`{"title":"not odata"}`), "spec.json"); ok {
		t.Fatalf("non-OData JSON detected")
	}
}

func TestParseRejectsMalformedSource(t *testing.T) {
	if _, err := Parse([]byte(`not odata`)); err == nil {
		t.Fatalf("Parse() error = nil, want malformed source error")
	}
	if _, err := Parse([]byte(`<root/>`)); err == nil {
		t.Fatalf("Parse() error = nil, want missing CSDL metadata error")
	}
}
