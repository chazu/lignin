// Package sdfx implements the kernel.Kernel interface using the
// github.com/deadsy/sdfx SDF-based CAD library.
package sdfx

import (
	"fmt"
	"math"

	"github.com/chazu/lignin/pkg/kernel"
	"github.com/deadsy/sdfx/render"
	"github.com/deadsy/sdfx/sdf"
	v3 "github.com/deadsy/sdfx/vec/v3"
)

// Compile-time interface check.
var _ kernel.Kernel = (*SdfxKernel)(nil)

// defaultMeshCells controls marching cubes tessellation resolution.
const defaultMeshCells = 200

// sdfxSolid wraps an sdf.SDF3 to implement kernel.Solid.
type sdfxSolid struct {
	s sdf.SDF3
}

// BoundingBox returns the axis-aligned bounding box.
func (s *sdfxSolid) BoundingBox() (min, max [3]float64) {
	bb := s.s.BoundingBox()
	min = [3]float64{bb.Min.X, bb.Min.Y, bb.Min.Z}
	max = [3]float64{bb.Max.X, bb.Max.Y, bb.Max.Z}
	return min, max
}

// SdfxKernel implements kernel.Kernel using sdfx.
type SdfxKernel struct{}

// New returns a new SdfxKernel.
func New() *SdfxKernel {
	return &SdfxKernel{}
}

// unwrap extracts the underlying sdf.SDF3 from a kernel.Solid.
func unwrap(s kernel.Solid) sdf.SDF3 {
	return s.(*sdfxSolid).s
}

// wrap creates a kernel.Solid from an sdf.SDF3.
func wrap(s sdf.SDF3) kernel.Solid {
	return &sdfxSolid{s: s}
}

// Box creates a box with the given dimensions. The resulting solid has its
// minimum corner at the origin (0,0,0) so that placement translations work
// intuitively — (place :at (vec3 10 0 0)) puts the board's corner at x=10.
// sdf.Box3D centers the box at the origin, so we translate by half-dimensions.
func (k *SdfxKernel) Box(x, y, z float64) kernel.Solid {
	s, err := sdf.Box3D(v3.Vec{X: x, Y: y, Z: z}, 0)
	if err != nil {
		panic(fmt.Sprintf("sdfx.Box3D: %v", err))
	}
	// Shift from center-origin to min-corner-origin.
	m := sdf.Translate3d(v3.Vec{X: x / 2, Y: y / 2, Z: z / 2})
	return wrap(sdf.Transform3D(s, m))
}

// Cylinder creates a cylinder with the given height and radius.
// The segments parameter is ignored since SDF represents smooth surfaces.
func (k *SdfxKernel) Cylinder(height, radius float64, segments int) kernel.Solid {
	s, err := sdf.Cylinder3D(height, radius, 0)
	if err != nil {
		panic(fmt.Sprintf("sdfx.Cylinder3D: %v", err))
	}
	return wrap(s)
}

// Union returns the union of two solids.
func (k *SdfxKernel) Union(a, b kernel.Solid) kernel.Solid {
	return wrap(sdf.Union3D(unwrap(a), unwrap(b)))
}

// Difference returns the difference a - b.
func (k *SdfxKernel) Difference(a, b kernel.Solid) kernel.Solid {
	return wrap(sdf.Difference3D(unwrap(a), unwrap(b)))
}

// Intersection returns the intersection of two solids.
func (k *SdfxKernel) Intersection(a, b kernel.Solid) kernel.Solid {
	return wrap(sdf.Intersect3D(unwrap(a), unwrap(b)))
}

// Translate moves a solid by (x, y, z).
func (k *SdfxKernel) Translate(s kernel.Solid, x, y, z float64) kernel.Solid {
	m := sdf.Translate3d(v3.Vec{X: x, Y: y, Z: z})
	return wrap(sdf.Transform3D(unwrap(s), m))
}

// Rotate rotates a solid by Euler angles (degrees) around X, Y, Z axes.
func (k *SdfxKernel) Rotate(s kernel.Solid, x, y, z float64) kernel.Solid {
	xRad := x * math.Pi / 180.0
	yRad := y * math.Pi / 180.0
	zRad := z * math.Pi / 180.0

	m := sdf.RotateZ(zRad).Mul(sdf.RotateY(yRad)).Mul(sdf.RotateX(xRad))
	return wrap(sdf.Transform3D(unwrap(s), m))
}

// ToMesh converts a solid to a triangle mesh using marching cubes.
func (k *SdfxKernel) ToMesh(s kernel.Solid) (*kernel.Mesh, error) {
	sdf3 := unwrap(s)

	renderer := render.NewMarchingCubesUniform(defaultMeshCells)
	triangles := render.ToTriangles(sdf3, renderer)

	numTri := len(triangles)
	numVerts := numTri * 3

	vertices := make([]float32, 0, numVerts*3)
	normals := make([]float32, 0, numVerts*3)
	indices := make([]uint32, 0, numVerts)

	for i, tri := range triangles {
		// Compute face normal.
		n := tri.Normal()
		nx := float32(n.X)
		ny := float32(n.Y)
		nz := float32(n.Z)

		for j := 0; j < 3; j++ {
			v := tri[j]
			vertices = append(vertices, float32(v.X), float32(v.Y), float32(v.Z))
			normals = append(normals, nx, ny, nz)
			indices = append(indices, uint32(i*3+j))
		}
	}

	return &kernel.Mesh{
		Vertices: vertices,
		Normals:  normals,
		Indices:  indices,
	}, nil
}
