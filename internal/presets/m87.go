package presets

import (
	"math"
	"math/rand"

	"galaxy/internal/budget"
	"galaxy/internal/galaxy"
	"galaxy/internal/ontology"
	"galaxy/internal/physics"
)

// M87 is the elliptical preset inspired by M87 (Virgo A).
type M87 struct{ c cache }

// NewM87 returns the preset.
func NewM87() *M87 { return &M87{} }

func (*M87) Info() galaxy.PresetInfo {
	return galaxy.PresetInfo{
		ID:          "m87-normalized",
		Name:        "Elliptical · M87-inspired",
		Morphology:  "elliptical",
		Kind:        "normalized",
		Version:     "0.1.0",
		Description: "Normalised educational elliptical with M87's components: spheroid, hot atmosphere, AGN jet and radio lobes, globular clusters and merger shells. Masses and sizes follow scaling relations for the chosen M_total.",
		Disclaimer:  "Synthetic 3D reconstruction. The real M87 is far more massive (halo ≳10¹³ M☉, black hole 6.5×10⁹ M☉) than this 10¹² M☉ educational galaxy.",
	}
}

// Default fractions of M200c for a 10¹² M☉ elliptical from scaling relations
// (the real M87 is ~10× more massive; see Info().Disclaimer).
const (
	m87FracStars = 0.03   // M*/M200 ≈ 3% at 10¹² M☉ (Moster et al. 2013)
	m87FracBH    = 1.5e-4 // M_BH ≈ 0.5% M* (Kormendy & Ho 2013)
	m87FracHot   = 0.04   // hot atmosphere, uncertain
	m87FracWarm  = 1e-5   // Hα filaments (Sparks et al. 2004), order of magnitude
	m87FracDust  = 1e-6   // nuclear dust disc, order of magnitude
)

// Effective radius from the early-type size–mass relation of Shen et al.
// (2003): R50 = 2.88×10⁻⁶ (M*/M☉)^0.56 kpc.
func m87Re() float64 {
	return 2.88e-6 * math.Pow(m87FracStars*1e12, 0.56)
}

func (*M87) DefaultBudget() budget.Budget {
	dm := remainder(m87FracStars, m87FracBH, m87FracHot, m87FracWarm, m87FracDust)
	cats := []budget.Category{
		{ID: string(ontology.DarkMatter), Fraction: dm},
		{ID: string(ontology.Stars), Fraction: m87FracStars},
		{ID: string(ontology.GasWarmIonized), Fraction: m87FracWarm},
		{ID: string(ontology.GasHotIonized), Fraction: m87FracHot},
		{ID: string(ontology.Dust), Fraction: m87FracDust},
		{ID: string(ontology.BlackHole), Fraction: m87FracBH},
		// Cold gas is absent in this preset (categories kept at 0%).
		{ID: string(ontology.GasHI), Fraction: 0},
		{ID: string(ontology.GasH2), Fraction: 0},
	}
	s := string(ontology.Stars)
	h := string(ontology.GasHotIonized)
	shares := []shareDef{
		{s, "stars.spheroid", 0.973},
		{s, "stars.globular_clusters", 0.005},
		{s, "stars.shells", 0.015},
		{s, "stars.dwarf", 0.007},
		{h, "gas.hot.atmosphere", 1},
		{h, "agn.jet", 0},
		{h, "agn.lobes", 0},
		{string(ontology.GasWarmIonized), "gas.warm.filaments", 1},
		{string(ontology.Dust), "dust.nuclear_disc", 1},
		{string(ontology.DarkMatter), "dm.halo", 1},
		{string(ontology.BlackHole), "bh.central", 1},
	}
	b := makeBudget(1e12, cats, shares)
	for i := range b.Shares[h] {
		if b.Shares[h][i].Component != "gas.hot.atmosphere" {
			b.Shares[h][i].Locked = true // negligible mass, kept at 0
		}
	}
	return b
}

