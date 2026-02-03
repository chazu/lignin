package graph

import (
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Test helpers for ValidationResult
// ---------------------------------------------------------------------------

// resultHasError returns true if result.Errors contains at least one entry
// whose Message contains substr.
func resultHasError(r ValidationResult, substr string) bool {
	for _, e := range r.Errors {
		if strings.Contains(e.Message, substr) {
			return true
		}
	}
	return false
}

// resultHasWarning returns true if result.Warnings contains at least one entry
// whose Message contains substr.
func resultHasWarning(r ValidationResult, substr string) bool {
	for _, w := range r.Warnings {
		if strings.Contains(w.Message, substr) {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// Tier 2 — Geometric validation tests
// ---------------------------------------------------------------------------

func TestValidateAll_ZeroDimensionBoard(t *testing.T) {
	g := New()

	boardID := NewNodeID("defpart/bad-board")
	groupID := NewNodeID("group/test")

	g.AddNode(&Node{
		ID: boardID, Kind: NodePrimitive, Name: "bad-board",
		Data: BoardData{
			PrimKind:   PrimBoard,
			Dimensions: Vec3{0, 200, 19}, // X is zero
			Grain:      AxisX,
		},
	})
	g.AddNode(&Node{
		ID: groupID, Kind: NodeGroup, Name: "root",
		Children: []NodeID{boardID},
		Data:     GroupData{},
	})
	g.AddRoot(groupID)

	result := ValidateAll(g)
	if !resultHasError(result, "dimension X") {
		t.Error("expected error about zero X dimension, got none")
		for _, e := range result.Errors {
			t.Logf("  error: %s", e.Message)
		}
	}
}

func TestValidateAll_NegativeDimensionBoard(t *testing.T) {
	g := New()

	boardID := NewNodeID("defpart/neg-board")
	groupID := NewNodeID("group/test")

	g.AddNode(&Node{
		ID: boardID, Kind: NodePrimitive, Name: "neg-board",
		Data: BoardData{
			PrimKind:   PrimBoard,
			Dimensions: Vec3{400, -5, 19}, // Y is negative
			Grain:      AxisX,
		},
	})
	g.AddNode(&Node{
		ID: groupID, Kind: NodeGroup, Name: "root",
		Children: []NodeID{boardID},
		Data:     GroupData{},
	})
	g.AddRoot(groupID)

	result := ValidateAll(g)
	if !resultHasError(result, "dimension Y") {
		t.Error("expected error about negative Y dimension, got none")
		for _, e := range result.Errors {
			t.Logf("  error: %s", e.Message)
		}
	}
}

func TestValidateAll_AllZeroDimensions(t *testing.T) {
	g := New()

	boardID := NewNodeID("defpart/zero-board")
	groupID := NewNodeID("group/test")

	g.AddNode(&Node{
		ID: boardID, Kind: NodePrimitive, Name: "zero-board",
		Data: BoardData{
			PrimKind:   PrimBoard,
			Dimensions: Vec3{0, 0, 0},
			Grain:      AxisX,
		},
	})
	g.AddNode(&Node{
		ID: groupID, Kind: NodeGroup, Name: "root",
		Children: []NodeID{boardID},
		Data:     GroupData{},
	})
	g.AddRoot(groupID)

	result := ValidateAll(g)

	// Should have errors for all three dimensions.
	errCount := 0
	for _, e := range result.Errors {
		if strings.Contains(e.Message, "dimension") && strings.Contains(e.Message, "must be positive") {
			errCount++
		}
	}
	if errCount != 3 {
		t.Errorf("expected 3 dimension errors, got %d", errCount)
		for _, e := range result.Errors {
			t.Logf("  error: %s", e.Message)
		}
	}
}

func TestValidateAll_SelfJoinProducesError(t *testing.T) {
	// Self-join is already caught by Tier 1 (validateJoinParts), but
	// ValidateAll should surface it in the Errors field.
	g := New()

	boardID := NewNodeID("defpart/board")
	joinID := NewNodeID("join/self")
	groupID := NewNodeID("group/test")

	g.AddNode(&Node{
		ID: boardID, Kind: NodePrimitive, Name: "board",
		Data: BoardData{PrimKind: PrimBoard, Dimensions: Vec3{400, 200, 19}, Grain: AxisX},
	})
	g.AddNode(&Node{
		ID: joinID, Kind: NodeJoin,
		Data: JoinData{
			Kind:   JoinButt,
			PartA:  boardID, FaceA: FaceLeft,
			PartB:  boardID, FaceB: FaceRight,
			Params: ButtJoinParams{},
		},
	})
	g.AddNode(&Node{
		ID: groupID, Kind: NodeGroup, Name: "root",
		Children: []NodeID{boardID, joinID},
		Data:     GroupData{},
	})
	g.AddRoot(groupID)

	result := ValidateAll(g)
	if !resultHasError(result, "self-join") {
		t.Error("expected self-join error from ValidateAll")
		for _, e := range result.Errors {
			t.Logf("  error: %s", e.Message)
		}
	}
}

func TestValidateAll_DuplicateJoin(t *testing.T) {
	g := New()

	frontID := NewNodeID("defpart/front")
	leftID := NewNodeID("defpart/left")
	join1ID := NewNodeID("join/1")
	join2ID := NewNodeID("join/2")
	groupID := NewNodeID("group/test")

	g.AddNode(&Node{
		ID: frontID, Kind: NodePrimitive, Name: "front",
		Data: BoardData{PrimKind: PrimBoard, Dimensions: Vec3{400, 200, 19}, Grain: AxisX},
	})
	g.AddNode(&Node{
		ID: leftID, Kind: NodePrimitive, Name: "left",
		Data: BoardData{PrimKind: PrimBoard, Dimensions: Vec3{262, 200, 19}, Grain: AxisX},
	})
	g.AddNode(&Node{
		ID: join1ID, Kind: NodeJoin,
		Data: JoinData{
			Kind:   JoinButt,
			PartA:  frontID, FaceA: FaceLeft,
			PartB:  leftID, FaceB: FaceFront,
			Params: ButtJoinParams{},
		},
	})
	// Duplicate: same parts, same faces.
	g.AddNode(&Node{
		ID: join2ID, Kind: NodeJoin,
		Data: JoinData{
			Kind:   JoinButt,
			PartA:  frontID, FaceA: FaceLeft,
			PartB:  leftID, FaceB: FaceFront,
			Params: ButtJoinParams{},
		},
	})
	g.AddNode(&Node{
		ID: groupID, Kind: NodeGroup, Name: "root",
		Children: []NodeID{frontID, leftID, join1ID, join2ID},
		Data:     GroupData{},
	})
	g.AddRoot(groupID)

	result := ValidateAll(g)
	if !resultHasError(result, "duplicate join") {
		t.Error("expected duplicate join error")
		for _, e := range result.Errors {
			t.Logf("  error: %s", e.Message)
		}
	}
}

func TestValidateAll_DuplicateJoinReversedOrder(t *testing.T) {
	// Duplicate join where second join has parts in reversed order.
	g := New()

	frontID := NewNodeID("defpart/front")
	leftID := NewNodeID("defpart/left")
	join1ID := NewNodeID("join/1")
	join2ID := NewNodeID("join/2")
	groupID := NewNodeID("group/test")

	g.AddNode(&Node{
		ID: frontID, Kind: NodePrimitive, Name: "front",
		Data: BoardData{PrimKind: PrimBoard, Dimensions: Vec3{400, 200, 19}, Grain: AxisX},
	})
	g.AddNode(&Node{
		ID: leftID, Kind: NodePrimitive, Name: "left",
		Data: BoardData{PrimKind: PrimBoard, Dimensions: Vec3{262, 200, 19}, Grain: AxisX},
	})
	g.AddNode(&Node{
		ID: join1ID, Kind: NodeJoin,
		Data: JoinData{
			Kind:   JoinButt,
			PartA:  frontID, FaceA: FaceLeft,
			PartB:  leftID, FaceB: FaceFront,
			Params: ButtJoinParams{},
		},
	})
	// Same pair, reversed: PartA=left, PartB=front.
	g.AddNode(&Node{
		ID: join2ID, Kind: NodeJoin,
		Data: JoinData{
			Kind:   JoinButt,
			PartA:  leftID, FaceA: FaceFront,
			PartB:  frontID, FaceB: FaceLeft,
			Params: ButtJoinParams{},
		},
	})
	g.AddNode(&Node{
		ID: groupID, Kind: NodeGroup, Name: "root",
		Children: []NodeID{frontID, leftID, join1ID, join2ID},
		Data:     GroupData{},
	})
	g.AddRoot(groupID)

	result := ValidateAll(g)
	if !resultHasError(result, "duplicate join") {
		t.Error("expected duplicate join error for reversed part order")
		for _, e := range result.Errors {
			t.Logf("  error: %s", e.Message)
		}
	}
}

func TestValidateAll_DifferentFacesNotDuplicate(t *testing.T) {
	// Two joins on the same parts but different faces should NOT be duplicates.
	g := New()

	frontID := NewNodeID("defpart/front")
	leftID := NewNodeID("defpart/left")
	join1ID := NewNodeID("join/1")
	join2ID := NewNodeID("join/2")
	groupID := NewNodeID("group/test")

	g.AddNode(&Node{
		ID: frontID, Kind: NodePrimitive, Name: "front",
		Data: BoardData{PrimKind: PrimBoard, Dimensions: Vec3{400, 200, 19}, Grain: AxisX},
	})
	g.AddNode(&Node{
		ID: leftID, Kind: NodePrimitive, Name: "left",
		Data: BoardData{PrimKind: PrimBoard, Dimensions: Vec3{262, 200, 19}, Grain: AxisX},
	})
	g.AddNode(&Node{
		ID: join1ID, Kind: NodeJoin,
		Data: JoinData{
			Kind:   JoinButt,
			PartA:  frontID, FaceA: FaceLeft,
			PartB:  leftID, FaceB: FaceFront,
			Params: ButtJoinParams{},
		},
	})
	// Different faces.
	g.AddNode(&Node{
		ID: join2ID, Kind: NodeJoin,
		Data: JoinData{
			Kind:   JoinButt,
			PartA:  frontID, FaceA: FaceTop,
			PartB:  leftID, FaceB: FaceBottom,
			Params: ButtJoinParams{},
		},
	})
	g.AddNode(&Node{
		ID: groupID, Kind: NodeGroup, Name: "root",
		Children: []NodeID{frontID, leftID, join1ID, join2ID},
		Data:     GroupData{},
	})
	g.AddRoot(groupID)

	result := ValidateAll(g)
	if resultHasError(result, "duplicate join") {
		t.Error("should not flag different-face joins as duplicates")
	}
}

func TestValidateAll_FastenerTooLong(t *testing.T) {
	g := New()

	frontID := NewNodeID("defpart/front")
	leftID := NewNodeID("defpart/left")
	fastenerID := NewNodeID("fastener/long-screw")
	joinID := NewNodeID("join/test")
	groupID := NewNodeID("group/test")

	// Front board: 400x200x19. Joining on FaceLeft => thickness along X = 400.
	// Left board: 262x200x19. Joining on FaceFront => thickness along Z = 19.
	// Combined thickness = 400 + 19 = 419.
	// Fastener length 500 > 419 => should warn.
	g.AddNode(&Node{
		ID: frontID, Kind: NodePrimitive, Name: "front",
		Data: BoardData{PrimKind: PrimBoard, Dimensions: Vec3{400, 200, 19}, Grain: AxisX},
	})
	g.AddNode(&Node{
		ID: leftID, Kind: NodePrimitive, Name: "left",
		Data: BoardData{PrimKind: PrimBoard, Dimensions: Vec3{262, 200, 19}, Grain: AxisX},
	})
	g.AddNode(&Node{
		ID: fastenerID, Kind: NodeFastener,
		Data: FastenerData{
			Kind:     FastenerScrew,
			Diameter: 4,
			Length:   500, // exceeds 400 + 19 = 419
			JoinRef:  joinID,
		},
	})
	g.AddNode(&Node{
		ID: joinID, Kind: NodeJoin,
		Data: JoinData{
			Kind:      JoinButt,
			PartA:     frontID, FaceA: FaceLeft,
			PartB:     leftID, FaceB: FaceFront,
			Params:    ButtJoinParams{},
			Fasteners: []NodeID{fastenerID},
		},
	})
	g.AddNode(&Node{
		ID: groupID, Kind: NodeGroup, Name: "root",
		Children: []NodeID{frontID, leftID, joinID, fastenerID},
		Data:     GroupData{},
	})
	g.AddRoot(groupID)

	result := ValidateAll(g)
	if !resultHasWarning(result, "fastener length") {
		t.Error("expected fastener-too-long warning")
		for _, w := range result.Warnings {
			t.Logf("  warning: %s", w.Message)
		}
		for _, e := range result.Errors {
			t.Logf("  error: %s", e.Message)
		}
	}
}

func TestValidateAll_FastenerFitsOk(t *testing.T) {
	g := New()

	frontID := NewNodeID("defpart/front")
	leftID := NewNodeID("defpart/left")
	fastenerID := NewNodeID("fastener/short-screw")
	joinID := NewNodeID("join/test")
	groupID := NewNodeID("group/test")

	// Joining on FaceLeft (X=400) + FaceFront (Z=19) = 419. Fastener = 30. OK.
	g.AddNode(&Node{
		ID: frontID, Kind: NodePrimitive, Name: "front",
		Data: BoardData{PrimKind: PrimBoard, Dimensions: Vec3{400, 200, 19}, Grain: AxisX},
	})
	g.AddNode(&Node{
		ID: leftID, Kind: NodePrimitive, Name: "left",
		Data: BoardData{PrimKind: PrimBoard, Dimensions: Vec3{262, 200, 19}, Grain: AxisX},
	})
	g.AddNode(&Node{
		ID: fastenerID, Kind: NodeFastener,
		Data: FastenerData{
			Kind:     FastenerScrew,
			Diameter: 4,
			Length:   30, // well within 400 + 19 = 419
			JoinRef:  joinID,
		},
	})
	g.AddNode(&Node{
		ID: joinID, Kind: NodeJoin,
		Data: JoinData{
			Kind:      JoinButt,
			PartA:     frontID, FaceA: FaceLeft,
			PartB:     leftID, FaceB: FaceFront,
			Params:    ButtJoinParams{},
			Fasteners: []NodeID{fastenerID},
		},
	})
	g.AddNode(&Node{
		ID: groupID, Kind: NodeGroup, Name: "root",
		Children: []NodeID{frontID, leftID, joinID, fastenerID},
		Data:     GroupData{},
	})
	g.AddRoot(groupID)

	result := ValidateAll(g)
	if resultHasWarning(result, "fastener length") {
		t.Error("should not warn about fastener that fits")
	}
}

// ---------------------------------------------------------------------------
// Tier 3 — Material warning tests
// ---------------------------------------------------------------------------

func TestValidateAll_EndGrainButtJoint(t *testing.T) {
	g := New()

	// Two boards with grain along X. Left and Right are end-grain faces.
	boardAID := NewNodeID("defpart/a")
	boardBID := NewNodeID("defpart/b")
	joinID := NewNodeID("join/endgrain")
	groupID := NewNodeID("group/test")

	g.AddNode(&Node{
		ID: boardAID, Kind: NodePrimitive, Name: "board-a",
		Data: BoardData{PrimKind: PrimBoard, Dimensions: Vec3{400, 200, 19}, Grain: AxisX},
	})
	g.AddNode(&Node{
		ID: boardBID, Kind: NodePrimitive, Name: "board-b",
		Data: BoardData{PrimKind: PrimBoard, Dimensions: Vec3{400, 200, 19}, Grain: AxisX},
	})
	// Both faces are end-grain: FaceLeft (A) and FaceRight (B) are both
	// perpendicular to grain X.
	g.AddNode(&Node{
		ID: joinID, Kind: NodeJoin,
		Data: JoinData{
			Kind:   JoinButt,
			PartA:  boardAID, FaceA: FaceLeft,
			PartB:  boardBID, FaceB: FaceRight,
			Params: ButtJoinParams{GlueUp: true},
		},
	})
	g.AddNode(&Node{
		ID: groupID, Kind: NodeGroup, Name: "root",
		Children: []NodeID{boardAID, boardBID, joinID},
		Data:     GroupData{},
	})
	g.AddRoot(groupID)

	result := ValidateAll(g)
	if !resultHasWarning(result, "end-grain") {
		t.Error("expected end-grain butt joint warning")
		for _, w := range result.Warnings {
			t.Logf("  warning: %s", w.Message)
		}
	}
}

func TestValidateAll_EndGrainGrainY(t *testing.T) {
	g := New()

	// Grain Y: end-grain faces are front and back.
	boardAID := NewNodeID("defpart/a")
	boardBID := NewNodeID("defpart/b")
	joinID := NewNodeID("join/endgrain-y")
	groupID := NewNodeID("group/test")

	g.AddNode(&Node{
		ID: boardAID, Kind: NodePrimitive, Name: "board-a",
		Data: BoardData{PrimKind: PrimBoard, Dimensions: Vec3{400, 200, 19}, Grain: AxisY},
	})
	g.AddNode(&Node{
		ID: boardBID, Kind: NodePrimitive, Name: "board-b",
		Data: BoardData{PrimKind: PrimBoard, Dimensions: Vec3{400, 200, 19}, Grain: AxisY},
	})
	g.AddNode(&Node{
		ID: joinID, Kind: NodeJoin,
		Data: JoinData{
			Kind:   JoinButt,
			PartA:  boardAID, FaceA: FaceFront,
			PartB:  boardBID, FaceB: FaceBack,
			Params: ButtJoinParams{},
		},
	})
	g.AddNode(&Node{
		ID: groupID, Kind: NodeGroup, Name: "root",
		Children: []NodeID{boardAID, boardBID, joinID},
		Data:     GroupData{},
	})
	g.AddRoot(groupID)

	result := ValidateAll(g)
	if !resultHasWarning(result, "end-grain") {
		t.Error("expected end-grain warning for grain Y front/back joint")
	}
}

func TestValidateAll_EndGrainGrainZ(t *testing.T) {
	g := New()

	// Grain Z: end-grain faces are top and bottom.
	boardAID := NewNodeID("defpart/a")
	boardBID := NewNodeID("defpart/b")
	joinID := NewNodeID("join/endgrain-z")
	groupID := NewNodeID("group/test")

	g.AddNode(&Node{
		ID: boardAID, Kind: NodePrimitive, Name: "board-a",
		Data: BoardData{PrimKind: PrimBoard, Dimensions: Vec3{400, 200, 19}, Grain: AxisZ},
	})
	g.AddNode(&Node{
		ID: boardBID, Kind: NodePrimitive, Name: "board-b",
		Data: BoardData{PrimKind: PrimBoard, Dimensions: Vec3{400, 200, 19}, Grain: AxisZ},
	})
	g.AddNode(&Node{
		ID: joinID, Kind: NodeJoin,
		Data: JoinData{
			Kind:   JoinButt,
			PartA:  boardAID, FaceA: FaceTop,
			PartB:  boardBID, FaceB: FaceBottom,
			Params: ButtJoinParams{},
		},
	})
	g.AddNode(&Node{
		ID: groupID, Kind: NodeGroup, Name: "root",
		Children: []NodeID{boardAID, boardBID, joinID},
		Data:     GroupData{},
	})
	g.AddRoot(groupID)

	result := ValidateAll(g)
	if !resultHasWarning(result, "end-grain") {
		t.Error("expected end-grain warning for grain Z top/bottom joint")
	}
}

func TestValidateAll_LongGrainButtJointNoWarning(t *testing.T) {
	g := New()

	// Grain X, joining on FaceTop and FaceBottom: these are NOT end-grain
	// faces for grain X. Should produce no end-grain warning.
	boardAID := NewNodeID("defpart/a")
	boardBID := NewNodeID("defpart/b")
	joinID := NewNodeID("join/longgrain")
	groupID := NewNodeID("group/test")

	g.AddNode(&Node{
		ID: boardAID, Kind: NodePrimitive, Name: "board-a",
		Data: BoardData{PrimKind: PrimBoard, Dimensions: Vec3{400, 200, 19}, Grain: AxisX},
	})
	g.AddNode(&Node{
		ID: boardBID, Kind: NodePrimitive, Name: "board-b",
		Data: BoardData{PrimKind: PrimBoard, Dimensions: Vec3{400, 200, 19}, Grain: AxisX},
	})
	g.AddNode(&Node{
		ID: joinID, Kind: NodeJoin,
		Data: JoinData{
			Kind:   JoinButt,
			PartA:  boardAID, FaceA: FaceTop,    // long-grain for X
			PartB:  boardBID, FaceB: FaceBottom,  // long-grain for X
			Params: ButtJoinParams{},
		},
	})
	g.AddNode(&Node{
		ID: groupID, Kind: NodeGroup, Name: "root",
		Children: []NodeID{boardAID, boardBID, joinID},
		Data:     GroupData{},
	})
	g.AddRoot(groupID)

	result := ValidateAll(g)
	if resultHasWarning(result, "end-grain") {
		t.Error("should NOT warn about long-grain to long-grain joint")
	}
}

func TestValidateAll_MixedGrainNoWarning(t *testing.T) {
	g := New()

	// One face is end-grain, the other is not. Should NOT warn (only warns
	// when BOTH faces are end-grain).
	boardAID := NewNodeID("defpart/a")
	boardBID := NewNodeID("defpart/b")
	joinID := NewNodeID("join/mixed")
	groupID := NewNodeID("group/test")

	g.AddNode(&Node{
		ID: boardAID, Kind: NodePrimitive, Name: "board-a",
		Data: BoardData{PrimKind: PrimBoard, Dimensions: Vec3{400, 200, 19}, Grain: AxisX},
	})
	g.AddNode(&Node{
		ID: boardBID, Kind: NodePrimitive, Name: "board-b",
		Data: BoardData{PrimKind: PrimBoard, Dimensions: Vec3{400, 200, 19}, Grain: AxisX},
	})
	g.AddNode(&Node{
		ID: joinID, Kind: NodeJoin,
		Data: JoinData{
			Kind:   JoinButt,
			PartA:  boardAID, FaceA: FaceLeft,  // end-grain for X
			PartB:  boardBID, FaceB: FaceTop,   // NOT end-grain for X
			Params: ButtJoinParams{},
		},
	})
	g.AddNode(&Node{
		ID: groupID, Kind: NodeGroup, Name: "root",
		Children: []NodeID{boardAID, boardBID, joinID},
		Data:     GroupData{},
	})
	g.AddRoot(groupID)

	result := ValidateAll(g)
	if resultHasWarning(result, "end-grain") {
		t.Error("should NOT warn when only one face is end-grain")
	}
}

// ---------------------------------------------------------------------------
// Valid graph produces no errors or warnings
// ---------------------------------------------------------------------------

func TestValidateAll_ValidGraph(t *testing.T) {
	g := buildValidBox()

	result := ValidateAll(g)
	if len(result.Errors) != 0 {
		t.Errorf("expected 0 errors, got %d", len(result.Errors))
		for _, e := range result.Errors {
			t.Logf("  error: %s", e.Message)
		}
	}
	if len(result.Warnings) != 0 {
		t.Errorf("expected 0 warnings, got %d", len(result.Warnings))
		for _, w := range result.Warnings {
			t.Logf("  warning: %s", w.Message)
		}
	}
}

func TestValidateAll_EmptyGraph(t *testing.T) {
	g := New()

	result := ValidateAll(g)
	if len(result.Errors) != 0 {
		t.Errorf("expected 0 errors on empty graph, got %d", len(result.Errors))
	}
	if len(result.Warnings) != 0 {
		t.Errorf("expected 0 warnings on empty graph, got %d", len(result.Warnings))
	}
}

// ---------------------------------------------------------------------------
// isEndGrainFace unit tests
// ---------------------------------------------------------------------------

func TestIsEndGrainFace(t *testing.T) {
	tests := []struct {
		grain Axis
		face  FaceID
		want  bool
	}{
		// Grain X: end-grain is left/right.
		{AxisX, FaceLeft, true},
		{AxisX, FaceRight, true},
		{AxisX, FaceTop, false},
		{AxisX, FaceBottom, false},
		{AxisX, FaceFront, false},
		{AxisX, FaceBack, false},
		// Grain Y: end-grain is front/back.
		{AxisY, FaceFront, true},
		{AxisY, FaceBack, true},
		{AxisY, FaceLeft, false},
		{AxisY, FaceRight, false},
		{AxisY, FaceTop, false},
		{AxisY, FaceBottom, false},
		// Grain Z: end-grain is top/bottom.
		{AxisZ, FaceTop, true},
		{AxisZ, FaceBottom, true},
		{AxisZ, FaceLeft, false},
		{AxisZ, FaceRight, false},
		{AxisZ, FaceFront, false},
		{AxisZ, FaceBack, false},
	}

	for _, tt := range tests {
		got := isEndGrainFace(tt.grain, tt.face)
		if got != tt.want {
			t.Errorf("isEndGrainFace(%s, %s) = %v, want %v", tt.grain, tt.face, got, tt.want)
		}
	}
}

// ---------------------------------------------------------------------------
// faceThickness unit tests
// ---------------------------------------------------------------------------

func TestFaceThickness(t *testing.T) {
	bd := BoardData{
		PrimKind:   PrimBoard,
		Dimensions: Vec3{X: 400, Y: 200, Z: 19},
	}

	tests := []struct {
		face FaceID
		want float64
	}{
		{FaceTop, 200},    // Y
		{FaceBottom, 200}, // Y
		{FaceLeft, 400},   // X
		{FaceRight, 400},  // X
		{FaceFront, 19},   // Z
		{FaceBack, 19},    // Z
	}

	for _, tt := range tests {
		got := faceThickness(bd, tt.face)
		if got != tt.want {
			t.Errorf("faceThickness(face=%s) = %f, want %f", tt.face, got, tt.want)
		}
	}
}
