---
title: "feat: Lignin MVP -- Programmable CAD for Woodworking"
type: feat
date: 2026-02-02
---

# Lignin MVP

## Overview

Build the minimum viable Lignin: a desktop application where users write Lisp code that defines woodworking parts and joints, and see the resulting 3D model update live. The MVP proves the full pipeline -- Lisp source to design graph to geometry to rendered viewport -- using a simple box with butt joints as the target design.

## Problem Statement / Motivation

Woodworkers who design programmatically have two options today: OpenSCAD (geometry-level, no woodworking semantics) or commercial parametric CAD (GUI-driven, not code-first). Lignin fills the gap: a code-driven modeler where joints, grain, and parts are first-class concepts, not boolean operations on boxes.

The MVP validates that this pipeline works end-to-end before investing in advanced joinery, stock mapping, or output generation.

## Proposed Solution

Four subsystems, each using the recommended technology from the research phase:

| Subsystem | Technology | Role |
|---|---|---|
| Lisp Engine | **zygomys** (pure Go, bytecode VM) | Evaluate user code into a design graph |
| Design Graph | Custom Go types (hybrid identity) | Immutable DAG of parts, joins, transforms |
| Geometry Kernel | **Manifold** via CGo (sdfx as interim) | Solid modeling + boolean ops with face tracking |
| UI Shell | **Wails** + CodeMirror 6 + Three.js | Editor + 3D viewport in a split-pane desktop app |

### Architecture Diagram

```
User Lisp Code
      |
      v
+------------------+
| zygomys Engine   |  Evaluate(source) -> DesignGraph
| (Go, sandboxed)  |
+------------------+
      |
      v
+------------------+
| Design Graph     |  Immutable DAG: parts, joins, groups
| (Go structs)     |  Hybrid identity: source-path + content hash
+------------------+
      |
      v
+------------------+
| Geometry Kernel  |  Tessellate parts, apply join geometry
| (Manifold/CGo)   |  Face ID tracking for joinery
+------------------+
      |
      v  (mesh data via Wails bindings)
+------------------+
| Wails Frontend   |  CodeMirror 6 editor (bottom)
| (TS + Three.js)  |  Three.js viewport (top)
+------------------+
```

## Implementation Phases

### Phase 1: Core Engine + Design Graph

Establish the Go project, integrate zygomys, and define the design graph types.

**Deliverables:**

- [ ] Go module init (`go.mod`) with zygomys dependency
- [ ] `pkg/engine/engine.go` -- `Engine.Evaluate(source string) (*DesignGraph, error)` wrapping zygomys in sandboxed mode
- [ ] `pkg/graph/` -- Core types: `Node`, `NodeID`, `DesignGraph`, `BoardData`, `JoinData`, `GroupData`, `FaceID`, `MaterialSpec`, `Vec3`, `Axis`
- [ ] Lisp builtins registered in zygomys: `defpart`, `board`, `material`, `butt-joint`, `assembly`, `part` (lookup), `place`, `vec3`, `screw`. zygomys native `def` used for variable binding.
- [ ] `pkg/graph/validate.go` -- Tier 1 structural validation: DAG cycle check, reference integrity, name uniqueness
- [ ] `pkg/engine/timeout.go` -- Evaluation timeout (5s hard limit) via goroutine + generation counter; stale results discarded
- [ ] Unit tests: evaluate simple Lisp source, produce a design graph, validate it
- [ ] Unit tests: timeout on infinite loop, empty source returns empty graph

**Target Lisp that must work:**
```lisp
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
```

### Phase 2: Geometry Kernel

Integrate a geometry backend behind an abstract `Kernel` interface. Start with sdfx (pure Go, no CGo) to unblock Phase 3, then add Manifold.

**Deliverables:**

- [ ] `pkg/kernel/kernel.go` -- `Kernel` interface: `Box`, `Cylinder`, `Union`, `Difference`, `Intersection`, `Translate`, `Rotate`, `ToMesh`
- [ ] `pkg/kernel/mesh.go` -- `Mesh` type: vertices, normals, face indices (JSON-serializable for the frontend)
- [ ] `pkg/kernel/sdfx/` -- sdfx implementation of `Kernel` (pure Go, no face tracking, for development)
- [ ] `pkg/kernel/manifold/` -- Manifold CGo implementation of `Kernel` (face ID tracking via `originalID`/`faceID`)
- [ ] `pkg/tessellate/tessellate.go` -- Walk a `DesignGraph`, produce `[]Mesh` using the `Kernel` (one mesh per part, joints applied as boolean ops)
- [ ] Build script or Makefile for compiling Manifold C bindings (`manifoldc`)
- [ ] Unit tests: box with a hole, two-part butt joint geometry

### Phase 3: Wails UI Shell

Desktop app with split-pane layout: CodeMirror 6 editor (bottom), Three.js viewport (top).

