import { useEffect, useRef, useState } from 'react';
import uPlot from 'uplot';
import 'uplot/dist/uPlot.min.css';
import { arr, ComponentState, Dataset, PresetInfo, Profiles } from '../api';
import { sup } from '../format';

type Kind = 'enclosed' | 'density' | 'surface' | 'vc' | 'fb';

const KINDS: { id: Kind; label: string; y: string; logY: boolean; x: string; quantity?: string }[] = [
  { id: 'enclosed', label: 'M(<r)', y: 'M(<r), M☉', logY: true, x: 'r, kpc', quantity: 'enclosed_mass' },
  { id: 'density', label: 'ρ(r)', y: `ρ(r), M☉/pc${sup(3)} (spherical average)`, logY: true, x: 'r, kpc', quantity: 'volume_density' },
  { id: 'surface', label: 'Σ(R)', y: `Σ(R), M☉/pc${sup(2)} (discs)`, logY: true, x: 'R (cylindrical), kpc', quantity: 'surface_density' },
  { id: 'vc', label: 'Vc(r)', y: 'Vc, km/s', logY: false, x: 'r, kpc', quantity: 'circular_velocity' },
  { id: 'fb', label: 'f_b(<r)', y: 'baryon fraction f_b(<r)', logY: false, x: 'r, kpc' },
];

const DATA_COLORS = ['#ff5ea8', '#5effc1', '#ffd45e', '#5ec8ff', '#c45eff'];

interface Props {
  profiles: Profiles | null;
  compare: Profiles | null;
  compareName: string;
  components: ComponentState[];
  show: string[]; // component ids to plot alongside the total; empty: per matter
  matterLabels: Record<string, string>;
  datasets: Dataset[];
  presets: PresetInfo[];
  currentPreset: string;
  compareId: string;
  setCompareId: (id: string) => void;
  onImport: () => void;
  onRemoveDataset: (id: string) => void;
}

type S = { id: string; label: string; y: (number | null)[]; color: string; dash?: number[]; points?: boolean; err?: (number | null)[] };

