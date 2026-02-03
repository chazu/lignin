# Lisp Evaluation Engine Research

## Options Evaluated

### 1. zygomys (https://github.com/glycerine/zygomys)

**Language:** Pure Go
**Size:** Small Go library
**Key Features:**
- 100% Go implementation
- Embeddable scripting language
- Designed for creating DSLs for Go projects
- Uses reflection to instantiate Go structs from scripts
- Modernized Lisp syntax with optional infix notation
- JSON and Msgpack support
- Provides interpreter and REPL
- Go API for integration

**Data Types:** Float (float64), Int (int64), Char, String, Symbol, List, Array, Hash
**Performance:** "Compiled-Go speed" for running Go methods on configured structs
**Embeddability:** High - native Go library, easy integration
**Production Status:** Production-tested
**Bidirectional Data Exchange:** Excellent - designed specifically for Go struct integration

### 2. fe (https://github.com/rxi/fe)

**Language:** ANSI C
**Size:** Tiny (< 800 sloc)
**Key Features:**
- Minimalist approach
- Very small codebase
- Supports macros
- Pure functional core

**Data Types:** Numbers, symbols, strings, pairs, lambdas, macros
**Performance:** Likely fast due to minimalism
**Embeddability:** Would need port to Go (significant work)
**Production Status:** Minimalist implementation
**Bidirectional Data Exchange:** Would need to be built after porting

### 3. Custom Minimal Lisp

**Language:** Go (would be built from scratch)
**Size:** Custom, optimized for CAD needs
**Key Features:**
- Tailored specifically for Lignin CAD DSL
- Could enforce immutability by design
- Pure-function bias built-in
- Deterministic evaluation guarantees
- Optimized for CAD operations
- No unnecessary features

**Data Types:** Customizable - numbers, strings, lists, functions, variables (MVP)
**Performance:** Could be optimized for CAD operations
**Embeddability:** Perfect - built as Go library
**Production Status:** New development
**Bidirectional Data Exchange:** Designed for CAD-specific data structures

## Feature Comparison Matrix

| Feature | zygomys | fe | Custom Minimal Lisp |
|---------|---------|----|---------------------|
| **Language** | Go | C | Go |
| **Embeddability in Go** | Native | Needs port | Native |
| **Data Types** | Comprehensive | Basic | Customizable |
| **Performance** | Good (Go-speed) | Likely good | Optimizable |
| **Bidirectional Data Exchange** | Excellent (reflection) | Needs work | Tailored for CAD |
| **Deterministic Evaluation** | ? | ? | Can enforce |
| **Immutable by Design** | ? | ? | Can enforce |
| **Pure-function Bias** | ? | ? | Can enforce |
| **Error Propagation** | ? | ? | Can design explicitly |
| **Incremental Re-evaluation** | Would need to build | Would need to build | Can design for it |
| **CAD DSL Optimization** | General-purpose | General-purpose | Tailored |
| **Development Effort** | Low (use existing) | High (port + adapt) | High (build from scratch) |
| **Maintenance Burden** | Community-maintained | Small community | Self-maintained |
| **Maturity** | Production-tested | Minimalist | New |
| **Macro Support** | Not mentioned | Yes | Not needed for MVP |

## Key Considerations for Lignin CAD

1. **Deterministic evaluation** - Critical for reproducible designs
2. **No global mutable state** - Required for functional purity
3. **Pure-function bias** - Aligns with mathematical CAD operations
4. **Explicit error propagation** - Important for user feedback
5. **Incremental re-evaluation** - Needed for live editing
6. **Embeddability in Go** - Must integrate with geometry kernel
7. **Evaluation speed** - Live evaluation on every keystroke
8. **Bidirectional data exchange** - Lisp ↔ Go struct communication

## Sample CAD DSL Syntax Needs

Based on PRD requirements, Lignin needs:
- Primitive definitions (box, cylinder, sphere, extrude)
- Transformations (translate, rotate, scale)
- Join operations (butt-joint, mortise-tenon, etc.)
- Part naming and references
- Grain direction specification
- Material properties
- Boolean operations (union, difference, intersection)