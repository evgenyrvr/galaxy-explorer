package galaxy

// Source is a bibliographic reference used by presets and physics.
type Source struct {
	ID       string `json:"id"`
	Citation string `json:"citation"`
	URL      string `json:"url"`
	UsedFor  string `json:"usedFor"`
}

// Sources is the reference registry. Values taken from these works are
// rounded literature values used to shape normalised educational presets;
// they are not fits performed by this program.
var Sources = map[string]Source{
	"planck2018":      {"planck2018", "Planck Collaboration 2020, A&A 641, A6 (Planck 2018 results VI)", "https://doi.org/10.1051/0004-6361/201833910", "Cosmology: H0, Ωm, Ωb; ρcrit and R200c."},
	"dutton2014":      {"dutton2014", "Dutton & Macciò 2014, MNRAS 441, 3359", "https://doi.org/10.1093/mnras/stu742", "NFW concentration–mass relation c200c(M200c)."},
	"nfw1997":         {"nfw1997", "Navarro, Frenk & White 1997, ApJ 490, 493", "https://doi.org/10.1086/304888", "NFW dark-matter density profile."},
	"hernquist1990":   {"hernquist1990", "Hernquist 1990, ApJ 356, 359", "https://doi.org/10.1086/168845", "Hernquist profile for bulges and ellipticals."},
	"freeman1970":     {"freeman1970", "Freeman 1970, ApJ 160, 811", "https://doi.org/10.1086/150474", "Exponential discs and their rotation curve."},
	"mcmillan2017":    {"mcmillan2017", "McMillan 2017, MNRAS 465, 76", "https://doi.org/10.1093/mnras/stw2759", "Functional form of gas discs with a central hole."},
	"cavaliere1976":   {"cavaliere1976", "Cavaliere & Fusco-Femiano 1976, A&A 49, 137", "https://ui.adsabs.harvard.edu/abs/1976A%26A....49..137C", "β-model for hot gas."},
	"miller2015":      {"miller2015", "Miller & Bregman 2015, ApJ 800, 14", "https://doi.org/10.1088/0004-637X/800/1/14", "Hot-halo β ≈ 0.5 for an L* spiral."},
	"haffner2009":     {"haffner2009", "Haffner et al. 2009, Rev. Mod. Phys. 81, 969", "https://doi.org/10.1103/RevModPhys.81.969", "Warm ionized medium layer, scale height ~1 kpc."},
	"courteau2011":    {"courteau2011", "Courteau et al. 2011, ApJ 739, 20", "https://doi.org/10.1088/0004-637X/739/1/20", "M31 bulge and disc structural parameters (disc Rd ≈ 5.3 kpc, bulge Re ≈ 1 kpc)."},
	"collins2011":     {"collins2011", "Collins et al. 2011, MNRAS 413, 1548", "https://doi.org/10.1111/j.1365-2966.2011.18234.x", "M31 thick disc (scale height ≈ 2.2 kpc, longer scale length than the thin disc)."},
	"tamm2012":        {"tamm2012", "Tamm et al. 2012, A&A 546, A4", "https://doi.org/10.1051/0004-6361/201220065", "M31 stellar mass ≈ 1.0–1.5×10¹¹ M☉ and halo mass ≈ 1×10¹² M☉."},
	"ibata2014":       {"ibata2014", "Ibata et al. 2014, ApJ 780, 128 (PAndAS)", "https://doi.org/10.1088/0004-637X/780/2/128", "M31 stellar halo: power law ρ ∝ r^−3.3, extends beyond 150 kpc."},
	"braun2009":       {"braun2009", "Braun et al. 2009, ApJ 695, 937", "https://doi.org/10.1088/0004-637X/695/2/937", "M31 H I mass ≈ 5×10⁹ M☉ and its ring-like distribution."},
	"nieten2006":      {"nieten2006", "Nieten et al. 2006, A&A 453, 459", "https://doi.org/10.1051/0004-6361:20035672", "M31 molecular gas ≈ 3.6×10⁸ M☉, concentrated in the 10 kpc ring."},
	"draine2014":      {"draine2014", "Draine et al. 2014, ApJ 780, 172", "https://doi.org/10.1088/0004-637X/780/2/172", "M31 dust mass ≈ 5×10⁷ M☉."},
	"bender2005":      {"bender2005", "Bender et al. 2005, ApJ 631, 280", "https://doi.org/10.1086/432434", "M31 central black hole ≈ 1.4×10⁸ M☉."},
	"lehner2020":      {"lehner2020", "Lehner et al. 2020, ApJ 900, 9 (Project AMIGA)", "https://doi.org/10.3847/1538-4357/aba49c", "M31 circumgalactic medium; ionized CGM mass ≳10¹⁰ M☉ (hot phase poorly constrained)."},
	"ibata2001":       {"ibata2001", "Ibata et al. 2001, Nature 412, 49", "https://doi.org/10.1038/35083506", "Discovery of M31's Giant Stellar Stream."},
	"fardal2006":      {"fardal2006", "Fardal et al. 2006, MNRAS 366, 1012", "https://doi.org/10.1111/j.1365-2966.2005.09864.x", "Giant Stellar Stream as debris of a nearly radial orbit."},
	"fardal2015":      {"fardal2015", "Fardal, Huang & Weinberg 2015, MNRAS 452, 301", "https://doi.org/10.1093/mnras/stv1198", "Particle-spray method for mock tidal streams."},
	"mcconnachie2012": {"mcconnachie2012", "McConnachie 2012, AJ 144, 4", "https://doi.org/10.1088/0004-6256/144/1/4", "Satellite masses, sizes and distances (M32, NGC 205)."},
	"moster2013":      {"moster2013", "Moster, Naab & White 2013, MNRAS 428, 3121", "https://doi.org/10.1093/mnras/sts261", "Stellar-to-halo mass relation: M*/M200 ≈ 3% at 10¹² M☉."},
	"kormendy2013":    {"kormendy2013", "Kormendy & Ho 2013, ARA&A 51, 511", "https://doi.org/10.1146/annurev-astro-082708-101811", "Black hole – bulge mass relation (M_BH ≈ 0.5% M_bulge)."},
	"shen2003":        {"shen2003", "Shen et al. 2003, MNRAS 343, 978", "https://doi.org/10.1046/j.1365-8711.2003.06740.x", "Size–mass relation of early-type galaxies."},
	"quinn1984":       {"quinn1984", "Quinn 1984, ApJ 279, 596", "https://doi.org/10.1086/161924", "Shells from phase wrapping of radial-merger debris."},
	"eht2019":         {"eht2019", "Event Horizon Telescope Collaboration 2019, ApJL 875, L1", "https://doi.org/10.3847/2041-8213/ab0ec7", "M87* mass ≈ 6.5×10⁹ M☉ (real M87, not the normalised preset)."},
	"harris2013":      {"harris2013", "Harris, Harris & Alessi 2013, ApJ 772, 82", "https://doi.org/10.1088/0004-637X/772/2/82", "Globular cluster system sizes; M87 hosts ~13 000 GCs."},
	"biretta1999":     {"biretta1999", "Biretta, Sparks & Macchetto 1999, ApJ 520, 621", "https://doi.org/10.1086/307499", "M87 optical jet (~1.5 kpc, knots, superluminal motion)."},
	"churazov2008":    {"churazov2008", "Churazov et al. 2008, MNRAS 388, 1062", "https://doi.org/10.1111/j.1365-2966.2008.13507.x", "M87 hot atmosphere and its mass profile."},
	"sparks2004":      {"sparks2004", "Sparks et al. 2004, ApJ 607, 294", "https://doi.org/10.1086/383608", "Hα filaments in M87."},
	"chemin2009":      {"chemin2009", "Chemin, Carignan & Foster 2009, ApJ 705, 1395", "https://arxiv.org/abs/0909.3846", "M31 H I rotation curve and Σ_HI (Table 4), H I mass 4.23×10⁹ M☉, stellar masses from R-band M/L."},
	"corbelli2010":    {"corbelli2010", "Corbelli et al. 2010, A&A 511, A89", "https://arxiv.org/abs/0912.4133", "M31 21-cm rotation curve with warp, ΛCDM NFW halo (C = 12, 1.2×10¹² M☉ for Δ = 98), compiled dynamical M(<R)."},
	"dame1993":        {"dame1993", "Dame et al. 1993, ApJ 418, 730", "https://doi.org/10.1086/173431", "M31 CO survey; molecular mass ≈ 3.5×10⁸ M☉ at 785 kpc (as adopted by Chemin+2009)."},
	"ford1994":        {"ford1994", "Ford et al. 1994, ApJL 435, L27", "https://doi.org/10.1086/187586", "Ionized gas/dust disc of ~100 pc around M87's nucleus."},
}