export function Charts(p: Props) {
  const [kind, setKind] = useState<Kind>('enclosed');
  const host = useRef<HTMLDivElement>(null);
  const plot = useRef<uPlot | null>(null);
  const [legend, setLegend] = useState<{ label: string; color: string; dash?: boolean; points?: boolean }[]>([]);
  const k = KINDS.find((x) => x.id === kind)!;
  const relevant = p.datasets.filter((d) => d.quantity === k.quantity);

  useEffect(() => {
    const el = host.current;
    if (!el || !p.profiles) return;
    const colors = new Map(p.components.map((c) => [c.id, c.visual.color]));
    const byId = new Map(p.components.map((c) => [c.id, c]));
    const listOf = (prof: Profiles) =>
      arr(kind === 'enclosed' ? prof.enclosed : kind === 'density' ? prof.density : kind === 'surface' ? prof.surface : prof.vc);

    // Without a selection, curves are summed per matter category (Vc in quadrature).
    const pick = (prof: Profiles, quadrature: boolean): S[] => {
      const all = listOf(prof);
      const total: S[] = all.filter((s) => s.id === 'total').map((s) => ({ ...s, color: '#ffffff' }));
      if (p.show.length > 0) return [...total, ...all.filter((s) => p.show.includes(s.id)).map((s) => ({ ...s, color: colors.get(s.id) ?? '#888' }))];
      const groups = new Map<string, { id: string; label: string; y: number[]; color: string }>();
      for (const s of all) {
        const c = byId.get(s.id);
        if (!c) continue;
        const key = `matter:${c.matter}`;
        const g = groups.get(key) ?? { id: key, label: p.matterLabels[c.matter] ?? c.matter, y: new Array(s.y.length).fill(0), color: c.visual.color };
        s.y.forEach((v, i) => (g.y[i] += quadrature ? v * v : v));
        groups.set(key, g);
      }
      const out: S[] = [...groups.values()].map((g) => ({ ...g, y: quadrature ? g.y.map(Math.sqrt) : g.y }));
      return [...total, ...out];
    };

    const model: S[] = kind === 'fb'
      ? [{ id: 'fb', label: 'f_b(<r)', y: arr(p.profiles.baryonFraction), color: '#ffffff' }]
      : pick(p.profiles, kind === 'vc');
    if (p.compare && kind !== 'surface') {
      const y = kind === 'fb' ? arr(p.compare.baryonFraction) : listOf(p.compare).find((s) => s.id === 'total')?.y;
      if (y) model.push({ id: 'compare', label: `${p.compareName} (total)`, y, color: '#9aa6c0', dash: [6, 4] });
    }

    // Merge the model grid with the radii of imported points.
    const grid = arr(p.profiles.r);
    const xs = [...new Set([...grid, ...relevant.flatMap((d) => arr(d.points).map((pt) => pt.r))])].sort((a, b) => a - b);
    const gridIndex = new Map(grid.map((r, i) => [r, i]));
    const series: S[] = model.map((s) => ({ ...s, y: xs.map((x) => (gridIndex.has(x) ? s.y[gridIndex.get(x)!] : null)) }));
    relevant.forEach((d, i) => {
      const at = new Map(arr(d.points).map((pt) => [pt.r, pt]));
      series.push({
        id: d.id, label: `${d.name} [${d.evidence.toLowerCase()}]`, color: DATA_COLORS[i % DATA_COLORS.length], points: true,
        y: xs.map((x) => at.get(x)?.v ?? null), err: xs.map((x) => at.get(x)?.err ?? null),
      });
    });

    // Log axes: drop values far below the largest one so the range stays ≤ 12 decades.
    const ymax = Math.max(1e-300, ...series.flatMap((s) => s.y.map((v) => v ?? 0)));
    const floor = ymax * 1e-12;
    const shown = series.filter((s) => !k.logY || s.y.some((v) => v != null && v > floor));
    const clean = (y: (number | null)[]) => y.map((v) => (v == null || (k.logY && !(v > floor)) ? null : v));
    const data = [xs, ...shown.map((s) => clean(s.y))] as uPlot.AlignedData;

    const errorBars = (u: uPlot) => {
      const ctx = u.ctx;
      const dpr = devicePixelRatio;
      for (const s of shown) {
        if (!s.err) continue;
        ctx.save();
        ctx.strokeStyle = s.color;
        ctx.lineWidth = 1.2 * dpr;
        xs.forEach((x, i) => {
          const v = s.y[i], e = s.err![i];
          if (v == null || !e || !(e > 0)) return;
          const lo = k.logY ? Math.max(v - e, floor) : v - e;
          const cx = u.valToPos(x, 'x', true);
          const y0 = u.valToPos(lo, 'y', true), y1 = u.valToPos(v + e, 'y', true);
          ctx.beginPath();
          ctx.moveTo(cx, y0);
          ctx.lineTo(cx, y1);
          ctx.moveTo(cx - 3 * dpr, y0);
          ctx.lineTo(cx + 3 * dpr, y0);
          ctx.moveTo(cx - 3 * dpr, y1);
          ctx.lineTo(cx + 3 * dpr, y1);
          ctx.stroke();
        });
        ctx.restore();
      }
    };

    const opts: uPlot.Options = {
      width: Math.max(200, el.clientWidth - 16),
      height: Math.max(100, el.clientHeight),
      scales: { x: { time: false, distr: 3 }, y: k.logY ? { distr: 3 } : { auto: true } },
      axes: [
        { label: k.x, stroke: '#9aa6c0', grid: { stroke: '#1d2436' }, ticks: { stroke: '#2a3350' }, values: (_u, v) => v.map(decadeLabel(v)) },
        {
          label: k.y, stroke: '#9aa6c0', grid: { stroke: '#1d2436' }, ticks: { stroke: '#2a3350' }, size: 64, space: 22,
          values: (u, v) => (k.logY ? v.map(decadeLabel(v, Math.max(3, Math.floor(u.bbox.height / devicePixelRatio / 22)))) : v.map((x) => String(+x.toPrecision(3)))),
        },
      ],
      legend: { show: false },
      cursor: { drag: { x: true, y: false } },
      hooks: { draw: [errorBars] },
      series: [
        { label: 'r' },
        ...shown.map((s) => ({
          label: s.label,
          stroke: s.color,
          width: s.id === 'total' ? 2 : 1.4,
          dash: s.dash,
          spanGaps: !s.points,
          paths: s.points ? () => null : undefined,
          points: s.points ? { show: true, size: 6, fill: s.color, stroke: s.color } : { show: false },
        })),
      ],
    };
    plot.current?.destroy();
    try {
      plot.current = new uPlot(opts, data, el);
    } catch (e) {
      el.textContent = `Plot error: ${String(e)}`;
      return;
    }
    setLegend(shown.map((s) => ({ label: s.label, color: s.color, dash: !!s.dash, points: s.points })));
    const ro = new ResizeObserver(() => plot.current?.setSize({ width: Math.max(200, el.clientWidth - 16), height: Math.max(100, el.clientHeight) }));
    ro.observe(el);
    return () => {
      ro.disconnect();
      plot.current?.destroy();
      plot.current = null;
    };
  }, [p.profiles, p.compare, kind, p.show.join(','), p.components, p.datasets]);

  return (
    <div className="charts">
      <div className="tabs">
        {KINDS.map((x) => (
          <button key={x.id} className={kind === x.id ? 'on' : ''} onClick={() => setKind(x.id)}>
            {x.label}
          </button>
        ))}
        <span className="sep" />
        <label className="small">
          Compare{' '}
          <select value={p.compareId} onChange={(e) => p.setCompareId(e.target.value)}>
            <option value="">—</option>
            {p.presets.filter((x) => x.id !== p.currentPreset).map((x) => (
              <option key={x.id} value={x.id}>{x.name}</option>
            ))}
          </select>
        </label>
        <button onClick={p.onImport} title="Import an observational radial profile (CSV with units, errors and source)">Import profile…</button>
        {p.datasets.map((d) => (
          <span key={d.id} className="chip" title={`${d.source} · ${d.quantity} [${d.unit}] · ${arr(d.points).length} points`}>
            {d.name}
            <button className="icon-btn" onClick={() => p.onRemoveDataset(d.id)} title="Remove dataset">×</button>
          </span>
        ))}
      </div>
      <div className="chart-legend">
        {legend.map((l) => (
          <span key={l.label}>
            <i className={l.points ? 'dot' : l.dash ? 'dash' : ''} style={{ background: l.dash ? undefined : l.color, borderColor: l.color }} />
            {l.label}
          </span>
        ))}
        <span className="muted">
          {kind === 'surface' && p.compare ? 'Comparison is not drawn for Σ (disc-only quantity). ' : ''}
          {relevant.length === 0 && p.datasets.length > 0 ? 'No imported data for this quantity. ' : ''}
          Model curves: synthetic reconstruction. {kind === 'vc' ? p.profiles?.note : ''}
        </span>
      </div>
      <div className="chart-host" ref={host} />
    </div>
  );
}

/** Labels only whole decades on log axes, thinning them to fit `maxLabels`. */
function decadeLabel(splits: number[], maxLabels = 12) {
  const decades = splits.filter((x) => x > 0 && Math.abs(Math.log10(x) - Math.round(Math.log10(x))) < 1e-9);
  const step = Math.max(1, Math.ceil(decades.length / maxLabels));
  return (x: number) => {
    if (!(x > 0)) return '';
    const e = Math.log10(x);
    if (Math.abs(e - Math.round(e)) > 1e-9) return '';
    if (Math.round(e) % step !== 0) return '';
    const n = Math.round(e);
    return n >= -2 && n <= 3 ? String(+x.toPrecision(3)) : `10${sup(n)}`;
  };
}
