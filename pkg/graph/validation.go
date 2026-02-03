// Package graph provides validation for design graphs.
package graph

import (
	"fmt"
)

// ValidationError represents a validation failure.
type ValidationError struct {
	Code    string
	Message string
	NodeID  NodeID
	PartID  PartID
}

func (e ValidationError) Error() string {
	context := ""
	if e.NodeID != "" {
		context = fmt.Sprintf(" (node: %s)", e.NodeID)
	}
	if e.PartID != "" {
		context = fmt.Sprintf(" (part: %s)", e.PartID)
	}
	return fmt.Sprintf("%s: %s%s", e.Code, e.Message, context)
}

// Validator validates design graphs and their components.
type Validator struct {
	graph *Graph
	pr    *PartRegistry
}

// NewValidator creates a new validator for a graph and part registry.
func NewValidator(graph *Graph, pr *PartRegistry) *Validator {
	return &Validator{
		graph: graph,
		pr:    pr,
	}
}

// Validate performs comprehensive validation of the entire design.
func (v *Validator) Validate() []ValidationError {
	var errors []ValidationError

	// Run all validation checks
	errors = append(errors, v.validateGraphStructure()...)
	errors = append(errors, v.validateNodes()...)
	errors = append(errors, v.validateParts()...)
	errors = append(errors, v.validateJoins()...)

	return errors
}

// validateGraphStructure validates the overall graph structure.
func (v *Validator) validateGraphStructure() []ValidationError {
	var errors []ValidationError

	// Check for cycles
	if cyclicNodes := v.detectCycles(); len(cyclicNodes) > 0 {
		errors = append(errors, ValidationError{
			Code:    "GRAPH_CYCLE",
			Message: fmt.Sprintf("Graph contains cycles in nodes: %v", cyclicNodes),
		})
	}

	// Check node consistency
	for nodeID, node := range v.graph.Nodes {
		// Verify all dependencies exist
		for _, depID := range node.Dependencies {
			if _, exists := v.graph.Nodes[depID]; !exists {
				errors = append(errors, ValidationError{
					Code:    "MISSING_DEPENDENCY",
					Message: fmt.Sprintf("Node references non-existent dependency: %s", depID),
					NodeID:  nodeID,
				})
			}
		}
	}

	// Check edge consistency
	for nodeID, deps := range v.graph.Edges {
		if _, exists := v.graph.Nodes[nodeID]; !exists {
			errors = append(errors, ValidationError{
				Code:    "ORPHAN_EDGE",
				Message: fmt.Sprintf("Edge references non-existent node: %s", nodeID),
			})
		}
		for _, depID := range deps {
			if _, exists := v.graph.Nodes[depID]; !exists {
				errors = append(errors, ValidationError{
					Code:    "INVALID_EDGE",
					Message: fmt.Sprintf("Edge references non-existent dependency: %s", depID),
					NodeID:  nodeID,
				})
			}
		}
	}

	return errors
}

// detectCycles detects cycles in the graph using DFS.
func (v *Validator) detectCycles() []NodeID {
	visited := make(map[NodeID]bool)
	recStack := make(map[NodeID]bool)
	var cycleNodes []NodeID

	var dfs func(NodeID) bool
	dfs = func(nodeID NodeID) bool {
		if !visited[nodeID] {
			visited[nodeID] = true
			recStack[nodeID] = true

			for _, neighbor := range v.graph.Edges[nodeID] {
				if !visited[neighbor] && dfs(neighbor) {
					cycleNodes = append(cycleNodes, neighbor)
					return true
				} else if recStack[neighbor] {
					cycleNodes = append(cycleNodes, neighbor)
					return true
				}
			}
		}
		recStack[nodeID] = false
		return false
	}

	for nodeID := range v.graph.Nodes {
		if dfs(nodeID) {
			cycleNodes = append(cycleNodes, nodeID)
		}
	}

	return cycleNodes
}

