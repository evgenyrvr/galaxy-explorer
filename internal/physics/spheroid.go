package physics

import (
	"math"
	"math/rand"
)

// RadialLaw is a spherical density shape ρ(m) (arbitrary normalisation),
// evaluated at the ellipsoidal radius m.
type RadialLaw struct {
	Kind   string
	Rho    func(m float64) float64
	Params map[string]float64
}

// Hernquist (1990, ApJ 356, 359): ρ ∝ 1 / [(m/a)(1 + m/a)³].
// The projected half-mass radius is Re ≈ 1.8153 a; it approximates a
// de Vaucouleurs (Sérsic n = 4) profile.
func Hernquist(a float64) RadialLaw {
	return RadialLaw{
		Kind:   "hernquist",
		Rho:    func(m float64) float64 { x := m / a; return 1 / (x * (1 + x) * (1 + x) * (1 + x)) },
		Params: map[string]float64{"a_kpc": a, "Re_kpc": 1.8153 * a},
	}
}

// NFW (Navarro, Frenk & White 1997, ApJ 490, 493): ρ ∝ 1 / [(m/rs)(1 + m/rs)²].
func NFW(rs, c float64) RadialLaw {
	return RadialLaw{
		Kind:   "nfw",
		Rho:    func(m float64) float64 { x := m / rs; return 1 / (x * (1 + x) * (1 + x)) },
		Params: map[string]float64{"rs_kpc": rs, "c200c": c},
	}
}

// Plummer: ρ ∝ (1 + m²/a²)^(−5/2).
func Plummer(a float64) RadialLaw {
	return RadialLaw{
		Kind:   "plummer",
		Rho:    func(m float64) float64 { x := m / a; return math.Pow(1+x*x, -2.5) },
		Params: map[string]float64{"a_kpc": a},
	}
}

// Beta model (Cavaliere & Fusco-Femiano 1976) for hot atmospheres:
// ρ ∝ [1 + (m/rc)²]^(−3β/2).
func BetaModel(rc, beta float64) RadialLaw {
	return RadialLaw{
		Kind:   "beta_model",
		Rho:    func(m float64) float64 { x := m / rc; return math.Pow(1+x*x, -1.5*beta) },
		Params: map[string]float64{"rc_kpc": rc, "beta": beta},
	}
}

// CoredPowerLaw: ρ ∝ (m² + rc²)^(−α/2), used for stellar haloes.
func CoredPowerLaw(rc, alpha float64) RadialLaw {
	return RadialLaw{
		Kind:   "cored_power_law",
		Rho:    func(m float64) float64 { return math.Pow(m*m+rc*rc, -alpha/2) },
		Params: map[string]float64{"rc_kpc": rc, "alpha": alpha},
	}
}

// Spheroid is an ellipsoidal distribution ρ(m), m² = x² + y²/p² + z²/q²
// (p, q ≤ 1), truncated at m ≤ mt, optionally rotated. Optional radial
// splits tag shells in m (e.g. inner/outer spheroid at Re).
type Spheroid struct {
	law       RadialLaw
	p, q      float64
	mt        float64
	rot       Mat3
	rotated   bool
	base      *CDFTable // CDF of m
	sph       *gridFunc // spherical fraction if non-spherical
	splits    []float64
	splitTags []string
}

// SpheroidOptions configures NewSpheroid.
type SpheroidOptions struct {
	P, Q      float64 // axis ratios y/x and z/x; 0 means 1
	Rotation  *Mat3
	Splits    []float64 // increasing boundaries in m
	SplitTags []string  // len(Splits)+1 names; empty string = untagged
}

// NewSpheroid builds a truncated, normalised ellipsoidal distribution.
func NewSpheroid(law RadialLaw, mt float64, o SpheroidOptions) *Spheroid {
	p, q := o.P, o.Q
	if p == 0 {
		p = 1
	}
	if q == 0 {
		q = 1
	}
	s := &Spheroid{law: law, p: p, q: q, mt: mt, rot: Identity}
	if o.Rotation != nil {
		s.rot = *o.Rotation
		s.rotated = true
	}
	s.base = NewCDFTable(func(m float64) float64 {
		if m <= 0 {
			return 0
		}
		return 4 * math.Pi * m * m * law.Rho(m)
	}, mt, 4096)
	if len(o.Splits) > 0 {
		s.splits = o.Splits
		s.splitTags = o.SplitTags
	}
	if p != 1 || q != 1 {
		s.sph = s.buildSphericalTable()
	}
	return s
}

