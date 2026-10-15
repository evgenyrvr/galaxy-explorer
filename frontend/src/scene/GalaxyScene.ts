import * as THREE from 'three';
import { OrbitControls } from 'three/examples/jsm/controls/OrbitControls.js';
import { ComponentState, fetchMarkers } from '../api';
import { markerFragment, markerVertex } from './shaders';

/** Style → rendering parameters. Purely scenographic. */
const STYLE: Record<string, { blending: THREE.Blending; minPx: number; maxPx: number; sharp: number; order: number }> = {
  dm: { blending: THREE.AdditiveBlending, minPx: 2, maxPx: 90, sharp: 0, order: 0 },
  gas: { blending: THREE.AdditiveBlending, minPx: 1.5, maxPx: 70, sharp: 0, order: 1 },
  stars: { blending: THREE.AdditiveBlending, minPx: 1.2, maxPx: 10, sharp: 0.3, order: 2 },
  clusters: { blending: THREE.AdditiveBlending, minPx: 3, maxPx: 9, sharp: 1, order: 3 },
  jet: { blending: THREE.AdditiveBlending, minPx: 1.5, maxPx: 12, sharp: 0.2, order: 3 },
  // Dust only absorbs: dst × (1 − a), a crude extinction of the light behind it.
  dust: { blending: THREE.CustomBlending, minPx: 1.5, maxPx: 40, sharp: 0, order: 4 },
};

interface Layer {
  id: string;
  points?: THREE.Points;
  sprite?: THREE.Sprite;
  material?: THREE.ShaderMaterial;
  geomKey: string;
  n: number;
  count: number;
  tags?: Float32Array;
  loading?: Promise<void>;
}

export interface CameraState {
  position: [number, number, number];
  target: [number, number, number];
  up: [number, number, number];
}

export interface SceneStats {
  fps: number;
  visibleMarkers: number;
  kpcPerPixel: number;
  distance: number;
}

export type PickHandler = (id: string | null, ev: MouseEvent) => void;

/** Markers of one component inside a screen rectangle. */
export interface BoxHit {
  id: string;
  inside: number;
  shown: number;
}
export type BoxHandler = (hits: BoxHit[], additive: boolean) => void;

export class GalaxyScene {
  readonly renderer: THREE.WebGLRenderer;
  readonly scene = new THREE.Scene();
  readonly camera: THREE.PerspectiveCamera;
  readonly controls: OrbitControls;
  private layers = new Map<string, Layer>();
  private comps = new Map<string, ComponentState>();
  private container: HTMLElement;
  private raf = 0;
  private frames = 0;
  private lastFps = performance.now();
  private fps = 0;
  private visible = new Set<string>();
  private filterMask = 0;
  private filterOr = false;
  private highlightMask = 0;
  private selected = new Set<string>();
  private region: { kind: number; rMin: number; rMax: number; zMax: number } = { kind: 0, rMin: 0, rMax: 0, zMax: 0 };
  private haloGroup = new THREE.Group();
  private labels = new Map<string, { el: HTMLDivElement; pos: THREE.Vector3 }>();
  private gizmoScene = new THREE.Scene();
  private gizmoCamera = new THREE.PerspectiveCamera(50, 1, 0.1, 10);
  private pickHandler: PickHandler | null = null;
  private contextHandler: PickHandler | null = null;
  private boxHandler: BoxHandler | null = null;
  private box: { x0: number; y0: number; el: HTMLDivElement } | null = null;
  private downAt: { x: number; y: number } | null = null;
  private resizeObserver: ResizeObserver;
  onError: (msg: string) => void = () => {};

