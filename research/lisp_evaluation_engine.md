# Lisp Evaluation Engine Research — Lignin

**Date:** 2026-02-02
**Context:** Selecting a Lisp implementation for Lignin's evaluation engine. The engine evaluates user code on every keystroke to produce an immutable design graph for a programmable woodworking CAD tool written in Go.

---

## 1. Feature Comparison Matrix

Six candidates were evaluated: three primary options (zygomys, fe, custom) and three additional Go Lisp implementations discovered during research (glisp, SteelSeries golisp, jig/lisp).

| Feature | zygomys | fe (rxi) | Custom Minimal | glisp (zhemao) | golisp (SteelSeries) | jig/lisp |
|---|---|---|---|---|---|---|
| **Language** | Pure Go | ANSI C | Go | Pure Go | Pure Go | Pure Go |
| **GitHub Stars** | 1,779 | 1,465 | N/A | 261 | 150 | 6 |
| **Last Commit** | Dec 2025 | Jul 2024 | N/A | Jan 2020 | May 2021 | Jan 2026 |
| **License** | BSD-2 | MIT | N/A | BSD-2 | BSD-3 | MPL-2.0 |
| **Open Issues** | 8 | 19 | N/A | 3 | 3 | 0 |
| **Maintenance** | Active | Stable/frozen | N/A | Abandoned | Abandoned | Active |
| **Numbers** | int64, float64 | float (double) | Configurable | int64, float64 | int, float | int, float |
| **Strings** | Yes | Yes | Configurable | Yes | Yes | Yes |
| **Lists** | Yes (+ arrays) | Yes (pairs) | Yes | Yes (+ arrays) | Yes | Yes (+ vectors) |
| **Hash Maps** | Yes (SexpHash) | No | Configurable | Yes | Yes (frames) | Yes |
| **Symbols** | Yes | Yes | Yes | Yes | Yes | Yes + keywords |
| **Functions** | Yes (lambda) | Yes (lambda) | Yes | Yes (lambda) | Yes (lambda) | Yes (fn) |
| **Macros** | Yes | Yes | Optional | Yes | Yes | Yes |
| **Closures** | Yes | Yes (lexical) | Yes | Yes | Yes | Yes |
| **Tail-Call Opt.** | Yes | No | Optional | Yes | No | No |
| **Concurrency** | No | No | Optional | Yes (channels) | Yes (channels) | Yes (futures) |
| **Sandboxing** | Yes (built-in) | No | Configurable | No | No | No |
| **GC Strategy** | Go GC | Mark-and-sweep (fixed buffer) | Go GC | Go GC | Go GC | Go GC |
| **Interpreter Type** | Bytecode VM | Tree-walking | Configurable | Stack-based VM | Tree-walking | Tree-walking |
| **Go Struct Interop** | Excellent (reflection, togo) | None (C API) | Full control | Moderate (API) | Moderate (primitives) | Good (L-notation) |
| **Embeddability** | Excellent | Requires CGo bridge | Native | Good | Good | Good |
| **Documentation** | Wiki + examples | README + doc/ | N/A | README + godoc | README + blog | README + godoc |
| **Infix Syntax** | Yes (optional, in `{}`) | No | Configurable | No | No | No |
| **Error Handling** | Stack traces | setjmp/longjmp | Configurable | Basic | Basic | Line numbers + stack traces |
| **JSON Interop** | Yes (+ msgpack) | No | Configurable | No | Yes | Yes |

### Key Observations

**zygomys** is the most feature-complete pure-Go option with active maintenance (commits in Dec 2025), deep Go struct interop via reflection, and a bytecode VM architecture that favors performance over tree-walking interpreters.

**fe** is elegant and minimal (~800 SLOC of ANSI C) but would require a CGo bridge for Go integration, adding ~40ns overhead per call. It lacks hash maps and has no Go struct interop. The project is intentionally frozen — the author does not merge pull requests.

**jig/lisp** is the most recently updated (Jan 2026) and has a clean Go-native API with `READ`/`EVAL`/`PRINT` separation, but has almost no community adoption (6 stars) and requires Go 1.25+.

