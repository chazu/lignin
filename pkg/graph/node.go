package graph

// NodeKind enumerates the types of nodes in the design graph.
type NodeKind int

const (
	NodePrimitive NodeKind = iota // geometric primitive (board, dowel)
	NodeTransform                 // spatial transformation (place)
	NodeJoin                      // joinery operation (butt-joint)
	NodeGroup                     // logical grouping (assembly)
	NodeDrill                     // hole/boring operation
	NodeFastener                  // fastener placement (screw)
)

func (k NodeKind) String() string {
	switch k {
	case NodePrimitive:
		return "primitive"
	case NodeTransform:
		return "transform"
	case NodeJoin:
		return "join"
	case NodeGroup:
		return "group"
	case NodeDrill:
		return "drill"
	case NodeFastener:
		return "fastener"
	default:
		return "unknown"
	}
}

// Node is the fundamental element of the design graph.
type Node struct {
	ID          NodeID      `json:"id"`
	Kind        NodeKind    `json:"kind"`
	Name        string      `json:"name,omitempty"`
	Source      SourceRef   `json:"source"`
	ContentHash ContentHash `json:"content_hash"`
	Children    []NodeID    `json:"children,omitempty"`
	Data        NodeData    `json:"data"`
}

// NodeData is the interface for kind-specific node payloads.
type NodeData interface {
	nodeData() // marker method restricting implementations to this package
}
