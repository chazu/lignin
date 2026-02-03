package engine

import (
	"fmt"
	"strings"
	"sync/atomic"

	"github.com/chazu/lignin/pkg/graph"
	zygo "github.com/glycerine/zygomys/zygo"
)

// ---------------------------------------------------------------------------
// Source preprocessing
// ---------------------------------------------------------------------------

// preprocessSource transforms Lignin Lisp source code before passing it to
// zygomys. It performs two transformations:
//
//  1. Keyword conversion: :keyword -> "__kw_keyword" (string literal)
//     This avoids the need to register keyword symbols as globals, which
//     would conflict with user-defined variables of the same name.
//
//  2. Kebab-case to underscore: butt-joint -> butt_joint
//     zygomys does not allow hyphens in identifiers (it interprets them
//     as the subtraction operator). This converts kebab-case identifiers
//     to underscore form outside of strings and comments.
//
// Both transformations respect string literal boundaries and line comments.
func preprocessSource(source string) string {
	result := make([]byte, 0, len(source)+len(source)/4)
	b := []byte(source)
	i := 0
	for i < len(b) {
		// Skip double-quoted string literals.
		if b[i] == '"' {
			result = append(result, b[i])
			i++
			for i < len(b) && b[i] != '"' {
				if b[i] == '\\' && i+1 < len(b) {
					result = append(result, b[i], b[i+1])
					i += 2
					continue
				}
				result = append(result, b[i])
				i++
			}
			if i < len(b) {
				result = append(result, b[i])
				i++
			}
			continue
		}
		// Skip backtick-quoted string literals.
		if b[i] == '`' {
			result = append(result, b[i])
			i++
			for i < len(b) && b[i] != '`' {
				result = append(result, b[i])
				i++
			}
			if i < len(b) {
				result = append(result, b[i])
				i++
			}
			continue
		}
		// Convert ; line comments to // comments for zygomys.
		// zygomys uses // for line comments, not the traditional Lisp ;.
		if b[i] == ';' {
			result = append(result, '/', '/')
			i++
			// Skip additional ; characters (;; style).
			for i < len(b) && b[i] == ';' {
				i++
			}
			for i < len(b) && b[i] != '\n' {
				result = append(result, b[i])
				i++
			}
			continue
		}
		// Transform :keyword to "__kw_keyword".
		if b[i] == ':' && i+1 < len(b) {
			// Preserve := (assignment operator).
			if b[i+1] == '=' {
				result = append(result, b[i], b[i+1])
				i += 2
				continue
			}
			// Check for keyword: colon followed by a letter.
			if isLetter(b[i+1]) {
				j := i + 1
				for j < len(b) && isKWChar(b[j]) {
					j++
				}
				kwName := string(b[i+1 : j])
				result = append(result, '"')
				result = append(result, []byte(kwPrefix)...)
				result = append(result, []byte(kwName)...)
				result = append(result, '"')
				i = j
				continue
			}
		}
		// Transform kebab-case identifiers: alpha-alpha -> alpha_alpha.
		// Only when hyphen sits between identifier characters (not a minus operator).
		if b[i] == '-' && i > 0 && i+1 < len(b) &&
			isIdentChar(b[i-1]) && isIdentStartChar(b[i+1]) {
			result = append(result, '_')
			i++
			continue
		}
		result = append(result, b[i])
		i++
	}
	return string(result)
}

