# M31 published tables

Transcribed programmatically from the LaTeX sources of the papers on arXiv
(no manual retyping). Radii are the authors' values for D = 785 ± 25 kpc
(McConnachie et al. 2005). All tables are INFERRED quantities: rotation
velocities come from tilted-ring fits to observed line-of-sight velocities,
Σ_HI from ring-averaged 21-cm emission, M(<R) from dynamical modelling.

| File | Paper | Table | Content |
|---|---|---|---|
| chemin2009_rotation_curve.csv | Chemin, Carignan & Foster 2009, ApJ 705, 1395 — arXiv:0909.3846 | Table 4 | H I rotation velocity ± error, 98 rings, 1.1–38 kpc |
| chemin2009_hi_surface_density.csv | same | Table 4 | Face-on Σ_HI (hydrogen only), 100 rings; integrates to 4.23×10⁹ M☉ |
| corbelli2010_rotation_curve.csv | Corbelli et al. 2010, A&A 511, A89 — arXiv:0912.4133 | Table 1 | H I rotation velocity ± error with warp model, 8.5–36.5 kpc |
| corbelli2010_enclosed_mass.csv | same | Table 3 | Compilation of dynamical M(<R) estimates to 560 kpc with asymmetric errors and original references |

Reproduce with [tools/extract_m31.py](../../../../tools/extract_m31.py):

```bash
mkdir -p src/0909.3846 src/0912.4133
curl -L https://arxiv.org/e-print/0909.3846 | tar xz -C src/0909.3846
curl -L https://arxiv.org/e-print/0912.4133 | tar xz -C src/0912.4133
python -I tools/extract_m31.py src internal/presets/data/m31
```
