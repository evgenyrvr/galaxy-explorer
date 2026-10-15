package physics

import (
	"math"
	"math/rand"
)

// PointMass is a central point mass (e.g. a supermassive black hole).
type PointMass struct{}

func (PointMass) Kind() string               { return "point_mass" }
func (PointMass) Params() map[string]float64 { return map[string]float64{} }
func (PointMass) MaxRadius() float64         { return 0 }
func (PointMass) TagNames() []string         { return nil }
func (PointMass) TagFraction(int) float64    { return 0 }
func (PointMass) TagFractionExact() bool     { return true }
func (PointMass) FixedCount() int            { return 1 }
func (PointMass) SphericalFraction(r float64) float64 {
	if r > 0 {
		return 1
	}
	return 0
}
func (PointMass) Sample(_ *rand.Rand, pos []float32, tags []uint8) {
	for i := range tags {
		put(pos, i, Vec3{})
		tags[i] = 0
	}
}

// FixedCounter is implemented by distributions of discrete physical objects
// (black hole, globular clusters) that must be drawn exactly once each.
type FixedCounter interface {
	FixedCount() int
}

// Sampled is a distribution represented by a fixed reference sample, used
// for structures without a closed-form density (tidal streams, shells,
// jets, filaments) and for discrete object populations.
type Sampled struct {
	kind     string
	params   map[string]float64
	ref      []Vec3
	tags     []uint8
	tagNames []string
	discrete bool
	sph      *CDFTable
	tagFrac  []float64
}

// NewSampled wraps a reference sample. If discrete is true, each reference
// point is one physical object and Sample returns all of them.
func NewSampled(kind string, params map[string]float64, ref []Vec3, tags []uint8, tagNames []string, discrete bool) *Sampled {
	if tags == nil {
		tags = make([]uint8, len(ref))
	}
	s := &Sampled{kind: kind, params: params, ref: ref, tags: tags, tagNames: tagNames, discrete: discrete}
	radii := make([]float64, len(ref))
	for i, v := range ref {
		radii[i] = v.Norm()
	}
	s.sph = NewCDFTableFromSamples(radii)
	s.tagFrac = make([]float64, len(tagNames)+1)
	for _, t := range tags {
		if int(t) < len(s.tagFrac) {
			s.tagFrac[t]++
		}
	}
	for i := range s.tagFrac {
		s.tagFrac[i] /= float64(len(ref))
	}
	return s
}

func (s *Sampled) Kind() string               { return s.kind }
func (s *Sampled) Params() map[string]float64 { return s.params }
func (s *Sampled) MaxRadius() float64         { return s.sph.Max() }
func (s *Sampled) TagNames() []string         { return s.tagNames }
func (s *Sampled) TagFractionExact() bool     { return false }
func (s *Sampled) SphericalFraction(r float64) float64 {
	return s.sph.At(r)
}
func (s *Sampled) TagFraction(k int) float64 {
	if k < 1 || k >= len(s.tagFrac) {
		return 0
	}
	return s.tagFrac[k]
}

// FixedCount returns the number of discrete objects, or 0 if continuous.
func (s *Sampled) FixedCount() int {
	if s.discrete {
		return len(s.ref)
	}
	return 0
}

func (s *Sampled) Sample(rng *rand.Rand, pos []float32, tags []uint8) {
	n := len(tags)
	if n <= len(s.ref) {
		perm := rng.Perm(len(s.ref))
		for i := 0; i < n; i++ {
			put(pos, i, s.ref[perm[i]])
			tags[i] = s.tags[perm[i]]
		}
		return
	}
	for i := 0; i < n; i++ {
		j := rng.Intn(len(s.ref))
		put(pos, i, s.ref[j])
		tags[i] = s.tags[j]
	}
}

// Offset places a distribution at a galactocentric position (satellites).
type Offset struct {
	inner  Distribution
	centre Vec3
	sph    *CDFTable
}

// NewOffset wraps inner at centre. The spherical fraction about the host
// centre is estimated from a deterministic reference sample.
func NewOffset(inner Distribution, centre Vec3, seed int64) *Offset {
	const nref = 100000
	rng := rand.New(rand.NewSource(seed))
	pos := make([]float32, 3*nref)
	tags := make([]uint8, nref)
	inner.Sample(rng, pos, tags)
	radii := make([]float64, nref)
	for i := 0; i < nref; i++ {
		v := Vec3{float64(pos[3*i]), float64(pos[3*i+1]), float64(pos[3*i+2])}.Add(centre)
		radii[i] = v.Norm()
	}
	return &Offset{inner: inner, centre: centre, sph: NewCDFTableFromSamples(radii)}
}

func (o *Offset) Kind() string { return o.inner.Kind() }
func (o *Offset) Params() map[string]float64 {
	p := map[string]float64{}
	for k, v := range o.inner.Params() {
		p[k] = v
	}
	p["centre_x_kpc"], p["centre_y_kpc"], p["centre_z_kpc"] = o.centre[0], o.centre[1], o.centre[2]
	p["centre_r_kpc"] = o.centre.Norm()
	return p
}
func (o *Offset) Sample(rng *rand.Rand, pos []float32, tags []uint8) {
	o.inner.Sample(rng, pos, tags)
	for i := range tags {
		pos[3*i] += float32(o.centre[0])
		pos[3*i+1] += float32(o.centre[1])
		pos[3*i+2] += float32(o.centre[2])
	}
}
func (o *Offset) SphericalFraction(r float64) float64 { return o.sph.At(r) }
func (o *Offset) MaxRadius() float64                  { return o.sph.Max() }
func (o *Offset) TagNames() []string                  { return o.inner.TagNames() }
func (o *Offset) TagFraction(k int) float64           { return o.inner.TagFraction(k) }
func (o *Offset) TagFractionExact() bool              { return o.inner.TagFractionExact() }
func (o *Offset) Centre() Vec3                        { return o.centre }

// gaussian returns a standard normal deviate.
func gaussian(rng *rand.Rand) float64 { return rng.NormFloat64() }

// Polyline jitter helper used by filament/jet builders.
func lerp(a, b Vec3, t float64) Vec3 { return a.Add(b.Sub(a).Scale(t)) }

// perpendicularBasis returns two unit vectors orthogonal to axis.
func perpendicularBasis(axis Vec3) (Vec3, Vec3) {
	a := axis.Unit()
	ref := Vec3{1, 0, 0}
	if math.Abs(a[0]) > 0.9 {
		ref = Vec3{0, 1, 0}
	}
	u := a.Cross(ref).Unit()
	v := a.Cross(u).Unit()
	return u, v
}
