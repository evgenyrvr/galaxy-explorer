package physics

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
)

// ArmPattern is an m-armed logarithmic spiral modulation of a disc's
// azimuthal distribution. It redistributes mass in azimuth only:
//
//	p(ψ) ∝ 1 + A cos ψ,  ψ = m [φ − φ0 − ln(R/R0) / tan(pitch)]
//
// so the azimuthally averaged Σ(R) — and therefore the total mass — is
// unchanged. Arms are density enhancements of existing matter, never extra mass.
type ArmPattern struct {
	M         int     `json:"m"`
	PitchDeg  float64 `json:"pitchDeg"`
	Amplitude float64 `json:"amplitude"` // 0 ≤ A ≤ 1
	HalfWidth float64 `json:"halfWidth"` // |ψ| < HalfWidth is tagged as an arm, rad
	Rin       float64 `json:"rinKpc"`
	Rout      float64 `json:"routKpc"`
	R0        float64 `json:"r0Kpc"`
	Phase0Deg float64 `json:"phase0Deg"`
}

// ArmTag returns the tag name of arm k (0-based).
func ArmTag(k int) string { return fmt.Sprintf("arm_%d", k+1) }

// angularFraction is the mass fraction with |ψ| < HalfWidth (all arms).
func (a *ArmPattern) angularFraction() float64 {
	d := a.HalfWidth
	return (2*d + 2*a.Amplitude*math.Sin(d)) / (2 * math.Pi)
}

// DiscShape selects the radial surface-density law.
type DiscShape int

const (
	// Exponential: Σ ∝ exp(−R/Rd) (Freeman 1970).
	Exponential DiscShape = iota
	// ExponentialHole: Σ ∝ exp(−Rm/R − R/Rd), a disc with a central
	// depression, as used for gas discs by McMillan (2017, MNRAS 465, 76).
	ExponentialHole
	// Tabulated: Σ(R) interpolated linearly in a measured profile, constant
	// inside the first radius and zero beyond the last one.
	Tabulated
)

// Disc is an axisymmetric thick disc with sech² vertical structure:
// ρ(R,z) = Σ(R) / (2 z0(R)) · sech²(z / z0(R)), optionally flaring and with
// a spiral-arm modulation.
type Disc struct {
	shape   DiscShape
	rd, rm  float64
	tabR    []float64
	tabS    []float64
	z0      float64
	rflare  float64
	rmax    float64
	arms    *ArmPattern
	rot     Mat3
	rotated bool
	norm    float64 // Σ normalisation so that the truncated disc has unit mass
	cyl     *CDFTable
	sph     *gridFunc
}

// DiscOptions configures NewDisc.
type DiscOptions struct {
	Shape      DiscShape
	Rd         float64   // scale length, kpc
	Rm         float64   // hole radius for ExponentialHole, kpc
	Z0         float64   // sech² scale height at R = 0, kpc
	RFlare     float64   // z0(R) = z0·sqrt(1 + (R/RFlare)²); 0 = no flare
	RMax       float64   // truncation radius, kpc
	TableR     []float64 // Tabulated: radii, kpc (increasing)
	TableSigma []float64 // Tabulated: Σ at TableR (any unit; normalised)
	Arms       *ArmPattern
	Rotation   *Mat3
}

// NewDisc builds a normalised disc distribution.
func NewDisc(o DiscOptions) *Disc {
	d := &Disc{shape: o.Shape, rd: o.Rd, rm: o.Rm, z0: o.Z0, rflare: o.RFlare, rmax: o.RMax, arms: o.Arms, rot: Identity,
		tabR: o.TableR, tabS: o.TableSigma}
	if d.shape == Tabulated {
		if len(d.tabR) < 2 || len(d.tabR) != len(d.tabS) {
			panic("physics: tabulated disc needs ≥ 2 matching radii and values")
		}
		d.rmax = math.Min(d.rmax, d.tabR[len(d.tabR)-1])
	}
	if o.Rotation != nil {
		d.rot = *o.Rotation
		d.rotated = true
	}
	d.norm = 1
	raw := NewCDFTableRaw(func(R float64) float64 { return 2 * math.Pi * R * d.sigmaShape(R) }, d.rmax, 4096)
	d.norm = 1 / raw.total
	d.cyl = raw.table
	d.sph = d.buildSphericalTable()
	return d
}

