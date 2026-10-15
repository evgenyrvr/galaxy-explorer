package obsdata

import (
	"math"
	"os"
	"strings"
	"testing"
)

const sigmaCSV = `# quantity: surface_density
# unit: Msun/kpc2
# radius_unit: pc
# evidence: ILLUSTRATIVE
# source: synthetic test file
r,value,err
500,2e8,1e7
1000,1e8,
2000,5e7,5e6
`

// AC-10: Σ(R) stays Σ, in M☉/pc², with no point added at the centre.
func TestAC10SurfaceDensityImport(t *testing.T) {
	d, err := Parse("test", []byte(sigmaCSV))
	if err != nil {
		t.Fatal(err)
	}
	if d.Quantity != SurfaceDensity || d.Unit != "Msun/pc2" {
		t.Fatalf("quantity %s unit %s", d.Quantity, d.Unit)
	}
	if len(d.Points) != 3 || d.Points[0].R != 0.5 {
		t.Fatalf("points %+v", d.Points)
	}
	if math.Abs(d.Points[0].V-200) > 1e-9 || math.Abs(d.Points[0].Err-10) > 1e-9 || d.Points[1].Err != 0 {
		t.Fatalf("unit conversion wrong: %+v", d.Points)
	}
}

func TestRejectsCrossQuantityUnits(t *testing.T) {
	bad := strings.Replace(sigmaCSV, "Msun/kpc2", "Msun/pc3", 1)
	if _, err := Parse("bad", []byte(bad)); err == nil || !strings.Contains(err.Error(), "never converted") {
		t.Fatalf("a volume-density unit was accepted for Σ: %v", err)
	}
}

func TestRequiresProvenance(t *testing.T) {
	for _, drop := range []string{"# source: synthetic test file\n", "# evidence: ILLUSTRATIVE\n"} {
		if _, err := Parse("x", []byte(strings.Replace(sigmaCSV, drop, "", 1))); err == nil {
			t.Errorf("accepted a file without %q", strings.TrimSpace(drop))
		}
	}
}

func TestArcsecNeedsDistance(t *testing.T) {
	csv := "# quantity: circular_velocity\n# unit: km/s\n# radius_unit: arcsec\n# evidence: OBSERVED\n# source: s\n100,200\n"
	if _, err := Parse("x", []byte(csv)); err == nil {
		t.Fatal("arcsec without distance accepted")
	}
	d, err := Parse("x", []byte(strings.Replace(csv, "# source: s", "# source: s\n# distance_kpc: 785", 1)))
	if err != nil {
		t.Fatal(err)
	}
	// 100″ at 785 kpc ≈ 0.3806 kpc.
	if math.Abs(d.Points[0].R-0.38058) > 1e-4 {
		t.Fatalf("r = %g", d.Points[0].R)
	}
}

func TestRejectsBadRows(t *testing.T) {
	for _, body := range []string{"0,1", "-1,1", "1,-5", "2,1\n1,1", "1,1,-1"} {
		csv := "# quantity: surface_density\n# unit: Msun/pc2\n# evidence: OBSERVED\n# source: s\n" + body + "\n"
		if _, err := Parse("x", []byte(csv)); err == nil {
			t.Errorf("accepted %q", body)
		}
	}
}

// The shipped example file must stay importable.
func TestExampleFile(t *testing.T) {
	data, err := os.ReadFile("../../docs/examples/synthetic-surface-density.csv")
	if err != nil {
		t.Fatal(err)
	}
	d, err := Parse("example", data)
	if err != nil {
		t.Fatal(err)
	}
	if d.Evidence != "ILLUSTRATIVE" || len(d.Points) != 8 {
		t.Fatalf("example: %s, %d points", d.Evidence, len(d.Points))
	}
}