// buildSphericalTable evaluates M(<r)/M = (1/4π)∮ F(r / g(n)) dΩ with
// g(n) = sqrt(nx² + p²ny² + q²nz²), which maps unit-sphere directions in the
// ellipsoid's "spherical" coordinates to physical radii.
func (s *Spheroid) buildSphericalTable() *gridFunc {
	mu, wmu := gaussLegendre(32)
	const nphi = 64
	rs := logGrid(s.mt*1e-6, s.mt, 400)
	ys := make([]float64, len(rs))
	for i, r := range rs {
		sum := 0.0
		for a := range mu {
			st := math.Sqrt(1 - mu[a]*mu[a])
			for b := 0; b < nphi; b++ {
				phi := 2 * math.Pi * (float64(b) + 0.5) / nphi
				nx, ny, nz := st*math.Cos(phi), st*math.Sin(phi), mu[a]
				g := math.Sqrt(nx*nx + s.p*s.p*ny*ny + s.q*s.q*nz*nz)
				sum += wmu[a] * s.base.At(r/g)
			}
		}
		ys[i] = sum * (2 * math.Pi / nphi) / (4 * math.Pi)
	}
	return &gridFunc{x: rs, y: ys}
}

func (s *Spheroid) Kind() string { return s.law.Kind }

func (s *Spheroid) Params() map[string]float64 {
	m := map[string]float64{"p": s.p, "q": s.q, "truncation_kpc": s.mt}
	for k, v := range s.law.Params {
		m[k] = v
	}
	return m
}

func (s *Spheroid) Sample(rng *rand.Rand, pos []float32, tags []uint8) {
	n := len(tags)
	for i := 0; i < n; i++ {
		m := s.base.Inverse(rng.Float64())
		d := randUnit(rng)
		v := Vec3{m * d[0], m * s.p * d[1], m * s.q * d[2]}
		if s.rotated {
			v = s.rot.Apply(v)
		}
		put(pos, i, v)
		tags[i] = s.tagFor(m)
	}
}

func (s *Spheroid) tagFor(m float64) uint8 {
	if len(s.splits) == 0 {
		return 0
	}
	idx := 0
	for idx < len(s.splits) && m >= s.splits[idx] {
		idx++
	}
	if s.splitTags[idx] == "" {
		return 0
	}
	return uint8(s.tagIndex(idx))
}

// tagIndex maps a split shell index to the 1-based local tag index.
func (s *Spheroid) tagIndex(shell int) int {
	k := 0
	for i := 0; i <= shell; i++ {
		if s.splitTags[i] != "" {
			k++
		}
	}
	return k
}

func (s *Spheroid) SphericalFraction(r float64) float64 {
	if s.sph == nil {
		return s.base.At(r)
	}
	return clamp01(s.sph.at(r))
}

func (s *Spheroid) MaxRadius() float64 { return s.mt }

func (s *Spheroid) TagNames() []string {
	var names []string
	for _, t := range s.splitTags {
		if t != "" {
			names = append(names, t)
		}
	}
	return names
}

func (s *Spheroid) TagFraction(k int) float64 {
	seen := 0
	for i, t := range s.splitTags {
		if t == "" {
			continue
		}
		seen++
		if seen != k {
			continue
		}
		lo, hi := 0.0, s.mt
		if i > 0 {
			lo = s.splits[i-1]
		}
		if i < len(s.splits) {
			hi = s.splits[i]
		}
		return s.base.At(hi) - s.base.At(lo)
	}
	return 0
}

func (s *Spheroid) TagFractionExact() bool { return true }

// EllipsoidalFraction returns the fraction of mass with ellipsoidal radius < m.
func (s *Spheroid) EllipsoidalFraction(m float64) float64 { return s.base.At(m) }
