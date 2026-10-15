import { useEffect, useRef, useState } from 'react';
import uPlot from 'uplot';
import 'uplot/dist/uPlot.min.css';
import { arr, ComponentState, Profiles } from '../api';
import { sci, sup } from '../format';

type Kind = 'enclosed' | 'density' | 'surface' | 'vc' | 'fb';

const KINDS: { id: Kind; label: string; y: string; logY: boolean; x: string }[] = [
  { id: 'enclosed', label: 'M(<r)', y: 'M(<r), M☉', logY: true, x: 'r, kpc' },
  { id: 'density', label: 'ρ(r)', y: `ρ(r), M☉/pc${sup(3)} (spherical average)`, logY: true, x: 'r, kpc' },
  { id: 'surface', label: 'Σ(R)', y: `Σ(R), M☉/pc${sup(2)} (discs)`, logY: true, x: 'R (cylindrical), kpc' },
  { id: 'vc', label: 'Vc(r)', y: 'Vc, km/s', logY: false, x: 'r, kpc' },
  { id: 'fb', label: 'f_b(<r)', y: 'baryon fraction f_b(<r)', logY: false, x: 'r, kpc' },
];

interface Props {
  profiles: Profiles | null;
  components: ComponentState[];
  show: string[]; // component ids to plot alongside the total; empty: per matter
  matterLabels: Record<string, string>;
}

export function Charts({ profiles, components, show, matterLabels }: Props) {
  const [kind, setKind] = useState<Kind>('enclosed');
  const host = useRef<HTMLDivElement>(null);
  const plot = useRef<uPlot | null>(null);
  const [legend, setLegend] = useState<{ label: string; color: string }[]>([]);

  useEffect(() => {
    const el = host.current;
    if (!el || !profiles) return;
    const k = KINDS.find((x) => x.id === kind)!;
    const colors = new Map(components.map((c) => [c.id, c.visual.color]));
    let series: { id: string; label: string; y: number[] }[] = [];
    type S = { id: string; label: string; y: number[] };
    const byId = new Map(components.map((c) => [c.id, c]));
    // Without a selection, curves are summed per matter category (Vc in quadrature).
    const pick = (list: S[] | null | undefined, quadrature = false): S[] => {
      const all = arr(list);
      const total = all.filter((s) => s.id === 'total');
      if (show.length > 0) return [...total, ...all.filter((s) => show.includes(s.id))];
      const groups = new Map<string, S>();
      for (const s of all) {
        const c = byId.get(s.id);
        if (!c) continue;
        const key = `matter:${c.matter}`;
        const g = groups.get(key) ?? { id: key, label: matterLabels[c.matter] ?? c.matter, y: new Array(s.y.length).fill(0) };
        s.y.forEach((v, i) => (g.y[i] += quadrature ? v * v : v));
        groups.set(key, g);
        if (!colors.has(key)) colors.set(key, c.visual.color);
      }
      const out = [...groups.values()];
      if (quadrature) out.forEach((g) => (g.y = g.y.map(Math.sqrt)));
      return [...total, ...out];
    };
    switch (kind) {
      case 'enclosed': series = pick(profiles.enclosed); break;
      case 'density': series = pick(profiles.density); break;
      case 'surface': series = pick(profiles.surface); break;
      case 'vc': series = pick(profiles.vc, true); break;
      case 'fb': series = [{ id: 'fb', label: 'f_b(<r)', y: arr(profiles.baryonFraction) }]; break;
    }
    // Log axes: drop values far below the largest one so the range stays ≤ 12 decades.
    const ymax = Math.max(1e-300, ...series.map((s) => Math.max(0, ...arr(s.y))));
    const floor = ymax * 1e-12;
    const clean = (y: number[]) => arr(y).map((v) => (k.logY && !(v > floor) ? null : v));
    series = series.filter((s) => !k.logY || arr(s.y).some((v) => v > floor));
    const data: uPlot.AlignedData = [arr(profiles.r), ...series.map((s) => clean(s.y))] as uPlot.AlignedData;
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
      series: [
        { label: 'r', value: (_u, v) => (v == null ? '—' : `${v.toPrecision(3)} kpc`) },
        ...series.map((s) => ({
          label: s.label,
          stroke: s.id === 'total' || s.id === 'fb' ? '#ffffff' : colors.get(s.id) ?? '#888',
          width: s.id === 'total' ? 2 : 1.4,
          value: (_u: uPlot, v: number | null) => (v == null ? '—' : sci(v, 3)),
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
    setLegend(series.map((s) => ({ label: s.label, color: s.id === 'total' || s.id === 'fb' ? '#ffffff' : colors.get(s.id) ?? '#888' })));
    const ro = new ResizeObserver(() => plot.current?.setSize({ width: Math.max(200, el.clientWidth - 16), height: Math.max(100, el.clientHeight) }));
    ro.observe(el);
    return () => {
      ro.disconnect();
      plot.current?.destroy();
      plot.current = null;
    };
  }, [profiles, kind, show.join(','), components]);

  return (
    <div className="charts">
      <div className="tabs">
        {KINDS.map((k) => (
          <button key={k.id} className={kind === k.id ? 'on' : ''} onClick={() => setKind(k.id)}>
            {k.label}
          </button>
        ))}
        <span className="muted small chart-note">
          {profiles?.note} Model curves of a synthetic reconstruction (not observations). R200c = {profiles ? profiles.r200c.toFixed(0) : '—'} kpc.
        </span>
      </div>
      <div className="chart-legend">
        {legend.map((l) => (
          <span key={l.label}><i style={{ background: l.color }} />{l.label}</span>
        ))}
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
