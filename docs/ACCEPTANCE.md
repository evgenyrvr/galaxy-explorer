# Acceptance matrix (spec §14)

| ID | Status | How it is verified |
|---|---|---|
| AC-01 Rotate thin disc 180°, see thickness edge-on | ✅ manual | Orbit controls; "Edge-on" view shows the sech² thickness and dust lane |
| AC-02 Only thin disc visible, inspector shows its mass | ✅ auto · 🟡 UI pending manual check | `TestAC03ArmsNotDoubleCounted` (thin-disc selection = budget mass); context menu "Isolate" |
| AC-03 Thin disc + arm 1 not double counted | ✅ auto | `TestAC03ArmsNotDoubleCounted`, `TestArmsConserveAzimuthalAverage`, `TestDiscNormalisationAndArms` |
| AC-04 Hide stars, gas/DM translucent, masses unchanged | 🟡 implemented, manual check pending | Visibility is frontend-only; budget never reads it. Status bar states it |
| AC-05 DM 80%, stars 15% at fixed total | ✅ auto | `TestAC05RemainderIsExplicit`: leave-unallocated refuses >100%, proportional rule distributes the remaining 5% explicitly |
| AC-06 M_total 10¹² → 2×10¹² | ✅ auto | `TestAC06TotalMassScales`, `TestAC06AndAC15` (R200c-dependent geometry updates, fixed geometry keeps its key) |
| AC-07 Satellite inside R200c | ✅ auto | `TestAC07Satellites`, `TestComponentsInsideR200c`; inspector shows "Inside R200c" |
| AC-08 Hot halo only | ✅ auto | `TestAC08HotOnly` |
| AC-09 Observed-only mode | 🟡 implemented, manual check pending (modes: observed only / observed + inferred) | "Data: observed only" hides all non-OBSERVED components and explains that the synthetic preset has none; no fake observations |
| AC-10 Import Σ(R) table | ✅ auto + manual | `TestAC10SurfaceDensityImport`, `TestRejectsCrossQuantityUnits`, `TestRequiresProvenance`: Σ stays Σ in M☉/pc², other quantities' units are refused, no points are added; plotted as points with error bars on the Σ(R) tab |
| AC-11 Save/load project | ✅ auto + manual | `TestRoundTrip` (budget, seed, UI document); camera, visibility, selection, region, comparison and imported datasets restored (checked by opening a project from the command line) |
| AC-12 Zoom to BH then to satellites | 🟡 implemented, manual check pending | "Nucleus" frames 10 pc, BH is a fixed-size schematic icon; logarithmic depth buffer, float64 → float32 in kpc around the centre |
| AC-13 Elliptical has no spiral disc | ✅ manual | M87 preset: spheroid, hot atmosphere, BH, jet, lobes, shells, GCs |
| AC-14 Negative mass / >100% | ✅ auto | `TestAC14Validation` |
| AC-15 Marker count independent of mass | ✅ auto | `TestAC06AndAC15`, `TestParticlesBuffer` (discrete objects drawn once) |

Region selection (sphere r, cylinder R with |z|): `TestRegionSelection` (analytic disc fraction checked against 1 − (1 + x)e⁻ˣ).

Box selection on screen: Alt + drag (checked in the running app).

Not yet covered:
Undo/Redo of visibility (only the budget is undoable), visual golden tests.
