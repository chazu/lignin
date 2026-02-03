# Lignin Architecture Design

## 1. Design Graph Architecture

### Core Principles
- Directed Acyclic Graph (DAG) structure
- Immutable nodes with stable identities
- Node-to-source expression mapping
- Pure functional transformations

### Node Types
1. **Primitive Nodes**: Basic geometric shapes (cuboid, cylinder, etc.)
2. **Transform Nodes**: Translation, rotation, scaling operations
3. **Join Nodes**: Woodworking-specific joinery operations
4. **Part Nodes**: Named components with metadata
5. **Group Nodes**: Logical groupings of other nodes

### Graph Structure
```go
type Graph struct {
    Nodes map[NodeID]*Node
    Edges map[NodeID][]NodeID  // Adjacency list
    Roots []NodeID             // Nodes with no dependencies
}
```

### Node Identity
- Content-addressed using hash of node content + dependencies
- Stable across evaluations when source unchanged
- Enables incremental re-evaluation

## 2. Part System

### Part Definition
```go
type Part struct {
    ID          PartID
    Name        string          // Human-readable identifier
    SourceExpr  string          // Lisp expression that created this part
    Metadata    PartMetadata
    Solids      []SolidID       // Geometry emitted by this part
    GrainAxis   GrainDirection  // Dominant grain direction (X, Y, or Z)
    Material    MaterialType    // Wood species, thickness, etc.
}
```

### Part Reference Mechanism
- Parts are named entities that can be referenced by name
- Names must be unique within a design
- References resolved at evaluation time
- Supports forward references (parts defined later)

### Grain Direction Representation
```go
type GrainDirection int

const (
    GrainX GrainDirection = iota  // Grain runs along X axis
    GrainY                        // Grain runs along Y axis
    GrainZ                        // Grain runs along Z axis
    GrainAny                      // No specific grain direction
)
```

## 3. Join Semantics

### Join Types
1. **Butt Joints**: Face-to-face connections
2. **Hole Operations**: Drilling for fasteners or joinery
3. **Fastener Embedding**: Screws, dowels, etc.
4. **Complex Joinery**: Dovetails, mortise & tenon, etc.

### Join Specification Format
```go
type JoinSpec struct {
    Type        JoinType
    PartA       PartID          // First part to join
    FaceA       FaceID          // Which face of PartA
    PartB       PartID          // Second part to join
    FaceB       FaceID          // Which face of PartB
    Clearance   float64         // Gap tolerance (mm)
    Parameters  map[string]any  // Type-specific parameters
}
```

### Clearance/Tolerance Model
- Global default tolerance configurable
- Per-join override possible
- Tolerance affects boolean subtraction size
- Material-aware tolerances (wood vs. metal)

### Hole Drilling Operations
```go
type HoleSpec struct {
    Position    Vector3     // Relative to part origin
    Direction   Vector3     // Normal vector for hole axis
    Diameter    float64     // Hole diameter (mm)
    Depth       float64     // Hole depth (mm)
    Threaded    bool        // Whether to create threads
    Counterbore *CounterboreSpec  // Optional counterbore
}
```

### Fastener Embedding
```go
type FastenerSpec struct {
    Type        FastenerType  // screw, dowel, nail, etc.
    Position    Vector3
    Direction   Vector3
    Size        string       // e.g., "#8", "3/8\" dowel"
    Length      float64      // Fastener length (mm)
    HeadType    HeadType     // flat, pan, round, etc.
}
```

## 4. Design Graph Schema (Go Structs)

```go
package lignin

// NodeID is a content-addressed identifier
type NodeID string

// Node represents a single operation in the design graph
type Node struct {
    ID          NodeID
    Type        NodeType
    SourceExpr  string          // Lisp expression that created this node
    Dependencies []NodeID       // Input nodes
    Properties  map[string]any  // Type-specific properties
    Metadata    NodeMetadata
    Outputs     []OutputRef     // References to emitted solids/parts
}

type NodeType int

const (
    NodeTypePrimitive NodeType = iota
    NodeTypeTransform
    NodeTypeJoin
    NodeTypePart
    NodeTypeGroup
)

type NodeMetadata struct {
    CreatedAt   time.Time
    EvaluatedAt time.Time
    Tags        []string  // Semantic tags for search/filtering
}

// PartID is a named reference to a part
type PartID string

// SolidID references a geometric solid in the geometry kernel
type SolidID string

// FaceID references a specific face of a solid
type FaceID struct {
    Solid SolidID
    Index int  // Face index in B-rep representation
}

// OutputRef describes what a node produces
type OutputRef struct {
    Type    OutputType
    ID      interface{}  // PartID, SolidID, or FaceID
    Name    string       // Optional human-readable name
}

type OutputType int

const (
    OutputTypeSolid OutputType = iota
    OutputTypePart
    OutputTypeFace
)
```

## 5. Part Naming and Reference API

### API Functions
```go
// Define a new part
func DefinePart(name string, solids []SolidID, metadata PartMetadata) PartID

// Reference an existing part by name
func GetPart(name string) (PartID, error)

// List all defined parts
func ListParts() []PartID

// Get part metadata
func GetPartMetadata(part PartID) PartMetadata

// Update part metadata (creates new version)
func UpdatePartMetadata(part PartID, metadata PartMetadata) PartID
```

