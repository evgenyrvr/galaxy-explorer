package physics

import (
	"math"
	"sort"
)

// MidplanePotential is implemented by distributions with an exact midplane
// potential. It returns Φ(R, z = 0)/(G·M) in kpc⁻¹, or NaN if not applicable.
type MidplanePotential interface {
	MidplanePhiPerGM(R float64) float64
}

// MidplanePhiPerGM is the potential of a razor-thin exponential disc in its
// plane (Binney & Tremaine 2008, eq. 2.168):
//
//	Φ(R, 0) = −π G Σ0 R [I0(y) K1(y) − I1(y) K0(y)],  y = R / (2 Rd).
func (d *Disc) MidplanePhiPerGM(R float64) float64 {
	if d.shape != Exponential || d.rotated || R <= 0 {
		return math.NaN()
	}
	y := R / (2 * d.rd)
	return -math.Pi * d.norm * R * (besselI0(y)*besselK1(y) - besselI1(y)*besselK0(y))
}

// MidplanePhiPerGM of a point mass is −1/r.
func (PointMass) MidplanePhiPerGM(R float64) float64 {
	if R <= 0 {
		return math.Inf(-1)
	}
	return -1 / R
}

// SphericalPotentialPerGM returns Φ(r)/(G·M) on grid for the spherically
// averaged mass distribution, with Φ(∞) = 0:
//
//	Φ(r) = −G [ M(<r)/r + ∫_r^∞ dM(r')/r' ].
//
// The integral is evaluated on a fine logarithmic grid; outside the
// distribution's extent Φ = −GM/r.
func SphericalPotentialPerGM(d Distribution, grid []float64) []float64 {
	out := make([]float64, len(grid))
	if _, ok := d.(PointMass); ok {
		for i, r := range grid {
			out[i] = -1 / r
		}
		return out
	}
	rmax := d.MaxRadius()
	if len(grid) > 0 {
		rmax = math.Max(rmax, grid[len(grid)-1])
	}
	rmin := 1e-5
	if len(grid) > 0 {
		rmin = math.Min(rmin, grid[0]*0.5)
	}
	const n = 4000
	rs := logGrid(rmin, rmax*1.0001, n)
	f := make([]float64, n)
	for i, r := range rs {
		f[i] = d.SphericalFraction(r)
	}
	// tail[i] = ∫_{r_i}^{rmax} dF / r'
	tail := make([]float64, n)
	for i := n - 2; i >= 0; i-- {
		dF := f[i+1] - f[i]
		tail[i] = tail[i+1] + dF/(0.5*(rs[i]+rs[i+1]))
	}
	phi := make([]float64, n)
	for i := range rs {
		phi[i] = -(f[i]/rs[i] + tail[i])
	}
	for i, r := range grid {
		switch {
		case r >= rs[n-1]:
			out[i] = -1 / r
		case r <= rs[0]:
			out[i] = phi[0]
		default:
			j := sort.SearchFloat64s(rs, r)
			t := math.Log(r/rs[j-1]) / math.Log(rs[j]/rs[j-1])
			out[i] = phi[j-1] + t*(phi[j]-phi[j-1])
		}
	}
	return out
}