  constructor(container: HTMLElement) {
    this.container = container;
    this.renderer = new THREE.WebGLRenderer({ antialias: true, logarithmicDepthBuffer: true, powerPreference: 'high-performance' });
    this.renderer.setPixelRatio(Math.min(window.devicePixelRatio, 2));
    this.renderer.setClearColor(0x05070d, 1);
    container.appendChild(this.renderer.domElement);
    this.camera = new THREE.PerspectiveCamera(50, 1, 1e-6, 1e5);
    this.camera.up.set(0, 0, 1);
    this.camera.position.set(45, -45, 30);
    this.controls = new OrbitControls(this.camera, this.renderer.domElement);
    this.controls.enableDamping = true;
    this.controls.dampingFactor = 0.08;
    this.controls.zoomSpeed = 1.4;
    this.controls.minDistance = 0.0005; // 0.5 pc
    this.controls.maxDistance = 3000;
    this.controls.screenSpacePanning = true;
    this.scene.add(this.haloGroup);
    this.buildGizmo();
    this.resizeObserver = new ResizeObserver(() => this.resize());
    this.resizeObserver.observe(container);
    this.resize();
    const el = this.renderer.domElement;
    // Alt + drag: box selection (the orbit controls are paused meanwhile).
    el.addEventListener('pointerdown', (e) => {
      if (e.button === 0 && e.altKey) {
        e.preventDefault();
        e.stopImmediatePropagation();
        this.controls.enabled = false;
        const div = document.createElement('div');
        div.className = 'box-select';
        this.container.appendChild(div);
        this.box = { x0: e.clientX, y0: e.clientY, el: div };
        el.setPointerCapture(e.pointerId);
        return;
      }
      this.downAt = { x: e.clientX, y: e.clientY };
    }, { capture: true });
    el.addEventListener('pointermove', (e) => {
      if (!this.box) return;
      const r = this.container.getBoundingClientRect();
      const x = Math.min(this.box.x0, e.clientX) - r.left, y = Math.min(this.box.y0, e.clientY) - r.top;
      Object.assign(this.box.el.style, { left: `${x}px`, top: `${y}px`, width: `${Math.abs(e.clientX - this.box.x0)}px`, height: `${Math.abs(e.clientY - this.box.y0)}px` });
    });
    el.addEventListener('pointerup', (e) => {
      if (this.box) {
        const b = this.box;
        this.box = null;
        b.el.remove();
        this.controls.enabled = true;
        if (Math.abs(e.clientX - b.x0) > 3 && Math.abs(e.clientY - b.y0) > 3) {
          this.boxHandler?.(this.boxSelect(b.x0, b.y0, e.clientX, e.clientY), e.ctrlKey || e.shiftKey || e.metaKey);
        }
        return;
      }
    }, { capture: true });
    el.addEventListener('pointerup', (e) => {
      if (!this.downAt || e.button !== 0) return;
      const moved = Math.hypot(e.clientX - this.downAt.x, e.clientY - this.downAt.y);
      this.downAt = null;
      if (moved < 4) this.pickHandler?.(this.pick(e), e);
    });
    el.addEventListener('contextmenu', (e) => {
      e.preventDefault();
      this.contextHandler?.(this.pick(e), e);
    });
    this.loop();
  }

  dispose() {
    cancelAnimationFrame(this.raf);
    this.resizeObserver.disconnect();
    this.controls.dispose();
    for (const l of this.layers.values()) this.disposeLayer(l);
    this.renderer.dispose();
    this.renderer.domElement.remove();
    for (const l of this.labels.values()) l.el.remove();
  }

  onPick(h: PickHandler) {
    this.pickHandler = h;
  }

  onContext(h: PickHandler) {
    this.contextHandler = h;
  }

  onBox(h: BoxHandler) {
    this.boxHandler = h;
  }

