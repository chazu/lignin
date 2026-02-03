package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

func main() {
	// Create app
	a := app.New()
	w := a.NewWindow("Lignin CAD - Fyne PoC")
	w.Resize(fyne.NewSize(800, 600))

	// Create editor placeholder (text area)
	editor := widget.NewMultiLineEntry()
	editor.SetPlaceHolder("// Lignin code editor placeholder\n// Syntax highlighting, line numbers would go here\n\n(defpart box [width height depth]\n  (solid (cube width height depth)))")
	editor.SetMinRowsVisible(15)

	// Create 3D viewport placeholder
	viewport := widget.NewLabel("3D Viewport Placeholder\n\n• OpenGL integration needed\n• Geometry rendering\n• Interactive visualization")
	viewport.Alignment = fyne.TextAlignCenter

	// Create split pane (editor on bottom, viewport on top)
	split := container.NewVSplit(
		container.NewBorder(nil, nil, nil, nil, viewport), // Top
		container.NewBorder(nil, nil, nil, nil, editor),   // Bottom
	)
	split.SetOffset(0.5) // Start at 50/50 split

	// Set window content
	w.SetContent(split)
	w.ShowAndRun()
}
