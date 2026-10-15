package budget

import (
	"errors"
	"math"
	"testing"
)

// specTable is the control example of TECHNICAL_SPECIFICATION.md §5.1.
func specTable() Budget {
	return Budget{
		TotalMass: 1e12,
		Mode:      FixTotal,
		Categories: []Category{
			{ID: "dark_matter", Fraction: 0.85},
			{ID: "stars", Fraction: 0.05},
			{ID: "gas.cold", Fraction: 0.015},
			{ID: "gas.warm_ionized", Fraction: 0.03},
			{ID: "gas.hot_ionized", Fraction: 0.0549},
			{ID: "black_hole", Fraction: 0.0001},
		},
		Shares: map[string][]Share{
			"stars": {{Component: "thin", Share: 0.6}, {Component: "thick", Share: 0.1}, {Component: "bulge", Share: 0.3}},
		},
	}
}

func near(a, b float64) bool { return math.Abs(a-b) <= 1e-6*math.Max(1, math.Abs(b)) }

func TestSpecTableMasses(t *testing.T) {
	b := specTable()
	if !b.Complete() {
		t.Fatalf("spec table must sum to 100%%, unallocated %g", b.Unallocated())
	}
	want := map[string]float64{
		"dark_matter": 8.5e11, "stars": 5e10, "gas.cold": 1.5e10,
		"gas.warm_ionized": 3e10, "gas.hot_ionized": 5.49e10, "black_hole": 1e8,
	}
	for id, m := range want {
		if !near(b.CategoryMass(id), m) {
			t.Errorf("%s: %g want %g", id, b.CategoryMass(id), m)
		}
	}
	if !near(b.ComponentMass("stars", "thin"), 3e10) {
		t.Errorf("thin disc mass %g", b.ComponentMass("stars", "thin"))
	}
}

// AC-05: DM → 80%, stars → 15% at fixed total mass.
func TestAC05RemainderIsExplicit(t *testing.T) {
	b := specTable()
	// Leaving others untouched would exceed 100%: must be refused, not silently accepted.
	if err := b.SetFraction("dark_matter", 0.80, LeaveUnallocated); err != nil {
		t.Fatal(err)
	}
	if !near(b.Unallocated(), 0.05) {
		t.Fatalf("after DM=80%% unallocated = %g, want 5%%", b.Unallocated())
	}
	if err := b.SetFraction("stars", 0.15, LeaveUnallocated); !errors.Is(err, ErrOverAllocated) {
		t.Fatalf("expected over-allocation error, got %v", err)
	}
	// With proportional redistribution and DM locked, the remaining 5% is
	// distributed over gas and BH in proportion to their previous sizes.
	b = specTable()
	_ = b.SetFraction("dark_matter", 0.80, Proportional)
	_ = b.SetLocked("dark_matter", true)
	if err := b.SetFraction("stars", 0.15, Proportional); err != nil {
		t.Fatal(err)
	}
	if !b.Complete() {
		t.Fatalf("budget incomplete: %g", b.Unallocated())
	}
	rest := 1 - b.CategoryFraction("dark_matter") - b.CategoryFraction("stars")
	if !near(rest, 0.05) {
		t.Fatalf("remaining %g, want 5%%", rest)
	}
	// Ratios between gas phases preserved.
	r0 := 0.015 / 0.03
	r1 := b.CategoryFraction("gas.cold") / b.CategoryFraction("gas.warm_ionized")
	if !near(r0, r1) {
		t.Fatalf("proportional rule changed ratios: %g vs %g", r0, r1)
	}
}

// AC-06: doubling M_total keeps fractions and scales masses.
func TestAC06TotalMassScales(t *testing.T) {
	b := specTable()
	before := b.ComponentMass("stars", "bulge")
	if err := b.SetTotalMass(2e12); err != nil {
		t.Fatal(err)
	}
	if !near(b.ComponentMass("stars", "bulge"), 2*before) {
		t.Fatalf("bulge mass did not scale")
	}
	if !near(b.CategoryFraction("stars"), 0.05) {
		t.Fatalf("fraction changed")
	}
}