  /** Counts drawn markers of each component inside a client-space rectangle. */
  private boxSelect(x0: number, y0: number, x1: number, y1: number): BoxHit[] {
    const rect = this.renderer.domElement.getBoundingClientRect();
    const toNdc = (x: number, y: number) => [((x - rect.left) / rect.width) * 2 - 1, -((y - rect.top) / rect.height) * 2 + 1];
    const [ax, ay] = toNdc(Math.min(x0, x1), Math.max(y0, y1));
    const [bx, by] = toNdc(Math.max(x0, x1), Math.min(y0, y1));
    this.camera.updateMatrixWorld();
    const m = new THREE.Matrix4().multiplyMatrices(this.camera.projectionMatrix, this.camera.matrixWorldInverse).elements;
    const hits: BoxHit[] = [];
    for (const l of this.layers.values()) {
      const vis = this.visible.has(l.id);
      if (l.sprite) {
        if (!vis) continue;
        const p = new THREE.Vector3(0, 0, 0).project(this.camera);
        hits.push({ id: l.id, shown: 1, inside: p.x >= ax && p.x <= bx && p.y >= ay && p.y <= by && p.z < 1 ? 1 : 0 });
        continue;
      }
      if (!l.points || !l.tags || !l.points.visible) continue;
      const pos = l.points.geometry.getAttribute('position').array as Float32Array;
      let inside = 0, shown = 0;
      for (let i = 0; i < l.tags.length; i++) {
        const inTag = this.filterMask === 0 || (l.tags[i] & this.filterMask) !== 0;
        const ok = this.filterMask === 0 ? vis : this.filterOr ? vis || inTag : vis && inTag;
        const x = pos[3 * i], y = pos[3 * i + 1], z = pos[3 * i + 2];
        if (!ok || !this.inRegion(x, y, z)) continue;
        shown++;
        const w = m[3] * x + m[7] * y + m[11] * z + m[15];
        if (w <= 0) continue;
        const px = (m[0] * x + m[4] * y + m[8] * z + m[12]) / w;
        const py = (m[1] * x + m[5] * y + m[9] * z + m[13]) / w;
        if (px >= ax && px <= bx && py >= ay && py <= by) inside++;
      }
      if (shown > 0) hits.push({ id: l.id, inside, shown });
    }
    return hits;
  }

  private resize() {
    const w = this.container.clientWidth || 1;
    const h = this.container.clientHeight || 1;
    this.renderer.setSize(w, h, false);
    this.renderer.domElement.style.width = `${w}px`;
    this.renderer.domElement.style.height = `${h}px`;
    this.camera.aspect = w / h;
    this.camera.updateProjectionMatrix();
  }

  /** Pixels per world unit at unit distance, in drawing-buffer pixels. */
  private pixelScale(): number {
    const h = this.renderer.domElement.height;
    return h / (2 * Math.tan((this.camera.fov * Math.PI) / 360));
  }

  // -------------------------------------------------------------------------
  // Components

  /** Creates/updates layers; refetches markers when geometry or count changes. */
  async setComponents(list: ComponentState[], quality: number) {
    const ids = new Set(list.map((c) => c.id));
    for (const [id, l] of this.layers) {
      if (!ids.has(id)) {
        this.disposeLayer(l);
        this.layers.delete(id);
      }
    }
    this.comps = new Map(list.map((c) => [c.id, c]));
    const jobs: Promise<void>[] = [];
    for (const c of list) {
      const n = c.fixedCount > 0 ? c.fixedCount : Math.max(1, Math.round(c.markers * quality));
      const l = this.layers.get(c.id);
      if (l && l.geomKey === c.geomKey && l.n === n) {
        this.applyStyle(l, c);
        continue;
      }
      jobs.push(this.loadLayer(c, n));
    }
    await Promise.all(jobs);
    this.applyState();
  }

  private async loadLayer(c: ComponentState, n: number) {
    if (c.visual.style === 'icon') {
      const old = this.layers.get(c.id);
      if (old?.sprite) return;
      const sprite = new THREE.Sprite(new THREE.SpriteMaterial({ map: blackHoleTexture(), sizeAttenuation: false, depthTest: false, transparent: true }));
      sprite.scale.set(0.035, 0.035, 1);
      sprite.renderOrder = 10;
      sprite.userData.componentId = c.id;
      this.scene.add(sprite);
      this.layers.set(c.id, { id: c.id, sprite, geomKey: c.geomKey, n: 1, count: 1 });
      this.addLabel(c.id, 'SMBH · schematic, not to scale', new THREE.Vector3(0, 0, 0));
      return;
    }
    try {
      const m = await fetchMarkers(c.id, n);
      const geom = new THREE.BufferGeometry();
      geom.setAttribute('position', new THREE.BufferAttribute(m.positions, 3));
      geom.setAttribute('tags', new THREE.BufferAttribute(m.tags, 1));
      geom.computeBoundingSphere();
      const st = STYLE[c.visual.style] ?? STYLE.stars;
      const material = new THREE.ShaderMaterial({
        glslVersion: THREE.GLSL3,
        vertexShader: markerVertex,
        fragmentShader: markerFragment,
        transparent: true,
        depthWrite: false,
        blending: st.blending,
        uniforms: {
          uSize: { value: c.visual.size },
          uScale: { value: this.pixelScale() },
          uMinPx: { value: st.minPx },
          uMaxPx: { value: st.maxPx },
          uVisible: { value: 1 },
          uFilterMask: { value: 0 },
          uFilterOr: { value: 0 },
          uHighlightMask: { value: 0 },
          uSelected: { value: 0 },
          uDim: { value: 0 },
          uRegion: { value: 0 },
          uRMin: { value: 0 },
          uRMax: { value: 0 },
          uZMax: { value: 0 },
          uColor: { value: new THREE.Color(c.visual.color) },
          uHiColor: { value: new THREE.Color('#7fe3ff') },
          uOpacity: { value: c.visual.opacity },
          uSharp: { value: st.sharp },
        },
      });
      if (st.blending === THREE.CustomBlending) {
        material.blendEquation = THREE.AddEquation;
        material.blendSrc = THREE.ZeroFactor;
        material.blendDst = THREE.OneMinusSrcAlphaFactor;
      }
      const points = new THREE.Points(geom, material);
      points.renderOrder = st.order;
      points.frustumCulled = false;
      points.userData.componentId = c.id;
      const old = this.layers.get(c.id);
      if (old) this.disposeLayer(old);
      this.scene.add(points);
      this.layers.set(c.id, { id: c.id, points, material, geomKey: c.geomKey, n, count: m.count, tags: m.tags });
    } catch (e) {
      this.onError(String(e));
    }
  }

