// Package graph defines the core design graph data structures for Lignin.
package graph

import (
	"time"
)

// NodeID is a content-addressed identifier for graph nodes.
type NodeID string

// PartID is a named reference to a part in the design.
type PartID string

// SolidID references a geometric solid in the geometry kernel.
type SolidID string

// FaceID references a specific face of a solid.
type FaceID struct {
	Solid SolidID
	Index int // Face index in B-rep representation
}

// NodeType represents the type of operation a node performs.
type NodeType int

const (
	NodeTypePrimitive NodeType = iota
	NodeTypeTransform
	NodeTypeJoin
	NodeTypePart
	NodeTypeGroup
)

// Node represents a single operation in the design graph.
type Node struct {
	ID           NodeID
	Type         NodeType
	SourceExpr   string                 // Lisp expression that created this node
	Dependencies []NodeID               // Input nodes
	Properties   map[string]interface{} // Type-specific properties
	Metadata     NodeMetadata
	Outputs      []OutputRef // References to emitted solids/parts
}

// NodeMetadata contains metadata about a node.
type NodeMetadata struct {
	CreatedAt   time.Time
	EvaluatedAt time.Time
	Tags        []string // Semantic tags for search/filtering
}

// OutputType indicates what type of output a node produces.
type OutputType int

const (
	OutputTypeSolid OutputType = iota
	OutputTypePart
	OutputTypeFace
)

// OutputRef describes what a node produces.
type OutputRef struct {
	Type OutputType
	ID   interface{} // PartID, SolidID, or FaceID
	Name string      // Optional human-readable name
}

// Graph represents the complete design graph.
type Graph struct {
	Nodes map[NodeID]*Node
	Edges map[NodeID][]NodeID // Adjacency list (dependencies)
	Roots []NodeID            // Nodes with no dependencies
}

// GrainDirection represents the dominant grain direction in a wood part.
type GrainDirection int

const (
	GrainX GrainDirection = iota // Grain runs along X axis
	GrainY                       // Grain runs along Y axis
	GrainZ                       // Grain runs along Z axis
	GrainAny                     // No specific grain direction
)

// PartMetadata contains metadata about a part.
type PartMetadata struct {
	Name      string
	GrainAxis GrainDirection
	Material  MaterialSpec
	Tags      []string
}

// MaterialSpec describes the material properties of a part.
type MaterialSpec struct {
	Type       string  // e.g., "oak", "maple", "plywood"
	Thickness  float64 // in mm
	Density    float64 // kg/m³
	Color      string  // Visual representation
	Properties map[string]interface{}
}

// JoinType represents the type of join operation.
type JoinType int

const (
	JoinTypeButt JoinType = iota
	JoinTypeHole
	JoinTypeFastener
	JoinTypeDovetail
	JoinTypeMortiseTenon
)

// JoinSpec specifies a join operation between two parts.
type JoinSpec struct {
	Type       JoinType
	PartA      PartID
	FaceA      FaceID
	PartB      PartID
	FaceB      FaceID
	Clearance  float64 // Gap tolerance in mm
	Parameters map[string]interface{}
}

// HoleSpec specifies a hole drilling operation.
type HoleSpec struct {
	Position  Vector3 // Relative to part origin
	Direction Vector3 // Normal vector for hole axis
	Diameter  float64 // Hole diameter in mm
	Depth     float64 // Hole depth in mm
	Threaded  bool    // Whether to create threads
	Counterbore *CounterboreSpec
}

// CounterboreSpec specifies counterbore parameters for a hole.
type CounterboreSpec struct {
	Diameter float64
	Depth    float64
}

// FastenerType represents the type of fastener.
type FastenerType int

const (
	FastenerTypeScrew FastenerType = iota
	FastenerTypeDowels
	FastenerTypeNail
	FastenerTypeBolt
)

// HeadType represents the type of fastener head.
type HeadType int

const (
	HeadTypeFlat HeadType = iota
	HeadTypePan
	HeadTypeRound
	HeadTypeCountersunk
)

// FastenerSpec specifies a fastener embedding operation.
type FastenerSpec struct {
	Type      FastenerType
	Position  Vector3
	Direction Vector3
	Size      string // e.g., "#8", "3/8\" dowel"
	Length    float64 // Fastener length in mm
	HeadType  HeadType
}

// Vector3 represents a 3D vector.
type Vector3 struct {
	X, Y, Z float64
}

// Part represents a named component in the design.
type Part struct {
	ID         PartID
	Name       string
	SourceExpr string
	Metadata   PartMetadata
	Solids     []SolidID // Geometry emitted by this part
	NodeID     NodeID    // Graph node that created this part
}

// Design represents a complete woodworking design.
type Design struct {
	Graph   *Graph
	Parts   map[PartID]*Part
	Version string // Design version identifier
}