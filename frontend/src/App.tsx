import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import * as THREE from 'three';
import { Quit } from '../wailsjs/runtime/runtime';
import { api, arr, Dataset, errorText, PresetInfo, profileRequest, Profiles, Region, regionActive, selection as toSel, Selection, SelectionReport, State } from './api';
import { length, mass, parseMass, pct } from './format';
import { CameraState, GalaxyScene, SceneStats } from './scene/GalaxyScene';
import { BudgetActions, BudgetEditor } from './ui/BudgetEditor';
import { Charts } from './ui/Charts';
import { Inspector } from './ui/Inspector';
import { RegionPanel } from './ui/RegionPanel';
import { StructureTree } from './ui/StructureTree';

type DataMode = 'all' | 'no-illustrative' | 'inferred' | 'observed';

interface UIDoc {
  camera: CameraState;
  visible: string[];
  selection: Selection;
  filterTags: string[];
  filterOr: boolean;
  dataMode: DataMode;
  quality: number;
  showHalo: boolean;
  showLeft?: boolean;
  showRight?: boolean;
  xLog?: boolean;
  region?: Region;
  hideOutside?: boolean;
  compareId?: string;
}

const QUALITIES = [0.25, 0.5, 1, 2];

export default function App() {
  const sceneHost = useRef<HTMLDivElement>(null);
  const scene = useRef<GalaxyScene | null>(null);
  const [presets, setPresets] = useState<PresetInfo[]>([]);
  const [state, setState] = useState<State | null>(null);
  const [visible, setVisible] = useState<Set<string>>(new Set());
  const [sel, setSel] = useState<Selection>({ components: [], tags: [] });
  const [report, setReport] = useState<SelectionReport | null>(null);
  const [profiles, setProfiles] = useState<Profiles | null>(null);
  const [filterTags, setFilterTags] = useState<string[]>([]);
  const [filterOr, setFilterOr] = useState(false);
  const [dataMode, setDataMode] = useState<DataMode>('all');
  const [quality, setQuality] = useState(1);
  const [showHalo, setShowHalo] = useState(true);
  const [tab, setTab] = useState<'inspector' | 'budget' | 'sources'>('inspector');
  const [toast, setToast] = useState<{ text: string; kind: 'error' | 'info' } | null>(null);
  const [stats, setStats] = useState<SceneStats | null>(null);
  const [menu, setMenu] = useState<{ x: number; y: number; id: string } | null>(null);
  const [loading, setLoading] = useState(false);
  const [showCharts, setShowCharts] = useState(true);
  const [showLeft, setShowLeft] = useState(true);
  const [showRight, setShowRight] = useState(true);
  const [chartsMax, setChartsMax] = useState(false);
  const [xLog, setXLog] = useState(true);
  const [region, setRegion] = useState<Region>({ kind: '', rMin: 0, rMax: 10, zMax: 0 });
  const [hideOutside, setHideOutside] = useState(true);
  const [datasets, setDatasets] = useState<Dataset[]>([]);
  const [compareId, setCompareId] = useState('');
  const [compare, setCompare] = useState<Profiles | null>(null);
  const pendingUI = useRef<UIDoc | null>(null);

  const notify = useCallback((text: string, kind: 'error' | 'info' = 'error') => {
    setToast({ text, kind });
    window.setTimeout(() => setToast((t) => (t?.text === text ? null : t)), kind === 'error' ? 6000 : 3000);
  }, []);

  // ---- scene lifecycle
  useEffect(() => {
    if (!sceneHost.current) return;
    const s = new GalaxyScene(sceneHost.current);
    s.onError = (m) => notify(m);
    scene.current = s;
    const t = window.setInterval(() => setStats(s.stats()), 500);
    return () => {
      window.clearInterval(t);
      s.dispose();
      scene.current = null;
    };
  }, [notify]);

  // ---- initial load
  useEffect(() => {
    (async () => {
      const p = await api.ListPresets();
      setPresets(p);
      const start = await api.StartupProject().catch((e) => {
        notify(errorText(e));
        return null;
      });
      if (start) await applyProject(start);
      else if (p.length) await loadPreset(p[0].id);
    })().catch((e) => notify(errorText(e)));
  }, []);

  const loadPreset = async (id: string) => {
    setLoading(true);
    try {
      const st = await api.LoadPreset(id);
      setSel({ components: [], tags: [] });
      setFilterTags([]);
      setVisible(new Set(arr(st.components).map((c) => c.id)));
      setState(st);
      setDatasets(arr(await api.Datasets()));
      setCompareId('');
      const sc = scene.current;
      if (sc) {
        sc.controls.target.set(0, 0, 0);
        sc.camera.position.set(45, -45, 30);
        sc.camera.up.set(0, 0, 1);
        sc.controls.update();
      }
    } catch (e) {
      notify(errorText(e));
    } finally {
      setLoading(false);
    }
  };

  // ---- push state into the scene
  useEffect(() => {
    if (!state || !scene.current) return;
    setLoading(true);
    scene.current.setComponents(arr(state.components), quality).finally(() => {
      setLoading(false);
      const ui = pendingUI.current;
      if (ui && scene.current) {
        scene.current.setCamera(ui.camera);
        pendingUI.current = null;
      }
    });
    scene.current.setHaloSphere(state.r200c, showHalo);
  }, [state, quality, showHalo]);

  const hiddenByMode = useMemo(() => {
    const h = new Set<string>();
    for (const c of arr(state?.components)) {
      if (dataMode === 'observed' && c.evidence !== 'OBSERVED') h.add(c.id);
      if (dataMode === 'no-illustrative' && c.evidence === 'ILLUSTRATIVE') h.add(c.id);
      if (dataMode === 'inferred' && c.evidence !== 'OBSERVED' && c.evidence !== 'INFERRED') h.add(c.id);
    }
    return h;
  }, [state, dataMode]);

  const effectiveVisible = useMemo(() => new Set([...visible].filter((id) => !hiddenByMode.has(id))), [visible, hiddenByMode]);

  const tagBit = useMemo(() => new Map(arr(state?.tags).map((t) => [t.id, t.bit])), [state]);
  const maskOf = (ids: string[]) => ids.reduce((m, id) => (tagBit.has(id) ? m | (1 << tagBit.get(id)!) : m), 0);

  useEffect(() => scene.current?.setVisibility(effectiveVisible), [effectiveVisible, state]);
  useEffect(() => scene.current?.setFilter(maskOf(filterTags), filterOr), [filterTags, filterOr, tagBit, state]);
  useEffect(() => scene.current?.setSelection(new Set(sel.components), maskOf(sel.tags)), [sel, tagBit, state]);

  const regionOn = regionActive(region);
  const fullSel: Selection = useMemo(() => ({ ...sel, region: regionOn ? region : undefined }), [sel, region, regionOn]);
  useEffect(() => {
    const kind = regionOn && hideOutside ? (region.kind === 'sphere' ? 1 : 2) : 0;
    scene.current?.setRegion(kind, region.rMin, region.rMax, region.zMax);
  }, [region, regionOn, hideOutside, state]);

  // ---- selection mass (computed in Go)
  useEffect(() => {
    if (!state) return;
    if (sel.components.length + sel.tags.length === 0 && !regionOn) return setReport(null);
    api.SelectionMass(toSel(fullSel)).then(setReport).catch((e) => notify(errorText(e)));
  }, [fullSel, state]);

  // ---- profiles
  const chartIds = sel.components.length ? sel.components : sel.tags.length ? arr(report?.parts).map((p) => p.component) : [];
  const matterLabels = useMemo(() => Object.fromEntries(arr(state?.categories).map((c) => [c.id, c.label])), [state]);
  useEffect(() => {
    if (!state) return;
    api.Profiles(profileRequest([])).then(setProfiles).catch((e) => notify(errorText(e)));
  }, [state]);
  useEffect(() => {
    if (!state || !compareId || compareId === state.preset.id) return setCompare(null);
    api.CompareProfiles(compareId).then(setCompare).catch((e) => notify(errorText(e)));
  }, [state, compareId]);
  const importProfile = async () => {
    try {
      setDatasets(await api.ImportProfile());
    } catch (e) {
      notify(errorText(e));
    }
  };

  // ---- budget actions
  const run = async (p: Promise<State>) => {
    try {
      setState(await p);
    } catch (e) {
      notify(errorText(e));
      api.State().then(setState).catch(() => {});
    }
  };
  const actions: BudgetActions = {
    setTotal: (m) => run(api.SetTotalMass(m)),
    setFraction: (c, f, r) => run(api.SetFraction(c, f, r)),
    setCategoryMass: (c, m) => run(api.SetCategoryMass(c, m)),
    setLocked: (c, l) => run(api.SetLocked(c, l)),
    setMode: (m) => run(api.SetMode(m)),
    normalize: () => run(api.Normalize()),
    assignRemainder: (c) => run(api.AssignRemainder(c)),
    undo: () => run(api.Undo()),
    redo: () => run(api.Redo()),
    reset: () => run(api.ResetBudget()),
  };

  // ---- selection helpers
  const selectComponent = (id: string | null, additive: boolean) => {
    if (!id) return additive ? undefined : setSel({ components: [], tags: [] });
    setSel((s) => {
      if (!additive) return { components: [id], tags: [] };
      const has = s.components.includes(id);
      return { ...s, components: has ? s.components.filter((x) => x !== id) : [...s.components, id] };
    });
    setTab('inspector');
  };
  const selectTag = (id: string, additive: boolean) => {
    setSel((s) => {
      if (!additive) return { components: [], tags: [id] };
      const has = s.tags.includes(id);
      return { ...s, tags: has ? s.tags.filter((x) => x !== id) : [...s.tags, id] };
    });
    setTab('inspector');
  };

  useEffect(() => {
    const s = scene.current;
    if (!s) return;
    s.onPick((id, ev) => {
      setMenu(null);
      selectComponent(id, ev.ctrlKey || ev.shiftKey || ev.metaKey);
    });
    s.onContext((id, ev) => {
      if (id) setMenu({ x: ev.clientX, y: ev.clientY, id });
    });
    s.onBox((hits, additive) => {
      // A component is boxed when ≥ 1% of its drawn markers (or its single
      // discrete marker) fall inside the rectangle.
      const ids = hits.filter((h) => h.inside > 0 && (h.inside / h.shown >= 0.01 || h.shown <= 1000)).map((h) => h.id);
      setSel((cur) => ({ components: additive ? [...new Set([...cur.components, ...ids])] : ids, tags: additive ? cur.tags : [] }));
      setTab('inspector');
      notify(ids.length ? `Box: ${ids.length} component(s) selected` : 'Box: nothing selected', 'info');
    });
  });

  // ---- close the context menu on any outside click
  useEffect(() => {
    if (!menu) return;
    const close = (e: PointerEvent) => {
      if (!(e.target as HTMLElement).closest('.context')) setMenu(null);
    };
    window.addEventListener('pointerdown', close);
    return () => window.removeEventListener('pointerdown', close);
  }, [menu]);

  // ---- keyboard: Ctrl+Z / Ctrl+Y, Escape
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if ((e.target as HTMLElement)?.tagName === 'INPUT') return;
      if (e.ctrlKey && e.key.toLowerCase() === 'z') actions.undo();
      if (e.ctrlKey && e.key.toLowerCase() === 'y') actions.redo();
      if (e.key === 'Escape') {
        if (chartsMax) return setChartsMax(false);
        setSel({ components: [], tags: [] });
        setMenu(null);
      }
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  });

  const zoomTo = (id: string) => {
    const s = scene.current;
    const c = arr(state?.components).find((x) => x.id === id);
    if (!s || !c) return;
    if (c.visual.style === 'icon') return s.frame(new THREE.Vector3(), 0.01); // 10 pc around the nucleus
    if (c.params?.centre_r_kpc !== undefined) return s.frame(s.componentCenter(id), Math.max(1, c.maxRadius > 0 ? 3 * (c.params.a_kpc ?? 1) : 2));
    s.frame(new THREE.Vector3(), Math.max(0.01, Math.min(c.maxRadius, 400)) * 0.8);
  };

  const jet = arr(state?.components).find((c) => c.structure === 'jet');

  // ---- files
  const uiDoc = (): UIDoc => ({
    camera: scene.current!.getCamera(), visible: [...visible], selection: sel, filterTags, filterOr, dataMode, quality, showHalo,
    region, hideOutside, compareId, showLeft, showRight, xLog,
  });
  const save = async () => {
    try {
      const p = await api.SaveProject(JSON.stringify(uiDoc()));
      if (p) notify(`Saved ${p}`, 'info');
    } catch (e) {
      notify(errorText(e));
    }
  };
  const open = async () => {
    try {
      const r = await api.OpenProject();
      if (r) await applyProject(r);
    } catch (e) {
      notify(errorText(e));
    }
  };
  const applyProject = async (r: Awaited<ReturnType<typeof api.OpenProject>>) => {
    try {
      const ui: Partial<UIDoc> = r.ui ? JSON.parse(r.ui) : {};
      setVisible(new Set(ui.visible ?? arr(r.state.components).map((c) => c.id)));
      setSel(ui.selection ?? { components: [], tags: [] });
      setFilterTags(ui.filterTags ?? []);
      setFilterOr(ui.filterOr ?? false);
      setDataMode(ui.dataMode ?? 'all');
      setQuality(ui.quality ?? 1);
      setShowHalo(ui.showHalo ?? true);
      setRegion(ui.region ?? { kind: '', rMin: 0, rMax: 10, zMax: 0 });
      setHideOutside(ui.hideOutside ?? true);
      setShowLeft(ui.showLeft ?? true);
      setShowRight(ui.showRight ?? true);
      setXLog(ui.xLog ?? true);
      setCompareId(ui.compareId ?? '');
      setDatasets(await api.Datasets());
      if (ui.camera) pendingUI.current = ui as UIDoc;
      setState(r.state);
      notify(`Opened ${r.path}`, 'info');
    } catch (e) {
      notify(errorText(e));
    }
  };
  const shot = async () => {
    try {
      const p = await api.SaveScreenshot(scene.current!.screenshot());
      if (p) notify(`Saved ${p}`, 'info');
    } catch (e) {
      notify(errorText(e));
    }
  };
  const csv = async () => {
    try {
      const p = await api.ExportMassCSV(toSel(fullSel));
      if (p) notify(`Saved ${p}`, 'info');
    } catch (e) {
      notify(errorText(e));
    }
  };

  const [totalText, setTotalText] = useState('');
  useEffect(() => setTotalText(state ? state.totalMass.toExponential(2) : ''), [state?.totalMass]);
  const commitTotal = () => {
    const v = parseMass(totalText);
    if (state && isFinite(v) && v !== state.totalMass) actions.setTotal(v);
    else if (state) setTotalText(state.totalMass.toExponential(2));
  };

  const visibleMass = useMemo(
    () => arr(state?.components).filter((c) => effectiveVisible.has(c.id)).reduce((s, c) => s + c.mass, 0),
    [state, effectiveVisible],
  );
  const scaleBar = stats ? niceScale(stats.kpcPerPixel * 120) : null;

  return (
    <div className="app">
      <header className="topbar">
        <div className="brand">Galaxy Explorer</div>
        <select value={state?.preset.id ?? ''} onChange={(e) => loadPreset(e.target.value)}>
          {presets.map((p) => (
            <option key={p.id} value={p.id}>{p.name}</option>
          ))}
        </select>
        <span className={`badge ${state?.preset.kind === 'observational' && !state.modified ? 'obs' : 'recon'}`} title={state?.preset.disclaimer}>
          {state?.preset.kind === 'observational'
            ? state.modified ? 'Modified reconstruction · edited observational preset' : 'Observational preset · inferred masses'
            : 'Synthetic reconstruction · normalised preset'}
        </span>
        <label className="small">
          M_total
          <input className="num wide" value={totalText} onChange={(e) => setTotalText(e.target.value)} onBlur={commitTotal} onKeyDown={(e) => e.key === 'Enter' && commitTotal()} /> M☉
        </label>
        <div className="sep" />
        <button onClick={() => scene.current?.view('top')}>Top</button>
        <button onClick={() => scene.current?.view('edge')}>Edge-on</button>
        <button onClick={() => scene.current?.view('iso')}>Iso</button>
        {jet && <button onClick={() => { const ax = scene.current?.componentAxis(jet.id); if (ax) scene.current?.view('axis', ax); }}>Jet axis</button>}
        <button onClick={() => scene.current?.frame(new THREE.Vector3(), 30)} title="Disc scale">30 kpc</button>
        <button onClick={() => scene.current?.frame(new THREE.Vector3(), state?.r200c ?? 200)} title="Halo scale">R200c</button>
        <button onClick={() => zoomTo('bh.central')} title="Nucleus scale">Nucleus</button>
        <div className="sep" />
        <label className="small"><input type="checkbox" checked={showHalo} onChange={(e) => setShowHalo(e.target.checked)} /> R200c</label>
        <label className="small">
          Markers
          <select value={quality} onChange={(e) => setQuality(parseFloat(e.target.value))}>
            {QUALITIES.map((q) => <option key={q} value={q}>×{q}</option>)}
          </select>
        </label>
        <label className="small">
          Data
          <select value={dataMode} onChange={(e) => setDataMode(e.target.value as DataMode)}>
            <option value="all">all (labelled)</option>
            <option value="no-illustrative">hide illustrative</option>
            <option value="inferred">observed + inferred</option>
            <option value="observed">observed only</option>
          </select>
        </label>
        <div className="file-actions">
          <button onClick={save}>Save</button>
          <button onClick={shot}>PNG</button>
          <button onClick={csv}>CSV</button>
          <button onClick={open}>Open</button>
          <button className="close-btn" onClick={() => Quit()} title="Quit Galaxy Explorer" aria-label="Quit">✕</button>
        </div>
      </header>

      <main className={`main ${showLeft ? '' : 'no-left'} ${showRight ? '' : 'no-right'} ${chartsMax && showCharts ? 'charts-max' : ''}`}>
        <aside className="left">{state && (
          <StructureTree
            state={state} visible={visible} hiddenByMode={hiddenByMode} setVisible={setVisible} selection={sel}
            onSelectComponent={selectComponent} onSelectTag={selectTag}
            filterTags={filterTags} setFilterTags={setFilterTags} filterOr={filterOr} setFilterOr={setFilterOr}
            onIsolate={() => setVisible(new Set(sel.components))}
          />
        )}
          <RegionPanel region={region} setRegion={setRegion} hideOutside={hideOutside} setHideOutside={setHideOutside} />
        </aside>

        <section className={`center ${showCharts ? '' : 'no-charts'}`}>
          <div className="scene" ref={sceneHost}>
            <button className="panel-toggle left-toggle" onClick={() => setShowLeft(!showLeft)} title={showLeft ? 'Hide the structure panel' : 'Show the structure panel'}>
              {showLeft ? '‹' : '›'}
            </button>
            <button className="panel-toggle right-toggle" onClick={() => setShowRight(!showRight)} title={showRight ? 'Hide the inspector panel' : 'Show the inspector panel'}>
              {showRight ? '›' : '‹'}
            </button>
            {loading && <div className="loading">Generating markers…</div>}
            {(dataMode === 'observed' || dataMode === 'inferred') && hiddenByMode.size === arr(state?.components).length && (
              <div className="banner">This preset contains no {dataMode === 'observed' ? 'directly observed' : 'observed or inferred'} components, so nothing is shown in this mode. Fake observational points are never generated.</div>
            )}
            {dataMode === 'observed' && state?.preset.kind === 'observational' && (
              <div className="banner subtle">Masses and profiles of this preset are inferred from observations (M/L, tilted rings, halo fits), not measured directly; use “observed + inferred”. The bundled tables are shown in the plots.</div>
            )}
            {scaleBar && (
              <div className="scalebar">
                <div className="bar" style={{ width: `${scaleBar.px(stats!.kpcPerPixel)}px` }} />
                <span>{length(scaleBar.kpc)}</span>
              </div>
            )}
          </div>
          {showCharts && state && <Charts
            profiles={profiles} compare={compare} compareName={presets.find((x) => x.id === compareId)?.name ?? ''}
            components={arr(state.components)} show={chartIds} matterLabels={matterLabels}
            datasets={datasets} presets={presets} currentPreset={state.preset.id} compareId={compareId} setCompareId={setCompareId}
            onImport={importProfile} onRemoveDataset={(id) => api.RemoveDataset(id).then(setDatasets)}
            xLog={xLog} setXLog={setXLog} expanded={chartsMax} onToggleExpand={() => setChartsMax(!chartsMax)}
          />}
        </section>

        <aside className="right">
          <div className="tabs">
            <button className={tab === 'inspector' ? 'on' : ''} onClick={() => setTab('inspector')}>Inspector</button>
            <button className={tab === 'budget' ? 'on' : ''} onClick={() => setTab('budget')}>Mass budget</button>
            <button className={tab === 'sources' ? 'on' : ''} onClick={() => setTab('sources')}>Sources</button>
          </div>
          <div className="right-body">
            {state && tab === 'inspector' && (
              <Inspector state={state} selection={sel} report={report} region={regionOn ? region : null}
                onShare={(c, k, s) => run(api.SetShare(c, k, s))} onShareLock={(c, k, l) => run(api.SetShareLocked(c, k, l))} />
            )}
            {state && tab === 'budget' && <BudgetEditor state={state} a={actions} />}
            {state && tab === 'sources' && <SourcesPanel state={state} />}
          </div>
        </aside>
      </main>

      <footer className="statusbar">
        <span>Markers drawn: {stats ? stats.visibleMarkers.toLocaleString() : '—'}</span>
        <span>Visible matter: {mass(visibleMass)} ({state ? pct(visibleMass / state.totalMass) : '—'}) — visibility does not change masses</span>
        <span className={state && Math.abs(state.unallocated) > 1e-9 ? 'bad' : 'ok'}>
          Mass budget: {state ? pct(1 - state.unallocated) : '—'}
        </span>
        <span>Data: model / {dataMode}</span>
        <span>View distance: {stats ? length(stats.distance) : '—'}</span>
        <span>{stats ? `${stats.fps.toFixed(0)} FPS` : ''}</span>
        <button className="link" onClick={() => setShowCharts(!showCharts)}>{showCharts ? 'Hide plots' : 'Show plots'}</button>
      </footer>

      {menu && (
        <div className="context" style={{ left: menu.x, top: menu.y }}>
          <div className="context-title">{arr(state?.components).find((c) => c.id === menu.id)?.name}</div>
          <button onClick={() => { setVisible(new Set([menu.id])); setMenu(null); }}>Isolate</button>
          <button onClick={() => { const v = new Set(visible); v.delete(menu.id); setVisible(v); setMenu(null); }}>Hide</button>
          <button onClick={() => { zoomTo(menu.id); setMenu(null); }}>Zoom to</button>
          <button onClick={() => { selectComponent(menu.id, false); setMenu(null); }}>Show info</button>
        </div>
      )}
      {toast && <div className={`toast ${toast.kind}`} onClick={() => setToast(null)}>{toast.text}</div>}
    </div>
  );
}

function SourcesPanel({ state }: { state: State }) {
  return (
    <div className="inspector">
      <div className="panel-title">Sources</div>
      <p className="note muted">Rounded literature values used to shape this normalised preset. They are not fits performed by this program.</p>
      {arr(state.sources).map((s) => (
        <div key={s.id} className="source">
          <a onClick={() => import('./api').then((m) => m.openURL(s.url))}>{s.citation}</a>
          <div className="muted small">{s.usedFor}</div>
        </div>
      ))}
    </div>
  );
}

/** A 1/2/5×10ⁿ length close to the target, for the scale bar. */
function niceScale(targetKpc: number) {
  const e = Math.floor(Math.log10(targetKpc));
  const base = Math.pow(10, e);
  const m = targetKpc / base;
  const nice = (m >= 5 ? 5 : m >= 2 ? 2 : 1) * base;
  return { kpc: nice, px: (kpcPerPx: number) => nice / kpcPerPx };
}
