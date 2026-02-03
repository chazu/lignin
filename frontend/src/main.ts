import './style.css';
import {Evaluate} from '../wailsjs/go/main/App';

document.querySelector('#app')!.innerHTML = `
  <div id="lignin-app">
    <div id="viewport">
      <div id="viewport-placeholder">3D Viewport (Three.js — Phase 3)</div>
    </div>
    <div id="editor">
      <textarea id="source" spellcheck="false" placeholder="Enter Lignin Lisp code...">(defpart "shelf"
  (board :length 600 :width 300 :thickness 18 :grain :x))</textarea>
    </div>
    <div id="status-bar">
      <span id="status-text">Ready</span>
      <span id="mesh-count"></span>
    </div>
  </div>
`;

const sourceEl = document.getElementById('source') as HTMLTextAreaElement;
const statusEl = document.getElementById('status-text')!;
const meshCountEl = document.getElementById('mesh-count')!;

let debounceTimer: number | undefined;

function evaluate() {
  const source = sourceEl.value;
  statusEl.textContent = 'Evaluating...';

  Evaluate(source)
    .then((result) => {
      if (result.errors && result.errors.length > 0) {
        const msgs = result.errors.map(
          (e: any) => e.line > 0 ? `Line ${e.line}: ${e.message}` : e.message
        );
        statusEl.textContent = msgs.join('; ');
        statusEl.classList.add('error');
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

sourceEl.addEventListener('input', () => {
  clearTimeout(debounceTimer);
  debounceTimer = window.setTimeout(evaluate, 300);
});

// Initial evaluation on load.
evaluate();
