//go:build manifold

// Package manifold provides a CGo-based geometry kernel binding to the
// Manifold library (https://github.com/elalish/manifold). Manifold provides
// guaranteed-manifold mesh boolean operations with face identity tracking.
//
// This package requires the Manifold C library (manifoldc) to be installed.
// Build with: go build -tags=manifold
//
// See the Makefile in this directory for instructions on building manifoldc
// from source.
package manifold

/*
#cgo CFLAGS: -I/usr/local/include
#cgo LDFLAGS: -L/usr/local/lib -lmanifoldc

#include <stdlib.h>
#include <manifold/manifoldc.h>
*/
import "C"

import (
	"fmt"
	"runtime"
	"unsafe"

	"github.com/chazu/lignin/pkg/kernel"
)

// Compile-time interface checks.
var _ kernel.Kernel = (*ManifoldKernel)(nil)
var _ kernel.Solid = (*manifoldSolid)(nil)

// manifoldSolid wraps a C ManifoldManifold pointer and implements kernel.Solid.
type manifoldSolid struct {
	ptr *C.ManifoldManifold
}

// BoundingBox returns the axis-aligned bounding box of the solid.
func (s *manifoldSolid) BoundingBox() (min, max [3]float64) {
	alloc := C.manifold_alloc_box()
	bbox := C.manifold_bounding_box(alloc, s.ptr)
	defer C.manifold_delete_box(bbox)

	min[0] = float64(C.manifold_box_min_x(bbox))
	min[1] = float64(C.manifold_box_min_y(bbox))
	min[2] = float64(C.manifold_box_min_z(bbox))
	max[0] = float64(C.manifold_box_max_x(bbox))
	max[1] = float64(C.manifold_box_max_y(bbox))
	max[2] = float64(C.manifold_box_max_z(bbox))
	return min, max
}

// newSolid wraps a C ManifoldManifold pointer with Go-side finalizer
// for automatic memory management.
func newSolid(ptr *C.ManifoldManifold) *manifoldSolid {
	s := &manifoldSolid{ptr: ptr}
	runtime.SetFinalizer(s, func(s *manifoldSolid) {
		if s.ptr != nil {
			C.manifold_delete_manifold(s.ptr)
			s.ptr = nil
		}
	})
	return s
}

// ManifoldKernel implements kernel.Kernel using the Manifold C library.
type ManifoldKernel struct{}

// New creates a new ManifoldKernel. Returns an error if the Manifold
// C library cannot be initialized.
func New() (kernel.Kernel, error) {
	return &ManifoldKernel{}, nil
}

// Box creates an axis-aligned box with the given dimensions.
// The box is centered at the origin.
func (k *ManifoldKernel) Box(x, y, z float64) kernel.Solid {
	alloc := C.manifold_alloc_manifold()
	ptr := C.manifold_cube(alloc,
		C.double(x), C.double(y), C.double(z),
		C.int(1), // center=true
	)
	return newSolid(ptr)
}

// Cylinder creates a cylinder along the Z axis with the given height,
// radius, and number of circular segments. The cylinder is centered
// at the origin.
func (k *ManifoldKernel) Cylinder(height, radius float64, segments int) kernel.Solid {
	alloc := C.manifold_alloc_manifold()
	ptr := C.manifold_cylinder(alloc,
		C.double(height),
		C.double(radius), // radius_low
		C.double(radius), // radius_high (same = not tapered)
		C.int(segments),
		C.int(1), // center=true
	)
	return newSolid(ptr)
}

// Union returns the boolean union of two solids.
func (k *ManifoldKernel) Union(a, b kernel.Solid) kernel.Solid {
	sa := a.(*manifoldSolid)
	sb := b.(*manifoldSolid)
	alloc := C.manifold_alloc_manifold()
	ptr := C.manifold_union(alloc, sa.ptr, sb.ptr)
	return newSolid(ptr)
}

// Difference returns the boolean difference (a minus b).
func (k *ManifoldKernel) Difference(a, b kernel.Solid) kernel.Solid {
	sa := a.(*manifoldSolid)
	sb := b.(*manifoldSolid)
	alloc := C.manifold_alloc_manifold()
	ptr := C.manifold_difference(alloc, sa.ptr, sb.ptr)
	return newSolid(ptr)
}

// Intersection returns the boolean intersection of two solids.
func (k *ManifoldKernel) Intersection(a, b kernel.Solid) kernel.Solid {
	sa := a.(*manifoldSolid)
	sb := b.(*manifoldSolid)
	alloc := C.manifold_alloc_manifold()
	ptr := C.manifold_intersection(alloc, sa.ptr, sb.ptr)
	return newSolid(ptr)
}

// Translate moves the solid by (x, y, z).
func (k *ManifoldKernel) Translate(s kernel.Solid, x, y, z float64) kernel.Solid {
	ms := s.(*manifoldSolid)
	alloc := C.manifold_alloc_manifold()
	ptr := C.manifold_translate(alloc, ms.ptr,
		C.double(x), C.double(y), C.double(z),
	)
	return newSolid(ptr)
}

// Rotate rotates the solid by Euler angles (in degrees) around the X, Y, Z axes.
func (k *ManifoldKernel) Rotate(s kernel.Solid, x, y, z float64) kernel.Solid {
	ms := s.(*manifoldSolid)
	alloc := C.manifold_alloc_manifold()
	ptr := C.manifold_rotate(alloc, ms.ptr,
		C.double(x), C.double(y), C.double(z),
	)
	return newSolid(ptr)
}

