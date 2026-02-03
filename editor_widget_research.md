# Embedded Editor Widget Evaluation for Lignin CAD

## Requirements for Lignin's Code Editor:
1. Syntax highlighting (Lisp-like syntax)
2. Line numbers  
3. Basic editing (copy/paste, cursor movement)
4. Real-time updates for live evaluation
5. Native integration (not web-based)

## Option Analysis

### Option 1: Build Custom Editor Widget

**Pros:**
- Full control over features
- Tight integration with GUI framework
- Optimized for Lisp syntax highlighting
- No external dependencies

**Cons:**
- Significant development effort (months)
- Complex text rendering and editing logic
- Cursor management, selection, scrolling
- Requires deep GUI framework knowledge

**Estimated effort:** 3-6 months for production-ready editor

### Option 2: Integrate Scintilla via CGo Binding

**Scintilla** (https://www.scintilla.org/)
- Mature, feature-rich text editor component
- Used by Notepad++, SciTE, others
- Syntax highlighting, line numbers, folding
- C++ library, would need Go binding

**Pros:**
- Battle-tested, full-featured
- Excellent performance
- Cross-platform

**Cons:**
- C++ dependency (CGo bridge)
- Complex integration with Go GUI frameworks
- Potential licensing issues (Scintilla License)
- Increased binary size

**Go bindings available?** 
- `go-scintilla` (seems unmaintained)
- Would likely need custom CGo wrapper

### Option 3: Use Prose or Similar Go Text Editor

**Prose** (https://github.com/jdkato/prose)
- Pure Go text processing library
- Natural language processing focus
- Not a GUI text editor widget

**Other Go text libraries:**
- `golang.org/x/text`: Unicode/text processing
- `github.com/peterh/liner`: Command line editor only
- No mature GUI text editor widgets found

### Option 4: Terminal-based Editor (TUI)

**Bubble Tea** + **Lip Gloss** (Charmbracelet)
- Terminal UI framework
- Could build text editor in terminal
- Would require separate terminal window
- Breaks integrated GUI requirement

### Option 5: WebView with Monaco/CodeMirror

**Monaco Editor** (VS Code's editor)
- Full-featured, excellent syntax highlighting
- WebAssembly could potentially embed

**Pros:**
- Full-featured editor immediately available
- Excellent Lisp syntax highlighting available
- Active development

**Cons:**
- Web-based (breaks native requirement)
- WebView adds complexity and overhead
- Integration challenges with Go GUI
- Memory overhead

## Framework-Specific Considerations

### Fyne Editor Options:
1. **Custom widget using `widget.Entry` as base** - Limited to single line
2. **Multi-line text area** - Basic editing only, no syntax highlighting
3. **Embed custom OpenGL text rendering** - Complex but possible

### Gio Editor Options:
1. **Immediate mode text rendering** - More flexible for custom widgets
2. **Build editor from primitive operations** - Complex but cleaner architecture
3. **Better performance for frequent updates** - Important for live evaluation

## Recommendation for Lignin

### Short-term (PoC/MVP):
- Use basic multi-line text input widget
- Implement minimal syntax highlighting via text coloring
- Add line numbers as separate widget
- Accept limited editor functionality initially

### Medium-term (v1.0):
- Build custom editor widget in chosen framework
- Start with basic features, expand gradually
- Prioritize Lisp syntax highlighting
- Integrate with live evaluation engine

### Long-term (future):
- Consider Scintilla integration if custom editor insufficient
- Evaluate WebView if native requirement relaxes

## Next Steps:
1. Test basic text input in both Fyne and Gio
2. Implement simple syntax highlighting prototype
3. Evaluate performance of real-time updates
4. Choose framework based on editor feasibility
