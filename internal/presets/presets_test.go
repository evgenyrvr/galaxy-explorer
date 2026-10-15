package presets

import (
	"math"
	"testing"

	"galaxy/internal/budget"
	"galaxy/internal/galaxy"
	"galaxy/internal/physics"
	"galaxy/internal/units"
)

func newModel(t *testing.T, id string) *galaxy.Model {
	t.Helper()
	p, err := ByID(id)
	if err != nil {
		t.Fatal(err)
	}
	m, err := galaxy.NewModel(p, 42)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func near(a, b float64) bool { return math.Abs(a-b) <= 1e-6*math.Max(math.Abs(a), math.Abs(b)) }

func TestDefaultBudgetsComplete(t *testing.T) {
	for _, p := range All() {
		b := p.DefaultBudget()
		for _, is := range b.Validate() {
			t.Errorf("%s: %s %s", p.Info().ID, is.Severity, is.Message)
		}
		m := newModel(t, p.Info().ID)
		s := m.Snapshot()
		sum := 0.0
		for _, c := range s.Components {
			sum += c.Mass
		}
		if !near(sum, s.TotalMass) {
			t.Errorf("%s: Σ component masses %g ≠ M_total %g", p.Info().ID, sum, s.TotalMass)
		}
		if s.BaryonFraction > s.CosmicBaryonFraction {
			t.Errorf("%s: baryon fraction %.3f exceeds cosmic %.3f", p.Info().ID, s.BaryonFraction, s.CosmicBaryonFraction)
		}
	}
}

// Every component must lie inside R200c for the M200c bookkeeping to hold.
func TestComponentsInsideR200c(t *testing.T) {
	for _, p := range All() {
		m := newModel(t, p.Info().ID)
		s := m.Snapshot()
		for _, c := range s.Components {
			if c.InsideR200c < 0.999 {
				t.Errorf("%s/%s: only %.4f of the mass inside R200c", p.Info().ID, c.ID, c.InsideR200c)
			}
		}
	}
}

// AC-02 / AC-03: thin disc alone gives its mass; adding arm 1 adds nothing.
func TestAC03ArmsNotDoubleCounted(t *testing.T) {
	m := newModel(t, "m31-normalized")
	disc := m.SelectionMass(galaxy.Selection{Components: []string{"stars.thin_disk"}})
	want := m.Budget().ComponentMass("stars", "stars.thin_disk")
	if !near(disc.Mass, want) {
		t.Fatalf("thin disc selection %g, want %g", disc.Mass, want)
	}
	both := m.SelectionMass(galaxy.Selection{Components: []string{"stars.thin_disk"}, Tags: []string{"arm_1"}})
	gasArm := 0.0
	for _, p := range both.Parts {
		if p.Component != "stars.thin_disk" {
			gasArm += p.Mass
		}
	}
	if !near(both.Mass-gasArm, disc.Mass) {
		t.Fatalf("thin disc + arm 1 counted stellar mass twice: %g vs %g", both.Mass-gasArm, disc.Mass)
	}
	arm := m.SelectionMass(galaxy.Selection{Tags: []string{"arm_1"}})
	if arm.Mass <= 0 || arm.Mass >= m.Budget().TotalMass*m.Budget().CategoryFraction("stars") {
		t.Fatalf("arm mass out of range: %g", arm.Mass)
	}
	// Both arms together never exceed the owners.
	arms := m.SelectionMass(galaxy.Selection{Tags: []string{"arm_1", "arm_2"}})
	for _, p := range arms.Parts {
		if p.Fraction > 1 {
			t.Fatalf("%s fraction %g > 1", p.Component, p.Fraction)
		}
	}
}

// AC-07: satellites are inside R200c and part of M200c; group sums are single-counted.
func TestAC07Satellites(t *testing.T) {
	m := newModel(t, "m31-normalized")
	all := m.SelectionMass(galaxy.Selection{Tags: []string{"satellites"}})
	one := m.SelectionMass(galaxy.Selection{Tags: []string{"satellite_m32"}})
	two := m.SelectionMass(galaxy.Selection{Tags: []string{"satellite_m32", "satellites"}, Components: []string{"stars.m32"}})
	if !near(all.Mass, two.Mass) {
		t.Fatalf("overlapping satellite selections double counted: %g vs %g", two.Mass, all.Mass)
	}
	if one.Mass >= all.Mass {
		t.Fatalf("M32 ≥ all satellites")
	}
	if !near(all.InsideR200c, all.Mass) {
		t.Fatalf("satellite mass partly outside R200c: %g of %g", all.InsideR200c, all.Mass)
	}
}

// AC-08: the hot halo selection contains only the hot phase.
func TestAC08HotOnly(t *testing.T) {
	m := newModel(t, "m31-normalized")
	r := m.SelectionMass(galaxy.Selection{Components: []string{"gas.hot.cgm"}})
	for k := range r.ByMatter {
		if k != "gas.hot_ionized" {
			t.Fatalf("hot halo selection includes %s", k)
		}
	}
}

// AC-06 and AC-15: masses follow the budget; markers do not change physics.
func TestAC06AndAC15(t *testing.T) {
	m := newModel(t, "m31-normalized")
	before := m.Snapshot()
	if err := m.Edit(func(b *budget.Budget) error { return b.SetTotalMass(2e12) }); err != nil {
		t.Fatal(err)
	}
	after := m.Snapshot()
	for i := range before.Components {
		if !near(after.Components[i].Mass, 2*before.Components[i].Mass) {
			t.Fatalf("%s did not scale", before.Components[i].ID)
		}
	}
	if after.R200c <= before.R200c {
		t.Fatalf("R200c must grow with M200c")
	}
	// R200c-dependent geometry gets a new key; fixed geometry keeps its key.
	for i := range before.Components {
		b, a := before.Components[i], after.Components[i]
		if b.ID == "stars.thin_disk" && a.GeomKey != b.GeomKey {
			t.Fatalf("thin disc geometry should be fixed")
		}
		if b.ID == "dm.halo" && a.GeomKey == b.GeomKey {
			t.Fatalf("dark halo must follow R200c")
		}
	}
	// Different marker counts, same mass.
	if _, err := m.Particles("stars.thin_disk", 1000); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Particles("stars.thin_disk", 50000); err != nil {
		t.Fatal(err)
	}
	if !near(m.Snapshot().Components[0].Mass, after.Components[0].Mass) {
		t.Fatalf("marker sampling changed the mass")
	}
}

func TestParticlesBuffer(t *testing.T) {
	m := newModel(t, "m31-normalized")
	buf, err := m.Particles("stars.globular_clusters", 99999)
	if err != nil {
		t.Fatal(err)
	}
	n := int(buf[0]) | int(buf[1])<<8 | int(buf[2])<<16 | int(buf[3])<<24
	if n != 450 {
		t.Fatalf("discrete clusters must be drawn exactly once, got %d", n)
	}
	if len(buf) != 4+16*n {
		t.Fatalf("buffer length %d", len(buf))
	}
}

func TestStreamAndShellsExtent(t *testing.T) {
	for _, tc := range []struct{ preset, comp string }{{"m31-normalized", "stars.giant_stream"}, {"m87-normalized", "stars.shells"}} {
		m := newModel(t, tc.preset)
		for _, c := range m.Snapshot().Components {
			if c.ID != tc.comp {
				continue
			}
			t.Logf("%s: max radius %.1f kpc, inside R200c %.4f", c.ID, c.MaxRadius, c.InsideR200c)
			if c.MaxRadius < 10 {
				t.Errorf("%s collapsed to the centre", c.ID)
			}
		}
	}
}

func TestProfilesUnits(t *testing.T) {
	m := newModel(t, "m31-normalized")
	p := m.Profiles(galaxy.ProfileRequest{})
	last := p.Enclosed[0].Y[len(p.R)-1]
	if !near(last, m.Budget().TotalMass) {
		t.Fatalf("M(<R200c) = %g, want M200c", last)
	}
	// Flat-ish rotation curve of a few hundred km/s for an M31-like galaxy.
	vmax := 0.0
	for i, r := range p.R {
		if r > 5 && r < 40 {
			vmax = math.Max(vmax, p.Vc[0].Y[i])
		}
	}
	if vmax < 180 || vmax > 320 {
		t.Fatalf("Vmax = %.0f km/s, outside the plausible range", vmax)
	}
	t.Logf("M31-inspired Vmax(5–40 kpc) = %.0f km/s", vmax)
	_ = physics.Vec3{}
}

// Region selections: analytic sphere/cylinder masses agree with the
// reference-sample estimate and never exceed the owners.
func TestRegionSelection(t *testing.T) {
	m := newModel(t, "m31-normalized")
	b := m.Budget()
	sphere := &galaxy.Region{Kind: "sphere", RMin: 0, RMax: 10}
	r := m.SelectionMass(galaxy.Selection{Components: []string{"stars.thin_disk"}, Region: sphere})
	if !r.Exact || r.Mass <= 0 || r.Mass >= b.ComponentMass("stars", "stars.thin_disk") {
		t.Fatalf("thin disc inside 10 kpc: %g (exact=%v)", r.Mass, r.Exact)
	}
	// Exponential disc: fraction inside R is 1 − (1 + x)e^−x, x = R/Rd (thin disc ⇒ sphere ≈ cylinder).
	x := 10 / 5.3
	want := (1 - (1+x)*math.Exp(-x)) / (1 - (1+32/5.3)*math.Exp(-32/5.3))
	if got := r.Parts[0].Fraction; math.Abs(got-want) > 0.01 {
		t.Fatalf("thin disc fraction inside 10 kpc = %.4f, want ≈ %.4f", got, want)
	}
	cyl := &galaxy.Region{Kind: "cylinder", RMin: 0, RMax: 10, ZMax: 0.5}
	c := m.SelectionMass(galaxy.Selection{Components: []string{"stars.thin_disk"}, Region: cyl})
	if c.Mass >= r.Mass {
		t.Fatalf("|z|<0.5 kpc slab must hold less than the full cylinder")
	}
	// Arm inside a region: sampled estimate, bounded by the whole arm.
	armAll := m.SelectionMass(galaxy.Selection{Tags: []string{"arm_1"}})
	armIn := m.SelectionMass(galaxy.Selection{Tags: []string{"arm_1"}, Region: sphere})
	if armIn.Exact || armIn.Mass <= 0 || armIn.Mass > armAll.Mass {
		t.Fatalf("arm in region %g vs arm %g", armIn.Mass, armAll.Mass)
	}
	// Region alone selects every component; a sphere of R200c holds M200c.
	all := m.SelectionMass(galaxy.Selection{Region: &galaxy.Region{Kind: "sphere", RMax: m.Snapshot().R200c * 1.0001}})
	if !near(all.Mass, b.TotalMass) && math.Abs(all.Mass/b.TotalMass-1) > 1e-3 {
		t.Fatalf("sphere of R200c holds %g of %g", all.Mass, b.TotalMass)
	}
}

// The central black hole is a point mass: M(<r) constant, Vc ∝ r^−1/2,
// no finite density — never a halo-like profile.
func TestBlackHoleProfiles(t *testing.T) {
	for _, id := range []string{"m31-normalized", "m31-observational", "m87-normalized"} {
		m := newModel(t, id)
		p := m.Profiles(galaxy.ProfileRequest{Components: []string{"bh.central"}})
		mbh := m.Budget().ComponentMass("black_hole", "bh.central")
		var enc, vc, rho []float64
		for _, s := range p.Enclosed {
			if s.ID == "bh.central" {
				enc = s.Y
			}
		}
		for _, s := range p.Vc {
			if s.ID == "bh.central" {
				vc = s.Y
			}
		}
		for _, s := range p.Density {
			if s.ID == "bh.central" {
				rho = s.Y
			}
		}
		for i, r := range p.R {
			if !near(enc[i], mbh) {
				t.Fatalf("%s: M_BH(<%.3g) = %g, want constant %g", id, r, enc[i], mbh)
			}
			if want := math.Sqrt(units.G * mbh / r); !near(vc[i], want) {
				t.Fatalf("%s: Vc_BH(%.3g) = %g, want Keplerian %g", id, r, vc[i], want)
			}
			if rho[i] != 0 {
				t.Fatalf("%s: point mass has finite density %g at r = %g", id, rho[i], r)
			}
		}
	}
}