// validateNodes validates individual nodes.
func (v *Validator) validateNodes() []ValidationError {
	var errors []ValidationError

	for nodeID, node := range v.graph.Nodes {
		// Validate node type
		if node.Type < NodeTypePrimitive || node.Type > NodeTypeGroup {
			errors = append(errors, ValidationError{
				Code:    "INVALID_NODE_TYPE",
				Message: fmt.Sprintf("Invalid node type: %v", node.Type),
				NodeID:  nodeID,
			})
		}

		// Validate source expression (non-empty)
		if node.SourceExpr == "" {
			errors = append(errors, ValidationError{
				Code:    "EMPTY_SOURCE",
				Message: "Node must have a source expression",
				NodeID:  nodeID,
			})
		}

		// Validate outputs based on node type
		switch node.Type {
		case NodeTypePart:
			if len(node.Outputs) == 0 {
				errors = append(errors, ValidationError{
					Code:    "NO_PART_OUTPUT",
					Message: "Part node must have at least one output",
					NodeID:  nodeID,
				})
			}
			// Check that part nodes reference valid part IDs
			for _, output := range node.Outputs {
				if output.Type == OutputTypePart {
					if partID, ok := output.ID.(PartID); ok {
						if _, err := v.pr.GetPartByID(partID); err != nil {
							errors = append(errors, ValidationError{
								Code:    "INVALID_PART_REFERENCE",
								Message: fmt.Sprintf("Part node references non-existent part: %s", partID),
								NodeID:  nodeID,
							})
						}
					}
				}
			}
		case NodeTypeJoin:
			// Join nodes should reference valid join specs
			if spec, ok := node.Properties["spec"].(JoinSpec); ok {
				// Validate join spec
				errors = append(errors, v.validateJoinSpec(spec, nodeID)...)
			} else {
				errors = append(errors, ValidationError{
					Code:    "MISSING_JOIN_SPEC",
					Message: "Join node must have a join specification",
					NodeID:  nodeID,
				})
			}
		}
	}

	return errors
}

// validateParts validates part definitions.
func (v *Validator) validateParts() []ValidationError {
	var errors []ValidationError

	for partID, part := range v.pr.parts {
		// Validate part name (non-empty)
		if part.Name == "" {
			errors = append(errors, ValidationError{
				Code:    "EMPTY_PART_NAME",
				Message: "Part must have a name",
				PartID:  partID,
			})
		}

		// Validate grain direction
		if part.Metadata.GrainAxis < GrainX || part.Metadata.GrainAxis > GrainAny {
			errors = append(errors, ValidationError{
				Code:    "INVALID_GRAIN",
				Message: fmt.Sprintf("Invalid grain direction: %v", part.Metadata.GrainAxis),
				PartID:  partID,
			})
		}

		// Validate material specification
		if part.Metadata.Material.Type == "" {
			errors = append(errors, ValidationError{
				Code:    "EMPTY_MATERIAL",
				Message: "Part must have a material type",
				PartID:  partID,
			})
		}

		// Validate solid references
		for _, solidID := range part.Solids {
			// Check that solid IDs are non-empty
			if solidID == "" {
				errors = append(errors, ValidationError{
					Code:    "EMPTY_SOLID_REF",
					Message: "Part references empty solid ID",
					PartID:  partID,
				})
			}
			// Note: We can't validate that solids exist in geometry kernel here
		}

		// Validate node reference
		if part.NodeID == "" {
			errors = append(errors, ValidationError{
				Code:    "MISSING_NODE_REF",
				Message: "Part must reference a graph node",
				PartID:  partID,
			})
		} else if _, exists := v.graph.Nodes[part.NodeID]; !exists {
			errors = append(errors, ValidationError{
				Code:    "INVALID_NODE_REF",
				Message: fmt.Sprintf("Part references non-existent node: %s", part.NodeID),
				PartID:  partID,
			})
		}
	}

	return errors
}

