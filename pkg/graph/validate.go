package graph

import "fmt"

// ValidationSeverity indicates whether a validation finding blocks evaluation
// or is merely informational.
type ValidationSeverity int

const (
	SeverityError   ValidationSeverity = iota // blocks evaluation
	SeverityWarning                           // informational
)

func (s ValidationSeverity) String() string {
	switch s {
	case SeverityError:
		return "error"
	case SeverityWarning:
		return "warning"
	default:
		return fmt.Sprintf("ValidationSeverity(%d)", int(s))
	}
}

// ValidationError describes a single validation finding.
type ValidationError struct {
	NodeID   NodeID             // which node has the problem (zero if graph-level)
	Message  string             // human-readable description
	Severity ValidationSeverity // error or warning
}

func (e ValidationError) Error() string {
	if e.NodeID.IsZero() {
		return fmt.Sprintf("[%s] %s", e.Severity, e.Message)
	}
	return fmt.Sprintf("[%s] node %s: %s", e.Severity, e.NodeID.Short(), e.Message)
}

// ValidationWarning describes a non-blocking advisory finding.
type ValidationWarning struct {
	NodeID  NodeID
	Message string
}

// ValidationResult bundles errors (blocking) and warnings (advisory)
// from all validation tiers.
type ValidationResult struct {
	Errors   []ValidationError
	Warnings []ValidationWarning
}

// Validate runs all Tier 1 structural validation checks on the design graph
// and returns a slice of validation errors. An empty slice means the graph is
// valid. This function is read-only and never mutates the graph.
func Validate(g *DesignGraph) []ValidationError {
	var errs []ValidationError
	errs = append(errs, validateDAG(g)...)
	errs = append(errs, validateReferences(g)...)
	errs = append(errs, validateNames(g)...)
	errs = append(errs, validateRoots(g)...)
	errs = append(errs, validateFaceIDs(g)...)
	errs = append(errs, validateJoinParts(g)...)
	return errs
}

// ValidateAll runs all validation tiers (structural, geometric, material)
// and returns a ValidationResult with separated errors and warnings.
func ValidateAll(g *DesignGraph) ValidationResult {
	// Tier 1: structural validation (existing).
	tier1 := Validate(g)

	// Tier 2: geometric validation.
	tier2Errs, tier2Warnings := validateGeometry(g)

	// Tier 3: material warnings.
	tier3Warnings := validateMaterial(g)

	// Separate Tier 1 findings into errors and warnings.
	var result ValidationResult
	for _, e := range tier1 {
		if e.Severity == SeverityWarning {
			result.Warnings = append(result.Warnings, ValidationWarning{
				NodeID:  e.NodeID,
				Message: e.Message,
			})
		} else {
			result.Errors = append(result.Errors, e)
		}
	}

	result.Errors = append(result.Errors, tier2Errs...)
	result.Warnings = append(result.Warnings, tier2Warnings...)
	result.Warnings = append(result.Warnings, tier3Warnings...)

	return result
}

// validateDAG checks for cycles using DFS with 3-color marking.
// White (0) = unvisited, gray (1) = in current DFS path, black (2) = fully explored.
// If we encounter a gray node during traversal, we have found a cycle.
func validateDAG(g *DesignGraph) []ValidationError {
	const (
		white = iota
		gray
		black
	)

	color := make(map[NodeID]int) // default zero = white
	var errs []ValidationError

	var visit func(id NodeID) bool // returns true if cycle found
	visit = func(id NodeID) bool {
		switch color[id] {
		case black:
			return false
		case gray:
			errs = append(errs, ValidationError{
				NodeID:   id,
				Message:  fmt.Sprintf("cycle detected: node %s is part of a cycle", id.Short()),
				Severity: SeverityError,
			})
			return true
		}

		color[id] = gray

		node, ok := g.Nodes[id]
		if !ok {
			// Dangling reference; handled by validateReferences.
			color[id] = black
			return false
		}

		// Walk Children edges.
		for _, childID := range node.Children {
			if visit(childID) {
				return true
			}
		}

		color[id] = black
		return false
	}

	// Start DFS from every node to catch disconnected components.
	for id := range g.Nodes {
		if color[id] == white {
			if visit(id) {
				// One cycle error is sufficient; stop early.
				break
			}
		}
	}

	return errs
}

