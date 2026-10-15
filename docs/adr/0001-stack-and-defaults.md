# ADR 0001 — Stack, platform and scientific defaults

Date: 2026-10-15 · Status: accepted

## Decisions

1. **Stack:** Go (physics, mass budget, presets, I/O) + Wails v2 + React/TypeScript/Vite +
   Three.js; plots with uPlot (small, fast, log axes).
2. **Platform:** Windows 11 only for the MVP; target GPU Intel Arc class or better.
   Measured on the development machine (Intel Arc 140T): ~280k markers at 75–115 FPS.
3. **Language:** UI, code, comments and commits in English. Specification stays in Russian.
4. **Presets:** geometry and default fractions from rounded literature values of known
   galaxies (M31, M87); masses follow the user's `M_total` (normalised mode).
5. **Halo boundary:** `M200c` inside `R200c`, Planck 2018 cosmology. Satellites inside
   `R200c` are carved out of the host categories (already part of `M200c`).
6. **Dust:** its own matter category, never inside the gas budget.
7. **Physics first:** profiles are physically motivated (NFW, Hernquist, exponential/sech²
   discs, β-model), streams and shells come from orbit integration in the preset's own
   mass model, the rotation curve uses the Freeman formula for exponential discs.

## Consequences

- Markers are fetched as binary buffers through a Wails asset-server middleware
  (`/api/particles`), not JSON, to keep 10⁵–10⁶ floats cheap.
- Geometry is fixed when masses change ("geometry fixed" mode, spec §5.3), except
  quantities tied to `R200c` (dark halo, hot gas, stellar-halo truncation).
- Orbit-based structures are integrated once in the default mass model and cached;
  they do not react to budget edits. This is stated in the component notes.