  private applyStyle(l: Layer, c: ComponentState) {
    if (!l.material) return;
    l.material.uniforms.uSize.value = c.visual.size;
    l.material.uniforms.uOpacity.value = c.visual.opacity;
    (l.material.uniforms.uColor.value as THREE.Color).set(c.visual.color);
  }

  private disposeLayer(l: Layer) {
    if (l.points) {
      this.scene.remove(l.points);
      l.points.geometry.dispose();
      l.material?.dispose();
    }
    if (l.sprite) {
      this.scene.remove(l.sprite);
      l.sprite.material.dispose();
      const lab = this.labels.get(l.id);
      lab?.el.remove();
      this.labels.delete(l.id);
    }
  }

  // -------------------------------------------------------------------------
  // Visibility, filters, selection

  setVisibility(visible: Set<string>) {
    this.visible = new Set(visible);
    this.applyState();
  }

  /** Structural filter: tag bitmask; orMode switches "matter AND structure" to OR. */
  setFilter(mask: number, orMode: boolean) {
    this.filterMask = mask;
    this.filterOr = orMode;
    this.applyState();
  }

  setSelection(components: Set<string>, tagMask: number) {
    this.selected = new Set(components);
    this.highlightMask = tagMask;
    this.applyState();
  }

  /** Hides markers outside a sphere (kind 1) or cylinder slab (kind 2); kind 0 disables. */
  setRegion(kind: number, rMin: number, rMax: number, zMax: number) {
    this.region = { kind, rMin, rMax, zMax };
    this.applyState();
  }

  private inRegion(x: number, y: number, z: number): boolean {
    const g = this.region;
    if (g.kind === 1) {
      const r = Math.hypot(x, y, z);
      return r >= g.rMin && r < g.rMax;
    }
    if (g.kind === 2) {
      const R = Math.hypot(x, y);
      return R >= g.rMin && R < g.rMax && (g.zMax <= 0 || Math.abs(z) < g.zMax);
    }
    return true;
  }

  private applyState() {
    const anySel = this.selected.size > 0 || this.highlightMask !== 0;
    for (const l of this.layers.values()) {
      const vis = this.visible.has(l.id);
      if (l.sprite) {
        l.sprite.visible = vis;
        const lab = this.labels.get(l.id);
        if (lab) lab.el.style.display = vis ? '' : 'none';
        continue;
      }
      if (!l.material || !l.points) continue;
      const u = l.material.uniforms;
      u.uVisible.value = vis ? 1 : 0;
      u.uFilterMask.value = this.filterMask;
      u.uFilterOr.value = this.filterOr ? 1 : 0;
      u.uHighlightMask.value = this.highlightMask;
      u.uSelected.value = this.selected.has(l.id) ? 1 : 0;
      u.uDim.value = anySel && !this.selected.has(l.id) ? 1 : 0;
      u.uRegion.value = this.region.kind;
      u.uRMin.value = this.region.rMin;
      u.uRMax.value = this.region.rMax;
      u.uZMax.value = this.region.zMax;
      // Skip draw calls for layers that cannot show anything.
      l.points.visible = vis || (this.filterMask !== 0 && this.filterOr && this.layerHasTag(l, this.filterMask));
    }
  }

