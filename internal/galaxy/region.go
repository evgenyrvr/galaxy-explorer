package galaxy

import (
	"math"
	"math/rand"

	"galaxy/internal/physics"
)

// Region restricts a selection in space (galaxy frame, kpc).
//
//	sphere:   rMin ≤ r < rMax                     (r spherical)
//	cylinder: rMin ≤ R < rMax and |z| < zMax      (R cylindrical; zMax ≤ 0: unbounded)
type Region struct {
	Kind string  `json:"kind"` // "", "sphere", "cylinder"
	RMin float64 `json:"rMin"`
	RMax float64 `json:"rMax"`
	ZMax float64 `json:"zMax"`
}

// Active reports whether the region restricts anything.
func (g *Region) Active() bool {
	return g != nil && (g.Kind == "sphere" || g.Kind == "cylinder") && g.RMax > g.RMin
}

func (g *Region) contains(x, y, z float64) bool {
	switch g.Kind {
	case "sphere":
		r := math.Sqrt(x*x + y*y + z*z)
		return r >= g.RMin && r < g.RMax
	case "cylinder":
		R := math.Hypot(x, y)
		return R >= g.RMin && R < g.RMax && (g.ZMax <= 0 || math.Abs(z) < g.ZMax)
	}
	return true
}

// refSampleSize is the size of the deterministic reference sample used for
// region × tag fractions without a closed form (statistical error ≈ 0.4%·√f).
const refSampleSize = 60000

type refSample struct {
	pos  []float32
	tags []uint8
}

// reference returns (and caches) the reference sample of a component.
// Callers hold m.mu.
func (m *Model) reference(c *Component) refSample {
	key := c.ID + "|" + c.GeomKey
	if s, ok := m.refCache[key]; ok {
		return s
	}
	n := refSampleSize
	if f, ok := c.Dist.(physics.FixedCounter); ok && f.FixedCount() > 0 {
		n = f.FixedCount()
	}
	rng := rand.New(rand.NewSource(int64(hash64("ref|"+key)) ^ m.seed))
	s := refSample{pos: make([]float32, 3*n), tags: make([]uint8, n)}
	c.Dist.Sample(rng, s.pos, s.tags)
	if m.refCache == nil || len(m.refCache) > 64 {
		m.refCache = map[string]refSample{}
	}
	m.refCache[key] = s
	return s
}

// regionFraction returns the fraction of c's mass that is both selected
// (whole component, or the given local tag indices) and inside the region.
func (m *Model) regionFraction(c *Component, whole bool, localTags map[int]bool, g *Region) (float64, bool) {
	if whole {
		switch g.Kind {
		case "sphere":
			return math.Max(0, c.Dist.SphericalFraction(g.RMax)-c.Dist.SphericalFraction(g.RMin)), analyticSpherical(c.Dist)
		case "cylinder":
			if d, ok := c.Dist.(*physics.Disc); ok {
				if f, ok := d.SlabFraction(g.RMin, g.RMax, g.ZMax); ok {
					return f, true
				}
			}
			if _, ok := c.Dist.(physics.PointMass); ok {
				return boolFrac(g.contains(0, 0, 0)), true
			}
		}
	}
	s := m.reference(c)
	n := len(s.tags)
	cnt := 0
	for i := 0; i < n; i++ {
		if !whole && !localTags[int(s.tags[i])] {
			continue
		}
		if g.contains(float64(s.pos[3*i]), float64(s.pos[3*i+1]), float64(s.pos[3*i+2])) {
			cnt++
		}
	}
	_, discrete := c.Dist.(physics.FixedCounter)
	return float64(cnt) / float64(n), discrete && whole
}

func boolFrac(b bool) float64 {
	if b {
		return 1
	}
	return 0
}

// analyticSpherical reports whether SphericalFraction comes from quadrature
// rather than from an empirical sample (streams, satellites, jets).
func analyticSpherical(d physics.Distribution) bool {
	switch d.(type) {
	case *physics.Spheroid, *physics.Disc, physics.PointMass:
		return true
	}
	return false
}
