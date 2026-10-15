// Package physics implements unit-mass spatial distributions of galaxy
// components, their enclosed-mass profiles and representative sampling.
//
// Every Distribution is normalised to unit mass. The physical mass of a
// component lives in the mass budget; a Distribution only says *where* that
// mass is. Visual markers are drawn from the distribution and each carries
// weight M/N, so physical masses never depend on the number of markers.
package physics

import (
	"math"
	"math/rand"
)

// Distribution is a unit-mass spatial distribution centred in the galaxy
// frame (kpc; disc plane XY, Z vertical).
type Distribution interface {
	// Kind is a short machine name, e.g. "exponential_disc".
	Kind() string
	// Params returns the geometric parameters for display and export.
	Params() map[string]float64
	// Sample fills pos (len 3n) with n positions and tags (len n) with local
	// tag indices (0 = untagged, k = TagNames()[k-1]).
	Sample(rng *rand.Rand, pos []float32, tags []uint8)
	// SphericalFraction returns M(<r)/M for a sphere of radius r about the
	// galaxy centre.
	SphericalFraction(r float64) float64
	// MaxRadius is the largest galactocentric radius that contains mass.
	MaxRadius() float64
	// TagNames lists local sub-structure tags carried by this distribution.
	TagNames() []string
	// TagFraction returns the mass fraction carried by local tag k (1-based).
	// Tags of one distribution are mutually disjoint.
	TagFraction(k int) float64
	// TagFractionExact reports whether TagFraction is analytic (true) or a
	// sampled estimate (false).
	TagFractionExact() bool
}

// DiscLike is implemented by axisymmetric disc distributions.
type DiscLike interface {
	// SurfaceDensity returns Σ(R) per unit mass, kpc⁻².
	SurfaceDensity(R float64) float64
	// CylindricalFraction returns M(<R)/M inside cylinder radius R.
	CylindricalFraction(R float64) float64
}

// MidplaneVc2 is implemented by distributions with an exact (non-spherical)
// midplane circular-velocity formula. It returns Vc²/(G·M) in kpc⁻¹.
type MidplaneVc2 interface {
	MidplaneVc2PerGM(R float64) float64
}

// Vec3 is a position or velocity in the galaxy frame.
type Vec3 [3]float64

func (a Vec3) Add(b Vec3) Vec3      { return Vec3{a[0] + b[0], a[1] + b[1], a[2] + b[2]} }
func (a Vec3) Sub(b Vec3) Vec3      { return Vec3{a[0] - b[0], a[1] - b[1], a[2] - b[2]} }
func (a Vec3) Scale(s float64) Vec3 { return Vec3{a[0] * s, a[1] * s, a[2] * s} }
func (a Vec3) Dot(b Vec3) float64   { return a[0]*b[0] + a[1]*b[1] + a[2]*b[2] }
func (a Vec3) Norm() float64        { return math.Sqrt(a.Dot(a)) }
func (a Vec3) Cross(b Vec3) Vec3 {
	return Vec3{a[1]*b[2] - a[2]*b[1], a[2]*b[0] - a[0]*b[2], a[0]*b[1] - a[1]*b[0]}
}
func (a Vec3) Unit() Vec3 {
	n := a.Norm()
	if n == 0 {
		return Vec3{0, 0, 1}
	}
	return a.Scale(1 / n)
}

// Mat3 is a row-major rotation matrix.
type Mat3 [3][3]float64

// Identity is the identity rotation.
var Identity = Mat3{{1, 0, 0}, {0, 1, 0}, {0, 0, 1}}

func (m Mat3) Apply(v Vec3) Vec3 {
	return Vec3{
		m[0][0]*v[0] + m[0][1]*v[1] + m[0][2]*v[2],
		m[1][0]*v[0] + m[1][1]*v[1] + m[1][2]*v[2],
		m[2][0]*v[0] + m[2][1]*v[1] + m[2][2]*v[2],
	}
}

func (m Mat3) Mul(o Mat3) Mat3 {
	var r Mat3
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			for k := 0; k < 3; k++ {
				r[i][j] += m[i][k] * o[k][j]
			}
		}
	}
	return r
}

// EulerZXZ builds a rotation from Euler angles in degrees: first rotate by
// alpha about Z, then by beta (inclination) about X, then by gamma about Z.
func EulerZXZ(alphaDeg, betaDeg, gammaDeg float64) Mat3 {
	rz := func(a float64) Mat3 {
		c, s := math.Cos(a), math.Sin(a)
		return Mat3{{c, -s, 0}, {s, c, 0}, {0, 0, 1}}
	}
	rx := func(a float64) Mat3 {
		c, s := math.Cos(a), math.Sin(a)
		return Mat3{{1, 0, 0}, {0, c, -s}, {0, s, c}}
	}
	d := math.Pi / 180
	return rz(gammaDeg * d).Mul(rx(betaDeg * d)).Mul(rz(alphaDeg * d))
}

// randUnit returns a direction uniformly distributed on the unit sphere.
func randUnit(rng *rand.Rand) Vec3 {
	z := 2*rng.Float64() - 1
	phi := 2 * math.Pi * rng.Float64()
	s := math.Sqrt(1 - z*z)
	return Vec3{s * math.Cos(phi), s * math.Sin(phi), z}
}

func put(pos []float32, i int, v Vec3) {
	pos[3*i] = float32(v[0])
	pos[3*i+1] = float32(v[1])
	pos[3*i+2] = float32(v[2])
}

func clamp01(x float64) float64 {
	if x < 0 {
		return 0
	}
	if x > 1 {
		return 1
	}
	return x
}