// validateReferences checks that every NodeID referenced anywhere in the graph
// points to a node that actually exists in g.Nodes.
func validateReferences(g *DesignGraph) []ValidationError {
	var errs []ValidationError

	for _, node := range g.Nodes {
		// Check Children references.
		for _, childID := range node.Children {
			if _, ok := g.Nodes[childID]; !ok {
				errs = append(errs, ValidationError{
					NodeID:   node.ID,
					Message:  fmt.Sprintf("child reference %s does not exist", childID.Short()),
					Severity: SeverityError,
				})
			}
		}

		// Check kind-specific data references.
		switch d := node.Data.(type) {
		case JoinData:
			if !d.PartA.IsZero() {
				if _, ok := g.Nodes[d.PartA]; !ok {
					errs = append(errs, ValidationError{
						NodeID:   node.ID,
						Message:  fmt.Sprintf("join part_a reference %s does not exist", d.PartA.Short()),
						Severity: SeverityError,
					})
				}
			}
			if !d.PartB.IsZero() {
				if _, ok := g.Nodes[d.PartB]; !ok {
					errs = append(errs, ValidationError{
						NodeID:   node.ID,
						Message:  fmt.Sprintf("join part_b reference %s does not exist", d.PartB.Short()),
						Severity: SeverityError,
					})
				}
			}
			for _, fid := range d.Fasteners {
				if _, ok := g.Nodes[fid]; !ok {
					errs = append(errs, ValidationError{
						NodeID:   node.ID,
						Message:  fmt.Sprintf("join fastener reference %s does not exist", fid.Short()),
						Severity: SeverityError,
					})
				}
			}

		case DrillData:
			if !d.TargetPart.IsZero() {
				if _, ok := g.Nodes[d.TargetPart]; !ok {
					errs = append(errs, ValidationError{
						NodeID:   node.ID,
						Message:  fmt.Sprintf("drill target_part reference %s does not exist", d.TargetPart.Short()),
						Severity: SeverityError,
					})
				}
			}

		case FastenerData:
			if !d.JoinRef.IsZero() {
				if _, ok := g.Nodes[d.JoinRef]; !ok {
					errs = append(errs, ValidationError{
						NodeID:   node.ID,
						Message:  fmt.Sprintf("fastener join_ref reference %s does not exist", d.JoinRef.Short()),
						Severity: SeverityError,
					})
				}
			}
		}
	}

	return errs
}

// validateNames checks that the NameIndex is injective (no two nodes share the
// same name) and that every entry in NameIndex points to an existing node.
func validateNames(g *DesignGraph) []ValidationError {
	var errs []ValidationError

	// Check that every NameIndex entry references an existing node.
	for name, id := range g.NameIndex {
		if _, ok := g.Nodes[id]; !ok {
			errs = append(errs, ValidationError{
				Message:  fmt.Sprintf("name index entry %q references non-existent node %s", name, id.Short()),
				Severity: SeverityError,
			})
		}
	}

	// Check injectivity: build a reverse map from NodeID to name, looking at
	// actual node Name fields. If two nodes share the same non-empty Name, error.
	nameToNodes := make(map[string][]NodeID)
	for id, node := range g.Nodes {
		if node.Name != "" {
			nameToNodes[node.Name] = append(nameToNodes[node.Name], id)
		}
	}
	for name, ids := range nameToNodes {
		if len(ids) > 1 {
			errs = append(errs, ValidationError{
				Message:  fmt.Sprintf("duplicate name %q assigned to %d nodes", name, len(ids)),
				Severity: SeverityError,
			})
		}
	}

	return errs
}

