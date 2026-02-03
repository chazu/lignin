package tessellate_test

import (
	"testing"

	"github.com/chazu/lignin/pkg/graph"
	"github.com/chazu/lignin/pkg/kernel"
	"github.com/chazu/lignin/pkg/kernel/sdfx"
	"github.com/chazu/lignin/pkg/tessellate"
)

// newKernel returns a fresh sdfx kernel for testing.
func newKernel() kernel.Kernel {
	return sdfx.New()
}

// makeBoard creates a board primitive node with the given name and dimensions.
func makeBoard(name string, x, y, z float64) *graph.Node {
	id := graph.NewNodeID(name)
	return &graph.Node{
		ID:   id,
		Kind: graph.NodePrimitive,
		Name: name,
		Data: graph.BoardData{
			PrimKind:   graph.PrimBoard,
			Dimensions: graph.Vec3{X: x, Y: y, Z: z},
			Grain:      graph.AxisX,
		},
	}
}

// makePlaceTransform creates a transform node with a translation.
func makePlaceTransform(name string, tx, ty, tz float64, children ...graph.NodeID) *graph.Node {
	id := graph.NewNodeID(name)
	t := graph.Vec3{X: tx, Y: ty, Z: tz}
	return &graph.Node{
		ID:       id,
		Kind:     graph.NodeTransform,
		Name:     name,
		Children: children,
		Data: graph.TransformData{
			Translation: &t,
		},
	}
}

// makeGroup creates a group node with children.
func makeGroup(name string, children ...graph.NodeID) *graph.Node {
	id := graph.NewNodeID(name)
	return &graph.Node{
		ID:       id,
		Kind:     graph.NodeGroup,
		Name:     name,
		Children: children,
		Data:     graph.GroupData{Description: name},
	}
}

// makeJoin creates a butt join node.
func makeJoin(name string, partA, partB graph.NodeID) *graph.Node {
	id := graph.NewNodeID(name)
	return &graph.Node{
		ID:   id,
		Kind: graph.NodeJoin,
		Name: name,
		Data: graph.JoinData{
			Kind:  graph.JoinButt,
			PartA: partA,
			PartB: partB,
			Params: graph.ButtJoinParams{
				GlueUp: true,
			},
		},
	}
}

func TestSingleBox(t *testing.T) {
	k := newKernel()
	g := graph.New()

	board := makeBoard("shelf", 600, 300, 18)
	g.AddNode(board)
	g.AddRoot(board.ID)

	meshes, err := tessellate.Tessellate(g, k)
	if err != nil {
		t.Fatalf("Tessellate failed: %v", err)
	}
	if len(meshes) != 1 {
		t.Fatalf("expected 1 mesh, got %d", len(meshes))
	}

	m := meshes[0]
	if m.IsEmpty() {
		t.Fatal("mesh should not be empty")
	}
	if m.PartName != "shelf" {
		t.Errorf("expected PartName %q, got %q", "shelf", m.PartName)
	}
	if m.VertexCount() == 0 {
		t.Error("mesh should have vertices")
	}
	if m.TriangleCount() == 0 {
		t.Error("mesh should have triangles")
	}
}

func TestTwoParts(t *testing.T) {
	k := newKernel()
	g := graph.New()

	side := makeBoard("side-panel", 400, 300, 18)
	top := makeBoard("top-panel", 600, 300, 18)
	g.AddNode(side)
	g.AddNode(top)
	g.AddRoot(side.ID)
	g.AddRoot(top.ID)

	meshes, err := tessellate.Tessellate(g, k)
	if err != nil {
		t.Fatalf("Tessellate failed: %v", err)
	}
	if len(meshes) != 2 {
		t.Fatalf("expected 2 meshes, got %d", len(meshes))
	}

	names := map[string]bool{}
	for _, m := range meshes {
		if m.IsEmpty() {
			t.Error("mesh should not be empty")
		}
		names[m.PartName] = true
	}

	if !names["side-panel"] {
		t.Error("missing mesh for side-panel")
	}
	if !names["top-panel"] {
		t.Error("missing mesh for top-panel")
	}
}