func (d *Disc) sigmaShape(R float64) float64 {
	switch d.shape {
	case ExponentialHole:
		if R <= 0 {
			return 0
		}
		return math.Exp(-d.rm/R - R/d.rd)
	case Tabulated:
		return interpTable(d.tabR, d.tabS, R)
	default:
		return math.Exp(-R / d.rd)
	}
}

func (d *Disc) scaleHeight(R float64) float64 {
	if d.rflare <= 0 {
		return d.z0
	}
	x := R / d.rflare
	return d.z0 * math.Sqrt(1+x*x)
}

// buildSphericalTable integrates ∫ 2πR Σ(R) tanh(√(r²−R²)/z0(R)) dR.
func (d *Disc) buildSphericalTable() *gridFunc {
	rmaxSph := math.Hypot(d.rmax, 10*d.scaleHeight(d.rmax))
	rs := logGrid(d.rmax*1e-5, rmaxSph, 300)
	ys := make([]float64, len(rs))
	const n = 3000
	for i, r := range rs {
		up := math.Min(r, d.rmax)
		h := up / n
		sum := 0.0
		for j := 0; j < n; j++ {
			R := (float64(j) + 0.5) * h
			zmax := math.Sqrt(r*r - R*R)
			sum += 2 * math.Pi * R * d.sigmaShape(R) * math.Tanh(zmax/d.scaleHeight(R))
		}
		ys[i] = clamp01(sum * h * d.norm)
	}
	return &gridFunc{x: rs, y: ys}
}

func (d *Disc) Kind() string {
	switch d.shape {
	case ExponentialHole:
		return "exponential_disc_with_hole"
	case Tabulated:
		return "tabulated_disc"
	}
	return "exponential_disc"
}

func (d *Disc) Params() map[string]float64 {
	p := map[string]float64{"z0_kpc": d.z0, "Rmax_kpc": d.rmax}
	if d.shape == Tabulated {
		p["table_points"] = float64(len(d.tabR))
	} else {
		p["Rd_kpc"] = d.rd
	}
	if d.shape == ExponentialHole {
		p["Rm_kpc"] = d.rm
	}
	if d.rflare > 0 {
		p["Rflare_kpc"] = d.rflare
	}
	if d.arms != nil {
		p["arms_m"] = float64(d.arms.M)
		p["arms_pitch_deg"] = d.arms.PitchDeg
		p["arms_amplitude"] = d.arms.Amplitude
	}
	return p
}

func (d *Disc) Sample(rng *rand.Rand, pos []float32, tags []uint8) {
	for i := range tags {
		R := d.cyl.Inverse(rng.Float64())
		u := 2*rng.Float64() - 1
		// Guard atanh(±1).
		u = math.Max(-0.999999, math.Min(0.999999, u))
		z := d.scaleHeight(R) * math.Atanh(u)
		phi, tag := d.samplePhi(rng, R)
		v := Vec3{R * math.Cos(phi), R * math.Sin(phi), z}
		if d.rotated {
			v = d.rot.Apply(v)
		}
		put(pos, i, v)
		tags[i] = tag
	}
}

func (d *Disc) samplePhi(rng *rand.Rand, R float64) (float64, uint8) {
	a := d.arms
	if a == nil || R < a.Rin || R > a.Rout {
		return 2 * math.Pi * rng.Float64(), 0
	}
	var psi float64
	for {
		psi = math.Pi * (2*rng.Float64() - 1)
		if rng.Float64()*(1+a.Amplitude) <= 1+a.Amplitude*math.Cos(psi) {
			break
		}
	}
	k := rng.Intn(a.M)
	phiArm := a.Phase0Deg*math.Pi/180 + 2*math.Pi*float64(k)/float64(a.M) +
		math.Log(R/a.R0)/math.Tan(a.PitchDeg*math.Pi/180)
	phi := phiArm + psi/float64(a.M)
	if math.Abs(psi) < a.HalfWidth {
		return phi, uint8(k + 1)
	}
	return phi, 0
}

