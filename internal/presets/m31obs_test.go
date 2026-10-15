package presets

import (
	"math"
	"testing"

	"galaxy/internal/budget"
	"galaxy/internal/galaxy"
	"galaxy/internal/obsdata"
	"galaxy/internal/units"
)

func TestM31ObsHaloConversion(t *testing.T) {
	p := NewM31Obs()
	rs, r200, m200, c200 := p.HaloConversion()
	// Definition of M200c: mean density inside R200c is 200 ρcrit.
	mean := m200 / (4 * math.Pi / 3 * r200 * r200 * r200)
	if math.Abs(mean/(200*units.Planck18.RhoCrit())-1) > 1e-6 {
		t.Fatalf("mean density %g is not 200 ρcrit", mean)
	}
	t.Logf("Corbelli NFW: rs = %.2f kpc → R200c = %.1f kpc, M200c = %.3g M☉, c200c = %.2f", rs, r200, m200, c200)
	if r200 < 200 || r200 > 240 || m200 < 1.0e12 || m200 > 1.35e12 {
		t.Fatalf("implausible conversion")
	}
	m, err := galaxy.NewModel(p, 1)
	if err != nil {
		t.Fatal(err)
	}
	s := m.Snapshot()
	if math.Abs(s.R200c/r200-1) > 1e-6 {
		t.Fatalf("model R200c %g ≠ conversion %g", s.R200c, r200)
	}
}

// The H I mass from integrating the tabulated Σ_HI is compared with the
// flux-based total of Chemin+2009 (4.23×10⁹ M☉, which also counts gas
// outside the regular disc).
func TestM31ObsHIConsistency(t *testing.T) {
	p := NewM31Obs()
	r, s := p.sigmaHI()
	mass := 0.0
	for i := 1; i < len(r); i++ {
		R := 0.5 * (r[i] + r[i-1])
		mass += 2 * math.Pi * R * (r[i] - r[i-1]) * 0.5 * (s[i] + s[i-1]) * 1e6 // M☉/pc² → M☉/kpc²
	}
	t.Logf("∫Σ_HI dA = %.3g M☉ vs flux-based 4.23e9", mass)
	if mass < 2.5e9 || mass > 5e9 {
		t.Fatalf("Σ_HI integral %.3g far from the flux-based mass", mass)
	}
}

// Physics check: the model's circular velocity against the measured H I
// rotation curves and the dynamical mass estimates.
func TestM31ObsAgainstData(t *testing.T) {
	p := NewM31Obs()
	m, err := galaxy.NewModel(p, 1)
	if err != nil {
		t.Fatal(err)
	}
	prof := m.Profiles(galaxy.ProfileRequest{RMin: 0.5, RMax: 600, N: 400})
	vc := prof.Vc[0].Y
	menc := prof.Enclosed[0].Y
	at := func(y []float64, r float64) float64 {
		for i := 1; i < len(prof.R); i++ {
			if prof.R[i] >= r {
				t0 := (r - prof.R[i-1]) / (prof.R[i] - prof.R[i-1])
				return y[i-1] + t0*(y[i]-y[i-1])
			}
		}
		return y[len(y)-1]
	}
	for _, d := range p.Datasets() {
		var chi2, rel float64
		n := 0
		for _, pt := range d.Points {
			var model float64
			switch d.Quantity {
			case obsdata.CircularVelocity:
				if pt.R < 8 || pt.R > 30 {
					continue
				}
				model = at(vc, pt.R)
			case obsdata.EnclosedMass:
				model = at(menc, pt.R)
			default:
				continue
			}
			err := pt.ErrHi
			if model < pt.V {
				err = pt.ErrLo
			}
			if err > 0 {
				chi2 += math.Pow((model-pt.V)/err, 2)
			}
			rel += math.Abs(model/pt.V - 1)
			n++
		}
		if n == 0 {
			continue
		}
		t.Logf("%-40s n=%3d  mean |model/data − 1| = %.3f  χ²/n = %.2f", d.Name, n, rel/float64(n), chi2/float64(n))
		if rel/float64(n) > 0.20 {
			t.Errorf("%s: model deviates by %.0f%% on average", d.Name, 100*rel/float64(n))
		}
	}
}

// Editing the budget of an observational preset marks it as modified.
func TestM31ObsModifiedFlag(t *testing.T) {
	m, _ := galaxy.NewModel(NewM31Obs(), 1)
	if m.Snapshot().Modified {
		t.Fatal("fresh preset reported as modified")
	}
	_ = m.Edit(func(b *budget.Budget) error { return b.SetTotalMass(2e12) })
	if !m.Snapshot().Modified {
		t.Fatal("edited preset not reported as modified")
	}
	m.Undo()
	if m.Snapshot().Modified {
		t.Fatal("undo back to defaults still modified")
	}
}

func TestM31ObsConcentrationShown(t *testing.T) {
	p := NewM31Obs()
	_, _, _, c := p.HaloConversion()
	m, _ := galaxy.NewModel(p, 1)
	if got := m.Snapshot().Concentration; math.Abs(got-c) > 1e-9 {
		t.Fatalf("state shows c200c = %g, preset halo has %g", got, c)
	}
}