### Lisp API Examples
```lisp
;; Define a part
(define-part "leg-front-left"
  :solids [cuboid-1]
  :grain :z
  :material {:type "oak" :thickness 45})

;; Reference a part in a join
(butt-join :part-a "leg-front-left"
           :face-a 3
           :part-b "apron-front"
           :face-b 1
           :clearance 0.5)
```

## 6. Join Operation Specification

### Join Operation Types
```go
type JoinType int

const (
    JoinTypeButt JoinType = iota
    JoinTypeHole
    JoinTypeFastener
    JoinTypeDovetail
    JoinTypeMortiseTenon
)
```

### Butt Joint Parameters
```go
type ButtJointParams struct {
    Clearance       float64
    Fasteners       []FastenerSpec  // Optional fasteners
    GlueSurface     bool            // Whether to create glue surface
    Reinforcement   ReinforcementType  // Optional reinforcement
}
```

### Hole Operation Parameters
```go
type HoleParams struct {
    Pattern         HolePattern     // Single, grid, circular array
    Spacing         float64         // For pattern holes
    Counterbore     *CounterboreSpec
    ThreadPitch     float64         // For threaded holes
}
```

## 7. Example: Design Graph for Simple Box

### Box Components
1. **4x Legs**: Vertical supports
2. **4x Aprons**: Horizontal frame members
3. **1x Top**: Box top surface
4. **8x Butt joints**: Leg-to-apron connections
5. **32x Screw holes**: Fastener locations

### Graph Structure
```
Primitive Nodes:
- leg-primitive (cuboid 50x50x750mm)
- apron-primitive (cuboid 100x50x600mm)
- top-primitive (cuboid 600x600x25mm)

Transform Nodes:
- leg-position-1 (translate leg to corner)
- leg-position-2, leg-position-3, leg-position-4
- apron-position-1..4
- top-position

Part Nodes:
- leg-1..4 (references transformed leg solids)
- apron-1..4 (references transformed apron solids)
- top (references transformed top solid)

Join Nodes:
- joint-1..8 (butt joints between legs and aprons)
- hole-1..32 (screw holes for joints)
```

### Lisp Representation
```lisp
(defprimitive leg :cuboid [50 50 750])
(defprimitive apron :cuboid [100 50 600])
(defprimitive top :cuboid [600 600 25])

(defpart leg-1 (translate leg [0 0 0]))
(defpart leg-2 (translate leg [550 0 0]))
(defpart leg-3 (translate leg [0 550 0]))
(defpart leg-4 (translate leg [550 550 0]))

(defpart apron-1 (translate apron [50 0 50]))
(defpart apron-2 (translate apron [50 0 550]))
(defpart apron-3 (rotate (translate apron [0 50 50]) [0 0 90]))
(defpart apron-4 (rotate (translate apron [550 50 50]) [0 0 90]))

(defpart top-surface (translate top [0 0 775]))

;; Butt joints
(butt-join :part-a "leg-1" :face-a 0
           :part-b "apron-1" :face-b 2
           :clearance 0.2)

;; Screw holes
(screw-hole :part "joint-1"
            :position [25 25 0]
            :diameter 4.5
            :depth 40)
```

## 8. Validation Rules

### Graph Validation
1. **No cycles**: Graph must be acyclic
2. **Node consistency**: All referenced nodes must exist
3. **Type safety**: Node inputs must match expected types
4. **Output consistency**: Nodes must emit valid outputs

### Part Validation
1. **Unique names**: No duplicate part names
2. **Valid references**: All referenced solids must exist
3. **Grain consistency**: Grain direction must be X, Y, Z, or Any
4. **Material properties**: Material specs must be valid

### Join Validation
1. **Face existence**: Referenced faces must exist
2. **Compatibility**: Joined faces must be compatible (similar size/orientation)
3. **Clearance bounds**: Clearance must be within reasonable limits
4. **Fastener feasibility**: Fastener size must fit within part dimensions

### Geometric Validation
1. **Solid integrity**: All solids must be manifold (watertight)
2. **Face normals**: All faces must have consistent orientation
3. **Non-zero volume**: Solids must have positive volume
4. **Self-intersection**: No self-intersecting geometry

## 9. Progressive Refinement Support

### Abstraction Levels
1. **Abstract parts**: Dimensions only, no specific geometry
2. **Parameterized parts**: Dimensions with parameters
3. **Concrete parts**: Specific geometry with material
4. **Stock-bound parts**: Mapped to physical stock

### Refinement Process
```go
type RefinementStage int

const (
    StageAbstract RefinementStage = iota
    StageParameterized
    StageConcrete
    StageStockMapped
)

type PartRefinement struct {
    CurrentStage  RefinementStage
    Constraints   []Constraint      // Design constraints
    Parameters    map[string]any    // Parameter values
    Alternatives  []Alternative     // Design alternatives
}
```

## 10. Key Constraints & Design Decisions

### Immutability
- All graph operations produce new graphs
- Node identities stable across evaluations
- Enables caching and incremental updates

### Content Addressing
- Node ID = hash(node content + dependency IDs)
- Enables deduplication
- Supports distributed caching

### Separation of Concerns
- Design graph handles semantic relationships
- Geometry kernel handles geometric operations
- Renderer handles visualization only

### Woodworking Semantics
- Join operations understand woodworking constraints
- Grain direction affects join feasibility
- Material properties influence tolerances

### MVP Limitations
- No module/import system
- Stock mapping optional/advisory
- Single-user, single-session focus