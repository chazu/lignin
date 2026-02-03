package graph

import "testing"

func TestNewDesignGraph(t *testing.T) {
	g := New()
	if g.Nodes == nil {
		t.Fatal("Nodes map should be initialized")
	}
	if g.NameIndex == nil {
		t.Fatal("NameIndex map should be initialized")
	}
	if g.Defaults.Clearance != DefaultClearance {
		t.Errorf("default clearance = %f, want %f", g.Defaults.Clearance, DefaultClearance)
	}
	if g.Defaults.Units != "mm" {
		t.Errorf("default units = %q, want %q", g.Defaults.Units, "mm")
	}
	if g.NodeCount() != 0 {
		t.Errorf("empty graph should have 0 nodes, got %d", g.NodeCount())
	}
}

func TestAddNodeAndLookup(t *testing.T) {
	g := New()

	id := NewNodeID("defpart/front")
	node := &Node{
		ID:   id,
		Kind: NodePrimitive,
		Name: "front",
		Data: BoardData{
			PrimKind:   PrimBoard,
			Dimensions: Vec3{400, 200, 19},
			Grain:      AxisZ,
			Material:   MaterialSpec{Species: "white-oak"},
		},
	}
	g.AddNode(node)
	g.AddRoot(id)

	if g.NodeCount() != 1 {
		t.Errorf("node count = %d, want 1", g.NodeCount())
	}

	// Lookup by name
	found := g.Lookup("front")
	if found == nil {
		t.Fatal("Lookup('front') returned nil")
	}
	if found.ID != id {
		t.Errorf("lookup returned wrong node")
	}

	// MustLookup
	must := g.MustLookup("front")
	if must.ID != id {
		t.Errorf("MustLookup returned wrong node")
	}

	// Lookup miss
	if g.Lookup("nonexistent") != nil {
		t.Error("Lookup should return nil for missing name")
	}

	// Get by ID
	got := g.Get(id)
	if got == nil || got.Name != "front" {
		t.Errorf("Get by ID failed")
	}

	// Roots
	if len(g.Roots) != 1 || g.Roots[0] != id {
		t.Errorf("roots = %v, want [%s]", g.Roots, id.Short())
	}
}

func TestMustLookupPanics(t *testing.T) {
	g := New()
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustLookup should panic on missing name")
		}
	}()
	g.MustLookup("missing")
}

func TestPartsAndJoins(t *testing.T) {
	g := New()

	frontID := NewNodeID("defpart/front")
	leftID := NewNodeID("defpart/left")
	joinID := NewNodeID("butt-joint/front-left")

	g.AddNode(&Node{
		ID: frontID, Kind: NodePrimitive, Name: "front",
		Data: BoardData{PrimKind: PrimBoard, Dimensions: Vec3{400, 200, 19}},
	})
	g.AddNode(&Node{
		ID: leftID, Kind: NodePrimitive, Name: "left",
		Data: BoardData{PrimKind: PrimBoard, Dimensions: Vec3{262, 200, 19}},
	})
	g.AddNode(&Node{
		ID: joinID, Kind: NodeJoin, Name: "",
		Data: JoinData{
			Kind:  JoinButt,
			PartA: frontID, FaceA: FaceLeft,
			PartB: leftID, FaceB: FaceFront,
			Params: ButtJoinParams{GlueUp: true},
		},
	})

	parts := g.Parts()
	if len(parts) != 2 {
		t.Errorf("Parts() count = %d, want 2", len(parts))
	}
	joins := g.Joins()
	if len(joins) != 1 {
		t.Errorf("Joins() count = %d, want 1", len(joins))
	}
}

func TestChildren(t *testing.T) {
	g := New()

	childID := NewNodeID("defpart/shelf")
	parentID := NewNodeID("assembly/bookcase")

	g.AddNode(&Node{
		ID: childID, Kind: NodePrimitive, Name: "shelf",
		Data: BoardData{PrimKind: PrimBoard, Dimensions: Vec3{600, 300, 19}},
	})
	g.AddNode(&Node{
		ID: parentID, Kind: NodeGroup, Name: "bookcase",
		Children: []NodeID{childID},
		Data:     GroupData{},
	})

	parent := g.Get(parentID)
	children := g.Children(parent)
	if len(children) != 1 {
		t.Fatalf("Children count = %d, want 1", len(children))
	}
	if children[0].Name != "shelf" {
		t.Errorf("child name = %q, want %q", children[0].Name, "shelf")
	}
}

func TestNodeIDDeterministic(t *testing.T) {
	a := NewNodeID("defpart/front")
	b := NewNodeID("defpart/front")
	if a != b {
		t.Error("same path should produce same NodeID")
	}

	c := NewNodeID("defpart/back")
	if a == c {
		t.Error("different paths should produce different NodeIDs")
	}
}

func TestNodeIDZero(t *testing.T) {
	var id NodeID
	if !id.IsZero() {
		t.Error("zero-value NodeID should be zero")
	}
	id = NewNodeID("something")
	if id.IsZero() {
		t.Error("non-zero NodeID should not be zero")
	}
}

func TestVec3(t *testing.T) {
	a := Vec3{1, 2, 3}
	b := Vec3{4, 5, 6}

	sum := a.Add(b)
	if sum != (Vec3{5, 7, 9}) {
		t.Errorf("Add = %v, want (5, 7, 9)", sum)
	}

	scaled := a.Scale(2)
	if scaled != (Vec3{2, 4, 6}) {
		t.Errorf("Scale = %v, want (2, 4, 6)", scaled)
	}
}

func TestFaceIDValid(t *testing.T) {
	for _, f := range []FaceID{FaceTop, FaceBottom, FaceLeft, FaceRight, FaceFront, FaceBack} {
		if !ValidFaceIDs[f] {
			t.Errorf("face %q should be valid", f)
		}
	}
	if ValidFaceIDs["diagonal"] {
		t.Error("invalid face should not be valid")
	}
}

func TestNodeDataInterface(t *testing.T) {
	// Verify all concrete types implement NodeData at compile time.
	var _ NodeData = BoardData{}
	var _ NodeData = DowelData{}
	var _ NodeData = TransformData{}
	var _ NodeData = GroupData{}
	var _ NodeData = JoinData{}
	var _ NodeData = DrillData{}
	var _ NodeData = FastenerData{}
}

func TestJoinParamsInterface(t *testing.T) {
	var _ JoinParams = ButtJoinParams{}
}

func TestStringers(t *testing.T) {
	if AxisX.String() != "X" {
		t.Errorf("AxisX.String() = %q", AxisX.String())
	}
	if NodePrimitive.String() != "primitive" {
		t.Errorf("NodePrimitive.String() = %q", NodePrimitive.String())
	}
	if JoinButt.String() != "butt" {
		t.Errorf("JoinButt.String() = %q", JoinButt.String())
	}
	if FastenerScrew.String() != "screw" {
		t.Errorf("FastenerScrew.String() = %q", FastenerScrew.String())
	}

	id := NewNodeID("test")
	if len(id.Short()) != 12 { // 6 bytes = 12 hex chars
		t.Errorf("Short() len = %d, want 12", len(id.Short()))
	}

	v := Vec3{1.5, 2.5, 3.5}
	if v.String() != "(1.5, 2.5, 3.5)" {
		t.Errorf("Vec3.String() = %q", v.String())
	}
}
