package graph_test

import (
	"testing"

	"github.com/chazu/lignin/pkg/graph"
)

func TestPartRegistry(t *testing.T) {
	pr := graph.NewPartRegistry()

	// Test DefinePart
	metadata := graph.PartMetadata{
		Name:      "test-part",
		GrainAxis: graph.GrainZ,
		Material: graph.MaterialSpec{
			Type: "oak",
		},
	}
	partID, err := pr.DefinePart("test-part", []graph.SolidID{"solid-1"}, metadata, "node-1")
	if err != nil {
		t.Fatalf("DefinePart failed: %v", err)
	}

	// Test GetPart
	part, err := pr.GetPart("test-part")
	if err != nil {
		t.Fatalf("GetPart failed: %v", err)
	}
	if part.Name != "test-part" {
		t.Errorf("Part name: got %v, want test-part", part.Name)
	}
	if part.ID != partID {
		t.Errorf("Part ID: got %v, want %v", part.ID, partID)
	}

	// Test GetPartByID
	part2, err := pr.GetPartByID(partID)
	if err != nil {
		t.Fatalf("GetPartByID failed: %v", err)
	}
	if part2.Name != "test-part" {
		t.Errorf("Part2 name: got %v, want test-part", part2.Name)
	}

	// Test ListParts
	parts := pr.ListParts()
	if len(parts) != 1 {
		t.Errorf("ListParts length: got %v, want 1", len(parts))
	}
	if parts[0] != partID {
		t.Errorf("ListParts[0]: got %v, want %v", parts[0], partID)
	}

	// Test duplicate part name
	_, err = pr.DefinePart("test-part", []graph.SolidID{"solid-2"}, metadata, "node-2")
	if err == nil {
		t.Error("Expected error for duplicate part name, got nil")
	}

	// Test GetPart for non-existent part
	_, err = pr.GetPart("non-existent")
	if err == nil {
		t.Error("Expected error for non-existent part, got nil")
	}
}

func TestGraphBuilder(t *testing.T) {
	gb := graph.NewGraphBuilder()

	// Create a primitive node
	primitiveID := gb.CreatePrimitiveNode("cuboid", graph.Vector3{X: 10, Y: 20, Z: 30})

	// Create a part node
	metadata := graph.PartMetadata{
		Name:      "test-part",
		GrainAxis: graph.GrainZ,
		Material: graph.MaterialSpec{
			Type: "oak",
		},
	}
	partNodeID, partID, err := gb.CreatePartNode("test-part", []graph.NodeID{primitiveID}, metadata)
	if err != nil {
		t.Fatalf("CreatePartNode failed: %v", err)
	}

	// Build the graph
	graph, pr := gb.Build()

	// Verify graph structure
	if len(graph.Nodes) != 2 {
		t.Errorf("Graph nodes count: got %v, want 2", len(graph.Nodes))
	}
	if len(graph.Roots) != 1 {
		t.Errorf("Graph roots count: got %v, want 1", len(graph.Roots))
	}
	if graph.Roots[0] != primitiveID {
		t.Errorf("Graph root: got %v, want %v", graph.Roots[0], primitiveID)
	}

	// Verify part registry
	part, err := pr.GetPart("test-part")
	if err != nil {
		t.Fatalf("GetPart failed: %v", err)
	}
	if part.NodeID != partNodeID {
		t.Errorf("Part NodeID: got %v, want %v", part.NodeID, partNodeID)
	}
	if part.ID != partID {
		t.Errorf("Part ID: got %v, want %v", part.ID, partID)
	}
}

func TestDesignBuilder(t *testing.T) {
	db := graph.NewDesignBuilder()

	// Add a primitive
	primitiveID := db.AddPrimitive("leg", "cuboid", graph.Vector3{X: 50, Y: 50, Z: 750})

	// Add a part
	partNodeID, partID, err := db.AddPart("leg-1", []graph.NodeID{primitiveID}, graph.GrainZ, "oak")
	if err != nil {
		t.Fatalf("AddPart failed: %v", err)
	}

	// Verify part was created
	if partNodeID == "" {
		t.Error("Part node ID is empty")
	}
	if partID == "" {
		t.Error("Part ID is empty")
	}

	// Build the design
	design := db.BuildDesign("1.0.0")

	// Verify design
	if design.Version != "1.0.0" {
		t.Errorf("Design version: got %v, want 1.0.0", design.Version)
	}
	if len(design.Graph.Nodes) != 2 {
		t.Errorf("Design graph nodes: got %v, want 2", len(design.Graph.Nodes))
	}
	if len(design.Parts) != 1 {
		t.Errorf("Design parts: got %v, want 1", len(design.Parts))
	}

	// Verify the part exists in design
	part, exists := design.Parts[partID]
	if !exists {
		t.Fatalf("Part %v not found in design", partID)
	}
	if part.Name != "leg-1" {
		t.Errorf("Part name: got %v, want leg-1", part.Name)
	}
}

func TestOutputRef(t *testing.T) {
	// Test OutputRef with PartID
	partOutput := graph.OutputRef{
		Type: graph.OutputTypePart,
		ID:   graph.PartID("test-part"),
		Name: "Test Part",
	}

	if partOutput.Type != graph.OutputTypePart {
		t.Errorf("Output type: got %v, want OutputTypePart", partOutput.Type)
	}
	if partOutput.Name != "Test Part" {
		t.Errorf("Output name: got %v, want Test Part", partOutput.Name)
	}

	partID, ok := partOutput.ID.(graph.PartID)
	if !ok {
		t.Fatal("Output ID is not a PartID")
	}
	if partID != "test-part" {
		t.Errorf("Part ID: got %v, want test-part", partID)
	}

	// Test OutputRef with SolidID
	solidOutput := graph.OutputRef{
		Type: graph.OutputTypeSolid,
		ID:   graph.SolidID("solid-1"),
		Name: "Test Solid",
	}

	if solidOutput.Type != graph.OutputTypeSolid {
		t.Errorf("Output type: got %v, want OutputTypeSolid", solidOutput.Type)
	}

	solidID, ok := solidOutput.ID.(graph.SolidID)
	if !ok {
		t.Fatal("Output ID is not a SolidID")
	}
	if solidID != "solid-1" {
		t.Errorf("Solid ID: got %v, want solid-1", solidID)
	}
}