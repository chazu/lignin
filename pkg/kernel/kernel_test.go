package kernel

import "testing"

// --- Mesh helper method tests ---

func TestMeshVertexCount(t *testing.T) {
	tests := []struct {
		name     string
		vertices []float32
		want     int
	}{
		{"empty", nil, 0},
		{"one vertex", []float32{1, 2, 3}, 1},
		{"four vertices", []float32{0, 0, 0, 1, 0, 0, 1, 1, 0, 0, 1, 0}, 4},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Mesh{Vertices: tt.vertices}
			if got := m.VertexCount(); got != tt.want {
				t.Errorf("VertexCount() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestMeshTriangleCount(t *testing.T) {
	tests := []struct {
		name    string
		indices []uint32
		want    int
	}{
		{"empty", nil, 0},
		{"one triangle", []uint32{0, 1, 2}, 1},
		{"two triangles", []uint32{0, 1, 2, 2, 3, 0}, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Mesh{Indices: tt.indices}
			if got := m.TriangleCount(); got != tt.want {
				t.Errorf("TriangleCount() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestMeshIsEmpty(t *testing.T) {
	t.Run("empty mesh", func(t *testing.T) {
		m := &Mesh{}
		if !m.IsEmpty() {
			t.Error("IsEmpty() = false for empty mesh, want true")
		}
	})
	t.Run("non-empty mesh", func(t *testing.T) {
		m := &Mesh{Vertices: []float32{1, 2, 3}}
		if m.IsEmpty() {
			t.Error("IsEmpty() = true for non-empty mesh, want false")
		}
	})
}

// --- Compile-time interface check with a stub kernel ---

// stubSolid is a minimal Solid implementation for testing.
type stubSolid struct {
	minBB, maxBB [3]float64
}

func (s *stubSolid) BoundingBox() (min, max [3]float64) {
	return s.minBB, s.maxBB
}

// stubKernel is a minimal Kernel implementation that proves the interface
// is satisfiable. All methods return trivial results.
type stubKernel struct{}

func (k *stubKernel) Box(x, y, z float64) Solid {
	return &stubSolid{
		minBB: [3]float64{0, 0, 0},
		maxBB: [3]float64{x, y, z},
	}
}

func (k *stubKernel) Cylinder(height, radius float64, _ int) Solid {
	return &stubSolid{
		minBB: [3]float64{-radius, -radius, 0},
		maxBB: [3]float64{radius, radius, height},
	}
}

func (k *stubKernel) Union(a, _ Solid) Solid       { return a }
func (k *stubKernel) Difference(a, _ Solid) Solid   { return a }
func (k *stubKernel) Intersection(a, _ Solid) Solid { return a }

func (k *stubKernel) Translate(s Solid, _, _, _ float64) Solid { return s }
func (k *stubKernel) Rotate(s Solid, _, _, _ float64) Solid    { return s }

func (k *stubKernel) ToMesh(_ Solid) (*Mesh, error) {
	return &Mesh{}, nil
}

// Compile-time checks that the stubs implement the interfaces.
var _ Solid = (*stubSolid)(nil)
var _ Kernel = (*stubKernel)(nil)

func TestStubKernelBoxBoundingBox(t *testing.T) {
	var k Kernel = &stubKernel{}
	s := k.Box(10, 20, 30)
	min, max := s.BoundingBox()
	if min != [3]float64{0, 0, 0} {
		t.Errorf("Box min = %v, want [0 0 0]", min)
	}
	if max != [3]float64{10, 20, 30} {
		t.Errorf("Box max = %v, want [10 20 30]", max)
	}
}

func TestStubKernelToMesh(t *testing.T) {
	var k Kernel = &stubKernel{}
	s := k.Box(1, 1, 1)
	m, err := k.ToMesh(s)
	if err != nil {
		t.Fatalf("ToMesh() error = %v", err)
	}
	if m == nil {
		t.Fatal("ToMesh() returned nil mesh")
	}
	if !m.IsEmpty() {
		t.Error("stub ToMesh() should return empty mesh")
	}
}