// ToMesh extracts a triangle mesh from the solid using Manifold's MeshGL
// format. Vertex positions and normals are interleaved in MeshGL; this
// method separates them into the kernel.Mesh flat-array layout.
func (k *ManifoldKernel) ToMesh(s kernel.Solid) (*kernel.Mesh, error) {
	ms := s.(*manifoldSolid)

	// Get MeshGL from the manifold.
	meshAlloc := C.manifold_alloc_meshgl()
	meshGL := C.manifold_get_meshgl(meshAlloc, ms.ptr)
	defer C.manifold_delete_meshgl(meshGL)

	numVert := int(C.manifold_meshgl_num_vert(meshGL))
	numTri := int(C.manifold_meshgl_num_tri(meshGL))

	if numVert == 0 || numTri == 0 {
		return &kernel.Mesh{}, nil
	}

	// MeshGL stores vertex properties in a flat float array.
	// The default layout has numProp properties per vertex.
	// The first 3 are always position (x, y, z).
	// If normals are present, they follow at indices 3, 4, 5.
	numProp := int(C.manifold_meshgl_num_prop(meshGL))

	// Extract the vertex property data.
	propLen := numVert * numProp
	propData := make([]float32, propLen)
	C.manifold_meshgl_vert_properties(
		(*C.float)(unsafe.Pointer(&propData[0])),
		meshGL,
	)

	// Extract triangle indices.
	triLen := numTri * 3
	indices := make([]uint32, triLen)
	C.manifold_meshgl_tri_verts(
		(*C.uint32_t)(unsafe.Pointer(&indices[0])),
		meshGL,
	)

	// Separate positions and normals from the interleaved property array.
	vertices := make([]float32, numVert*3)
	var normals []float32
	hasNormals := numProp >= 6
	if hasNormals {
		normals = make([]float32, numVert*3)
	}

	for i := 0; i < numVert; i++ {
		base := i * numProp
		// Positions are always at indices 0, 1, 2.
		vertices[i*3+0] = propData[base+0]
		vertices[i*3+1] = propData[base+1]
		vertices[i*3+2] = propData[base+2]
		// Normals at indices 3, 4, 5 if present.
		if hasNormals {
			normals[i*3+0] = propData[base+3]
			normals[i*3+1] = propData[base+4]
			normals[i*3+2] = propData[base+5]
		}
	}

	if !hasNormals {
		// Compute flat normals from triangle faces as a fallback.
		normals = computeFlatNormals(vertices, indices)
	}

	mesh := &kernel.Mesh{
		Vertices: vertices,
		Normals:  normals,
		Indices:  indices,
	}

	if mesh.VertexCount() != numVert {
		return nil, fmt.Errorf("manifold: vertex count mismatch: got %d, expected %d",
			mesh.VertexCount(), numVert)
	}

	return mesh, nil
}

// computeFlatNormals generates per-vertex normals by averaging the face normals
// of all triangles incident on each vertex. This is a fallback when MeshGL
// does not include normals in the vertex properties.
func computeFlatNormals(vertices []float32, indices []uint32) []float32 {
	numVerts := len(vertices) / 3
	normals := make([]float32, numVerts*3)

	numTris := len(indices) / 3
	for t := 0; t < numTris; t++ {
		i0 := indices[t*3+0]
		i1 := indices[t*3+1]
		i2 := indices[t*3+2]

		// Triangle vertex positions.
		ax, ay, az := float64(vertices[i0*3]), float64(vertices[i0*3+1]), float64(vertices[i0*3+2])
		bx, by, bz := float64(vertices[i1*3]), float64(vertices[i1*3+1]), float64(vertices[i1*3+2])
		cx, cy, cz := float64(vertices[i2*3]), float64(vertices[i2*3+1]), float64(vertices[i2*3+2])

		// Edge vectors.
		e1x, e1y, e1z := bx-ax, by-ay, bz-az
		e2x, e2y, e2z := cx-ax, cy-ay, cz-az

		// Cross product (unnormalized face normal).
		nx := float32(e1y*e2z - e1z*e2y)
		ny := float32(e1z*e2x - e1x*e2z)
		nz := float32(e1x*e2y - e1y*e2x)

		// Accumulate into each vertex of this triangle.
		for _, idx := range []uint32{i0, i1, i2} {
			normals[idx*3+0] += nx
			normals[idx*3+1] += ny
			normals[idx*3+2] += nz
		}
	}

	// Normalize.
	for i := 0; i < numVerts; i++ {
		nx := float64(normals[i*3+0])
		ny := float64(normals[i*3+1])
		nz := float64(normals[i*3+2])
		length := sqrt64(nx*nx + ny*ny + nz*nz)
		if length > 1e-12 {
			normals[i*3+0] = float32(nx / length)
			normals[i*3+1] = float32(ny / length)
			normals[i*3+2] = float32(nz / length)
		}
	}

	return normals
}

// sqrt64 computes the square root without importing math to keep the
// dependency footprint minimal. Uses Newton's method.
func sqrt64(x float64) float64 {
	if x <= 0 {
		return 0
	}
	z := x
	for i := 0; i < 20; i++ {
		z = (z + x/z) / 2
	}
	return z
}
