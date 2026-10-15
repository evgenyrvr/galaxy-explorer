import { useEffect, useState } from 'react';
import { Region, regionActive } from '../api';

interface Props {
  region: Region;
  setRegion: (r: Region) => void;
  hideOutside: boolean;
  setHideOutside: (v: boolean) => void;
}

/** Spatial selection: sphere in r, or cylinder in R with an optional |z| limit (kpc). */
export function RegionPanel({ region, setRegion, hideOutside, setHideOutside }: Props) {
  const [draft, setDraft] = useState({ rMin: String(region.rMin), rMax: String(region.rMax), zMax: region.zMax > 0 ? String(region.zMax) : '' });
  useEffect(() => setDraft({ rMin: String(region.rMin), rMax: String(region.rMax), zMax: region.zMax > 0 ? String(region.zMax) : '' }), [region]);
  const commit = (kind = region.kind) => {
    const rMin = Math.max(0, parseFloat(draft.rMin) || 0);
    const rMax = parseFloat(draft.rMax);
    const zMax = parseFloat(draft.zMax);
    setRegion({ kind, rMin, rMax: isFinite(rMax) ? rMax : region.rMax, zMax: isFinite(zMax) && zMax > 0 ? zMax : 0 });
  };
  const invalid = region.kind !== '' && !regionActive(region);
  return (
    <div className="region">
      <div className="section-label">Region (kpc)</div>
      <div className="row gap wrap">
        <select value={region.kind} onChange={(e) => commit(e.target.value as Region['kind'])}>
          <option value="">none</option>
          <option value="sphere">sphere r</option>
          <option value="cylinder">cylinder R, |z|</option>
        </select>
        {region.kind !== '' && (
          <label className="small" title="Hide markers outside the region (masses are always computed exactly in Go)">
            <input type="checkbox" checked={hideOutside} onChange={(e) => setHideOutside(e.target.checked)} /> hide outside
          </label>
        )}
      </div>
      {region.kind !== '' && (
        <div className="row gap wrap region-inputs">
          <label className="small">{region.kind === 'sphere' ? 'r' : 'R'} from <input className="num short" value={draft.rMin} onChange={(e) => setDraft({ ...draft, rMin: e.target.value })} onBlur={() => commit()} onKeyDown={(e) => e.key === 'Enter' && commit()} /></label>
          <label className="small">to <input className="num short" value={draft.rMax} onChange={(e) => setDraft({ ...draft, rMax: e.target.value })} onBlur={() => commit()} onKeyDown={(e) => e.key === 'Enter' && commit()} /></label>
          {region.kind === 'cylinder' && (
            <label className="small">|z| &lt; <input className="num short" placeholder="∞" value={draft.zMax} onChange={(e) => setDraft({ ...draft, zMax: e.target.value })} onBlur={() => commit()} onKeyDown={(e) => e.key === 'Enter' && commit()} /></label>
          )}
        </div>
      )}
      {invalid && <div className="error">The outer radius must exceed the inner one.</div>}
      {regionActive(region) && <div className="hint">The inspector shows the selection inside the region; with nothing selected, all matter in the region.</div>}
    </div>
  );
}
