# Changelog

## 0.3.3 — 2026-10-15

- Fix (plots): the central black hole was drawn in the same white as the total curve, so
  with the black hole selected the DM-dominated total looked like its profile. The black
  hole is now orange, and with a selection the total is a grey dashed reference.
- Plots note that a point mass has no finite ρ or Σ (δ-function at r = 0) and a Keplerian Vc.
- Regression test: black-hole M(<r) is constant, Vc = √(GM/r), ρ = 0 in all presets.

## 0.3.2 — 2026-10-15

- Fix: expanding the plots to the whole window froze the window (≈2 FPS, empty plot).
  The collapsed 3D scene kept rendering into a degenerate 1-pixel viewport; it is now
  paused while it has no area and resumes on Collapse with the same camera.
- Frontend errors are appended to %TEMP%\galaxy-explorer.log for diagnosis.

## 0.3.1 — 2026-10-15

- Side panels can be hidden with the ‹ › handles on the scene edges to enlarge the scene.
- Plots can be expanded to the whole window (⤢ Expand / ⤡ Collapse, or Esc).
- Radius axis of the plots switches between logarithmic and linear (x: log / lin).
- Panel visibility and the axis mode are saved in projects.
- Fix: hiding the plots no longer leaves an empty strip below the scene.

## 0.3.0 — 2026-10-15

- Observational preset **M31 observational**: masses from Chemin+2009 (stellar disc and
  bulge, H I, H₂ via Dame+1993), Draine+2014 (dust), Bender+2005 (black hole) and the
  Corbelli+2010 ΛCDM NFW halo (C = 12, M₉₈ = 1.2×10¹² M☉) re-expressed as M200c for
  Planck 2018 together with the baryons (R200c = 222 kpc, M200c = 1.17×10¹² M☉).
  The H I disc uses the measured Σ_HI table as its radial profile.
- Bundled published tables (Chemin+2009 rotation curve and Σ_HI, Corbelli+2010 rotation
  curve and compiled dynamical M(<R)), transcribed from the arXiv LaTeX sources.
- Model-vs-data tests: ∫Σ_HI dA = 4.23×10⁹ M☉ (equals the flux-based mass); Vc agrees
  with Corbelli+2010 to 5% and with Chemin+2009 to 10% (model ~7–10% low at 10–30 kpc
  with M/L_disc = 1.7); dynamical M(<R) to 560 kpc with χ²/n ≈ 1.0. No parameter was fitted.
- "Modified reconstruction" badge when an observational preset's budget is edited;
  data mode "observed + inferred".
- Profile import: asymmetric errors (err_minus, err_plus), equal radii, text columns.
- Box selection: Alt + drag in the scene selects every component with ≥ 1% of its drawn
  markers inside the rectangle (Ctrl/Shift adds to the selection).
- Script tools/extract_m31.py reproduces the bundled tables from the arXiv sources.

## 0.2.0 — 2026-10-15

V1, first part.

- Quit button (✕) next to Open.
- Region selection: sphere (r) or cylinder (R, |z|). Masses in Go — analytic for
  spheroids and untilted discs, otherwise from a deterministic 60 000-point reference
  sample (flagged as an estimate). Optional hiding of markers outside the region.
- Observational profile import (CSV): Σ(R), ρ(r), Vc(r), M(<r) with units, 1σ errors,
  evidence level and mandatory source; strict units, no Σ↔ρ conversion, radii in
  kpc, pc or arcsec (with distance). Shown as points with error bars; saved in projects.
- Preset comparison: total curves of another preset at the same M_total, dashed.
- `galaxy.exe project.galaxy.json` opens a project from the command line.
- Example import file: docs/examples/synthetic-surface-density.csv (synthetic values).

## 0.1.0 — 2026-10-15

Stage 0 and MVP core.

- Go physics core: unit-mass distributions (exponential and holed discs with sech²
  vertical structure and flare, triaxial Hernquist, NFW, β-model, cored power law,
  Plummer, point mass, sampled structures), spiral arms as mass-conserving azimuthal
  modulation, Freeman rotation curve, leapfrog orbits and particle-spray debris.
- Mass budget engine: non-overlapping categories, component shares, fix-total /
  fix-components modes, leave-unallocated / proportional rules, locks, normalise,
  assign remainder, validation, undo/redo.
- Presets: M31-inspired spiral (thin/thick disc, bulge, halo, nucleus, 450 GCs, Giant
  Stellar Stream, M32, NGC 205, H I, H₂, WIM, hot CGM, dust, NFW halo, SMBH) and
  M87-inspired elliptical (triaxial spheroid with inner/outer regions, GCs, shells,
  dwarf satellite, hot atmosphere, jet, radio lobes, Hα filaments, nuclear dust disc,
  NFW halo, SMBH), with a source registry.
- Wails app: 3D scene (~280k markers), structure tree with labels and AND/OR filters,
  picking, context menu, inspector, budget editor, plots, data-mode filter, project
  save/load, PNG and CSV export.
- Tests: Go unit tests for physics, budget, presets and project round trip (AC-03,
  05, 06, 07, 08, 11, 14, 15); frontend format tests.
