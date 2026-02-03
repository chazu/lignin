import { EditorState } from '@codemirror/state';
import {
  EditorView,
  lineNumbers,
  keymap,
  gutter,
  GutterMarker,
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
// Public API
// ---------------------------------------------------------------------------

/**
 * Create a CodeMirror 6 editor inside the given parent element.
 *
 * @param parent  - The DOM element to mount the editor into.
 * @param initialDoc - The initial document text.
 * @param onChange - Called whenever the document content changes.
 * @returns The EditorView instance.
 */
export function createEditor(
  parent: HTMLElement,
  initialDoc: string,
  onChange: (doc: string) => void,
): EditorView {
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
