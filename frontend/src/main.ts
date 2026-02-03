import './style.css';
import { Evaluate } from '../wailsjs/go/main/App';
import { createEditor, setErrors } from './editor/editor';

const DEFAULT_SOURCE = `(defpart "shelf"
  (board :length 600 :width 300 :thickness 18 :grain :x))`;

document.querySelector('#app')!.innerHTML = `
  <div id="lignin-app">
    <div id="viewport">
      <div id="viewport-placeholder">3D Viewport (Three.js — Phase 3)</div>
    </div>
    <div id="editor">
      <div id="editor-container"></div>
    </div>
    <div id="status-bar">
      <span id="status-text">Ready</span>
      <span id="mesh-count"></span>
    </div>
  </div>
`;

const statusEl = document.getElementById('status-text')!;
const meshCountEl = document.getElementById('mesh-count')!;
const editorContainer = document.getElementById('editor-container')!;

let debounceTimer: number | undefined;

function evaluate(source: string) {
  statusEl.textContent = 'Evaluating...';

  Evaluate(source)
    .then((result) => {
      if (result.errors && result.errors.length > 0) {
        const msgs = result.errors.map(
          (e: { line: number; message: string }) =>
            e.line > 0 ? `Line ${e.line}: ${e.message}` : e.message,
        );
        statusEl.textContent = msgs.join('; ');
        statusEl.classList.add('error');

        // Mark error lines in the gutter
        const lineErrors = result.errors
          .filter((e: { line: number; message: string }) => e.line > 0)
          .map((e: { line: number; message: string }) => ({
            line: e.line,
            message: e.message,
          }));
        setErrors(view, lineErrors);
      } else {
        statusEl.classList.remove('error');
        const count = result.meshes ? result.meshes.length : 0;
        statusEl.textContent = 'OK';
        meshCountEl.textContent = `${count} part${count !== 1 ? 's' : ''}`;
        // Mesh data available at result.meshes for Three.js (Phase 3)
        console.log('Eval result:', result);
      }
    })
    .catch((err) => {
      statusEl.textContent = `Error: ${err}`;
      statusEl.classList.add('error');
    });
}

function onDocChange(doc: string) {
  clearTimeout(debounceTimer);
  debounceTimer = window.setTimeout(() => evaluate(doc), 300);
}

const view = createEditor(editorContainer, DEFAULT_SOURCE, onDocChange);

// Initial evaluation on load.
evaluate(DEFAULT_SOURCE);
