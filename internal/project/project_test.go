package project

import (
	"encoding/json"
	"strings"
	"testing"

	"galaxy/internal/budget"
	"galaxy/internal/galaxy"
	"galaxy/internal/presets"
)

// AC-11: save and load restore fractions, seed and UI state.
func TestRoundTrip(t *testing.T) {
	p, _ := presets.ByID("m31-normalized")
	m, err := galaxy.NewModel(p, 7)
	if err != nil {
		t.Fatal(err)
	}
	_ = m.Edit(func(b *budget.Budget) error { return b.SetFraction("stars", 0.12, budget.Proportional) })
	_ = m.Edit(func(b *budget.Budget) error { return b.SetTotalMass(1.5e12) })
	ui := json.RawMessage(`{"camera":{"position":[1,2,3]},"hidden":["dm.halo"]}`)
	f := New(m, "test", nil, ui)
	data, _ := json.Marshal(f)

	g, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	m2, _ := galaxy.NewModel(p, 1)
	if err := m2.Replace(g.Budget, g.Seed); err != nil {
		t.Fatal(err)
	}
	if m2.Seed() != 7 {
		t.Fatalf("seed not restored")
	}
	b1, b2 := m.Budget(), m2.Budget()
	if b1.TotalMass != b2.TotalMass || b1.CategoryFraction("stars") != b2.CategoryFraction("stars") {
		t.Fatalf("budget not restored")
	}
	if string(g.UI) != string(ui) {
		t.Fatalf("UI state not restored: %s", g.UI)
	}
	csv, err := MassCSV(m2.Snapshot(), nil, nil)
	if err != nil || !strings.Contains(string(csv), "stars.thin_disk") {
		t.Fatalf("csv: %v", err)
	}
}

func TestRejectsForeignFiles(t *testing.T) {
	if _, err := Parse([]byte(`{"schema":"other"}`)); err == nil {
		t.Fatal("foreign schema accepted")
	}
}
