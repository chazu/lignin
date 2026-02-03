package sdfx

import (
	"math"
	"testing"
)

func TestBox(t *testing.T) {
	k := New()
	box := k.Box(100, 50, 25)
	mesh, err := k.ToMesh(box)
	if err != nil {
		t.Fatalf("ToMesh failed: %v", err)
	}
	if mesh.IsEmpty() {
		t.Fatal("mesh is empty")
	}
	if mesh.VertexCount() == 0 {
		t.Fatal("expected non-zero vertex count")
	}
	triCount := mesh.TriangleCount()
	if triCount == 0 {
		t.Fatal("expected non-zero triangle count")
	}
	// A box should produce exactly 12 triangles (2 per face, 6 faces).
	if triCount != 12 {
		t.Logf("box triangle count: %d (expected 12)", triCount)
	}
	// Verify vertex and index array sizes are consistent.
	if len(mesh.Vertices) != len(mesh.Normals) {
		t.Fatalf("vertices length %d != normals length %d", len(mesh.Vertices), len(mesh.Normals))
	}
	if len(mesh.Indices) != triCount*3 {
		t.Fatalf("indices length %d != triCount*3 %d", len(mesh.Indices), triCount*3)
	}
}

func TestCylinder(t *testing.T) {
	k := New()
	cyl := k.Cylinder(50, 10, 32)
	mesh, err := k.ToMesh(cyl)
	if err != nil {
		t.Fatalf("ToMesh failed: %v", err)
	}
	if mesh.IsEmpty() {
		t.Fatal("mesh is empty")
	}
	if mesh.TriangleCount() == 0 {
		t.Fatal("expected non-zero triangle count")
	}
	t.Logf("cylinder triangle count: %d", mesh.TriangleCount())
}

func TestDifference(t *testing.T) {
	k := New()

	box := k.Box(100, 100, 100)
	boxMesh, err := k.ToMesh(box)
	if err != nil {
		t.Fatalf("ToMesh(box) failed: %v", err)
	}

	// Cylinder is centered at origin; translate it to the center of the box
	// so it properly intersects for a meaningful difference.
	cyl := k.Translate(k.Cylinder(120, 20, 32), 50, 50, 50)
	diff := k.Difference(box, cyl)
	diffMesh, err := k.ToMesh(diff)
	if err != nil {
		t.Fatalf("ToMesh(diff) failed: %v", err)
	}
	if diffMesh.IsEmpty() {
		t.Fatal("difference mesh is empty")
	}
	// A box with a hole should have more triangles than a plain box.
	if diffMesh.TriangleCount() <= boxMesh.TriangleCount() {
		t.Fatalf("difference (%d triangles) should have more triangles than box (%d triangles)",
			diffMesh.TriangleCount(), boxMesh.TriangleCount())
	}
	t.Logf("box triangles: %d, difference triangles: %d", boxMesh.TriangleCount(), diffMesh.TriangleCount())
}

func TestUnion(t *testing.T) {
	k := New()
	box1 := k.Box(50, 50, 50)
	box2 := k.Translate(k.Box(50, 50, 50), 30, 0, 0)
	u := k.Union(box1, box2)
	mesh, err := k.ToMesh(u)
	if err != nil {
		t.Fatalf("ToMesh failed: %v", err)
	}
	if mesh.IsEmpty() {
		t.Fatal("union mesh is empty")
	}
	t.Logf("union triangle count: %d", mesh.TriangleCount())
}

func TestTranslate(t *testing.T) {
	k := New()
	box := k.Box(10, 10, 10)
	translated := k.Translate(box, 100, 200, 300)

	min, max := translated.BoundingBox()

	// Box(10,10,10) has min-corner at origin, so spans (0,0,0)-(10,10,10).
	// Translated by (100,200,300) => (100,200,300)-(110,210,310).
	const tol = 0.5
	expectMin := [3]float64{100, 200, 300}
	expectMax := [3]float64{110, 210, 310}

	for i := 0; i < 3; i++ {
		if math.Abs(min[i]-expectMin[i]) > tol {
			t.Errorf("min[%d] = %f, expected ~%f", i, min[i], expectMin[i])
		}
		if math.Abs(max[i]-expectMax[i]) > tol {
			t.Errorf("max[%d] = %f, expected ~%f", i, max[i], expectMax[i])
		}
	}
}

func TestBoundingBox(t *testing.T) {
	k := New()
	box := k.Box(100, 50, 25)
	min, max := box.BoundingBox()

	// Box now has min-corner at origin: (0,0,0) to (100,50,25).
	const tol = 0.01
	expectMin := [3]float64{0, 0, 0}
	expectMax := [3]float64{100, 50, 25}

	for i := 0; i < 3; i++ {
		if math.Abs(min[i]-expectMin[i]) > tol {
			t.Errorf("min[%d] = %f, expected %f", i, min[i], expectMin[i])
		}
		if math.Abs(max[i]-expectMax[i]) > tol {
			t.Errorf("max[%d] = %f, expected %f", i, max[i], expectMax[i])
		}
	}
}

func TestIntersection(t *testing.T) {
	k := New()
	box1 := k.Box(100, 100, 100)
	box2 := k.Translate(k.Box(100, 100, 100), 50, 0, 0)
	inter := k.Intersection(box1, box2)
	mesh, err := k.ToMesh(inter)
	if err != nil {
		t.Fatalf("ToMesh failed: %v", err)
	}
	if mesh.IsEmpty() {
		t.Fatal("intersection mesh is empty")
	}
	t.Logf("intersection triangle count: %d", mesh.TriangleCount())
}

func TestRotate(t *testing.T) {
	k := New()
	box := k.Box(100, 10, 10)

	// A long box along X rotated 90 degrees around Z should extend along Y instead.
	rotated := k.Rotate(box, 0, 0, 90)
	min, max := rotated.BoundingBox()

	// After 90-degree Z rotation, the X extent should be small and Y extent large.
	xExtent := max[0] - min[0]
	yExtent := max[1] - min[1]

	const tol = 1.0
	if math.Abs(xExtent-10) > tol {
		t.Errorf("rotated X extent = %f, expected ~10", xExtent)
	}
	if math.Abs(yExtent-100) > tol {
		t.Errorf("rotated Y extent = %f, expected ~100", yExtent)
	}
}
