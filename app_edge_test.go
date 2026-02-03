package main

import (
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// 1. Empty editor: empty string -> 0 meshes, 0 errors.
//    (TestE2EEmptySource already exists; this verifies additional invariants.)
// ---------------------------------------------------------------------------

func TestE2EEmptySourceExtended(t *testing.T) {
	app := NewApp()
	result := app.Evaluate("")

	if len(result.Errors) != 0 {
		t.Errorf("expected 0 errors for empty source, got %d", len(result.Errors))
	}
	if len(result.Meshes) != 0 {
		t.Errorf("expected 0 meshes for empty source, got %d", len(result.Meshes))
	}
	if len(result.Warnings) != 0 {
		t.Errorf("expected 0 warnings for empty source, got %d", len(result.Warnings))
	}
	// Ensure slices are non-nil (JSON should serialize as [] not null).
	if result.Meshes == nil {
		t.Error("Meshes should be non-nil empty slice, got nil")
	}
	if result.Errors == nil {
		t.Error("Errors should be non-nil empty slice, got nil")
	}
	if result.Warnings == nil {
		t.Error("Warnings should be non-nil empty slice, got nil")
	}
}

// ---------------------------------------------------------------------------
// 2. Syntax error mid-expression: unmatched parens -> eval error, 0 meshes.
//    Extends TestE2ESyntaxError to verify error has line > 0 or a message.
// ---------------------------------------------------------------------------

func TestE2ESyntaxErrorWithLineInfo(t *testing.T) {
	app := NewApp()

	// Put valid code on line 1, broken code on line 2 so line info is meaningful.
	source := "(+ 1 2)\n(defpart \"test\""
	result := app.Evaluate(source)

	if len(result.Errors) == 0 {
		t.Fatal("expected at least one eval error for unmatched parens")
	}
	if len(result.Meshes) != 0 {
		t.Errorf("expected 0 meshes on syntax error, got %d", len(result.Meshes))
	}

	// Verify the error has a non-empty message.
	e := result.Errors[0]
	if e.Message == "" {
		t.Error("syntax error should have a non-empty message")
	}

	// The error should ideally have line info > 0 (line 2+).
	// We log regardless, but assert message is present.
	t.Logf("syntax error: line=%d, col=%d, message=%q", e.Line, e.Col, e.Message)
}

func TestE2ESyntaxErrorSingleLineMissingParen(t *testing.T) {
	app := NewApp()

	result := app.Evaluate("(+ 1 2")

	if len(result.Errors) == 0 {
		t.Fatal("expected eval error for missing closing paren")
	}
	if len(result.Meshes) != 0 {
		t.Errorf("expected 0 meshes, got %d", len(result.Meshes))
	}

	e := result.Errors[0]
	if e.Message == "" {
		t.Error("error message should not be empty")
	}
}

// ---------------------------------------------------------------------------
// 3. Undefined part reference: (part "nonexistent") in assembly -> eval error.
// ---------------------------------------------------------------------------

func TestE2EUndefinedPartReference(t *testing.T) {
	app := NewApp()

	source := `
(defpart "shelf"
  (board :length 600 :width 300 :thickness 18 :grain :x))

(assembly "unit"
  (place (part "nonexistent") :at (vec3 0 0 0)))
`
	result := app.Evaluate(source)

	if len(result.Errors) == 0 {
		t.Fatal("expected eval error for undefined part reference")
	}

	// The error should mention the missing part name.
	found := false
	for _, e := range result.Errors {
		if strings.Contains(e.Message, "nonexistent") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected error mentioning 'nonexistent', got: %v", result.Errors)
	}

	if len(result.Meshes) != 0 {
		t.Errorf("expected 0 meshes on error, got %d", len(result.Meshes))
	}
}

func TestE2EUndefinedPartReferenceStandalone(t *testing.T) {
	app := NewApp()

	// Standalone part reference without any defpart.
	source := `(part "ghost")`
	result := app.Evaluate(source)

	if len(result.Errors) == 0 {
		t.Fatal("expected eval error for referencing undefined part")
	}

	found := false
	for _, e := range result.Errors {
		if strings.Contains(e.Message, "ghost") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected error mentioning 'ghost', got: %v", result.Errors)
	}
}

// ---------------------------------------------------------------------------
// 4. Zero-dimension board: board with length=0 -> error or degenerate mesh.
// ---------------------------------------------------------------------------

func TestE2EZeroDimensionBoard(t *testing.T) {
	app := NewApp()

	source := `(defpart "bad" (board :length 0 :width 100 :thickness 19 :grain :x))`
	result := app.Evaluate(source)

	// The system should either produce an error or produce a (possibly empty)
	// mesh without panicking. Either outcome is acceptable; panicking is not.
	if len(result.Errors) > 0 {
		t.Logf("zero-dimension board produced error (acceptable): %s", result.Errors[0].Message)
		return
	}

	// If no error, the mesh may exist but possibly be empty/degenerate.
	t.Logf("zero-dimension board produced %d meshes (no error)", len(result.Meshes))
}

func TestE2EAllZeroDimensions(t *testing.T) {
	app := NewApp()

	source := `(defpart "void" (board :length 0 :width 0 :thickness 0 :grain :x))`
	result := app.Evaluate(source)

	// Must not panic. Error or empty mesh are both acceptable.
	if len(result.Errors) > 0 {
		t.Logf("all-zero dimensions produced error (acceptable): %s", result.Errors[0].Message)
		return
	}

	t.Logf("all-zero dimensions produced %d meshes (no error)", len(result.Meshes))
}

func TestE2ENegativeDimension(t *testing.T) {
	app := NewApp()

	source := `(defpart "negative" (board :length -100 :width 100 :thickness 19 :grain :x))`
	result := app.Evaluate(source)

	// Must not panic. Error or mesh are both acceptable.
	if len(result.Errors) > 0 {
		t.Logf("negative dimension produced error (acceptable): %s", result.Errors[0].Message)
		return
	}

	t.Logf("negative dimension produced %d meshes (no error)", len(result.Meshes))
}

// ---------------------------------------------------------------------------
// 5. Rapid evaluation (debounce simulation): no panics, no data races.
//    Run with `go test -race` to detect data races.
// ---------------------------------------------------------------------------

func TestE2ERapidEvaluation(t *testing.T) {
	// Simulates debounce: rapid sequential calls to Evaluate on the same App.
	// The engine holds a mutex, so rapid sequential calls exercise the
	// generation-counter and timeout paths. We verify no panics occur.
	//
	// Note: we call Evaluate sequentially because zygomys has internal
	// global state that is not safe for concurrent sandbox creation.
	// In production, the engine mutex serializes calls anyway.
	app := NewApp()

	sources := []string{
		`(defpart "a" (board :length 100 :width 50 :thickness 10 :grain :x))`,
		`(defpart "b" (board :length 200 :width 100 :thickness 20 :grain :y))`,
		`(+ 1 2)`,
		``,
		`(defpart "c" (board :length 300 :width 150 :thickness 30 :grain :z))`,
		`(defpart "d" (board :length 400 :width 200 :thickness 18 :grain :x))`,
		`(+ 100 200)`,
		``,
		`(defpart "e" (board :length 500 :width 250 :thickness 25 :grain :z))`,
		`(defpart "f" (board :length 600 :width 300 :thickness 18 :grain :x))`,
	}

	for i, source := range sources {
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("iteration %d panicked: %v", i, r)
				}
			}()
			result := app.Evaluate(source)
			// Just ensure no panic. Results vary by source.
			_ = result
		}()
	}
}

