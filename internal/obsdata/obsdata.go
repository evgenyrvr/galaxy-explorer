// Package obsdata imports radial profiles measured or inferred from
// observations. Values are kept in their physical quantity: a surface
// density is never converted into a volume density, and no points are
// added (in particular none at r = 0).
package obsdata

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
)

// Quantity is the physical quantity of a profile.
type Quantity string

const (
	SurfaceDensity   Quantity = "surface_density"   // Σ(R), canonical M☉/pc²
	VolumeDensity    Quantity = "volume_density"    // ρ(r), canonical M☉/pc³
	CircularVelocity Quantity = "circular_velocity" // Vc(r) or rotation curve, km/s
	EnclosedMass     Quantity = "enclosed_mass"     // M(<r), M☉
)

// CanonicalUnit is the unit profiles are stored and plotted in.
var CanonicalUnit = map[Quantity]string{
	SurfaceDensity:   "Msun/pc2",
	VolumeDensity:    "Msun/pc3",
	CircularVelocity: "km/s",
	EnclosedMass:     "Msun",
}

// unitFactors converts accepted units to the canonical unit of each quantity.
var unitFactors = map[Quantity]map[string]float64{
	SurfaceDensity:   {"msun/pc2": 1, "msun/kpc2": 1e-6},
	VolumeDensity:    {"msun/pc3": 1, "msun/kpc3": 1e-9},
	CircularVelocity: {"km/s": 1},
	EnclosedMass:     {"msun": 1, "1e10msun": 1e10},
}

// Point is one measurement with 1σ errors below and above the value
// (canonical unit; 0 = no error given). Label carries per-row text such as
// the original reference or tracer.
type Point struct {
	R     float64 `json:"r"` // kpc
	V     float64 `json:"v"`
	ErrLo float64 `json:"errLo"`
	ErrHi float64 `json:"errHi"`
	Label string  `json:"label,omitempty"`
}

// Dataset is an imported profile with provenance.
type Dataset struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Quantity     Quantity `json:"quantity"`
	Unit         string   `json:"unit"`         // canonical
	OriginalUnit string   `json:"originalUnit"` // as written in the file
	RadiusUnit   string   `json:"radiusUnit"`   // as written in the file
	DistanceKpc  float64  `json:"distanceKpc,omitempty"`
	Evidence     string   `json:"evidence"` // OBSERVED | INFERRED | RECONSTRUCTED | ILLUSTRATIVE
	Source       string   `json:"source"`
	URL          string   `json:"url,omitempty"`
	Notes        string   `json:"notes,omitempty"`
	Points       []Point  `json:"points"`
}

// arcsecToRad converts arcseconds to radians.
const arcsecToRad = math.Pi / (180 * 3600)