**glisp** and **golisp** are both abandoned (last commits 2020 and 2021 respectively) and should not be considered for new projects.

---

## 2. Detailed Candidate Analysis

### 2.1 zygomys (glycerine/zygomys)

**Architecture:** Bytecode-compiled VM interpreter written in pure Go. The parser compiles Lisp source into VM instructions which are then executed. This is notably faster than tree-walking for repeated evaluation.

**Go Integration API:**
```go
// Create interpreter instance
env := zygo.NewZlisp()

// Load and evaluate source code
err := env.LoadString(`(def board-width 3.5)`)
result, err := env.Run()

// Register Go functions callable from Lisp
env.AddFunction("make-board", func(env *zygo.Zlisp, name string, args []zygo.Sexp) (zygo.Sexp, error) {
    // Extract arguments, construct Go objects, return Sexp
    width := args[0].(zygo.SexpFloat).Val
    return zygo.SexpHash{...}, nil
})

// Bind Go data into Lisp environment
env.AddGlobal("PI", &zygo.SexpFloat{Val: 3.14159})
```

**Data Type Mapping:** Every Lisp value implements the `Sexp` interface. Bidirectional mapping is explicit: `SexpInt` <-> `int64`, `SexpFloat` <-> `float64`, `SexpStr` <-> `string`, `SexpArray` <-> `[]Sexp`, `SexpHash` <-> Go structs (via reflection and `togo`).

**Strengths for Lignin:**
- Bytecode VM is faster for repeated evaluation (keystroke-driven re-eval)
- Deep Go struct interop via reflection — design graph nodes can be native Go structs
- Built-in sandboxing prevents user scripts from filesystem/syscall access
- Active maintenance with commits through Dec 2025
- Deterministic evaluation with no implicit global state
- `Clear()` + `LoadString()` + `Run()` cycle supports incremental re-evaluation

**Weaknesses:**
- Larger codebase means more surface area for bugs
- The macro system and infix syntax add complexity Lignin may not need
- Some Clojure-isms may confuse users expecting standard Lisp
- Documentation, while present (wiki), is not comprehensive

**Performance:** Benchmarked at ~1.85 seconds for parsing and evaluating a 5MB JSON file. The bytecode VM avoids repeated tree traversal, which matters for keystroke-frequency re-evaluation.

---

### 2.2 fe (rxi/fe)

**Architecture:** Tree-walking interpreter in ~800 lines of ANSI C. Uses a fixed-size pre-allocated memory buffer with no malloc during execution. Mark-and-sweep GC operates within the fixed memory envelope.

**Integration with Go (via CGo):**
```go
// #include "fe.h"
import "C"
import "unsafe"

func NewFeContext(bufSize int) *C.fe_Context {
    buf := C.malloc(C.size_t(bufSize))
    return C.fe_open(buf, C.int(bufSize))
}

func Eval(ctx *C.fe_Context, code string) {
    // Would require: parse C string, call fe_read, fe_eval
    // Each call crosses CGo boundary (~40ns overhead)
}
```

**Strengths for Lignin:**
- Extremely small and auditable (~800 SLOC)
- Fixed memory model means predictable allocation — no GC pauses
- Elegant, minimal design — easy to reason about
- The fixed-buffer memory model is appealing for deterministic evaluation

**Weaknesses:**
- **CGo bridge required** — adds ~40ns per call, complicates cross-compilation, breaks `go test -race`
- No hash maps — would need to build association lists or extend fe
- No Go struct interop — all data must be marshaled across the C/Go boundary
- No tail-call optimization — deep recursion could stack overflow
- Project is intentionally frozen; author will not merge PRs
- Cannot produce macOS universal binaries without extra CGo configuration
- Cross-platform builds become significantly harder (macOS + Linux target)

**Performance:** The tree-walking interpreter is inherently slower than a bytecode VM for repeated evaluation. However, for small expressions (typical CAD DSL), the difference may be negligible. The fixed-buffer GC avoids pause-time variance.

---

