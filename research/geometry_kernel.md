# Geometry Kernel Research for Lignin

**Date:** 2026-02-02
**Project:** Lignin -- Programmable CAD for Woodworking (Go)
**Purpose:** Evaluate geometry kernel options for solid modeling and boolean operations on furniture-scale models.

---

## 1. Survey Report

### 1.1 Manifold (elalish/manifold)

**Repository:** https://github.com/elalish/manifold
**Description:** A C++ geometry library dedicated to creating and operating on manifold triangle meshes. Provides the first known guaranteed-manifold mesh boolean algorithm -- robust to edge cases that have historically been an open problem. Used by BRL-CAD, Godot Engine, and others. Supports primitives (Cube, Sphere, Cylinder, Extrude, Revolve), boolean operations (+, -, ^), face ID tracking via originalID/faceID, and level-set SDF evaluation. Builds with no required external dependencies.

**Maturity:** High. Actively maintained by Emmett Lalish (Google). Used in production by BRL-CAD (raised boolean success rate from 88.9% to 98.9%). Latest PyPI release: manifold3d v2.2.0. Regular releases on GitHub.

**Last Activity:** Continuous through 2025 and into 2026. Active discussions, issues, and releases.

**Go Integration Approach:** No existing Go bindings. Manifold provides official C FFI bindings (`manifoldc`), designed explicitly to facilitate bindings from other static languages. Go would call `manifoldc.so`/`manifoldc.dylib` via CGo. The C API is described as "pretty clean" and was originally created for OCaml bindings using the same pattern.

**Key Strengths for Lignin:**
- **Face identity tracking:** Manifolds track relationships to inputs via `originalID` and `faceID` through `MeshGL`. After boolean operations, every output triangle references the input face it is a subset of. Coplanar triangles sharing the same faceID can be reassembled into polygonal faces. This directly supports Lignin's semantic face tagging requirement.
- **Guaranteed manifold output:** No edge cases or caveats. Deterministic.
- **Boolean robustness:** Touching cubes merge correctly, equal-height differences produce through-holes, and a mesh minus itself produces an empty mesh.
- **Tolerance handling:** Provides `GetTolerance()` and `GetEpsilon()` for precision tracking.
- **Performance:** Extensive parallelization; BRL-CAD saw 7x speedup over previous approach.
- **No UI concerns:** Pure library, no GUI dependencies.

**Viability: VIABLE -- Top Candidate**

---

### 1.2 OpenCASCADE Technology (OCCT)

**Repository:** https://github.com/Open-Cascade-SAS/OCCT
**Description:** The only open-source full-scale B-rep (boundary representation) solid modeling toolkit. Object-oriented C++ class library providing NURBS surfaces, analytic geometry, topology data structures, boolean operations, STEP/IGES I/O, and a full suite of CAD modeling algorithms. Used as the kernel for FreeCAD, CadQuery, and many commercial products.

**Maturity:** Very High. Decades of development. Industrial-grade. Released under LGPL-2.1.

**Last Activity:** Continuous. Official releases from Open Cascade SAS (now part of Capgemini).

**Go Integration Approach:** No existing Go bindings. OCCT is a massive C++ library. Integration would require:
1. Writing a C wrapper layer (`extern "C"`) around the needed OCCT C++ classes.
2. Compiling as a shared library linking against OCCT.
3. Calling the C wrapper from Go via CGo.
4. Building a high-level Go API on top.

Alternatively, SWIG could auto-generate Go bindings from C++ headers, but SWIG+Go support is limited and the OCCT API surface is enormous.

**Key Strengths for Lignin:**
- True B-rep with NURBS surfaces, exact geometry.
- Full face/edge/vertex topology -- ideal for semantic tagging.
- Mature boolean operations with extensive edge-case handling.
- STEP file I/O for interoperability.

**Key Weaknesses for Lignin:**
- Enormous API surface; wrapping even a subset is a major undertaking.
- Complex C++ build system; cross-platform compilation is non-trivial.
- Heavy dependency footprint (~100+ MB of libraries).
- No existing Go bindings -- would be a greenfield effort of months.
- LGPL-2.1 licensing requires careful handling for distribution.

**Viability: MARGINAL** -- Viable in theory but the effort to create Go bindings is disproportionate to Lignin's scope. Better suited if an existing binding existed.

