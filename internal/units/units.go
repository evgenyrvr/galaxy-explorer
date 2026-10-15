// Package units holds physical constants and the adopted cosmology.
//
// Internal unit system: length in kpc, mass in M☉, velocity in km/s.
// Time is then measured in kpc/(km/s) ≈ 0.9778 Gyr.
package units

import "math"

const (
	// G is Newton's constant in kpc (km/s)^2 / M☉.
	G = 4.30091727e-6

	// KpcPerKmsInGyr converts the internal time unit kpc/(km/s) to Gyr.
	KpcPerKmsInGyr = 0.977792

	// PcPerKpc converts kpc to pc.
	PcPerKpc = 1000.0
)

// Cosmology describes a flat ΛCDM model at z = 0.
type Cosmology struct {
	Name   string  `json:"name"`
	H0     float64 `json:"h0"`     // km/s/Mpc
	OmegaM float64 `json:"omegaM"` // total matter density parameter
	OmegaB float64 `json:"omegaB"` // baryon density parameter
	Source string  `json:"source"` // reference id
}

// Planck18 is Planck 2018 VI, TT,TE,EE+lowE+lensing+BAO (Table 2, last column).
var Planck18 = Cosmology{
	Name:   "Planck 2018",
	H0:     67.66,
	OmegaM: 0.3111,
	OmegaB: 0.04897,
	Source: "planck2018",
}

// Hubble returns the dimensionless Hubble parameter h = H0 / 100.
func (c Cosmology) Hubble() float64 { return c.H0 / 100 }

// RhoCrit returns the critical density at z = 0 in M☉/kpc^3.
// ρcrit = 3 H0² / (8πG). H0 in km/s/kpc = H0[km/s/Mpc] / 1000.
func (c Cosmology) RhoCrit() float64 {
	h := c.H0 / 1000.0
	return 3 * h * h / (8 * math.Pi * G)
}

// BaryonFraction returns the cosmic baryon fraction Ωb/Ωm.
func (c Cosmology) BaryonFraction() float64 { return c.OmegaB / c.OmegaM }

// R200c returns the radius (kpc) inside which the mean density is
// 200 ρcrit for the given enclosed mass M200c (M☉).
func (c Cosmology) R200c(m200c float64) float64 {
	return math.Cbrt(3 * m200c / (4 * math.Pi * 200 * c.RhoCrit()))
}

// V200c returns the circular velocity at R200c in km/s.
func (c Cosmology) V200c(m200c float64) float64 {
	return math.Sqrt(G * m200c / c.R200c(m200c))
}

// ConcentrationDM14 is the z = 0 NFW concentration–mass relation for M200c
// from Dutton & Macciò (2014, MNRAS 441, 3359), Planck cosmology:
// log10 c = 0.905 − 0.101 log10(M200c / 10^12 h^-1 M☉).
func (c Cosmology) ConcentrationDM14(m200c float64) float64 {
	x := m200c / (1e12 / c.Hubble())
	return math.Pow(10, 0.905-0.101*math.Log10(x))
}