// AC-14: negative masses and >100% are blocked.
func TestAC14Validation(t *testing.T) {
	b := specTable()
	if err := b.SetFraction("stars", -0.1, LeaveUnallocated); !errors.Is(err, ErrNegative) {
		t.Errorf("negative fraction: %v", err)
	}
	if err := b.SetCategoryMass("stars", -5); !errors.Is(err, ErrNegative) {
		t.Errorf("negative mass: %v", err)
	}
	if err := b.SetFraction("stars", 0.2, LeaveUnallocated); !errors.Is(err, ErrOverAllocated) {
		t.Errorf("sum > 100%%: %v", err)
	}
	if err := b.SetTotalMass(0); !errors.Is(err, ErrBadTotal) {
		t.Errorf("zero total: %v", err)
	}
	if err := b.SetFraction("stars", 1.5, Proportional); !errors.Is(err, ErrOverAllocated) {
		t.Errorf(">100%% single: %v", err)
	}
	if !b.Complete() {
		t.Errorf("failed edits must leave the budget unchanged")
	}
}

func TestFixComponentsMode(t *testing.T) {
	b := specTable()
	if err := b.SetMode(FixComponents); err != nil {
		t.Fatal(err)
	}
	dm := b.CategoryMass("dark_matter")
	if err := b.SetCategoryMass("stars", 1e11); err != nil {
		t.Fatal(err)
	}
	if !near(b.CategoryMass("dark_matter"), dm) {
		t.Fatalf("other masses must be fixed")
	}
	if !near(b.TotalMass, 1e12+5e10) {
		t.Fatalf("total %g", b.TotalMass)
	}
	if !b.Complete() {
		t.Fatalf("fractions must sum to 1")
	}
	// Setting a fraction keeps the others' masses too.
	if err := b.SetFraction("stars", 0.2, LeaveUnallocated); err != nil {
		t.Fatal(err)
	}
	if !near(b.CategoryFraction("stars"), 0.2) || !near(b.CategoryMass("dark_matter"), dm) {
		t.Fatalf("fix_components SetFraction wrong: f=%g", b.CategoryFraction("stars"))
	}
}

func TestNormalizeAndRemainder(t *testing.T) {
	b := specTable()
	_ = b.SetFraction("dark_matter", 0.5, LeaveUnallocated)
	_ = b.SetLocked("black_hole", true)
	if err := b.Normalize(); err != nil {
		t.Fatal(err)
	}
	if !b.Complete() || !near(b.CategoryFraction("black_hole"), 0.0001) {
		t.Fatalf("normalize failed")
	}
	b = specTable()
	_ = b.SetFraction("dark_matter", 0.8, LeaveUnallocated)
	if err := b.AssignRemainder("gas.hot_ionized"); err != nil {
		t.Fatal(err)
	}
	if !b.Complete() || !near(b.CategoryFraction("gas.hot_ionized"), 0.1049) {
		t.Fatalf("assign remainder failed: %g", b.CategoryFraction("gas.hot_ionized"))
	}
}

func TestSharesStayNormalised(t *testing.T) {
	b := specTable()
	_ = b.SetShareLocked("stars", "bulge", true)
	if err := b.SetShare("stars", "thin", 0.5); err != nil {
		t.Fatal(err)
	}
	if !near(b.ShareOf("stars", "bulge"), 0.3) || !near(b.ShareOf("stars", "thick"), 0.2) {
		t.Fatalf("shares: thick %g bulge %g", b.ShareOf("stars", "thick"), b.ShareOf("stars", "bulge"))
	}
	if err := b.SetShare("stars", "thin", 0.9); !errors.Is(err, ErrNoRoom) {
		t.Fatalf("locked share exceeded: %v", err)
	}
	for _, is := range b.Validate() {
		if is.Severity == "error" {
			t.Fatal(is.Message)
		}
	}
}

func TestHistory(t *testing.T) {
	h := NewHistory(10)
	b := specTable()
	h.Push(b)
	_ = b.SetFraction("stars", 0.01, LeaveUnallocated)
	b, ok := h.Undo(b)
	if !ok || !near(b.CategoryFraction("stars"), 0.05) {
		t.Fatalf("undo failed")
	}
	b, ok = h.Redo(b)
	if !ok || !near(b.CategoryFraction("stars"), 0.01) {
		t.Fatalf("redo failed")
	}
}
