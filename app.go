package main

import (
	"context"
	"log"
	"os"

	"github.com/chazu/lignin/pkg/engine"
	"github.com/chazu/lignin/pkg/kernel"
	"github.com/chazu/lignin/pkg/kernel/sdfx"
	"github.com/chazu/lignin/pkg/tessellate"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// colorPalette is a default palette used to assign distinct colors to parts.
var colorPalette = []string{
	"#4A90D9", "#E67E22", "#2ECC71", "#9B59B6",
	"#E74C3C", "#1ABC9C", "#F39C12", "#3498DB",
}

// App is the Wails backend. It exposes methods to the frontend via bindings.
type App struct {
	ctx    context.Context
	engine *engine.Engine
	kernel kernel.Kernel
}

// MeshData is the JSON-serializable mesh format sent to the frontend.
type MeshData struct {
	Vertices []float32 `json:"vertices"`
	Normals  []float32 `json:"normals"`
	Indices  []uint32  `json:"indices"`
	PartName string    `json:"partName"`
	Color    string    `json:"color"`
}

// EvalErrorData is a JSON-serializable eval error for the frontend.
type EvalErrorData struct {
	Line    int    `json:"line"`
	Col     int    `json:"col"`
	Message string `json:"message"`
}

// EvalResult is the full result returned to the frontend.
type EvalResult struct {
	Meshes   []MeshData      `json:"meshes"`
	Errors   []EvalErrorData `json:"errors"`
	Warnings []EvalErrorData `json:"warnings"`
}

// FileResult is returned by OpenFile with the file contents and path.
type FileResult struct {
	Content string `json:"content"`
	Path    string `json:"path"`
}

// NewApp creates a new App with an engine and the sdfx kernel.
func NewApp() *App {
	return &App{
		engine: engine.NewEngine(),
		kernel: sdfx.New(),
	}
}

// startup is called by Wails on app startup. The context is saved
// so we can call Wails runtime methods later if needed.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// Evaluate takes Lisp source and returns mesh data + errors.
// This is the primary binding called by the frontend editor.
func (a *App) Evaluate(source string) EvalResult {
	result := EvalResult{
		Meshes:   []MeshData{},
		Errors:   []EvalErrorData{},
		Warnings: []EvalErrorData{},
	}

	// Step 1: Evaluate the Lisp source into a design graph.
	g, evalErrs, err := a.engine.Evaluate(source)
	if err != nil {
		// Fatal error (panic, timeout, etc.)
		log.Printf("Evaluate fatal error: %v", err)
		result.Errors = append(result.Errors, EvalErrorData{
			Line:    0,
			Col:     0,
			Message: err.Error(),
		})
		return result
	}

	// Step 2: Convert eval errors to the frontend format.
	if len(evalErrs) > 0 {
		for _, e := range evalErrs {
			result.Errors = append(result.Errors, EvalErrorData{
				Line:    e.Line,
				Col:     e.Col,
				Message: e.Message,
			})
		}
		return result
	}

	// Step 3: Tessellate the design graph into triangle meshes.
	meshes, err := tessellate.Tessellate(g, a.kernel)
	if err != nil {
		log.Printf("Tessellate error: %v", err)
		result.Errors = append(result.Errors, EvalErrorData{
			Line:    0,
			Col:     0,
			Message: "tessellation failed: " + err.Error(),
		})
		return result
	}

	// Step 4: Convert kernel meshes to the frontend MeshData format.
	for i, m := range meshes {
		color := colorPalette[i%len(colorPalette)]
		result.Meshes = append(result.Meshes, MeshData{
			Vertices: m.Vertices,
			Normals:  m.Normals,
			Indices:  m.Indices,
			PartName: m.PartName,
			Color:    color,
		})
	}

	return result
}

// ligninFileFilter is the dialog filter for .lignin files.
var ligninFileFilter = runtime.FileFilter{
	DisplayName: "Lignin Files (*.lignin)",
	Pattern:     "*.lignin",
}

// OpenFile shows an open file dialog and returns the file contents + path.
func (a *App) OpenFile() (FileResult, error) {
	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Open Lignin File",
		Filters: []runtime.FileFilter{
			ligninFileFilter,
		},
	})
	if err != nil {
		return FileResult{}, err
	}
	// User cancelled the dialog.
	if path == "" {
		return FileResult{}, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return FileResult{}, err
	}
	return FileResult{
		Content: string(data),
		Path:    path,
	}, nil
}

// SaveFile saves content to the given path (or shows a save dialog if path is empty).
func (a *App) SaveFile(content string, path string) (string, error) {
	if path == "" {
		var err error
		path, err = runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
			Title:           "Save Lignin File",
			DefaultFilename: "untitled.lignin",
			Filters: []runtime.FileFilter{
				ligninFileFilter,
			},
		})
		if err != nil {
			return "", err
		}
		// User cancelled the dialog.
		if path == "" {
			return "", nil
		}
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return "", err
	}
	return path, nil
}

// SetTitle updates the window title.
func (a *App) SetTitle(title string) {
	runtime.WindowSetTitle(a.ctx, title)
}
