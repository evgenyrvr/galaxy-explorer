package presets

import (
	"math"

	"galaxy/internal/budget"
	"galaxy/internal/galaxy"
	"galaxy/internal/ontology"
	"galaxy/internal/physics"
)

// M31 is the spiral preset inspired by the Andromeda galaxy.
type M31 struct{ c cache }

// NewM31 returns the preset.
func NewM31() *M31 { return &M31{} }

func (*M31) Info() galaxy.PresetInfo {
	return galaxy.PresetInfo{
		ID:          "m31-normalized",
		Name:        "Spiral · M31-inspired",
		Morphology:  "spiral",
		Kind:        "normalized",
		Version:     "0.1.0",
		Description: "Normalised educational spiral. Structure, scale lengths and default mass fractions follow rounded literature values for M31 (Andromeda).",
		Disclaimer:  "Synthetic 3D reconstruction. M31 is a morphological guide; masses follow the chosen M_total, and 3D positions are model-based, not observed.",
	}
}

// Default fractions of M200c, rounded from the literature for M31 with
// M200c ≈ 10¹² M☉ (Tamm et al. 2012). Hot CGM is the least constrained entry.
const (
	m31FracBH    = 1.4e-4 // 1.4×10⁸ M☉ (Bender et al. 2005)
	m31FracStars = 0.10   // 1.0×10¹¹ M☉ (Tamm et al. 2012)
	m31FracHI    = 5.2e-3 // 5.2×10⁹ M☉ (Braun et al. 2009)
	m31FracH2    = 3.6e-4 // 3.6×10⁸ M☉ (Nieten et al. 2006)
	m31FracDust  = 5.4e-5 // 5.4×10⁷ M☉ (Draine et al. 2014)
	m31FracWIM   = 1.0e-3 // 10⁹ M☉, order-of-magnitude
	m31FracHot   = 0.04   // 4×10¹⁰ M☉, CGM (Lehner et al. 2020: ≳10¹⁰ ionized), very uncertain
)

func (*M31) DefaultBudget() budget.Budget {
	dm := remainder(m31FracBH, m31FracStars, m31FracHI, m31FracH2, m31FracDust, m31FracWIM, m31FracHot)
	cats := []budget.Category{
		{ID: string(ontology.DarkMatter), Fraction: dm},
		{ID: string(ontology.Stars), Fraction: m31FracStars},
		{ID: string(ontology.GasHI), Fraction: m31FracHI},
		{ID: string(ontology.GasH2), Fraction: m31FracH2},
		{ID: string(ontology.GasWarmIonized), Fraction: m31FracWIM},
		{ID: string(ontology.GasHotIonized), Fraction: m31FracHot},
		{ID: string(ontology.Dust), Fraction: m31FracDust},
		{ID: string(ontology.BlackHole), Fraction: m31FracBH},
	}
	s := string(ontology.Stars)
	shares := []shareDef{
		{s, "stars.thin_disk", 0.4653},
		{s, "stars.thick_disk", 0.13},
		{s, "stars.bulge", 0.30},
		{s, "stars.halo", 0.08},
		{s, "stars.nucleus", 0.0003},
		{s, "stars.globular_clusters", 0.0014},
		{s, "stars.giant_stream", 0.003},
		{s, "stars.m32", 0.01},
		{s, "stars.ngc205", 0.01},
		{string(ontology.GasHI), "gas.hi.disk", 1},
		{string(ontology.GasH2), "gas.h2.disk", 1},
		{string(ontology.GasWarmIonized), "gas.wim.layer", 1},
		{string(ontology.GasHotIonized), "gas.hot.cgm", 1},
		{string(ontology.Dust), "dust.disk", 1},
		{string(ontology.DarkMatter), "dm.halo", 1},
		{string(ontology.BlackHole), "bh.central", 1},
	}
	return makeBudget(1e12, cats, shares)
}

