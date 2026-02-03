package graph

import "fmt"

// ---------------------------------------------------------------------------
// Tier 2 — Geometric validation (errors + warnings)
// ---------------------------------------------------------------------------

// validateGeometry runs all Tier 2 geometric checks.
// Returns errors (blocking) and warnings (advisory) separately.
func validateGeometry(g *DesignGraph) ([]ValidationError, []ValidationWarning) {
	var errs []ValidationError
	var warnings []ValidationWarning

	errs = append(errs, validateNonZeroDimensions(g)...)
	errs = append(errs, validateDuplicateJoins(g)...)

	fastenerWarnings := validateFastenerLength(g)
	warnings = append(warnings, fastenerWarnings...)

	return errs, warnings
}

// validateNonZeroDimensions checks that every BoardData has positive X, Y, Z.
func validateNonZeroDimensions(g *DesignGraph) []ValidationError {
	var errs []ValidationError

	for _, node := range g.Nodes {
		bd, ok := node.Data.(BoardData)
		if !ok {
			continue
		}

		if bd.Dimensions.X <= 0 {
			errs = append(errs, ValidationError{
				NodeID:   node.ID,
				Message:  fmt.Sprintf("board dimension X is %.4f, must be positive", bd.Dimensions.X),
				Severity: SeverityError,
			})
		}
		if bd.Dimensions.Y <= 0 {
			errs = append(errs, ValidationError{
				NodeID:   node.ID,
				Message:  fmt.Sprintf("board dimension Y is %.4f, must be positive", bd.Dimensions.Y),
				Severity: SeverityError,
			})
		}
		if bd.Dimensions.Z <= 0 {
			errs = append(errs, ValidationError{
				NodeID:   node.ID,
				Message:  fmt.Sprintf("board dimension Z is %.4f, must be positive", bd.Dimensions.Z),
				Severity: SeverityError,
			})
		}
	}

	return errs
}

// joinKey produces a canonical key for a pair of parts + faces so that
// (A,faceA,B,faceB) and (B,faceB,A,faceA) are treated as the same join.
type joinKey struct {
	partLo, partHi NodeID
	faceLo, faceHi FaceID
}

func makeJoinKey(partA NodeID, faceA FaceID, partB NodeID, faceB FaceID) joinKey {
	// Canonical ordering: compare the raw bytes of the NodeIDs.
	if partA.String() < partB.String() {
		return joinKey{partLo: partA, partHi: partB, faceLo: faceA, faceHi: faceB}
	}
	if partA.String() > partB.String() {
		return joinKey{partLo: partB, partHi: partA, faceLo: faceB, faceHi: faceA}
	}
	// Same part (self-join, caught by Tier 1), order by face.
	if string(faceA) <= string(faceB) {
		return joinKey{partLo: partA, partHi: partB, faceLo: faceA, faceHi: faceB}
	}
	return joinKey{partLo: partB, partHi: partA, faceLo: faceB, faceHi: faceA}
}

// validateDuplicateJoins checks that no two join nodes connect the same
// pair of parts on the same faces.
func validateDuplicateJoins(g *DesignGraph) []ValidationError {
	var errs []ValidationError
	seen := make(map[joinKey]NodeID) // first join node that used this key

	for _, node := range g.Nodes {
		jd, ok := node.Data.(JoinData)
		if !ok {
			continue
		}

		key := makeJoinKey(jd.PartA, jd.FaceA, jd.PartB, jd.FaceB)
		if firstID, exists := seen[key]; exists {
			errs = append(errs, ValidationError{
				NodeID:   node.ID,
				Message:  fmt.Sprintf("duplicate join: same part-face pair already joined by node %s", firstID.Short()),
				Severity: SeverityError,
			})
		} else {
			seen[key] = node.ID
		}
	}

	return errs
}

