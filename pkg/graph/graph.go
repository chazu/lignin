package graph

import "fmt"

// DefaultClearance is the default joint clearance in mm.
const DefaultClearance = 0.25

// GlobalDefaults contains graph-wide default settings.
type GlobalDefaults struct {
	Clearance float64      `json:"clearance"` // default joint clearance mm
	Material  MaterialSpec `json:"material"`  // default material for new parts
	Units     string       `json:"units"`     // "mm" (only option for MVP)
}

// DesignGraph is the top-level immutable data structure produced by Lisp evaluation.
// It is never mutated in place; each evaluation produces a new graph.
type DesignGraph struct {
	Nodes     map[NodeID]*Node   `json:"nodes"`
	Roots     []NodeID           `json:"roots"`
	NameIndex map[string]NodeID  `json:"name_index"`
	Defaults  GlobalDefaults     `json:"defaults"`
	Version   uint64             `json:"version"`
}

// New creates an empty DesignGraph with default settings.
func New() *DesignGraph {
	return &DesignGraph{
		Nodes:     make(map[NodeID]*Node),
		NameIndex: make(map[string]NodeID),
		Defaults: GlobalDefaults{
			Clearance: DefaultClearance,
			Units:     "mm",
		},
	}
}

// AddNode adds a node to the graph. It does not check for duplicates.
func (g *DesignGraph) AddNode(n *Node) {
	g.Nodes[n.ID] = n
	if n.Name != "" {
		g.NameIndex[n.Name] = n.ID
	}
}

// AddRoot registers a node ID as a root of the graph.
func (g *DesignGraph) AddRoot(id NodeID) {
	g.Roots = append(g.Roots, id)
}

// Lookup returns the node with the given user-assigned name, or nil.
func (g *DesignGraph) Lookup(name string) *Node {
	id, ok := g.NameIndex[name]
	if !ok {
		return nil
	}
	return g.Nodes[id]
}

// MustLookup returns the node with the given name, or panics.
func (g *DesignGraph) MustLookup(name string) *Node {
	n := g.Lookup(name)
	if n == nil {
		panic(fmt.Sprintf("graph: no node named %q", name))
	}
	return n
}

// Get returns the node with the given ID, or nil.
func (g *DesignGraph) Get(id NodeID) *Node {
	return g.Nodes[id]
}

// Parts returns all primitive nodes in the graph.
func (g *DesignGraph) Parts() []*Node {
	var parts []*Node
	for _, n := range g.Nodes {
		if n.Kind == NodePrimitive {
			parts = append(parts, n)
		}
	}
	return parts
}

// Joins returns all join nodes in the graph.
func (g *DesignGraph) Joins() []*Node {
	var joins []*Node
	for _, n := range g.Nodes {
		if n.Kind == NodeJoin {
			joins = append(joins, n)
		}
	}
	return joins
}

// Children returns the child nodes of the given node.
func (g *DesignGraph) Children(n *Node) []*Node {
	children := make([]*Node, 0, len(n.Children))
	for _, cid := range n.Children {
		if c := g.Nodes[cid]; c != nil {
			children = append(children, c)
		}
	}
	return children
}

// NodeCount returns the total number of nodes.
func (g *DesignGraph) NodeCount() int {
	return len(g.Nodes)
}
