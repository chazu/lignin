import './style.css';
import { Evaluate } from '../wailsjs/go/main/App';
import { createEditor, setErrors, highlightLine } from './editor/editor';
import { createViewport, type MeshData } from './viewport/viewport';
import { createFileManager, type FileState } from './file-io';
import type { EditorView } from '@codemirror/view';

// ---------------------------------------------------------------------------
// Default source
// ---------------------------------------------------------------------------

const DEFAULT_SOURCE = `;; Open-top box -- 5 boards with butt joints
;; Outer: 400 wide x 300 deep x 200 tall (mm)
;; Board :length/:width/:thickness map to X/Y/Z axes.

(def thickness 19)
(def oak (material :species "white-oak"))

;; Front and back panels (vertical, facing Z)
(defpart "front"
  (board :length 400 :width 200 :thickness thickness
         :grain :x :material oak))

(defpart "back"
  (board :length 400 :width 200 :thickness thickness
         :grain :x :material oak))

;; Side panels (vertical, facing X)
(defpart "left"
  (board :length thickness :width 200 :thickness 262
         :grain :z :material oak))

(defpart "right"
  (board :length thickness :width 200 :thickness 262
         :grain :z :material oak))

;; Bottom panel (horizontal)
(defpart "bottom"
  (board :length 362 :width thickness :thickness 262
         :grain :x :material oak))

(assembly "box"
  (place (part "front")  :at (vec3 0 0 0))
  (place (part "back")   :at (vec3 0 0 281))
  (place (part "left")   :at (vec3 0 0 19))
  (place (part "right")  :at (vec3 381 0 19))
  (place (part "bottom") :at (vec3 19 0 19))

  (butt-joint
    :part-a (part "front") :face-a :left
    :part-b (part "left")  :face-b :front
    :fasteners
      (list
        (screw :diameter 4 :length 50 :position (vec3 0 50 0))
        (screw :diameter 4 :length 50 :position (vec3 0 150 0))))

  (butt-joint
    :part-a (part "front") :face-a :right
    :part-b (part "right") :face-b :front
    :fasteners
      (list
        (screw :diameter 4 :length 50 :position (vec3 0 50 0))
        (screw :diameter 4 :length 50 :position (vec3 0 150 0)))))`;

// ---------------------------------------------------------------------------
// DOM structure
// ---------------------------------------------------------------------------

document.querySelector('#app')!.innerHTML = `
  <div id="lignin-app">
    <div id="viewport"></div>
    <div id="divider"></div>
    <div id="editor-container"></div>
    <div id="status-bar">
      <span id="status-text">Ready</span>
      <span id="file-state"></span>
      <span id="mesh-count"></span>
    </div>
  </div>
`;

const ligninApp = document.getElementById('lignin-app')!;
const viewportEl = document.getElementById('viewport')!;
const dividerEl = document.getElementById('divider')!;
const editorContainer = document.getElementById('editor-container')!;
const statusEl = document.getElementById('status-text')!;
const meshCountEl = document.getElementById('mesh-count')!;
const fileStateEl = document.getElementById('file-state')!;

// ---------------------------------------------------------------------------
// Three.js Viewport
// ---------------------------------------------------------------------------

const viewport = createViewport(viewportEl);

// ---------------------------------------------------------------------------
// CodeMirror Editor
// ---------------------------------------------------------------------------

let view: EditorView;
let debounceTimer: number | undefined;
let evalGeneration = 0;

function onDocChange(doc: string): void {
  fileManager.markDirty();
  clearTimeout(debounceTimer);
  debounceTimer = window.setTimeout(() => evaluate(doc), 300);
}

view = createEditor(editorContainer, DEFAULT_SOURCE, onDocChange, (partName) => {
  // Editor cursor moved into/out of a defpart block -- highlight in viewport.
  viewport.highlightPart(partName);
});

// ---------------------------------------------------------------------------
// Viewport -> Editor: clicking a mesh highlights defpart in the editor
// ---------------------------------------------------------------------------

viewport.onPartSelect = (partName: string) => {
  const doc = view.state.doc.toString();
  // Search for (defpart "partName" in the document.
  const needle = `(defpart "${partName}"`;
  const idx = doc.indexOf(needle);
  if (idx !== -1) {
    const line = view.state.doc.lineAt(idx);
    highlightLine(view, line.number);
  }
};

// ---------------------------------------------------------------------------
// File Manager
// ---------------------------------------------------------------------------

function getContent(): string {
  return view.state.doc.toString();
}

function setContent(content: string): void {
  view.dispatch({
    changes: { from: 0, to: view.state.doc.length, insert: content },
  });
}

function onFileStateChange(state: FileState): void {
  if (state.path) {
    const filename = state.path.split('/').pop()!.split('\\').pop()!;
    fileStateEl.textContent = filename + (state.dirty ? ' *' : '');
  } else {
    fileStateEl.textContent = state.dirty ? 'Untitled *' : '';
  }
}

