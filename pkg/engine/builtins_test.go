package engine

import (
	"testing"

	"github.com/chazu/lignin/pkg/graph"
)

// ---------------------------------------------------------------------------
// Preprocessing tests
// ---------------------------------------------------------------------------

func TestPreprocessKeywords(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect string
	}{
		{
			name:   "simple keyword",
			input:  `(material :species "oak")`,
			expect: `(material "__kw_species" "oak")`,
		},
		{
			name:   "multiple keywords",
			input:  `(board :length 400 :width 200)`,
			expect: `(board "__kw_length" 400 "__kw_width" 200)`,
		},
		{
			name:   "keyword in string preserved",
			input:  `"thing with :keyword inside"`,
			expect: `"thing with :keyword inside"`,
		},
		{
			name:   "assignment operator preserved",
			input:  `(def x := 10)`,
			expect: `(def x := 10)`,
		},
		{
			name:   "kebab-case identifier",
			input:  `(butt-joint :part-a ref)`,
			expect: `(butt_joint "__kw_part-a" ref)`,
		},
		{
			name:   "minus operator preserved",
			input:  `(- 10 5)`,
			expect: `(- 10 5)`,
		},
		{
			name:   "comment converted to // style",
			input:  `;; comment with :keyword`,
			expect: `// comment with :keyword`,
		},
		{
			name:   "single semicolon comment",
			input:  `; simple comment`,
			expect: `// simple comment`,
		},
		{
			name:   "hyphen in keyword preserved",
			input:  `:head-dia`,
			expect: `"__kw_head-dia"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := preprocessSource(tt.input)
			if got != tt.expect {
				t.Errorf("preprocessSource(%q) = %q, want %q", tt.input, got, tt.expect)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Simple board test
// ---------------------------------------------------------------------------

func TestSimpleBoard(t *testing.T) {
	eng := NewEngine()

	source := `
(defpart "shelf"
  (board :length 600 :width 300 :thickness 19 :grain :z
         :material (material :species "walnut")))
`
	g, evalErrs, err := eng.Evaluate(source)
	if err != nil {
		t.Fatalf("fatal error: %v", err)
	}
	if len(evalErrs) > 0 {
		t.Fatalf("eval errors: %v", evalErrs)
	}
	if g == nil {
		t.Fatal("expected non-nil graph")
	}
	if g.NodeCount() != 1 {
		t.Fatalf("expected 1 node, got %d", g.NodeCount())
	}

	shelf := g.Lookup("shelf")
	if shelf == nil {
		t.Fatal("expected node named 'shelf'")
	}
	if shelf.Kind != graph.NodePrimitive {
		t.Errorf("expected NodePrimitive, got %s", shelf.Kind)
	}

	bd, ok := shelf.Data.(graph.BoardData)
	if !ok {
		t.Fatalf("expected BoardData, got %T", shelf.Data)
	}
	if bd.Dimensions.X != 600 {
		t.Errorf("expected length=600, got %f", bd.Dimensions.X)
	}
	if bd.Dimensions.Y != 300 {
		t.Errorf("expected width=300, got %f", bd.Dimensions.Y)
	}
	if bd.Dimensions.Z != 19 {
		t.Errorf("expected thickness=19, got %f", bd.Dimensions.Z)
	}
	if bd.Grain != graph.AxisZ {
		t.Errorf("expected grain=Z, got %s", bd.Grain)
	}
	if bd.Material.Species != "walnut" {
		t.Errorf("expected species=walnut, got %q", bd.Material.Species)
	}
}

// ---------------------------------------------------------------------------
// Variable reference test
// ---------------------------------------------------------------------------

func TestVariableReference(t *testing.T) {
	eng := NewEngine()

	source := `
(def t 19)
(defpart "side"
  (board :length 400 :width 200 :thickness t :grain :z
         :material (material :species "oak")))
`
	g, evalErrs, err := eng.Evaluate(source)
	if err != nil {
		t.Fatalf("fatal error: %v", err)
	}
	if len(evalErrs) > 0 {
		t.Fatalf("eval errors: %v", evalErrs)
	}

	side := g.Lookup("side")
	if side == nil {
		t.Fatal("expected node named 'side'")
	}

	bd, ok := side.Data.(graph.BoardData)
	if !ok {
		t.Fatalf("expected BoardData, got %T", side.Data)
	}
	if bd.Dimensions.Z != 19 {
		t.Errorf("expected thickness=19 (from variable), got %f", bd.Dimensions.Z)
	}
	if bd.Material.Species != "oak" {
		t.Errorf("expected species=oak, got %q", bd.Material.Species)
	}
}

// ---------------------------------------------------------------------------
// Assembly with placement test
// ---------------------------------------------------------------------------

func TestAssemblyWithPlacement(t *testing.T) {
	eng := NewEngine()

	source := `
(def oak (material :species "white-oak"))
(defpart "top" (board :length 400 :width 200 :thickness 19 :grain :z :material oak))
(defpart "leg" (board :length 200 :width 50 :thickness 50 :grain :z :material oak))

(assembly "table"
  (place (part "top") :at (vec3 0 0 200))
  (place (part "leg") :at (vec3 0 0 0)))
`
	g, evalErrs, err := eng.Evaluate(source)
	if err != nil {
		t.Fatalf("fatal error: %v", err)
	}
	if len(evalErrs) > 0 {
		t.Fatalf("eval errors: %v", evalErrs)
	}

	// 2 primitives + 2 transforms + 1 group = 5 nodes
	if g.NodeCount() != 5 {
		t.Fatalf("expected 5 nodes, got %d", g.NodeCount())
	}

	// Check primitives exist.
	topNode := g.Lookup("top")
	if topNode == nil {
		t.Fatal("expected node named 'top'")
	}
	if topNode.Kind != graph.NodePrimitive {
		t.Errorf("top: expected NodePrimitive, got %s", topNode.Kind)
	}

	legNode := g.Lookup("leg")
	if legNode == nil {
		t.Fatal("expected node named 'leg'")
	}

	// Check assembly exists and has children.
	table := g.Lookup("table")
	if table == nil {
		t.Fatal("expected node named 'table'")
	}
	if table.Kind != graph.NodeGroup {
		t.Errorf("table: expected NodeGroup, got %s", table.Kind)
	}
	if len(table.Children) != 2 {
		t.Errorf("table: expected 2 children, got %d", len(table.Children))
	}

	// Check roots.
	if len(g.Roots) != 1 {
		t.Errorf("expected 1 root, got %d", len(g.Roots))
	}

	// Check transform nodes exist and have correct translations.
	transforms := 0
	for _, n := range g.Nodes {
		if n.Kind == graph.NodeTransform {
			transforms++
			td, ok := n.Data.(graph.TransformData)
			if !ok {
				t.Errorf("transform node: expected TransformData, got %T", n.Data)
			}
			if td.Translation == nil {
				t.Error("transform node: expected non-nil translation")
			}
		}
	}
	if transforms != 2 {
		t.Errorf("expected 2 transform nodes, got %d", transforms)
	}
}

// ---------------------------------------------------------------------------
// Butt joint test
// ---------------------------------------------------------------------------

func TestButtJoint(t *testing.T) {
	eng := NewEngine()

	source := `
(def oak (material :species "white-oak"))
(defpart "front" (board :length 400 :width 200 :thickness 19 :grain :z :material oak))
(defpart "left"  (board :length 262 :width 200 :thickness 19 :grain :z :material oak))

(assembly "corner"
  (place (part "front") :at (vec3 0 0 0))
  (place (part "left")  :at (vec3 0 0 19))
  (butt-joint
    :part-a (part "front") :face-a :left
    :part-b (part "left")  :face-b :front))
`
	g, evalErrs, err := eng.Evaluate(source)
	if err != nil {
		t.Fatalf("fatal error: %v", err)
	}
	if len(evalErrs) > 0 {
		t.Fatalf("eval errors: %v", evalErrs)
	}

	// Find the join node.
	joins := g.Joins()
	if len(joins) != 1 {
		t.Fatalf("expected 1 join node, got %d", len(joins))
	}

	join := joins[0]
	if join.Kind != graph.NodeJoin {
		t.Errorf("expected NodeJoin, got %s", join.Kind)
	}

	jd, ok := join.Data.(graph.JoinData)
	if !ok {
		t.Fatalf("expected JoinData, got %T", join.Data)
	}
	if jd.Kind != graph.JoinButt {
		t.Errorf("expected JoinButt, got %s", jd.Kind)
	}

	// Check part references.
	frontNode := g.Lookup("front")
	leftNode := g.Lookup("left")
	if frontNode == nil || leftNode == nil {
		t.Fatal("expected front and left nodes")
	}
	if jd.PartA != frontNode.ID {
		t.Error("expected PartA to reference 'front'")
	}
	if jd.FaceA != graph.FaceLeft {
		t.Errorf("expected FaceA=left, got %s", jd.FaceA)
	}
	if jd.PartB != leftNode.ID {
		t.Error("expected PartB to reference 'left'")
	}
	if jd.FaceB != graph.FaceFront {
		t.Errorf("expected FaceB=front, got %s", jd.FaceB)
	}
}

// ---------------------------------------------------------------------------
// Part lookup error test
// ---------------------------------------------------------------------------

func TestPartLookupError(t *testing.T) {
	eng := NewEngine()

	source := `(part "nonexistent")`
	_, evalErrs, err := eng.Evaluate(source)
	if err != nil {
		t.Fatalf("expected non-fatal eval error, got fatal: %v", err)
	}
	if len(evalErrs) == 0 {
		t.Fatal("expected at least one eval error for missing part")
	}

	found := false
	for _, e := range evalErrs {
		if e.Message != "" {
			found = true
		}
	}
	if !found {
		t.Error("eval error should have a non-empty message")
	}
}

// ---------------------------------------------------------------------------
// Full box example test
// ---------------------------------------------------------------------------

func TestFullBoxExample(t *testing.T) {
	eng := NewEngine()

	source := `
(def thickness 19)
(def oak (material :species "white-oak"))

(defpart "front"
  (board :length 400 :width 200 :thickness thickness
         :grain :z :material oak))

(defpart "bottom"
  (board :length 362 :width 262 :thickness thickness
         :grain :z :material oak))

(defpart "left"
  (board :length 262 :width 200 :thickness thickness
         :grain :z :material oak))

(assembly "box"
  (place (part "front")  :at (vec3 0 0 0))
  (place (part "left")   :at (vec3 0 0 19))
  (place (part "bottom") :at (vec3 19 0 19))

  (butt-joint
    :part-a (part "front") :face-a :left
    :part-b (part "left")  :face-b :front
    :fasteners
      (list
        (screw :diameter 4 :length 50 :position (vec3 0 50 0))
        (screw :diameter 4 :length 50 :position (vec3 0 150 0)))))
`
	g, evalErrs, err := eng.Evaluate(source)
	if err != nil {
		t.Fatalf("fatal error: %v", err)
	}
	if len(evalErrs) > 0 {
		t.Fatalf("eval errors: %v", evalErrs)
	}
	if g == nil {
		t.Fatal("expected non-nil graph")
	}

	// Expected nodes:
	// 3 primitives (front, bottom, left)
	// 3 transforms (place for each)
	// 1 group (assembly "box")
	// 1 join (butt-joint)
	// 2 fasteners (screws)
	// Total: 10
	if g.NodeCount() != 10 {
		t.Fatalf("expected 10 nodes, got %d", g.NodeCount())
	}

	// Verify primitives.
	front := g.Lookup("front")
	if front == nil {
		t.Fatal("missing 'front' node")
	}
	frontBd := front.Data.(graph.BoardData)
	if frontBd.Dimensions.X != 400 {
		t.Errorf("front length: expected 400, got %f", frontBd.Dimensions.X)
	}
	if frontBd.Dimensions.Z != 19 {
		t.Errorf("front thickness: expected 19, got %f", frontBd.Dimensions.Z)
	}
	if frontBd.Material.Species != "white-oak" {
		t.Errorf("front material: expected white-oak, got %q", frontBd.Material.Species)
	}

	bottom := g.Lookup("bottom")
	if bottom == nil {
		t.Fatal("missing 'bottom' node")
	}
	bottomBd := bottom.Data.(graph.BoardData)
	if bottomBd.Dimensions.X != 362 {
		t.Errorf("bottom length: expected 362, got %f", bottomBd.Dimensions.X)
	}

	left := g.Lookup("left")
	if left == nil {
		t.Fatal("missing 'left' node")
	}

	// Verify assembly.
	box := g.Lookup("box")
	if box == nil {
		t.Fatal("missing 'box' assembly node")
	}
	if box.Kind != graph.NodeGroup {
		t.Errorf("box: expected NodeGroup, got %s", box.Kind)
	}
	// 3 places + 1 butt-joint = 4 children
	if len(box.Children) != 4 {
		t.Errorf("box: expected 4 children, got %d", len(box.Children))
	}

	// Verify roots.
	if len(g.Roots) != 1 {
		t.Errorf("expected 1 root, got %d", len(g.Roots))
	}
	if g.Roots[0] != box.ID {
		t.Error("expected box to be the root")
	}

	// Verify join.
	joins := g.Joins()
	if len(joins) != 1 {
		t.Fatalf("expected 1 join, got %d", len(joins))
	}
	jd := joins[0].Data.(graph.JoinData)
	if jd.Kind != graph.JoinButt {
		t.Errorf("expected JoinButt, got %s", jd.Kind)
	}
	if jd.PartA != front.ID {
		t.Error("join PartA should reference 'front'")
	}
	if jd.FaceA != graph.FaceLeft {
		t.Errorf("join FaceA: expected left, got %s", jd.FaceA)
	}
	if jd.PartB != left.ID {
		t.Error("join PartB should reference 'left'")
	}
	if jd.FaceB != graph.FaceFront {
		t.Errorf("join FaceB: expected front, got %s", jd.FaceB)
	}

	// Verify fasteners.
	if len(jd.Fasteners) != 2 {
		t.Fatalf("expected 2 fasteners, got %d", len(jd.Fasteners))
	}

	for i, fid := range jd.Fasteners {
		fn := g.Get(fid)
		if fn == nil {
			t.Fatalf("fastener %d: node not found", i)
		}
		if fn.Kind != graph.NodeFastener {
			t.Errorf("fastener %d: expected NodeFastener, got %s", i, fn.Kind)
		}
		fd := fn.Data.(graph.FastenerData)
		if fd.Kind != graph.FastenerScrew {
			t.Errorf("fastener %d: expected FastenerScrew, got %s", i, fd.Kind)
		}
		if fd.Diameter != 4 {
			t.Errorf("fastener %d: expected diameter=4, got %f", i, fd.Diameter)
		}
		if fd.Length != 50 {
			t.Errorf("fastener %d: expected length=50, got %f", i, fd.Length)
		}
	}

	// Verify screw positions differ.
	f0 := g.Get(jd.Fasteners[0]).Data.(graph.FastenerData)
	f1 := g.Get(jd.Fasteners[1]).Data.(graph.FastenerData)
	if f0.Position.Y == f1.Position.Y {
		t.Error("expected screws to have different Y positions")
	}
	if f0.Position.Y != 50 {
		t.Errorf("screw 0: expected Y=50, got %f", f0.Position.Y)
	}
	if f1.Position.Y != 150 {
		t.Errorf("screw 1: expected Y=150, got %f", f1.Position.Y)
	}

	// Verify transform nodes point to correct children.
	transforms := 0
	for _, n := range g.Nodes {
		if n.Kind == graph.NodeTransform {
			transforms++
			if len(n.Children) != 1 {
				t.Errorf("transform: expected 1 child, got %d", len(n.Children))
			}
			td := n.Data.(graph.TransformData)
			if td.Translation == nil {
				t.Error("transform: expected non-nil translation")
			}
		}
	}
	if transforms != 3 {
		t.Errorf("expected 3 transforms, got %d", transforms)
	}
}

// ---------------------------------------------------------------------------
// Vec3 test
// ---------------------------------------------------------------------------

func TestVec3(t *testing.T) {
	eng := NewEngine()

	source := `
(defpart "panel" (board :length 100 :width 100 :thickness 10 :grain :z :material (material :species "plywood")))
(assembly "positioned"
  (place (part "panel") :at (vec3 10.5 20.3 30.7)))
`
	g, evalErrs, err := eng.Evaluate(source)
	if err != nil {
		t.Fatalf("fatal error: %v", err)
	}
	if len(evalErrs) > 0 {
		t.Fatalf("eval errors: %v", evalErrs)
	}

	// Find the transform node.
	for _, n := range g.Nodes {
		if n.Kind == graph.NodeTransform {
			td := n.Data.(graph.TransformData)
			if td.Translation == nil {
				t.Fatal("expected non-nil translation")
			}
			if td.Translation.X != 10.5 {
				t.Errorf("expected X=10.5, got %f", td.Translation.X)
			}
			if td.Translation.Y != 20.3 {
				t.Errorf("expected Y=20.3, got %f", td.Translation.Y)
			}
			if td.Translation.Z != 30.7 {
				t.Errorf("expected Z=30.7, got %f", td.Translation.Z)
			}
			return
		}
	}
	t.Fatal("no transform node found")
}

// ---------------------------------------------------------------------------
// Material with optional fields test
// ---------------------------------------------------------------------------

func TestMaterialOptionalFields(t *testing.T) {
	eng := NewEngine()

	source := `
(defpart "premium"
  (board :length 100 :width 100 :thickness 25
         :grain :z
         :material (material :species "walnut" :thickness 25.4 :grade "FAS")))
`
	g, evalErrs, err := eng.Evaluate(source)
	if err != nil {
		t.Fatalf("fatal error: %v", err)
	}
	if len(evalErrs) > 0 {
		t.Fatalf("eval errors: %v", evalErrs)
	}

	n := g.Lookup("premium")
	if n == nil {
		t.Fatal("expected node named 'premium'")
	}
	bd := n.Data.(graph.BoardData)
	if bd.Material.Species != "walnut" {
		t.Errorf("expected species=walnut, got %q", bd.Material.Species)
	}
	if bd.Material.Thickness != 25.4 {
		t.Errorf("expected material thickness=25.4, got %f", bd.Material.Thickness)
	}
	if bd.Material.Grade != "FAS" {
		t.Errorf("expected grade=FAS, got %q", bd.Material.Grade)
	}
}

// ---------------------------------------------------------------------------
// Screw with head-dia test
// ---------------------------------------------------------------------------

func TestScrewWithHeadDia(t *testing.T) {
	eng := NewEngine()

	source := `
(def oak (material :species "oak"))
(defpart "a" (board :length 100 :width 100 :thickness 19 :grain :z :material oak))
(defpart "b" (board :length 100 :width 100 :thickness 19 :grain :z :material oak))
(assembly "pair"
  (place (part "a") :at (vec3 0 0 0))
  (place (part "b") :at (vec3 0 0 19))
  (butt-joint
    :part-a (part "a") :face-a :top
    :part-b (part "b") :face-b :bottom
    :fasteners (list
      (screw :diameter 5 :length 40 :position (vec3 50 50 0) :head-dia 10))))
`
	g, evalErrs, err := eng.Evaluate(source)
	if err != nil {
		t.Fatalf("fatal error: %v", err)
	}
	if len(evalErrs) > 0 {
		t.Fatalf("eval errors: %v", evalErrs)
	}

	// Find fastener node.
	for _, n := range g.Nodes {
		if n.Kind == graph.NodeFastener {
			fd := n.Data.(graph.FastenerData)
			if fd.HeadDia != 10 {
				t.Errorf("expected head-dia=10, got %f", fd.HeadDia)
			}
			if fd.Diameter != 5 {
				t.Errorf("expected diameter=5, got %f", fd.Diameter)
			}
			return
		}
	}
	t.Fatal("no fastener node found")
}

// ---------------------------------------------------------------------------
// Empty source produces empty graph (regression)
// ---------------------------------------------------------------------------

func TestEmptySourceStillWorks(t *testing.T) {
	eng := NewEngine()
	g, evalErrs, err := eng.Evaluate("")
	if err != nil {
		t.Fatalf("fatal error: %v", err)
	}
	if len(evalErrs) > 0 {
		t.Fatalf("eval errors: %v", evalErrs)
	}
	if g == nil {
		t.Fatal("expected non-nil graph")
	}
	if g.NodeCount() != 0 {
		t.Errorf("expected empty graph, got %d nodes", g.NodeCount())
	}
}

// ---------------------------------------------------------------------------
// Plain arithmetic still works (regression)
// ---------------------------------------------------------------------------

func TestArithmeticStillWorks(t *testing.T) {
	eng := NewEngine()
	g, evalErrs, err := eng.Evaluate("(+ 1 2)")
	if err != nil {
		t.Fatalf("fatal error: %v", err)
	}
	if len(evalErrs) > 0 {
		t.Fatalf("eval errors: %v", evalErrs)
	}
	if g == nil {
		t.Fatal("expected non-nil graph")
	}
}