### 2.3 jig/lisp

**Architecture:** Tree-walking interpreter derived from the `kanaka/mal` (Make A Lisp) implementation. Pure Go with clean `READ`/`EVAL`/`PRINT` separation.

**Go Integration API:**
```go
ns := env.NewEnv()
_ = nscore.Load(ns)

// Evaluate a string
ast, err := lisp.READ("(+ 1 2)", types.NewCursorFile("input"), ns)
result, err := lisp.EVAL(context.Background(), ast, ns)
output := lisp.PRINT(result) // "3"

// Direct AST construction from Go (L-notation)
ast := lnotation.L("+", 10, 20)
result, _ := lisp.EVAL(context.Background(), ast, ns)
```

**Strengths for Lignin:**
- Most recently maintained (commits in Jan 2026)
- Clean, idiomatic Go API
- Context-aware evaluation (`context.Background()`) — supports cancellation
- L-notation allows constructing AST directly from Go without string parsing
- Good error messages with line numbers and stack traces
- Thread-safe atoms

**Weaknesses:**
- Virtually no community adoption (6 stars, 0 forks)
- Requires Go 1.25+ (very recent)
- Tree-walking interpreter — slower than bytecode for repeated eval
- Based on mal, which is a teaching implementation, not a production runtime
- MPL-2.0 license has copyleft implications for modifications to the library itself

---

### 2.4 Custom Minimal Lisp

**Architecture:** A purpose-built Lisp interpreter for the Lignin CAD DSL, implemented in Go.

**What it would look like:**
```go
// Core types
type Value interface{ String() string }
type Number struct{ Val float64 }
type Symbol struct{ Name string }
type List   struct{ Items []Value }
type Fn     struct{ Params []Symbol; Body Value; Env *Env }

// Environment with lexical scoping
type Env struct {
    bindings map[string]Value
    parent   *Env
}

// Core loop: parse -> eval -> design graph
func Evaluate(source string) (*DesignGraph, error) {
    tokens := Tokenize(source)
    ast, err := Parse(tokens)
    if err != nil { return nil, err }
    env := NewRootEnv()     // fresh env per evaluation (deterministic)
    RegisterCADBuiltins(env) // board, joint, dimension, etc.
    result, err := Eval(ast, env)
    if err != nil { return nil, err }
    return result.(*DesignGraph), nil
}
```

**Estimated effort:** 1,500-3,000 lines of Go for a minimal interpreter with: tokenizer, recursive-descent parser, tree-walking evaluator, environment/scoping, numeric types, strings, lists, functions, `let`/`def`/`if`/`do` special forms, and error propagation.

**Strengths for Lignin:**
- Complete control over semantics, error messages, and performance
- Can enforce Lignin's constraints at the language level (no mutation, deterministic eval)
- Zero external dependencies
- Data types map directly to Go — no marshaling layer
- Can optimize specifically for incremental re-evaluation (dirty tracking, caching)
- Can design the syntax to be woodworking-friendly from day one
- Smallest possible attack surface

**Weaknesses:**
- Significant upfront engineering investment (weeks, not hours)
- Must implement and test: tokenizer, parser, evaluator, GC (or rely on Go GC), error handling
- Risk of subtle bugs in core evaluation semantics
- No community to report bugs or contribute fixes
- Every standard library function must be hand-implemented

---

## 3. Sample CAD DSL Syntax

Below is how user-facing woodworking DSL code would look under each option.

### 3.1 zygomys Syntax

```lisp
;; Define material
(def white-oak (material "White Oak" :hardwood
  :density 0.75
  :grain-direction :long))

;; Define a board with dimensions in inches
(def shelf-board
  (board "Shelf"
    :width 11.25
    :height 0.75
    :length 36.0
    :material white-oak))

;; Define a side panel
(def side-panel
  (board "Side Panel"
    :width 11.25
    :height 30.0
    :length 0.75
    :material white-oak))

;; Create a dado joint
(def shelf-dado
  (joint :dado
    :housing side-panel
    :tenon shelf-board
    :depth 0.375
    :offset-from-bottom 15.0))

;; Assemble into a bookcase
(def bookcase
  (assembly "Simple Bookcase"
    :parts [side-panel shelf-board]
    :joints [shelf-dado]))

;; Infix math also available inside {}
(def shelf-span { 36.0 - (2 * 0.75) })
```

