import * as THREE from 'three';
import { OrbitControls } from 'three/examples/jsm/controls/OrbitControls';

/**
 * Mesh data received from the Go backend via Wails bindings.
 */
export interface MeshData {
  vertices: number[];   // flat [x0,y0,z0, x1,y1,z1, ...]
  normals: number[];    // flat [nx0,ny0,nz0, ...]
  indices: number[];    // flat [i0,i1,i2, ...] triangles
  partName: string;
  color: string;        // hex color like "#4A90D9"
}

export interface Viewport {
  /** Replace all part meshes in the scene with new geometry. Camera position is preserved. */
  updateMeshes(meshes: MeshData[]): void;
  /** Show or hide a stale-data overlay on the viewport. */
  setStale(stale: boolean): void;
  /** Dispose of all Three.js resources. Call when the viewport is removed from the DOM. */
  dispose(): void;
}

/**
 * Tag used to identify part meshes in the scene so we can remove them
 * without touching lights, grid, or other helpers.
 */
const PART_MESH_TAG = 'lignin-part';

/**
 * Creates a Three.js viewport inside the given container element.
 *
 * The viewport sets up a scene, camera, orbit controls, lighting, a ground
 * grid, and an animation loop. It exposes methods to update geometry,
 * indicate stale state, and clean up resources.
 */
export function createViewport(container: HTMLElement): Viewport {
  // ---- Scene ----
  const scene = new THREE.Scene();
  scene.background = new THREE.Color(0x2a3a4a);

  // ---- Camera ----
  // Position to comfortably see a ~600mm wide woodworking piece.
  const aspect = container.clientWidth / Math.max(container.clientHeight, 1);
  const camera = new THREE.PerspectiveCamera(50, aspect, 0.1, 100000);
  camera.position.set(500, 400, 700);
  camera.lookAt(0, 0, 0);

  // ---- Renderer ----
  const renderer = new THREE.WebGLRenderer({ antialias: true });
  renderer.setPixelRatio(window.devicePixelRatio);
  renderer.setSize(container.clientWidth, container.clientHeight);
  container.appendChild(renderer.domElement);

  // ---- OrbitControls ----
  const controls = new OrbitControls(camera, renderer.domElement);
  controls.enableDamping = true;
  controls.dampingFactor = 0.12;
  controls.target.set(0, 0, 0);
  controls.update();

  // ---- Lighting ----
  const ambientLight = new THREE.AmbientLight(0x404040);
  scene.add(ambientLight);

  const directionalLight = new THREE.DirectionalLight(0xffffff, 1.0);
  directionalLight.position.set(500, 800, 600); // upper-right-front
  scene.add(directionalLight);

  // ---- Grid ----
  const gridHelper = new THREE.GridHelper(2000, 40, 0x445566, 0x334455);
  scene.add(gridHelper);

  // ---- Stale overlay ----
  const overlay = document.createElement('div');
  overlay.style.cssText = [
    'position: absolute',
    'inset: 0',
    'background: rgba(0, 0, 0, 0.35)',
    'pointer-events: none',
    'display: none',
    'z-index: 10',
    'transition: opacity 0.2s ease',
  ].join(';');
  // Ensure the container can anchor the overlay.
  if (getComputedStyle(container).position === 'static') {
    container.style.position = 'relative';
  }
  container.appendChild(overlay);

  // ---- Tracked part meshes ----
  const partMeshes: THREE.Mesh[] = [];

  // ---- Resize handling ----
  const resizeObserver = new ResizeObserver(() => {
    const w = container.clientWidth;
    const h = Math.max(container.clientHeight, 1);
    camera.aspect = w / h;
    camera.updateProjectionMatrix();
    renderer.setSize(w, h);
  });
  resizeObserver.observe(container);

  // ---- Animation loop ----
  let animationId = 0;
  let disposed = false;

  function animate() {
    if (disposed) return;
    animationId = requestAnimationFrame(animate);
    controls.update();
    renderer.render(scene, camera);
  }
  animate();

  // ---- Public API ----

  function updateMeshes(meshes: MeshData[]): void {
    // Remove existing part meshes from the scene and free resources.
    for (const mesh of partMeshes) {
      scene.remove(mesh);
      mesh.geometry.dispose();
      const mat = mesh.material;
      if (Array.isArray(mat)) {
        mat.forEach((m) => m.dispose());
      } else {
        mat.dispose();
      }
    }
    partMeshes.length = 0;

    // Build new meshes from the incoming data.
    for (const data of meshes) {
      const geometry = new THREE.BufferGeometry();

      geometry.setAttribute(
        'position',
        new THREE.Float32BufferAttribute(data.vertices, 3),
      );
      geometry.setAttribute(
        'normal',
        new THREE.Float32BufferAttribute(data.normals, 3),
      );
      geometry.setIndex(
        new THREE.Uint32BufferAttribute(new Uint32Array(data.indices), 1),
      );

      const material = new THREE.MeshStandardMaterial({
        color: new THREE.Color(data.color),
        flatShading: false,
        metalness: 0.1,
        roughness: 0.65,
      });

      const mesh = new THREE.Mesh(geometry, material);
      mesh.userData[PART_MESH_TAG] = true;
      mesh.name = data.partName;
      scene.add(mesh);
      partMeshes.push(mesh);
    }

    // NOTE: Camera/controls are intentionally left untouched so the user's
    // orbit position persists across evaluations.
  }

  function setStale(stale: boolean): void {
    overlay.style.display = stale ? 'block' : 'none';
  }

  function dispose(): void {
    disposed = true;
    cancelAnimationFrame(animationId);
    resizeObserver.disconnect();

    // Dispose part meshes.
    for (const mesh of partMeshes) {
      scene.remove(mesh);
      mesh.geometry.dispose();
      const mat = mesh.material;
      if (Array.isArray(mat)) {
        mat.forEach((m) => m.dispose());
      } else {
        mat.dispose();
      }
    }
    partMeshes.length = 0;

    // Dispose renderer.
    renderer.dispose();
    renderer.domElement.remove();

    // Remove overlay.
    overlay.remove();

    controls.dispose();
  }

  return { updateMeshes, setStale, dispose };
}
