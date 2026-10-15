import { useEffect, useState } from 'react';
import { arr, ComponentState, openURL, Selection, SelectionReport, State } from '../api';
import { EVIDENCE_LABEL, length, mass, pct, sci } from '../format';

interface Props {
  state: State;
  selection: Selection;
  report: SelectionReport | null;
  onShare: (cat: string, comp: string, share: number) => void;
  onShareLock: (cat: string, comp: string, locked: boolean) => void;
}

// Present-day mean stellar mass for a Kroupa-like IMF, used only for the
// "≈ number of stars" estimate.
const MEAN_STELLAR_MASS = 0.5;

export function Inspector({ state, selection, report, onShare, onShareLock }: Props) {
  const comps = new Map(arr(state.components).map((c) => [c.id, c]));
  const cats = new Map(arr(state.categories).map((c) => [c.id, c]));
  const tags = new Map(arr(state.tags).map((t) => [t.id, t]));
  const empty = selection.components.length === 0 && selection.tags.length === 0;

  if (empty) return <GalaxySummary state={state} />;

  const single = selection.components.length === 1 && selection.tags.length === 0 ? comps.get(selection.components[0]) : undefined;
  const singleTag = selection.tags.length === 1 && selection.components.length === 0 ? tags.get(selection.tags[0]) : undefined;

  return (
    <div className="inspector">
      <div className="panel-title">
        Selection
        {single && <span className={`badge ev-${single.evidence}`}>{EVIDENCE_LABEL[single.evidence]}</span>}
      </div>
      <div className="big-number">{report ? mass(report.mass) : '…'}</div>
      <div className="muted">
        {report ? `${pct(report.fractionOfTotal)} of M200c` : ''}
        {report && !report.exact && ' · tag fraction estimated from the reference sample'}
      </div>

      {single && <ComponentDetails c={single} state={state} onShare={onShare} onShareLock={onShareLock} categoryLabel={cats.get(single.matter)?.label ?? single.matter} />}

      {singleTag && (
        <div className="block">
          <div className="kv"><span>Label</span><b>{singleTag.label}</b></div>
          {singleTag.note && <p className="note">{singleTag.note}</p>}
          <p className="note">A structural label is not a kind of matter: its mass is the single-counted sum of the matter it labels.</p>
        </div>
      )}

      {report && arr(report.parts).length > 0 && !single && (
        <div className="block">
          <div className="section-label">By mass owner</div>
          {arr(report.parts).map((p) => (
            <div className="kv" key={p.component}>
              <span>
                {comps.get(p.component)?.name ?? p.component}
                {p.fraction < 1 && <span className="muted"> ({pct(p.fraction)} of it)</span>}
              </span>
              <b>{mass(p.mass)}</b>
            </div>
          ))}
          <div className="section-label">By matter</div>
          {Object.entries(report.byMatter ?? {}).map(([k, v]) => (
            <div className="kv" key={k}>
              <span>{cats.get(k)?.label ?? k}</span>
              <b>{mass(v)}</b>
            </div>
          ))}
        </div>
      )}
      {report && report.insideR200c < report.mass * (1 - 1e-9) && (
        <div className="warn">Part of this selection lies outside R200c: {mass(report.mass - report.insideR200c)}.</div>
      )}
    </div>
  );
}