### 3.2 fe Syntax

```lisp
;; fe uses a more minimal syntax — no keywords, no hash maps

(def white-oak (material "White Oak" "hardwood" 0.75))

(def shelf-board
  (board "Shelf" 11.25 0.75 36.0 white-oak))

(def side-panel
  (board "Side Panel" 11.25 30.0 0.75 white-oak))

(def shelf-dado
  (dado side-panel shelf-board 0.375 15.0))

;; No hash maps means assembly is positional
(def bookcase
  (assembly "Simple Bookcase"
    (list side-panel shelf-board)
    (list shelf-dado)))

;; Math is prefix only
(def shelf-span (- 36.0 (* 2 0.75)))
```

### 3.3 jig/lisp Syntax

```lisp
;; Clojure-inspired syntax with keywords and hash-maps

(def white-oak
  {:type "hardwood"
   :name "White Oak"
   :density 0.75
   :grain :long})

(def shelf-board
  (board {:name "Shelf"
          :width 11.25
          :height 0.75
          :length 36.0
          :material white-oak}))

(def side-panel
  (board {:name "Side Panel"
          :width 11.25
          :height 30.0
          :length 0.75
          :material white-oak}))

(def shelf-dado
  (joint :dado
         {:housing side-panel
          :tenon shelf-board
          :depth 0.375
          :offset-from-bottom 15.0}))

(def bookcase
  (assembly "Simple Bookcase"
    {:parts [side-panel shelf-board]
     :joints [shelf-dado]}))
```

### 3.4 Custom Minimal Lisp Syntax

```lisp
;; Purpose-built syntax: clean, no unnecessary features

(material white-oak "White Oak"
  (hardwood)
  (density 0.75)
  (grain long))

(board shelf
  (name "Shelf")
  (dim 36.0 11.25 0.75)
  (material white-oak))

(board side-panel
  (name "Side Panel")
  (dim 0.75 11.25 30.0)
  (material white-oak))

(dado shelf-joint
  (housing side-panel)
  (insert shelf)
  (depth 0.375)
  (from-bottom 15.0))

(assembly bookcase "Simple Bookcase"
  (parts shelf side-panel)
  (joints shelf-joint))

;; Functions for parametric design
(defn bookcase-section (width n-shelves spacing)
  (let (height (* n-shelves spacing))
    (board side (dim 0.75 width height) (material white-oak))
    (map (fn (i)
      (board shelf-i (dim width 0.75 11.25) (material white-oak)))
      (range n-shelves))))
```

---

## 4. Minimal PoC Description — Top Candidate: zygomys

The recommended integration approach uses zygomys as the evaluation engine. Below is a concrete description of how the parser + evaluator integration would work.

### 4.1 Architecture

```
User Code (Lisp source string)
        |
        v
  +-----------+
  | zygomys   |  LoadString() -> bytecode -> Run()
  | Zlisp VM  |
  +-----------+
        |
        v
  Design Graph (Go structs: Board, Joint, Assembly, etc.)
        |
        v
  Geometry Kernel -> Renderer
```

### 4.2 Core Integration Code

