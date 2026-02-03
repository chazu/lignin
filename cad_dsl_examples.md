# CAD DSL Syntax Examples

## 1. zygomys-based DSL

```lisp
;; Primitive definitions
(defn box [width height depth]
  {:type "box"
   :width width
   :height height
   :depth depth})

(defn cylinder [radius height]
  {:type "cylinder"
   :radius radius
   :height height})

(defn sphere [radius]
  {:type "sphere"
   :radius radius})

;; Transformations
(defn translate [shape x y z]
  (assoc shape :position [x y z]))

(defn rotate [shape x y z]
  (assoc shape :rotation [x y z]))

(defn scale [shape factor]
  (assoc shape :scale factor))

;; Boolean operations
(defn union [a b]
  {:type "union"
   :operands [a b]})

(defn difference [a b]
  {:type "difference"
   :operands [a b]})

(defn intersect [a b]
  {:type "intersection"
   :operands [a b]})

;; Join operations
(defn butt-joint [part-a face-a part-b face-b]
  {:type "butt-joint"
   :part-a part-a
   :face-a face-a
   :part-b part-b
   :face-b face-b})

(defn mortise-tenon [mortise-part tenon-part]
  {:type "mortise-tenon"
   :mortise mortise-part
   :tenon tenon-part})

;; Example: Simple table
(def table-top (box 100 5 50))
(def leg (box 5 70 5))

(def leg1 (translate leg 10 0 10))
(def leg2 (translate leg 85 0 10))
(def leg3 (translate leg 10 0 40))
(def leg4 (translate leg 85 0 40))

(def table
  (union table-top
         (union leg1
                (union leg2
                       (union leg3 leg4)))))

;; Material and grain specification
(defn with-material [shape material-type]
  (assoc shape :material material-type))

(defn with-grain [shape direction]
  (assoc shape :grain-direction direction))

(def oak-table
  (-> table
      (with-material "oak")
      (with-grain [1 0 0])))
```

## 2. fe-based DSL (if ported to Go)

```lisp
; Primitive definitions
(def (box width height depth)
  (list 'box width height depth))

(def (cylinder radius height)
  (list 'cylinder radius height))

; Transformations
(def (translate shape x y z)
  (list 'translate shape x y z))

(def (rotate shape x y z)
  (list 'rotate shape x y z))

; Boolean operations
(def (union a b)
  (list 'union a b))

(def (difference a b)
  (list 'difference a b))

; Example usage
(set table-top (box 100 5 50))
(set leg (box 5 70 5))

(set table
  (union table-top
    (union (translate leg 10 0 10)
      (union (translate leg 85 0 10)
        (union (translate leg 10 0 40)
          (translate leg 85 0 40))))))
```

## 3. Custom Minimal Lisp DSL

```lisp
; Tailored specifically for CAD operations
; Simpler, more focused syntax

; Primitives - direct function calls
(box 100 5 50)        ; returns a box primitive
(cylinder 5 10)       ; returns a cylinder
(sphere 7)            ; returns a sphere

; Transformations - prefix operations
(translate (box 10 20 30) 50 0 0)
(rotate (cylinder 3 15) 0 45 0)
(scale (sphere 5) 2.0)

; Boolean operations
(union
  (box 10 10 10)
  (cylinder 3 10))

(difference
  (box 20 20 20)
  (cylinder 5 25))

; Join operations - domain-specific
(butt-joint
  :part-a top
  :face-a :front
  :part-b leg
  :face-b :top)

(mortise-tenon
  :mortise rail
  :tenon stile)

; Part naming and references
(defpart top (box 100 5 50))
(defpart leg (box 5 70 5))

; Grain direction
(with-grain top :x-dominant)
(with-grain leg :z-dominant)

; Material properties
(with-material top "walnut")
(with-material leg "walnut")
```

## 4. Comparison of Approaches

### zygomys advantages for DSL:
- **Rich data structures**: Native support for maps/lists
- **Go integration**: Can define Go structs and use them in Lisp
- **Mature library**: Error handling, debugging tools
- **Flexible syntax**: Can support both traditional Lisp and domain-specific forms

### fe advantages:
- **Extreme minimalism**: Very small codebase
- **Pure functional**: Good match for mathematical CAD operations
- **Macro support**: Could enable powerful DSL extensions

### Custom Lisp advantages:
- **Tailored optimization**: Can be optimized specifically for CAD operations
- **Deterministic by design**: Can enforce immutability and purity
- **Simplified syntax**: Can hide Lisp complexity for end users
- **Direct integration**: No abstraction layers between CAD engine and Lisp

## 5. Recommended Approach: zygomys

For Lignin's MVP, **zygomys is recommended** because:

1. **Time to market**: Already implemented in Go, production-ready
2. **Go integration**: Designed specifically for embedding in Go applications
3. **Reflection support**: Can easily marshal between Lisp data and Go structs
4. **Rich feature set**: Supports all data types needed for CAD
5. **Community support**: Active maintenance, documentation available
6. **Performance**: Go-native implementation provides good speed

The custom Lisp approach would be ideal long-term but would delay MVP significantly. fe would require a port from C to Go, which adds risk and development time.

zygomys provides the best balance of features, maturity, and integration with Go - which is critical for Lignin's architecture.