const fileManager = createFileManager(getContent, setContent, onFileStateChange);

// ---------------------------------------------------------------------------
// Evaluation loop with generation counter
// ---------------------------------------------------------------------------

interface EvalError {
  line: number;
  message: string;
}

interface EvalResult {
  meshes?: MeshData[];
  errors?: EvalError[];
}

function evaluate(source: string): void {
  evalGeneration++;
  const thisGeneration = evalGeneration;

  statusEl.textContent = 'Evaluating...';
  statusEl.classList.remove('error');

  Evaluate(source)
    .then((result: EvalResult) => {
      // Discard stale results from older evaluations that completed late.
      if (thisGeneration !== evalGeneration) return;

      if (result.errors && result.errors.length > 0) {
        // Error path: keep last meshes, dim viewport, show gutter errors.
        viewport.setStale(true);

        const lineErrors = result.errors
          .filter((e) => e.line > 0)
          .map((e) => ({ line: e.line, message: e.message }));
        setErrors(view, lineErrors);

        const msgs = result.errors.map((e) =>
          e.line > 0 ? `Line ${e.line}: ${e.message}` : e.message,
        );
        statusEl.textContent = msgs.join('; ');
        statusEl.classList.add('error');
      } else {
        // Success path: update meshes, clear stale, clear errors.
        const meshes: MeshData[] = result.meshes ?? [];
        viewport.updateMeshes(meshes);
        viewport.setStale(false);
        setErrors(view, []);

        statusEl.classList.remove('error');
        statusEl.textContent = 'OK';
        const count = meshes.length;
        meshCountEl.textContent = `${count} part${count !== 1 ? 's' : ''}`;
      }
    })
    .catch((err: unknown) => {
      if (thisGeneration !== evalGeneration) return;

      viewport.setStale(true);
      statusEl.textContent = `Error: ${err}`;
      statusEl.classList.add('error');
    });
}

// ---------------------------------------------------------------------------
// Keyboard shortcuts
// ---------------------------------------------------------------------------

document.addEventListener('keydown', (e: KeyboardEvent) => {
  const mod = e.metaKey || e.ctrlKey;

  if (mod && e.key === 's') {
    e.preventDefault();
    fileManager.save();
  }

  if (mod && e.key === 'o') {
    e.preventDefault();
    fileManager.open().then(() => {
      // After opening a file, trigger evaluation with the new content.
      const content = getContent();
      clearTimeout(debounceTimer);
      evaluate(content);
    });
  }
});

// ---------------------------------------------------------------------------
// Draggable divider
// ---------------------------------------------------------------------------

function initDividerDrag(): void {
  let isDragging = false;
  let startY = 0;
  let startViewportHeight = 0;
  let totalHeight = 0;

  const DIVIDER_HEIGHT = 4;
  const STATUS_BAR_HEIGHT = 28;
  const MIN_PANEL_HEIGHT = 60;

  dividerEl.addEventListener('pointerdown', (e: PointerEvent) => {
    isDragging = true;
    startY = e.clientY;
    startViewportHeight = viewportEl.getBoundingClientRect().height;
    totalHeight = ligninApp.getBoundingClientRect().height;

    dividerEl.classList.add('active');
    dividerEl.setPointerCapture(e.pointerId);
    document.body.style.cursor = 'row-resize';
    // Prevent text selection while dragging
    document.body.style.userSelect = 'none';
  });

  dividerEl.addEventListener('pointermove', (e: PointerEvent) => {
    if (!isDragging) return;

    const dy = e.clientY - startY;
    const availableHeight = totalHeight - DIVIDER_HEIGHT - STATUS_BAR_HEIGHT;
    let newViewportHeight = startViewportHeight + dy;

    // Clamp
    newViewportHeight = Math.max(MIN_PANEL_HEIGHT, newViewportHeight);
    newViewportHeight = Math.min(
      availableHeight - MIN_PANEL_HEIGHT,
      newViewportHeight,
    );

    const editorHeight = availableHeight - newViewportHeight;

    ligninApp.style.gridTemplateRows =
      `${newViewportHeight}px ${DIVIDER_HEIGHT}px ${editorHeight}px ${STATUS_BAR_HEIGHT}px`;
  });

  const stopDrag = () => {
    if (!isDragging) return;
    isDragging = false;
    dividerEl.classList.remove('active');
    document.body.style.cursor = '';
    document.body.style.userSelect = '';
  };

  dividerEl.addEventListener('pointerup', stopDrag);
  dividerEl.addEventListener('pointercancel', stopDrag);
}

initDividerDrag();

// ---------------------------------------------------------------------------
// Initial evaluation
// ---------------------------------------------------------------------------

evaluate(DEFAULT_SOURCE);
