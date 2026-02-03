# GUI Framework Comparison for Lignin CAD
## Research for issue li-wyd

## Frameworks Evaluated
1. **Fyne** (https://fyne.io/) - Higher-level widgets, more mature, stable API
2. **Gio** (https://gioui.org/) - Lower-level, immediate mode, lighter weight

## Feature Comparison Matrix

| Feature | Fyne | Gio |
|---------|------|-----|
| **Approach** | Retained mode, higher-level widgets | Immediate mode, lower-level, declarative |
| **Stars/Activity** | 27.9k stars, 201 contributors, 12k+ commits, v2.7.2 (2026-01-06) | 2.1k stars, 62 contributors, 3k+ commits, zero major version |
| **Platform Support** | Desktop, mobile, web | Linux, macOS, Windows, Android, iOS, FreeBSD, OpenBSD, WebAssembly |
| **Widget Library** | Mature, extensive widget library | Basic widgets, but extensible |
| **Rendering** | Traditional retained mode rendering | Efficient vector renderer, migrating to compute-shader-based |
| **Community** | Larger community, more established | Smaller but active community |
| **Documentation** | Well-documented with examples | Documentation at gioui.org |
| **License** | Open source (view LICENSE file) | Open source (view LICENSE file) |
| **Development Model** | GitHub-centric | Sourcehut + mailing lists |
| **Design Philosophy** | Material Design inspired | Immediate mode, performance-focused |
| **Dependencies** | 1.5k forks | 1.8k dependent projects |

## Requirements Analysis for Lignin

### Must-Have Requirements:
1. **Native app (not web-based)** - Both support native desktop
2. **Split pane layout** - Both can implement (Fyne has built-in containers, Gio has split widget)
3. **Cross-platform: macOS, Linux** - Both support both platforms
4. **Live evaluation loop** - Both support real-time UI updates

### Code Editor Requirements:
1. **Syntax highlighting** - Neither has built-in rich text editor widgets
2. **Line numbers** - Would need custom implementation
3. **Basic editing** - Basic text input available in both
4. **Embedded editor widget evaluation** - Requires 3rd party integration or custom widget

## Code Editor Options Analysis

### Option 1: Build Custom Text Editor Widget
- Pros: Full control, tight integration
- Cons: Significant development effort, syntax highlighting complex

### Option 2: Integrate Existing Text Editor Library
- **Scintilla** (via binding): Mature, feature-rich, but C++ dependency
- **Prose** (Go text editor): Pure Go, simpler
- **Terminal-based** (tview/bubbletea): Console UI, not GUI

### Option 3: WebView with Monaco/CodeMirror
- Pros: Full-featured editors available
- Cons: Web-based, breaks native requirement, adds complexity

## PoC Considerations

### Split-pane Implementation:
1. **Fyne**: Use `container.NewHSplit()` or `container.NewVSplit()`
2. **Gio**: Use built-in Split widget

### 3D Viewport Placeholder:
1. **OpenGL integration**: Both support OpenGL context
2. **Placeholder approach**: Simple colored rectangle initially
3. **Future integration**: WebGL or custom OpenGL renderer

## Performance Considerations

### Fyne:
- Higher-level abstraction may have overhead
- Mature optimization for common use cases
- Good for rapid development

### Gio:
- Immediate mode is generally more performant
- Efficient vector renderer
- Compute-shader migration promises better performance
- Lower memory footprint

## Recommendation Factors

### For Lignin's needs:
1. **Deterministic evaluation** - Both frameworks can support
2. **Editor-driven workflow** - Both support text input
3. **No direct geometry manipulation** - Both can enforce
4. **Clear error/warning surfacing** - Both support UI updates

### Critical Decision Points:
1. **Editor complexity**: Custom editor widget will be significant work in either framework
2. **3D integration**: OpenGL support needed in both
3. **Development velocity**: Fyne may be faster for basic UI
4. **Performance**: Gio may be better for frequent UI updates

## Next Steps for PoC

1. Create minimal app in both frameworks
2. Implement split-pane layout
3. Add basic text input area (editor placeholder)
4. Add 3D viewport placeholder
5. Compare development experience and performance
6. Evaluate editor integration feasibility