// validateJoins validates join specifications.
func (v *Validator) validateJoins() []ValidationError {
	var errors []ValidationError

	// Check all join nodes
	for nodeID, node := range v.graph.Nodes {
		if node.Type == NodeTypeJoin {
			if spec, ok := node.Properties["spec"].(JoinSpec); ok {
				errors = append(errors, v.validateJoinSpec(spec, nodeID)...)
			}
		}
	}

	return errors
}

// validateJoinSpec validates a single join specification.
func (v *Validator) validateJoinSpec(spec JoinSpec, nodeID NodeID) []ValidationError {
	var errors []ValidationError

	// Validate join type
	if spec.Type < JoinTypeButt || spec.Type > JoinTypeMortiseTenon {
		errors = append(errors, ValidationError{
			Code:    "INVALID_JOIN_TYPE",
			Message: fmt.Sprintf("Invalid join type: %v", spec.Type),
			NodeID:  nodeID,
		})
	}

	// Validate part references
	partA, errA := v.pr.GetPartByID(spec.PartA)
	partB, errB := v.pr.GetPartByID(spec.PartB)

	if errA != nil {
		errors = append(errors, ValidationError{
			Code:    "INVALID_PART_A",
			Message: fmt.Sprintf("Join references non-existent part: %s", spec.PartA),
			NodeID:  nodeID,
		})
	}
	if errB != nil {
		errors = append(errors, ValidationError{
			Code:    "INVALID_PART_B",
			Message: fmt.Sprintf("Join references non-existent part: %s", spec.PartB),
			NodeID:  nodeID,
		})
	}

	// If parts exist, validate face indices
	if partA != nil && partB != nil {
		// Check that parts have solids for face references
		if len(partA.Solids) == 0 {
			errors = append(errors, ValidationError{
				Code:    "NO_SOLIDS_PART_A",
				Message: fmt.Sprintf("Part %s has no solids for face reference", spec.PartA),
				NodeID:  nodeID,
			})
		}
		if len(partB.Solids) == 0 {
			errors = append(errors, ValidationError{
				Code:    "NO_SOLIDS_PART_B",
				Message: fmt.Sprintf("Part %s has no solids for face reference", spec.PartB),
				NodeID:  nodeID,
			})
		}

		// Note: Face index validation would require geometry kernel access
		// to check if face indices are valid for the referenced solids
	}

	// Validate clearance (positive, reasonable value)
	if spec.Clearance < 0 {
		errors = append(errors, ValidationError{
			Code:    "NEGATIVE_CLEARANCE",
			Message: "Clearance must be non-negative",
			NodeID:  nodeID,
		})
	}
	if spec.Clearance > 10.0 { // Arbitrary reasonable maximum
		errors = append(errors, ValidationError{
			Code:    "EXCESSIVE_CLEARANCE",
			Message: fmt.Sprintf("Clearance %vmm is excessively large", spec.Clearance),
			NodeID:  nodeID,
		})
	}

	// Validate join-specific parameters
	switch spec.Type {
	case JoinTypeButt:
		errors = append(errors, v.validateButtJoinParams(spec, nodeID)...)
	case JoinTypeHole:
		errors = append(errors, v.validateHoleJoinParams(spec, nodeID)...)
	case JoinTypeFastener:
		errors = append(errors, v.validateFastenerJoinParams(spec, nodeID)...)
	}

	return errors
}

// validateButtJoinParams validates butt joint parameters.
func (v *Validator) validateButtJoinParams(spec JoinSpec, nodeID NodeID) []ValidationError {
	var errors []ValidationError

	// Check for required parameters
	if fasteners, ok := spec.Parameters["fasteners"].([]FastenerSpec); ok {
		for i, fastener := range fasteners {
			errors = append(errors, v.validateFastenerSpec(fastener, nodeID, i)...)
		}
	}

	return errors
}

