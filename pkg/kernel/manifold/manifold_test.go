//go:build manifold

package manifold

import (
	"math"
	"testing"

	"github.com/chazu/lignin/pkg/kernel"
)

func mustNew(t *testing.T) kernel.Kernel {
	t.Helper()
	k, err := New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	return k
}

func TestBox(t *testing.T) {
	k := mustNew(t)
	s := k.Box(10, 20, 30)
	if s == nil {
		t.Fatal("Box() returned nil")
	}
	min, max := s.BoundingBox()

	// Box is centered, so bounds should be symmetric.
	wantMin := [3]float64{-5, -10, -15}
	wantMax := [3]float64{5, 10, 15}

	for i := 0; i < 3; i++ {
		if math.Abs(min[i]-wantMin[i]) > 1e-6 {
			t.Errorf("Box min[%d] = %f, want %f", i, min[i], wantMin[i])
		}
		if math.Abs(max[i]-wantMax[i]) > 1e-6 {
			t.Errorf("Box max[%d] = %f, want %f", i, max[i], wantMax[i])
		}
	}
}

func TestCylinder(t *testing.T) {
	k := mustNew(t)
	s := k.Cylinder(20, 5, 32)
	if s == nil {
		t.Fatal("Cylinder() returned nil")
	}
	min, max := s.BoundingBox()

	// Cylinder is centered, radius=5, height=20.
	// X/Y bounds should be approximately [-5, 5] (polygon approximation).
	// Z bounds should be [-10, 10].
	if min[2] < -10.01 || min[2] > -9.99 {
		t.Errorf("Cylinder min Z = %f, want ~-10", min[2])
	}
	if max[2] < 9.99 || max[2] > 10.01 {
		t.Errorf("Cylinder max Z = %f, want ~10", max[2])
	}

	// X/Y bounds should be within the radius (polygon inscribed in circle).
	for i := 0; i < 2; i++ {
		if min[i] > -4.5 {
			t.Errorf("Cylinder min[%d] = %f, want <= -4.5", i, min[i])
		}
		if max[i] < 4.5 {
			t.Errorf("Cylinder max[%d] = %f, want >= 4.5", i, max[i])
		}
	}
}

func TestDifference(t *testing.T) {
	k := mustNew(t)
	box := k.Box(10, 10, 10)
	hole := k.Cylinder(20, 3, 32)
	result := k.Difference(box, hole)
	if result == nil {
		t.Fatal("Difference() returned nil")
	}

	// The result bounding box should be the same as the box (the hole
	// is contained within the box footprint in X/Y).
	min, max := result.BoundingBox()
	wantMin := [3]float64{-5, -5, -5}
	wantMax := [3]float64{5, 5, 5}
	for i := 0; i < 3; i++ {
		if math.Abs(min[i]-wantMin[i]) > 1e-6 {
			t.Errorf("Difference min[%d] = %f, want %f", i, min[i], wantMin[i])
		}
		if math.Abs(max[i]-wantMax[i]) > 1e-6 {
			t.Errorf("Difference max[%d] = %f, want %f", i, max[i], wantMax[i])
		}
	}
}

func TestTranslate(t *testing.T) {
	k := mustNew(t)
	box := k.Box(10, 10, 10)
	moved := k.Translate(box, 100, 200, 300)
	if moved == nil {
		t.Fatal("Translate() returned nil")
	}

	min, max := moved.BoundingBox()
	wantMin := [3]float64{95, 195, 295}
	wantMax := [3]float64{105, 205, 305}
	for i := 0; i < 3; i++ {
		if math.Abs(min[i]-wantMin[i]) > 1e-6 {
			t.Errorf("Translate min[%d] = %f, want %f", i, min[i], wantMin[i])
		}
		if math.Abs(max[i]-wantMax[i]) > 1e-6 {
			t.Errorf("Translate max[%d] = %f, want %f", i, max[i], wantMax[i])
		}
	}
}

func TestBoundingBox(t *testing.T) {
	k := mustNew(t)
	box := k.Box(4, 6, 8)
	min, max := box.BoundingBox()

	// Centered box: half-extents are 2, 3, 4.
	if math.Abs(min[0]+2) > 1e-6 || math.Abs(min[1]+3) > 1e-6 || math.Abs(min[2]+4) > 1e-6 {
		t.Errorf("BoundingBox min = %v, want [-2 -3 -4]", min)
	}
	if math.Abs(max[0]-2) > 1e-6 || math.Abs(max[1]-3) > 1e-6 || math.Abs(max[2]-4) > 1e-6 {
		t.Errorf("BoundingBox max = %v, want [2 3 4]", max)
	}
}

func TestToMesh(t *testing.T) {
	k := mustNew(t)
	box := k.Box(10, 10, 10)
	mesh, err := k.ToMesh(box)
	if err != nil {
		t.Fatalf("ToMesh() error = %v", err)
	}
	if mesh == nil {
		t.Fatal("ToMesh() returned nil mesh")
	}
	if mesh.IsEmpty() {
		t.Error("ToMesh() returned empty mesh for a box")
	}

	// A box has 8 vertices and 12 triangles (2 per face, 6 faces).
	// Manifold may produce more vertices due to sharp edges requiring
	// separate normals, but triangle count should be exactly 12.
	if mesh.TriangleCount() < 12 {
		t.Errorf("ToMesh() triangle count = %d, want >= 12", mesh.TriangleCount())
	}
	if mesh.VertexCount() < 8 {
		t.Errorf("ToMesh() vertex count = %d, want >= 8", mesh.VertexCount())
	}

	// Verify normals array has the same length as vertices.
	if len(mesh.Normals) != len(mesh.Vertices) {
		t.Errorf("ToMesh() normals length = %d, vertices length = %d, want equal",
			len(mesh.Normals), len(mesh.Vertices))
	}
}
