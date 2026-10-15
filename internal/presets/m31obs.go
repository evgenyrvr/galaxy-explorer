package presets

import (
	"embed"
	"math"
	"strconv"

	"galaxy/internal/budget"
	"galaxy/internal/galaxy"
	"galaxy/internal/obsdata"
	"galaxy/internal/ontology"
	"galaxy/internal/physics"
	"galaxy/internal/units"
)

//go:embed data/m31/*.csv
var m31Data embed.FS

// M31Obs is the observational M31 preset: masses and radial profiles taken
// from published analyses, with their tables bundled for comparison.
//
// Mass model (all values M☉):
//
//	stellar disc  7.1×10¹⁰   Chemin+2009, R-band, M/L_d = 1.7 (SSP)
//	bulge         2.4×10¹⁰   Chemin+2009, dynamical M/L_b = 0.8 (M* = 9.5×10¹⁰)
//	H I           4.23×10⁹   Chemin+2009, integrated 21-cm flux, D = 785 kpc
//	H₂            3.5×10⁸    Dame+1993 CO survey, as adopted by Chemin+2009
//	dust          5.4×10⁷    Draine+2014
//	black hole    1.4×10⁸    Bender+2005
//	dark matter   NFW, C = 12, M(Δ = 98, H0 = 71) = 1.2×10¹² (Corbelli+2010 ΛCDM fit),
//	              converted to M200c for Planck 2018 together with the baryons.
//
// Warm and hot ionized gas are not constrained by these works and are left at 0.
type M31Obs struct {
	c        cache
	datasets []obsdata.Dataset
	halo     m31Halo
}

// m31Halo is the Corbelli+2010 NFW halo re-expressed as M200c.
type m31Halo struct {
	rs            float64 // kpc, physical NFW scale radius
	m200c         float64 // total (DM + baryons) inside R200c
	dm200c        float64 // dark matter inside R200c
	r200c         float64
	c200c         float64
	m98, c98, r98 float64
}

const (
	m31ObsDisc  = 7.1e10
	m31ObsBulge = 2.4e10
	m31ObsHI    = 4.23e9
	m31ObsH2    = 3.5e8
	m31ObsDust  = 5.4e7
	m31ObsBH    = 1.4e8
)

// NewM31Obs builds the preset and parses its bundled tables.
func NewM31Obs() *M31Obs {
	p := &M31Obs{}
	for _, f := range []string{
		"chemin2009_rotation_curve.csv",
		"corbelli2010_rotation_curve.csv",
		"chemin2009_hi_surface_density.csv",
		"corbelli2010_enclosed_mass.csv",
	} {
		data, err := m31Data.ReadFile("data/m31/" + f)
		if err != nil {
			panic(err)
		}
		d, err := obsdata.Parse(f, data)
		if err != nil {
			panic("presets: bundled table " + f + ": " + err.Error())
		}
		d.ID = "m31obs." + f
		p.datasets = append(p.datasets, d)
	}
	p.halo = convertCorbelliHalo(units.Planck18, m31ObsDisc+m31ObsBulge+m31ObsHI+m31ObsH2+m31ObsDust+m31ObsBH)
	return p
}

// convertCorbelliHalo turns the NFW halo of Corbelli+2010 (C = 12 and
// M = 1.2×10¹² M☉ for Δ = 98 ρcrit with H0 = 71) into its physical profile
// and finds R200c in our cosmology where (M_DM(<R) + M_baryons) equals
// 200 ρcrit · 4/3 π R³. The baryons lie well inside R200c.
func convertCorbelliHalo(cosmo units.Cosmology, baryons float64) m31Halo {
	const m98, c98, h0 = 1.2e12, 12.0, 71.0
	rhoCrit71 := 3 * (h0 / 1000) * (h0 / 1000) / (8 * math.Pi * units.G)
	r98 := math.Cbrt(3 * m98 / (4 * math.Pi * 98 * rhoCrit71))
	rs := r98 / c98
	mu := func(x float64) float64 { return math.Log(1+x) - x/(1+x) }
	mdm := func(r float64) float64 { return m98 * mu(r/rs) / mu(c98) }
	rc := cosmo.RhoCrit()
	f := func(r float64) float64 { return mdm(r) + baryons - 200*rc*4*math.Pi/3*r*r*r }
	lo, hi := 50.0, 600.0
	for i := 0; i < 200; i++ {
		mid := 0.5 * (lo + hi)
		if f(mid) > 0 {
			lo = mid
		} else {
			hi = mid
		}
	}
	r200 := 0.5 * (lo + hi)
	return m31Halo{rs: rs, r200c: r200, dm200c: mdm(r200), m200c: mdm(r200) + baryons, c200c: r200 / rs, m98: m98, c98: c98, r98: r98}
}

