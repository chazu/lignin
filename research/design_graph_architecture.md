# Lignin Design Graph Architecture

**Status:** Research / Proposal
**Date:** 2026-02-02

---

## Table of Contents

1. [Design Graph Schema](#1-design-graph-schema)
2. [Part Naming and Reference API](#2-part-naming-and-reference-api)
3. [Join Operation Specification Format](#3-join-operation-specification-format)
4. [Worked Example: Simple Box](#4-worked-example-simple-box)
5. [Validation Rules](#5-validation-rules)
6. [Research Notes](#6-research-notes)

---

## 1. Design Graph Schema

### 1.1 Overview

The Lignin design graph is an immutable directed acyclic graph (DAG) produced by
evaluating Lisp source code. Each evaluation produces a complete, self-contained
graph. The graph represents the design at the semantic level: named parts with
material intent, spatial transforms, and joinery operations that encode
woodworking meaning beyond mere boolean subtraction.

The graph is **not** a CSG tree in the OpenSCAD sense. OpenSCAD's internal
representation is a tree of union/difference/intersection operations over
geometric primitives. Lignin's graph operates one level higher: nodes represent
**parts** (semantic woodworking entities with grain, material, and name) and
**joins** (woodworking connections with intent), not raw geometric operations.
The geometry kernel consumes the design graph and produces solids; the design
graph itself is geometry-agnostic.

### 1.2 Node Identity: Content-Addressed with Stable Source Keys

**Decision: Hybrid approach -- source-expression keys for identity, content hashing for change detection.**

#### Trade-off Analysis

| Approach | Strengths | Weaknesses |
|---|---|---|
| **UUID (random)** | Simple, stable across edits, no computation | No dedup, no integrity verification, identity divorced from content |
| **Content-addressed (Merkle/CID)** | Automatic dedup, self-verifying, change detection is free | Any edit changes the hash, breaking identity; must build bottom-up |
| **Source-expression key** | Stable when code structure is preserved; human-readable; maps directly to Lisp forms | Requires source location tracking; can break on refactors |

The hybrid approach works as follows:

- **NodeID** is a deterministic hash of the source expression path (the chain of
  Lisp forms from root to the expression that produced this node). This gives
  stability: renaming a variable does not change the identity of the part it
  defines, but reorganizing the code structure does. This is acceptable because
  the graph is re-derived on every evaluation anyway.
- **ContentHash** is a separate field computed from the node's semantic content
  (type, parameters, children references). This enables efficient diff between
  two graph versions: same NodeID + different ContentHash = "this part changed."
- **UserName** is an optional human-assigned name (e.g., `"left-side"`) that
  provides stable reference across code refactors.

This mirrors how Git uses content-addressed storage internally but provides
stable ref names (branches, tags) for human use.

### 1.3 Proposed Go Struct Definitions

```go
package graph

import (
    "crypto/sha256"
    "fmt"
)

// NodeID is a deterministic identifier derived from source expression path.
// It is stable across evaluations that preserve code structure.
type NodeID [32]byte

// ContentHash is a hash of the node's semantic content, used for change detection.
type ContentHash [32]byte

// SourceRef points back to the Lisp expression that produced this node.
type SourceRef struct {
    File   string // source file (empty string for single-file MVP)
    Line   int    // line number (1-based)
    Col    int    // column number (1-based)
    FormID string // unique identifier for the S-expression
}

// Vec3 is a 3D vector used for dimensions, positions, and directions.
type Vec3 struct {
    X, Y, Z float64
}

// Axis represents a principal axis, used for grain direction and face selection.
type Axis int

const (
    AxisX Axis = iota // typically width
    AxisY             // typically height
    AxisZ             // typically depth/length
)

// AxisDirection combines an axis with a sign to identify a face.
type AxisDirection struct {
    Axis     Axis
    Positive bool // true = +X/+Y/+Z face, false = -X/-Y/-Z face
}

// FaceID identifies one of the six faces of a rectangular part.
// For non-rectangular parts, this extends to named semantic faces.
type FaceID string

const (
    FaceTop    FaceID = "top"    // +Y
    FaceBottom FaceID = "bottom" // -Y
    FaceLeft   FaceID = "left"   // -X
    FaceRight  FaceID = "right"  // +X
    FaceFront  FaceID = "front"  // -Z
    FaceBack   FaceID = "back"   // +Z
)

// --------------------------------------------------------------------------
// Core Node Types
// --------------------------------------------------------------------------

// NodeKind enumerates the types of nodes in the design graph.
type NodeKind int

const (
    NodePrimitive  NodeKind = iota // a geometric primitive (board, panel, dowel)
    NodeTransform                  // a spatial transformation
    NodeJoin                       // a joinery operation
    NodeGroup                      // a logical grouping (assembly, subassembly)
    NodeDrill                      // a hole/boring operation
    NodeFastener                   // a fastener placement (screw, dowel pin)
)

// Node is the fundamental element of the design graph.
type Node struct {
    ID          NodeID      `json:"id"`
    Kind        NodeKind    `json:"kind"`
    Name        string      `json:"name,omitempty"`        // user-assigned name
    Source      SourceRef   `json:"source"`                // back-reference to Lisp code
    ContentHash ContentHash `json:"content_hash"`          // for change detection
    Children    []NodeID    `json:"children,omitempty"`    // ordered child references
    Data        NodeData    `json:"data"`                  // kind-specific payload
}

// NodeData is a tagged union for kind-specific node data.
// In Go, we model this as an interface with concrete types.
type NodeData interface {
    nodeData() // marker method
}

// --------------------------------------------------------------------------
// Primitive Node Data
// --------------------------------------------------------------------------

// PrimitiveKind distinguishes between different primitive shapes.
type PrimitiveKind int

const (
    PrimBoard PrimitiveKind = iota // rectangular solid (the most common)
    PrimPanel                      // thin rectangular sheet
    PrimDowel                      // cylindrical solid
    PrimCustom                     // user-defined geometry
)

// BoardData represents a rectangular piece of lumber.
type BoardData struct {
    PrimKind   PrimitiveKind `json:"prim_kind"`
    Dimensions Vec3          `json:"dimensions"`      // length x width x thickness
    Grain      Axis          `json:"grain"`           // dominant grain direction
    Material   MaterialSpec  `json:"material"`        // material intent
    Origin     Vec3          `json:"origin"`          // position in world space
}

func (BoardData) nodeData() {}

// MaterialSpec describes the intended material, advisory only.
type MaterialSpec struct {
    Species   string  `json:"species,omitempty"`    // e.g., "white-oak", "walnut"
    Thickness float64 `json:"thickness,omitempty"`  // nominal thickness in mm
    Grade     string  `json:"grade,omitempty"`      // e.g., "FAS", "select"
    Notes     string  `json:"notes,omitempty"`      // free-form
}

// DowelData represents a cylindrical piece (dowel, turning blank).
type DowelData struct {
    PrimKind PrimitiveKind `json:"prim_kind"`
    Diameter float64       `json:"diameter"`
    Length   float64       `json:"length"`
    Grain    Axis          `json:"grain"`
    Material MaterialSpec  `json:"material"`
    Origin   Vec3          `json:"origin"`
}

func (DowelData) nodeData() {}

// --------------------------------------------------------------------------
// Transform Node Data
// --------------------------------------------------------------------------

// TransformData represents a spatial transformation applied to children.
type TransformData struct {
    Translation *Vec3    `json:"translation,omitempty"`
    Rotation    *Vec3    `json:"rotation,omitempty"`    // Euler angles in degrees
    // Future: full 4x4 matrix for compound transforms
}

func (TransformData) nodeData() {}

// --------------------------------------------------------------------------
// Group Node Data
// --------------------------------------------------------------------------

// GroupData represents a logical grouping (assembly, subassembly).
type GroupData struct {
    Description string `json:"description,omitempty"`
}

func (GroupData) nodeData() {}

// --------------------------------------------------------------------------
// Join Node Data
// --------------------------------------------------------------------------

// JoinKind enumerates the types of woodworking joints.
type JoinKind int

const (
    JoinButt       JoinKind = iota // simple butt joint
    JoinRabbet                     // rabbet (rebate)
    JoinDado                       // dado (housing)
    JoinMortise                    // mortise and tenon
    JoinDovetail                   // dovetail
    JoinMiter                      // miter joint
    JoinBiscuit                    // biscuit joint
    JoinDowelJoint                 // dowel joint (not dowel primitive)
    JoinHalfLap                    // half-lap joint
    JoinTongueGroove               // tongue and groove
)

// JoinData specifies how two parts are connected.
type JoinData struct {
    Kind      JoinKind     `json:"kind"`
    PartA     NodeID       `json:"part_a"`          // first part
    FaceA     FaceID       `json:"face_a"`          // which face of part A
    PartB     NodeID       `json:"part_b"`          // second part
    FaceB     FaceID       `json:"face_b"`          // which face of part B
    Clearance float64      `json:"clearance"`       // gap in mm (default from global)
    Params    JoinParams   `json:"params"`          // kind-specific parameters
    Fasteners []NodeID     `json:"fasteners,omitempty"` // associated fastener nodes
}

func (JoinData) nodeData() {}

// JoinParams is a tagged union for joint-specific parameters.
type JoinParams interface {
    joinParams() // marker method
}

// ButtJoinParams -- simple butt joint, minimal parameters.
type ButtJoinParams struct {
    // Butt joints have no special geometry beyond face contact.
    // Strength comes entirely from fasteners/adhesive.
    GlueUp bool `json:"glue_up"` // whether glue is intended
}

func (ButtJoinParams) joinParams() {}

// RabbetJoinParams specifies rabbet dimensions.
type RabbetJoinParams struct {
    Depth float64 `json:"depth"` // how deep the rabbet is cut
    Width float64 `json:"width"` // how wide the rabbet is
}

func (RabbetJoinParams) joinParams() {}

// DadoJoinParams specifies dado dimensions.
type DadoJoinParams struct {
    Depth float64 `json:"depth"`    // depth of the dado channel
    Width float64 `json:"width"`    // width of the channel (should match mating part)
    Inset float64 `json:"inset"`    // distance from the edge (for stopped dados)
}

func (DadoJoinParams) joinParams() {}

// MortiseJoinParams specifies mortise and tenon dimensions.
type MortiseJoinParams struct {
    MortiseDepth  float64 `json:"mortise_depth"`
    MortiseWidth  float64 `json:"mortise_width"`
    MortiseHeight float64 `json:"mortise_height"`
    TenonLength   float64 `json:"tenon_length"`
    Offset        Vec3    `json:"offset"`          // position on the face
    Haunched      bool    `json:"haunched"`        // whether the tenon is haunched
}

func (MortiseJoinParams) joinParams() {}

// DovetailJoinParams specifies dovetail geometry.
type DovetailJoinParams struct {
    NumTails  int     `json:"num_tails"`
    TailAngle float64 `json:"tail_angle"` // degrees, typically 7-14
    PinWidth  float64 `json:"pin_width"`
    HalfBlind bool    `json:"half_blind"` // vs through dovetail
}

func (DovetailJoinParams) joinParams() {}

// --------------------------------------------------------------------------
// Drill Node Data
// --------------------------------------------------------------------------

// DrillData specifies a hole operation on a part.
type DrillData struct {
    TargetPart NodeID  `json:"target_part"`
    Face       FaceID  `json:"face"`           // which face the hole enters from
    Position   Vec3    `json:"position"`       // position on the face (local coords)
    Diameter   float64 `json:"diameter"`       // hole diameter in mm
    Depth      float64 `json:"depth"`          // hole depth (0 = through)
    Countersink *float64 `json:"countersink,omitempty"` // countersink diameter
    CounterBore *float64 `json:"counterbore,omitempty"` // counterbore diameter
}

func (DrillData) nodeData() {}

// --------------------------------------------------------------------------
// Fastener Node Data
// --------------------------------------------------------------------------

// FastenerKind enumerates fastener types.
type FastenerKind int

const (
    FastenerScrew    FastenerKind = iota
    FastenerNail
    FastenerDowelPin
    FastenerBolt
    FastenerBiscuit
)

// FastenerData specifies a fastener placed through a join.
type FastenerData struct {
    Kind     FastenerKind `json:"kind"`
    Diameter float64      `json:"diameter"`     // shank diameter in mm
    Length   float64      `json:"length"`       // total fastener length in mm
    HeadDia  float64      `json:"head_dia"`     // head diameter (for countersink calc)
    Position Vec3         `json:"position"`     // position relative to the join
    JoinRef  NodeID       `json:"join_ref"`     // which join this fastener belongs to
    // Screw-specific
    PilotHoleDia  float64 `json:"pilot_hole_dia,omitempty"`
    ClearanceHoleDia float64 `json:"clearance_hole_dia,omitempty"`
}

func (FastenerData) nodeData() {}

// --------------------------------------------------------------------------
// The Design Graph
// --------------------------------------------------------------------------

// DesignGraph is the top-level immutable data structure.
// It is produced entirely by Lisp evaluation and never mutated in place.
type DesignGraph struct {
    // Nodes maps node IDs to their definitions.
    Nodes map[NodeID]*Node `json:"nodes"`

    // Roots contains the top-level node IDs (entry points into the DAG).
    Roots []NodeID `json:"roots"`

    // NameIndex maps user-assigned names to node IDs for fast lookup.
    NameIndex map[string]NodeID `json:"name_index"`

    // GlobalDefaults contains graph-wide default settings.
    Defaults GlobalDefaults `json:"defaults"`

    // Version is incremented on each evaluation, for tracking.
    Version uint64 `json:"version"`
}

// GlobalDefaults contains default values inherited by nodes.
type GlobalDefaults struct {
    Clearance  float64      `json:"clearance"`   // default joint clearance in mm
    Material   MaterialSpec `json:"material"`     // default material for new parts
    Units      string       `json:"units"`        // "mm" or "in" (mm preferred)
}
```

### 1.4 Edge Semantics

Edges in the design graph are implicit: they are encoded by the `Children`
field on each node. This keeps the data structure simple and avoids the
complexity of a separate edge table. The edge types are determined by the
relationship between parent and child node kinds:

| Parent Kind | Child Kind | Meaning |
|---|---|---|
| Group | any | "contains" -- logical grouping |
| Transform | any | "transforms" -- spatial modification of child |
| Join | Primitive | "connects" -- the join references two parts |
| Drill | Primitive | "modifies" -- the drill targets a part |
| Fastener | Join | "secures" -- the fastener belongs to a join |

The DAG property is enforced structurally: a node may appear as a child
of multiple parents (structural sharing), but cycles are impossible because
nodes are created bottom-up during Lisp evaluation. A node can only reference
nodes that were created before it.

### 1.5 Immutability Model

The design graph follows a persistent data structure pattern inspired by
Clojure's approach and Merkle DAGs (as used in IPFS and Git):

1. **Construction is bottom-up.** Leaf nodes (primitives) are created first,
   then joins and transforms reference them, then groups collect everything.
2. **No mutation after creation.** Once a node is added to the graph, it is
   never modified. "Editing" produces a new graph that shares unchanged nodes
   with the previous version (structural sharing).
3. **The evaluator owns the graph.** Only the Lisp evaluation engine creates
   graphs. The geometry kernel, renderer, and UI consume them read-only.
4. **Diffing is cheap.** Two graphs can be compared by walking their root sets
   and comparing ContentHash values. Unchanged subtrees can be skipped entirely.

---

## 2. Part Naming and Reference API

### 2.1 How Parts are Registered

Parts are created by Lisp forms that evaluate to design graph nodes. The Lisp
API provides a `defpart` form that names a part and registers it in the graph's
`NameIndex`:

```lisp
;; Define a board part named "left-side"
(defpart "left-side"
  (board :length 600 :width 300 :thickness 19
         :grain :z      ; grain runs along Z axis (length)
         :material (material :species "white-oak")))

;; Define another part
(defpart "bottom"
  (board :length 600 :width 400 :thickness 19
         :grain :z
         :material (material :species "white-oak")))
```

When the evaluator encounters `defpart`, it:

1. Evaluates the body form to produce a `Node` with `Kind = NodePrimitive`.
2. Sets `Node.Name` to the provided string.
3. Registers the mapping `"left-side" -> NodeID` in `DesignGraph.NameIndex`.
4. Returns the `NodeID` for use in subsequent expressions.

### 2.2 How Parts are Looked Up

Parts can be referenced by name in subsequent Lisp forms:

```lisp
;; Reference by name in a join
(butt-joint
  :part-a (part "left-side") :face-a :right
  :part-b (part "bottom")    :face-b :top)
```

The `part` function performs a lookup in the `NameIndex`. If the name is not
found, evaluation fails with an error referencing the source location.

### 2.3 Part Reference API (Go Side)

```go
// Lookup retrieves a node by its user-assigned name.
// Returns nil, false if the name is not registered.
func (g *DesignGraph) Lookup(name string) (*Node, bool) {
    id, ok := g.NameIndex[name]
    if !ok {
        return nil, false
    }
    node, ok := g.Nodes[id]
    return node, ok
}

// MustLookup retrieves a node by name or panics.
// Used internally during graph construction.
func (g *DesignGraph) MustLookup(name string) *Node {
    node, ok := g.Lookup(name)
    if !ok {
        panic(fmt.Sprintf("part not found: %q", name))
    }
    return node
}

// Parts returns all nodes with Kind == NodePrimitive, in insertion order.
func (g *DesignGraph) Parts() []*Node {
    var parts []*Node
    for _, id := range g.insertionOrder { // maintained internally
        node := g.Nodes[id]
        if node.Kind == NodePrimitive {
            parts = append(parts, node)
        }
    }
    return parts
}

// Joins returns all nodes with Kind == NodeJoin.
func (g *DesignGraph) Joins() []*Node {
    var joins []*Node
    for _, id := range g.insertionOrder {
        node := g.Nodes[id]
        if node.Kind == NodeJoin {
            joins = append(joins, node)
        }
    }
    return joins
}

// Descendants returns all nodes reachable from the given node ID.
func (g *DesignGraph) Descendants(id NodeID) []*Node {
    visited := make(map[NodeID]bool)
    var result []*Node
    g.walkDescendants(id, visited, &result)
    return result
}

func (g *DesignGraph) walkDescendants(id NodeID, visited map[NodeID]bool, result *[]*Node) {
    if visited[id] {
        return
    }
    visited[id] = true
    node, ok := g.Nodes[id]
    if !ok {
        return
    }
    *result = append(*result, node)
    for _, childID := range node.Children {
        g.walkDescendants(childID, visited, result)
    }
}
```

### 2.4 Grain Direction Representation

Grain direction is represented as an `Axis` value on each part. The convention:

- **AxisX**: Grain runs along the X dimension (width of the board).
- **AxisY**: Grain runs along the Y dimension (height/thickness).
- **AxisZ**: Grain runs along the Z dimension (length of the board).

For most lumber, grain runs along the longest dimension. The Lisp API defaults
grain to the longest axis of the declared dimensions but allows explicit
override:

```lisp
;; Grain defaults to longest axis (Z here, since length=600 is largest)
(board :length 600 :width 300 :thickness 19)

;; Explicit grain override (e.g., cross-grain panel)
(board :length 600 :width 300 :thickness 19 :grain :x)
```

Grain direction matters for:
- **Validation:** Warning when a joint crosses grain in a structurally
  unsound way (e.g., mortise along the grain instead of across).
- **Stock mapping:** Grain must align between the design part and
  the physical stock board.
- **Rendering:** Visual indication of grain direction.

### 2.5 Parts that Emit Multiple Solids

A single design graph node may emit zero or more geometry solids. This supports:

- **Zero solids:** A group node or an abstract part not yet fully specified.
- **One solid:** The common case -- a board emits one rectangular solid.
- **Multiple solids:** A join node may emit modified versions of both parts
  it connects (e.g., a dado joint emits part A with a channel cut and part B
  unchanged). A complex assembly might emit all its constituent solids.

The multiplicity is tracked by the geometry kernel, not the design graph itself.
The design graph node knows its semantic identity; the geometry kernel
determines what geometry to produce from it.

---

## 3. Join Operation Specification Format

### 3.1 Design Principles

Lignin's join system encodes **woodworking intent**, not boolean operations.
This is a critical distinction from OpenSCAD-style CSG, where a dado joint
would be modeled as `difference(board, dado_channel_cube)`. In Lignin, a dado
joint is a first-class semantic operation that:

1. Knows which two parts are being connected.
2. Knows which faces are in contact.
3. Carries joint-specific parameters (depth, width, clearance).
4. Emits the correct geometry modifications to both parts.
5. Can be validated for structural soundness (grain, dimensions).
6. Can generate a cut list with joint-specific instructions.

### 3.2 Face Selection Model

Faces are identified by semantic names relative to the part's local
coordinate frame. For rectangular parts (boards, panels), the six faces are:

```
        +Y (top)
         |
         |
  -X ----+---- +X
 (left)  |   (right)
         |
        -Y (bottom)

  -Z (front) / +Z (back)
```

| FaceID | Axis Direction | Typical Woodworking Name |
|---|---|---|
| `top` | +Y | Top face |
| `bottom` | -Y | Bottom face |
| `left` | -X | Left end |
| `right` | +X | Right end |
| `front` | -Z | Front face |
| `back` | +Z | Back face |

For boards where length >> width >> thickness, the convention is:
- **length** runs along Z (grain direction, typically)
- **width** runs along X
- **thickness** runs along Y

This means:
- `top` / `bottom` = the **broad faces** (width x length)
- `left` / `right` = the **end grain faces** (width x thickness)
- `front` / `back` = the **edge faces** (length x thickness)

The face selection model is inspired by CadQuery's selector system but
simplified for the woodworking domain. CadQuery uses direction strings like
`">Z"` (topmost face along Z) with combinators. Lignin uses semantic names
because woodworking parts are overwhelmingly rectangular and the six-face
model is natural.

For future non-rectangular parts, faces can be identified by:
- **Semantic tag:** `"tenon-shoulder"`, `"mortise-wall"`
- **Normal direction selector:** `"+Y"`, `"-Z"` (CadQuery-style fallback)

### 3.3 Joint Specification in Lisp

```lisp
;; Butt joint: right face of left-side meets top face of bottom
(butt-joint
  :part-a (part "left-side") :face-a :right
  :part-b (part "bottom")    :face-b :top
  :clearance 0.25            ; 0.25mm gap (optional, overrides global)
  :fasteners
    (list
      (screw :diameter 4 :length 50 :position (vec3 50 0 100))
      (screw :diameter 4 :length 50 :position (vec3 50 0 300))))

;; Dado joint: bottom shelf sits in a dado cut into the side panel
(dado-joint
  :part-a (part "side-panel") :face-a :front   ; dado is cut into this face
  :part-b (part "shelf")      :face-b :left    ; this edge sits in the dado
  :depth 9.5                                    ; half the panel thickness
  :inset 100                                    ; 100mm from the bottom edge
  :clearance 0.25)

;; Mortise and tenon: rail connects to leg
(mortise-tenon
  :mortise-part (part "front-leg") :mortise-face :front
  :tenon-part   (part "side-rail") :tenon-face   :left
  :mortise-depth 30
  :mortise-width 25
  :mortise-height 50
  :tenon-length 28   ; 2mm shorter than mortise for glue pocket
  :offset (vec3 0 -20 0))  ; position on the face

;; Rabbet joint: back panel sits in a rabbet along the case sides
(rabbet-joint
  :part-a (part "case-side")  :face-a :back
  :part-b (part "back-panel") :face-b :front
  :depth 6      ; rabbet depth
  :width 12)    ; rabbet width (should match back panel thickness)

;; Dovetail joint: drawer front connects to drawer side
(dovetail-joint
  :tail-part (part "drawer-side")  :tail-face :front
  :pin-part  (part "drawer-front") :pin-face  :left
  :num-tails 4
  :tail-angle 11   ; degrees
  :half-blind true)
```

### 3.4 Clearance and Tolerance Model

Clearance is specified at two levels:

1. **Global default:** Set in the design graph's `GlobalDefaults.Clearance`.
   This is the gap applied to all joints unless overridden. A good default
   for hand-tool work is 0.25mm (0.010"); for CNC, 0.15mm (0.006").

2. **Per-joint override:** Each join node can specify its own clearance,
   overriding the global default. This supports mixed construction (e.g.,
   tight-fitting dovetails alongside looser dado joints).

The clearance is applied by the geometry kernel when generating the actual
solid geometry. The design graph stores the clearance value; the kernel
decides how to distribute it (e.g., enlarge the mortise, shrink the tenon,
or split the difference).

```go
// EffectiveClearance returns the clearance for a join node,
// falling back to the global default.
func (g *DesignGraph) EffectiveClearance(joinNode *Node) float64 {
    jd, ok := joinNode.Data.(JoinData)
    if !ok {
        return g.Defaults.Clearance
    }
    if jd.Clearance > 0 {
        return jd.Clearance
    }
    return g.Defaults.Clearance
}
```

### 3.5 Hole Drilling Operations

Drill operations are specified as separate nodes that reference a target part:

```lisp
;; Drill a pilot hole into the left-side panel
(drill
  :part (part "left-side")
  :face :right                        ; hole enters from the right face
  :position (vec3 50 9.5 100)         ; 50mm from left, centered, 100mm from front
  :diameter 3.2                       ; pilot hole for #8 screw
  :depth 25)                          ; 25mm deep

;; Drill with countersink
(drill
  :part (part "left-side")
  :face :right
  :position (vec3 50 9.5 100)
  :diameter 5                         ; clearance hole for #8 screw
  :depth 0                            ; through hole
  :countersink 9.5)                   ; countersink for #8 flat-head screw

;; Drill with counterbore (for plugged screws)
(drill
  :part (part "left-side")
  :face :right
  :position (vec3 50 9.5 100)
  :diameter 5
  :depth 0
  :counterbore 9.5                    ; counterbore diameter
  :counterbore-depth 6)               ; depth for 3/8" plug
```

### 3.6 Fastener Embedding

Fasteners are associated with joins. When a screw is specified on a butt
joint, the system can automatically generate the required drill operations
(pilot hole in part B, clearance hole in part A, optional countersink/
counterbore in part A):

```lisp
;; Screw fastener with auto-generated holes
(screw
  :diameter 4          ; #8 screw, 4mm shank
  :length 50           ; 50mm total length
  :head :flat          ; flat head (countersink) vs :pan (no countersink)
  :position (vec3 50 0 100)  ; position on the joint face
  :pilot-hole 3.2      ; pilot hole diameter
  :clearance-hole 5)   ; clearance hole diameter
```

The fastener node references its parent join node. The geometry kernel
uses the fastener specification plus the join's face geometry to compute
drill positions and orientations for both parts.

---

## 4. Worked Example: Simple Box

A simple open-top box with 4 sides and a bottom, joined with butt joints
and screws.

### 4.1 Dimensions

- Interior: 400mm x 300mm x 200mm (L x W x H)
- Material: 19mm white oak
- Joints: butt joints, bottom sits on top of lower edges of sides
- Fasteners: #8 x 50mm screws, 2 per corner, 3 along bottom edges

### 4.2 Lisp Source

```lisp
(define box-height 200)
(define box-length 400)
(define box-width 300)
(define thickness 19)

(define oak (material :species "white-oak" :thickness thickness))

;; The four sides: front/back run the full length, left/right fit between them.
;; This is a "through" construction where front/back overlap left/right.

(defpart "front"
  (board :length box-length :width box-height :thickness thickness
         :grain :z :material oak))

(defpart "back"
  (board :length box-length :width box-height :thickness thickness
         :grain :z :material oak))

(defpart "left"
  (board :length (- box-width (* 2 thickness)) :width box-height :thickness thickness
         :grain :z :material oak))

(defpart "right"
  (board :length (- box-width (* 2 thickness)) :width box-height :thickness thickness
         :grain :z :material oak))

(defpart "bottom"
  (board :length (- box-length (* 2 thickness))
         :width (- box-width (* 2 thickness))
         :thickness thickness
         :grain :z :material oak))

;; Position the parts
(assembly "box"
  ;; Front panel at z=0
  (place (part "front") :at (vec3 0 0 0))

  ;; Back panel at z = box-width - thickness
  (place (part "back") :at (vec3 0 0 (- box-width thickness)))

  ;; Left side between front and back
  (place (part "left") :at (vec3 0 0 thickness))

  ;; Right side
  (place (part "right") :at (vec3 (- box-length thickness) 0 thickness))

  ;; Bottom panel, resting on lower edges
  (place (part "bottom") :at (vec3 thickness 0 thickness))

  ;; Corner joints: front-left, front-right, back-left, back-right
  (butt-joint
    :part-a (part "front") :face-a :left
    :part-b (part "left")  :face-b :front
    :fasteners
      (list
        (screw :diameter 4 :length 50 :position (vec3 0 50 0))
        (screw :diameter 4 :length 50 :position (vec3 0 150 0))))

  (butt-joint
    :part-a (part "front") :face-a :right
    :part-b (part "right") :face-b :front
    :fasteners
      (list
        (screw :diameter 4 :length 50 :position (vec3 0 50 0))
        (screw :diameter 4 :length 50 :position (vec3 0 150 0))))

  (butt-joint
    :part-a (part "back") :face-a :left
    :part-b (part "left") :face-b :back
    :fasteners
      (list
        (screw :diameter 4 :length 50 :position (vec3 0 50 0))
        (screw :diameter 4 :length 50 :position (vec3 0 150 0))))

  (butt-joint
    :part-a (part "back") :face-a :right
    :part-b (part "right") :face-b :back
    :fasteners
      (list
        (screw :diameter 4 :length 50 :position (vec3 0 50 0))
        (screw :diameter 4 :length 50 :position (vec3 0 150 0))))

  ;; Bottom joints
  (butt-joint
    :part-a (part "front")  :face-a :bottom
    :part-b (part "bottom") :face-b :front
    :fasteners
      (list
        (screw :diameter 4 :length 50 :position (vec3 100 0 0))
        (screw :diameter 4 :length 50 :position (vec3 200 0 0))
        (screw :diameter 4 :length 50 :position (vec3 300 0 0))))

  (butt-joint
    :part-a (part "back")   :face-a :bottom
    :part-b (part "bottom") :face-b :back
    :fasteners
      (list
        (screw :diameter 4 :length 50 :position (vec3 100 0 0))
        (screw :diameter 4 :length 50 :position (vec3 200 0 0))
        (screw :diameter 4 :length 50 :position (vec3 300 0 0))))

  (butt-joint
    :part-a (part "left")   :face-a :bottom
    :part-b (part "bottom") :face-b :left
    :fasteners
      (list
        (screw :diameter 4 :length 50 :position (vec3 0 0 50))
        (screw :diameter 4 :length 50 :position (vec3 0 0 130))))

  (butt-joint
    :part-a (part "right")  :face-a :bottom
    :part-b (part "bottom") :face-b :right
    :fasteners
      (list
        (screw :diameter 4 :length 50 :position (vec3 0 0 50))
        (screw :diameter 4 :length 50 :position (vec3 0 0 130)))))
```

### 4.3 Resulting Design Graph (Serialized)

Below is the full data structure that the evaluator produces. Node IDs
are shown as abbreviated hex strings for readability.

```json
{
  "version": 1,
  "defaults": {
    "clearance": 0.25,
    "material": { "species": "white-oak", "thickness": 19 },
    "units": "mm"
  },
  "roots": ["node:group:box"],
  "name_index": {
    "front": "node:a1b2",
    "back": "node:c3d4",
    "left": "node:e5f6",
    "right": "node:g7h8",
    "bottom": "node:i9j0"
  },
  "nodes": {
    "node:a1b2": {
      "id": "node:a1b2",
      "kind": "primitive",
      "name": "front",
      "source": { "line": 10, "col": 1, "form_id": "defpart:front" },
      "data": {
        "prim_kind": "board",
        "dimensions": { "x": 400, "y": 200, "z": 19 },
        "grain": "z",
        "material": { "species": "white-oak", "thickness": 19 },
        "origin": { "x": 0, "y": 0, "z": 0 }
      }
    },
    "node:c3d4": {
      "id": "node:c3d4",
      "kind": "primitive",
      "name": "back",
      "source": { "line": 14, "col": 1, "form_id": "defpart:back" },
      "data": {
        "prim_kind": "board",
        "dimensions": { "x": 400, "y": 200, "z": 19 },
        "grain": "z",
        "material": { "species": "white-oak", "thickness": 19 },
        "origin": { "x": 0, "y": 0, "z": 281 }
      }
    },
    "node:e5f6": {
      "id": "node:e5f6",
      "kind": "primitive",
      "name": "left",
      "source": { "line": 18, "col": 1, "form_id": "defpart:left" },
      "data": {
        "prim_kind": "board",
        "dimensions": { "x": 262, "y": 200, "z": 19 },
        "grain": "z",
        "material": { "species": "white-oak", "thickness": 19 },
        "origin": { "x": 0, "y": 0, "z": 19 }
      }
    },
    "node:g7h8": {
      "id": "node:g7h8",
      "kind": "primitive",
      "name": "right",
      "source": { "line": 22, "col": 1, "form_id": "defpart:right" },
      "data": {
        "prim_kind": "board",
        "dimensions": { "x": 262, "y": 200, "z": 19 },
        "grain": "z",
        "material": { "species": "white-oak", "thickness": 19 },
        "origin": { "x": 381, "y": 0, "z": 19 }
      }
    },
    "node:i9j0": {
      "id": "node:i9j0",
      "kind": "primitive",
      "name": "bottom",
      "source": { "line": 26, "col": 1, "form_id": "defpart:bottom" },
      "data": {
        "prim_kind": "board",
        "dimensions": { "x": 362, "y": 262, "z": 19 },
        "grain": "z",
        "material": { "species": "white-oak", "thickness": 19 },
        "origin": { "x": 19, "y": 0, "z": 19 }
      }
    },
    "node:join:fl": {
      "id": "node:join:fl",
      "kind": "join",
      "source": { "line": 36, "col": 3, "form_id": "butt-joint:front-left" },
      "data": {
        "kind": "butt",
        "part_a": "node:a1b2",
        "face_a": "left",
        "part_b": "node:e5f6",
        "face_b": "front",
        "clearance": 0.25,
        "params": { "glue_up": true },
        "fasteners": ["node:screw:fl1", "node:screw:fl2"]
      }
    },
    "node:join:fr": {
      "id": "node:join:fr",
      "kind": "join",
      "source": { "line": 43, "col": 3, "form_id": "butt-joint:front-right" },
      "data": {
        "kind": "butt",
        "part_a": "node:a1b2",
        "face_a": "right",
        "part_b": "node:g7h8",
        "face_b": "front",
        "clearance": 0.25,
        "params": { "glue_up": true },
        "fasteners": ["node:screw:fr1", "node:screw:fr2"]
      }
    },
    "node:join:bl": {
      "id": "node:join:bl",
      "kind": "join",
      "source": { "line": 50, "col": 3, "form_id": "butt-joint:back-left" },
      "data": {
        "kind": "butt",
        "part_a": "node:c3d4",
        "face_a": "left",
        "part_b": "node:e5f6",
        "face_b": "back",
        "clearance": 0.25,
        "params": { "glue_up": true },
        "fasteners": ["node:screw:bl1", "node:screw:bl2"]
      }
    },
    "node:join:br": {
      "id": "node:join:br",
      "kind": "join",
      "source": { "line": 57, "col": 3, "form_id": "butt-joint:back-right" },
      "data": {
        "kind": "butt",
        "part_a": "node:c3d4",
        "face_a": "right",
        "part_b": "node:g7h8",
        "face_b": "back",
        "clearance": 0.25,
        "params": { "glue_up": true },
        "fasteners": ["node:screw:br1", "node:screw:br2"]
      }
    },
    "node:join:fb": {
      "id": "node:join:fb",
      "kind": "join",
      "source": { "line": 64, "col": 3, "form_id": "butt-joint:front-bottom" },
      "data": {
        "kind": "butt",
        "part_a": "node:a1b2",
        "face_a": "bottom",
        "part_b": "node:i9j0",
        "face_b": "front",
        "clearance": 0.25,
        "params": { "glue_up": true },
        "fasteners": ["node:screw:fb1", "node:screw:fb2", "node:screw:fb3"]
      }
    },
    "node:join:bb": {
      "id": "node:join:bb",
      "kind": "join",
      "source": { "line": 72, "col": 3, "form_id": "butt-joint:back-bottom" },
      "data": {
        "kind": "butt",
        "part_a": "node:c3d4",
        "face_a": "bottom",
        "part_b": "node:i9j0",
        "face_b": "back",
        "clearance": 0.25,
        "params": { "glue_up": true },
        "fasteners": ["node:screw:bb1", "node:screw:bb2", "node:screw:bb3"]
      }
    },
    "node:join:lb": {
      "id": "node:join:lb",
      "kind": "join",
      "source": { "line": 80, "col": 3, "form_id": "butt-joint:left-bottom" },
      "data": {
        "kind": "butt",
        "part_a": "node:e5f6",
        "face_a": "bottom",
        "part_b": "node:i9j0",
        "face_b": "left",
        "clearance": 0.25,
        "params": { "glue_up": true },
        "fasteners": ["node:screw:lb1", "node:screw:lb2"]
      }
    },
    "node:join:rb": {
      "id": "node:join:rb",
      "kind": "join",
      "source": { "line": 87, "col": 3, "form_id": "butt-joint:right-bottom" },
      "data": {
        "kind": "butt",
        "part_a": "node:g7h8",
        "face_a": "bottom",
        "part_b": "node:i9j0",
        "face_b": "right",
        "clearance": 0.25,
        "params": { "glue_up": true },
        "fasteners": ["node:screw:rb1", "node:screw:rb2"]
      }
    },
    "node:screw:fl1": {
      "id": "node:screw:fl1",
      "kind": "fastener",
      "source": { "line": 40, "col": 9, "form_id": "screw:front-left-1" },
      "data": {
        "kind": "screw",
        "diameter": 4,
        "length": 50,
        "head_dia": 8,
        "position": { "x": 0, "y": 50, "z": 0 },
        "join_ref": "node:join:fl",
        "pilot_hole_dia": 3.2,
        "clearance_hole_dia": 5
      }
    },
    "node:group:box": {
      "id": "node:group:box",
      "kind": "group",
      "name": "box",
      "source": { "line": 32, "col": 1, "form_id": "assembly:box" },
      "children": [
        "node:a1b2", "node:c3d4", "node:e5f6", "node:g7h8", "node:i9j0",
        "node:join:fl", "node:join:fr", "node:join:bl", "node:join:br",
        "node:join:fb", "node:join:bb", "node:join:lb", "node:join:rb"
      ],
      "data": {
        "description": "Simple open-top box"
      }
    }
  }
}
```

Note: Only one fastener node is shown in full (`node:screw:fl1`) for brevity.
The actual graph contains 18 screw nodes (2 per corner x 4 corners = 8 for
sides, plus 3 per bottom edge x 2 long edges + 2 per bottom edge x 2 short
edges = 10 for bottom).

### 4.4 Graph Topology Diagram

```
                    node:group:box
                   /   |    |   \   \
                  /    |    |    \    \-------...
                 /     |    |     \
        node:a1b2  node:c3d4  node:e5f6  node:g7h8  node:i9j0
        (front)    (back)     (left)     (right)    (bottom)
              \      |        /     \      |        /
               \     |       /       \     |       /
            node:join:fl  node:join:fr  node:join:bl  ...
               |     |
        node:screw:fl1  node:screw:fl2
```

The group node "box" has all parts and joins as children. Each join node
references two parts (via PartA/PartB in its data, not via Children). Each
fastener node references its join. This creates a DAG -- parts are referenced
by both the group and the joins, but there are no cycles.

---

## 5. Validation Rules

### 5.1 Structural Validation (Always Enforced)

These rules ensure the graph is well-formed regardless of domain semantics.

#### 5.1.1 DAG Property (No Cycles)

The graph must be acyclic. Since nodes are constructed bottom-up during
Lisp evaluation, cycles are structurally impossible in normal operation.
However, validation should still check:

```go
// ValidateDAG checks for cycles using DFS with coloring.
// Returns an error describing the cycle if one is found.
func (g *DesignGraph) ValidateDAG() error {
    const (
        white = iota // unvisited
        gray         // in current DFS path
        black        // fully explored
    )
    color := make(map[NodeID]int)

    var visit func(id NodeID, path []NodeID) error
    visit = func(id NodeID, path []NodeID) error {
        switch color[id] {
        case gray:
            return fmt.Errorf("cycle detected: %v -> %v", path, id)
        case black:
            return nil
        }
        color[id] = gray
        path = append(path, id)

        node, ok := g.Nodes[id]
        if !ok {
            return fmt.Errorf("dangling reference: %v", id)
        }

        for _, childID := range node.Children {
            if err := visit(childID, path); err != nil {
                return err
            }
        }

        // Also check join references
        if jd, ok := node.Data.(JoinData); ok {
            if err := visit(jd.PartA, path); err != nil {
                return err
            }
            if err := visit(jd.PartB, path); err != nil {
                return err
            }
        }

        color[id] = black
        return nil
    }

    for _, rootID := range g.Roots {
        if err := visit(rootID, nil); err != nil {
            return err
        }
    }
    return nil
}
```

#### 5.1.2 Reference Integrity

Every NodeID referenced anywhere in the graph must exist in `Nodes`:

- All IDs in `Roots` must exist.
- All IDs in any node's `Children` must exist.
- `JoinData.PartA` and `JoinData.PartB` must exist and be `NodePrimitive`.
- `JoinData.Fasteners` must all exist and be `NodeFastener`.
- `DrillData.TargetPart` must exist and be `NodePrimitive`.
- `FastenerData.JoinRef` must exist and be `NodeJoin`.

```go
func (g *DesignGraph) ValidateReferences() []error {
    var errs []error
    for id, node := range g.Nodes {
        for _, childID := range node.Children {
            if _, ok := g.Nodes[childID]; !ok {
                errs = append(errs, fmt.Errorf(
                    "node %v references missing child %v", id, childID))
            }
        }
        switch d := node.Data.(type) {
        case JoinData:
            if a, ok := g.Nodes[d.PartA]; !ok {
                errs = append(errs, fmt.Errorf(
                    "join %v references missing part_a %v", id, d.PartA))
            } else if a.Kind != NodePrimitive {
                errs = append(errs, fmt.Errorf(
                    "join %v part_a %v is not a primitive", id, d.PartA))
            }
            if b, ok := g.Nodes[d.PartB]; !ok {
                errs = append(errs, fmt.Errorf(
                    "join %v references missing part_b %v", id, d.PartB))
            } else if b.Kind != NodePrimitive {
                errs = append(errs, fmt.Errorf(
                    "join %v part_b %v is not a primitive", id, d.PartB))
            }
        case DrillData:
            if _, ok := g.Nodes[d.TargetPart]; !ok {
                errs = append(errs, fmt.Errorf(
                    "drill %v references missing target %v", id, d.TargetPart))
            }
        case FastenerData:
            if _, ok := g.Nodes[d.JoinRef]; !ok {
                errs = append(errs, fmt.Errorf(
                    "fastener %v references missing join %v", id, d.JoinRef))
            }
        }
    }
    return errs
}
```

#### 5.1.3 Name Uniqueness

The `NameIndex` must be injective: no two nodes share the same user name.
This is enforced during graph construction by the evaluator.

#### 5.1.4 Root Reachability

Every node in the graph should be reachable from at least one root. Orphan
nodes indicate a bug in the evaluator. (Warning, not error, since orphans
are harmless.)

### 5.2 Geometric Validation (Tier 1: Geometry-Only)

These rules check geometric consistency without considering material.

#### 5.2.1 Non-Zero Dimensions

All part dimensions must be positive:

```go
func validateDimensions(node *Node) error {
    switch d := node.Data.(type) {
    case BoardData:
        if d.Dimensions.X <= 0 || d.Dimensions.Y <= 0 || d.Dimensions.Z <= 0 {
            return fmt.Errorf("part %q has zero or negative dimension: %v",
                node.Name, d.Dimensions)
        }
    case DowelData:
        if d.Diameter <= 0 || d.Length <= 0 {
            return fmt.Errorf("dowel %q has zero or negative dimension", node.Name)
        }
    }
    return nil
}
```

#### 5.2.2 Joint Face Compatibility

For a butt joint, the two selected faces must be compatible for contact:
- They should be roughly the same size (within tolerance), or one should
  be contained within the other.
- The face normals should be anti-parallel (facing each other).

#### 5.2.3 No Self-Joins

A join must connect two different parts: `PartA != PartB`.

#### 5.2.4 No Duplicate Joins

Two joins should not connect the same pair of parts on the same faces.

#### 5.2.5 Fastener Length Check

A fastener's length should not exceed the combined thickness of the parts
it passes through at the join point.

#### 5.2.6 Drill Depth Check

A drill operation's depth should not exceed the part dimension along the
drill axis. A through-hole (depth=0) is always valid.

### 5.3 Material-Aware Validation (Tier 2: Warnings)

These produce warnings, not fatal errors.

#### 5.3.1 Grain Direction Warnings

- **End-grain butt joint:** If a butt joint connects two end-grain faces,
  warn that the joint will be weak. End grain does not glue well.
- **Cross-grain dado:** If a dado runs parallel to the grain instead of
  across it, warn about reduced strength.
- **Short-grain tenon:** If a tenon's grain runs across its length rather
  than along it, warn about breakage risk.

```go
func validateGrain(g *DesignGraph, joinNode *Node) []Warning {
    jd := joinNode.Data.(JoinData)
    partA := g.Nodes[jd.PartA]
    partB := g.Nodes[jd.PartB]

    grainA := partA.Data.(BoardData).Grain
    grainB := partB.Data.(BoardData).Grain

    faceAxisA := faceToAxis(jd.FaceA)
    faceAxisB := faceToAxis(jd.FaceB)

    var warnings []Warning

    // End-grain to end-grain butt joint check
    if jd.Kind == JoinButt {
        if grainA == faceAxisA.Axis && grainB == faceAxisB.Axis {
            warnings = append(warnings, Warning{
                Node:    joinNode.ID,
                Message: "end-grain to end-grain butt joint: very weak bond",
                Level:   WarnMaterial,
            })
        }
    }

    return warnings
}
```

#### 5.3.2 Minimum Thickness Warnings

- Rabbet depth should not exceed 1/2 the part thickness.
- Dado depth should not exceed 1/3 the part thickness.
- Mortise width should not exceed 1/3 the part thickness.

#### 5.3.3 Material Mismatch

If two parts in a join have different species, warn about differential
wood movement. (Not an error -- mixed species is sometimes intentional.)

### 5.4 Stock-Aware Validation (Tier 3: Advisory)

These validations only run when stock mapping is active.

- Part dimensions must fit within available stock.
- Grain direction must be achievable from the stock board.
- Warnings when stock waste exceeds a threshold.

Per the PRD, stock allocation failures produce warnings, not errors.

---

## 6. Research Notes

### 6.1 Lessons from OpenSCAD

OpenSCAD uses a CSG tree as its internal representation, where every node is
either a primitive (cube, sphere, cylinder), a boolean operation (union,
difference, intersection), or a transform (multmatrix). This tree is then
"normalized" into a form suitable for rendering by OpenCSG or evaluation by
CGAL's Nef polyhedron engine.

**What Lignin borrows:** The DAG structure with primitives and transforms as
nodes, bottom-up construction, and the concept of a normalized form suitable
for consumption by a geometry backend.

**What Lignin changes:** Lignin elevates the abstraction level. Instead of
raw CSG boolean operations, Lignin has semantic join nodes that carry
woodworking intent. A dado joint in OpenSCAD is `difference(board, channel)`.
In Lignin, it is `(dado-joint :part-a ... :part-b ... :depth ...)`. The
geometry kernel translates the semantic join into the appropriate boolean
operations internally.

**Reference:** [OpenSCAD CSG File Format](https://github.com/openscad/openscad/wiki/CSG-File-Format),
[OpenSCAD Wikipedia](https://en.wikipedia.org/wiki/OpenSCAD)

### 6.2 Lessons from CadQuery

CadQuery's workplane-and-selector model is powerful for programmatic face
selection. Its string-based selector syntax (`">Z"`, `"<X"`) allows concise
specification of which face to operate on, with combinators for complex
selections.

**What Lignin borrows:** The idea of selecting faces by semantic identity
(direction-based naming). CadQuery's selector classes (NearestToPointSelector,
ParallelDirSelector, etc.) inform Lignin's future extensibility path for
non-rectangular parts.

**What Lignin simplifies:** Since woodworking parts are overwhelmingly
rectangular, Lignin uses a fixed six-face naming scheme (top/bottom/left/
right/front/back) instead of a general selector language. This makes the
common case trivial while leaving room for extension.

**Reference:** [CadQuery Selectors Reference](https://cadquery.readthedocs.io/en/latest/selectors.html),
[CadQuery API Reference](https://cadquery.readthedocs.io/en/latest/apireference.html)

### 6.3 Lessons from GhostSCAD (Go)

GhostSCAD demonstrates that a Go-based CAD system can use a tree of nodes
implementing a common `Primitive` interface, with complex shapes composed
from simple ones via a `Build()` method. Its separation of the program's
abstract syntax tree from the geometry tree is directly relevant to Lignin.

**What Lignin borrows:** The interface-based node type system, the separation
of language evaluation from geometry construction, and the convention of
complex shapes having a builder that constructs geometry from primitives.

**Reference:** [GhostSCAD](https://jany.st/post/2022-04-04-ghostscad-marrying-openscad-and-golang.html),
[GhostSCAD primitives package](https://pkg.go.dev/github.com/ljanyst/ghostscad/primitive)

### 6.4 Lessons from sdfx (Go SDF Library)

The sdfx library demonstrates Go-native 3D CAD using signed distance functions.
It shows that Go is viable for CAD work with good performance, and that the
SDF approach makes CSG operations (union, difference, intersection) trivial
as min/max operations on distance fields. Filleting and chamfering -- which
are important for woodworking -- become straightforward with SDFs.

**Relevance to Lignin:** The geometry kernel could potentially use an SDF
approach rather than B-rep for initial prototyping, since SDF boolean
operations are simpler to implement. However, the PRD specifies B-rep or
half-edge, and B-rep provides exact geometry needed for dimensioned output.
SDF could be a preview/approximation backend.

**Reference:** [sdfx](https://github.com/deadsy/sdfx),
[soypat/sdf](https://github.com/soypat/sdf)

### 6.5 Lessons from Tsugite (Computational Wood Joinery)

Tsugite is a research system for interactive design of wood joints for CNC
fabrication. It uses a voxel grid as its design space, where each voxel
belongs to one timber. The system evaluates joints in real time for
slidability, fabricability, and durability with respect to fiber direction.

**What Lignin borrows:** The concept of evaluating joint feasibility against
grain direction. Tsugite's real-time feedback model -- where the system
warns about invalid or suboptimal joints as the user designs -- directly
informs Lignin's tiered validation approach. Tsugite also demonstrates that
fabrication constraints (CNC bit radius causing rounded inner corners) should
be tracked as metadata, not baked into the design geometry.

**Reference:** [Tsugite paper (UIST 2020)](https://www.ma-la.com/tsugite/Tsugite_UIST20.pdf),
[Tsugite GitHub](https://github.com/marialarsson/tsugite)

### 6.6 Content-Addressed vs UUID Identity

The Merkle DAG pattern (used by IPFS, Git, and databases like DefraDB and
Fireproof) provides self-verifying, deduplicating node identity through
content hashing. However, pure content addressing makes identity unstable
across edits -- any change to a node changes its hash and all ancestors.

**Lignin's hybrid approach** uses source-expression-path hashing for stable
identity (similar to Git's branch refs pointing to content-addressed commits)
and content hashing for change detection (similar to Git's tree hash
comparison). This gives the best of both worlds: stable references for the
UI and renderer, plus efficient diffing for incremental re-evaluation.

**Reference:** [IPFS Merkle DAG](https://docs.ipfs.tech/concepts/merkle-dag/),
[merkledag-core](https://github.com/greglook/merkledag-core)

### 6.7 Clearance and Tolerance in Practice

Research and community practice converge on these tolerance values for
woodworking joints:

| Context | Typical Clearance |
|---|---|
| Hand-cut joints | 0.25 - 0.50 mm |
| CNC-cut plywood joints | 0.15 - 0.25 mm (0.006" - 0.010") |
| CNC-cut hardwood joints | 0.10 - 0.20 mm |
| Glue joints (no gap) | 0.05 - 0.10 mm |
| Sliding fits (drawers) | 0.50 - 1.00 mm |

CNC bit runout (0.05 - 0.10 mm effective diameter increase) and bit wear
should be accounted for. Lignin's clearance model intentionally does not
try to model these machine-specific factors -- it stores the **design intent**
clearance. Machine calibration is a CAM concern, which is explicitly a
non-goal per the PRD.

**Reference:** [CNC Tolerance and Fit](https://whatmakeart.com/digital-fabrication/cnc/cnc-tolerance-and-fit/),
[Swedish Wood Joinery Handbook](https://www.swedishwood.com/siteassets/5-publikationer/pdfer/joinery-handbook.pdf)

### 6.8 Woodworking Joint Type Summary

| Joint Type | Strength | Complexity | Primary Use | Geometry Modification |
|---|---|---|---|---|
| **Butt** | Weak (needs fasteners) | Simple | Rough construction, boxes | None (face contact only) |
| **Rabbet** | Moderate | Simple | Box corners, back panels | Channel cut along edge |
| **Dado** | Moderate-Strong | Simple | Shelving, case goods | Channel cut across face |
| **Mortise & Tenon** | Very Strong | Moderate | Frame construction, tables | Pocket + protruding tongue |
| **Dovetail** | Very Strong | Complex | Drawers, fine boxes | Interlocking pins and tails |
| **Half-Lap** | Moderate | Simple | Frames, face frames | Half-thickness removal on both parts |
| **Miter** | Weak (needs reinforcement) | Moderate | Picture frames, moldings | 45-degree cut on both parts |
| **Tongue & Groove** | Moderate | Moderate | Panel glue-ups, flooring | Matching tongue and channel |
| **Biscuit** | Moderate | Simple (with tool) | Panel glue-ups, alignment | Slot in both parts |
| **Dowel Joint** | Moderate-Strong | Simple (with jig) | General purpose | Holes in both parts |

**Reference:** [Types of Wood Joints](https://woodworkingworld.org/types-of-woodworking-joints-a-complete-guide/),
[Wagner Meters Guide](https://www.wagnermeters.com/moisture-meters/wood-info/the-ultimate-guide-to-woodworking-joints/)

---

## Appendix A: Open Questions

1. **Board dimension ordering convention.** The PRD does not specify whether
   `Dimensions.X` means "length" or "width." This document proposes:
   X = width, Y = thickness, Z = length (grain direction). But "length along
   Z" is non-obvious. An alternative: use named fields (`Length`, `Width`,
   `Thickness`) and map them to axes based on grain direction.

2. **Transform composition.** Should transforms be composable (a transform
   node can be a child of another transform), or should only a single
   transform be applied per part? Composable transforms are more flexible
   but make the graph harder to reason about.

3. **Join ownership.** Should joins be children of the group that contains
   their parts, or should they be siblings? The worked example above makes
   them siblings within the group, but there is a case for making them
   children of a "joinery" sub-group.

4. **Face selection for non-rectangular parts.** The six-face model works
   for boards and panels. For turned parts (lathe work), compound-curved
   parts, or parts modified by joinery (e.g., a board with a dado cut),
   how should faces be identified? One approach: semantic tags applied by
   the operation that created the face.

5. **Incremental re-evaluation strategy.** The design graph is re-derived
   on every evaluation. For large designs, this could be slow. The
   ContentHash-based diffing enables skipping unchanged subtrees, but the
   exact mechanism needs design. Should the evaluator cache intermediate
   results? Should the geometry kernel cache solids keyed by ContentHash?

6. **Multi-solid emission.** When a join modifies two parts, does it emit
   new "modified part" nodes, or does the geometry kernel maintain a
   separate mapping from original-part-NodeID to modified-solid? The latter
   keeps the design graph clean but requires the kernel to maintain state.

---

## Appendix B: Dimension Convention Detail

For the MVP, boards use named dimensions to avoid axis confusion:

```go
// BoardDimensions uses named fields instead of raw Vec3 to avoid
// confusion about which axis is which.
type BoardDimensions struct {
    Length    float64 `json:"length"`    // along grain direction
    Width     float64 `json:"width"`     // perpendicular to grain, broad face
    Thickness float64 `json:"thickness"` // perpendicular to grain, narrow dimension
}

// ToVec3 converts to a Vec3 based on the grain axis convention.
// Grain axis gets the Length value.
// For AxisZ grain (default): X=Width, Y=Thickness, Z=Length
// For AxisX grain: X=Length, Y=Thickness, Z=Width
// For AxisY grain: X=Width, Y=Length, Z=Thickness
func (d BoardDimensions) ToVec3(grain Axis) Vec3 {
    switch grain {
    case AxisZ:
        return Vec3{X: d.Width, Y: d.Thickness, Z: d.Length}
    case AxisX:
        return Vec3{X: d.Length, Y: d.Thickness, Z: d.Width}
    case AxisY:
        return Vec3{X: d.Width, Y: d.Length, Z: d.Thickness}
    default:
        return Vec3{X: d.Width, Y: d.Thickness, Z: d.Length}
    }
}
```

This eliminates the ambiguity identified in Open Question #1 while keeping
the internal representation axis-aligned for the geometry kernel.