func TestE2ERapidEvaluationAlternating(t *testing.T) {
	// Alternates between valid and invalid sources rapidly.
	// Ensures the engine recovers cleanly between error and success states.
	app := NewApp()

	sources := []string{
		`(defpart "ok" (board :length 100 :width 50 :thickness 10 :grain :x))`,
		`(defpart "broken"`,
		``,
		`(part "missing")`,
		`(defpart "also-ok" (board :length 200 :width 100 :thickness 20 :grain :y))`,
		`(+ 1 2)`,
		`;; just a comment`,
		`(defpart "fine" (board :length 300 :width 150 :thickness 30 :grain :z))`,
		`(undefined-func 1 2 3)`,
		`(defpart "last" (board :length 400 :width 200 :thickness 18 :grain :x))`,
	}

	for i, source := range sources {
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("iteration %d panicked on source %q: %v", i, source, r)
				}
			}()
			result := app.Evaluate(source)
			_ = result
		}()
	}
}

// ---------------------------------------------------------------------------
// 6. Large dimensions: very large board -> valid mesh without crash.
// ---------------------------------------------------------------------------

func TestE2ELargeDimensions(t *testing.T) {
	app := NewApp()

	source := `(defpart "huge" (board :length 10000 :width 10000 :thickness 19 :grain :x))`
	result := app.Evaluate(source)

	if len(result.Errors) > 0 {
		t.Fatalf("unexpected errors for large board: %v", result.Errors)
	}
	if len(result.Meshes) != 1 {
		t.Fatalf("expected 1 mesh for large board, got %d", len(result.Meshes))
	}

	m := result.Meshes[0]
	if len(m.Vertices) == 0 {
		t.Error("large board mesh should have vertices")
	}
	if len(m.Normals) == 0 {
		t.Error("large board mesh should have normals")
	}
	if len(m.Indices) == 0 {
		t.Error("large board mesh should have indices")
	}
	if m.PartName != "huge" {
		t.Errorf("expected part name 'huge', got %q", m.PartName)
	}
}

