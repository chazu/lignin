package graph_test

import (
	"testing"
	"time"

	"github.com/chazu/lignin/pkg/graph"
)

func TestNodeTypes(t *testing.T) {
	tests := []struct {
		name     string
		nodeType graph.NodeType
		wantStr  string
	}{
		{"Primitive", graph.NodeTypePrimitive, "NodeTypePrimitive"},
		{"Transform", graph.NodeTypeTransform, "NodeTypeTransform"},
		{"Join", graph.NodeTypeJoin, "NodeTypeJoin"},
		{"Part", graph.NodeTypePart, "NodeTypePart"},
		{"Group", graph.NodeTypeGroup, "NodeTypeGroup"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Just verify the constants are defined
			if tt.nodeType < graph.NodeTypePrimitive || tt.nodeType > graph.NodeTypeGroup {
				t.Errorf("NodeType %v out of range", tt.nodeType)
			}
		})
	}
}

func TestGrainDirection(t *testing.T) {
	tests := []struct {
		name   string
		grain  graph.GrainDirection
		wantOK bool
	}{
		{"GrainX", graph.GrainX, true},
		{"GrainY", graph.GrainY, true},
		{"GrainZ", graph.GrainZ, true},
		{"GrainAny", graph.GrainAny, true},
		{"Invalid", graph.GrainDirection(99), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.grain >= graph.GrainX && tt.grain <= graph.GrainAny
			if isValid != tt.wantOK {
				t.Errorf("GrainDirection %v: got valid=%v, want valid=%v", tt.grain, isValid, tt.wantOK)
			}
		})
	}
}

func TestJoinType(t *testing.T) {
	tests := []struct {
		name     string
		joinType graph.JoinType
		wantOK   bool
	}{
		{"Butt", graph.JoinTypeButt, true},
		{"Hole", graph.JoinTypeHole, true},
		{"Fastener", graph.JoinTypeFastener, true},
		{"Dovetail", graph.JoinTypeDovetail, true},
		{"MortiseTenon", graph.JoinTypeMortiseTenon, true},
		{"Invalid", graph.JoinType(99), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.joinType >= graph.JoinTypeButt && tt.joinType <= graph.JoinTypeMortiseTenon
			if isValid != tt.wantOK {
				t.Errorf("JoinType %v: got valid=%v, want valid=%v", tt.joinType, isValid, tt.wantOK)
			}
		})
	}
}

func TestNodeCreation(t *testing.T) {
	now := time.Now()
	node := &graph.Node{
		ID:         "test-node-1",
		Type:       graph.NodeTypePrimitive,
		SourceExpr: "(primitive :cuboid [10 20 30])",
		Properties: map[string]interface{}{
			"type":       "cuboid",
			"dimensions": graph.Vector3{X: 10, Y: 20, Z: 30},
		},
		Metadata: graph.NodeMetadata{
			CreatedAt:   now,
			EvaluatedAt: now,
			Tags:        []string{"primitive", "cuboid"},
		},
	}

	if node.ID != "test-node-1" {
		t.Errorf("Node ID: got %v, want test-node-1", node.ID)
	}
	if node.Type != graph.NodeTypePrimitive {
		t.Errorf("Node Type: got %v, want NodeTypePrimitive", node.Type)
	}
	if node.SourceExpr != "(primitive :cuboid [10 20 30])" {
		t.Errorf("SourceExpr: got %v, want (primitive :cuboid [10 20 30])", node.SourceExpr)
	}
}

func TestPartMetadata(t *testing.T) {
	metadata := graph.PartMetadata{
		Name:      "test-part",
		GrainAxis: graph.GrainZ,
		Material: graph.MaterialSpec{
			Type:      "oak",
			Thickness: 25.0,
			Density:   700.0,
			Color:     "brown",
		},
		Tags: []string{"leg", "vertical"},
	}

	if metadata.Name != "test-part" {
		t.Errorf("Part name: got %v, want test-part", metadata.Name)
	}
	if metadata.GrainAxis != graph.GrainZ {
		t.Errorf("Grain axis: got %v, want GrainZ", metadata.GrainAxis)
	}
	if metadata.Material.Type != "oak" {
		t.Errorf("Material type: got %v, want oak", metadata.Material.Type)
	}
	if metadata.Material.Thickness != 25.0 {
		t.Errorf("Material thickness: got %v, want 25.0", metadata.Material.Thickness)
	}
}

func TestJoinSpec(t *testing.T) {
	spec := graph.JoinSpec{
		Type: graph.JoinTypeButt,
		PartA: "leg-1",
		FaceA: graph.FaceID{Solid: "solid-1", Index: 0},
		PartB: "apron-1",
		FaceB: graph.FaceID{Solid: "solid-2", Index: 2},
		Clearance: 0.2,
		Parameters: map[string]interface{}{
			"fasteners": []graph.FastenerSpec{},
		},
	}

	if spec.Type != graph.JoinTypeButt {
		t.Errorf("Join type: got %v, want JoinTypeButt", spec.Type)
	}
	if spec.PartA != "leg-1" {
		t.Errorf("PartA: got %v, want leg-1", spec.PartA)
	}
	if spec.PartB != "apron-1" {
		t.Errorf("PartB: got %v, want apron-1", spec.PartB)
	}
	if spec.Clearance != 0.2 {
		t.Errorf("Clearance: got %v, want 0.2", spec.Clearance)
	}
	if spec.FaceA.Index != 0 {
		t.Errorf("FaceA index: got %v, want 0", spec.FaceA.Index)
	}
	if spec.FaceB.Index != 2 {
		t.Errorf("FaceB index: got %v, want 2", spec.FaceB.Index)
	}
}

func TestVector3(t *testing.T) {
	v := graph.Vector3{X: 1.5, Y: 2.5, Z: 3.5}

	if v.X != 1.5 {
		t.Errorf("Vector X: got %v, want 1.5", v.X)
	}
	if v.Y != 2.5 {
		t.Errorf("Vector Y: got %v, want 2.5", v.Y)
	}
	if v.Z != 3.5 {
		t.Errorf("Vector Z: got %v, want 3.5", v.Z)
	}
}