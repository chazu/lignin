package kernel

// Mesh is a triangle mesh suitable for rendering.
// All arrays are flat: vertices has 3 floats per vertex (x,y,z),
// normals has 3 floats per vertex, indices has 3 uint32s per triangle.
type Mesh struct {
	Vertices []float32 `json:"vertices"` // [x0,y0,z0, x1,y1,z1, ...]
	Normals  []float32 `json:"normals"`  // [nx0,ny0,nz0, ...]
	Indices  []uint32  `json:"indices"`  // [i0,i1,i2, ...] triangles
	PartName string    `json:"partName"` // which design graph part this came from
}

// VertexCount returns the number of vertices.
func (m *Mesh) VertexCount() int {
	return len(m.Vertices) / 3
}

// TriangleCount returns the number of triangles.
func (m *Mesh) TriangleCount() int {
	return len(m.Indices) / 3
}

// IsEmpty returns true if the mesh has no geometry.
func (m *Mesh) IsEmpty() bool {
	return len(m.Vertices) == 0
}