func TestE2EVeryLargeDimensions(t *testing.T) {
	app := NewApp()

	// 100,000 mm = 100 meters. Extreme but should not crash.
	source := `(defpart "giant" (board :length 100000 :width 50000 :thickness 100 :grain :z))`
	result := app.Evaluate(source)

	if len(result.Errors) > 0 {
		// An error for extreme dimensions is acceptable.
		t.Logf("very large dimensions produced error (acceptable): %s", result.Errors[0].Message)
		return
	}
	if len(result.Meshes) != 1 {
		t.Fatalf("expected 1 mesh, got %d", len(result.Meshes))
	}
	if len(result.Meshes[0].Vertices) == 0 {
		t.Error("mesh should have vertices")
	}
}

// ---------------------------------------------------------------------------
// 7. Multiple assemblies: two assemblies in one source -> meshes from both.
// ---------------------------------------------------------------------------

func TestE2EMultipleAssemblies(t *testing.T) {
	app := NewApp()

	source := `
(def oak (material :species "white-oak"))

(defpart "shelf-a"
  (board :length 600 :width 300 :thickness 18 :grain :x :material oak))

(defpart "shelf-b"
  (board :length 400 :width 200 :thickness 18 :grain :x :material oak))

(assembly "unit-a"
  (place (part "shelf-a") :at (vec3 0 0 0)))

(assembly "unit-b"
  (place (part "shelf-b") :at (vec3 700 0 0)))
`
	result := app.Evaluate(source)

	if len(result.Errors) > 0 {
		for _, e := range result.Errors {
			t.Errorf("eval error: %s", e.Message)
		}
		t.FailNow()
	}

	// Two assemblies, each with one part -> 2 meshes.
	if len(result.Meshes) != 2 {
		t.Fatalf("expected 2 meshes from two assemblies, got %d", len(result.Meshes))
	}

	names := make(map[string]bool)
	for _, m := range result.Meshes {
		names[m.PartName] = true
		if len(m.Vertices) == 0 {
			t.Errorf("mesh %q should have vertices", m.PartName)
		}
		if m.Color == "" {
			t.Errorf("mesh %q should have a color assigned", m.PartName)
		}
	}

	if !names["shelf-a"] {
		t.Error("missing mesh for shelf-a")
	}
	if !names["shelf-b"] {
		t.Error("missing mesh for shelf-b")
	}
}

func TestE2EMultipleAssembliesWithSharedParts(t *testing.T) {
	app := NewApp()

	source := `
(def oak (material :species "white-oak"))

(defpart "panel"
  (board :length 300 :width 200 :thickness 18 :grain :x :material oak))

(defpart "rail"
  (board :length 300 :width 50 :thickness 18 :grain :x :material oak))

(assembly "frame-a"
  (place (part "panel") :at (vec3 0 0 0))
  (place (part "rail")  :at (vec3 0 200 0)))

(assembly "frame-b"
  (place (part "panel") :at (vec3 500 0 0))
  (place (part "rail")  :at (vec3 500 200 0)))
`
	result := app.Evaluate(source)

	if len(result.Errors) > 0 {
		for _, e := range result.Errors {
			t.Errorf("eval error: %s", e.Message)
		}
		t.FailNow()
	}

	// Two assemblies, each referencing the same 2 parts.
	// Each assembly places 2 parts, so expect 4 meshes total.
	if len(result.Meshes) != 4 {
		t.Fatalf("expected 4 meshes from two assemblies sharing parts, got %d", len(result.Meshes))
	}
}

// ---------------------------------------------------------------------------
// 8. Part with only defpart, no assembly: standalone defpart -> 1 mesh
//    (tessellator fallback for no-roots graphs).
// ---------------------------------------------------------------------------

func TestE2EStandaloneDefpart(t *testing.T) {
	app := NewApp()

	source := `(defpart "shelf" (board :length 600 :width 300 :thickness 18 :grain :x))`
	result := app.Evaluate(source)

	if len(result.Errors) > 0 {
		for _, e := range result.Errors {
			t.Errorf("eval error: %s", e.Message)
		}
		t.FailNow()
	}

	// No assembly means no roots, so tessellator falls back to all primitives.
	if len(result.Meshes) != 1 {
		t.Fatalf("expected 1 mesh from standalone defpart, got %d", len(result.Meshes))
	}
	if result.Meshes[0].PartName != "shelf" {
		t.Errorf("expected part name 'shelf', got %q", result.Meshes[0].PartName)
	}
	if len(result.Meshes[0].Vertices) == 0 {
		t.Error("standalone defpart mesh should have vertices")
	}
}