  private layerHasTag(l: Layer, mask: number) {
    if (!l.tags) return false;
    for (let i = 0; i < l.tags.length; i++) if ((l.tags[i] & mask) !== 0) return true;
    return false;
  }

  /** Number of markers currently drawn (after matter and structural filters). */
  visibleMarkerCount(): number {
    let n = 0;
    for (const l of this.layers.values()) {
      const vis = this.visible.has(l.id);
      if (l.sprite) {
        n += vis && this.inRegion(0, 0, 0) ? 1 : 0;
        continue;
      }
      if (!l.tags || !l.points) continue;
      if (this.filterMask === 0 && this.region.kind === 0) {
        n += vis ? l.count : 0;
        continue;
      }
      const pos = l.points.geometry.getAttribute('position').array as Float32Array;
      for (let i = 0; i < l.tags.length; i++) {
        const inTag = this.filterMask === 0 || (l.tags[i] & this.filterMask) !== 0;
        const shown = this.filterMask === 0 ? vis : this.filterOr ? vis || inTag : vis && inTag;
        if (shown && this.inRegion(pos[3 * i], pos[3 * i + 1], pos[3 * i + 2])) n++;
      }
    }
    return n;
  }

  setHaloSphere(radius: number, show: boolean) {
    this.haloGroup.clear();
    const lab = this.labels.get('__r200');
    if (lab) {
      lab.el.remove();
      this.labels.delete('__r200');
    }
    if (!show || !(radius > 0)) return;
    const mat = new THREE.LineBasicMaterial({ color: 0x4a5a80, transparent: true, opacity: 0.55 });
    const pts: THREE.Vector3[] = [];
    for (let i = 0; i <= 256; i++) {
      const a = (i / 256) * Math.PI * 2;
      pts.push(new THREE.Vector3(Math.cos(a) * radius, Math.sin(a) * radius, 0));
    }
    const ring = new THREE.BufferGeometry().setFromPoints(pts);
    for (const rot of [new THREE.Euler(0, 0, 0), new THREE.Euler(Math.PI / 2, 0, 0), new THREE.Euler(0, Math.PI / 2, 0)]) {
      const l = new THREE.Line(ring, mat);
      l.rotation.copy(rot);
      this.haloGroup.add(l);
    }
    this.addLabel('__r200', 'R200c', new THREE.Vector3(radius * 0.7071, 0, radius * 0.7071));
  }

  private addLabel(id: string, text: string, pos: THREE.Vector3) {
    this.labels.get(id)?.el.remove();
    const el = document.createElement('div');
    el.className = 'scene-label';
    el.textContent = text;
    this.container.appendChild(el);
    this.labels.set(id, { el, pos });
  }

  // -------------------------------------------------------------------------
  // Picking