---

### 1.3 CGAL (Computational Geometry Algorithms Library)

**Repository:** https://github.com/CGAL/cgal
**Description:** Open-source C++ library of computational geometry algorithms. Provides exact arithmetic, Nef polyhedra boolean operations, mesh processing, and many other algorithms. Used extensively in academia and GIS.

**Maturity:** Very High. Decades of development. Released under GPL/LGPL dual license.

**Last Activity:** Continuous. Active releases and maintenance.

**Go Integration Approach:** No Go bindings exist. A 2015 GitHub issue (#518) requesting Go bindings is labeled "Stalled." CGAL has official SWIG bindings for Python/Java only. Creating Go bindings would follow the same C-wrapper + CGo pattern as OCCT.

**Key Weaknesses for Lignin:**
- GPL license for some components is problematic.
- Mesh-based rather than true B-rep (Nef polyhedra are complex).
- Even more complex C++ template-heavy codebase than OCCT -- harder to wrap.
- No C FFI layer exists (unlike Manifold).

**Viability: NOT VIABLE** -- Licensing issues, no C API, extreme wrapping complexity.

---

### 1.4 sdfx (deadsy/sdfx)

**Repository:** https://github.com/deadsy/sdfx
**Description:** Pure Go CAD package using signed distance functions (SDFs). Objects are defined with Go code and rendered to STL/3MF files. Provides 2D and 3D primitives, CSG operations (union, difference, intersection), filleting, chamfering, extrusion, and revolution. Uses Marching Cubes for mesh generation.

**Maturity:** Moderate. 172 importers on pkg.go.dev. Actively maintained.

**Last Activity:** January 2025 (module); November 2025 (sdf sub-package). MIT license.

**Go Integration Approach:** Pure Go. Import directly. No CGo required.

**Key API Surface:**
```go
type SDF3 interface {
    Evaluate(p r3.Vec) float64
    BoundingBox() d3.Box
}

func Box3D(size v3.Vec, round float64) (SDF3, error)
func Cylinder3D(height, radius, round float64) (SDF3, error)
func Difference3D(s0, s1 SDF3) SDF3
func Union3D(s0, s1 SDF3) SDF3
func Intersect3D(s0, s1 SDF3) SDF3
func Extrude3D(sdf SDF2, height float64) (SDF3, error)
```

**Key Strengths:**
- Pure Go -- no CGo, no external dependencies, trivial cross-compilation.
- Clean SDF3 interface that maps well to Lignin's design graph.
- Boolean operations are mathematically trivial with SDFs (min/max of distance fields).
- Filleting and chamfering are natural with SDFs.
- Deterministic evaluation.

**Key Weaknesses for Lignin:**
- **No face identity tracking.** SDFs have no concept of faces, edges, or vertices. Geometry is sampled at render time via Marching Cubes. There is no way to tag a face for joinery operations or track which face of a mortise corresponds to which input.
- **No B-rep output.** Only triangle meshes (STL/3MF). No STEP export.
- **Approximate geometry.** Flat surfaces are tessellated; exact planes are not preserved.
- **No tolerance model.** SDFs rely on sampling resolution, not geometric tolerance.
- Nil SDF handling causes panics (reported by soypat/sdf fork).
- Incorrect degenerate triangle calculations (reported by soypat/sdf fork).

**Viability: MARGINAL** -- Excellent for quick prototyping but lacks face identity tracking, which is a core Lignin requirement.

---

### 1.5 gsdf (soypat/gsdf)

**Repository:** https://github.com/soypat/gsdf
**Description:** GPU-accelerated successor to soypat/sdf. Redesigns APIs to be vectorized for GPU execution. Generates GLSL shaders for real-time visualization. Pure Go with optional GPU acceleration.

**Maturity:** Early/Active. Successor to the archived soypat/sdf (archived August 2024).

**Last Activity:** 2024-2025 (active development).

**Go Integration Approach:** Pure Go. Import directly.

**Key Weaknesses for Lignin:** Same fundamental SDF limitations as sdfx -- no face identity, no B-rep, approximate geometry. Additionally, GPU-focused design is orthogonal to Lignin's batch/deterministic evaluation model.

**Viability: NOT VIABLE** -- Wrong architecture for Lignin's needs. GPU focus adds complexity without addressing core requirements.

---

### 1.6 celer/csg

**Repository:** https://github.com/celer/csg
**Description:** Pure Go CSG library based on csg.js. Implements mesh-based boolean operations using BSP trees. Supports cube, sphere, cylinder primitives and subtract/union/intersect. Exports ASCII STL.

**Maturity:** Low. 4 stars, 2 forks. No recent releases. The author recommends sdfx instead.

**Last Activity:** Inactive. No major releases in years.

**Viability: NOT VIABLE** -- Unmaintained, minimal, author recommends alternatives.

---

### 1.7 reactivego/csg

**Repository:** https://github.com/reactivego/csg
**Description:** Pure Go CSG on meshes using BSP trees. Port of csg.js. Supports union, subtract, intersect on cube/sphere/cylinder primitives. Handles coplanar polygon edge cases.

**Maturity:** Low. 7 stars. MIT license.

**Last Activity:** December 2021. No recent updates.

**Key Weaknesses:** Unmaintained. Mesh-based BSP approach does not scale well. No face tracking. No tolerance model.

**Viability: NOT VIABLE** -- Unmaintained, no face tracking, does not meet requirements.

---

### 1.8 GhostSCAD (ljanyst/ghostscad)

**Repository:** https://github.com/ljanyst/ghostscad
**Description:** Write CAD models in Go, compile to OpenSCAD language. Uses Go as a metaprogramming layer over OpenSCAD's CGAL-based kernel.

**Maturity:** Low-Moderate. Niche project.

**Last Activity:** 2022-2023.

**Key Weaknesses:** Depends on OpenSCAD runtime. Not a kernel -- just a code generator. Cannot track faces or provide programmatic access to geometry results.

**Viability: NOT VIABLE** -- Not a geometry kernel; just generates OpenSCAD code.

---

### 1.9 libfive

**Repository:** https://github.com/libfive/libfive
**Description:** C++ infrastructure for solid modeling using implicit functions (f-reps). Provides a C API (`libfive.h`) with bindings for Python and Guile Scheme. Exposes mathematical shape definitions and meshing.

**Maturity:** Moderate. Maintained by Matt Keeter (formerly nTopology). LGPL-2.1 for library, GPL-2 for GUI.

**Last Activity:** Active through 2025.

**Go Integration Approach:** No Go bindings exist. Has a C API (`libfive.h`) suitable for CGo wrapping.

**Key Weaknesses for Lignin:** Same implicit/SDF limitations -- no face identity tracking, approximate geometry. Adding CGo complexity for the same fundamental limitations as pure-Go sdfx is not justified.

**Viability: NOT VIABLE** -- SDF limitations plus CGo complexity without compensating advantages over sdfx.

---

### 1.10 Fornjot

**Repository:** https://github.com/hannobraun/fornjot
**Description:** Early-stage B-rep CAD kernel written in Rust. Focused on mechanical CAD (3D printing, machining, woodworking). Code-first modeling approach.

**Maturity:** Early. Self-described as "unsuited for real-world use cases" currently. v0.49.0 (March 2024).

**Last Activity:** Active blog updates through January 2026. Ongoing development.

**Go Integration Approach:** Would require Rust-to-C FFI export, then CGo. No C API exists. Double FFI bridge adds significant complexity.

**Viability: NOT VIABLE** -- Too early-stage, no C API, double FFI bridge.

---

## 2. Comparison Matrix

| Library | Repr Type | Go Native | CGo Needed | Boolean Ops | Face ID Tracking | Tolerance Model | Performance | Maintained |
|---|---|---|---|---|---|---|---|---|
| **Manifold** | Triangle mesh (manifold-guaranteed) | No | Yes (C FFI) | Robust, guaranteed manifold | Yes (originalID + faceID) | Yes (epsilon, tolerance) | Excellent (parallel) | Active |
| **OpenCASCADE** | B-rep (NURBS) | No | Yes (C++ wrapper) | Mature, industrial | Yes (full topology) | Yes (extensive) | Good | Active |
| **CGAL** | Nef polyhedra / mesh | No | Yes (C++ wrapper) | Exact arithmetic | Partial | Yes | Variable | Active |
| **sdfx** | SDF (implicit) | Yes | No | Trivial (min/max) | **No** | **No** | Good | Active |
| **gsdf** | SDF (GPU) | Yes | No | Trivial (min/max) | **No** | **No** | Excellent (GPU) | Active |
| **celer/csg** | Mesh (BSP) | Yes | No | Basic | **No** | **No** | Poor | Inactive |
| **reactivego/csg** | Mesh (BSP) | Yes | No | Basic | **No** | **No** | Poor | Inactive |
| **GhostSCAD** | OpenSCAD codegen | Yes | No | Via OpenSCAD | **No** | Via OpenSCAD | N/A | Inactive |
| **libfive** | F-rep (implicit) | No | Yes (C API) | Trivial (implicit) | **No** | **No** | Good | Active |
| **Fornjot** | B-rep | No | Yes (Rust FFI) | Early-stage | Planned | Planned | Unknown | Active |

---

## 3. Integration Approach for Top Candidates

### 3.1 Manifold via CGo (Recommended)

Manifold provides official C bindings (`manifoldc`) specifically designed for language interoperability. The integration approach:

#### Build Pipeline

1. **Build manifoldc as a shared library:**
   ```bash
   cmake -DMANIFOLD_CBIND=ON -DBUILD_SHARED_LIBS=ON ..
   make
   # Produces libmanifold.so and libmanifoldc.so (or .dylib on macOS)
   ```

2. **Install headers and libraries** to a known path (e.g., `/usr/local` or vendored in the project).

3. **Create Go package `manifold`** with CGo directives:
   ```go
   package manifold

   // #cgo CFLAGS: -I/path/to/manifold/include
   // #cgo LDFLAGS: -L/path/to/manifold/lib -lmanifoldc -lmanifold
   // #include <manifold/manifoldc.h>
   import "C"
   ```

#### Go API Layer

The Go wrapper would provide idiomatic types:

```go
package manifold

// Solid represents an immutable manifold solid.
type Solid struct {
    ptr *C.ManifoldManifold
}

// Box creates a box solid with the given dimensions.
func Box(x, y, z float64, center bool) *Solid { ... }

// Cylinder creates a cylinder with given height and radius.
func Cylinder(height, radiusLow, radiusHigh float64, segments int) *Solid { ... }

// Subtract returns the boolean difference: s - other.
func (s *Solid) Subtract(other *Solid) *Solid { ... }

// Union returns the boolean union: s + other.
func (s *Solid) Union(other *Solid) *Solid { ... }

// Intersect returns the boolean intersection: s & other.
func (s *Solid) Intersect(other *Solid) *Solid { ... }

// MeshGL returns the triangle mesh with face ID tracking.
func (s *Solid) MeshGL() *MeshGL { ... }

// OriginalID returns the unique ID assigned to this solid for tracking
// through boolean operations.
func (s *Solid) OriginalID() uint32 { ... }
```

#### Face Identity Tracking Architecture

Manifold's face tracking maps directly to Lignin's requirements:

1. **When creating a primitive**, call `ReserveIDs(1)` to get a unique ID. Store this in the design graph node.
2. **After boolean operations**, retrieve the output `MeshGL` which contains `runOriginalID` and `faceID` arrays.
3. **Reconstruct semantic faces** by grouping triangles that share the same `(originalID, faceID)` pair.
4. **Map back to design graph** using the originalID to find which input primitive each face came from.
5. **Semantic tags** (e.g., "top face of board A") can be maintained in a Go-side map: `map[uint32]map[int]FaceTag` keyed by `(originalID, faceID)`.

#### Memory Management

The C API uses opaque pointers. Go's `runtime.SetFinalizer` can ensure cleanup:

```go
func newSolid(ptr *C.ManifoldManifold) *Solid {
    s := &Solid{ptr: ptr}
    runtime.SetFinalizer(s, func(s *Solid) {
        C.manifold_delete_manifold(s.ptr)
    })
    return s
}
```

#### Cross-Platform Build

- **macOS:** Build manifoldc with CMake + Clang. Install via Homebrew or vendor.
- **Linux:** Build manifoldc with CMake + GCC. Package as system dependency or vendor.
- Manifold has zero required dependencies, simplifying the build.

---

### 3.2 OpenCASCADE via CGo (Fallback)

If Manifold's mesh-based representation proves insufficient (e.g., exact NURBS surfaces are needed for CNC output -- though this is a non-goal per the PRD), OpenCASCADE could be wrapped. The approach would mirror `opencascade-rs` (Rust bindings):

1. Write a thin C wrapper (`lignin_occt.h` / `lignin_occt.cpp`) using `extern "C"` around:
   - `BRepPrimAPI_MakeBox`, `BRepPrimAPI_MakeCylinder`, `BRepPrimAPI_MakeSphere`
   - `BRepAlgoAPI_Fuse` (union), `BRepAlgoAPI_Cut` (difference), `BRepAlgoAPI_Common` (intersection)
   - `TopExp_Explorer` for face/edge traversal
   - `StlAPI_Writer` for STL export
2. Compile against OCCT as a shared library.
3. Call from Go via CGo.

This is a significantly larger effort than the Manifold approach and is not recommended unless B-rep NURBS surfaces become a hard requirement.

---

## 4. Custom Kernel Fallback

If no existing solution meets all requirements, a custom kernel could be built. This section outlines the approach.

### 4.1 Representation Choice: Half-Edge B-rep

For woodworking CAD, a half-edge boundary representation is the best fit:

- **B-rep** stores explicit faces, edges, and vertices with their connectivity.
- **Half-edge** is a specific B-rep data structure where each edge is stored as two directed half-edges, enabling O(1) traversal of adjacent faces and edges.
- This directly supports face identity tracking and semantic tagging.

**Trade-offs vs. alternatives:**

| Criterion | B-rep (Half-Edge) | CSG Tree | SDF (Implicit) |
|---|---|---|---|
| Face identity | Native | Must evaluate to get faces | No faces until meshed |
| Boolean ops | Complex to implement | Trivial (tree nodes) | Trivial (min/max) |
| Exact geometry | Yes (analytic surfaces) | Depends on evaluation | No (sampled) |
| Semantic tagging | Direct (face/edge metadata) | Indirect (tree paths) | Not possible |
| Implementation effort | Very High | Low | Low |
| Tolerance handling | Essential and complex | Deferred to evaluator | Resolution-based |

### 4.2 Core Primitives (Go Interfaces)

```go
package kernel

import "github.com/lignin/lignin/math/vec3"

// Solid is the core interface for a 3D solid body.
type Solid interface {
    // Faces returns all faces of this solid.
    Faces() []Face
    // Edges returns all edges of this solid.
    Edges() []Edge
    // Vertices returns all vertices of this solid.
    Vertices() []Vertex
    // BoundingBox returns the axis-aligned bounding box.
    BoundingBox() Box3
    // Volume returns the volume of the solid.
    Volume() float64
}

// Face represents a bounded surface on a solid.
type Face interface {
    // ID returns the stable identity of this face.
    ID() FaceID
    // Surface returns the underlying geometric surface.
    Surface() Surface
    // Edges returns the bounding edges of this face.
    Edges() []Edge
    // Normal returns the outward-pointing normal at a point.
    Normal(u, v float64) vec3.Vec
    // Tag returns semantic metadata attached to this face.
    Tag() FaceTag
}

// FaceID is a stable identifier that survives boolean operations.
type FaceID struct {
    OriginalSolid uint32  // Which input solid this face originated from
    OriginalFace  uint32  // Which face of that solid
    Generation    uint32  // How many boolean ops deep
}

// FaceTag holds semantic metadata for joinery.
type FaceTag struct {
    Name      string            // e.g., "top", "end_grain"
    Material  string            // e.g., "walnut"
    GrainDir  vec3.Vec          // Grain direction vector
    Metadata  map[string]string // Extensible key-value pairs
}

// Surface is a geometric surface definition.
type Surface interface {
    // PointAt evaluates the surface at parametric coordinates.
    PointAt(u, v float64) vec3.Vec
    // NormalAt returns the surface normal at parametric coordinates.
    NormalAt(u, v float64) vec3.Vec
    // Type returns the surface type (plane, cylinder, sphere, etc.)
    Type() SurfaceType
}
```

### 4.3 Boolean Operation Interface

```go
// BooleanOp performs CSG boolean operations on solids.
type BooleanOp interface {
    Union(a, b Solid) (Solid, error)
    Difference(a, b Solid) (Solid, error)
    Intersection(a, b Solid) (Solid, error)
}

// Tolerance controls numerical precision for boolean operations.
type Tolerance struct {
    Linear  float64 // Distance tolerance (e.g., 1e-6 meters)
    Angular float64 // Angle tolerance in radians
}
```

### 4.4 Implementation Complexity Assessment

Building a custom B-rep kernel with robust boolean operations is an extremely large undertaking:

- **Boolean operations on B-rep** are among the hardest algorithms in computational geometry. They require surface-surface intersection, trimming, topology reconstruction, and extensive edge-case handling.
- **OpenCASCADE** has had decades of development by hundreds of engineers to reach its current robustness.
- **Manifold** took years of focused research to achieve guaranteed-manifold boolean results on meshes alone (simpler than NURBS B-rep).
- A custom kernel would likely require 6-12+ months of full-time development just for basic boolean operations, with many edge cases remaining.

**Recommendation:** A custom kernel should only be pursued if the project has multi-year funding and dedicated geometry algorithm expertise. For Lignin, wrapping an existing kernel is strongly preferred.

---

## 5. Sample Code: Box with a Hole

### Using Manifold via CGo (Recommended Approach)

```go
package main

import (
    "fmt"
    "github.com/lignin/lignin/kernel/manifold"
)

func main() {
    // Create a 100x50x25mm board (typical woodworking stock)
    board := manifold.Box(100, 50, 25, true) // centered

    // Create a 10mm diameter hole (for a dowel joint)
    hole := manifold.Cylinder(
        30,   // height (taller than board to ensure clean cut)
        5,    // radius (10mm diameter / 2)
        5,    // same radius top and bottom
        32,   // circular segments
    )

    // Position the hole: centered on the board's top face
    // Translate to board center (already centered) and ensure it
    // passes fully through the board
    hole = hole.Translate(0, 0, 0)

    // Boolean difference: board minus hole
    result := board.Subtract(hole)

    // Inspect face tracking
    mesh := result.MeshGL()
    fmt.Printf("Result: %d vertices, %d triangles\n",
        result.NumVert(), result.NumTri())

    // Face identity: find faces that came from the board vs. the hole
    boardID := board.OriginalID()
    holeID := hole.OriginalID()

    for _, run := range mesh.Runs() {
        switch run.OriginalID {
        case boardID:
            fmt.Printf("  Board face (faceID=%d): %d triangles\n",
                run.FaceID, run.NumTriangles)
        case holeID:
            fmt.Printf("  Hole face (faceID=%d): %d triangles\n",
                run.FaceID, run.NumTriangles)
        }
    }

    // Semantic tagging for joinery
    // After boolean, we can tag the cylindrical face of the hole
    // for the joinery system to reference
    tags := manifold.NewTagMap()
    for _, run := range mesh.Runs() {
        if run.OriginalID == holeID {
            tags.Set(run.OriginalID, run.FaceID, manifold.FaceTag{
                Name: "dowel_hole",
                Type: "cylindrical_bore",
                Metadata: map[string]string{
                    "diameter": "10mm",
                    "joint":    "dowel",
                },
            })
        }
    }

    // Export to STL for visualization
    manifold.ExportSTL(result, "board_with_hole.stl")
}
```

### Using sdfx (Pure Go Fallback -- No Face Tracking)

```go
package main

import (
    "log"

    "github.com/deadsy/sdfx/obj"
    "github.com/deadsy/sdfx/render"
    "github.com/deadsy/sdfx/sdf"
    v3 "github.com/deadsy/sdfx/vec/v3"
)

func main() {
    // Create a 100x50x25mm board
    board, err := sdf.Box3D(v3.Vec{X: 100, Y: 50, Z: 25}, 0)
    if err != nil {
        log.Fatal(err)
    }

    // Create a 10mm diameter through-hole
    hole, err := sdf.Cylinder3D(30, 5, 0)
    if err != nil {
        log.Fatal(err)
    }

    // Boolean difference
    result := sdf.Difference3D(board, hole)

    // Render to STL (Marching Cubes -- no face identity available)
    renderer := render.NewMarchingCubesOctree(300)
    render.ToSTL(result, "board_with_hole.stl", renderer)

    // NOTE: No way to identify which faces are the hole bore,
    // which are the board top/bottom, etc. All face identity is lost
    // in the SDF -> mesh conversion. This is the fundamental limitation
    // for Lignin's joinery system.
}
```

---

## 6. Final Recommendation

### Primary Recommendation: Manifold via CGo

**Manifold** is the recommended geometry kernel for Lignin, integrated via CGo bindings against the official C FFI (`manifoldc`).

#### Rationale

1. **Face identity tracking** -- Manifold's `originalID`/`faceID` system directly maps to Lignin's requirement for stable face identity through boolean operations. This is the single most differentiating feature among all candidates. No pure-Go library offers this.

2. **Boolean robustness** -- Manifold provides the first guaranteed-manifold mesh boolean algorithm. For a deterministic CAD tool, this guarantee is essential. Edge cases in boolean operations are the #1 source of failures in CAD kernels; Manifold eliminates them.

3. **Adequate representation** -- While Manifold uses triangle meshes rather than NURBS B-rep, this is sufficient for furniture-scale woodworking:
   - Flat surfaces (boards, panels) are represented exactly as coplanar triangle groups.
   - Cylindrical surfaces (dowel holes, turned legs) are approximated by triangle fans at configurable resolution -- more than adequate for woodworking tolerances (typically 0.5mm+).
   - Lignin's non-goals explicitly exclude CNC/CAM, so NURBS precision is not required.

4. **Minimal dependency** -- Manifold has zero required dependencies, making cross-platform builds straightforward. The library compiles with standard C++ toolchains on macOS and Linux.

5. **Active maintenance** -- Backed by Google (author is a Google engineer), used in production by major projects (BRL-CAD, Godot), with continuous development.

6. **Reasonable integration effort** -- The C API was designed for exactly this use case. A focused Go binding wrapping primitives, booleans, and MeshGL output could be completed in 1-2 weeks. The API surface needed for Lignin is small (perhaps 20-30 C functions).

7. **Performance** -- Extensive parallelization. Furniture-scale models (dozens of parts) will evaluate in milliseconds.

#### Known Trade-offs

- **CGo overhead:** Each Go-to-C call has ~100ns overhead. For Lignin's batch evaluation model (not real-time interaction), this is negligible.
- **Cross-compilation complexity:** CGo requires a C/C++ toolchain, making `go build` alone insufficient. This is manageable with build scripts or Docker-based CI.
- **Not true B-rep:** Manifold uses triangulated meshes, not NURBS surfaces. For woodworking this is acceptable. If Lignin ever needed exact NURBS (e.g., for CNC post-processing), OpenCASCADE would need to be reconsidered.
- **Mesh resolution trade-off:** Curved surfaces require choosing a segment count. For furniture-scale work, 32-64 segments per full circle provides sub-millimeter accuracy.

### Secondary Recommendation: sdfx for Rapid Prototyping

During early development, **sdfx** (pure Go, zero dependencies) can serve as a stand-in geometry backend for testing the Lisp evaluation engine and design graph without the CGo build complexity. It supports the same conceptual operations (primitives + booleans) but without face tracking. The Lignin kernel interface should be designed as a Go interface so that the backend can be swapped:

```go
// kernel.go -- abstract interface
type Kernel interface {
    Box(x, y, z float64) Solid
    Cylinder(height, radius float64, segments int) Solid
    Sphere(radius float64, segments int) Solid
    Extrude(profile Profile2D, height float64) Solid
    Union(a, b Solid) Solid
    Difference(a, b Solid) Solid
    Intersection(a, b Solid) Solid
}

// Two implementations:
// - kernel/sdfx/    (pure Go, rapid prototyping, no face tracking)
// - kernel/manifold/ (CGo, production, full face tracking)
```

This allows the team to develop the Lisp engine and design graph in parallel with the Manifold CGo bindings.

### Rejected Alternatives Summary

| Option | Reason for Rejection |
|---|---|
| OpenCASCADE | Wrapping effort disproportionate; no existing Go bindings; LGPL concerns |
| CGAL | GPL licensing; no C API; extreme wrapping complexity |
| gsdf | SDF limitations (no face tracking); GPU-focused architecture mismatch |
| celer/csg, reactivego/csg | Unmaintained; no face tracking; poor scalability |
| GhostSCAD | Not a kernel; OpenSCAD dependency; no programmatic access to results |
| libfive | SDF limitations + CGo complexity without compensating advantages |
| Fornjot | Too early-stage; no C API; double FFI bridge |
| Custom B-rep kernel | Multi-year effort; unsuitable for project scope |

---

## Appendix A: Manifold C API Key Functions (from manifoldc.h)

Based on the Manifold class reference at https://manifoldcad.org/docs/html/classmanifold_1_1_manifold.html, the C API exposes these core operations (C function names follow the pattern `manifold_<operation>`):

**Primitives:**
- `manifold_cube(alloc, x, y, z, center)` -- Create a box
- `manifold_sphere(alloc, radius, segments)` -- Create a sphere
- `manifold_cylinder(alloc, height, radius_low, radius_high, segments, center)` -- Create a cylinder
- `manifold_extrude(alloc, polygons, height, divisions, twist, scale_x, scale_y)` -- Extrude 2D profile
- `manifold_revolve(alloc, polygons, segments, degrees)` -- Revolve 2D profile

**Booleans:**
- `manifold_boolean(alloc, a, b, op)` -- General boolean (union/difference/intersection)
- `manifold_union(alloc, a, b)` -- Boolean union
- `manifold_difference(alloc, a, b)` -- Boolean difference
- `manifold_intersection(alloc, a, b)` -- Boolean intersection
- `manifold_split(alloc_a, alloc_b, a, cutter)` -- Split into intersection + difference

**Transforms:**
- `manifold_translate(alloc, m, x, y, z)` -- Translate
- `manifold_rotate(alloc, m, x, y, z)` -- Rotate (Euler angles)
- `manifold_scale(alloc, m, x, y, z)` -- Scale
- `manifold_mirror(alloc, m, nx, ny, nz)` -- Mirror

**Queries:**
- `manifold_num_vert(m)` -- Vertex count
- `manifold_num_tri(m)` -- Triangle count
- `manifold_volume(m)` -- Volume
- `manifold_surface_area(m)` -- Surface area
- `manifold_bounding_box(alloc, m)` -- Bounding box
- `manifold_original_id(m)` -- Get original ID
- `manifold_reserve_ids(n)` -- Reserve unique IDs

**Mesh I/O:**
- `manifold_get_meshgl(alloc, m)` -- Get MeshGL with face tracking data
- `manifold_meshgl_num_vert(m)` -- MeshGL vertex count
- `manifold_meshgl_num_tri(m)` -- MeshGL triangle count
- `manifold_meshgl_run_original_id(m)` -- Get original ID array
- `manifold_meshgl_face_id(m)` -- Get face ID array

**Memory:**
- `manifold_delete_manifold(m)` -- Free a manifold object
- `manifold_alloc_manifold()` -- Allocate memory for a manifold

## Appendix B: References

- Manifold Repository: https://github.com/elalish/manifold
- Manifold API Reference: https://manifoldcad.org/docs/html/classmanifold_1_1_manifold.html
- Manifold C Bindings Discussion: https://github.com/elalish/manifold/discussions/284
- Manifold Wiki: https://github.com/elalish/manifold/wiki/Manifold-Library
- sdfx Repository: https://github.com/deadsy/sdfx
- sdfx Go Package: https://pkg.go.dev/github.com/deadsy/sdfx
- soypat/gsdf Repository: https://github.com/soypat/gsdf
- OpenCASCADE: https://dev.opencascade.org/
- opencascade-rs (Rust bindings -- architectural reference): https://github.com/bschwind/opencascade-rs
- CGAL Go Bindings Issue: https://github.com/CGAL/cgal/issues/518
- Fornjot: https://github.com/hannobraun/fornjot
- libfive: https://github.com/libfive/libfive
- SDF Limitations for CAD: https://incoherency.co.uk/blog/stories/sdf-thoughts.html
- B-rep vs Implicit Modeling: https://www.ntop.com/resources/blog/understanding-the-basics-of-b-reps-and-implicits/
- CGo Documentation: https://go.dev/blog/cgo