func (*M31) Tags() []ontology.Tag {
	return []ontology.Tag{
		{ID: "arm_1", Label: "Spiral arm 1", Family: "arms", Note: "Density enhancement of existing stars, gas and dust; adds no mass."},
		{ID: "arm_2", Label: "Spiral arm 2", Family: "arms", Note: "Density enhancement of existing stars, gas and dust; adds no mass."},
		{ID: "satellites", Label: "Satellites (all)", Family: "satellites", Note: "Inside R200c, so already part of M200c."},
		{ID: "satellite_m32", Label: "M32", Family: "satellites"},
		{ID: "satellite_ngc205", Label: "NGC 205 (M110)", Family: "satellites"},
		{ID: "tidal_features", Label: "Tidal features", Family: "tidal", Note: "Merger debris."},
		{ID: "disc_region", Label: "Galactic disc (all phases)", Family: "regions", Note: "Thin + thick stellar discs and all disc gas/dust."},
	}
}

// m31Arms is shared by all disc phases so arm tags line up between stars and gas.
func m31Arms(amplitude float64) *physics.ArmPattern {
	return &physics.ArmPattern{M: 2, PitchDeg: 10, Amplitude: amplitude, HalfWidth: 0.55, Rin: 4, Rout: 24, R0: 4, Phase0Deg: 0}
}

func (p *M31) Build(env galaxy.Env) []*galaxy.Component {
	return p.components(env, p.stream(env))
}

