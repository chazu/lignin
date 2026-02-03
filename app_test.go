package main

import (
	"os"
	"testing"
)

// TestE2EBoxExample exercises the full pipeline: Lisp source → engine → graph
// → tessellate → meshes. This is the same path that the Wails Evaluate binding
// takes, but without the Wails runtime.
func TestE2EBoxExample(t *testing.T) {
	app := NewApp()

	source, err := os.ReadFile("examples/box.lignin")
	if err != nil {
		t.Fatalf("failed to read box.lignin: %v", err)
	}

	result := app.Evaluate(string(source))

	// No errors expected.
	if len(result.Errors) > 0 {
		for _, e := range result.Errors {
			t.Errorf("eval error (line %d): %s", e.Line, e.Message)
		}
		t.FailNow()
	}

	// Expect 5 meshes: front, back, left, right, bottom.
	if len(result.Meshes) != 5 {
		t.Fatalf("expected 5 meshes, got %d", len(result.Meshes))
	}

	expectedParts := map[string]bool{
		"front":  false,
		"back":   false,
		"left":   false,
		"right":  false,
		"bottom": false,
	}

	for _, m := range result.Meshes {
		if _, ok := expectedParts[m.PartName]; !ok {
			t.Errorf("unexpected part name: %q", m.PartName)
			continue
		}
		expectedParts[m.PartName] = true

		// Each mesh must have non-empty geometry.
		if len(m.Vertices) == 0 {
			t.Errorf("part %q: no vertices", m.PartName)
		}
		if len(m.Normals) == 0 {
			t.Errorf("part %q: no normals", m.PartName)
		}
		if len(m.Indices) == 0 {
			t.Errorf("part %q: no indices", m.PartName)
		}

		// Must have a color assigned.
		if m.Color == "" {
			t.Errorf("part %q: no color assigned", m.PartName)
		}
	}

	for name, found := range expectedParts {
		if !found {
			t.Errorf("missing mesh for part %q", name)
		}
	}
}

// TestE2EEmptySource ensures the pipeline handles empty input gracefully.
func TestE2EEmptySource(t *testing.T) {
	app := NewApp()
	result := app.Evaluate("")

	if len(result.Errors) > 0 {
		t.Errorf("unexpected errors for empty source: %v", result.Errors)
	}
	if len(result.Meshes) != 0 {
		t.Errorf("expected 0 meshes for empty source, got %d", len(result.Meshes))
	}
}

// TestE2ESyntaxError ensures eval errors are reported, not fatal errors.
func TestE2ESyntaxError(t *testing.T) {
	app := NewApp()
	result := app.Evaluate("(defpart \"test\"")

	if len(result.Errors) == 0 {
		t.Fatal("expected eval errors for syntax error")
	}
	if len(result.Meshes) != 0 {
		t.Errorf("expected 0 meshes on error, got %d", len(result.Meshes))
	}
}

// TestE2ESingleBoard ensures a minimal single-board source renders one mesh.
func TestE2ESingleBoard(t *testing.T) {
	app := NewApp()
	source := `(defpart "shelf" (board :length 600 :width 300 :thickness 18 :grain :x))`
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
	if result.Meshes[0].PartName != "shelf" {
		t.Errorf("expected part name 'shelf', got %q", result.Meshes[0].PartName)
	}
}