func TestPartWithTransform(t *testing.T) {
	k := newKernel()
	g := graph.New()

	board := makeBoard("shelf", 100, 50, 10)
	g.AddNode(board)

	// Place the board at an offset of (200, 100, 50).
	place := makePlaceTransform("place-shelf", 200, 100, 50, board.ID)
	g.AddNode(place)
	g.AddRoot(place.ID)

	meshes, err := tessellate.Tessellate(g, k)
	if err != nil {
		t.Fatalf("Tessellate failed: %v", err)
	}
	if len(meshes) != 1 {
		t.Fatalf("expected 1 mesh, got %d", len(meshes))
	}

	m := meshes[0]
	if m.IsEmpty() {
		t.Fatal("mesh should not be empty")
	}
	if m.PartName != "shelf" {
		t.Errorf("expected PartName %q, got %q", "shelf", m.PartName)
	}

	// Verify that mesh vertices are offset. Box has min-corner at origin,
	// so a 100x50x10 board placed at (200,100,50) spans (200,100,50)-(300,150,60).
	// Centroid should be near (250, 125, 55).
	var cx, cy, cz float64
	n := m.VertexCount()
	for i := 0; i < n; i++ {
		cx += float64(m.Vertices[i*3])
		cy += float64(m.Vertices[i*3+1])
		cz += float64(m.Vertices[i*3+2])
	}
	cx /= float64(n)
	cy /= float64(n)
	cz /= float64(n)

	// Use a generous tolerance since marching cubes is approximate.
	const tol = 20.0
	if abs(cx-250) > tol {
		t.Errorf("centroid X = %.1f, expected near 250", cx)
	}
	if abs(cy-125) > tol {
		t.Errorf("centroid Y = %.1f, expected near 125", cy)
	}
	if abs(cz-55) > tol {
		t.Errorf("centroid Z = %.1f, expected near 55", cz)
	}
}

func TestAssembly(t *testing.T) {
	k := newKernel()
	g := graph.New()

	left := makeBoard("left-side", 400, 300, 18)
	right := makeBoard("right-side", 400, 300, 18)
	top := makeBoard("top", 600, 300, 18)
	g.AddNode(left)
	g.AddNode(right)
	g.AddNode(top)

	placeLeft := makePlaceTransform("place-left", 0, 0, 0, left.ID)
	placeRight := makePlaceTransform("place-right", 582, 0, 0, right.ID)
	placeTop := makePlaceTransform("place-top", 300, 400, 0, top.ID)
	g.AddNode(placeLeft)
	g.AddNode(placeRight)
	g.AddNode(placeTop)

	assembly := makeGroup("bookshelf", placeLeft.ID, placeRight.ID, placeTop.ID)
	g.AddNode(assembly)
	g.AddRoot(assembly.ID)

	meshes, err := tessellate.Tessellate(g, k)
	if err != nil {
		t.Fatalf("Tessellate failed: %v", err)
	}
	if len(meshes) != 3 {
		t.Fatalf("expected 3 meshes, got %d", len(meshes))
	}

	names := map[string]bool{}
	for _, m := range meshes {
		if m.IsEmpty() {
			t.Errorf("mesh %q should not be empty", m.PartName)
		}
		names[m.PartName] = true
	}

	for _, want := range []string{"left-side", "right-side", "top"} {
		if !names[want] {
			t.Errorf("missing mesh for %q", want)
		}
	}
}

func TestEmptyGraph(t *testing.T) {
	k := newKernel()
	g := graph.New()

	meshes, err := tessellate.Tessellate(g, k)
	if err != nil {
		t.Fatalf("Tessellate failed: %v", err)
	}
	if len(meshes) != 0 {
		t.Fatalf("expected 0 meshes, got %d", len(meshes))
	}
}

func TestJoinIgnored(t *testing.T) {
	k := newKernel()
	g := graph.New()

	sideA := makeBoard("side-a", 400, 300, 18)
	sideB := makeBoard("side-b", 600, 300, 18)
	g.AddNode(sideA)
	g.AddNode(sideB)

	joint := makeJoin("butt-joint-1", sideA.ID, sideB.ID)
	g.AddNode(joint)

	// All three are roots: two parts and one joint.
	g.AddRoot(sideA.ID)
	g.AddRoot(sideB.ID)
	g.AddRoot(joint.ID)

	meshes, err := tessellate.Tessellate(g, k)
	if err != nil {
		t.Fatalf("Tessellate failed: %v", err)
	}

	// Only 2 meshes from the parts; the joint produces none.
	if len(meshes) != 2 {
		t.Fatalf("expected 2 meshes, got %d", len(meshes))
	}

	names := map[string]bool{}
	for _, m := range meshes {
		names[m.PartName] = true
	}
	if !names["side-a"] {
		t.Error("missing mesh for side-a")
	}
	if !names["side-b"] {
		t.Error("missing mesh for side-b")
	}
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
