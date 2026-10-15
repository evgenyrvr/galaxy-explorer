package physics

import (
	"math"
	"math/rand"
	"testing"

	"galaxy/internal/units"
)

func TestHernquistMatchesAnalytic(t *testing.T) {
	a, mt := 1.0, 50.0
	s := NewSpheroid(Hernquist(a), mt, SpheroidOptions{})
	norm := mt * mt / ((mt + a) * (mt + a))
	for _, r := range []float64{0.1, 0.5, 1, 3, 10, 30} {
		want := r * r / ((r + a) * (r + a)) / norm
		got := s.SphericalFraction(r)
		if math.Abs(got-want) > 1e-5 {
			t.Errorf("r=%g: got %.8f want %.8f", r, got, want)
		}
	}
	if s.SphericalFraction(mt) != 1 {
		t.Errorf("fraction at truncation must be 1")
	}
}

func TestNFWMatchesAnalytic(t *testing.T) {
	rs, c := 25.0, 8.4
	r200 := rs * c
	s := NewSpheroid(NFW(rs, c), r200, SpheroidOptions{})
	m := func(x float64) float64 { return math.Log(1+x) - x/(1+x) }
	for _, r := range []float64{1, 10, 50, 150} {
		want := m(r/rs) / m(c)
		if got := s.SphericalFraction(r); math.Abs(got-want) > 1e-5 {
			t.Errorf("r=%g: got %.7f want %.7f", r, got, want)
		}
	}
}

func TestFlattenedSpheroidSphericalFraction(t *testing.T) {
	s := NewSpheroid(Hernquist(2), 40, SpheroidOptions{P: 0.8, Q: 0.5})
	rng := rand.New(rand.NewSource(1))
	n := 200000
	pos := make([]float32, 3*n)
	tags := make([]uint8, n)
	s.Sample(rng, pos, tags)
	for _, r := range []float64{1, 3, 8} {
		cnt := 0
		for i := 0; i < n; i++ {
			x, y, z := float64(pos[3*i]), float64(pos[3*i+1]), float64(pos[3*i+2])
			if x*x+y*y+z*z < r*r {
				cnt++
			}
		}
		mc := float64(cnt) / float64(n)
		if got := s.SphericalFraction(r); math.Abs(got-mc) > 0.005 {
			t.Errorf("r=%g: quadrature %.4f vs Monte Carlo %.4f", r, got, mc)
		}
	}
}

func TestSpheroidSplitTags(t *testing.T) {
	re := 4.0
	s := NewSpheroid(Hernquist(re/1.8153), 80, SpheroidOptions{Splits: []float64{re}, SplitTags: []string{"inner", "outer"}})
	if f := s.TagFraction(1) + s.TagFraction(2); math.Abs(f-1) > 1e-12 {
		t.Fatalf("inner+outer = %g, want 1", f)
	}
}

func TestDiscNormalisationAndArms(t *testing.T) {
	arms := &ArmPattern{M: 2, PitchDeg: 12, Amplitude: 0.6, HalfWidth: 0.7, Rin: 3, Rout: 20, R0: 3}
	d := NewDisc(DiscOptions{Shape: Exponential, Rd: 5, Z0: 0.3, RMax: 30, Arms: arms})
	if f := d.CylindricalFraction(30); math.Abs(f-1) > 1e-9 {
		t.Fatalf("cylindrical fraction at Rmax = %g", f)
	}
	if f := d.SphericalFraction(d.MaxRadius()); math.Abs(f-1) > 1e-4 {
		t.Fatalf("spherical fraction at max radius = %g", f)
	}
	// Σ integrates to 1.
	sum, h := 0.0, 30.0/20000
	for i := 0; i < 20000; i++ {
		R := (float64(i) + 0.5) * h
		sum += 2 * math.Pi * R * d.SurfaceDensity(R) * h
	}
	if math.Abs(sum-1) > 1e-4 {
		t.Fatalf("∫Σ dA = %g", sum)
	}
	// Analytic arm fractions agree with sampling.
	rng := rand.New(rand.NewSource(7))
	n := 400000
	pos := make([]float32, 3*n)
	tags := make([]uint8, n)
	d.Sample(rng, pos, tags)
	counts := make([]int, 3)
	for _, tg := range tags {
		counts[tg]++
	}
	for k := 1; k <= 2; k++ {
		mc := float64(counts[k]) / float64(n)
		if got := d.TagFraction(k); math.Abs(got-mc) > 0.004 {
			t.Errorf("arm %d: analytic %.4f vs sampled %.4f", k, got, mc)
		}
	}
}

func TestArmsConserveAzimuthalAverage(t *testing.T) {
	// The radial distribution of a disc with arms equals that without arms.
	arms := &ArmPattern{M: 2, PitchDeg: 10, Amplitude: 0.9, HalfWidth: 0.5, Rin: 2, Rout: 25, R0: 2}
	with := NewDisc(DiscOptions{Rd: 5, Z0: 0.3, RMax: 30, Arms: arms})
	without := NewDisc(DiscOptions{Rd: 5, Z0: 0.3, RMax: 30})
	for _, R := range []float64{1, 5, 10, 20} {
		if with.CylindricalFraction(R) != without.CylindricalFraction(R) {
			t.Fatalf("arms changed the radial mass profile at R=%g", R)
		}
	}
}

func TestFreemanRotationCurvePeak(t *testing.T) {
	rd := 5.0
	d := NewDisc(DiscOptions{Rd: rd, Z0: 0.3, RMax: 60})
	best, bestR := 0.0, 0.0
	for R := 0.5; R < 30; R += 0.01 {
		v := d.MidplaneVc2PerGM(R)
		if v > best {
			best, bestR = v, R
		}
	}
	// Freeman disc: peak at R ≈ 2.15 Rd with Vc² ≈ 0.3870 G M / Rd.
	if math.Abs(bestR/rd-2.15) > 0.03 {
		t.Errorf("peak at %.3f Rd", bestR/rd)
	}
	if math.Abs(best*rd-0.387) > 0.005 {
		t.Errorf("peak Vc²/(GM/Rd) = %.4f", best*rd)
	}
}

func TestLeapfrogConservesEnergy(t *testing.T) {
	m := 1e11
	menc := func(float64) float64 { return m }
	p := Phase{X: Vec3{20, 0, 0}, V: Vec3{0, 80, 10}}
	energy := func(p Phase) float64 { return 0.5*p.V.Dot(p.V) - units.G*m/p.X.Norm() }
	e0 := energy(p)
	p1 := IntegrateOrbit(menc, p, 5, 0.002, 0)
	if rel := math.Abs((energy(p1) - e0) / e0); rel > 1e-3 {
		t.Fatalf("relative energy drift %g", rel)
	}
}
