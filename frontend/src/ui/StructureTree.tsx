import { useMemo, useState } from 'react';
import { arr, ComponentState, Selection, State } from '../api';
import { EVIDENCE_LABEL, pct } from '../format';

const GROUP_ORDER = ['Stars', 'Gas', 'Dust', 'Dark matter', 'Black hole'];

interface Props {
  state: State;
  visible: Set<string>;
  hiddenByMode: Set<string>;
  setVisible: (v: Set<string>) => void;
  selection: Selection;
  onSelectComponent: (id: string, additive: boolean) => void;
  onSelectTag: (id: string, additive: boolean) => void;
  filterTags: string[];
  setFilterTags: (t: string[]) => void;
  filterOr: boolean;
  setFilterOr: (v: boolean) => void;
  onIsolate: () => void;
}

export function StructureTree(p: Props) {
  const [query, setQuery] = useState('');
  const [collapsed, setCollapsed] = useState<Record<string, boolean>>({});
  const groups = useMemo(() => {
    const byMatter = new Map(arr(p.state.categories).map((c) => [c.id, c]));
    const g = new Map<string, ComponentState[]>();
    for (const c of arr(p.state.components)) {
      const group = byMatter.get(c.matter)?.group ?? 'Other';
      if (!g.has(group)) g.set(group, []);
      g.get(group)!.push(c);
    }
    return [...g.entries()].sort((a, b) => GROUP_ORDER.indexOf(a[0]) - GROUP_ORDER.indexOf(b[0]));
  }, [p.state]);

  const q = query.trim().toLowerCase();
  const matches = (c: ComponentState) => !q || c.name.toLowerCase().includes(q) || c.structure.includes(q) || c.matter.includes(q);

  const toggle = (ids: string[], on: boolean) => {
    const v = new Set(p.visible);
    for (const id of ids) on ? v.add(id) : v.delete(id);
    p.setVisible(v);
  };

  const families = useMemo(() => {
    const f = new Map<string, typeof p.state.tags>();
    for (const t of arr(p.state.tags)) {
      if (!f.has(t.family)) f.set(t.family, []);
      f.get(t.family)!.push(t);
    }
    return [...f.entries()];
  }, [p.state]);

  return (
    <div className="panel tree">
      <div className="panel-title">Structures</div>
      <input className="search" placeholder="Search components…" value={query} onChange={(e) => setQuery(e.target.value)} />
      <div className="row gap">
        <button onClick={() => p.setVisible(new Set(arr(p.state.components).map((c) => c.id)))}>Show all</button>
        <button onClick={() => p.setVisible(new Set())}>Hide all</button>
        <button onClick={p.onIsolate} disabled={p.selection.components.length === 0} title="Show only the selected components">
          Isolate
        </button>
      </div>

      <div className="section-label">Matter</div>
      {groups.map(([group, comps]) => {
        const shown = comps.filter(matches);
        if (shown.length === 0) return null;
        const ids = comps.map((c) => c.id);
        const nOn = ids.filter((id) => p.visible.has(id)).length;
        const state = nOn === 0 ? 'off' : nOn === ids.length ? 'on' : 'partial';
        return (
          <div key={group} className="group">
            <div className="tree-row group-row">
              <span className="caret" onClick={() => setCollapsed({ ...collapsed, [group]: !collapsed[group] })}>
                {collapsed[group] ? '▸' : '▾'}
              </span>
              <TriCheck state={state} onChange={(on) => toggle(ids, on)} />
              <span className="tree-name">{group}</span>
            </div>
            {!collapsed[group] &&
              shown.map((c) => {
                const sel = p.selection.components.includes(c.id);
                const blocked = p.hiddenByMode.has(c.id);
                return (
                  <div key={c.id} className={`tree-row child ${sel ? 'selected' : ''} ${blocked ? 'blocked' : ''}`}>
                    <input type="checkbox" checked={p.visible.has(c.id)} onChange={(e) => toggle([c.id], e.target.checked)} />
                    <span className="swatch" style={{ background: c.visual.color }} />
                    <span className="tree-name" onClick={(e) => p.onSelectComponent(c.id, e.ctrlKey || e.shiftKey || e.metaKey)} title={c.notes}>
                      {c.name}
                    </span>
                    <span className={`ev ev-${c.evidence}`} title={`${EVIDENCE_LABEL[c.evidence]}${blocked ? ' — hidden by data mode' : ''}`} />
                    <span className="tree-pct">{pct(c.fractionOfTotal)}</span>
                  </div>
                );
              })}
          </div>
        );
      })}

      <div className="section-label">
        Structural labels
        <label className="small push-right" title="AND: show markers that are both visible matter and carry the label. OR: also show labelled markers of hidden matter.">
          <input type="checkbox" checked={p.filterOr} onChange={(e) => p.setFilterOr(e.target.checked)} /> OR
        </label>
      </div>
      {families.map(([fam, tags]) => (
        <div key={fam} className="group">
          <div className="tree-row group-row muted">{fam}</div>
          {tags.map((t) => {
            const sel = p.selection.tags.includes(t.id);
            const filt = p.filterTags.includes(t.id);
            return (
              <div key={t.id} className={`tree-row child ${sel ? 'selected' : ''}`}>
                <button
                  className={`icon-btn ${filt ? 'on' : ''}`}
                  title="Filter the scene to this label"
                  onClick={() => p.setFilterTags(filt ? p.filterTags.filter((x) => x !== t.id) : [...p.filterTags, t.id])}
                >
                  ⧩
                </button>
                <span className="tree-name" onClick={(e) => p.onSelectTag(t.id, e.ctrlKey || e.shiftKey || e.metaKey)} title={t.note}>
                  {t.label}
                </span>
                <span className="tree-pct">{pct(t.mass / p.state.totalMass)}</span>
              </div>
            );
          })}
        </div>
      ))}
      <div className="hint">Click to select · Ctrl/Shift to add · Alt+drag in the scene for box selection · right-click for actions. Labels never add mass: their mass is the sum of the owners' matter.</div>
    </div>
  );
}

function TriCheck({ state, onChange }: { state: 'on' | 'off' | 'partial'; onChange: (on: boolean) => void }) {
  return (
    <input
      type="checkbox"
      checked={state === 'on'}
      ref={(el) => {
        if (el) el.indeterminate = state === 'partial';
      }}
      onChange={() => onChange(state !== 'on')}
    />
  );
}