// faceThickness returns the thickness of a board along the axis perpendicular
// to the given face. For a board with dimensions (X, Y, Z):
//   - top/bottom faces have thickness along Y
//   - left/right faces have thickness along X
//   - front/back faces have thickness along Z
func faceThickness(bd BoardData, face FaceID) float64 {
	switch face {
	case FaceTop, FaceBottom:
		return bd.Dimensions.Y
	case FaceLeft, FaceRight:
		return bd.Dimensions.X
	case FaceFront, FaceBack:
		return bd.Dimensions.Z
	default:
		return 0
	}
}

// validateFastenerLength checks that fastener length does not exceed
// combined thickness of both joined boards (for butt joints).
func validateFastenerLength(g *DesignGraph) []ValidationWarning {
	var warnings []ValidationWarning

	for _, node := range g.Nodes {
		jd, ok := node.Data.(JoinData)
		if !ok {
			continue
		}

		if jd.Kind != JoinButt {
			continue
		}

		// Look up both parts as boards.
		partANode := g.Nodes[jd.PartA]
		partBNode := g.Nodes[jd.PartB]
		if partANode == nil || partBNode == nil {
			continue // dangling references handled by Tier 1
		}

		bdA, okA := partANode.Data.(BoardData)
		bdB, okB := partBNode.Data.(BoardData)
		if !okA || !okB {
			continue // non-board parts; skip
		}

		combinedThickness := faceThickness(bdA, jd.FaceA) + faceThickness(bdB, jd.FaceB)

		for _, fastenerID := range jd.Fasteners {
			fNode := g.Nodes[fastenerID]
			if fNode == nil {
				continue
			}
			fd, ok := fNode.Data.(FastenerData)
			if !ok {
				continue
			}
			if fd.Length > combinedThickness {
				warnings = append(warnings, ValidationWarning{
					NodeID: fNode.ID,
					Message: fmt.Sprintf(
						"fastener length %.1fmm exceeds combined board thickness %.1fmm at joint %s",
						fd.Length, combinedThickness, node.ID.Short(),
					),
				})
			}
		}
	}

	return warnings
}

// ---------------------------------------------------------------------------
// Tier 3 — Material warnings
// ---------------------------------------------------------------------------

// isEndGrainFace returns true if the given face is an end-grain face
// for a board with the specified grain direction.
//
// The end-grain faces are the faces perpendicular to the grain axis:
//   - Grain X: end-grain faces are left and right
//   - Grain Y: end-grain faces are front and back
//   - Grain Z: end-grain faces are top and bottom
func isEndGrainFace(grain Axis, face FaceID) bool {
	switch grain {
	case AxisX:
		return face == FaceLeft || face == FaceRight
	case AxisY:
		return face == FaceFront || face == FaceBack
	case AxisZ:
		return face == FaceTop || face == FaceBottom
	default:
		return false
	}
}

// validateMaterial runs all Tier 3 material advisory checks.
func validateMaterial(g *DesignGraph) []ValidationWarning {
	var warnings []ValidationWarning
	warnings = append(warnings, validateEndGrainButtJoint(g)...)
	return warnings
}

// validateEndGrainButtJoint warns when a butt joint connects two end-grain
// faces. End-grain to end-grain butt joints have very poor glue adhesion.
func validateEndGrainButtJoint(g *DesignGraph) []ValidationWarning {
	var warnings []ValidationWarning

	for _, node := range g.Nodes {
		jd, ok := node.Data.(JoinData)
		if !ok {
			continue
		}

		if jd.Kind != JoinButt {
			continue
		}

		partANode := g.Nodes[jd.PartA]
		partBNode := g.Nodes[jd.PartB]
		if partANode == nil || partBNode == nil {
			continue
		}

		bdA, okA := partANode.Data.(BoardData)
		bdB, okB := partBNode.Data.(BoardData)
		if !okA || !okB {
			continue
		}

		if isEndGrainFace(bdA.Grain, jd.FaceA) && isEndGrainFace(bdB.Grain, jd.FaceB) {
			warnings = append(warnings, ValidationWarning{
				NodeID:  node.ID,
				Message: "end-grain to end-grain butt joint has poor glue adhesion; consider a different joint type or reinforcement",
			})
		}
	}

	return warnings
}
