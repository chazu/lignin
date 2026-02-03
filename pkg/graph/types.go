package graph

import (
	"crypto/sha256"
	"fmt"
)

// NodeID is a deterministic identifier derived from source expression path.
// Stable across evaluations that preserve code structure.
type NodeID [32]byte

// ZeroID is the zero-value NodeID.
var ZeroID NodeID

// NewNodeID creates a NodeID by hashing the given path string.
func NewNodeID(path string) NodeID {
	return sha256.Sum256([]byte(path))
}

// IsZero reports whether the NodeID is the zero value.
func (id NodeID) IsZero() bool {
	return id == ZeroID
}

// Short returns a truncated hex string for display.
func (id NodeID) Short() string {
	return fmt.Sprintf("%x", id[:6])
}

func (id NodeID) String() string {
	return fmt.Sprintf("%x", id[:])
}

// ContentHash is a hash of the node's semantic content, used for change detection.
type ContentHash [32]byte

// SourceRef points back to the Lisp expression that produced a node.
type SourceRef struct {
	File   string `json:"file,omitempty"` // source file (empty for single-file MVP)
	Line   int    `json:"line"`           // 1-based line number
	Col    int    `json:"col"`            // 1-based column number
	FormID string `json:"form_id"`        // unique identifier for the S-expression
}

// Vec3 is a 3D vector for dimensions, positions, and directions.
type Vec3 struct {
	X, Y, Z float64
}

// Add returns v + other.
func (v Vec3) Add(other Vec3) Vec3 {
	return Vec3{v.X + other.X, v.Y + other.Y, v.Z + other.Z}
}

// Scale returns v * s.
func (v Vec3) Scale(s float64) Vec3 {
	return Vec3{v.X * s, v.Y * s, v.Z * s}
}

func (v Vec3) String() string {
	return fmt.Sprintf("(%.1f, %.1f, %.1f)", v.X, v.Y, v.Z)
}

// Axis represents a principal axis for grain direction and face selection.
type Axis int

const (
	AxisX Axis = iota
	AxisY
	AxisZ
)

func (a Axis) String() string {
	switch a {
	case AxisX:
		return "X"
	case AxisY:
		return "Y"
	case AxisZ:
		return "Z"
	default:
		return fmt.Sprintf("Axis(%d)", int(a))
	}
}

// FaceID identifies one of the six faces of a rectangular part.
type FaceID string

const (
	FaceTop    FaceID = "top"    // +Y
	FaceBottom FaceID = "bottom" // -Y
	FaceLeft   FaceID = "left"   // -X
	FaceRight  FaceID = "right"  // +X
	FaceFront  FaceID = "front"  // -Z
	FaceBack   FaceID = "back"   // +Z
)

// ValidFaceIDs is the set of valid face identifiers.
var ValidFaceIDs = map[FaceID]bool{
	FaceTop: true, FaceBottom: true,
	FaceLeft: true, FaceRight: true,
	FaceFront: true, FaceBack: true,
}
