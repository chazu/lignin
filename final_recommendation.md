# Final Recommendation: GUI Framework for Lignin CAD
## Issue: li-wyd • Date: 2026-02-02

## Executive Summary

**Recommended Framework: Gio (gioui.org)**

**Primary Rationale:** Gio's immediate mode architecture better aligns with Lignin's deterministic evaluation model and provides better performance for real-time code editing with live evaluation.

## Detailed Analysis

### 1. Architecture Alignment

| Aspect | Fyne | Gio | Advantage |
|--------|------|-----|-----------|
| **Evaluation Model** | Retained mode (stateful) | Immediate mode (stateless) | **Gio**: Matches Lignin's immutable design graph |
| **UI Updates** | Widget state management | Declarative re-rendering | **Gio**: Better for frequent live evaluation updates |
| **Performance** | Good for standard apps | Excellent for frequent updates | **Gio**: Lower overhead for real-time editing |

### 2. Editor Integration Feasibility

Both frameworks require significant work for a proper code editor, but:

**Fyne:**
- Higher-level widgets limit customization
- `widget.Entry` and `widget.MultiLineEntry` are basic
- Custom editor would fight framework abstractions

**Gio:**
- Immediate mode allows pixel-level control
- Can build editor from primitives more cleanly
- Better performance for syntax highlighting updates
- More aligned with custom widget development

### 3. 3D Viewport Integration

Both support OpenGL, but:
- **Gio**: Already uses efficient vector/compute-shader rendering
- **Fyne**: More abstracted OpenGL integration
- **Both**: Would need custom OpenGL context for 3D rendering

### 4. Development Experience

**Fyne Advantages:**
- Larger community (27.9k vs 2.1k stars)
- More documentation and examples
- Faster for standard UI development
- Material Design widgets reduce design work

**Gio Advantages:**
- Cleaner architecture for custom components
- Better performance characteristics
- More aligned with Lignin's engineering philosophy
- Smaller, focused API surface

### 5. Long-term Maintainability

**Risk Assessment:**

| Risk | Fyne | Gio |
|------|------|-----|
| **Community longevity** | Lower risk (larger) | Moderate risk (smaller but dedicated) |
| **API stability** | Higher (v2.7.2) | Lower (zero major version) |
| **Customization needs** | May fight framework | More accommodating |
| **Performance scaling** | May hit limits | Designed for efficiency |

## Implementation Roadmap

### Phase 1: MVP (3-4 months)
1. Basic split-pane layout with Gio
2. Simple text area with line numbers
3. Basic Lisp syntax highlighting via text coloring
4. 3D viewport placeholder
5. Live evaluation integration

### Phase 2: Enhanced Editor (3-4 months)
1. Improved syntax highlighting
2. Code completion for Lisp functions
3. Error highlighting
4. Basic editor features (copy/paste, undo)

### Phase 3: Production Editor (3-6 months)
1. Full-featured code editor
2. Advanced syntax highlighting
3. Multiple cursor support
4. Editor preferences

## Alternative: Hybrid Approach

Consider starting with **Fyne** for rapid prototyping, then migrating to **Gio** for production if needed.

**Pros:** Faster initial development, can validate UI concepts
**Cons:** Migration cost, potential architecture mismatch

## Final Decision Rationale

Choosing **Gio** because:

1. **Architectural alignment**: Immediate mode matches Lignin's stateless evaluation
2. **Performance**: Better for real-time editing with live evaluation
3. **Customization**: Cleaner path to building a custom code editor
4. **Philosophical fit**: Matches Lignin's focus on determinism and clarity

**Mitigation for Gio's risks:**
- Smaller community mitigated by clean API design
- API stability less critical for custom widgets
- Can contribute back to Gio community

## Next Immediate Actions

1. **Setup**: Create Gio-based project structure
2. **PoC Enhancement**: Extend Gio PoC with basic editor features
3. **Integration Plan**: Design Lisp evaluation engine interface
4. **Team Review**: Present findings to Lignin team

## Conclusion

**Gio** provides the better technical foundation for Lignin's specific needs, despite Fyne's advantages in general UI development. The architectural alignment with Lignin's deterministic model outweighs the community size difference.

**Recommended:** Proceed with Gio implementation.