package graph

import (
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

// buildValidBox creates a valid 2-part box graph (front + left + butt joint)
// with all nodes reachable from a group root. This mirrors the existing test
// graph from graph_test.go but adds a group root and a join node.
func buildValidBox() *DesignGraph {
	g := New()

	frontID := NewNodeID("defpart/front")
	leftID := NewNodeID("defpart/left")
	joinID := NewNodeID("butt-joint/front-left")
	groupID := NewNodeID("assembly/box")

	g.AddNode(&Node{
		ID: frontID, Kind: NodePrimitive, Name: "front",
		Data: BoardData{PrimKind: PrimBoard, Dimensions: Vec3{400, 200, 19}},
	})
	g.AddNode(&Node{
		ID: leftID, Kind: NodePrimitive, Name: "left",
		Data: BoardData{PrimKind: PrimBoard, Dimensions: Vec3{262, 200, 19}},
	})
	g.AddNode(&Node{
		ID: joinID, Kind: NodeJoin,
		Data: JoinData{
			Kind:  JoinButt,
			PartA: frontID, FaceA: FaceLeft,
			PartB: leftID, FaceB: FaceFront,
			Params: ButtJoinParams{GlueUp: true},
		},
	})
	g.AddNode(&Node{
		ID:       groupID,
		Kind:     NodeGroup,
		Name:     "box",
		Children: []NodeID{frontID, leftID, joinID},
		Data:     GroupData{Description: "simple box"},
	})
	g.AddRoot(groupID)

	return g
}

// hasError returns true if errs contains at least one error-severity finding
// whose message contains substr.
func hasError(errs []ValidationError, substr string) bool {
	for _, e := range errs {
		if e.Severity == SeverityError && strings.Contains(e.Message, substr) {
			return true
		}
	}
	return false
}

// hasWarning returns true if errs contains at least one warning-severity
// finding whose message contains substr.
func hasWarning(errs []ValidationError, substr string) bool {
	for _, e := range errs {
		if e.Severity == SeverityWarning && strings.Contains(e.Message, substr) {
			return true
		}
	}
	return false
}

// errorCount returns the number of error-severity findings.
func errorCount(errs []ValidationError) int {
	n := 0
	for _, e := range errs {
		if e.Severity == SeverityError {
			n++
		}
	}
	return n
}

// warningCount returns the number of warning-severity findings.
func warningCount(errs []ValidationError) int {
	n := 0
	for _, e := range errs {
		if e.Severity == SeverityWarning {
			n++
		}
	}
	return n
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestValidate_ValidGraph(t *testing.T) {
	g := buildValidBox()
	errs := Validate(g)
	if len(errs) != 0 {
		for _, e := range errs {
			t.Errorf("unexpected validation error: %s", e)
		}
	}
}

func TestValidate_EmptyGraph(t *testing.T) {
	g := New()
	errs := Validate(g)
	if len(errs) != 0 {
		for _, e := range errs {
			t.Errorf("unexpected validation error on empty graph: %s", e)
		}
	}
}

func TestValidate_CycleDetection(t *testing.T) {
	g := New()

	aID := NewNodeID("a")
	bID := NewNodeID("b")
	cID := NewNodeID("c")

	// Create a cycle: a -> b -> c -> a
	g.AddNode(&Node{
		ID: aID, Kind: NodeGroup, Name: "a",
		Children: []NodeID{bID},
		Data:     GroupData{},
	})
	g.AddNode(&Node{
		ID: bID, Kind: NodeGroup, Name: "b",
		Children: []NodeID{cID},
		Data:     GroupData{},
	})
	g.AddNode(&Node{
		ID: cID, Kind: NodeGroup, Name: "c",
		Children: []NodeID{aID},
		Data:     GroupData{},
	})
	g.AddRoot(aID)

	errs := Validate(g)
	if !hasError(errs, "cycle") {
		t.Error("expected cycle detection error, got none")
		for _, e := range errs {
			t.Logf("  %s", e)
		}
	}
}

func TestValidate_DanglingReference(t *testing.T) {
	g := New()

	parentID := NewNodeID("parent")
	missingID := NewNodeID("missing-child")

	g.AddNode(&Node{
		ID: parentID, Kind: NodeGroup, Name: "parent",
		Children: []NodeID{missingID},
		Data:     GroupData{},
	})
	g.AddRoot(parentID)

	errs := Validate(g)
	if !hasError(errs, "does not exist") {
		t.Error("expected dangling reference error, got none")
		for _, e := range errs {
			t.Logf("  %s", e)
		}
	}
}

func TestValidate_DanglingJoinReference(t *testing.T) {
	g := New()

	frontID := NewNodeID("defpart/front")
	missingID := NewNodeID("defpart/missing")
	joinID := NewNodeID("join/test")
	groupID := NewNodeID("group/test")

	g.AddNode(&Node{
		ID: frontID, Kind: NodePrimitive, Name: "front",
		Data: BoardData{PrimKind: PrimBoard, Dimensions: Vec3{400, 200, 19}},
	})
	g.AddNode(&Node{
		ID: joinID, Kind: NodeJoin,
		Data: JoinData{
			Kind:  JoinButt,
			PartA: frontID, FaceA: FaceLeft,
			PartB: missingID, FaceB: FaceRight,
			Params: ButtJoinParams{},
		},
	})
	g.AddNode(&Node{
		ID: groupID, Kind: NodeGroup, Name: "group",
		Children: []NodeID{frontID, joinID},
		Data:     GroupData{},
	})
	g.AddRoot(groupID)

	errs := Validate(g)
	if !hasError(errs, "part_b reference") {
		t.Error("expected dangling join part_b error, got none")
		for _, e := range errs {
			t.Logf("  %s", e)
		}
	}
}

func TestValidate_DanglingFastenerReference(t *testing.T) {
	g := New()

	frontID := NewNodeID("defpart/front")
	leftID := NewNodeID("defpart/left")
	joinID := NewNodeID("join/test")
	missingFastenerID := NewNodeID("fastener/missing")
	groupID := NewNodeID("group/test")

	g.AddNode(&Node{
		ID: frontID, Kind: NodePrimitive, Name: "front",
		Data: BoardData{PrimKind: PrimBoard, Dimensions: Vec3{400, 200, 19}},
	})
	g.AddNode(&Node{
		ID: leftID, Kind: NodePrimitive, Name: "left",
		Data: BoardData{PrimKind: PrimBoard, Dimensions: Vec3{262, 200, 19}},
	})
	g.AddNode(&Node{
		ID: joinID, Kind: NodeJoin,
		Data: JoinData{
			Kind:      JoinButt,
			PartA:     frontID, FaceA: FaceLeft,
			PartB:     leftID, FaceB: FaceFront,
			Params:    ButtJoinParams{},
			Fasteners: []NodeID{missingFastenerID},
		},
	})
	g.AddNode(&Node{
		ID: groupID, Kind: NodeGroup, Name: "group",
		Children: []NodeID{frontID, leftID, joinID},
		Data:     GroupData{},
	})
	g.AddRoot(groupID)

	errs := Validate(g)
	if !hasError(errs, "fastener reference") {
		t.Error("expected dangling fastener reference error, got none")
		for _, e := range errs {
			t.Logf("  %s", e)
		}
	}
}

func TestValidate_DanglingDrillTarget(t *testing.T) {
	g := New()

	drillID := NewNodeID("drill/test")
	missingPartID := NewNodeID("defpart/missing")
	groupID := NewNodeID("group/test")

	g.AddNode(&Node{
		ID: drillID, Kind: NodeDrill,
		Data: DrillData{
			TargetPart: missingPartID,
			Face:       FaceTop,
			Diameter:   5,
			Depth:      10,
		},
	})
	g.AddNode(&Node{
		ID: groupID, Kind: NodeGroup, Name: "group",
		Children: []NodeID{drillID},
		Data:     GroupData{},
	})
	g.AddRoot(groupID)

	errs := Validate(g)
	if !hasError(errs, "target_part reference") {
		t.Error("expected dangling drill target_part error, got none")
		for _, e := range errs {
			t.Logf("  %s", e)
		}
	}
}

func TestValidate_DanglingFastenerJoinRef(t *testing.T) {
	g := New()

	fastenerID := NewNodeID("fastener/test")
	missingJoinID := NewNodeID("join/missing")
	groupID := NewNodeID("group/test")

	g.AddNode(&Node{
		ID: fastenerID, Kind: NodeFastener,
		Data: FastenerData{
			Kind:     FastenerScrew,
			Diameter: 4,
			Length:   50,
			JoinRef:  missingJoinID,
		},
	})
	g.AddNode(&Node{
		ID: groupID, Kind: NodeGroup, Name: "group",
		Children: []NodeID{fastenerID},
		Data:     GroupData{},
	})
	g.AddRoot(groupID)

	errs := Validate(g)
	if !hasError(errs, "join_ref reference") {
		t.Error("expected dangling fastener join_ref error, got none")
		for _, e := range errs {
			t.Logf("  %s", e)
		}
	}
}

func TestValidate_DuplicateName(t *testing.T) {
	g := New()

	id1 := NewNodeID("defpart/a")
	id2 := NewNodeID("defpart/b")
	groupID := NewNodeID("group/test")

	g.AddNode(&Node{
		ID: id1, Kind: NodePrimitive, Name: "shelf",
		Data: BoardData{PrimKind: PrimBoard, Dimensions: Vec3{600, 300, 19}},
	})
	// Manually add a second node with the same name. AddNode will overwrite
	// the NameIndex entry, but the first node still has Name="shelf".
	node2 := &Node{
		ID: id2, Kind: NodePrimitive, Name: "shelf",
		Data: BoardData{PrimKind: PrimBoard, Dimensions: Vec3{600, 300, 19}},
	}
	g.Nodes[id2] = node2
	// Note: g.NameIndex["shelf"] now points to id1 (from AddNode), but id2
	// also has Name "shelf". The validator checks node Name fields directly.

	g.AddNode(&Node{
		ID: groupID, Kind: NodeGroup, Name: "group",
		Children: []NodeID{id1, id2},
		Data:     GroupData{},
	})
	g.AddRoot(groupID)

	errs := Validate(g)
	if !hasError(errs, "duplicate name") {
		t.Error("expected duplicate name error, got none")
		for _, e := range errs {
			t.Logf("  %s", e)
		}
	}
}

func TestValidate_InvalidFaceID(t *testing.T) {
	g := New()

	frontID := NewNodeID("defpart/front")
	leftID := NewNodeID("defpart/left")
	joinID := NewNodeID("join/test")
	groupID := NewNodeID("group/test")

	g.AddNode(&Node{
		ID: frontID, Kind: NodePrimitive, Name: "front",
		Data: BoardData{PrimKind: PrimBoard, Dimensions: Vec3{400, 200, 19}},
	})
	g.AddNode(&Node{
		ID: leftID, Kind: NodePrimitive, Name: "left",
		Data: BoardData{PrimKind: PrimBoard, Dimensions: Vec3{262, 200, 19}},
	})
	g.AddNode(&Node{
		ID: joinID, Kind: NodeJoin,
		Data: JoinData{
			Kind:   JoinButt,
			PartA:  frontID, FaceA: FaceID("diagonal"), // invalid
			PartB:  leftID, FaceB: FaceFront,
			Params: ButtJoinParams{},
		},
	})
	g.AddNode(&Node{
		ID: groupID, Kind: NodeGroup, Name: "group",
		Children: []NodeID{frontID, leftID, joinID},
		Data:     GroupData{},
	})
	g.AddRoot(groupID)

	errs := Validate(g)
	if !hasError(errs, "invalid face_a") {
		t.Error("expected invalid face_a error, got none")
		for _, e := range errs {
			t.Logf("  %s", e)
		}
	}
}

func TestValidate_InvalidFaceB(t *testing.T) {
	g := New()

	frontID := NewNodeID("defpart/front")
	leftID := NewNodeID("defpart/left")
	joinID := NewNodeID("join/test")
	groupID := NewNodeID("group/test")

	g.AddNode(&Node{
		ID: frontID, Kind: NodePrimitive, Name: "front",
		Data: BoardData{PrimKind: PrimBoard, Dimensions: Vec3{400, 200, 19}},
	})
	g.AddNode(&Node{
		ID: leftID, Kind: NodePrimitive, Name: "left",
		Data: BoardData{PrimKind: PrimBoard, Dimensions: Vec3{262, 200, 19}},
	})
	g.AddNode(&Node{
		ID: joinID, Kind: NodeJoin,
		Data: JoinData{
			Kind:   JoinButt,
			PartA:  frontID, FaceA: FaceLeft,
			PartB:  leftID, FaceB: FaceID("inside"), // invalid
			Params: ButtJoinParams{},
		},
	})
	g.AddNode(&Node{
		ID: groupID, Kind: NodeGroup, Name: "group",
		Children: []NodeID{frontID, leftID, joinID},
		Data:     GroupData{},
	})
	g.AddRoot(groupID)

	errs := Validate(g)
	if !hasError(errs, "invalid face_b") {
		t.Error("expected invalid face_b error, got none")
		for _, e := range errs {
			t.Logf("  %s", e)
		}
	}
}

func TestValidate_SelfJoin(t *testing.T) {
	g := New()

	frontID := NewNodeID("defpart/front")
	joinID := NewNodeID("join/self")
	groupID := NewNodeID("group/test")

	g.AddNode(&Node{
		ID: frontID, Kind: NodePrimitive, Name: "front",
		Data: BoardData{PrimKind: PrimBoard, Dimensions: Vec3{400, 200, 19}},
	})
	g.AddNode(&Node{
		ID: joinID, Kind: NodeJoin,
		Data: JoinData{
			Kind:   JoinButt,
			PartA:  frontID, FaceA: FaceLeft,
			PartB:  frontID, FaceB: FaceRight,
			Params: ButtJoinParams{},
		},
	})
	g.AddNode(&Node{
		ID: groupID, Kind: NodeGroup, Name: "group",
		Children: []NodeID{frontID, joinID},
		Data:     GroupData{},
	})
	g.AddRoot(groupID)

	errs := Validate(g)
	if !hasError(errs, "self-join") {
		t.Error("expected self-join error, got none")
		for _, e := range errs {
			t.Logf("  %s", e)
		}
	}
}

func TestValidate_OrphanNode(t *testing.T) {
	g := New()

	frontID := NewNodeID("defpart/front")
	orphanID := NewNodeID("defpart/orphan")
	groupID := NewNodeID("group/test")

	g.AddNode(&Node{
		ID: frontID, Kind: NodePrimitive, Name: "front",
		Data: BoardData{PrimKind: PrimBoard, Dimensions: Vec3{400, 200, 19}},
	})
	g.AddNode(&Node{
		ID: orphanID, Kind: NodePrimitive, Name: "orphan",
		Data: BoardData{PrimKind: PrimBoard, Dimensions: Vec3{100, 100, 19}},
	})
	g.AddNode(&Node{
		ID:       groupID,
		Kind:     NodeGroup,
		Name:     "group",
		Children: []NodeID{frontID}, // orphanID not included
		Data:     GroupData{},
	})
	g.AddRoot(groupID)

	errs := Validate(g)
	if !hasWarning(errs, "orphan") {
		t.Error("expected orphan warning, got none")
		for _, e := range errs {
			t.Logf("  %s", e)
		}
	}
	// Orphan should be a warning, not an error.
	if errorCount(errs) != 0 {
		t.Errorf("expected 0 errors for orphan-only graph, got %d", errorCount(errs))
		for _, e := range errs {
			t.Logf("  %s", e)
		}
	}
}

func TestValidate_JoinReferencingNonPrimitive(t *testing.T) {
	g := New()

	frontID := NewNodeID("defpart/front")
	subgroupID := NewNodeID("group/sub")
	joinID := NewNodeID("join/bad")
	groupID := NewNodeID("group/test")

	g.AddNode(&Node{
		ID: frontID, Kind: NodePrimitive, Name: "front",
		Data: BoardData{PrimKind: PrimBoard, Dimensions: Vec3{400, 200, 19}},
	})
	g.AddNode(&Node{
		ID: subgroupID, Kind: NodeGroup, Name: "sub",
		Data: GroupData{Description: "not a primitive"},
	})
	g.AddNode(&Node{
		ID: joinID, Kind: NodeJoin,
		Data: JoinData{
			Kind:   JoinButt,
			PartA:  frontID, FaceA: FaceLeft,
			PartB:  subgroupID, FaceB: FaceRight, // group, not primitive
			Params: ButtJoinParams{},
		},
	})
	g.AddNode(&Node{
		ID:       groupID,
		Kind:     NodeGroup,
		Name:     "root",
		Children: []NodeID{frontID, subgroupID, joinID},
		Data:     GroupData{},
	})
	g.AddRoot(groupID)

	errs := Validate(g)
	if !hasError(errs, "not primitive") {
		t.Error("expected non-primitive join part error, got none")
		for _, e := range errs {
			t.Logf("  %s", e)
		}
	}
}

func TestValidate_NameIndexPointsToMissingNode(t *testing.T) {
	g := New()

	groupID := NewNodeID("group/test")
	missingID := NewNodeID("defpart/ghost")

	g.AddNode(&Node{
		ID: groupID, Kind: NodeGroup, Name: "root",
		Data: GroupData{},
	})
	g.AddRoot(groupID)

	// Manually inject a stale name index entry.
	g.NameIndex["ghost"] = missingID

	errs := Validate(g)
	if !hasError(errs, "non-existent node") {
		t.Error("expected stale name index error, got none")
		for _, e := range errs {
			t.Logf("  %s", e)
		}
	}
}

func TestValidate_RootReferencesNonExistentNode(t *testing.T) {
	g := New()

	missingRootID := NewNodeID("root/missing")
	g.AddRoot(missingRootID)

	errs := Validate(g)
	if !hasError(errs, "root reference") {
		t.Error("expected missing root error, got none")
		for _, e := range errs {
			t.Logf("  %s", e)
		}
	}
}

func TestValidate_JoinPartANonPrimitive(t *testing.T) {
	g := New()

	transformID := NewNodeID("transform/t")
	boardID := NewNodeID("defpart/board")
	joinID := NewNodeID("join/bad")
	groupID := NewNodeID("group/test")

	g.AddNode(&Node{
		ID: transformID, Kind: NodeTransform, Name: "tx",
		Data: TransformData{Translation: &Vec3{10, 0, 0}},
	})
	g.AddNode(&Node{
		ID: boardID, Kind: NodePrimitive, Name: "board",
		Data: BoardData{PrimKind: PrimBoard, Dimensions: Vec3{400, 200, 19}},
	})
	g.AddNode(&Node{
		ID: joinID, Kind: NodeJoin,
		Data: JoinData{
			Kind:   JoinButt,
			PartA:  transformID, FaceA: FaceLeft, // transform, not primitive
			PartB:  boardID, FaceB: FaceRight,
			Params: ButtJoinParams{},
		},
	})
	g.AddNode(&Node{
		ID:       groupID,
		Kind:     NodeGroup,
		Name:     "root",
		Children: []NodeID{transformID, boardID, joinID},
		Data:     GroupData{},
	})
	g.AddRoot(groupID)

	errs := Validate(g)
	if !hasError(errs, "part_a") && !hasError(errs, "not primitive") {
		t.Error("expected non-primitive part_a error, got none")
		for _, e := range errs {
			t.Logf("  %s", e)
		}
	}
}

func TestValidate_MultipleErrors(t *testing.T) {
	// Graph with multiple problems: self-join + invalid face + orphan.
	g := New()

	frontID := NewNodeID("defpart/front")
	orphanID := NewNodeID("defpart/orphan")
	joinID := NewNodeID("join/bad")
	groupID := NewNodeID("group/test")

	g.AddNode(&Node{
		ID: frontID, Kind: NodePrimitive, Name: "front",
		Data: BoardData{PrimKind: PrimBoard, Dimensions: Vec3{400, 200, 19}},
	})
	g.AddNode(&Node{
		ID: orphanID, Kind: NodePrimitive, Name: "orphan",
		Data: BoardData{PrimKind: PrimBoard, Dimensions: Vec3{100, 100, 19}},
	})
	g.AddNode(&Node{
		ID: joinID, Kind: NodeJoin,
		Data: JoinData{
			Kind:   JoinButt,
			PartA:  frontID, FaceA: FaceID("upward"), // invalid face
			PartB:  frontID, FaceB: FaceRight,         // self-join
			Params: ButtJoinParams{},
		},
	})
	g.AddNode(&Node{
		ID:       groupID,
		Kind:     NodeGroup,
		Name:     "root",
		Children: []NodeID{frontID, joinID},
		Data:     GroupData{},
	})
	g.AddRoot(groupID)

	errs := Validate(g)

	if !hasError(errs, "self-join") {
		t.Error("expected self-join error")
	}
	if !hasError(errs, "invalid face_a") {
		t.Error("expected invalid face_a error")
	}
	if !hasWarning(errs, "orphan") {
		t.Error("expected orphan warning")
	}
}

func TestValidationError_String(t *testing.T) {
	// Graph-level error (zero NodeID).
	e1 := ValidationError{
		Message:  "test graph error",
		Severity: SeverityError,
	}
	if !strings.Contains(e1.Error(), "error") {
		t.Errorf("expected 'error' in string, got %q", e1.Error())
	}
	if !strings.Contains(e1.Error(), "test graph error") {
		t.Errorf("expected message in string, got %q", e1.Error())
	}

	// Node-level warning.
	e2 := ValidationError{
		NodeID:   NewNodeID("test"),
		Message:  "test node warning",
		Severity: SeverityWarning,
	}
	if !strings.Contains(e2.Error(), "warning") {
		t.Errorf("expected 'warning' in string, got %q", e2.Error())
	}
	if !strings.Contains(e2.Error(), "node") {
		t.Errorf("expected 'node' in string, got %q", e2.Error())
	}
}