function ComponentDetails({ c, state, onShare, onShareLock, categoryLabel }: {
  c: ComponentState; state: State; categoryLabel: string;
  onShare: Props['onShare']; onShareLock: Props['onShareLock'];
}) {
  const [share, setShare] = useState((c.share * 100).toFixed(3));
  useEffect(() => setShare((c.share * 100).toFixed(3)), [c.share]);
  const markers = c.fixedCount > 0 ? c.fixedCount : c.markers;
  const params = Object.entries(c.params ?? {}).sort();
  const sources = new Map(arr(state.sources).map((s) => [s.id, s]));
  const commit = () => {
    const v = parseFloat(share) / 100;
    if (isFinite(v) && Math.abs(v - c.share) > 1e-12) onShare(c.matter, c.id, v);
  };
  return (
    <div className="block">
      <div className="comp-name">{c.name}</div>
      <div className="kv"><span>Matter</span><b>{categoryLabel}</b></div>
      <div className="kv"><span>Structure</span><b>{c.structure}</b></div>
      {arr(c.tags).length + arr(c.localTags).length > 0 && (
        <div className="kv"><span>Labels</span><b>{[...arr(c.tags), ...arr(c.localTags)].join(', ')}</b></div>
      )}
      <div className="kv"><span>Share of total</span><b>{pct(c.fractionOfTotal)}</b></div>
      <div className="kv">
        <span>Share of {categoryLabel.toLowerCase()}</span>
        <span className="row gap">
          <input className="num" value={share} disabled={c.shareLocked} onChange={(e) => setShare(e.target.value)} onBlur={commit} onKeyDown={(e) => e.key === 'Enter' && commit()} />%
          <label className="small" title="Lock this share while others are edited">
            <input type="checkbox" checked={c.shareLocked} onChange={(e) => onShareLock(c.matter, c.id, e.target.checked)} /> lock
          </label>
        </span>
      </div>
      <div className="kv"><span>Inside R200c</span><b>{pct(c.insideR200c)}</b></div>
      <div className="kv"><span>Extent</span><b>{c.maxRadius > 0 ? `r ≤ ${length(c.maxRadius)}` : 'point'}</b></div>
      {c.params?.centre_r_kpc !== undefined && <div className="kv"><span>Distance from centre</span><b>{length(c.params.centre_r_kpc)}</b></div>}
      <div className="kv">
        <span>Markers</span>
        <b>{markers.toLocaleString()} {c.fixedCount > 0 ? '(one per object)' : `· ≈ ${sci(c.mass / Math.max(markers, 1))} M☉ each`}</b>
      </div>
      {c.matter === 'stars' && c.fixedCount === 0 && c.mass > 0 && (
        <div className="kv"><span>≈ physical stars</span><b>{sci(c.mass / MEAN_STELLAR_MASS)}</b></div>
      )}
      <div className="section-label">Model · {c.kind}</div>
      {params.map(([k, v]) => (
        <div className="kv small" key={k}><span>{k}</span><b>{sci(v, 4)}</b></div>
      ))}
      {c.notes && <p className="note">{c.notes}</p>}
      {arr(c.sources).length > 0 && (
        <>
          <div className="section-label">Sources</div>
          {arr(c.sources).map((id) => {
            const s = sources.get(id);
            return (
              <div key={id} className="source">
                <a onClick={() => s?.url && openURL(s.url)}>{s?.citation ?? id}</a>
                {s?.usedFor && <div className="muted small">{s.usedFor}</div>}
              </div>
            );
          })}
        </>
      )}
    </div>
  );
}

function GalaxySummary({ state }: { state: State }) {
  return (
    <div className="inspector">
      <div className="panel-title">Galaxy</div>
      <div className="comp-name">{state.preset.name}</div>
      <p className="note">{state.preset.description}</p>
      <div className="warn subtle">{state.preset.disclaimer}</div>
      <div className="kv"><span>M200c</span><b>{mass(state.totalMass)}</b></div>
      <div className="kv"><span>R200c</span><b>{length(state.r200c)}</b></div>
      <div className="kv"><span>V200c</span><b>{state.v200c.toFixed(0)} km/s</b></div>
      <div className="kv"><span>NFW c200c</span><b>{state.concentration.toFixed(2)}</b></div>
      <div className="kv"><span>Baryon fraction</span><b>{pct(state.baryonFraction)} (cosmic {pct(state.cosmicBaryonFraction)})</b></div>
      <div className="kv"><span>Cosmology</span><b>{state.cosmology.name}, H0 = {state.cosmology.h0}</b></div>
      <div className="kv"><span>Seed</span><b>{state.seed}</b></div>
      <p className="note">{state.haloDefinition}</p>
      <p className="note muted">Select something in the tree or click a marker in the scene. Visibility never changes physical masses.</p>
    </div>
  );
}