func (p *M31Obs) Info() galaxy.PresetInfo {
	return galaxy.PresetInfo{
		ID:          "m31-observational",
		Name:        "Spiral · M31 observational",
		Morphology:  "spiral",
		Kind:        "observational",
		Version:     "0.1.0",
		Description: "M31 with published mass estimates (Chemin+2009, Corbelli+2010 and others) and the measured H I surface-density profile; the H I rotation curves, Σ_HI and dynamical M(<R) are bundled for comparison.",
		Disclaimer:  "Masses and radial profiles are inferred from observations under the authors' assumptions (M/L, tilted rings, NFW halo). The 3D vertical structure and the halo shape are reconstructions. Editing the budget turns it into a modified reconstruction.",
	}
}

// Datasets returns the bundled published tables.
func (p *M31Obs) Datasets() []obsdata.Dataset { return append([]obsdata.Dataset(nil), p.datasets...) }

func (p *M31Obs) DefaultBudget() budget.Budget {
	h := p.halo
	tot := h.m200c
	cats := []budget.Category{
		{ID: string(ontology.DarkMatter), Fraction: remainder(m31ObsDisc/tot, m31ObsBulge/tot, m31ObsHI/tot, m31ObsH2/tot, m31ObsDust/tot, m31ObsBH/tot)},
		{ID: string(ontology.Stars), Fraction: (m31ObsDisc + m31ObsBulge) / tot},
		{ID: string(ontology.GasHI), Fraction: m31ObsHI / tot},
		{ID: string(ontology.GasH2), Fraction: m31ObsH2 / tot},
		{ID: string(ontology.GasWarmIonized), Fraction: 0},
		{ID: string(ontology.GasHotIonized), Fraction: 0},
		{ID: string(ontology.Dust), Fraction: m31ObsDust / tot},
		{ID: string(ontology.BlackHole), Fraction: m31ObsBH / tot},
	}
	s := string(ontology.Stars)
	shares := []shareDef{
		{s, "stars.disc", m31ObsDisc / (m31ObsDisc + m31ObsBulge)},
		{s, "stars.bulge", m31ObsBulge / (m31ObsDisc + m31ObsBulge)},
		{string(ontology.GasHI), "gas.hi.disc", 1},
		{string(ontology.GasH2), "gas.h2.disc", 1},
		{string(ontology.Dust), "dust.disc", 1},
		{string(ontology.DarkMatter), "dm.halo", 1},
		{string(ontology.BlackHole), "bh.central", 1},
	}
	return makeBudget(tot, cats, shares)
}

func (*M31Obs) Tags() []ontology.Tag {
	return []ontology.Tag{
		{ID: "disc_region", Label: "Galactic disc (all phases)", Family: "regions", Note: "Stellar disc and all disc gas and dust."},
	}
}

// sigmaHI returns the Chemin+2009 Σ_HI table (kpc, M☉/pc²).
func (p *M31Obs) sigmaHI() ([]float64, []float64) {
	for _, d := range p.datasets {
		if d.Quantity == obsdata.SurfaceDensity {
			r := make([]float64, len(d.Points))
			s := make([]float64, len(d.Points))
			for i, pt := range d.Points {
				r[i], s[i] = pt.R, pt.V
			}
			return r, s
		}
	}
	panic("presets: Σ_HI table missing")
}

