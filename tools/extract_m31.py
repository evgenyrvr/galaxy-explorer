"""Transcribe M31 tables from arXiv LaTeX sources into profile CSVs."""
import re
import sys

sp, out = sys.argv[1], sys.argv[2]
BS = "\\"


def cells(line):
    return [x.strip().rstrip(BS).strip() for x in line.split("&")]


tex = open(sp + "/0909.3846/lc1.tex", encoding="latin-1").read().splitlines()
i = next(k for k, l in enumerate(tex) if "begin{deluxetable}{ccccccccccc}" in l)
s = next(k for k in range(i, len(tex)) if "startdata" in tex[k])
rows = []
for l in tex[s + 1:]:
    if "enddata" in l:
        break
    c = cells(l)
    if len(c) >= 11:
        rows.append(c)

common = (
    "# distance_kpc: 785\n"
    "# url: https://arxiv.org/abs/0909.3846\n"
    "# source: Chemin, Carignan & Foster 2009, ApJ 705, 1395 (Table 4, transcribed from the arXiv source)\n"
)
with open(out + "/chemin2009_rotation_curve.csv", "w", encoding="utf-8", newline="\n") as f:
    f.write(
        "# name: M31 H I rotation curve (Chemin+2009)\n# quantity: circular_velocity\n# unit: km/s\n"
        "# radius_unit: kpc\n# evidence: INFERRED\n" + common +
        "# notes: Tilted-ring fit to the observed 21-cm line-of-sight velocity field (DRAO), radii for D = 785 kpc."
        " For cold gas Vrot ≈ Vc. Rings inside 2.3 kpc have large errors.\nr_kpc,vrot_kms,err_kms\n"
    )
    n = 0
    for c in rows:
        if "nodata" in c[8]:
            continue
        f.write(f"{c[1]},{c[8]},{c[9]}\n")
        n += 1
    print("chemin rc rows", n)
with open(out + "/chemin2009_hi_surface_density.csv", "w", encoding="utf-8", newline="\n") as f:
    f.write(
        "# name: M31 Σ_HI (Chemin+2009)\n# quantity: surface_density\n# unit: Msun/pc2\n"
        "# radius_unit: kpc\n# evidence: INFERRED\n" + common +
        "# notes: Face-on H I surface density averaged over tilted rings, total H I emission,"
        " hydrogen only (no helium correction). No errors are tabulated.\nr_kpc,sigma_hi_msun_pc2\n"
    )
    for c in rows:
        f.write(f"{c[1]},{c[10]}\n")
    print("chemin sigma rows", len(rows))

tex = open(sp + "/0912.4133/ec_m31.tex", encoding="latin-1").read().splitlines()
i = next(k for k, l in enumerate(tex) if BS + "label{rotcur}" in l)
rc = []
for l in tex[i:]:
    if "end{tabular}" in l:
        break
    c = cells(l)
    if len(c) == 8 and re.match(r"^[0-9.]+$", c[0]):
        rc.append(c)
with open(out + "/corbelli2010_rotation_curve.csv", "w", encoding="utf-8", newline="\n") as f:
    f.write(
        "# name: M31 H I rotation curve (Corbelli+2010)\n# quantity: circular_velocity\n# unit: km/s\n"
        "# radius_unit: kpc\n# evidence: INFERRED\n# distance_kpc: 785\n"
        "# source: Corbelli et al. 2010, A&A 511, A89 (Table 1, transcribed from the arXiv source)\n"
        "# url: https://arxiv.org/abs/0912.4133\n"
        "# notes: Tilted-ring model including the warp, 21-cm mosaic at 466 pc resolution, D = 785 kpc.\n"
        "r_kpc,vrot_kms,err_kms\n"
    )
    for c in rc:
        f.write(f"{c[0]},{c[6]},{c[7]}\n")
    print("corbelli rc rows", len(rc))

i = next(k for k, l in enumerate(tex) if BS + "label{largedat}" in l)
mr = []
for l in tex[i:]:
    if "end{tabular}" in l:
        break
    c = [x.strip() for x in l.split("&")]
    if len(c) < 5 or not re.match(r"^[0-9.]+$", c[1]):
        continue
    ref = re.sub(r"\\citet\{([^}]*)\}", r"\1", c[0]).strip()
    m = re.match(r"([0-9.]+)\$\^\{\+([0-9.]+)\}_\{-([0-9.]+)\}\$", c[2].replace(" ", ""))
    obj = re.sub(r"\\footnote.*", "", c[4]).strip().rstrip(BS).strip()
    mr.append((float(c[1]), m.group(1), m.group(3), m.group(2), ref, obj))
mr.sort(key=lambda r: r[0])
with open(out + "/corbelli2010_enclosed_mass.csv", "w", encoding="utf-8", newline="\n") as f:
    f.write(
        "# name: M31 M(<R) from dynamical tracers (compiled by Corbelli+2010)\n# quantity: enclosed_mass\n"
        "# unit: 1e10Msun\n# radius_unit: kpc\n# evidence: INFERRED\n"
        "# source: Corbelli et al. 2010, A&A 511, A89 (Table 3, compilation; original references in the ref column)\n"
        "# url: https://arxiv.org/abs/0912.4133\n"
        "# notes: Dynamical mass estimates from globular clusters, satellites, the stellar stream and 21-cm data;"
        " asymmetric 1σ errors; several estimates share a radius.\n"
        "r_kpc,mass_1e10msun,err_minus,err_plus,ref,tracer\n"
    )
    for r in mr:
        f.write(f"{r[0]:g},{r[1]},{r[2]},{r[3]},{r[4]},{r[5]}\n")
    print("mass rows", len(mr))
