package main

import (
	"gioui.org/app"
	"gioui.org/op"
	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget/material"
	"gioui.org/widget"
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
						// Draw viewport background
						viewportColor := color.NRGBA{R: 40, G: 40, B: 60, A: 255}
						viewportWidget := func(gtx layout.Context) layout.Dimensions {
							return widget.Rect{
								Color: viewportColor,
								Size:  gtx.Constraints.Max,
							}.Layout(gtx)
						}
						viewport := layout.Stack{}.Layout(gtx,
							layout.Expanded(viewportWidget),
							layout.Stacked(func(gtx layout.Context) layout.Dimensions {
								viewportLabel := material.Body1(th, "3D Viewport Placeholder\n• OpenGL integration needed\n• Geometry rendering\n• Interactive visualization")
								viewportLabel.Color = color.NRGBA{R: 200, G: 200, B: 220, A: 255}
								return viewportLabel.Layout(gtx)
							}),
						)
						return viewport
					}),
					
					// Bottom: Code editor placeholder (50%)
					layout.Flexed(0.5, func(gtx layout.Context) layout.Dimensions {
						// Draw editor background
						editorColor := color.NRGBA{R: 30, G: 30, B: 40, A: 255}
						editorWidget := func(gtx layout.Context) layout.Dimensions {
							return widget.Rect{
								Color: editorColor,
								Size:  gtx.Constraints.Max,
							}.Layout(gtx)
						}
						editor := layout.Stack{}.Layout(gtx,
							layout.Expanded(editorWidget),
							layout.Stacked(func(gtx layout.Context) layout.Dimensions {
								editorLabel := material.Body1(th, "// Lignin code editor placeholder\n// Syntax highlighting, line numbers would go here\n\n(defpart box [width height depth]\n  (solid (cube width height depth)))")
								editorLabel.Color = color.NRGBA{R: 180, G: 220, B: 180, A: 255}
								return editorLabel.Layout(gtx)
							}),
						)
						return editor
					}),
				)
				
				e.Frame(gtx.Ops)
			}
		}
	}()
	
	app.Main()
}
