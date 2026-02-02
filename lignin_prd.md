# Lignin — Engineering PRD v1.0

**Project:** Lignin – Programmable CAD for Woodworking
**Implementation Language:** Go
**Audience:** Engineers / Contributors
**Status:** v1.0

---

## 1. System Objectives

Lignin provides a deterministic, code-driven modeling environment for woodworking designs.
The system prioritizes semantic clarity (what a part *is* and *why it exists*) over direct geometric manipulation.

---

## 2. Architectural Overview

Lignin is composed of four primary subsystems:

- Lisp Evaluation Engine
- Semantic Design Graph
- Geometry Kernel
- Rendering & Visualization Layer
- Minimal UI Shell

Subsystems communicate exclusively through immutable data structures.

---

## 3. Lisp Evaluation Engine

The Lisp engine evaluates user code into an immutable AST and produces a design graph.

### Requirements
- Deterministic evaluation
- No global mutable state
- Pure-function bias
- Explicit error propagation
- Incremental re-evaluation on source changes

---

## 4. Design Graph

The design graph represents primitives, transformations, and join operations.

### Properties
- Directed acyclic graph (DAG)
- Stable node identities
- Nodes reference source expressions
- Nodes may emit zero or more solids

The design graph is immutable and re-derived on evaluation.

---

## 5. Geometry Kernel

The geometry kernel performs solid modeling and boolean operations.

### Requirements
- B-rep or half-edge representation
- Tolerance-aware boolean operations
- Stable face identity tracking
- Semantic tagging of faces and edges

The kernel must be deterministic and free of UI concerns.

---

## 6. Joinery System

Joinery operations are higher-level semantic constructs layered atop the geometry kernel.

### Join Requirements
- Accept semantic parts as input
- Emit modified solids
- Preserve provenance metadata
- Validate grain and dimensional feasibility

Joinery encodes woodworking intent, not just boolean subtraction.

---

## 7. Stock & Material Model

Stock mapping is **optional and advisory**.

### Rules
- Parts may be abstract or stock-bound
- Stock allocation does not affect geometry
- Allocation failures produce warnings, not errors
- Material intent and grain direction exist independently of stock

This supports progressive refinement from design to build planning.

---

## 8. Validation & Diagnostics

Validation operates in tiers:

1. Geometry-only validation
2. Material-aware validation
3. Stock-aware validation

Only geometric impossibilities produce fatal errors.

---

## 9. Rendering Layer

### Responsibilities
- Visualize solids
- Highlight joins and diagnostics
- Support exploded views

The renderer must never mutate design state.

---

## 10. UI Shell

The UI shell binds the editor and viewport.

### Requirements
- Editor-driven workflow
- Live evaluation
- No direct geometry manipulation
- Clear error and warning surfacing

---

## 11. Output Generation

Outputs are derived from the design graph:

- Abstract cut lists
- Optional stock-grouped cut lists
- Dimensioned views

All outputs must be reproducible from source code alone.

---

## 12. Non-Functional Requirements

- Cross-platform support (macOS, Linux)
- Deterministic builds
- Version-control friendly
- Fast incremental re-evaluation

---

## 13. Explicit Non-Goals

- CNC / CAM
- Automatic nesting or optimization
- Constraint solvers
- Collaborative editing
- Photorealistic rendering

---

## 14. Definition of Done

Lignin v1.0 is complete when:

- Full furniture designs are modeled entirely in Lisp
- The design graph is inspectable and traceable
- Outputs match real-world woodworking expectations
- Stock mapping remains optional
- No UI-driven modeling paths exist
