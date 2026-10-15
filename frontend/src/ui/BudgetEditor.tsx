import { useEffect, useState } from 'react';
import { arr, State } from '../api';
import { mass, parseMass, pct, sci } from '../format';

export interface BudgetActions {
  setTotal: (m: number) => void;
  setFraction: (cat: string, f: number, rule: string) => void;
  setCategoryMass: (cat: string, m: number) => void;
  setLocked: (cat: string, locked: boolean) => void;
  setMode: (mode: string) => void;
  normalize: () => void;
  assignRemainder: (cat: string) => void;
  undo: () => void;
  redo: () => void;
  reset: () => void;
}

export function BudgetEditor({ state, a }: { state: State; a: BudgetActions }) {
  const [rule, setRule] = useState('leave_unallocated');
  const [target, setTarget] = useState('');
  const fixComp = state.mode === 'fix_components';
  const cats = arr(state.categories);
  const unalloc = state.unallocated;
  const complete = Math.abs(unalloc) <= 1e-9;

  return (
    <div className="budget">
      <div className="panel-title">Mass budget</div>
      <div className="kv">
        <span>M_total (M200c)</span>
        <MassInput value={state.totalMass} onCommit={a.setTotal} />
      </div>
      <div className="row gap wrap">
        <label className="small">
          <input type="radio" checked={!fixComp} onChange={() => a.setMode('fix_total')} /> Fix total mass
        </label>
        <label className="small" title="Requires a complete (100%) budget">
          <input type="radio" checked={fixComp} onChange={() => a.setMode('fix_components')} /> Fix component masses
        </label>
      </div>
      {!fixComp && (
        <div className="kv small">
          <span>When a share changes</span>
          <select value={rule} onChange={(e) => setRule(e.target.value)}>
            <option value="leave_unallocated">leave the rest unallocated</option>
            <option value="proportional">rebalance unlocked shares proportionally</option>
          </select>
        </div>
      )}
      <table className="budget-table">
        <thead>
          <tr>
            <th>Category</th>
            <th>% of total</th>
            <th>Mass, M☉</th>
            <th title="Locked categories are not changed by rebalancing">🔒</th>
          </tr>
        </thead>
        <tbody>
          {cats.map((c) => (
            <tr key={c.id} title={c.note}>
              <td>{c.label}</td>
              <td>
                <PercentInput value={c.fraction} disabled={c.locked} onCommit={(f) => a.setFraction(c.id, f, rule)} />
              </td>
              <td>{fixComp ? <MassInput value={c.mass} disabled={c.locked} onCommit={(m) => a.setCategoryMass(c.id, m)} /> : <span className="mono">{sci(c.mass)}</span>}</td>
              <td>
                <input type="checkbox" checked={c.locked} onChange={(e) => a.setLocked(c.id, e.target.checked)} />
              </td>
            </tr>
          ))}
          <tr className={complete ? 'total ok' : 'total bad'}>
            <td>{unalloc >= 0 ? 'Unallocated' : 'Over-allocated'}</td>
            <td className="mono">{pct(Math.abs(unalloc))}</td>
            <td className="mono">{sci(Math.abs(state.unallocatedMass))}</td>
            <td />
          </tr>
        </tbody>
      </table>
      {!complete && (
        <div className="warn">
          {pct(unalloc)} of M_total ({mass(state.unallocatedMass)}) is not assigned to any matter. Assign it or normalise:
          <div className="row gap wrap">
            <button onClick={a.normalize}>Normalise unlocked</button>
            <select value={target} onChange={(e) => setTarget(e.target.value)}>
              <option value="">give remainder to…</option>
              {cats.filter((c) => !c.locked).map((c) => (
                <option key={c.id} value={c.id}>{c.label}</option>
              ))}
            </select>
            <button disabled={!target} onClick={() => target && a.assignRemainder(target)}>Assign</button>
          </div>
        </div>
      )}
      {arr(state.issues).filter((i) => i.severity === 'error').map((i, k) => (
        <div key={k} className="error">{i.message}</div>
      ))}
      <div className="kv"><span>Baryon fraction</span><b>{pct(state.baryonFraction)} <span className="muted">(cosmic Ωb/Ωm = {pct(state.cosmicBaryonFraction)})</span></b></div>
      {state.baryonFraction > state.cosmicBaryonFraction && <div className="warn subtle">Baryon fraction exceeds the cosmic value — unusual for a galaxy halo.</div>}
      <div className="row gap">
        <button onClick={a.undo} disabled={!state.canUndo}>Undo</button>
        <button onClick={a.redo} disabled={!state.canRedo}>Redo</button>
        <button onClick={a.reset}>Reset preset</button>
      </div>
      <p className="note muted">
        Categories never overlap: each unit of mass has one owner. Shares of components inside a category are edited in the inspector.
        Geometry is fixed when masses change, except the halo boundary R200c, which follows M200c.
      </p>
    </div>
  );
}

function PercentInput({ value, disabled, onCommit }: { value: number; disabled?: boolean; onCommit: (f: number) => void }) {
  const fmt = (v: number) => (v * 100 >= 0.01 || v === 0 ? (v * 100).toFixed(4) : (v * 100).toPrecision(3));
  const [s, setS] = useState(fmt(value));
  useEffect(() => setS(fmt(value)), [value]);
  const commit = () => {
    const v = parseFloat(s) / 100;
    if (!isFinite(v)) return setS(fmt(value));
    if (Math.abs(v - value) > 1e-15) onCommit(v);
  };
  return <input className="num" value={s} disabled={disabled} onChange={(e) => setS(e.target.value)} onBlur={commit} onKeyDown={(e) => e.key === 'Enter' && commit()} />;
}

export function MassInput({ value, disabled, onCommit }: { value: number; disabled?: boolean; onCommit: (m: number) => void }) {
  const fmt = (v: number) => v.toExponential(3);
  const [s, setS] = useState(fmt(value));
  useEffect(() => setS(fmt(value)), [value]);
  const commit = () => {
    const v = parseMass(s);
    if (!isFinite(v)) return setS(fmt(value));
    if (Math.abs(v - value) > 1e-12 * Math.abs(value)) onCommit(v);
  };
  return <input className="num wide" value={s} disabled={disabled} onChange={(e) => setS(e.target.value)} onBlur={commit} onKeyDown={(e) => e.key === 'Enter' && commit()} />;
}