```go
package engine

import (
    "fmt"
    "sync"

    zygo "github.com/glycerine/zygomys/zygo"
)

// DesignGraph is the immutable output of evaluation.
type DesignGraph struct {
    Boards     []Board
    Joints     []Joint
    Assemblies []Assembly
    Errors     []EvalError
}

// Engine wraps the zygomys interpreter for Lignin evaluation.
type Engine struct {
    mu sync.Mutex
}

func NewEngine() *Engine {
    return &Engine{}
}

// Evaluate takes Lisp source and produces a new DesignGraph.
// Called on every keystroke. Deterministic: fresh env each time.
func (e *Engine) Evaluate(source string) (*DesignGraph, error) {
    e.mu.Lock()
    defer e.mu.Unlock()

    // Fresh interpreter per evaluation — deterministic, no mutable state
    env := zygo.NewZlispSandbox()
    defer env.Close()

    // Register CAD domain functions
    graph := &DesignGraph{}
    registerBuiltins(env, graph)

    // Load and evaluate user source
    err := env.LoadString(source)
    if err != nil {
        return nil, fmt.Errorf("parse error: %w", err)
    }

    _, err = env.Run()
    if err != nil {
        return nil, fmt.Errorf("eval error: %w", err)
    }

    return graph, nil
}

// registerBuiltins adds CAD-specific functions to the Lisp environment.
func registerBuiltins(env *zygo.Zlisp, graph *DesignGraph) {
    // (board name :width W :height H :length L :material M)
    env.AddFunction("board", func(env *zygo.Zlisp, name string, args []zygo.Sexp) (zygo.Sexp, error) {
        if len(args) < 1 {
            return zygo.SexpNull, fmt.Errorf("board requires a name")
        }
        boardName, ok := args[0].(zygo.SexpStr)
        if !ok {
            return zygo.SexpNull, fmt.Errorf("board name must be a string")
        }

        b := Board{Name: string(boardName)}
        // Parse keyword arguments from remaining args...
        // (implementation parses :width, :height, :length, :material)
        parseKeywordArgs(args[1:], &b)

        graph.Boards = append(graph.Boards, b)

        // Return a hash representing this board for use in joins
        hash, _ := zygo.MakeHash(nil)
        hash.HashSet(zygo.SexpStr("name"), &zygo.SexpStr(b.Name))
        hash.HashSet(zygo.SexpStr("id"), &zygo.SexpInt{Val: int64(len(graph.Boards) - 1)})
        return hash, nil
    })

    // (joint :type housing tenon :depth D :offset O)
    env.AddFunction("joint", func(env *zygo.Zlisp, name string, args []zygo.Sexp) (zygo.Sexp, error) {
        // Extract joint type, participating boards, parameters
        // Append to graph.Joints
        return zygo.SexpNull, nil
    })

    // (assembly name :parts [...] :joints [...])
    env.AddFunction("assembly", func(env *zygo.Zlisp, name string, args []zygo.Sexp) (zygo.Sexp, error) {
        // Build assembly from parts and joints
        // Append to graph.Assemblies
        return zygo.SexpNull, nil
    })

    // (material name :type T :density D :grain G)
    env.AddFunction("material", func(env *zygo.Zlisp, name string, args []zygo.Sexp) (zygo.Sexp, error) {
        // Return a material descriptor hash
        return zygo.SexpNull, nil
    })

    // Standard math is already built-in: +, -, *, /, etc.
}
```

### 4.3 Keystroke-Driven Evaluation Loop

```go
// In the UI event loop (pseudocode)
func onSourceChanged(newSource string) {
    // Debounce: wait 50ms after last keystroke
    graph, err := engine.Evaluate(newSource)
    if err != nil {
        ui.ShowError(err)
        return
    }
    renderer.Update(graph)
}
```

### 4.4 Incremental Re-evaluation Strategy

For MVP, full re-evaluation on each change is acceptable given zygomys's bytecode VM speed. For optimization later:

1. **Debounce** keystroke events (50-100ms window)
2. **Cache** the previous DesignGraph and diff against new output
3. **Source hashing** — skip evaluation if source hash is unchanged
4. **Future:** structural diffing of the AST to identify changed subtrees (requires custom work regardless of engine choice)

---

## 5. Final Recommendation

### Recommendation: zygomys (glycerine/zygomys)

**Confidence: High**

### Rationale

