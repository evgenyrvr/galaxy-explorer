// Thin client over the Go bindings. All mass logic lives in Go.
import * as Go from '../wailsjs/go/main/App';
import { galaxy } from '../wailsjs/go/models';
import { BrowserOpenURL } from '../wailsjs/runtime/runtime';

export type State = galaxy.State;
export type ComponentState = galaxy.ComponentState;
export type TagState = galaxy.TagState;
export type SelectionReport = galaxy.SelectionReport;
export type Profiles = galaxy.Profiles;
export type PresetInfo = galaxy.PresetInfo;

export interface Selection {
  components: string[];
  tags: string[];
}

export const api = Go;

export function openURL(url: string) {
  BrowserOpenURL(url);
}

export function selection(sel: Selection): galaxy.Selection {
  return galaxy.Selection.createFrom(sel);
}

export function profileRequest(components: string[], rMax = 0): galaxy.ProfileRequest {
  return galaxy.ProfileRequest.createFrom({ components, rMin: 0.01, rMax, n: 160 });
}

/** Marker buffer for one component: positions (kpc) and tag bitmasks. */
export interface Markers {
  count: number;
  positions: Float32Array;
  tags: Float32Array; // bitmask stored as float (exact below 2^24)
}

export async function fetchMarkers(id: string, n: number): Promise<Markers> {
  const res = await fetch(`/api/particles?id=${encodeURIComponent(id)}&n=${n}`);
  if (!res.ok) throw new Error(`markers for ${id}: ${res.status} ${await res.text()}`);
  const buf = await res.arrayBuffer();
  const view = new DataView(buf);
  const count = view.getUint32(0, true);
  const positions = new Float32Array(buf, 4, count * 3);
  const tagsU = new Uint32Array(buf, 4 + count * 12, count);
  const tags = new Float32Array(count);
  for (let i = 0; i < count; i++) tags[i] = tagsU[i];
  return { count, positions: new Float32Array(positions), tags };
}

/** Normalises Go nil slices (null) to empty arrays. */
export function arr<T>(x: T[] | null | undefined): T[] {
  return x ?? [];
}

export function errorText(e: unknown): string {
  if (typeof e === 'string') return e;
  if (e instanceof Error) return e.message;
  return JSON.stringify(e);
}