func TestE2EMultipleStandaloneDefparts(t *testing.T) {
	app := NewApp()

	source := `
(defpart "top" (board :length 600 :width 300 :thickness 18 :grain :x))
(defpart "bottom" (board :length 600 :width 300 :thickness 18 :grain :x))
`
	result := app.Evaluate(source)

	if len(result.Errors) > 0 {
		for _, e := range result.Errors {
			t.Errorf("eval error: %s", e.Message)
		}
		t.FailNow()
	}

	// Two standalone defparts, no assembly -> tessellator produces 2 meshes.
	if len(result.Meshes) != 2 {
		t.Fatalf("expected 2 meshes from two standalone defparts, got %d", len(result.Meshes))
	}

	names := make(map[string]bool)
	for _, m := range result.Meshes {
		names[m.PartName] = true
	}
	if !names["top"] {
		t.Error("missing mesh for 'top'")
	}
	if !names["bottom"] {
		t.Error("missing mesh for 'bottom'")
	}
}

// ---------------------------------------------------------------------------
// 9. Comments only: source that is only comments -> 0 meshes, 0 errors.
// ---------------------------------------------------------------------------

func TestE2ECommentsOnly(t *testing.T) {
	app := NewApp()

	source := `
;; This is a comment
;; Another comment
; And another
`
	result := app.Evaluate(source)

	if len(result.Errors) > 0 {
		t.Errorf("unexpected errors for comments-only source: %v", result.Errors)
	}
	if len(result.Meshes) != 0 {
		t.Errorf("expected 0 meshes for comments-only source, got %d", len(result.Meshes))
	}
}

func TestE2ECommentsWithWhitespace(t *testing.T) {
	app := NewApp()

	source := `
  ;; leading whitespace
  ;; trailing whitespace
  ; tabs	everywhere
`
	result := app.Evaluate(source)

	if len(result.Errors) > 0 {
		t.Errorf("unexpected errors for comments+whitespace source: %v", result.Errors)
	}
	if len(result.Meshes) != 0 {
		t.Errorf("expected 0 meshes, got %d", len(result.Meshes))
	}
}

// ---------------------------------------------------------------------------
// 10. Nested expressions: def with arithmetic, then use in board.
// ---------------------------------------------------------------------------

func TestE2ENestedArithmeticDef(t *testing.T) {
	app := NewApp()

	source := `
(def w (* 2 150))
(defpart "wide-shelf"
  (board :length w :width 200 :thickness 18 :grain :x))
`
	result := app.Evaluate(source)

	if len(result.Errors) > 0 {
		for _, e := range result.Errors {
			t.Errorf("eval error: %s", e.Message)
		}
		t.FailNow()
	}

	if len(result.Meshes) != 1 {
		t.Fatalf("expected 1 mesh, got %d", len(result.Meshes))
	}
	if result.Meshes[0].PartName != "wide-shelf" {
		t.Errorf("expected part name 'wide-shelf', got %q", result.Meshes[0].PartName)
	}
	if len(result.Meshes[0].Vertices) == 0 {
		t.Error("mesh should have vertices")
	}
}

func TestE2EComplexArithmeticExpressions(t *testing.T) {
	app := NewApp()

	source := `
(def base-length 400)
(def margin 19)
(def inner-length (- base-length (* 2 margin)))
(def thickness 19)

(defpart "inner-panel"
  (board :length inner-length :width 200 :thickness thickness :grain :x))
`
	result := app.Evaluate(source)

	if len(result.Errors) > 0 {
		for _, e := range result.Errors {
			t.Errorf("eval error: %s", e.Message)
		}
		t.FailNow()
	}

	if len(result.Meshes) != 1 {
		t.Fatalf("expected 1 mesh, got %d", len(result.Meshes))
	}

	// inner-length = 400 - 2*19 = 362. The mesh should have non-empty geometry.
	if len(result.Meshes[0].Vertices) == 0 {
		t.Error("mesh should have vertices for computed dimensions")
	}
}