| Criterion | zygomys | Runner-up (jig/lisp) | Why zygomys wins |
|---|---|---|---|
| Go interop | Reflection-based struct mapping, `Sexp` interface, `AddFunction`/`AddGlobal` | READ/EVAL/PRINT API, L-notation | zygomys has deeper bidirectional interop; Go structs can be directly composed from Lisp |
| Performance | Bytecode VM | Tree-walking | Bytecode is faster for repeated evaluation, critical at keystroke frequency |
| Maturity | 1,779 stars, 8+ years, production use | 6 stars, ~1 year, derived from teaching impl | zygomys has vastly more real-world testing |
| Maintenance | Active (Dec 2025 commits) | Active (Jan 2026 commits) | Both active, but zygomys has longer track record |
| Sandboxing | Built-in `NewZlispSandbox()` | None | Lignin needs to prevent user scripts from filesystem access |
| Cross-platform | Pure Go, compiles everywhere Go runs | Pure Go | Tie — both are pure Go with no CGo |
| Data types | int64, float64, string, symbol, list, array, hash | int, float, string, symbol, list, vector, hash-map | Both sufficient; zygomys types map more directly to Go primitives |
| Error handling | Stack traces, `GetStackTrace()`, recoverable | Line numbers, stack traces | Both adequate; zygomys `Clear()`+reload cycle fits keystroke re-eval |

### Why not the others?

**fe:** The CGo bridge is a dealbreaker. It adds build complexity, breaks race detection, complicates cross-compilation (macOS + Linux), and provides no Go struct interop. The minimal design is beautiful but unsuited for embedding in a Go application.

**jig/lisp:** Promising API design, but essentially untested in production (6 stars, 0 forks). Building on a `mal`-derived tree-walking interpreter for a keystroke-evaluated CAD tool is risky. The MPL-2.0 license also adds friction if the library itself needs modification.

**glisp / golisp:** Both abandoned. Not viable for a new project.

**Custom Minimal Lisp:** Tempting for full control, but the engineering cost (estimated 3-6 weeks for a robust implementation with proper error handling, scoping, and GC interaction) outweighs the benefit when zygomys already provides 90% of what Lignin needs. A custom implementation remains a viable escape hatch if zygomys proves inadequate — the `Sexp` interface pattern can be replicated.

### Migration Path

If zygomys proves insufficient (performance, bugs, abandonment), the recommended migration path is:

1. **Short-term fallback:** jig/lisp — similar API surface, pure Go, active development
2. **Long-term fallback:** Custom minimal Lisp — at that point, the team will have deep knowledge of what the DSL actually needs, making a purpose-built implementation more tractable

### Getting Started

```bash
go get github.com/glycerine/zygomys/zygo
```

Then implement the PoC described in Section 4, starting with:
1. A `board` builtin that returns a `SexpHash`
2. A `material` builtin
3. An `Evaluate(source) -> DesignGraph` function
4. Wire to a simple text input that re-evaluates on change

This minimal vertical slice proves the entire pipeline: Lisp source -> zygomys evaluation -> Go design graph structs.

---

## Appendix: Sources

- [glycerine/zygomys](https://github.com/glycerine/zygomys) — Primary repository, 1,779 stars, BSD-2-Clause
- [zygomys Go API wiki](https://github.com/glycerine/zygomys/wiki/Go-API) — Embedding documentation
- [zygomys Language wiki](https://github.com/glycerine/zygomys/wiki/Language) — Language reference
- [rxi/fe](https://github.com/rxi/fe) — 1,465 stars, MIT, ~800 SLOC ANSI C
- [zhemao/glisp](https://github.com/zhemao/glisp) — 261 stars, BSD-2-Clause, ancestor of zygomys
- [SteelSeries/golisp](https://github.com/SteelSeries/golisp) — 150 stars, BSD-3-Clause, Scheme-flavored
- [jig/lisp](https://github.com/jig/lisp) — 6 stars, MPL-2.0, mal-derived, pure Go
- [CGO Performance in Go 1.21](https://shane.ai/posts/cgo-performance-in-go1.21/) — ~40ns per CGo call
- [CockroachDB: Cost and Complexity of Cgo](https://www.cockroachlabs.com/blog/the-cost-and-complexity-of-cgo/) — Production CGo analysis
- [kanaka/mal](https://github.com/kanaka/mal) — Make A Lisp, basis for jig/lisp
