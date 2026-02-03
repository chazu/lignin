package graph

// ---------------------------------------------------------------------------
// Material
// ---------------------------------------------------------------------------

// MaterialSpec describes the intended material. Advisory only.
type MaterialSpec struct {
	Species   string  `json:"species,omitempty"`   // e.g. "white-oak", "walnut"
	Thickness float64 `json:"thickness,omitempty"` // nominal thickness in mm
	Grade     string  `json:"grade,omitempty"`     // e.g. "FAS", "select"
	Notes     string  `json:"notes,omitempty"`
}

// ---------------------------------------------------------------------------
// Primitives
// ---------------------------------------------------------------------------

// PrimitiveKind distinguishes between primitive shapes.
type PrimitiveKind int

const (
	PrimBoard PrimitiveKind = iota // rectangular solid
	PrimDowel                      // cylindrical solid
)

// BoardData represents a rectangular piece of lumber.
type BoardData struct {
	PrimKind   PrimitiveKind `json:"prim_kind"`
	Dimensions Vec3          `json:"dimensions"` // length x width x thickness in mm
	Grain      Axis          `json:"grain"`      // dominant grain direction
	Material   MaterialSpec  `json:"material"`
}

func (BoardData) nodeData() {}

// DowelData represents a cylindrical piece (dowel rod, turned stock).
type DowelData struct {
	PrimKind PrimitiveKind `json:"prim_kind"`
	Diameter float64       `json:"diameter"` // mm
	Length   float64       `json:"length"`   // mm
	Grain    Axis          `json:"grain"`
	Material MaterialSpec  `json:"material"`
}

func (DowelData) nodeData() {}

// ---------------------------------------------------------------------------
// Transform
// ---------------------------------------------------------------------------

// TransformData represents a spatial transformation applied to a child node.
// Created by the (place ...) Lisp form.
type TransformData struct {
	Translation *Vec3 `json:"translation,omitempty"`
	Rotation    *Vec3 `json:"rotation,omitempty"` // Euler angles in degrees
}

func (TransformData) nodeData() {}

// ---------------------------------------------------------------------------
// Group
// ---------------------------------------------------------------------------

// GroupData represents a logical grouping (assembly, subassembly).
// Created by the (assembly ...) Lisp form.
type GroupData struct {
	Description string `json:"description,omitempty"`
}

func (GroupData) nodeData() {}

// ---------------------------------------------------------------------------
// Join
// ---------------------------------------------------------------------------

// JoinKind enumerates woodworking joint types.
type JoinKind int

const (
	JoinButt     JoinKind = iota // butt joint (MVP)
	JoinRabbet                   // rabbet (post-MVP)
	JoinDado                     // dado (post-MVP)
	JoinMortise                  // mortise and tenon (post-MVP)
	JoinDovetail                 // dovetail (post-MVP)
)

func (k JoinKind) String() string {
	switch k {
	case JoinButt:
		return "butt"
	case JoinRabbet:
		return "rabbet"
	case JoinDado:
		return "dado"
	case JoinMortise:
		return "mortise"
	case JoinDovetail:
		return "dovetail"
	default:
		return "unknown"
	}
}

// JoinData specifies how two parts are connected.
// For MVP, joints are metadata-only: they validate face contact and carry
// fastener specs but produce no geometry modifications.
type JoinData struct {
	Kind      JoinKind `json:"kind"`
	PartA     NodeID   `json:"part_a"`
	FaceA     FaceID   `json:"face_a"`
	PartB     NodeID   `json:"part_b"`
	FaceB     FaceID   `json:"face_b"`
	Clearance float64  `json:"clearance"` // gap in mm (0 = use global default)
	Params    JoinParams `json:"params"`
	Fasteners []NodeID `json:"fasteners,omitempty"`
}

func (JoinData) nodeData() {}

// JoinParams is the interface for joint-specific parameters.
type JoinParams interface {
	joinParams()
}

// ButtJoinParams holds parameters for a butt joint.
// Butt joints have no special geometry; strength comes from fasteners/adhesive.
type ButtJoinParams struct {
	GlueUp bool `json:"glue_up"`
}

func (ButtJoinParams) joinParams() {}

// ---------------------------------------------------------------------------
// Drill
// ---------------------------------------------------------------------------

// DrillData specifies a hole operation on a part.
type DrillData struct {
	TargetPart  NodeID  `json:"target_part"`
	Face        FaceID  `json:"face"`
	Position    Vec3    `json:"position"`              // on-face local coords
	Diameter    float64 `json:"diameter"`              // mm
	Depth       float64 `json:"depth"`                 // mm, 0 = through
	Countersink *float64 `json:"countersink,omitempty"` // countersink diameter
	CounterBore *float64 `json:"counterbore,omitempty"` // counterbore diameter
}

func (DrillData) nodeData() {}

// ---------------------------------------------------------------------------
// Fastener
// ---------------------------------------------------------------------------

// FastenerKind enumerates fastener types.
type FastenerKind int

const (
	FastenerScrew FastenerKind = iota
	FastenerNail
	FastenerDowelPin
	FastenerBolt
)

func (k FastenerKind) String() string {
	switch k {
	case FastenerScrew:
		return "screw"
	case FastenerNail:
		return "nail"
	case FastenerDowelPin:
		return "dowel-pin"
	case FastenerBolt:
		return "bolt"
	default:
		return "unknown"
	}
}

// FastenerData specifies a fastener placed through a join.
type FastenerData struct {
	Kind             FastenerKind `json:"kind"`
	Diameter         float64      `json:"diameter"`       // shank diameter mm
	Length           float64      `json:"length"`         // total length mm
	HeadDia          float64      `json:"head_dia"`       // head diameter mm
	Position         Vec3         `json:"position"`       // relative to the join
	JoinRef          NodeID       `json:"join_ref"`       // which join this belongs to
	PilotHoleDia     float64      `json:"pilot_hole_dia,omitempty"`
	ClearanceHoleDia float64      `json:"clearance_hole_dia,omitempty"`
}

func (FastenerData) nodeData() {}