func TestE2ENestedDefWithDivision(t *testing.T) {
	app := NewApp()

	source := `
(def total 600)
(def half (/ total 2))
(defpart "half-shelf"
  (board :length half :width 200 :thickness 18 :grain :x))
`
	result := app.Evaluate(source)

	if len(result.Errors) > 0 {
		for _, e := range result.Errors {
			t.Errorf("eval error: %s", e.Message)
		}
		t.FailNow()
	}

	if len(result.Meshes) != 1 {
		t.Fatalf("expected 1 mesh, got %d", len(result.Meshes))
	}
}

// ---------------------------------------------------------------------------
// Additional edge cases
// ---------------------------------------------------------------------------

func TestE2EWhitespaceOnly(t *testing.T) {
	app := NewApp()
	result := app.Evaluate("   \n\t\n   \n")

	if len(result.Errors) != 0 {
		t.Errorf("expected 0 errors for whitespace-only source, got %d", len(result.Errors))
	}
	if len(result.Meshes) != 0 {
		t.Errorf("expected 0 meshes for whitespace-only source, got %d", len(result.Meshes))
	}
}

func TestE2EDefpartMissingBody(t *testing.T) {
	app := NewApp()

	// defpart with name but no board expression.
	source := `(defpart "oops")`
	result := app.Evaluate(source)

	if len(result.Errors) == 0 {
		t.Fatal("expected eval error for defpart with no body")
	}
}

func TestE2EAssemblyNoChildren(t *testing.T) {
	app := NewApp()

	// An assembly with just a name and no place/joint children.
	source := `(assembly "empty-asm")`
	result := app.Evaluate(source)

	// Should not panic. May produce 0 meshes or an error -- both are acceptable.
	if len(result.Errors) > 0 {
		t.Logf("empty assembly produced error (acceptable): %s", result.Errors[0].Message)
		return
	}
	if len(result.Meshes) != 0 {
		t.Errorf("expected 0 meshes for empty assembly, got %d", len(result.Meshes))
	}
}

func TestE2EFloatingPointDimensions(t *testing.T) {
	app := NewApp()

	source := `(defpart "precise" (board :length 123.456 :width 78.9 :thickness 12.7 :grain :z))`
	result := app.Evaluate(source)

	if len(result.Errors) > 0 {
		for _, e := range result.Errors {
			t.Errorf("eval error: %s", e.Message)
		}
		t.FailNow()
	}

	if len(result.Meshes) != 1 {
		t.Fatalf("expected 1 mesh, got %d", len(result.Meshes))
	}
	if len(result.Meshes[0].Vertices) == 0 {
		t.Error("floating-point dimension mesh should have vertices")
	}
}

func TestE2EColorPaletteWrapping(t *testing.T) {
	app := NewApp()

	// Create more parts than the palette has colors to ensure wrapping works.
	source := `
(def oak (material :species "oak"))
(defpart "p1" (board :length 100 :width 50 :thickness 10 :grain :x :material oak))
(defpart "p2" (board :length 100 :width 50 :thickness 10 :grain :x :material oak))
(defpart "p3" (board :length 100 :width 50 :thickness 10 :grain :x :material oak))
(defpart "p4" (board :length 100 :width 50 :thickness 10 :grain :x :material oak))
(defpart "p5" (board :length 100 :width 50 :thickness 10 :grain :x :material oak))
(defpart "p6" (board :length 100 :width 50 :thickness 10 :grain :x :material oak))
(defpart "p7" (board :length 100 :width 50 :thickness 10 :grain :x :material oak))
(defpart "p8" (board :length 100 :width 50 :thickness 10 :grain :x :material oak))
(defpart "p9" (board :length 100 :width 50 :thickness 10 :grain :x :material oak))

(assembly "many"
  (place (part "p1") :at (vec3 0 0 0))
  (place (part "p2") :at (vec3 110 0 0))
  (place (part "p3") :at (vec3 220 0 0))
  (place (part "p4") :at (vec3 330 0 0))
  (place (part "p5") :at (vec3 440 0 0))
  (place (part "p6") :at (vec3 550 0 0))
  (place (part "p7") :at (vec3 660 0 0))
  (place (part "p8") :at (vec3 770 0 0))
  (place (part "p9") :at (vec3 880 0 0)))
`
	result := app.Evaluate(source)

	if len(result.Errors) > 0 {
		for _, e := range result.Errors {
			t.Errorf("eval error: %s", e.Message)
		}
		t.FailNow()
	}

	if len(result.Meshes) != 9 {
		t.Fatalf("expected 9 meshes, got %d", len(result.Meshes))
	}

	// All meshes must have a non-empty color (palette wraps around).
	for _, m := range result.Meshes {
		if m.Color == "" {
			t.Errorf("mesh %q should have a color assigned (palette wrapping)", m.PartName)
		}
	}
}

