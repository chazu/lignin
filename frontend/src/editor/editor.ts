import { EditorState } from '@codemirror/state';
import {
  EditorView,
  lineNumbers,
  keymap,
  gutter,
  GutterMarker,
  Decoration,
  type DecorationSet,
} from '@codemirror/view';
import { bracketMatching } from '@codemirror/language';
import { closeBrackets, closeBracketsKeymap } from '@codemirror/autocomplete';
import {
  defaultKeymap,
  historyKeymap,
  history,
} from '@codemirror/commands';
import { search, searchKeymap } from '@codemirror/search';
import { StateEffect, StateField, RangeSet } from '@codemirror/state';
import { lispLanguage, lispHighlight } from './lisp-syntax';

// ---------------------------------------------------------------------------
// Dark theme (Catppuccin Mocha inspired, matching the existing app)
// ---------------------------------------------------------------------------

const darkTheme = EditorView.theme(
  {
    '&': {
      backgroundColor: '#1e1e2e',
      color: '#cdd6f4',
      height: '100%',
      fontSize: '14px',
    },
    '.cm-content': {
      fontFamily: '"SF Mono", "Fira Code", "Cascadia Code", monospace',
      lineHeight: '1.5',
      caretColor: '#f5e0dc',
    },
    '&.cm-focused .cm-cursor': {
      borderLeftColor: '#f5e0dc',
    },
    '.cm-gutters': {
      backgroundColor: '#181825',
      color: '#6c7086',
      border: 'none',
    },
    '.cm-activeLineGutter': {
      backgroundColor: '#1e1e2e',
      color: '#cdd6f4',
    },
    '.cm-activeLine': {
      backgroundColor: '#21213590',
    },
    '.cm-selectionBackground, &.cm-focused .cm-selectionBackground': {
      backgroundColor: '#45475a !important',
    },
    '.cm-matchingBracket': {
      backgroundColor: '#45475a',
      outline: '1px solid #89b4fa',
    },
    '.cm-searchMatch': {
      backgroundColor: '#f9e2af33',
    },
    '.cm-searchMatch.cm-searchMatch-selected': {
      backgroundColor: '#f9e2af55',
    },
    // Error gutter markers
    '.cm-error-gutter-marker': {
      color: '#f38ba8',
      fontSize: '14px',
      lineHeight: '1.5',
      paddingLeft: '2px',
    },
  },
  { dark: true },
);

// ---------------------------------------------------------------------------
// Error gutter
// ---------------------------------------------------------------------------

interface ErrorInfo {
  line: number;
  message: string;
}

const setErrorEffect = StateEffect.define<ErrorInfo[]>();

class ErrorMarker extends GutterMarker {
  constructor(readonly message: string) {
    super();
  }
  toDOM(): Node {
    const span = document.createElement('span');
    span.className = 'cm-error-gutter-marker';
    span.textContent = '\u25CF'; // filled circle
    span.title = this.message;
    return span;
  }
}

const errorField = StateField.define<RangeSet<GutterMarker>>({
  create() {
    return RangeSet.empty;
  },
  update(markers, tr) {
    for (const effect of tr.effects) {
      if (effect.is(setErrorEffect)) {
        const newMarkers: Array<{ from: number; marker: GutterMarker }> = [];
        for (const err of effect.value) {
          if (err.line >= 1 && err.line <= tr.state.doc.lines) {
            const lineStart = tr.state.doc.line(err.line).from;
            newMarkers.push({ from: lineStart, marker: new ErrorMarker(err.message) });
          }
        }
        // Sort markers by position (required by RangeSet)
        newMarkers.sort((a, b) => a.from - b.from);
        return RangeSet.of(newMarkers.map((m) => m.marker.range(m.from)));
      }
    }
    // On document changes, clear error markers (they will be re-set after next eval)
    if (tr.docChanged) {
      return RangeSet.empty;
    }
    return markers;
  },
});

const errorGutter = gutter({
  class: 'cm-error-gutter',
  markers: (view) => view.state.field(errorField),
});

// ---------------------------------------------------------------------------
// Part highlight decoration (used when clicking a mesh in the viewport)
// ---------------------------------------------------------------------------

const setHighlightLineEffect = StateEffect.define<number | null>();

const highlightLineDeco = Decoration.line({ class: 'cm-highlighted-part-line' });