**Deliverables:**

- [ ] Wails v2 project scaffold with Go backend + Svelte or vanilla TS frontend
- [ ] Go backend exposes `Evaluate(source string) EvalResult` binding -- returns mesh data + errors/warnings
- [ ] Frontend: CodeMirror 6 editor with custom Lisp syntax mode (keywords: `defpart`, `board`, `material`, `butt-joint`, `assembly`, `part`)
- [ ] Frontend: Three.js viewport with OrbitControls, ambient + directional lighting, per-part coloring
- [ ] Frontend: CSS Grid split layout with draggable divider
- [ ] Frontend: Status bar showing eval errors/warnings with line numbers
- [ ] Debounced evaluation: editor `onChange` -> 300ms debounce -> call Go `Evaluate` -> update Three.js scene
- [ ] Error gutter integration: mark lines with errors in CodeMirror
- [ ] Stale-on-error: viewport retains last good render with dimming indicator on eval failure
- [ ] Camera persistence: orbit/pan/zoom state preserved across scene updates
- [ ] File I/O: Cmd-S save, Cmd-O open, dirty indicator in title bar, unsaved-changes warning on close (`.lignin` extension)

### Phase 4: Integration + MVP Polish

Wire everything together and make the box example work end to end.

**Deliverables:**

- [ ] End-to-end: type box Lisp code -> see 3D box render live
- [ ] Per-part coloring in viewport (different color per named part)
- [ ] Click part in viewport -> highlight corresponding `defpart` in editor (or vice versa, click defpart -> highlight part)
- [ ] Tier 2 geometric validation: non-zero dimensions, no self-joins, no duplicate joins
- [ ] Tier 3 material warnings: end-grain butt joint warning
- [ ] Default example file loaded on startup (the box example)
- [ ] Edge case tests: empty editor, syntax error mid-typing, undefined part ref, zero-dimension board, rapid typing (debounce)
- [ ] Cross-platform build: macOS + Linux (Makefile or build script)

## Design Decisions (from SpecFlow analysis)

These questions were surfaced during flow analysis and need explicit answers:

**Viewport on error: keep last successful render (stale-on-error).**
When evaluation fails (syntax error, undefined part, etc.), the 3D viewport retains the last successfully rendered scene with a visual dimming indicator. This is standard behavior in OpenSCAD, ShaderToy, and similar live-coding tools. Clearing the viewport on every intermediate keystroke would make the tool unusable.

**Evaluation timeout: 5-second hard limit.**
zygomys has no built-in timeout. User code can contain infinite loops (`(define (f) (f)) (f)`). Run evaluation in a goroutine; if it exceeds 5 seconds, kill it and return a timeout error. Accept that this is imperfect (goroutines cannot be forcibly killed in Go) -- for MVP, use `runtime.Goexit` or a generation-counter discard pattern.

**`define` and `place` are in the MVP DSL.**
The target Lisp on its own is too minimal to build the box example from the design graph research. `define` (variable binding) is needed for parametric dimensions. `place` (part positioning) is needed because without it all parts overlap at the origin. zygomys supports `def` natively; `place` is a new builtin that creates a transform node.

**Butt joints are metadata-only for MVP.**
Joints do not automatically position parts. The user positions parts manually with `place`. Joints are declarative annotations that: (a) validate face contact, (b) carry fastener specs, (c) enable future geometry operations (dado cuts, mortise pockets). For MVP, a butt joint produces no geometry modifications -- it is a validation and documentation node.

**File open/save is required for MVP.**
Without save, users lose all work on close. Minimum: Cmd-S saves to a `.lignin` file, Cmd-O opens one, dirty indicator in title bar, unsaved-changes warning on close. Files are plain text Lisp source.

**Partial results: all-or-nothing for MVP.**
If evaluation fails, no meshes are returned. The `EvalResult` struct supports both `Meshes` and `Errors` fields for future partial-result support, but Phase 1 implements all-or-nothing (simpler).

**Camera persists across evaluations.**
Three.js camera position, rotation, and zoom are preserved when the scene geometry updates. Only mesh objects are replaced; the camera and controls are never recreated.

**Units are millimeters.**
All numeric dimensions in the DSL are mm. The `GlobalDefaults.Units` field defaults to `"mm"`. Inches may be added post-MVP.

**MVP ships with sdfx kernel; Manifold is opt-in.**
The distributed binary uses sdfx (pure Go, no CGo dependencies). Manifold CGo bindings are built and tested but require the user to install the Manifold C library. Part selection in the viewport works at the whole-mesh level (one mesh per part), sidestepping sdfx's lack of face tracking.

**Windows is not a target for MVP.**
macOS and Linux only, per the PRD. Wails supports Windows, so it may work, but it is not tested or guaranteed.

## Technical Considerations

