// Package graph provides the design graph API for Lignin.
package graph

import (
	"fmt"
	"time"
)

// PartRegistry manages part definitions and references.
type PartRegistry struct {
	parts map[PartID]*Part
	names map[string]PartID // Name -> PartID mapping
}

// NewPartRegistry creates a new empty part registry.
func NewPartRegistry() *PartRegistry {
	return &PartRegistry{
		parts: make(map[PartID]*Part),
		names: make(map[string]PartID),
	}
}

// DefinePart creates a new part definition.
func (pr *PartRegistry) DefinePart(name string, solids []SolidID, metadata PartMetadata, nodeID NodeID) (PartID, error) {
	if _, exists := pr.names[name]; exists {
		return "", fmt.Errorf("part name '%s' already defined", name)
	}

	partID := PartID(name)
	part := &Part{
		ID:         partID,
		Name:       name,
		Solids:     solids,
		Metadata:   metadata,
		NodeID:     nodeID,
	}

	pr.parts[partID] = part
	pr.names[name] = partID
	return partID, nil
}

// GetPart returns a part by name.
func (pr *PartRegistry) GetPart(name string) (*Part, error) {
	partID, exists := pr.names[name]
	if !exists {
		return nil, fmt.Errorf("part '%s' not found", name)
	}
	return pr.parts[partID], nil
}

// GetPartByID returns a part by PartID.
func (pr *PartRegistry) GetPartByID(id PartID) (*Part, error) {
	part, exists := pr.parts[id]
	if !exists {
		return nil, fmt.Errorf("part with ID '%s' not found", id)
	}
	return part, nil
}

// ListParts returns all defined part IDs.
func (pr *PartRegistry) ListParts() []PartID {
	parts := make([]PartID, 0, len(pr.parts))
	for id := range pr.parts {
		parts = append(parts, id)
	}
	return parts
}

// UpdatePartMetadata updates a part's metadata, creating a new version.
func (pr *PartRegistry) UpdatePartMetadata(partID PartID, metadata PartMetadata) (PartID, error) {
	part, exists := pr.parts[partID]
	if !exists {
		return "", fmt.Errorf("part '%s' not found", partID)
	}

	// Create new part with updated metadata
	newPart := *part
	newPart.Metadata = metadata
	newPartID := PartID(fmt.Sprintf("%s_v%d", part.Name, len(pr.parts)))

	pr.parts[newPartID] = &newPart
	pr.names[newPart.Name] = newPartID // Overwrite name mapping

	return newPartID, nil
}

// DeletePart removes a part definition.
func (pr *PartRegistry) DeletePart(partID PartID) error {
	part, exists := pr.parts[partID]
	if !exists {
		return fmt.Errorf("part '%s' not found", partID)
	}

	delete(pr.parts, partID)
	delete(pr.names, part.Name)
	return nil
}

// GraphBuilder provides a fluent API for building design graphs.
type GraphBuilder struct {
	graph *Graph
	pr    *PartRegistry
}

// NewGraphBuilder creates a new graph builder.
func NewGraphBuilder() *GraphBuilder {
	return &GraphBuilder{
		graph: &Graph{
			Nodes: make(map[NodeID]*Node),
			Edges: make(map[NodeID][]NodeID),
			Roots: []NodeID{},
		},
		pr: NewPartRegistry(),
	}
}

// AddNode adds a new node to the graph.
func (gb *GraphBuilder) AddNode(node *Node) error {
	if _, exists := gb.graph.Nodes[node.ID]; exists {
		return fmt.Errorf("node '%s' already exists", node.ID)
	}

	gb.graph.Nodes[node.ID] = node
	gb.graph.Edges[node.ID] = node.Dependencies

	// If node has no dependencies, add to roots
	if len(node.Dependencies) == 0 {
		gb.graph.Roots = append(gb.graph.Roots, node.ID)
	}

	return nil
}

// CreatePrimitiveNode creates a primitive geometry node.
func (gb *GraphBuilder) CreatePrimitiveNode(primitiveType string, dimensions Vector3) NodeID {
	nodeID := generateNodeID("primitive", primitiveType, dimensions)
	node := &Node{
		ID:         nodeID,
		Type:       NodeTypePrimitive,
		SourceExpr: fmt.Sprintf("(primitive :%s %v)", primitiveType, dimensions),
		Properties: map[string]interface{}{
			"type":       primitiveType,
			"dimensions": dimensions,
		},
		Metadata: NodeMetadata{
			CreatedAt: time.Now(),
			Tags:      []string{"primitive", primitiveType},
		},
	}

	gb.AddNode(node)
	return nodeID
}

// CreateTransformNode creates a transformation node.
func (gb *GraphBuilder) CreateTransformNode(transformType string, params map[string]interface{}, dependencies []NodeID) NodeID {
	nodeID := generateNodeID("transform", transformType, params)
	node := &Node{
		ID:           nodeID,
		Type:         NodeTypeTransform,
		Dependencies: dependencies,
		SourceExpr:   fmt.Sprintf("(%s ...)", transformType),
		Properties:   params,
		Metadata: NodeMetadata{
			CreatedAt: time.Now(),
			Tags:      []string{"transform", transformType},
		},
	}

	gb.AddNode(node)
	return nodeID
}

