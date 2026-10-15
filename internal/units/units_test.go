package units

import (
	"math"
	"testing"
)

func TestRhoCritPlanck18(t *testing.T) {
	// 2.775e11 h² M☉/Mpc³ = 277.5 h² M☉/kpc³.
	want := 277.5 * Planck18.Hubble() * Planck18.Hubble()
	got := Planck18.RhoCrit()
	if math.Abs(got-want)/want > 1e-3 {
		t.Fatalf("RhoCrit = %g, want %g", got, want)
	}
}

func TestR200cDefinition(t *testing.T) {
	m := 1e12
	r := Planck18.R200c(m)
	meanRho := m / (4.0 / 3.0 * math.Pi * r * r * r)
	if math.Abs(meanRho/(200*Planck18.RhoCrit())-1) > 1e-12 {
		t.Fatalf("mean density inside R200c is not 200 ρcrit")
	}
	if r < 200 || r > 220 {
		t.Fatalf("R200c(1e12) = %g kpc, expected ≈ 211 kpc", r)
	}
}

func TestConcentration(t *testing.T) {
	c := Planck18.ConcentrationDM14(1e12)
	if c < 8 || c > 9 {
		t.Fatalf("c(1e12) = %g, expected ≈ 8.4", c)
	}
}