**zygomys integration:**
- Fresh `NewZlispSandbox()` per evaluation cycle ensures determinism and prevents user code from accessing the filesystem.
- `AddFunction` registers each CAD builtin. Keyword args parsed from zygomys `SexpHash` or alternating symbol/value pairs.
- Evaluation errors include source location via zygomys stack traces.

**Design graph identity:**
- `NodeID` = SHA-256 of the source expression path (stable across evals that preserve structure).
- `ContentHash` = SHA-256 of semantic content (enables cheap diff between graph versions).
- `NameIndex` maps user names to `NodeID` for `(part "name")` lookups.

**Geometry kernel interface:**
- Abstract `Kernel` interface allows swapping sdfx (dev) and Manifold (production) without changing calling code.
- Manifold CGo: ~20-30 C functions to wrap. `runtime.SetFinalizer` for cleanup. Zero required dependencies for Manifold build.
- Face tracking: Manifold `originalID`/`faceID` maps design graph nodes to output mesh faces. Required for future joint face highlighting.

**Wails data flow:**
- Go `Evaluate` binding returns `EvalResult{Meshes []MeshData, Errors []EvalError, Warnings []EvalWarning}`.
- `MeshData` is JSON: `{vertices: Float32Array, normals: Float32Array, indices: Uint32Array, partName: string, color: string}`.
- For large meshes, consider binary transfer (Wails supports `[]byte` return).
- Three.js creates `BufferGeometry` from mesh data, one `Mesh` per part.

**Performance budget:**
- Target: <500ms from keystroke to viewport update (300ms debounce + <200ms eval+render).
- Furniture-scale models (dozens of parts) tessellate in <50ms with Manifold.
- zygomys bytecode VM handles re-eval of typical DSL files in <10ms.

## Acceptance Criteria

- [ ] User can type Lisp code defining boards, materials, butt joints, and assemblies
- [ ] 3D viewport updates live (within 500ms of last keystroke)
- [ ] A 5-part open-top box with butt joints renders correctly
- [ ] Syntax errors display in the editor gutter with line numbers
- [ ] Validation warnings (e.g., end-grain butt joint) appear in status bar
- [ ] Builds and runs on macOS and Linux
- [ ] Design graph is fully immutable -- renderer never mutates state
- [ ] Fresh evaluation per keystroke cycle (deterministic, no mutable globals)

## Dependencies & Risks

| Risk | Mitigation |
|---|---|
| zygomys keyword-arg parsing may not match the DSL design | Prototype the `board` builtin early; fall back to hash-map args if keyword pairs are awkward |
| Manifold CGo bindings: cross-compilation complexity | Use sdfx as stand-in for development; contain CGo to `pkg/kernel/manifold/` behind interface |
| Wails webview WebGL performance on Linux | Test early on target distros; Three.js is well-tested in webviews |
| gvcode / CodeMirror Lisp syntax mode | Lisp syntax is simple (parens, symbols, strings, numbers); a Lezer grammar is ~50 lines |
| zygomys maintenance (single maintainer) | jig/lisp as short-term fallback; custom Lisp as long-term escape hatch |

## Post-MVP (PRD gaps acknowledged)

These PRD requirements are intentionally deferred from the MVP:

- **Advanced joints:** Dado, rabbet, mortise-tenon, dovetail (MVP has butt joints only)
- **Exploded views** (PRD Section 9)
- **Output generation:** Cut lists, dimensioned views (PRD Section 11)
- **Design graph inspection UI** (PRD: "inspectable and traceable")
- **Stock/material mapping** (PRD Section 7 -- advisory, deferred)
- **Incremental re-evaluation** (MVP does full re-eval per keystroke)
- **Manifold as default kernel** (MVP ships sdfx; Manifold is opt-in)
- **Face-level selection in viewport** (MVP does whole-part selection)
- **Inch unit support** (MVP is mm only)

## References & Research

- [Lisp Engine Research](../../research/lisp_evaluation_engine.md) -- zygomys recommendation, PoC code
- [Design Graph Architecture](../../research/design_graph_architecture.md) -- Go struct definitions, validation rules, worked box example
- [Geometry Kernel Research](../../research/geometry_kernel.md) -- Manifold recommendation, sdfx fallback, Kernel interface
- [GUI Framework Research](../../research/gui_framework.md) -- Wails + CodeMirror + Three.js recommendation
- [Lignin PRD](../../lignin_prd.md) -- System objectives, non-goals, definition of done
- [zygomys](https://github.com/glycerine/zygomys) -- Lisp engine
- [Manifold](https://github.com/elalish/manifold) -- Geometry kernel
- [Wails](https://wails.io/) -- Desktop app framework
- [CodeMirror 6](https://codemirror.net/) -- Code editor
- [Three.js](https://threejs.org/) -- 3D renderer
