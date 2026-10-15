// Package presets defines the galaxy templates. Normalised presets use
// rounded literature values of real galaxies as *morphological guides*; their
// masses follow the user's M_total, not the real galaxy.
package presets

import (
	"fmt"
	"math"
	"math/rand"
	"sync"

	"galaxy/internal/budget"
	"galaxy/internal/galaxy"
	"galaxy/internal/physics"
)

// All returns the available presets.
func All() []galaxy.Preset {
	return []galaxy.Preset{NewM31(), NewM31Obs(), NewM87()}
}

// ByID finds a preset.
func ByID(id string) (galaxy.Preset, error) {
	for _, p := range All() {
		if p.Info().ID == id {
			return p, nil
		}
	}
	return nil, fmt.Errorf("unknown preset %q", id)
}

// cache memoises expensive distributions (orbit integrations, quadratures).
type cache struct {
	mu sync.Mutex
	m  map[string]physics.Distribution
}

func (c *cache) get(key string, build func() physics.Distribution) physics.Distribution {
	c.mu.Lock()
	if d, ok := c.m[key]; ok {
		c.mu.Unlock()
		return d
	}
	c.mu.Unlock()
	// Build without holding the lock: builders may recurse into the cache.
	d := build()
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.m == nil {
		c.m = map[string]physics.Distribution{}
	}
	c.m[key] = d
	return d
}

// withoutNil drops components whose distribution was omitted.
func withoutNil(cs []*galaxy.Component) []*galaxy.Component {
	out := cs[:0]
	for _, c := range cs {
		if c.Dist != nil {
			out = append(out, c)
		}
	}
	return out
}

// defaultMasses returns component masses for a budget.
func defaultMasses(b budget.Budget, comps []*galaxy.Component) map[string]float64 {
	m := map[string]float64{}
	for _, c := range comps {
		m[c.ID] = b.ComponentMass(string(c.Matter), c.ID)
	}
	return m
}

// shareDef declares a component's default share inside its category.
type shareDef struct {
	cat   string
	comp  string
	share float64
}

func makeBudget(total float64, fractions []budget.Category, shares []shareDef) budget.Budget {
	b := budget.Budget{TotalMass: total, Mode: budget.FixTotal, Categories: fractions, Shares: map[string][]budget.Share{}}
	for _, s := range shares {
		b.Shares[s.cat] = append(b.Shares[s.cat], budget.Share{Component: s.comp, Share: s.share})
	}
	return b
}

// remainder returns 1 − Σ others, used so the default budget sums to exactly 100%.
func remainder(others ...float64) float64 {
	s := 0.0
	for _, o := range others {
		s += o
	}
	return 1 - s
}

// discreteFrom draws n discrete objects from a distribution.
func discreteFrom(kind string, d physics.Distribution, n int, seed int64, params map[string]float64) *physics.Sampled {
	rng := rand.New(rand.NewSource(seed))
	pos := make([]float32, 3*n)
	tags := make([]uint8, n)
	d.Sample(rng, pos, tags)
	ref := make([]physics.Vec3, n)
	for i := range ref {
		ref[i] = physics.Vec3{float64(pos[3*i]), float64(pos[3*i+1]), float64(pos[3*i+2])}
	}
	p := map[string]float64{"objects": float64(n)}
	for k, v := range d.Params() {
		p[k] = v
	}
	for k, v := range params {
		p[k] = v
	}
	return physics.NewSampled(kind, p, ref, nil, nil, true)
}

// jetSample builds a narrow conical jet with knots along axis.
func jetSample(axis physics.Vec3, length, halfAngleDeg float64, knots []float64, n int, seed int64) []physics.Vec3 {
	rng := rand.New(rand.NewSource(seed))
	a := axis.Unit()
	u, v := perpendicularBasis(a)
	tanA := math.Tan(halfAngleDeg * math.Pi / 180)
	out := make([]physics.Vec3, 0, n)
	for i := 0; i < n; i++ {
		var s float64
		if len(knots) > 0 && rng.Float64() < 0.45 {
			k := knots[rng.Intn(len(knots))]
			s = k + 0.02*length*rng.NormFloat64()
		} else {
			s = length * math.Sqrt(rng.Float64())
		}
		s = math.Max(0.001, math.Min(length, s))
		w := tanA * s * math.Sqrt(rng.Float64())
		phi := 2 * math.Pi * rng.Float64()
		p := a.Scale(s).Add(u.Scale(w * math.Cos(phi))).Add(v.Scale(w * math.Sin(phi)))
		out = append(out, p)
	}
	return out
}

// filamentSample builds curved filaments radiating from the centre.
func filamentSample(count int, rmin, rmax, width float64, n int, seed int64) []physics.Vec3 {
	rng := rand.New(rand.NewSource(seed))
	type fil struct{ pts []physics.Vec3 }
	fils := make([]fil, count)
	for f := range fils {
		dir := physics.Vec3{rng.NormFloat64(), rng.NormFloat64(), 0.5 * rng.NormFloat64()}.Unit()
		pos := dir.Scale(rmin)
		steps := 60
		step := (rmax - rmin) / float64(steps) * (0.6 + 0.8*rng.Float64())
		for s := 0; s < steps; s++ {
			dir = dir.Add(physics.Vec3{rng.NormFloat64(), rng.NormFloat64(), rng.NormFloat64()}.Scale(0.12)).Unit()
			pos = pos.Add(dir.Scale(step))
			fils[f].pts = append(fils[f].pts, pos)
		}
	}
	out := make([]physics.Vec3, 0, n)
	for i := 0; i < n; i++ {
		f := fils[rng.Intn(count)]
		p := f.pts[rng.Intn(len(f.pts))]
		out = append(out, p.Add(physics.Vec3{rng.NormFloat64(), rng.NormFloat64(), rng.NormFloat64()}.Scale(width)))
	}
	return out
}

func perpendicularBasis(axis physics.Vec3) (physics.Vec3, physics.Vec3) {
	ref := physics.Vec3{1, 0, 0}
	if math.Abs(axis[0]) > 0.9 {
		ref = physics.Vec3{0, 1, 0}
	}
	u := axis.Cross(ref).Unit()
	v := axis.Cross(u).Unit()
	return u, v
}

func r200Key(id string, r200 float64) string { return fmt.Sprintf("%s@%.9g", id, r200) }