// validateHoleJoinParams validates hole operation parameters.
func (v *Validator) validateHoleJoinParams(spec JoinSpec, nodeID NodeID) []ValidationError {
	var errors []ValidationError

	// Check hole specifications
	if holes, ok := spec.Parameters["holes"].([]HoleSpec); ok {
		for i, hole := range holes {
			errors = append(errors, v.validateHoleSpec(hole, nodeID, i)...)
		}
	} else {
		errors = append(errors, ValidationError{
			Code:    "MISSING_HOLES",
			Message: "Hole join must specify holes",
			NodeID:  nodeID,
		})
	}

	return errors
}

// validateFastenerJoinParams validates fastener operation parameters.
func (v *Validator) validateFastenerJoinParams(spec JoinSpec, nodeID NodeID) []ValidationError {
	var errors []ValidationError

	// Check fastener specifications
	if fasteners, ok := spec.Parameters["fasteners"].([]FastenerSpec); ok {
		for i, fastener := range fasteners {
			errors = append(errors, v.validateFastenerSpec(fastener, nodeID, i)...)
		}
	} else {
		errors = append(errors, ValidationError{
			Code:    "MISSING_FASTENERS",
			Message: "Fastener join must specify fasteners",
			NodeID:  nodeID,
		})
	}

	return errors
}

// validateHoleSpec validates a hole specification.
func (v *Validator) validateHoleSpec(hole HoleSpec, nodeID NodeID, index int) []ValidationError {
	var errors []ValidationError

	// Validate diameter (positive)
	if hole.Diameter <= 0 {
		errors = append(errors, ValidationError{
			Code:    "INVALID_HOLE_DIAMETER",
			Message: fmt.Sprintf("Hole %d diameter must be positive", index),
			NodeID:  nodeID,
		})
	}

	// Validate depth (positive)
	if hole.Depth <= 0 {
		errors = append(errors, ValidationError{
			Code:    "INVALID_HOLE_DEPTH",
			Message: fmt.Sprintf("Hole %d depth must be positive", index),
			NodeID:  nodeID,
		})
	}

	// Validate direction (non-zero vector)
	if hole.Direction.X == 0 && hole.Direction.Y == 0 && hole.Direction.Z == 0 {
		errors = append(errors, ValidationError{
			Code:    "ZERO_HOLE_DIRECTION",
			Message: fmt.Sprintf("Hole %d direction cannot be zero vector", index),
			NodeID:  nodeID,
		})
	}

	return errors
}

// validateFastenerSpec validates a fastener specification.
func (v *Validator) validateFastenerSpec(fastener FastenerSpec, nodeID NodeID, index int) []ValidationError {
	var errors []ValidationError

	// Validate fastener type
	if fastener.Type < FastenerTypeScrew || fastener.Type > FastenerTypeBolt {
		errors = append(errors, ValidationError{
			Code:    "INVALID_FASTENER_TYPE",
			Message: fmt.Sprintf("Fastener %d has invalid type: %v", index, fastener.Type),
			NodeID:  nodeID,
		})
	}

	// Validate length (positive)
	if fastener.Length <= 0 {
		errors = append(errors, ValidationError{
			Code:    "INVALID_FASTENER_LENGTH",
			Message: fmt.Sprintf("Fastener %d length must be positive", index),
			NodeID:  nodeID,
		})
	}

	// Validate size (non-empty)
	if fastener.Size == "" {
		errors = append(errors, ValidationError{
			Code:    "EMPTY_FASTENER_SIZE",
			Message: fmt.Sprintf("Fastener %d must have a size specification", index),
			NodeID:  nodeID,
		})
	}

	// Validate direction (non-zero vector)
	if fastener.Direction.X == 0 && fastener.Direction.Y == 0 && fastener.Direction.Z == 0 {
		errors = append(errors, ValidationError{
			Code:    "ZERO_FASTENER_DIRECTION",
			Message: fmt.Sprintf("Fastener %d direction cannot be zero vector", index),
			NodeID:  nodeID,
		})
	}

	return errors
}

// ValidateDesign validates a complete Design object.
func ValidateDesign(design *Design) []ValidationError {
	pr := &PartRegistry{parts: design.Parts}
	validator := NewValidator(design.Graph, pr)
	return validator.Validate()
}