const highlightLineField = StateField.define<DecorationSet>({
  create() {
    return Decoration.none;
  },
  update(decos, tr) {
    for (const effect of tr.effects) {
      if (effect.is(setHighlightLineEffect)) {
        if (effect.value === null) {
          return Decoration.none;
        }
        const lineNum = effect.value;
        if (lineNum >= 1 && lineNum <= tr.state.doc.lines) {
          const lineStart = tr.state.doc.line(lineNum).from;
          return Decoration.set([highlightLineDeco.range(lineStart)]);
        }
        return Decoration.none;
      }
    }
    if (tr.docChanged) {
      return Decoration.none;
    }
    return decos;
  },
  provide: (f) => EditorView.decorations.from(f),
});

// ---------------------------------------------------------------------------
// Defpart cursor detection helpers
// ---------------------------------------------------------------------------

/**
 * Given a document string and a cursor position, determine if the cursor is
 * inside a `(defpart "name" ...)` form. Returns the part name or null.
 */
function defpartAtCursor(doc: string, pos: number): string | null {
  // Walk backwards from pos to find the most recent unmatched `(defpart "..."`
  // We look for the pattern (defpart "name" and check bracket balance.
  const re = /\(defpart\s+"([^"]+)"/g;
  let bestMatch: string | null = null;
  let bestStart = -1;
  let match: RegExpExecArray | null;

  while ((match = re.exec(doc)) !== null) {
    const formStart = match.index;
    if (formStart > pos) break;
    // Check if pos is within this form by counting parens from formStart.
    let depth = 0;
    let formEnd = -1;
    for (let i = formStart; i < doc.length; i++) {
      if (doc[i] === '(') depth++;
      else if (doc[i] === ')') {
        depth--;
        if (depth === 0) {
          formEnd = i;
          break;
        }
      }
    }
    if (formEnd === -1) formEnd = doc.length; // unterminated
    if (pos >= formStart && pos <= formEnd) {
      bestMatch = match[1];
      bestStart = formStart;
    }
  }
  // Suppress unused variable warning -- bestStart is used for the search
  // logic above (tracking which defpart the cursor falls within).
  void bestStart;

  return bestMatch;
}

// ---------------------------------------------------------------------------
// Public API
// ---------------------------------------------------------------------------

/**
 * Create a CodeMirror 6 editor inside the given parent element.
 *
 * @param parent  - The DOM element to mount the editor into.
 * @param initialDoc - The initial document text.
 * @param onChange - Called whenever the document content changes.
 * @param onPartSelect - Called when the cursor moves into/out of a defpart block.
 * @returns The EditorView instance.
 */
export function createEditor(
  parent: HTMLElement,
  initialDoc: string,
  onChange: (doc: string) => void,
  onPartSelect?: (partName: string | null) => void,
): EditorView {
  let lastDetectedPart: string | null = null;

  const state = EditorState.create({
    doc: initialDoc,
    extensions: [
      lineNumbers(),
      history(),
      bracketMatching(),
      closeBrackets(),
      search(),
      lispLanguage,
      lispHighlight,
      darkTheme,
      errorField,
      errorGutter,
      highlightLineField,
      keymap.of([
        ...closeBracketsKeymap,
        ...defaultKeymap,
        ...historyKeymap,
        ...searchKeymap,
      ]),
      EditorView.updateListener.of((update) => {
        if (update.docChanged) {
          onChange(update.state.doc.toString());
        }
        // Detect cursor movement into/out of defpart blocks.
        if (onPartSelect && (update.docChanged || update.selectionSet)) {
          const cursor = update.state.selection.main.head;
          const doc = update.state.doc.toString();
          const partName = defpartAtCursor(doc, cursor);
          if (partName !== lastDetectedPart) {
            lastDetectedPart = partName;
            onPartSelect(partName);
          }
        }
      }),
    ],
  });

  return new EditorView({ state, parent });
}

/**
 * Display error markers in the gutter for the given lines.
 *
 * @param view   - The EditorView to update.
 * @param errors - An array of `{line, message}` objects (1-based line numbers).
 */
export function setErrors(
  view: EditorView,
  errors: Array<{ line: number; message: string }>,
): void {
  view.dispatch({ effects: setErrorEffect.of(errors) });
}

/**
 * Scroll to and visually highlight a line in the editor.
 * Pass `null` for lineNumber to clear the highlight.
 *
 * @param view       - The EditorView instance.
 * @param lineNumber - 1-based line number to highlight, or null to clear.
 */
export function highlightLine(
  view: EditorView,
  lineNumber: number | null,
): void {
  view.dispatch({ effects: setHighlightLineEffect.of(lineNumber) });

  if (lineNumber !== null && lineNumber >= 1 && lineNumber <= view.state.doc.lines) {
    const line = view.state.doc.line(lineNumber);
    view.dispatch({
      effects: EditorView.scrollIntoView(line.from, { y: 'center' }),
    });
  }
}