  private pick(e: MouseEvent): string | null {
    const rect = this.renderer.domElement.getBoundingClientRect();
    const ndc = new THREE.Vector2(((e.clientX - rect.left) / rect.width) * 2 - 1, -((e.clientY - rect.top) / rect.height) * 2 + 1);
    const ray = new THREE.Raycaster();
    ray.setFromCamera(ndc, this.camera);
    const dist = this.camera.position.distanceTo(this.controls.target);
    // ~6 px tolerance at the focus distance.
    const kpcPerPx = (2 * dist * Math.tan((this.camera.fov * Math.PI) / 360)) / rect.height;
    ray.params.Points = { threshold: 6 * kpcPerPx };
    let best: { id: string; score: number } | null = null;
    // Black hole icon: screen-space test.
    for (const l of this.layers.values()) {
      if (l.sprite && l.sprite.visible) {
        const p = new THREE.Vector3().setFromMatrixPosition(l.sprite.matrixWorld).project(this.camera);
        const dx = ((p.x - ndc.x) * rect.width) / 2;
        const dy = ((p.y - ndc.y) * rect.height) / 2;
        if (Math.hypot(dx, dy) < 14 && p.z < 1) return l.id;
      }
    }
    const targets: THREE.Object3D[] = [];
    for (const l of this.layers.values()) if (l.points && l.points.visible) targets.push(l.points);
    const hits = ray.intersectObjects(targets, false);
    for (const h of hits) {
      const id = h.object.userData.componentId as string;
      const l = this.layers.get(id);
      if (!l || h.index === undefined) continue;
      const vis = this.visible.has(id);
      if (this.filterMask !== 0 && l.tags) {
        const inTag = (l.tags[h.index] & this.filterMask) !== 0;
        if (this.filterOr ? !(vis || inTag) : !(vis && inTag)) continue;
      } else if (!vis) continue;
      if (!this.inRegion(h.point.x, h.point.y, h.point.z)) continue;
      const c = this.comps.get(id);
      // Prefer compact markers (stars, clusters) over diffuse fields at equal angular distance.
      const diffuse = c && (c.visual.style === 'gas' || c.visual.style === 'dm') ? 3 : 1;
      const score = ((h.distanceToRay ?? 0) / Math.max(h.distance, 1e-9)) * diffuse;
      if (!best || score < best.score) best = { id, score };
    }
    return best?.id ?? null;
  }

  // -------------------------------------------------------------------------
  // Camera

  view(kind: 'top' | 'edge' | 'iso' | 'axis', axis?: THREE.Vector3) {
    const t = this.controls.target.clone();
    const d = this.camera.position.distanceTo(t);
    let dir: THREE.Vector3;
    let up = new THREE.Vector3(0, 0, 1);
    switch (kind) {
      case 'top':
        dir = new THREE.Vector3(0, 0, 1);
        up = new THREE.Vector3(0, 1, 0);
        break;
      case 'edge':
        dir = new THREE.Vector3(0, -1, 0);
        break;
      case 'axis':
        dir = (axis ?? new THREE.Vector3(1, 0, 0)).clone().normalize();
        up = Math.abs(dir.z) > 0.9 ? new THREE.Vector3(0, 1, 0) : new THREE.Vector3(0, 0, 1);
        break;
      default:
        dir = new THREE.Vector3(1, -1, 0.75).normalize();
    }
    this.camera.up.copy(up);
    this.camera.position.copy(t.add(dir.multiplyScalar(d)));
    this.controls.update();
  }

  /** Frames a sphere of the given radius around a point. */
  frame(center: THREE.Vector3, radius: number) {
    const dir = this.camera.position.clone().sub(this.controls.target).normalize();
    const d = Math.max(radius, 0.0005) / Math.tan((this.camera.fov * Math.PI) / 360) * 1.1;
    this.controls.target.copy(center);
    this.camera.position.copy(center.clone().add(dir.multiplyScalar(d)));
    this.controls.update();
  }

  /** Centre of a component's markers (satellites) or the origin. */
  componentCenter(id: string): THREE.Vector3 {
    const l = this.layers.get(id);
    if (!l?.points) return new THREE.Vector3();
    const pos = l.points.geometry.getAttribute('position') as THREE.BufferAttribute;
    const c = new THREE.Vector3();
    const n = pos.count;
    for (let i = 0; i < n; i++) c.add(new THREE.Vector3(pos.getX(i), pos.getY(i), pos.getZ(i)));
    return c.multiplyScalar(1 / Math.max(n, 1));
  }

  /** Principal direction of a component's markers (e.g. the jet axis). */
  componentAxis(id: string): THREE.Vector3 | null {
    const l = this.layers.get(id);
    if (!l?.points) return null;
    const pos = l.points.geometry.getAttribute('position') as THREE.BufferAttribute;
    const a = new THREE.Vector3();
    for (let i = 0; i < pos.count; i++) a.add(new THREE.Vector3(pos.getX(i), pos.getY(i), pos.getZ(i)));
    return a.lengthSq() > 0 ? a.normalize() : null;
  }

  getCamera(): CameraState {
    const p = this.camera.position, t = this.controls.target, u = this.camera.up;
    return { position: [p.x, p.y, p.z], target: [t.x, t.y, t.z], up: [u.x, u.y, u.z] };
  }

