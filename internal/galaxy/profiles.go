package galaxy

import (
	"math"

	"galaxy/internal/physics"
	"galaxy/internal/units"
)

// Series is one curve.
type Series struct {
	ID    string    `json:"id"`
	Label string    `json:"label"`
	Y     []float64 `json:"y"`
}

// Profiles are radial curves of the current mass model. Units are fixed per
// quantity and never mixed: M(<r) in M☉, ρ(r) in M☉/pc³ (spherically
// averaged, volume density), Σ(R) in M☉/pc² (surface density, discs only),
// Vc in km/s.
type Profiles struct {
	R          []float64 `json:"r"` // kpc (spherical r, or cylindrical R for Σ)
	Enclosed   []Series  `json:"enclosed"`
	Density    []Series  `json:"density"`
	Surface    []Series  `json:"surface"`
	Vc         []Series  `json:"vc"`
	BaryonFrac []float64 `json:"baryonFraction"` // f_b(<r), excluding the black hole
	R200c      float64   `json:"r200c"`
	Note       string    `json:"note"`
	Provenance string    `json:"provenance"`
}

// ProfileRequest selects components and range.
type ProfileRequest struct {
	Components []string `json:"components"` // empty: all
	RMin       float64  `json:"rMin"`       // kpc
	RMax       float64  `json:"rMax"`       // kpc; 0: R200c
	N          int      `json:"n"`
}

// Profiles computes radial profiles.
func (m *Model) Profiles(req ProfileRequest) Profiles {
	m.mu.Lock()
	defer m.mu.Unlock()
	rmax := req.RMax
	if rmax <= 0 {
		rmax = m.r200c
	}
	rmin := req.RMin
	if rmin <= 0 {
		rmin = 0.01
	}
	n := req.N
	if n < 16 {
		n = 160
	}
	grid := make([]float64, n)
	l := math.Log(rmax / rmin)
	for i := range grid {
		grid[i] = rmin * math.Exp(l*float64(i)/float64(n-1))
	}
	want := map[string]bool{}
	for _, id := range req.Components {
		want[id] = true
	}
	p := Profiles{R: grid, R200c: m.r200c,
		Note:       "Vc uses the Freeman (1970) midplane formula for pure exponential stellar discs and the spherical approximation G·M(<r)/r for all other components.",
		Provenance: string(m.preset.Info().Kind),
	}
	total := make([]float64, n)
	baryon := make([]float64, n)
	vc2 := make([]float64, n)
	for _, c := range m.comps {
		mass := m.componentMass(c)
		enc := make([]float64, n)
		for i, r := range grid {
			enc[i] = mass * c.Dist.SphericalFraction(r)
			total[i] += enc[i]
			if info, ok := matterBaryonic(c); ok && info {
				baryon[i] += enc[i]
			}
		}
		// Circular velocity contribution.
		for i, r := range grid {
			var v2 float64
			if d, ok := c.Dist.(physics.MidplaneVc2); ok {
				if x := d.MidplaneVc2PerGM(r); !math.IsNaN(x) {
					v2 = units.G * mass * x
				} else {
					v2 = units.G * enc[i] / r
				}
			} else {
				v2 = units.G * enc[i] / r
			}
			vc2[i] += v2
		}
		if len(want) > 0 && !want[c.ID] {
			continue
		}
		if mass <= 0 {
			continue
		}
		p.Enclosed = append(p.Enclosed, Series{ID: c.ID, Label: c.Name, Y: enc})
		p.Density = append(p.Density, Series{ID: c.ID, Label: c.Name, Y: densityFromEnclosed(grid, enc)})
		if d, ok := c.Dist.(physics.DiscLike); ok {
			sig := make([]float64, n)
			for i, R := range grid {
				sig[i] = mass * d.SurfaceDensity(R) / 1e6 // M☉/kpc² → M☉/pc²
			}
			p.Surface = append(p.Surface, Series{ID: c.ID, Label: c.Name, Y: sig})
		}
		vcc := make([]float64, n)
		for i, r := range grid {
			if d, ok := c.Dist.(physics.MidplaneVc2); ok {
				if x := d.MidplaneVc2PerGM(r); !math.IsNaN(x) {
					vcc[i] = math.Sqrt(math.Max(0, units.G*mass*x))
					continue
				}
			}
			vcc[i] = math.Sqrt(units.G * enc[i] / r)
		}
		p.Vc = append(p.Vc, Series{ID: c.ID, Label: c.Name, Y: vcc})
	}
	p.Enclosed = append([]Series{{ID: "total", Label: "Total", Y: total}}, p.Enclosed...)
	p.Density = append([]Series{{ID: "total", Label: "Total", Y: densityFromEnclosed(grid, total)}}, p.Density...)
	vt := make([]float64, n)
	for i := range vt {
		vt[i] = math.Sqrt(math.Max(0, vc2[i]))
	}
	p.Vc = append([]Series{{ID: "total", Label: "Total", Y: vt}}, p.Vc...)
	p.BaryonFrac = make([]float64, n)
	for i := range total {
		if total[i] > 0 {
			p.BaryonFrac[i] = baryon[i] / total[i]
		}
	}
	return p
}

func matterBaryonic(c *Component) (bool, bool) {
	switch c.Matter {
	case "dark_matter", "black_hole":
		return false, true
	default:
		return true, true
	}
}

// densityFromEnclosed returns the spherically averaged ρ = dM/dr / (4πr²) in M☉/pc³.
func densityFromEnclosed(r, enc []float64) []float64 {
	n := len(r)
	out := make([]float64, n)
	for i := 0; i < n; i++ {
		a, b := i-1, i+1
		if a < 0 {
			a = 0
		}
		if b >= n {
			b = n - 1
		}
		dm := enc[b] - enc[a]
		dr := r[b] - r[a]
		if dr <= 0 {
			continue
		}
		out[i] = math.Max(0, dm/dr/(4*math.Pi*r[i]*r[i])) / 1e9
	}
	return out
}

// EnclosedFrom builds M(<r) of a set of components for a budget, used for
// orbit integration of streams and shells.
func EnclosedFrom(masses map[string]float64, comps []*Component, rmax float64) physics.EnclosedMass {
	grid := make([]float64, 600)
	l := math.Log(rmax / 1e-3)
	for i := range grid {
		grid[i] = 1e-3 * math.Exp(l*float64(i)/float64(len(grid)-1))
	}
	ys := make([]float64, len(grid))
	for _, c := range comps {
		mass := masses[c.ID]
		for i, r := range grid {
			ys[i] += mass * c.Dist.SphericalFraction(r)
		}
	}
	return func(r float64) float64 {
		if r <= grid[0] {
			return ys[0] * r / grid[0]
		}
		if r >= grid[len(grid)-1] {
			return ys[len(ys)-1]
		}
		lo, hi := 0, len(grid)-1
		for hi-lo > 1 {
			mid := (lo + hi) / 2
			if grid[mid] > r {
				hi = mid
			} else {
				lo = mid
			}
		}
		t := (r - grid[lo]) / (grid[hi] - grid[lo])
		return ys[lo] + t*(ys[hi]-ys[lo])
	}
}