// CreatePartNode creates a part definition node.
func (gb *GraphBuilder) CreatePartNode(name string, solidNodes []NodeID, metadata PartMetadata) (NodeID, PartID, error) {
	// Convert solid node IDs to SolidIDs
	solids := make([]SolidID, len(solidNodes))
	for i, nodeID := range solidNodes {
		solids[i] = SolidID(nodeID)
	}

	nodeID := generateNodeID("part", name, solids)
	node := &Node{
		ID:         nodeID,
		Type:       NodeTypePart,
		Dependencies: solidNodes,
		SourceExpr: fmt.Sprintf("(define-part \"%s\" ...)", name),
		Properties: map[string]interface{}{
			"name":     name,
			"metadata": metadata,
		},
		Metadata: NodeMetadata{
			CreatedAt: time.Now(),
			Tags:      []string{"part", name},
		},
		Outputs: []OutputRef{
			{
				Type: OutputTypePart,
				Name: name,
			},
		},
	}

	if err := gb.AddNode(node); err != nil {
		return "", "", err
	}

	partID, err := gb.pr.DefinePart(name, solids, metadata, nodeID)
	return nodeID, partID, err
}

// CreateJoinNode creates a join operation node.
func (gb *GraphBuilder) CreateJoinNode(joinType JoinType, spec JoinSpec, dependencies []NodeID) NodeID {
	joinTypeStr := joinTypeToString(joinType)
	nodeID := generateNodeID("join", joinTypeStr, spec)
	node := &Node{
		ID:           nodeID,
		Type:         NodeTypeJoin,
		Dependencies: dependencies,
		SourceExpr:   fmt.Sprintf("(%s-join ...)", joinTypeStr),
		Properties: map[string]interface{}{
			"type": joinType,
			"spec": spec,
		},
		Metadata: NodeMetadata{
			CreatedAt: time.Now(),
			Tags:      []string{"join", joinTypeStr},
		},
	}

	gb.AddNode(node)
	return nodeID
}

// joinTypeToString converts a JoinType to its string representation.
func joinTypeToString(joinType JoinType) string {
	switch joinType {
	case JoinTypeButt:
		return "butt"
	case JoinTypeHole:
		return "hole"
	case JoinTypeFastener:
		return "fastener"
	case JoinTypeDovetail:
		return "dovetail"
	case JoinTypeMortiseTenon:
		return "mortise-tenon"
	default:
		return "unknown"
	}
}

// Build returns the completed graph and part registry.
func (gb *GraphBuilder) Build() (*Graph, *PartRegistry) {
	return gb.graph, gb.pr
}

// generateNodeID creates a content-addressed node ID.
// In a real implementation, this would hash the content.
func generateNodeID(prefix string, content ...interface{}) NodeID {
	// Simplified implementation - would use proper content hashing
	return NodeID(fmt.Sprintf("%s_%v", prefix, content))
}

// DesignBuilder provides a high-level API for building complete designs.
type DesignBuilder struct {
	gb *GraphBuilder
}

// NewDesignBuilder creates a new design builder.
func NewDesignBuilder() *DesignBuilder {
	return &DesignBuilder{
		gb: NewGraphBuilder(),
	}
}

// AddPrimitive adds a primitive shape to the design.
func (db *DesignBuilder) AddPrimitive(name, shape string, dimensions Vector3) NodeID {
	return db.gb.CreatePrimitiveNode(shape, dimensions)
}

// AddPart adds a named part to the design.
func (db *DesignBuilder) AddPart(name string, solidNodes []NodeID, grain GrainDirection, material string) (NodeID, PartID, error) {
	metadata := PartMetadata{
		Name:      name,
		GrainAxis: grain,
		Material: MaterialSpec{
			Type: material,
		},
	}
	return db.gb.CreatePartNode(name, solidNodes, metadata)
}

// AddJoin adds a join operation between two parts.
func (db *DesignBuilder) AddJoin(joinType JoinType, partA, partB PartID, faceA, faceB int, clearance float64) (NodeID, error) {
	partAObj, err := db.gb.pr.GetPartByID(partA)
	if err != nil {
		return "", err
	}
	partBObj, err := db.gb.pr.GetPartByID(partB)
	if err != nil {
		return "", err
	}

	// Get solid IDs from parts (simplified - assumes one solid per part)
	var solidA, solidB SolidID
	if len(partAObj.Solids) > 0 {
		solidA = partAObj.Solids[0]
	}
	if len(partBObj.Solids) > 0 {
		solidB = partBObj.Solids[0]
	}

	spec := JoinSpec{
		Type: joinType,
		PartA: partA,
		FaceA: FaceID{Solid: solidA, Index: faceA},
		PartB: partB,
		FaceB: FaceID{Solid: solidB, Index: faceB},
		Clearance: clearance,
		Parameters: make(map[string]interface{}),
	}

	// Dependencies are the part nodes
	dependencies := []NodeID{NodeID(partAObj.NodeID), NodeID(partBObj.NodeID)}
	nodeID := db.gb.CreateJoinNode(joinType, spec, dependencies)
	return nodeID, nil
}

// BuildDesign creates a complete Design object.
func (db *DesignBuilder) BuildDesign(version string) *Design {
	graph, pr := db.gb.Build()
	return &Design{
		Graph:   graph,
		Parts:   pr.parts,
		Version: version,
	}
}