// components builds all components; gss == nil omits the stream (used to
// build the potential the stream itself is integrated in).
func (p *M31) components(env galaxy.Env, gss physics.Distribution) []*galaxy.Component {
	r200 := env.R200c
	c200 := env.Cosmology.ConcentrationDM14(env.TotalMass)
	disc := []string{"disc_region"}

	thin := p.c.get("thin", func() physics.Distribution {
		return physics.NewDisc(physics.DiscOptions{Shape: physics.Exponential, Rd: 5.3, Z0: 0.7, RMax: math.Min(32, r200), Arms: m31Arms(0.45)})
	})
	thick := p.c.get("thick", func() physics.Distribution {
		return physics.NewDisc(physics.DiscOptions{Shape: physics.Exponential, Rd: 7.0, Z0: 4.4, RMax: math.Min(40, r200)})
	})
	bulge := p.c.get("bulge", func() physics.Distribution {
		return physics.NewSpheroid(physics.Hernquist(0.55), 20, physics.SpheroidOptions{P: 1, Q: 0.8})
	})
	haloT := math.Min(150, r200)
	halo := p.c.get(r200Key("halo", haloT), func() physics.Distribution {
		return physics.NewSpheroid(physics.CoredPowerLaw(3, 3.3), haloT, physics.SpheroidOptions{})
	})
	nucleus := p.c.get("nucleus", func() physics.Distribution {
		return physics.NewSpheroid(physics.Plummer(0.004), 0.05, physics.SpheroidOptions{})
	})
	gcs := p.c.get("gcs", func() physics.Distribution {
		base := physics.NewSpheroid(physics.CoredPowerLaw(2, 3.5), 100, physics.SpheroidOptions{})
		return discreteFrom("globular_cluster_system", base, 450, env.Seed+31, map[string]float64{"mean_mass_note_msun": 3e5})
	})
	m32 := p.c.get("m32", func() physics.Distribution {
		return physics.NewOffset(physics.NewSpheroid(physics.Plummer(0.1), 1.5, physics.SpheroidOptions{P: 0.9, Q: 0.8}), physics.Vec3{-3.9, 3.2, -3.0}, env.Seed+32)
	})
	n205 := p.c.get("ngc205", func() physics.Distribution {
		return physics.NewOffset(physics.NewSpheroid(physics.Plummer(0.5), 4, physics.SpheroidOptions{P: 0.8, Q: 0.55}), physics.Vec3{5.5, -4.5, 38}, env.Seed+33)
	})
	hi := p.c.get("hi", func() physics.Distribution {
		return physics.NewDisc(physics.DiscOptions{Shape: physics.ExponentialHole, Rd: 8, Rm: 18, Z0: 0.3, RFlare: 15, RMax: math.Min(40, r200), Arms: m31Arms(0.5)})
	})
	h2 := p.c.get("h2", func() physics.Distribution {
		return physics.NewDisc(physics.DiscOptions{Shape: physics.ExponentialHole, Rd: 4, Rm: 25, Z0: 0.1, RMax: math.Min(20, r200), Arms: m31Arms(0.7)})
	})
	wim := p.c.get("wim", func() physics.Distribution {
		return physics.NewDisc(physics.DiscOptions{Shape: physics.Exponential, Rd: 7, Z0: 2.0, RMax: math.Min(30, r200), Arms: m31Arms(0.3)})
	})
	dust := p.c.get("dust", func() physics.Distribution {
		return physics.NewDisc(physics.DiscOptions{Shape: physics.ExponentialHole, Rd: 5, Rm: 24, Z0: 0.2, RMax: math.Min(25, r200), Arms: m31Arms(0.6)})
	})
	hot := p.c.get(r200Key("hot", r200), func() physics.Distribution {
		return physics.NewSpheroid(physics.BetaModel(5, 0.5), r200, physics.SpheroidOptions{})
	})
	dm := p.c.get(r200Key("dm", r200), func() physics.Distribution {
		return physics.NewSpheroid(physics.NFW(r200/c200, c200), r200, physics.SpheroidOptions{})
	})

	comps := []*galaxy.Component{
		{ID: "stars.thin_disk", Name: "Thin stellar disc", Matter: ontology.Stars, Structure: "thin_disk", Tags: disc, Evidence: ontology.Reconstructed,
			Sources: []string{"courteau2011", "freeman1970"}, Dist: thin, GeomKey: "thin",
			Notes:  "Exponential disc, Rd = 5.3 kpc (Courteau et al. 2011); sech² scale height z0 = 0.7 kpc (h ≈ 0.35 kpc). Two-armed logarithmic spiral, pitch 10°, as azimuthal density modulation.",
			Visual: galaxy.Visual{Style: "stars", Color: "#cfe0ff", Size: 0.18, Opacity: 0.5, Markers: 90000}},
		{ID: "stars.thick_disk", Name: "Thick stellar disc", Matter: ontology.Stars, Structure: "thick_disk", Tags: disc, Evidence: ontology.Reconstructed,
			Sources: []string{"collins2011"}, Dist: thick, GeomKey: "thick",
			Notes:  "Exponential disc, Rd = 7 kpc, exponential scale height h ≈ 2.2 kpc (z0 = 2h) after Collins et al. (2011).",
			Visual: galaxy.Visual{Style: "stars", Color: "#ffe6c4", Size: 0.22, Opacity: 0.6, Markers: 30000}},
		{ID: "stars.bulge", Name: "Bulge", Matter: ontology.Stars, Structure: "bulge", Evidence: ontology.Reconstructed,
			Sources: []string{"courteau2011", "hernquist1990"}, Dist: bulge, GeomKey: "bulge",
			Notes:  "Hernquist profile a = 0.55 kpc (Re ≈ 1 kpc), flattening q = 0.8.",
			Visual: galaxy.Visual{Style: "stars", Color: "#ffd59a", Size: 0.12, Opacity: 0.55, Markers: 40000}},
		{ID: "stars.halo", Name: "Stellar halo", Matter: ontology.Stars, Structure: "stellar_halo", Evidence: ontology.Reconstructed,
			Sources: []string{"ibata2014"}, Dist: halo, GeomKey: r200Key("halo", haloT),
			Notes:  "Cored power law ρ ∝ (r² + 3²)^(−3.3/2) to 150 kpc (PAndAS, Ibata et al. 2014).",
			Visual: galaxy.Visual{Style: "stars", Color: "#e9d8ff", Size: 0.6, Opacity: 0.35, Markers: 15000}},
		{ID: "stars.nucleus", Name: "Nuclear star cluster", Matter: ontology.Stars, Structure: "nucleus", Evidence: ontology.Reconstructed,
			Sources: []string{"bender2005"}, Dist: nucleus, GeomKey: "nucleus",
			Notes:  "Plummer sphere a = 4 pc. M31's real nucleus is an eccentric double (P1/P2) disc, not modelled here.",
			Visual: galaxy.Visual{Style: "stars", Color: "#fff3d6", Size: 0.002, Opacity: 1, Markers: 3000}},
		{ID: "stars.globular_clusters", Name: "Globular clusters", Matter: ontology.Stars, Structure: "globular_clusters", Evidence: ontology.Illustrative,
			Sources: []string{"harris2013"}, Dist: gcs, GeomKey: "gcs",
			Notes:  "450 clusters (≈ M31's confirmed GC count), mean ≈ 3×10⁵ M☉; positions drawn from ρ ∝ r^−3.5. Each marker is one cluster.",
			Visual: galaxy.Visual{Style: "clusters", Color: "#fff1a8", Size: 0.35, Opacity: 1, Markers: 450}},
		{ID: "stars.giant_stream", Name: "Giant Stellar Stream", Matter: ontology.Stars, Structure: "stream_1", Tags: []string{"tidal_features"}, Evidence: ontology.Illustrative,
			Sources: []string{"ibata2001", "fardal2006", "fardal2015"}, Dist: gss, GeomKey: "gss",
			Notes:  "Tidal debris of a ~2×10⁹ M☉ progenitor on a nearly radial orbit, integrated in this preset's own (default) mass model with a particle-spray method. Shape is illustrative, not fitted to PAndAS.",
			Visual: galaxy.Visual{Style: "stars", Color: "#ffb38a", Size: 0.5, Opacity: 0.55, Markers: 12000}},
		{ID: "stars.m32", Name: "M32 (satellite)", Matter: ontology.Stars, Structure: "satellite_m32", Tags: []string{"satellites", "satellite_m32"}, Evidence: ontology.Reconstructed,
			Sources: []string{"mcconnachie2012"}, Dist: m32, GeomKey: "m32",
			Notes:  "Compact elliptical, Plummer a = 0.1 kpc. The 3D offset (~6 kpc) is a reconstruction: only its sky position is observed.",
			Visual: galaxy.Visual{Style: "stars", Color: "#ffe2b0", Size: 0.04, Opacity: 0.9, Markers: 5000}},
		{ID: "stars.ngc205", Name: "NGC 205 (satellite)", Matter: ontology.Stars, Structure: "satellite_ngc205", Tags: []string{"satellites", "satellite_ngc205"}, Evidence: ontology.Reconstructed,
			Sources: []string{"mcconnachie2012"}, Dist: n205, GeomKey: "ngc205",
			Notes:  "Dwarf elliptical, Plummer a = 0.5 kpc. Placed ~40 kpc from the centre, consistent with its larger distance than M31; exact 3D position uncertain.",
			Visual: galaxy.Visual{Style: "stars", Color: "#ffe2b0", Size: 0.12, Opacity: 0.8, Markers: 6000}},
		{ID: "gas.hi.disk", Name: "H I disc", Matter: ontology.GasHI, Structure: "gas_disc", Tags: disc, Evidence: ontology.Reconstructed,
			Sources: []string{"braun2009", "mcmillan2017"}, Dist: hi, GeomKey: "hi",
			Notes:  "Σ ∝ exp(−18/R − R/8), peaking near 12 kpc like M31's H I ring; flaring scale height.",
			Visual: galaxy.Visual{Style: "gas", Color: "#6fb4ff", Size: 1.4, Opacity: 0.12, Markers: 25000}},
		{ID: "gas.h2.disk", Name: "Molecular gas (H₂)", Matter: ontology.GasH2, Structure: "gas_disc", Tags: disc, Evidence: ontology.Reconstructed,
			Sources: []string{"nieten2006", "mcmillan2017"}, Dist: h2, GeomKey: "h2",
			Notes:  "Σ ∝ exp(−25/R − R/4), concentrated in the 10 kpc ring; strongly arm-modulated.",
			Visual: galaxy.Visual{Style: "gas", Color: "#ff7fa8", Size: 0.6, Opacity: 0.18, Markers: 12000}},
		{ID: "gas.wim.layer", Name: "Warm ionized layer", Matter: ontology.GasWarmIonized, Structure: "wim_layer", Tags: disc, Evidence: ontology.Illustrative,
			Sources: []string{"haffner2009"}, Dist: wim, GeomKey: "wim",
			Notes:  "Diffuse ionized ('Reynolds') layer, h ≈ 1 kpc. Mass is an order-of-magnitude estimate.",
			Visual: galaxy.Visual{Style: "gas", Color: "#ff6b5c", Size: 1.6, Opacity: 0.06, Markers: 10000}},
		{ID: "gas.hot.cgm", Name: "Hot corona / CGM", Matter: ontology.GasHotIonized, Structure: "cgm", Evidence: ontology.Reconstructed,
			Sources: []string{"lehner2020", "miller2015", "cavaliere1976"}, Dist: hot, GeomKey: r200Key("hot", r200),
			Notes:  "β-model (β = 0.5, rc = 5 kpc) truncated at R200c. The hot CGM mass of M31 is poorly constrained.",
			Visual: galaxy.Visual{Style: "gas", Color: "#ffb347", Size: 18, Opacity: 0.035, Markers: 8000}},
		{ID: "dust.disk", Name: "Dust disc", Matter: ontology.Dust, Structure: "dust_disc", Tags: disc, Evidence: ontology.Reconstructed,
			Sources: []string{"draine2014"}, Dist: dust, GeomKey: "dust",
			Notes:  "Follows the gas ring (peak ≈ 11 kpc). Counted as its own category, not inside the gas budget.",
			Visual: galaxy.Visual{Style: "dust", Color: "#5a3a22", Size: 1.2, Opacity: 0.22, Markers: 14000}},
		{ID: "dm.halo", Name: "Dark matter halo", Matter: ontology.DarkMatter, Structure: "dark_halo", Evidence: ontology.Reconstructed,
			Sources: []string{"nfw1997", "dutton2014", "tamm2012"}, Dist: dm, GeomKey: r200Key("dm", r200),
			Notes:  "NFW profile truncated at R200c, concentration from Dutton & Macciò (2014). Gravitationally inferred distribution, not optical emission.",
			Visual: galaxy.Visual{Style: "dm", Color: "#8f7dff", Size: 22, Opacity: 0.03, Markers: 12000}},
		{ID: "bh.central", Name: "Central black hole", Matter: ontology.BlackHole, Structure: "nucleus", Evidence: ontology.Reconstructed,
			Sources: []string{"bender2005"}, Dist: physics.PointMass{}, GeomKey: "bh",
			Notes:  "Point mass. Drawn as a schematic icon; its event horizon (~4×10⁻⁶ pc) is not to scale.",
			Visual: galaxy.Visual{Style: "icon", Color: "#ffffff", Size: 1, Opacity: 1, Markers: 1}},
	}
	return withoutNil(comps)
}

// stream integrates the Giant Stellar Stream in the preset's default mass
// model. Geometry is fixed: it does not change when the user edits masses.
func (p *M31) stream(env galaxy.Env) physics.Distribution {
	return p.c.get("gss", func() physics.Distribution {
		def := p.DefaultBudget()
		r200 := env.Cosmology.R200c(def.TotalMass)
		base := p.components(galaxy.Env{TotalMass: def.TotalMass, R200c: r200, Cosmology: env.Cosmology, Seed: env.Seed}, nil)
		menc := galaxy.EnclosedFrom(defaultMasses(def, base), base, r200)
		pts := physics.ParticleSpray(menc, physics.SprayOptions{
			Start:    physics.Phase{X: physics.Vec3{-40, 30, 55}, V: physics.Vec3{12, -8, 10}},
			Duration: 1.4, ReleaseStart: 0.35, ReleaseEnd: 0.75,
			ProgMass: 2e9, SigmaV: 12, N: 12000, Seed: env.Seed + 77, Soft: 0.3, RMax: 0.8 * r200,
		})
		return physics.NewSampled("particle_spray_stream", map[string]float64{"progenitor_mass_msun": 2e9, "duration_gyr": 1.4}, pts, nil, nil, false)
	})
}