func (*M87) Tags() []ontology.Tag {
	return []ontology.Tag{
		{ID: "inner_spheroid", Label: "Inner spheroid (m < Re)", Family: "regions", Note: "Ellipsoidal radius below the effective radius."},
		{ID: "outer_envelope", Label: "Outer envelope (m ≥ Re)", Family: "regions"},
		{ID: "tidal_features", Label: "Tidal shells", Family: "tidal", Note: "Phase-wrapped debris of a radial minor merger (Quinn 1984)."},
		{ID: "satellites", Label: "Satellites (all)", Family: "satellites", Note: "Inside R200c, so already part of M200c."},
		{ID: "agn", Label: "Active nucleus (BH, jet, lobes)", Family: "agn"},
	}
}

// jetAxis is the jet direction in the galaxy frame (unit vector).
var jetAxis = physics.Vec3{0.82, 0.35, 0.45}

func (p *M87) Build(env galaxy.Env) []*galaxy.Component {
	return p.components(env, p.shells(env))
}

func (p *M87) components(env galaxy.Env, shells physics.Distribution) []*galaxy.Component {
	r200 := env.R200c
	c200 := env.Cosmology.ConcentrationDM14(env.TotalMass)
	re := m87Re()
	a := re / 1.8153

	spheroid := p.c.get("spheroid", func() physics.Distribution {
		return physics.NewSpheroid(physics.Hernquist(a), 30*re, physics.SpheroidOptions{
			P: 0.95, Q: 0.88, Splits: []float64{re}, SplitTags: []string{"inner_spheroid", "outer_envelope"},
		})
	})
	nGC := int(math.Round(m87FracStars * 1e12 * 0.005 / 2e5))
	gcs := p.c.get("gcs", func() physics.Distribution {
		base := physics.NewSpheroid(physics.CoredPowerLaw(1.5, 3.0), 60, physics.SpheroidOptions{})
		return discreteFrom("globular_cluster_system", base, nGC, env.Seed+87, map[string]float64{"mean_mass_note_msun": 2e5})
	})
	dwarf := p.c.get("dwarf", func() physics.Distribution {
		return physics.NewOffset(physics.NewSpheroid(physics.Plummer(0.6), 4, physics.SpheroidOptions{P: 0.85, Q: 0.7}), physics.Vec3{25, -12, 8}, env.Seed+88)
	})
	hot := p.c.get(r200Key("hot", r200), func() physics.Distribution {
		return physics.NewSpheroid(physics.BetaModel(2, 0.5), r200, physics.SpheroidOptions{})
	})
	dm := p.c.get(r200Key("dm", r200), func() physics.Distribution {
		return physics.NewSpheroid(physics.NFW(r200/c200, c200), r200, physics.SpheroidOptions{})
	})
	jet := p.c.get("jet", func() physics.Distribution {
		knots := []float64{0.08, 0.25, 0.55, 0.75, 1.0, 1.1, 1.3} // HST-1, D, E/F, I, A, B, C (approx. projected kpc)
		pts := jetSample(jetAxis, 1.5, 2.5, knots, 4000, env.Seed+89)
		return physics.NewSampled("conical_jet", map[string]float64{"length_kpc": 1.5, "half_opening_deg": 2.5}, pts, nil, nil, false)
	})
	lobes := p.c.get("lobes", func() physics.Distribution {
		rng := rand.New(rand.NewSource(env.Seed + 90))
		ax := jetAxis.Unit()
		pts := make([]physics.Vec3, 0, 6000)
		for i := 0; i < 6000; i++ {
			side := 1.0
			if i%2 == 1 {
				side = -1
			}
			c := ax.Scale(side * 2.2)
			r := 0.7 * math.Cbrt(rng.Float64())
			d := physics.Vec3{rng.NormFloat64(), rng.NormFloat64(), rng.NormFloat64()}.Unit()
			pts = append(pts, c.Add(d.Scale(r)))
		}
		return physics.NewSampled("radio_lobes", map[string]float64{"lobe_distance_kpc": 2.2, "lobe_radius_kpc": 0.7}, pts, nil, nil, false)
	})
	fil := p.c.get("filaments", func() physics.Distribution {
		return physics.NewSampled("filaments", map[string]float64{"filaments": 9, "extent_kpc": 8}, filamentSample(9, 0.5, 8, 0.08, 8000, env.Seed+91), nil, nil, false)
	})
	tilt := physics.EulerZXZ(20, 65, 0)
	dust := p.c.get("dust", func() physics.Distribution {
		return physics.NewDisc(physics.DiscOptions{Shape: physics.Exponential, Rd: 0.03, Z0: 0.004, RMax: 0.15, Rotation: &tilt})
	})
	agn := []string{"agn"}

	comps := []*galaxy.Component{
		{ID: "stars.spheroid", Name: "Stellar spheroid", Matter: ontology.Stars, Structure: "spheroid", Evidence: ontology.Reconstructed,
			Sources: []string{"hernquist1990", "shen2003", "moster2013"}, Dist: spheroid, GeomKey: "spheroid",
			Notes:  "Triaxial Hernquist profile, Re ≈ 2.4 kpc from the Shen et al. (2003) size–mass relation, axis ratios 1 : 0.95 : 0.88. Tagged into inner spheroid and outer envelope at Re.",
			Visual: galaxy.Visual{Style: "stars", Color: "#ffe1b8", Size: 0.12, Opacity: 0.85, Markers: 140000}},
		{ID: "stars.globular_clusters", Name: "Globular clusters", Matter: ontology.Stars, Structure: "globular_clusters", Evidence: ontology.Illustrative,
			Sources: []string{"harris2013"}, Dist: gcs, GeomKey: "gcs",
			Notes:  "Rich GC system, more extended than the starlight (ρ ∝ r^−3). Number scaled from M87's ~13 000 clusters to this stellar mass. Each marker is one cluster.",
			Visual: galaxy.Visual{Style: "clusters", Color: "#fff1a8", Size: 0.3, Opacity: 1, Markers: nGC}},
		{ID: "stars.shells", Name: "Tidal shells", Matter: ontology.Stars, Structure: "shells", Tags: []string{"tidal_features"}, Evidence: ontology.Illustrative,
			Sources: []string{"quinn1984", "fardal2015"}, Dist: shells, GeomKey: "shells",
			Notes:  "Debris of a 5×10⁸ M☉ satellite on a nearly radial orbit, integrated for 3 Gyr in this preset's default mass model. Shells form by phase wrapping.",
			Visual: galaxy.Visual{Style: "stars", Color: "#ffc89a", Size: 0.35, Opacity: 0.5, Markers: 15000}},
		{ID: "stars.dwarf", Name: "Dwarf satellite", Matter: ontology.Stars, Structure: "satellite_de1", Tags: []string{"satellites"}, Evidence: ontology.Illustrative,
			Dist: dwarf, GeomKey: "dwarf",
			Notes:  "Illustrative dwarf elliptical, Plummer a = 0.6 kpc, ~29 kpc from the centre.",
			Visual: galaxy.Visual{Style: "stars", Color: "#ffe2b0", Size: 0.15, Opacity: 0.8, Markers: 5000}},
		{ID: "gas.hot.atmosphere", Name: "Hot X-ray atmosphere", Matter: ontology.GasHotIonized, Structure: "hot_atmosphere", Evidence: ontology.Reconstructed,
			Sources: []string{"churazov2008", "cavaliere1976"}, Dist: hot, GeomKey: r200Key("hot", r200),
			Notes:  "β-model (β = 0.5, rc = 2 kpc) truncated at R200c.",
			Visual: galaxy.Visual{Style: "gas", Color: "#ffb347", Size: 14, Opacity: 0.04, Markers: 12000}},
		{ID: "agn.jet", Name: "Relativistic jet", Matter: ontology.GasHotIonized, Structure: "jet", Tags: agn, Evidence: ontology.Reconstructed,
			Sources: []string{"biretta1999"}, Dist: jet, GeomKey: "jet",
			Notes:  "~1.5 kpc conical jet with knots (HST-1, D, E, F, I, A, B, C). Its mass is negligible (≪10⁻⁶ of the hot gas) and kept at 0; shown for structure. The counter-jet is Doppler-dimmed and not drawn.",
			Visual: galaxy.Visual{Style: "jet", Color: "#8fd8ff", Size: 0.05, Opacity: 0.9, Markers: 4000}},
		{ID: "agn.lobes", Name: "Inner radio lobes", Matter: ontology.GasHotIonized, Structure: "radio_lobes", Tags: agn, Evidence: ontology.Illustrative,
			Dist: lobes, GeomKey: "lobes",
			Notes:  "Schematic radio lobes (relativistic plasma); mass negligible, kept at 0.",
			Visual: galaxy.Visual{Style: "gas", Color: "#c58bff", Size: 0.5, Opacity: 0.1, Markers: 6000}},
		{ID: "gas.warm.filaments", Name: "Hα filaments", Matter: ontology.GasWarmIonized, Structure: "filaments", Evidence: ontology.Illustrative,
			Sources: []string{"sparks2004"}, Dist: fil, GeomKey: "filaments",
			Notes:  "Illustrative warm ionized filaments within ~8 kpc; morphology inspired by M87's Hα filaments.",
			Visual: galaxy.Visual{Style: "gas", Color: "#ff6b5c", Size: 0.25, Opacity: 0.25, Markers: 8000}},
		{ID: "dust.nuclear_disc", Name: "Nuclear dust disc", Matter: ontology.Dust, Structure: "nuclear_disc", Evidence: ontology.Reconstructed,
			Sources: []string{"ford1994"}, Dist: dust, GeomKey: "dust",
			Notes:  "~100 pc tilted gas/dust disc around the nucleus.",
			Visual: galaxy.Visual{Style: "dust", Color: "#6b4426", Size: 0.01, Opacity: 0.6, Markers: 3000}},
		{ID: "dm.halo", Name: "Dark matter halo", Matter: ontology.DarkMatter, Structure: "dark_halo", Evidence: ontology.Reconstructed,
			Sources: []string{"nfw1997", "dutton2014"}, Dist: dm, GeomKey: r200Key("dm", r200),
			Notes:  "NFW profile truncated at R200c with the Dutton & Macciò (2014) concentration. Gravitationally inferred, not optical emission.",
			Visual: galaxy.Visual{Style: "dm", Color: "#8f7dff", Size: 22, Opacity: 0.03, Markers: 12000}},
		{ID: "bh.central", Name: "Central black hole", Matter: ontology.BlackHole, Structure: "nucleus", Tags: agn, Evidence: ontology.Reconstructed,
			Sources: []string{"kormendy2013", "eht2019"}, Dist: physics.PointMass{}, GeomKey: "bh",
			Notes:  "Point mass from the M_BH–M_bulge relation for this preset. The real M87* is 6.5×10⁹ M☉ (EHT 2019). Schematic icon, not to scale.",
			Visual: galaxy.Visual{Style: "icon", Color: "#3fe0c5", Size: 1, Opacity: 1, Markers: 1}},
	}
	return withoutNil(comps)
}

// shells integrates radial-merger debris in the default mass model.
func (p *M87) shells(env galaxy.Env) physics.Distribution {
	return p.c.get("shells", func() physics.Distribution {
		def := p.DefaultBudget()
		r200 := env.Cosmology.R200c(def.TotalMass)
		base := p.components(galaxy.Env{TotalMass: def.TotalMass, R200c: r200, Cosmology: env.Cosmology, Seed: env.Seed}, nil)
		menc := galaxy.EnclosedFrom(defaultMasses(def, base), base, r200)
		pts := physics.ParticleSpray(menc, physics.SprayOptions{
			Start:    physics.Phase{X: physics.Vec3{45, 10, 20}, V: physics.Vec3{0, 6, -4}},
			Duration: 3.0, ReleaseStart: 0.12, ReleaseEnd: 0.45,
			ProgMass: 5e8, SigmaV: 8, N: 15000, Seed: env.Seed + 92, Soft: 0.2, RMax: 0.8 * r200,
		})
		return physics.NewSampled("particle_spray_shells", map[string]float64{"progenitor_mass_msun": 5e8, "duration_gyr": 3}, pts, nil, nil, false)
	})
}
