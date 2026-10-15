# Changelog

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