// validateRoots checks that every root ID references an existing node and
// warns about orphan nodes (nodes unreachable from any root).
func validateRoots(g *DesignGraph) []ValidationError {
	var errs []ValidationError

	// Check that each root references an existing node.
	for _, rid := range g.Roots {
		if _, ok := g.Nodes[rid]; !ok {
			errs = append(errs, ValidationError{
				Message:  fmt.Sprintf("root reference %s does not exist", rid.Short()),
				Severity: SeverityError,
			})
		}
	}

	// Orphan detection: BFS from all roots through Children edges.
	if len(g.Nodes) == 0 {
		return errs
	}

	reachable := make(map[NodeID]bool)
	queue := make([]NodeID, 0, len(g.Roots))
	for _, rid := range g.Roots {
		if _, ok := g.Nodes[rid]; ok {
			if !reachable[rid] {
				reachable[rid] = true
				queue = append(queue, rid)
			}
		}
	}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		node := g.Nodes[current]
		if node == nil {
			continue
		}

		// Traverse Children edges.
		for _, childID := range node.Children {
			if !reachable[childID] {
				reachable[childID] = true
				queue = append(queue, childID)
			}
		}

		// Also traverse join/drill/fastener data references to reach
		// nodes that are only referenced via data fields.
		switch d := node.Data.(type) {
		case JoinData:
			if !d.PartA.IsZero() && !reachable[d.PartA] {
				reachable[d.PartA] = true
				queue = append(queue, d.PartA)
			}
			if !d.PartB.IsZero() && !reachable[d.PartB] {
				reachable[d.PartB] = true
				queue = append(queue, d.PartB)
			}
			for _, fid := range d.Fasteners {
				if !reachable[fid] {
					reachable[fid] = true
					queue = append(queue, fid)
				}
			}
		case DrillData:
			if !d.TargetPart.IsZero() && !reachable[d.TargetPart] {
				reachable[d.TargetPart] = true
				queue = append(queue, d.TargetPart)
			}
		case FastenerData:
			if !d.JoinRef.IsZero() && !reachable[d.JoinRef] {
				reachable[d.JoinRef] = true
				queue = append(queue, d.JoinRef)
			}
		}
	}

	// Report any unreachable nodes as warnings.
	for id, node := range g.Nodes {
		if !reachable[id] {
			name := node.Name
			if name == "" {
				name = id.Short()
			}
			errs = append(errs, ValidationError{
				NodeID:   id,
				Message:  fmt.Sprintf("node %q is not reachable from any root (orphan)", name),
				Severity: SeverityWarning,
			})
		}
	}

	return errs
}

// validateFaceIDs checks that every FaceID used in JoinData is a valid face
// (top/bottom/left/right/front/back).
func validateFaceIDs(g *DesignGraph) []ValidationError {
	var errs []ValidationError

	for _, node := range g.Nodes {
		if jd, ok := node.Data.(JoinData); ok {
			if !ValidFaceIDs[jd.FaceA] {
				errs = append(errs, ValidationError{
					NodeID:   node.ID,
					Message:  fmt.Sprintf("invalid face_a %q", jd.FaceA),
					Severity: SeverityError,
				})
			}
			if !ValidFaceIDs[jd.FaceB] {
				errs = append(errs, ValidationError{
					NodeID:   node.ID,
					Message:  fmt.Sprintf("invalid face_b %q", jd.FaceB),
					Severity: SeverityError,
				})
			}
		}
	}

	return errs
}

// validateJoinParts checks that join nodes reference primitive nodes for
// PartA and PartB, and that a join does not reference the same part for both
// (no self-joins).
func validateJoinParts(g *DesignGraph) []ValidationError {
	var errs []ValidationError

	for _, node := range g.Nodes {
		jd, ok := node.Data.(JoinData)
		if !ok {
			continue
		}

		// Self-join check.
		if jd.PartA == jd.PartB {
			errs = append(errs, ValidationError{
				NodeID:   node.ID,
				Message:  "join references the same part for both part_a and part_b (self-join)",
				Severity: SeverityError,
			})
		}

		// PartA must be a primitive.
		if partA, ok := g.Nodes[jd.PartA]; ok {
			if partA.Kind != NodePrimitive {
				errs = append(errs, ValidationError{
					NodeID:   node.ID,
					Message:  fmt.Sprintf("join part_a %s is %s, not primitive", jd.PartA.Short(), partA.Kind),
					Severity: SeverityError,
				})
			}
		}

		// PartB must be a primitive.
		if partB, ok := g.Nodes[jd.PartB]; ok {
			if partB.Kind != NodePrimitive {
				errs = append(errs, ValidationError{
					NodeID:   node.ID,
					Message:  fmt.Sprintf("join part_b %s is %s, not primitive", jd.PartB.Short(), partB.Kind),
					Severity: SeverityError,
				})
			}
		}
	}

	return errs
}