// Parse reads a profile CSV:
//
//	# quantity: surface_density | volume_density | circular_velocity | enclosed_mass
//	# unit: Msun/pc2            (unit of the value and error columns)
//	# radius_unit: kpc | pc | arcsec   (arcsec requires # distance_kpc)
//	# evidence: OBSERVED | INFERRED | RECONSTRUCTED | ILLUSTRATIVE
//	# source: citation           (required)
//	# name, url, notes, distance_kpc: optional
//	r,value,err                      (symmetric 1σ error)
//	r,value,err_minus,err_plus       (asymmetric errors)
//	1.0,250,30
//
// The header row is optional; error columns are optional; further text
// columns (reference, tracer) are kept as the point label. Radii must not
// decrease; equal radii are allowed for independent estimates.
func Parse(name string, data []byte) (Dataset, error) {
	d := Dataset{Name: name}
	meta := map[string]string{}
	var body bytes.Buffer
	for _, line := range strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n") {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "#") {
			kv := strings.SplitN(strings.TrimSpace(t[1:]), ":", 2)
			if len(kv) == 2 {
				meta[strings.ToLower(strings.TrimSpace(kv[0]))] = strings.TrimSpace(kv[1])
			}
			continue
		}
		if t != "" {
			body.WriteString(t + "\n")
		}
	}
	d.Quantity = Quantity(strings.ToLower(meta["quantity"]))
	factors, ok := unitFactors[d.Quantity]
	if !ok {
		return d, fmt.Errorf("unknown or missing '# quantity:' (use surface_density, volume_density, circular_velocity or enclosed_mass)")
	}
	d.OriginalUnit = meta["unit"]
	f, ok := factors[normUnit(d.OriginalUnit)]
	if !ok {
		return d, fmt.Errorf("unit %q is not a unit of %s (accepted: %s); quantities are never converted into each other, e.g. Σ into ρ",
			d.OriginalUnit, d.Quantity, strings.Join(keys(factors), ", "))
	}
	d.Unit = CanonicalUnit[d.Quantity]
	d.RadiusUnit = strings.ToLower(meta["radius_unit"])
	if d.RadiusUnit == "" {
		d.RadiusUnit = "kpc"
	}
	var rf float64
	switch d.RadiusUnit {
	case "kpc":
		rf = 1
	case "pc":
		rf = 1e-3
	case "arcsec":
		dist, err := strconv.ParseFloat(meta["distance_kpc"], 64)
		if err != nil || !(dist > 0) {
			return d, errors.New("radius_unit arcsec needs '# distance_kpc:' (angular radii are converted with the small-angle formula)")
		}
		d.DistanceKpc = dist
		rf = dist * arcsecToRad
	default:
		return d, fmt.Errorf("unknown radius_unit %q (kpc, pc or arcsec)", d.RadiusUnit)
	}
	d.Evidence = strings.ToUpper(meta["evidence"])
	switch d.Evidence {
	case "OBSERVED", "INFERRED", "RECONSTRUCTED", "ILLUSTRATIVE":
	case "":
		return d, errors.New("missing '# evidence:' (OBSERVED, INFERRED, RECONSTRUCTED or ILLUSTRATIVE)")
	default:
		return d, fmt.Errorf("unknown evidence level %q", d.Evidence)
	}
	d.Source = meta["source"]
	if d.Source == "" {
		return d, errors.New("missing '# source:' — every dataset needs an attribution")
	}
	if n := meta["name"]; n != "" {
		d.Name = n
	}
	d.URL, d.Notes = meta["url"], meta["notes"]

	rd := csv.NewReader(&body)
	rd.FieldsPerRecord = -1
	rd.TrimLeadingSpace = true
	row := 0
	errLo, errHi := 2, 2 // column indices of the errors
	for {
		rec, err := rd.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return d, fmt.Errorf("CSV: %w", err)
		}
		row++
		if len(rec) < 2 {
			return d, fmt.Errorf("row %d: need at least radius and value", row)
		}
		r, err1 := strconv.ParseFloat(rec[0], 64)
		v, err2 := strconv.ParseFloat(rec[1], 64)
		if err1 != nil || err2 != nil {
			if row == 1 && len(d.Points) == 0 {
				// Header row: locate asymmetric error columns by name.
				for i, h := range rec {
					h = strings.ToLower(h)
					if strings.Contains(h, "minus") || strings.Contains(h, "lo") && strings.Contains(h, "err") {
						errLo = i
					}
					if strings.Contains(h, "plus") || strings.Contains(h, "hi") && strings.Contains(h, "err") {
						errHi = i
					}
				}
				continue
			}
			return d, fmt.Errorf("row %d: not a number", row)
		}
		if row == 1 && len(rec) >= 4 && isNumber(rec[2]) && isNumber(rec[3]) {
			errLo, errHi = 2, 3 // headerless r,value,err_minus,err_plus
		}
		lo, err := errorAt(rec, errLo)
		if err != nil {
			return d, fmt.Errorf("row %d: %w", row, err)
		}
		hi, err := errorAt(rec, errHi)
		if err != nil {
			return d, fmt.Errorf("row %d: %w", row, err)
		}
		if !(r > 0) || math.IsInf(r, 0) {
			return d, fmt.Errorf("row %d: radius must be positive", row)
		}
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return d, fmt.Errorf("row %d: value is not finite", row)
		}
		if v < 0 && d.Quantity != CircularVelocity {
			return d, fmt.Errorf("row %d: %s cannot be negative", row, d.Quantity)
		}
		var label []string
		for i := 2; i < len(rec); i++ {
			if i != errLo && i != errHi && strings.TrimSpace(rec[i]) != "" && !isNumber(rec[i]) {
				label = append(label, strings.TrimSpace(rec[i]))
			}
		}
		d.Points = append(d.Points, Point{R: r * rf, V: v * f, ErrLo: lo * f, ErrHi: hi * f, Label: strings.Join(label, " · ")})
	}
	if len(d.Points) == 0 {
		return d, errors.New("no data rows")
	}
	for i := 1; i < len(d.Points); i++ {
		if d.Points[i].R < d.Points[i-1].R {
			return d, fmt.Errorf("radii must not decrease (data row %d)", i+1)
		}
	}
	return d, nil
}

func normUnit(u string) string {
	u = strings.ToLower(strings.ReplaceAll(u, " ", ""))
	r := strings.NewReplacer("m☉", "msun", "m_sun", "msun", "solmass", "msun", "²", "2", "³", "3", "^", "", "kms-1", "km/s", "kms^-1", "km/s")
	return r.Replace(u)
}

func keys(m map[string]float64) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	return out
}

func isNumber(s string) bool {
	_, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	return err == nil
}

// errorAt reads an optional non-negative error column.
func errorAt(rec []string, i int) (float64, error) {
	if i >= len(rec) || strings.TrimSpace(rec[i]) == "" {
		return 0, nil
	}
	e, err := strconv.ParseFloat(strings.TrimSpace(rec[i]), 64)
	if err != nil || e < 0 || math.IsInf(e, 0) {
		return 0, errors.New("errors must be non-negative numbers")
	}
	return e, nil
}
