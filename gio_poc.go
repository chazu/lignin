package main

import (
	"gioui.org/app"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/widget/material"
	"image/color"
)

func main() {
	go func() {
		w := app.NewWindow(
			app.Title("Lignin CAD - Gio PoC"),
			app.Size(800, 600),
		)
		
		th := material.NewTheme()
		
		var ops op.Ops
		for {
			e := <-w.Events()
			switch e := e.(type) {
			case system.DestroyEvent:
				return
			case system.FrameEvent:
				gtx := layout.NewContext(&ops, e)
				
				// Draw split pane
				layout.Flex{
					Axis: layout.Vertical,
				}.Layout(gtx,
					// Top: 3D viewport placeholder (50%)
					layout.Flexed(0.5, func(gtx layout.Context) layout.Dimensions {
						// Draw viewport background
						viewportColor := color.NRGBA{R: 40, G: 40, B: 60, A: 255}
						paint.FillShape(gtx.Ops, viewportColor, clip.Rect{
							Max: gtx.Constraints.Max,
						}.Op())
						
						// Draw placeholder text
						viewportLabel := material.Body1(th, "3D Viewport Placeholder\n• OpenGL integration needed\n• Geometry rendering\n• Interactive visualization")
						viewportLabel.Color = color.NRGBA{R: 200, G: 200, B: 220, A: 255}
						return viewportLabel.Layout(gtx)
					}),
					
					// Bottom: Code editor placeholder (50%)
					layout.Flexed(0.5, func(gtx layout.Context) layout.Dimensions {
						// Draw editor background
						editorColor := color.NRGBA{R: 30, G: 30, B: 40, A: 255}
						paint.FillShape(gtx.Ops, editorColor, clip.Rect{
							Max: gtx.Constraints.Max,
						}.Op())
						
						// Draw placeholder text
						editorLabel := material.Body1(th, "// Lignin code editor placeholder\n// Syntax highlighting, line numbers would go here\n\n(defpart box [width height depth]\n  (solid (cube width height depth)))")
						editorLabel.Color = color.NRGBA{R: 180, G: 220, B: 180, A: 255}
						return editorLabel.Layout(gtx)
					}),
				)
				
				e.Frame(gtx.Ops)
			}
		}
	}()
	
	app.Main()
}
