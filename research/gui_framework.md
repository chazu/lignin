# Lignin GUI Framework Research

**Date:** 2026-02-02
**Context:** Selecting a native GUI framework for Lignin, a programmable CAD tool for woodworking written in Go. The UI is a split-pane layout: bottom half is a Lisp code editor with syntax highlighting, top half is a 3D viewport rendering solids from the design graph.

**Key PRD constraints:**
- Cross-platform: macOS, Linux (Windows nice-to-have)
- Editor-driven workflow, no direct geometry manipulation
- Live evaluation loop (rate-limited) -- editor changes trigger re-evaluation
- The renderer must never mutate design state
- Clear error and warning surfacing
- Deterministic, code-driven modeling

---

## 1. Feature Comparison Matrix

| Feature | Fyne | Gio | Wails | gotk4 | giu (imgui) | Cogent Core | Ebitengine |
|---|---|---|---|---|---|---|---|
| **Widget set** | Rich (buttons, lists, trees, tabs, forms) | Minimal (editor, buttons, lists -- build your own) | Full web ecosystem (React/Vue/Svelte) | Full GTK4 widget set | Dear ImGui widgets (buttons, inputs, trees, plots) | Rich (Material 3 style, trees, tables, editors) | None built-in (Guigui alpha) |
| **3D integration** | No public API; render-to-texture via canvas.Raster workaround (issue #129 open) | GPU package exposes OpenGL/Vulkan render targets; GLFW example exists | WebGL/Three.js in frontend webview | GtkGLArea widget -- native OpenGL embedding | Render to FBO, display as ImGui::Image -- well-documented pattern | Native xyz package with WebGPU; xyzcore embeds 3D in 2D | 2D engine only; no 3D |
| **Code editor** | No built-in editor widget; TextGrid is read-only; issue #3183 open | gvcode: third-party editor with syntax highlighting, line numbers, autocomplete | Monaco, CodeMirror, or any web editor | GtkSourceView via gotk4-sourceview: full-featured code editor | No built-in; would need custom imgui text editor | Cogent Code: full editor with syntax highlighting, parse framework | None |
| **Platform support** | macOS, Linux, Windows, iOS, Android, WASM | macOS, Linux, Windows, iOS, Android, FreeBSD, WASM | macOS, Linux, Windows | Linux (native), macOS/Windows (with GTK4 installed) | macOS, Linux, Windows | macOS, Linux, Windows, iOS, Android, WASM | macOS, Linux, Windows, mobile, WASM |
| **Maturity** | Stable (v2.x, 6+ years) | Stable-ish (v0.9.x, 5+ years, API still evolving) | Stable (v2.x, v3 alpha) | Usable but alpha-quality (v0.3.x, memory leaks possible) | Stable (v0.9.x) | Young (initial release 2024, active development) | Mature for 2D games (12 years), GUI alpha |
| **GitHub stars** | ~28,000 | ~1,800 (GitHub mirror; primary on sr.ht) | ~32,400 | ~640 | ~2,500 | ~2,300 | ~11,000 |
| **Community size** | Large (100+ contributors, Slack, Discord, conferences) | Small-medium (mailing list, dedicated sponsors) | Large (75+ contributors, Discord, growing ecosystem) | Small (primarily one maintainer) | Medium (active forks and users) | Small (academic origins, small team) | Large (game dev community) |
| **License** | BSD-3 | MIT / Unlicense | MIT | AGPL-3 (generator) / MPL-2 (generated bindings) | MIT | BSD-3 | Apache-2.0 |
| **CGo required** | Yes (OpenGL/GLFW) | Yes (platform libs) | Yes (webview) | Yes (GTK4 C libs) | Yes (GLFW/OpenGL/cimgui) | Yes (WebGPU) | Yes (platform-dependent) |

### Quick Assessment

- **Best widget ecosystem (native Go):** Fyne or Cogent Core
- **Best 3D integration story:** Cogent Core (native), gotk4 (GtkGLArea), giu (FBO pattern), Wails (WebGL)
- **Best code editor story:** gotk4+GtkSourceView (best), Cogent Core (built-in), Gio+gvcode (good), Wails+Monaco/CodeMirror (easiest)
- **Best cross-platform:** Fyne, Gio, Wails (all excellent); gotk4 weakest (GTK4 dependency on macOS)
- **Safest bet (community/maintenance):** Fyne or Wails

---

## 2. Code Editor Evaluation

The code editor is critical for Lignin. It needs: syntax highlighting for a custom Lisp dialect, line numbers, basic editing (undo/redo, selection, clipboard), and the ability to trigger re-evaluation on changes (with rate-limiting).

### Option A: Gio + gvcode

**gvcode** (github.com/oligo/gvcode) is a purpose-built code editor component for Gio.

**Features available:**
- PieceTable-backed text buffer (efficient for large files)
- Syntax highlighting via text styles (programmable -- you provide the token-to-style mapping)
- Built-in line numbers
- Bracket pair auto-completion
- Auto-indent
- Undo/redo (built into PieceTable)
- Horizontal scrolling for long lines
- Command registry for shortcuts
- Auto-completion API

**How syntax highlighting would work:**
gvcode allows applying text styles per-region. You would integrate a Lisp tokenizer (or use tree-sitter bindings) to classify tokens (keywords, strings, numbers, comments, parens) and map them to gvcode style ranges. On each edit, re-tokenize the visible region and reapply styles. This is a manual integration but straightforward for a Lisp dialect with simple syntax.

**Concerns:**
- gvcode is a work-in-progress; APIs may change
- Community adoption is small (forked by Chapar project, last update Oct 2025)
- No built-in language support -- you write the tokenizer
- Gio's immediate-mode paradigm requires understanding the frame loop

**Verdict:** Good option. The editor component exists and has the right features. The main risk is gvcode's maturity and the small community.

### Option B: gotk4 + GtkSourceView

**GtkSourceView** is a battle-tested GNOME library that powers gedit, GNOME Builder, and many other editors. Go bindings exist via `gotk4-sourceview`.

**Features available:**
- Syntax highlighting with language definition files (XML-based spec)
- Line numbers (built-in)
- Bracket matching
- Undo/redo (built-in)
- Code completion system
- Search and replace
- Vim emulation mode
- Style schemes (color themes)
- Snippets

**How syntax highlighting would work:**
Write a GtkSourceView language definition file (.lang XML) for the Lignin Lisp dialect. This is a well-documented format used across the GNOME ecosystem. Register it with the LanguageManager, create a SourceBuffer with that language, and attach it to a SourceView widget. Highlighting is automatic.

**Concerns:**
- GTK4 must be installed on the target system (non-trivial on macOS -- requires Homebrew/MacPorts)
- gotk4 is alpha quality with known memory leaks
- AGPL-3 license on the generator (MPL-2 on generated code) -- check compatibility
- Single primary maintainer (diamondburned)
- Cross-platform deployment is harder than pure-Go solutions

**Verdict:** Best code editor experience by far, but the GTK4 dependency and gotk4's immaturity introduce significant friction, especially for macOS deployment.

### Option C: Wails + Monaco Editor / CodeMirror

Using a web-based code editor in a Wails app.

**Features available (Monaco):**
- Full VS Code editor experience
- Syntax highlighting via TextMate grammars or Monarch tokenizer
- Line numbers, minimap, bracket matching
- IntelliSense / auto-completion APIs
- Undo/redo, multi-cursor, find/replace
- Extensive theming

**Features available (CodeMirror 6):**
- Lighter weight than Monaco
- Excellent Lezer-based parser for syntax highlighting
- Line numbers, bracket matching
- Extensible with plugins
- Better performance for simpler use cases

**How syntax highlighting would work:**
For Monaco: write a Monarch tokenizer definition (JSON/JS) for the Lisp dialect. For CodeMirror: write a Lezer grammar or use the existing Lisp/Clojure grammar as a starting point. Both approaches are well-documented with large communities.

**Concerns:**
- Web-based -- adds JS/HTML layer between Go backend and editor
- Communication between Go and editor goes through Wails bindings (function calls, events)
- Not a "native" feel, though modern webviews are very capable
- Memory overhead of webview (though much less than Electron)
- WebGL in webview for 3D viewport works but may have performance ceilings

**Verdict:** Easiest path to a polished code editor. The web ecosystem for code editors is vastly more mature than any native Go option. Trade-off is the web layer adds architectural complexity and a non-native feel.

### Option D: Cogent Core (Built-in Editor)

Cogent Core includes **Cogent Code**, a full code editor written in Go.

**Features available:**
- Syntax highlighting via the `parse` interactive parser framework
- Line numbers
- Emacs-like keybindings
- Code completion
- Full text editing (undo/redo, selection, clipboard)
- Integrated into the Cogent Core widget system

**How syntax highlighting would work:**
Write a parser definition for the Lisp dialect using Cogent Core's `parse` framework, or use an existing Lisp definition if one is provided. The editor widget handles the rest.

**Concerns:**
- Cogent Core is very young (initial release mid-2024)
- Small community (~2.3k stars, small team)
- API stability is uncertain
- Documentation is improving but not as extensive as Fyne or GTK

**Verdict:** Intriguing because it solves both the editor AND 3D viewport problems in one framework. But the youth and small community of the project is a significant risk factor.

### Editor Recommendation

**For the safest, most feature-rich editor:** Wails + CodeMirror 6 (or Monaco). The web code editor ecosystem is 10 years ahead of native Go options.

**For a native-Go solution:** Gio + gvcode is the best balance of features, flexibility, and Go-nativeness. It requires writing a Lisp tokenizer but that is straightforward.

**Dark horse:** Cogent Core -- if its code editor is sufficiently mature, it solves both problems at once.

---

## 3. 3D Viewport Integration

Lignin needs to render solid geometry (B-rep solids from the design graph) in a 3D viewport with orbit/pan/zoom controls, diagnostic overlays, and optional exploded views. The renderer must be read-only (never mutate design state).

### Option A: Gio -- Custom OpenGL/Vulkan via GPU Package

**Approach:**
Gio's `gioui.org/gpu` package provides `OpenGLRenderTarget` and `VulkanRenderTarget` structs. The GLFW integration example (`gio-example/glfw/main.go`) demonstrates embedding Gio into an external OpenGL context. The reverse (embedding OpenGL content into Gio) requires rendering to a framebuffer and compositing into the Gio frame.

**Architecture:**
1. Create a Gio window with a split layout
2. For the 3D viewport region, render your scene to an OpenGL FBO
3. Read back the FBO as a texture or image
4. Display the result in the Gio layout using an image widget or custom paint operation
5. Forward mouse/keyboard events from the viewport region to your 3D camera controller

**Challenges:**
- Gio's immediate-mode rendering means you need to carefully manage the OpenGL context sharing
- The GLFW example shows Gio running inside an OpenGL app, not the other way around
- Performance depends on texture upload path (GPU-to-GPU vs GPU-to-CPU-to-GPU roundtrip)
- No built-in 3D scenegraph -- you build everything from scratch using go-gl or similar

**Feasibility:** Medium. Possible but requires significant low-level OpenGL work. Gio's architecture is flexible enough, but there is no paved path for embedding 3D content.

### Option B: Wails -- WebGL / Three.js Frontend

**Approach:**
Use Three.js or Babylon.js in the web frontend to render 3D geometry. Go backend computes geometry and sends mesh data (vertices, normals, face indices) to the frontend via Wails bindings. The frontend renders using WebGL/WebGL2.

**Architecture:**
1. Go backend evaluates Lisp, produces design graph, tessellates solids into triangle meshes
2. Mesh data is serialized and sent to the JS frontend via Wails event system or bound function calls
3. Three.js scene renders the meshes with orbit controls, highlighting, etc.
4. Diagnostic overlays (warnings, errors) rendered as HTML elements or Three.js sprites
5. All state management stays in Go; frontend is purely a display layer

**Challenges:**
- Serialization overhead for mesh data (mitigated by binary transfer, SharedArrayBuffer, or similar)
- WebGL performance ceiling in webview (typically good enough for CAD models, not for millions of triangles)
- WebGL2 support varies by platform webview (good on modern macOS/Linux/Windows)
- Debugging requires both Go and JS toolchains
- Two rendering ecosystems to maintain (Go backend + JS frontend)

**Feasibility:** High. This is the most well-trodden path. Three.js is battle-tested for 3D visualization. The webview's WebGL support is sufficient for woodworking CAD models (typically low poly count).

### Option C: gotk4 -- GtkGLArea

**Approach:**
GTK4's `GtkGLArea` widget provides a native OpenGL context for rendering custom 3D content within a GTK window. You connect to the `render` signal and issue OpenGL draw calls directly.

**Architecture:**
1. Create a GTK4 window with a GtkPaned (split pane)
2. Top pane: GtkGLArea with depth buffer and stencil buffer enabled
3. Bottom pane: GtkSourceView for the code editor
4. Connect GtkGLArea's `render` signal to your Go rendering function
5. Use go-gl bindings for OpenGL calls within the render callback
6. Queue redraws when the design graph changes

**Challenges:**
- gotk4 bindings for GtkGLArea may have gaps or bugs (alpha quality)
- OpenGL function loading in GTK4 context requires care (GNOME Discourse has several threads on this)
- GTK4's rendering is already GL-based, so context sharing needs careful management
- macOS deployment requires GTK4 installation (Homebrew)

**Feasibility:** Medium-High. GtkGLArea is specifically designed for this use case, and it is a well-tested GTK4 feature. The risk is in the gotk4 bindings quality.

### Option D: giu (Dear ImGui) -- FBO Render-to-Texture

**Approach:**
This is the classic Dear ImGui pattern for embedding 3D viewports. Render your 3D scene to an OpenGL framebuffer object (FBO), then display the FBO's color attachment texture as an `ImGui::Image()` in a giu window.

**Architecture:**
1. Create a giu window with a split layout (top = image widget, bottom = text editor)
2. Each frame: render 3D scene to an FBO using go-gl
3. Display the FBO texture via `giu.Image()` or `giu.Custom()` widget
4. Handle mouse events in the image region for orbit/pan/zoom
5. Use giu's text input widgets for code editing (limited -- no syntax highlighting)

**Challenges:**
- giu has no code editor widget with syntax highlighting (would need to build one or use a basic text input)
- The FBO-to-image pattern is well-documented in the C++ ImGui world but less so in Go/giu specifically
- giu's event handling for the image region requires custom logic
- ImGui's retained-mode text widgets are limited for a code editor use case

**Feasibility:** High for the 3D part, Low for the editor part. This is the go-to approach for tools/editors in the ImGui world, but the editor side is weak.

### Option E: Cogent Core -- Native xyz Package

**Approach:**
Cogent Core includes the `xyz` package, a complete 3D scenegraph framework rendered via WebGPU. The `xyzcore` package integrates xyz scenes directly into 2D Cogent Core windows.

**Architecture:**
1. Create a Cogent Core window with a split layout
2. Top pane: `xyzcore.Scene` widget containing your 3D scenegraph
3. Bottom pane: Cogent Code editor widget
4. When the design graph changes, rebuild the xyz scenegraph (meshes, materials, lights)
5. The xyz package handles rendering, camera controls, and picking

**Challenges:**
- WebGPU backend is modern but less mature than OpenGL for desktop apps
- Cogent Core is very young; API may change
- Small community means less help when things go wrong
- Performance characteristics for CAD-scale models are unknown
- The xyz scenegraph may need extension for CAD-specific features (face highlighting, exploded views)

**Feasibility:** High in theory -- the framework is specifically designed for this. Risk is in Cogent Core's maturity.

### 3D Viewport Recommendation

**Most proven path:** Wails + Three.js. Enormous ecosystem, well-understood performance characteristics, trivial to add orbit controls, picking, highlighting, and exploded views.

**Best native integration:** Cogent Core (xyz package provides exactly what we need) or gotk4 (GtkGLArea is purpose-built for this).

**Most flexible for a native Go app:** giu FBO pattern for the viewport, but you sacrifice the editor.

---

## 4. PoC Architecture -- Recommended Approach

After evaluating all options, I recommend two viable paths for the PoC, depending on whether you prioritize native purity or pragmatic development speed.

### Path A: Gio + gvcode + Custom OpenGL Renderer (Native Go)

This is the "pure Go" path. It requires the most upfront work but gives full control and a single-language codebase.

```
+-------------------------------------------------------+
|                    Gio Window                          |
|                                                       |
|  +--------------------------------------------------+ |
|  |           3D Viewport (top half)                  | |
|  |                                                   | |
|  |   OpenGL FBO rendered offscreen via go-gl         | |
|  |   Composited into Gio layout as texture/image     | |
|  |   Mouse events -> orbit/pan/zoom controller       | |
|  |                                                   | |
|  +--------------------------------------------------+ |
|  |         Horizontal split bar (draggable)          | |
|  +--------------------------------------------------+ |
|  |           Code Editor (bottom half)               | |
|  |                                                   | |
|  |   gvcode editor widget                            | |
|  |   Custom Lisp tokenizer for syntax highlighting   | |
|  |   OnChange callback -> rate-limited eval trigger  | |
|  |                                                   | |
|  +--------------------------------------------------+ |
|  | Status bar: errors, warnings, eval status         | |
|  +--------------------------------------------------+ |
+-------------------------------------------------------+
```

**Data flow:**
1. User edits code in gvcode editor
2. OnChange fires (rate-limited via debounce timer, e.g., 300ms)
3. Lisp engine evaluates source, produces design graph (immutable)
4. Design graph is tessellated into triangle meshes by geometry kernel
5. Mesh data is passed to the OpenGL renderer (read-only; renderer never mutates state)
6. Renderer updates FBO; Gio redraws viewport region
7. Errors/warnings from evaluation are displayed in status bar

**Split pane implementation in Gio:**
Gio uses a `layout.Flex` with `layout.Flexed` children. A split pane would be implemented as a vertical flex with two flexed regions and a draggable bar between them. The `gio-x` extensions package may have a split widget, or it can be built with ~50 lines of layout code.

**Pros:**
- Pure Go codebase (single language, single build)
- Full control over rendering pipeline
- Gio is lightweight and performant
- gvcode provides a solid starting point for the editor

**Cons:**
- OpenGL integration with Gio requires manual FBO management
- gvcode is young and may have gaps
- Smaller community for help
- More code to write for basic infrastructure

### Path B: Wails + CodeMirror 6 + Three.js (Hybrid Go/Web)

This is the pragmatic path. It leverages the massive web ecosystem for the two hardest UI problems (code editor and 3D viewport) while keeping all logic in Go.

```
+-------------------------------------------------------+
|                   Wails Window                         |
|  Go Backend                    Web Frontend            |
|  +-----------+                 +---------------------+ |
|  | Lisp Eval |  <-- events --> | Three.js Viewport   | |
|  | Engine    |                 | (top half)           | |
|  +-----------+                 |                      | |
|  | Design    |  <-- bindings   | Orbit controls       | |
|  | Graph     |       -->       | Diagnostic overlays  | |
|  +-----------+                 +---------------------+ |
|  | Geometry  |                 | Split bar            | |
|  | Kernel    |                 +---------------------+ |
|  +-----------+                 | CodeMirror 6 Editor  | |
|  | Tessellat.|  --> mesh data  | (bottom half)        | |
|  +-----------+                 |                      | |
|                                | Lisp syntax mode     | |
|                                | Line numbers         | |
|                                | Error gutters        | |
|                                +---------------------+ |
|                                | Status bar           | |
|                                +---------------------+ |
+-------------------------------------------------------+
```

**Data flow:**
1. User edits code in CodeMirror editor
2. CodeMirror onChange fires, debounced, calls Go backend via Wails binding
3. Go backend evaluates Lisp, produces design graph
4. Geometry kernel tessellates solids into meshes
5. Mesh data sent to frontend (via Wails event or return value)
6. Three.js updates scene geometry
7. Errors/warnings sent to frontend, displayed in editor gutters and status bar

**Frontend tech stack:**
- Svelte or vanilla TypeScript (keep it simple)
- CodeMirror 6 with a custom Lisp language mode
- Three.js with OrbitControls
- CSS Grid or flexbox for split layout (trivial)

**Pros:**
- CodeMirror 6 is the best code editor available, period
- Three.js has every 3D feature you could want
- Wails produces small binaries (~4MB vs Electron's ~100MB)
- Huge community and ecosystem for both libraries
- Fastest path to a polished, usable PoC
- Easy to add features later (minimap, autocomplete, etc.)

**Cons:**
- Two languages (Go + TypeScript/JavaScript)
- Serialization boundary between Go and web frontend
- Debugging spans two runtimes
- "Not native" -- uses system webview
- WebGL performance ceiling (likely fine for woodworking CAD)

### Path C: Cogent Core (Unified Native -- Experimental)

This is the "all-in-one" path. Cogent Core provides both a code editor and 3D viewport in a single Go framework.

```
+-------------------------------------------------------+
|                 Cogent Core Window                     |
|                                                       |
|  +--------------------------------------------------+ |
|  |         xyzcore.Scene (3D viewport)               | |
|  |         WebGPU-rendered scenegraph                | |
|  |         Camera controls, picking, highlighting    | |
|  +--------------------------------------------------+ |
|  |         core.Splits (built-in split pane)         | |
|  +--------------------------------------------------+ |
|  |         texteditor.Editor (code editor)           | |
|  |         Syntax highlighting via parse framework   | |
|  |         Line numbers, undo/redo, keybindings      | |
|  +--------------------------------------------------+ |
|  |         Status bar: errors, warnings              | |
|  +--------------------------------------------------+ |
+-------------------------------------------------------+
```

**Pros:**
- Single framework solves both problems
- Pure Go, single build
- 3D is first-class (xyz package)
- Code editor is built-in
- WebGPU is the future of cross-platform GPU rendering

**Cons:**
- Very young framework (2024 initial release)
- Small community
- API stability uncertain
- Risk of framework-level bugs blocking progress
- WebGPU support on Linux may require specific drivers

---

## 5. Final Recommendation

### Primary Recommendation: Wails + CodeMirror 6 + Three.js

**Rationale:**

For Lignin's specific requirements -- a code editor and a 3D viewport as the two primary UI elements -- the web ecosystem is overwhelmingly more mature than any native Go GUI option.

1. **The code editor problem is solved.** CodeMirror 6 (or Monaco) provides everything Lignin needs: syntax highlighting, line numbers, error gutters, auto-completion, and more. Building an equivalent in a native Go framework would take months and produce an inferior result.

2. **The 3D viewport problem is solved.** Three.js handles mesh rendering, orbit controls, picking, highlighting, exploded views, and more. The WebGL performance ceiling is well above what woodworking CAD models require.

3. **All logic stays in Go.** The Lisp engine, design graph, geometry kernel, and validation all remain pure Go. The frontend is purely a display layer, consistent with the PRD requirement that the renderer never mutates design state.

4. **Wails is mature and lightweight.** At 32,000+ stars with active v3 development, it is well-maintained. Binaries are small (~4MB). The system webview avoids Electron's overhead.

5. **Development velocity.** The web ecosystem allows rapid iteration on the UI without recompiling Go code (Wails hot reload). Finding developers who know HTML/CSS/JS + Go is much easier than finding developers who know Gio or Cogent Core.

6. **Cross-platform.** Wails supports macOS, Linux, and Windows. WebGL works in all modern system webviews.

### Secondary Recommendation (Native Go): Gio + gvcode

If a native Go solution is strongly preferred, Gio with gvcode is the best option. The immediate-mode architecture gives full control, and gvcode provides a reasonable starting point for the code editor. The 3D viewport would require custom OpenGL integration (FBO render-to-texture composited into the Gio layout), which is significant work but feasible.

**Choose Gio if:**
- Single-language codebase is a hard requirement
- You want maximum control over the rendering pipeline
- You are willing to invest more upfront in infrastructure
- You plan to eventually need tight integration between the editor and viewport (e.g., clicking a 3D face highlights the corresponding Lisp expression)

### Watch List: Cogent Core

Cogent Core is the most architecturally aligned framework for Lignin (it literally has a code editor AND a 3D viewport). If the project matures over the next 6-12 months, it could become the best option. Check back on:
- API stability
- Community growth
- WebGPU driver support on target platforms
- Performance with CAD-scale geometry

### Not Recommended

- **Fyne:** No code editor widget, no public API for 3D rendering. Both critical features are missing. The open issues (#129, #3183) suggest these are years away.
- **gotk4:** GtkSourceView + GtkGLArea would technically work, but GTK4 on macOS is a deployment headache, gotk4 is alpha quality with memory leaks, and the community is too small for a project dependency.
- **giu (Dear ImGui):** Excellent for the 3D viewport (FBO pattern), but has no code editor widget. You would need to build one from scratch, which defeats the purpose.
- **Ebitengine:** A 2D game engine. Wrong tool for this job. The Guigui GUI framework is alpha and will not be ready in time.

---

## Appendix: Source Links

- [Fyne GitHub](https://github.com/fyne-io/fyne) -- ~28k stars, BSD-3
- [Fyne Issue #129: Expose OpenGL from Driver](https://github.com/fyne-io/fyne/issues/129)
- [Fyne Issue #3183: Add code widget](https://github.com/fyne-io/fyne/issues/3183)
- [Gio UI](https://gioui.org/) -- MIT/Unlicense
- [Gio GitHub Mirror](https://github.com/gioui/gio) -- ~1.8k stars
- [Gio GPU Package](https://pkg.go.dev/gioui.org/gpu)
- [Gio GLFW Example](https://github.com/gioui/gio-example/blob/main/glfw/main.go)
- [gvcode: Code Editor for Gio](https://github.com/oligo/gvcode)
- [Wails GitHub](https://github.com/wailsapp/wails) -- ~32.4k stars, MIT
- [Wails v3 Alpha](https://v3alpha.wails.io/)
- [gotk4 GitHub](https://github.com/diamondburned/gotk4) -- ~640 stars, MPL-2 (generated)
- [gotk4-sourceview](https://pkg.go.dev/github.com/diamondburned/gotk4-sourceview/pkg/gtksource/v3)
- [GtkGLArea Docs](https://docs.gtk.org/gtk4/class.GLArea.html)
- [giu GitHub](https://github.com/AllenDang/giu) -- ~2.5k stars, MIT
- [Cogent Core GitHub](https://github.com/cogentcore/core) -- ~2.3k stars, BSD-3
- [Cogent Core xyz Package](https://pkg.go.dev/cogentcore.org/core/xyz)
- [Cogent Code](https://pkg.go.dev/cogentcore.org/cogent/code)
- [Ebitengine](https://ebitengine.org/) -- ~11k stars, Apache-2.0
- [Three.js](https://threejs.org/)
- [CodeMirror 6](https://codemirror.net/)
- [Monaco Editor](https://microsoft.github.io/monaco-editor/)
- [Go GUI Projects List](https://github.com/go-graphics/go-gui-projects)
- [LogRocket: Best GUI Frameworks for Go](https://blog.logrocket.com/best-gui-frameworks-go/)
