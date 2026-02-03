// Package kernel defines the abstract geometry kernel interface.
// Implementations (sdfx, manifold) provide solid modeling and
// boolean operations behind this interface. The kernel abstraction
// allows swapping backends without changing the rest of the system.
package kernel

// Solid is an opaque handle to a geometry kernel solid.
// Implementations wrap their internal representation.
type Solid interface {
	// BoundingBox returns the axis-aligned bounding box.
	BoundingBox() (min, max [3]float64)
}

// Kernel is the abstract geometry kernel interface.
// Implementations (sdfx, manifold) provide solid modeling behind this interface.
type Kernel interface {
	// Primitives
	Box(x, y, z float64) Solid
	Cylinder(height, radius float64, segments int) Solid

	// Boolean operations
	Union(a, b Solid) Solid
	Difference(a, b Solid) Solid
	Intersection(a, b Solid) Solid

	// Transforms
	Translate(s Solid, x, y, z float64) Solid
	Rotate(s Solid, x, y, z float64) Solid // Euler angles in degrees

	// Mesh output
	ToMesh(s Solid) (*Mesh, error)
}