func (d *Disc) SphericalFraction(r float64) float64 { return clamp01(d.sph.at(r)) }

func (d *Disc) MaxRadius() float64 { return d.sph.x[len(d.sph.x)-1] }

func (d *Disc) TagNames() []string {
	if d.arms == nil {
		return nil
	}
	n := make([]string, d.arms.M)
	for k := range n {
		n[k] = ArmTag(k)
	}
	return n
}

func (d *Disc) TagFraction(k int) float64 {
	a := d.arms
	if a == nil || k < 1 || k > a.M {
		return 0
	}
	radial := d.cyl.At(a.Rout) - d.cyl.At(a.Rin)
	return radial * a.angularFraction() / float64(a.M)
}

func (d *Disc) TagFractionExact() bool { return true }

func (d *Disc) SurfaceDensity(R float64) float64 {
	if R > d.rmax {
		return 0
	}
	return d.norm * d.sigmaShape(R)
}

func (d *Disc) CylindricalFraction(R float64) float64 { return d.cyl.At(R) }

// SlabFraction returns the mass fraction with rmin ≤ R < rmax and |z| < zmax
// (zmax ≤ 0: no height limit). ok is false for tilted discs, whose disc
// frame differs from the galaxy frame.
func (d *Disc) SlabFraction(rmin, rmax, zmax float64) (f float64, ok bool) {
	if d.rotated {
		return 0, false
	}
	if zmax <= 0 {
		return d.cyl.At(rmax) - d.cyl.At(rmin), true
	}
	lo, hi := math.Max(0, rmin), math.Min(rmax, d.rmax)
	if hi <= lo {
		return 0, true
	}
	const n = 4000
	h := (hi - lo) / n
	sum := 0.0
	for j := 0; j < n; j++ {
		R := lo + (float64(j)+0.5)*h
		sum += 2 * math.Pi * R * d.sigmaShape(R) * math.Tanh(zmax/d.scaleHeight(R))
	}
	return clamp01(sum * h * d.norm), true
}

// MidplaneVc2PerGM implements the Freeman (1970) razor-thin exponential disc
// rotation curve: Vc² = 4πGΣ0 Rd y² [I0(y)K0(y) − I1(y)K1(y)], y = R/(2Rd).
// Only valid for the pure exponential shape; the truncation beyond RMax is
// accounted for through the normalisation.
func (d *Disc) MidplaneVc2PerGM(R float64) float64 {
	if d.shape != Exponential || d.rotated || R <= 0 {
		return math.NaN()
	}
	y := R / (2 * d.rd)
	sigma0 := d.norm // Σ0 per unit mass
	return 4 * math.Pi * sigma0 * d.rd * y * y * (besselI0(y)*besselK0(y) - besselI1(y)*besselK1(y))
}

// cdfRaw holds an unnormalised integral alongside the normalised table.
type cdfRaw struct {
	table *CDFTable
	total float64
}

// NewCDFTableRaw is NewCDFTable that also returns the raw integral.
func NewCDFTableRaw(w func(x float64) float64, xmax float64, n int) cdfRaw {
	// Integrate once to get the raw total with the same grid as the table.
	xmin := xmax * 1e-7
	l := math.Log(xmax / xmin)
	prev := 0.0
	total := 0.0
	for i := 1; i <= n; i++ {
		b := xmin * math.Exp(l*float64(i-1)/float64(n-1))
		m := 0.5 * (prev + b)
		total += (b - prev) / 6 * (w(prev) + 4*w(m) + w(b))
		prev = b
	}
	return cdfRaw{table: NewCDFTable(w, xmax, n), total: total}
}

// interpTable interpolates linearly; constant below the first radius and
// zero beyond the last.
func interpTable(xs, ys []float64, x float64) float64 {
	n := len(xs)
	if x <= xs[0] {
		return math.Max(0, ys[0])
	}
	if x > xs[n-1] {
		return 0
	}
	i := sort.SearchFloat64s(xs, x)
	t := (x - xs[i-1]) / (xs[i] - xs[i-1])
	return math.Max(0, ys[i-1]+t*(ys[i]-ys[i-1]))
}
