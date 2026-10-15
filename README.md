# Galaxy Explorer

Interactive desktop 3D atlas of galaxy structure with exact, single-counted
mass bookkeeping. Go owns the physics and the mass budget; Three.js renders
representative markers; Wails provides the Windows desktop shell.

**Status:** MVP + V1 (v0.3.0). See [CHANGELOG.md](CHANGELOG.md) and
the acceptance matrix in [docs/ACCEPTANCE.md](docs/ACCEPTANCE.md).

Specification and role skills (Russian): [TECHNICAL_SPECIFICATION.md](TECHNICAL_SPECIFICATION.md),
[SKILLS_INDEX.md](SKILLS_INDEX.md), [skills/](skills/).

## Download (Windows)

A ready-to-run build is attached to each release: open
[Releases](https://github.com/evgenyrvr/galaxy-explorer/releases/latest), download
`GalaxyExplorer-<version>-windows-amd64.zip`, unpack it and start `GalaxyExplorer.exe`.
Requires Windows 10/11 64-bit with the Microsoft Edge WebView2 Runtime (preinstalled on
Windows 11) and a WebGL 2 GPU. The executable is not code-signed, so SmartScreen may ask
for confirmation. No compilation is needed.

## What it does

- **M31 observational** preset: published masses (Chemin+2009, Corbelli+2010, Draine+2014,
  Bender+2005), measured Σ_HI profile, bundled rotation curves and dynamical masses
  ([internal/presets/data/m31](internal/presets/data/m31/README.md)).
- Two normalised presets: **M31-inspired spiral** and **M87-inspired elliptical**,
  default `M200c = 10¹² M☉`, structure and fractions from rounded literature values
  (sources listed in the app and in [internal/galaxy/sources.go](internal/galaxy/sources.go)).
- 3D scene: orbit / zoom / pan from parsecs (nucleus) to hundreds of kpc (R200c),
  views top / edge-on / iso / jet axis, scale bar, axes gizmo.
- Structure tree: matter visibility with tri-state groups, structural labels
  (arms, satellites, tidal features, regions) as filters with AND/OR logic,
  click / Ctrl-click selection in the tree and in the scene, context menu.
- Inspector: single-counted mass of any selection, share of total, mass inside
  R200c, model parameters, evidence level, sources.
- Mass budget editor: M_total, category fractions, locks, "fix total" vs "fix
  component masses", leave-unallocated vs proportional rebalancing, normalise,
  assign remainder, undo/redo; per-component shares in the inspector.
- Plots: `M(<r)`, `ρ(r)` [M☉/pc³], `Σ(R)` [M☉/pc²], `Vc(r)`, `f_b(<r)` — never mixing
  surface and volume density.
- Region selection: sphere `r` or cylinder `R` with `|z|` limit; masses computed in Go.
- Import of observational radial profiles (CSV with units, errors, evidence and source),
  overlaid as points with error bars; comparison with another preset on the same axes.
- Save/load project JSON (also `galaxy.exe file.galaxy.json`), PNG screenshot, mass CSV.

### Profile CSV format

```text
# quantity: surface_density      (volume_density | circular_velocity | enclosed_mass)
# unit: Msun/pc2                 (Msun/kpc2; Msun/pc3, Msun/kpc3; km/s; Msun)
# radius_unit: kpc               (pc | arcsec, with # distance_kpc: 785)
# evidence: OBSERVED             (INFERRED | RECONSTRUCTED | ILLUSTRATIVE)
# source: Author et al. 2020, Journal 1, 2   (required)
r,value,err
2.0,410,35
```

See [docs/examples/synthetic-surface-density.csv](docs/examples/synthetic-surface-density.csv) (synthetic numbers).

## Physics model (short)

| Component | Model |
|---|---|
| Stellar / gas / dust discs | Exponential (Freeman 1970) or exponential with central hole (McMillan 2017) × sech² vertical, optional flare |
| Spiral arms | Azimuthal modulation `1 + A cos ψ` of existing disc matter — Σ(R) and mass unchanged; arm mass is analytic |
| Bulge, elliptical spheroid | Triaxial Hernquist (1990); spherical M(<r) by angular quadrature |
| Stellar halo, GC system | Cored power law |
| Hot gas | β-model (Cavaliere & Fusco-Femiano 1976) truncated at R200c |
| Dark matter | NFW truncated at R200c, `c(M)` of Dutton & Macciò (2014) |
| Halo boundary | `M200c` within `R200c`, Planck 2018 cosmology |
| Giant Stellar Stream, shells | Particle-spray tidal debris (Fardal+2015) integrated with leapfrog in the preset's own mass model |
| Rotation curve | Freeman disc formula (Bessel functions) for exponential stellar discs, spherical approximation otherwise |

Every component mass is `M_total × f_category × share`, so each unit of mass has
exactly one owner. Markers carry weight `M/N` and never define physical masses.

## Development

Requirements (Windows): Go ≥ 1.25, Node ≥ 20, Wails CLI v2, WebView2 runtime.

```powershell
go install github.com/wailsapp/wails/v2/cmd/wails@latest
wails dev                 # live development
wails build               # build\bin\galaxy.exe
go test ./...             # physics, budget, presets, project
cd frontend; npm test     # frontend unit tests
```

`GALAXY_PRESET=m87-normalized` opens a different preset at start-up.

## Layout

```text
main.go, app.go           Wails entry point and bound API (+ /api/particles binary endpoint)
internal/units            constants, Planck 2018, R200c, c(M)
internal/physics          distributions, CDF tables, discs, spheroids, orbits, Bessel functions
internal/ontology         matter types, evidence levels, structural tags
internal/budget           mass budget engine, validation, undo/redo
internal/galaxy           model: components, selection mass, profiles, markers, sources
internal/presets          M31-inspired and M87-inspired presets
internal/obsdata          observational profile import
internal/project          project file format and CSV export
frontend/src/scene        Three.js scene, shaders, picking
frontend/src/ui           tree, inspector, budget editor, charts
docs/                     ADRs and acceptance matrix
```