  setCamera(s: CameraState) {
    this.camera.position.set(...s.position);
    this.controls.target.set(...s.target);
    this.camera.up.set(...s.up);
    this.controls.update();
  }

  screenshot(): string {
    this.renderer.render(this.scene, this.camera);
    return this.renderer.domElement.toDataURL('image/png');
  }

  stats(): SceneStats {
    const dist = this.camera.position.distanceTo(this.controls.target);
    const h = this.container.clientHeight || 1;
    return {
      fps: this.fps,
      visibleMarkers: this.visibleMarkerCount(),
      kpcPerPixel: (2 * dist * Math.tan((this.camera.fov * Math.PI) / 360)) / h,
      distance: dist,
    };
  }

  // -------------------------------------------------------------------------
  // Render loop

  private buildGizmo() {
    const add = (dir: THREE.Vector3, color: number) => this.gizmoScene.add(new THREE.ArrowHelper(dir, new THREE.Vector3(), 1, color, 0.25, 0.12));
    add(new THREE.Vector3(1, 0, 0), 0xff6060);
    add(new THREE.Vector3(0, 1, 0), 0x60ff80);
    add(new THREE.Vector3(0, 0, 1), 0x6090ff);
  }

  private loop = () => {
    this.raf = requestAnimationFrame(this.loop);
    this.controls.update();
    // Keep the near plane proportional to the zoom for depth precision.
    const dist = this.camera.position.distanceTo(this.controls.target);
    this.camera.near = Math.max(1e-7, dist * 1e-4);
    this.camera.far = Math.max(1e4, dist * 100);
    this.camera.updateProjectionMatrix();
    const scale = this.pixelScale();
    for (const l of this.layers.values()) if (l.material) l.material.uniforms.uScale.value = scale;

    const r = this.renderer;
    const size = r.getSize(new THREE.Vector2()); // CSS pixels
    r.setScissorTest(false);
    r.setViewport(0, 0, size.x, size.y);
    r.render(this.scene, this.camera);

    // Axes gizmo in the lower-left corner.
    const g = 90;
    this.gizmoCamera.position.copy(this.camera.position).sub(this.controls.target).setLength(3.2);
    this.gizmoCamera.up.copy(this.camera.up);
    this.gizmoCamera.lookAt(0, 0, 0);
    r.autoClear = false;
    r.setScissorTest(true);
    r.setScissor(0, 0, g, g);
    r.setViewport(0, 0, g, g);
    r.clearDepth();
    r.render(this.gizmoScene, this.gizmoCamera);
    r.setScissorTest(false);
    r.autoClear = true;

    this.updateLabels();
    this.frames++;
    const now = performance.now();
    if (now - this.lastFps > 500) {
      this.fps = (this.frames * 1000) / (now - this.lastFps);
      this.frames = 0;
      this.lastFps = now;
    }
  };

  private updateLabels() {
    const w = this.container.clientWidth, h = this.container.clientHeight;
    for (const { el, pos } of this.labels.values()) {
      const p = pos.clone().project(this.camera);
      if (p.z > 1 || p.z < -1) {
        el.style.opacity = '0';
        continue;
      }
      el.style.opacity = '';
      el.style.transform = `translate(${((p.x + 1) / 2) * w + 14}px, ${((1 - p.y) / 2) * h - 8}px)`;
    }
  }
}

let bhTex: THREE.Texture | null = null;
function blackHoleTexture(): THREE.Texture {
  if (bhTex) return bhTex;
  const c = document.createElement('canvas');
  c.width = c.height = 64;
  const g = c.getContext('2d')!;
  g.strokeStyle = 'rgba(255,255,255,0.95)';
  g.lineWidth = 3;
  g.beginPath();
  g.arc(32, 32, 22, 0, Math.PI * 2);
  g.stroke();
  g.fillStyle = '#000';
  g.beginPath();
  g.arc(32, 32, 19, 0, Math.PI * 2);
  g.fill();
  g.strokeStyle = 'rgba(255,190,120,0.9)';
  g.lineWidth = 2;
  g.beginPath();
  g.ellipse(32, 32, 30, 7, -0.35, 0, Math.PI * 2);
  g.stroke();
  bhTex = new THREE.CanvasTexture(c);
  bhTex.colorSpace = THREE.SRGBColorSpace;
  return bhTex;
}