func isLetter(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func isKWChar(c byte) bool {
	return isLetter(c) || (c >= '0' && c <= '9') || c == '-' || c == '_'
}

func isIdentChar(c byte) bool {
	return isLetter(c) || (c >= '0' && c <= '9') || c == '_'
}

func isIdentStartChar(c byte) bool {
	return isLetter(c)
}

// ---------------------------------------------------------------------------
// Custom Sexp types for passing Go values through the zygomys environment
// ---------------------------------------------------------------------------

// sexpMaterial wraps a graph.MaterialSpec so it can be passed between builtins.
type sexpMaterial struct {
	spec graph.MaterialSpec
}

func (m *sexpMaterial) SexpString(ps *zygo.PrintState) string {
	return fmt.Sprintf("(material :species %q)", m.spec.Species)
}
func (m *sexpMaterial) Type() *zygo.RegisteredType { return nil }

// sexpBoard wraps a graph.BoardData so it can be returned from `board`
// and consumed by `defpart`.
type sexpBoard struct {
	data graph.BoardData
}

func (b *sexpBoard) SexpString(ps *zygo.PrintState) string {
	return fmt.Sprintf("(board %.0fx%.0fx%.0f)", b.data.Dimensions.X, b.data.Dimensions.Y, b.data.Dimensions.Z)
}
func (b *sexpBoard) Type() *zygo.RegisteredType { return nil }

// sexpNodeRef wraps a graph.NodeID so it can be passed between builtins.
type sexpNodeRef struct {
	id   graph.NodeID
	name string // human-readable name for error messages
}

func (n *sexpNodeRef) SexpString(ps *zygo.PrintState) string {
	if n.name != "" {
		return fmt.Sprintf("(noderef %q)", n.name)
	}
	return fmt.Sprintf("(noderef %s)", n.id.Short())
}
func (n *sexpNodeRef) Type() *zygo.RegisteredType { return nil }

// sexpVec3 wraps a graph.Vec3.
type sexpVec3 struct {
	vec graph.Vec3
}

func (v *sexpVec3) SexpString(ps *zygo.PrintState) string {
	return fmt.Sprintf("(vec3 %.1f %.1f %.1f)", v.vec.X, v.vec.Y, v.vec.Z)
}
func (v *sexpVec3) Type() *zygo.RegisteredType { return nil }

// ---------------------------------------------------------------------------
// Keyword argument parsing
// ---------------------------------------------------------------------------

// kwPrefix is the marker prepended to keyword names by preprocessSource.
const kwPrefix = "__kw_"

// isKW checks if a Sexp is a preprocessed keyword string.
// Returns the keyword name (without prefix) and true if it is.
func isKW(s zygo.Sexp) (string, bool) {
	str, ok := s.(*zygo.SexpStr)
	if !ok {
		return "", false
	}
	if strings.HasPrefix(str.S, kwPrefix) {
		return str.S[len(kwPrefix):], true
	}
	return "", false
}

// kwArgs holds the result of parsing a mixed positional+keyword argument list.
type kwArgs struct {
	kw         map[string]zygo.Sexp
	positional []zygo.Sexp
}

// parseArgs separates args into keyword and positional arguments.
// Keywords are identified by the __kw_ prefix added during preprocessing.
func parseArgs(args []zygo.Sexp) kwArgs {
	result := kwArgs{kw: make(map[string]zygo.Sexp)}
	i := 0
	for i < len(args) {
		name, ok := isKW(args[i])
		if ok {
			if i+1 < len(args) {
				result.kw[name] = args[i+1]
				i += 2
			} else {
				// Keyword at end with no value — treat as flag with nil.
				result.kw[name] = zygo.SexpNull
				i++
			}
		} else {
			result.positional = append(result.positional, args[i])
			i++
		}
	}
	return result
}

// ---------------------------------------------------------------------------
// Value extraction helpers
// ---------------------------------------------------------------------------

// toFloat64 extracts a float64 from a Sexp (SexpInt or SexpFloat).
func toFloat64(s zygo.Sexp) (float64, error) {
	switch v := s.(type) {
	case *zygo.SexpInt:
		return float64(v.Val), nil
	case *zygo.SexpFloat:
		return v.Val, nil
	}
	return 0, fmt.Errorf("expected number, got %T (%s)", s, s.SexpString(nil))
}

// toString extracts a string from a Sexp.
func toString(s zygo.Sexp) (string, error) {
	if str, ok := s.(*zygo.SexpStr); ok {
		return str.S, nil
	}
	return "", fmt.Errorf("expected string, got %T (%s)", s, s.SexpString(nil))
}

// toKeywordString extracts a keyword name or plain string from a Sexp.
// Handles both preprocessed keywords (__kw_z) and plain strings ("z").
func toKeywordString(s zygo.Sexp) (string, error) {
	str, ok := s.(*zygo.SexpStr)
	if !ok {
		return "", fmt.Errorf("expected keyword or string, got %T (%s)", s, s.SexpString(nil))
	}
	if strings.HasPrefix(str.S, kwPrefix) {
		return str.S[len(kwPrefix):], nil
	}
	return str.S, nil
}

// toAxis converts a keyword or string to a graph.Axis.
func toAxis(s zygo.Sexp) (graph.Axis, error) {
	name, err := toKeywordString(s)
	if err != nil {
		return 0, fmt.Errorf("expected axis keyword (:x, :y, :z): %w", err)
	}
	switch name {
	case "x":
		return graph.AxisX, nil
	case "y":
		return graph.AxisY, nil
	case "z":
		return graph.AxisZ, nil
	}
	return 0, fmt.Errorf("invalid axis %q, expected x, y, or z", name)
}

// toFaceID converts a keyword or string to a graph.FaceID.
func toFaceID(s zygo.Sexp) (graph.FaceID, error) {
	name, err := toKeywordString(s)
	if err != nil {
		return "", fmt.Errorf("expected face keyword: %w", err)
	}
	fid := graph.FaceID(name)
	if !graph.ValidFaceIDs[fid] {
		return "", fmt.Errorf("invalid face %q, expected top/bottom/left/right/front/back", name)
	}
	return fid, nil
}

// toNodeRef extracts a NodeID from a sexpNodeRef.
func toNodeRef(s zygo.Sexp) (graph.NodeID, error) {
	if ref, ok := s.(*sexpNodeRef); ok {
		return ref.id, nil
	}
	return graph.ZeroID, fmt.Errorf("expected node reference, got %T (%s)", s, s.SexpString(nil))
}

// toVec3 extracts a Vec3 from a sexpVec3.
func toVec3(s zygo.Sexp) (graph.Vec3, error) {
	if v, ok := s.(*sexpVec3); ok {
		return v.vec, nil
	}
	return graph.Vec3{}, fmt.Errorf("expected vec3, got %T (%s)", s, s.SexpString(nil))
}

// toMaterial extracts a MaterialSpec from a sexpMaterial.
func toMaterial(s zygo.Sexp) (graph.MaterialSpec, error) {
	if m, ok := s.(*sexpMaterial); ok {
		return m.spec, nil
	}
	return graph.MaterialSpec{}, fmt.Errorf("expected material, got %T (%s)", s, s.SexpString(nil))
}

// sexpListToSlice converts a SexpPair (Lisp list) or SexpArray to a Go slice.
func sexpListToSlice(s zygo.Sexp) ([]zygo.Sexp, error) {
	switch v := s.(type) {
	case *zygo.SexpPair:
		return zygo.ListToArray(v)
	case *zygo.SexpArray:
		return v.Val, nil
	case *zygo.SexpSentinel:
		if v == zygo.SexpNull {
			return nil, nil
		}
	}
	return nil, fmt.Errorf("expected list or array, got %T", s)
}

// ---------------------------------------------------------------------------
// Node ID generation
// ---------------------------------------------------------------------------

// nodeCounter provides unique suffixes for anonymous nodes.
var nodeCounter uint64

func nextNodeSuffix() string {
	n := atomic.AddUint64(&nodeCounter, 1)
	return fmt.Sprintf("_anon_%d", n)
}

// ---------------------------------------------------------------------------
// Builtin registration
// ---------------------------------------------------------------------------

// registerBuiltins installs all Lignin DSL builtins into a zygomys environment.
// The builtins operate on the provided DesignGraph, populating it during evaluation.
//
// Source code must be preprocessed with preprocessSource() before evaluation so
// that :keyword tokens are converted to recognizable string literals.
func registerBuiltins(env *zygo.Zlisp, g *graph.DesignGraph) {

	// -----------------------------------------------------------------------
	// (material :species "white-oak" :thickness 19 :grade "FAS")
	// -----------------------------------------------------------------------
	env.AddFunction("material", func(env *zygo.Zlisp, name string, args []zygo.Sexp) (zygo.Sexp, error) {
		pa := parseArgs(args)
		spec := graph.MaterialSpec{}

		if v, ok := pa.kw["species"]; ok {
			s, err := toString(v)
			if err != nil {
				return zygo.SexpNull, fmt.Errorf("material: species: %w", err)
			}
			spec.Species = s
		}
		if v, ok := pa.kw["thickness"]; ok {
			f, err := toFloat64(v)
			if err != nil {
				return zygo.SexpNull, fmt.Errorf("material: thickness: %w", err)
			}
			spec.Thickness = f
		}
		if v, ok := pa.kw["grade"]; ok {
			s, err := toString(v)
			if err != nil {
				return zygo.SexpNull, fmt.Errorf("material: grade: %w", err)
			}
			spec.Grade = s
		}

		return &sexpMaterial{spec: spec}, nil
	})

	// -----------------------------------------------------------------------
	// (board :length 400 :width 200 :thickness 19 :grain :z :material oak)
	// -----------------------------------------------------------------------
	env.AddFunction("board", func(env *zygo.Zlisp, name string, args []zygo.Sexp) (zygo.Sexp, error) {
		pa := parseArgs(args)
		bd := graph.BoardData{PrimKind: graph.PrimBoard}

		if v, ok := pa.kw["length"]; ok {
			f, err := toFloat64(v)
			if err != nil {
				return zygo.SexpNull, fmt.Errorf("board: length: %w", err)
			}
			bd.Dimensions.X = f
		}
		if v, ok := pa.kw["width"]; ok {
			f, err := toFloat64(v)
			if err != nil {
				return zygo.SexpNull, fmt.Errorf("board: width: %w", err)
			}
			bd.Dimensions.Y = f
		}
		if v, ok := pa.kw["thickness"]; ok {
			f, err := toFloat64(v)
			if err != nil {
				return zygo.SexpNull, fmt.Errorf("board: thickness: %w", err)
			}
			bd.Dimensions.Z = f
		}
		if v, ok := pa.kw["grain"]; ok {
			a, err := toAxis(v)
			if err != nil {
				return zygo.SexpNull, fmt.Errorf("board: grain: %w", err)
			}
			bd.Grain = a
		}
		if v, ok := pa.kw["material"]; ok {
			m, err := toMaterial(v)
			if err != nil {
				return zygo.SexpNull, fmt.Errorf("board: material: %w", err)
			}
			bd.Material = m
		}

		return &sexpBoard{data: bd}, nil
	})

	// -----------------------------------------------------------------------
	// (defpart "name" (board ...))
	// -----------------------------------------------------------------------
	env.AddFunction("defpart", func(env *zygo.Zlisp, name string, args []zygo.Sexp) (zygo.Sexp, error) {
		if len(args) < 2 {
			return zygo.SexpNull, fmt.Errorf("defpart requires a name and a body expression")
		}

		partName, err := toString(args[0])
		if err != nil {
			return zygo.SexpNull, fmt.Errorf("defpart: name: %w", err)
		}

		var nodeData graph.NodeData
		switch body := args[1].(type) {
		case *sexpBoard:
			nodeData = body.data
		default:
			return zygo.SexpNull, fmt.Errorf("defpart: expected board expression, got %T", args[1])
		}

		id := graph.NewNodeID(partName)
		node := &graph.Node{
			ID:   id,
			Kind: graph.NodePrimitive,
			Name: partName,
			Data: nodeData,
		}
		g.AddNode(node)

		return &sexpNodeRef{id: id, name: partName}, nil
	})

	// -----------------------------------------------------------------------
	// (part "name")
	// -----------------------------------------------------------------------
	env.AddFunction("part", func(env *zygo.Zlisp, name string, args []zygo.Sexp) (zygo.Sexp, error) {
		if len(args) < 1 {
			return zygo.SexpNull, fmt.Errorf("part requires a name argument")
		}

		partName, err := toString(args[0])
		if err != nil {
			return zygo.SexpNull, fmt.Errorf("part: name: %w", err)
		}

		n := g.Lookup(partName)
		if n == nil {
			return zygo.SexpNull, fmt.Errorf("part: no part named %q", partName)
		}

		return &sexpNodeRef{id: n.ID, name: partName}, nil
	})

	// -----------------------------------------------------------------------
	// (vec3 1 2 3)
	// -----------------------------------------------------------------------
	env.AddFunction("vec3", func(env *zygo.Zlisp, name string, args []zygo.Sexp) (zygo.Sexp, error) {
		if len(args) != 3 {
			return zygo.SexpNull, fmt.Errorf("vec3 requires exactly 3 arguments, got %d", len(args))
		}

		x, err := toFloat64(args[0])
		if err != nil {
			return zygo.SexpNull, fmt.Errorf("vec3: x: %w", err)
		}
		y, err := toFloat64(args[1])
		if err != nil {
			return zygo.SexpNull, fmt.Errorf("vec3: y: %w", err)
		}
		z, err := toFloat64(args[2])
		if err != nil {
			return zygo.SexpNull, fmt.Errorf("vec3: z: %w", err)
		}

		return &sexpVec3{vec: graph.Vec3{X: x, Y: y, Z: z}}, nil
	})

	// -----------------------------------------------------------------------
	// (place (part "front") :at (vec3 0 0 19))
	// -----------------------------------------------------------------------
	env.AddFunction("place", func(env *zygo.Zlisp, name string, args []zygo.Sexp) (zygo.Sexp, error) {
		pa := parseArgs(args)

		if len(pa.positional) < 1 {
			return zygo.SexpNull, fmt.Errorf("place requires a part reference as first argument")
		}

		childID, err := toNodeRef(pa.positional[0])
		if err != nil {
			return zygo.SexpNull, fmt.Errorf("place: part: %w", err)
		}

		td := graph.TransformData{}
		if v, ok := pa.kw["at"]; ok {
			vec, err := toVec3(v)
			if err != nil {
				return zygo.SexpNull, fmt.Errorf("place: at: %w", err)
			}
			td.Translation = &vec
		}

		// Generate a deterministic ID from the child node name.
		childNode := g.Get(childID)
		idPath := "place/" + nextNodeSuffix()
		if childNode != nil && childNode.Name != "" {
			idPath = "place/" + childNode.Name
		}
		id := graph.NewNodeID(idPath)

		node := &graph.Node{
			ID:       id,
			Kind:     graph.NodeTransform,
			Children: []graph.NodeID{childID},
			Data:     td,
		}
		g.AddNode(node)

		return &sexpNodeRef{id: id}, nil
	})

	// -----------------------------------------------------------------------
	// (butt-joint :part-a ref :face-a :left :part-b ref :face-b :front
	//             :clearance 0.5 :fasteners (list ...))
	//
	// Note: registered as "butt_joint" because zygomys does not support
	// hyphens in identifiers. The preprocessor converts butt-joint to
	// butt_joint in the source.
	// -----------------------------------------------------------------------
	env.AddFunction("butt_joint", func(env *zygo.Zlisp, name string, args []zygo.Sexp) (zygo.Sexp, error) {
		pa := parseArgs(args)
		jd := graph.JoinData{
			Kind:   graph.JoinButt,
			Params: graph.ButtJoinParams{},
		}

		if v, ok := pa.kw["part-a"]; ok {
			id, err := toNodeRef(v)
			if err != nil {
				return zygo.SexpNull, fmt.Errorf("butt-joint: part-a: %w", err)
			}
			jd.PartA = id
		}
		if v, ok := pa.kw["face-a"]; ok {
			f, err := toFaceID(v)
			if err != nil {
				return zygo.SexpNull, fmt.Errorf("butt-joint: face-a: %w", err)
			}
			jd.FaceA = f
		}
		if v, ok := pa.kw["part-b"]; ok {
			id, err := toNodeRef(v)
			if err != nil {
				return zygo.SexpNull, fmt.Errorf("butt-joint: part-b: %w", err)
			}
			jd.PartB = id
		}
		if v, ok := pa.kw["face-b"]; ok {
			f, err := toFaceID(v)
			if err != nil {
				return zygo.SexpNull, fmt.Errorf("butt-joint: face-b: %w", err)
			}
			jd.FaceB = f
		}
		if v, ok := pa.kw["clearance"]; ok {
			c, err := toFloat64(v)
			if err != nil {
				return zygo.SexpNull, fmt.Errorf("butt-joint: clearance: %w", err)
			}
			jd.Clearance = c
		}
		if v, ok := pa.kw["fasteners"]; ok {
			items, err := sexpListToSlice(v)
			if err != nil {
				return zygo.SexpNull, fmt.Errorf("butt-joint: fasteners: %w", err)
			}
			for _, item := range items {
				fid, err := toNodeRef(item)
				if err != nil {
					return zygo.SexpNull, fmt.Errorf("butt-joint: fastener entry: %w", err)
				}
				jd.Fasteners = append(jd.Fasteners, fid)
			}
		}

		idPath := "butt-joint/" + nextNodeSuffix()
		id := graph.NewNodeID(idPath)

		node := &graph.Node{
			ID:   id,
			Kind: graph.NodeJoin,
			Data: jd,
		}
		g.AddNode(node)

		return &sexpNodeRef{id: id}, nil
	})

	// -----------------------------------------------------------------------
	// (screw :diameter 4 :length 50 :position (vec3 0 50 0) :head-dia 8)
	// -----------------------------------------------------------------------
	env.AddFunction("screw", func(env *zygo.Zlisp, name string, args []zygo.Sexp) (zygo.Sexp, error) {
		pa := parseArgs(args)
		fd := graph.FastenerData{Kind: graph.FastenerScrew}

		if v, ok := pa.kw["diameter"]; ok {
			f, err := toFloat64(v)
			if err != nil {
				return zygo.SexpNull, fmt.Errorf("screw: diameter: %w", err)
			}
			fd.Diameter = f
		}
		if v, ok := pa.kw["length"]; ok {
			f, err := toFloat64(v)
			if err != nil {
				return zygo.SexpNull, fmt.Errorf("screw: length: %w", err)
			}
			fd.Length = f
		}
		if v, ok := pa.kw["position"]; ok {
			vec, err := toVec3(v)
			if err != nil {
				return zygo.SexpNull, fmt.Errorf("screw: position: %w", err)
			}
			fd.Position = vec
		}
		if v, ok := pa.kw["head-dia"]; ok {
			f, err := toFloat64(v)
			if err != nil {
				return zygo.SexpNull, fmt.Errorf("screw: head-dia: %w", err)
			}
			fd.HeadDia = f
		}

		idPath := "screw/" + nextNodeSuffix()
		id := graph.NewNodeID(idPath)

		node := &graph.Node{
			ID:   id,
			Kind: graph.NodeFastener,
			Data: fd,
		}
		g.AddNode(node)

		return &sexpNodeRef{id: id}, nil
	})

	// -----------------------------------------------------------------------
	// (assembly "name" (place ...) (place ...) (butt-joint ...) ...)
	// -----------------------------------------------------------------------
	env.AddFunction("assembly", func(env *zygo.Zlisp, name string, args []zygo.Sexp) (zygo.Sexp, error) {
		if len(args) < 1 {
			return zygo.SexpNull, fmt.Errorf("assembly requires a name argument")
		}

		asmName, err := toString(args[0])
		if err != nil {
			return zygo.SexpNull, fmt.Errorf("assembly: name: %w", err)
		}

		var children []graph.NodeID
		for i := 1; i < len(args); i++ {
			ref, ok := args[i].(*sexpNodeRef)
			if !ok {
				return zygo.SexpNull, fmt.Errorf("assembly: child %d: expected node reference, got %T (%s)",
					i, args[i], args[i].SexpString(nil))
			}
			children = append(children, ref.id)
		}

		id := graph.NewNodeID(asmName)
		node := &graph.Node{
			ID:       id,
			Kind:     graph.NodeGroup,
			Name:     asmName,
			Children: children,
			Data:     graph.GroupData{},
		}
		g.AddNode(node)
		g.AddRoot(id)

		return &sexpNodeRef{id: id, name: asmName}, nil
	})
}
