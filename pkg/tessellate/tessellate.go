// Package tessellate walks a design graph and produces triangle meshes
// using a geometry kernel. One mesh is produced per part.
package tessellate

import (
	"fmt"

	"github.com/chazu/lignin/pkg/graph"
	"github.com/chazu/lignin/pkg/kernel"
)

// transformStack accumulates spatial transforms during graph traversal.
type transformStack struct {
	translations []graph.Vec3
	rotations    []graph.Vec3
}

func newTransformStack() *transformStack {
	return &transformStack{}
}

func (ts *transformStack) pushTranslation(v graph.Vec3) {
	ts.translations = append(ts.translations, v)
}

func (ts *transformStack) pushRotation(v graph.Vec3) {
	ts.rotations = append(ts.rotations, v)
}

func (ts *transformStack) pop() {
	if len(ts.translations) > 0 {
		ts.translations = ts.translations[:len(ts.translations)-1]
	}
	if len(ts.rotations) > 0 {
		ts.rotations = ts.rotations[:len(ts.rotations)-1]
	}
}

// accumulatedTranslation returns the sum of all translations on the stack.
func (ts *transformStack) accumulatedTranslation() graph.Vec3 {
	var sum graph.Vec3
	for _, t := range ts.translations {
		sum = sum.Add(t)
	}
	return sum
}

// accumulatedRotation returns the sum of all rotations on the stack.
func (ts *transformStack) accumulatedRotation() graph.Vec3 {
	var sum graph.Vec3
	for _, r := range ts.rotations {
		sum = sum.Add(r)
	}
	return sum
}

// Tessellate walks the design graph and produces one triangle mesh per
// primitive part using the provided geometry kernel. The tessellator is
// read-only and never mutates the graph.
func Tessellate(g *graph.DesignGraph, k kernel.Kernel) ([]*kernel.Mesh, error) {
	if g == nil {
		return nil, nil
	}

	var meshes []*kernel.Mesh
	ts := newTransformStack()

	for _, rootID := range g.Roots {
		root := g.Get(rootID)
		if root == nil {
			continue
		}
		collected, err := walkNode(g, k, root, ts)
		if err != nil {
			return nil, fmt.Errorf("tessellate: error walking root %s: %w", rootID.Short(), err)
		}
		meshes = append(meshes, collected...)
	}

	return meshes, nil
}

// walkNode recursively traverses a node and its children, collecting meshes.
func walkNode(g *graph.DesignGraph, k kernel.Kernel, n *graph.Node, ts *transformStack) ([]*kernel.Mesh, error) {
	switch n.Kind {
	case graph.NodePrimitive:
		return handlePrimitive(k, n, ts)

	case graph.NodeTransform:
		return handleTransform(g, k, n, ts)

	case graph.NodeGroup:
		return handleGroup(g, k, n, ts)

	case graph.NodeJoin:
		// MVP: butt joints are metadata-only, skip.
		return nil, nil

	case graph.NodeFastener:
		// Future: generate fastener geometry.
		return nil, nil

	case graph.NodeDrill:
		// Future: generate drill geometry.
		return nil, nil

	default:
		return nil, fmt.Errorf("unknown node kind: %v", n.Kind)
	}
}

// handlePrimitive creates geometry for a primitive node.
func handlePrimitive(k kernel.Kernel, n *graph.Node, ts *transformStack) ([]*kernel.Mesh, error) {
	var solid kernel.Solid

	switch data := n.Data.(type) {
	case graph.BoardData:
		solid = k.Box(data.Dimensions.X, data.Dimensions.Y, data.Dimensions.Z)
	case graph.DowelData:
		solid = k.Cylinder(data.Length, data.Diameter/2, 32)
	default:
		return nil, fmt.Errorf("primitive node %s has unsupported data type %T", n.ID.Short(), n.Data)
	}

	// Apply accumulated rotation first, then translation.
	rot := ts.accumulatedRotation()
	if rot.X != 0 || rot.Y != 0 || rot.Z != 0 {
		solid = k.Rotate(solid, rot.X, rot.Y, rot.Z)
	}

	trans := ts.accumulatedTranslation()
	if trans.X != 0 || trans.Y != 0 || trans.Z != 0 {
		solid = k.Translate(solid, trans.X, trans.Y, trans.Z)
	}

	mesh, err := k.ToMesh(solid)
	if err != nil {
		return nil, fmt.Errorf("tessellate: ToMesh failed for node %s: %w", n.ID.Short(), err)
	}

	// Set the part name: prefer the node's Name, fall back to short ID.
	if n.Name != "" {
		mesh.PartName = n.Name
	} else {
		mesh.PartName = n.ID.Short()
	}

	return []*kernel.Mesh{mesh}, nil
}

// handleTransform pushes the transform, recurses into children, then pops.
func handleTransform(g *graph.DesignGraph, k kernel.Kernel, n *graph.Node, ts *transformStack) ([]*kernel.Mesh, error) {
	td, ok := n.Data.(graph.TransformData)
	if !ok {
		return nil, fmt.Errorf("transform node %s has unexpected data type %T", n.ID.Short(), n.Data)
	}

	// Push transform onto the stack.
	translation := graph.Vec3{}
	rotation := graph.Vec3{}
	if td.Translation != nil {
		translation = *td.Translation
	}
	if td.Rotation != nil {
		rotation = *td.Rotation
	}
	ts.pushTranslation(translation)
	ts.pushRotation(rotation)

	var meshes []*kernel.Mesh
	for _, child := range g.Children(n) {
		collected, err := walkNode(g, k, child, ts)
		if err != nil {
			ts.pop()
			return nil, err
		}
		meshes = append(meshes, collected...)
	}

	ts.pop()
	return meshes, nil
}

// handleGroup recurses into children transparently.
func handleGroup(g *graph.DesignGraph, k kernel.Kernel, n *graph.Node, ts *transformStack) ([]*kernel.Mesh, error) {
	var meshes []*kernel.Mesh
	for _, child := range g.Children(n) {
		collected, err := walkNode(g, k, child, ts)
		if err != nil {
			return nil, err
		}
		meshes = append(meshes, collected...)
	}
	return meshes, nil
}
