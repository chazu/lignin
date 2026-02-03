package main

import (
	"gioui.org/app"
	"gioui.org/op"
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget/material"
	"image/color"
)

func main() {
	go func() {
		w := new(app.Window)
		w.Option(app.Title("Lignin CAD - Gio PoC"))
		w.Option(app.Size(unit.Dp(800), unit.Dp(600)))
		
		th := material.NewTheme()
		var ops op.Ops
		
		for {
			e := w.Event()
			switch e := e.(type) {
			case app.DestroyEvent:
				return
			case app.FrameEvent:
				gtx := app.NewContext(&ops, e)
				
				// Draw split pane
				layout.Flex{
					Axis: layout.Vertical,
				}.Layout(gtx,
					// Top: 3D viewport placeholder (50%)
					layout.Flexed(0.5, func(gtx layout.Context) layout.Dimensions {
						// Simple colored area with text
						viewportLabel := material.Body1(th, "3D Viewport Placeholder\n• OpenGL integration needed\n• Geometry rendering\n• Interactive visualization")
						viewportLabel.Color = color.NRGBA{R: 200, G: 200, B: 220, A: 255}
						return viewportLabel.Layout(gtx)
					}),
					
					// Bottom: Code editor placeholder (50%)
					layout.Flexed(0.5, func(gtx layout.Context) layout.Dimensions {
						// Simple colored area with text
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