func (p *M31Obs) Build(env galaxy.Env) []*galaxy.Component {
	r200 := env.R200c
	disc := []string{"disc_region"}
	stars := p.c.get("disc", func() physics.Distribution {
		return physics.NewDisc(physics.DiscOptions{Shape: physics.Exponential, Rd: 4.5, Z0: 0.7, RMax: math.Min(35, r200)})
	})
	bulge := p.c.get("bulge", func() physics.Distribution {
		return physics.NewSpheroid(physics.Hernquist(0.55), 20, physics.SpheroidOptions{P: 1, Q: 0.8})
	})
	hi := p.c.get("hi", func() physics.Distribution {
		r, s := p.sigmaHI()
		return physics.NewDisc(physics.DiscOptions{Shape: physics.Tabulated, TableR: r, TableSigma: s, Z0: 0.3, RFlare: 15, RMax: 40})
	})
	h2 := p.c.get("h2", func() physics.Distribution {
		return physics.NewDisc(physics.DiscOptions{Shape: physics.ExponentialHole, Rd: 4, Rm: 25, Z0: 0.1, RMax: math.Min(20, r200)})
	})
	dust := p.c.get("dust", func() physics.Distribution {
		return physics.NewDisc(physics.DiscOptions{Shape: physics.ExponentialHole, Rd: 5, Rm: 24, Z0: 0.2, RMax: math.Min(25, r200)})
	})
	rs := p.halo.rs
	dm := p.c.get(r200Key("dm", r200), func() physics.Distribution {
		return physics.NewSpheroid(physics.NFW(rs, r200/rs), r200, physics.SpheroidOptions{})
	})
	h := p.halo
	return []*galaxy.Component{
		{ID: "stars.disc", Name: "Stellar disc", Matter: ontology.Stars, Structure: "disc", Tags: disc, Evidence: ontology.Inferred,
			Sources: []string{"chemin2009", "corbelli2010"}, Dist: stars, GeomKey: "disc",
			Notes:  "Mass 7.1×10¹⁰ M☉ from R-band light with M/L = 1.7 (Chemin+2009). Exponential scale length 4.5 kpc (near-IR value used in the Corbelli+2010 fits); sech² thickness z0 = 0.7 kpc is a reconstruction.",
			Visual: galaxy.Visual{Style: "stars", Color: "#cfe0ff", Size: 0.18, Opacity: 0.5, Markers: 100000}},
		{ID: "stars.bulge", Name: "Bulge", Matter: ontology.Stars, Structure: "bulge", Evidence: ontology.Inferred,
			Sources: []string{"chemin2009", "courteau2011", "hernquist1990"}, Dist: bulge, GeomKey: "bulge",
			Notes:  "Mass 2.4×10¹⁰ M☉: Chemin+2009 total M* = 9.5×10¹⁰ M☉ with the dynamical bulge M/L = 0.8, minus the disc. Hernquist shape with Re ≈ 1 kpc (Courteau+2011), q = 0.8.",
			Visual: galaxy.Visual{Style: "stars", Color: "#ffd59a", Size: 0.12, Opacity: 0.55, Markers: 40000}},
		{ID: "gas.hi.disc", Name: "H I disc (measured Σ_HI)", Matter: ontology.GasHI, Structure: "gas_disc", Tags: disc, Evidence: ontology.Inferred,
			Sources: []string{"chemin2009"}, Dist: hi, GeomKey: "hi",
			Notes:  "Radial profile interpolated in the Chemin+2009 Σ_HI table (100 rings to 38 kpc); mass 4.23×10⁹ M☉ from the integrated 21-cm flux. Hydrogen only, no helium. The flaring vertical structure is a reconstruction; the warp is not modelled.",
			Visual: galaxy.Visual{Style: "gas", Color: "#6fb4ff", Size: 1.4, Opacity: 0.12, Markers: 30000}},
		{ID: "gas.h2.disc", Name: "Molecular gas (H₂)", Matter: ontology.GasH2, Structure: "gas_disc", Tags: disc, Evidence: ontology.Inferred,
			Sources: []string{"dame1993", "chemin2009", "nieten2006"}, Dist: h2, GeomKey: "h2",
			Notes:  "Mass 3.5×10⁸ M☉ from CO (Dame+1993, scaled to 785 kpc by Chemin+2009). The ring-shaped radial profile peaking near 10 kpc is a parametric reconstruction.",
			Visual: galaxy.Visual{Style: "gas", Color: "#ff7fa8", Size: 0.6, Opacity: 0.18, Markers: 12000}},
		{ID: "dust.disc", Name: "Dust disc", Matter: ontology.Dust, Structure: "dust_disc", Tags: disc, Evidence: ontology.Inferred,
			Sources: []string{"draine2014"}, Dist: dust, GeomKey: "dust",
			Notes:  "Mass 5.4×10⁷ M☉ (Draine+2014); ring-like radial profile is a parametric reconstruction.",
			Visual: galaxy.Visual{Style: "dust", Color: "#5a3a22", Size: 1.2, Opacity: 0.22, Markers: 14000}},
		{ID: "dm.halo", Name: "Dark matter halo (Corbelli+2010 NFW)", Matter: ontology.DarkMatter, Structure: "dark_halo", Evidence: ontology.Inferred,
			Sources: []string{"corbelli2010", "nfw1997", "planck2018"}, Dist: dm, GeomKey: r200Key("dm", r200),
			Notes: "NFW fit to the 21-cm rotation curve: C = 12, M = 1.2×10¹² M☉ for Δ = 98 ρcrit (H0 = 71), i.e. rs = " + fmtKpc(h.rs) +
				". Re-expressed for Planck 2018 as R200c = " + fmtKpc(h.r200c) + ", c200c = " + fmtFloat(h.c200c) + ". Gravitationally inferred, not emission.",
			Visual: galaxy.Visual{Style: "dm", Color: "#8f7dff", Size: 22, Opacity: 0.03, Markers: 12000}},
		{ID: "bh.central", Name: "Central black hole", Matter: ontology.BlackHole, Structure: "nucleus", Evidence: ontology.Inferred,
			Sources: []string{"bender2005"}, Dist: physics.PointMass{}, GeomKey: "bh",
			Notes:  "1.4×10⁸ M☉ (+0.9/−0.3) from nuclear stellar dynamics (Bender+2005). Schematic icon, not to scale.",
			Visual: galaxy.Visual{Style: "icon", Color: "#ffffff", Size: 1, Opacity: 1, Markers: 1}},
	}
}

// HaloConversion exposes the halo re-expression for tests and display.
func (p *M31Obs) HaloConversion() (rs, r200c, m200c, c200c float64) {
	return p.halo.rs, p.halo.r200c, p.halo.m200c, p.halo.c200c
}

func fmtKpc(x float64) string { return fmtFloat(x) + " kpc" }

// fmtFloat rounds to one decimal for notes.
func fmtFloat(x float64) string { return strconv.FormatFloat(math.Round(x*10)/10, 'f', -1, 64) }
