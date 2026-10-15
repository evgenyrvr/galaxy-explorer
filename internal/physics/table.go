package physics

import (
	"math"
	"sort"
)

// CDFTable is a tabulated, monotone cumulative distribution on [0, xmax].
type CDFTable struct {
	x   []float64
	cdf []float64
}

// NewCDFTable integrates the non-negative density w(x) on [0, xmax] with a
// log-spaced grid (finer near the centre) and normalises it to 1. The density
// w is the 1D measure, e.g. 4πr²ρ(r) or 2πRΣ(R).
func NewCDFTable(w func(x float64) float64, xmax float64, n int) *CDFTable {
	if n < 16 {
		n = 16
	}
	xmin := xmax * 1e-7
	xs := make([]float64, n+1)
	xs[0] = 0
	lr := math.Log(xmax / xmin)
	for i := 1; i <= n; i++ {
		xs[i] = xmin * math.Exp(lr*float64(i-1)/float64(n-1))
	}
	cdf := make([]float64, n+1)
	for i := 1; i <= n; i++ {
		a, b := xs[i-1], xs[i]
		m := 0.5 * (a + b)
		// Simpson's rule on each interval.
		cdf[i] = cdf[i-1] + (b-a)/6*(w(a)+4*w(m)+w(b))
	}
	total := cdf[n]
	if total <= 0 || math.IsNaN(total) || math.IsInf(total, 0) {
		panic("physics: density integrates to a non-positive or non-finite mass")
	}
	for i := range cdf {
		cdf[i] /= total
	}
	return &CDFTable{x: xs, cdf: cdf}
}

// NewCDFTableFromSamples builds an empirical CDF from sample values.
func NewCDFTableFromSamples(v []float64) *CDFTable {
	s := append([]float64(nil), v...)
	sort.Float64s(s)
	n := len(s)
	xs := make([]float64, n+1)
	cdf := make([]float64, n+1)
	for i := 0; i < n; i++ {
		xs[i+1] = s[i]
		cdf[i+1] = float64(i+1) / float64(n)
	}
	return &CDFTable{x: xs, cdf: cdf}
}

// Max returns the upper bound of the table.
func (t *CDFTable) Max() float64 { return t.x[len(t.x)-1] }

// At returns the CDF at x with linear interpolation.
func (t *CDFTable) At(x float64) float64 {
	if x <= 0 {
		return 0
	}
	n := len(t.x)
	if x >= t.x[n-1] {
		return 1
	}
	i := sort.SearchFloat64s(t.x, x)
	if i == 0 {
		return t.cdf[0]
	}
	x0, x1 := t.x[i-1], t.x[i]
	if x1 == x0 {
		return t.cdf[i]
	}
	return t.cdf[i-1] + (t.cdf[i]-t.cdf[i-1])*(x-x0)/(x1-x0)
}

// Inverse returns x such that CDF(x) = u (u ∈ [0,1]).
func (t *CDFTable) Inverse(u float64) float64 {
	n := len(t.cdf)
	if u <= 0 {
		return 0
	}
	if u >= 1 {
		return t.x[n-1]
	}
	i := sort.SearchFloat64s(t.cdf, u)
	if i == 0 {
		return t.x[0]
	}
	c0, c1 := t.cdf[i-1], t.cdf[i]
	if c1 == c0 {
		return t.x[i]
	}
	return t.x[i-1] + (t.x[i]-t.x[i-1])*(u-c0)/(c1-c0)
}

// gaussLegendre returns nodes and weights on [-1, 1].
func gaussLegendre(n int) (x, w []float64) {
	x = make([]float64, n)
	w = make([]float64, n)
	for i := 0; i < n; i++ {
		z := math.Cos(math.Pi * (float64(i) + 0.75) / (float64(n) + 0.5))
		for it := 0; it < 100; it++ {
			p1, p2 := 1.0, 0.0
			for j := 1; j <= n; j++ {
				p3 := p2
				p2 = p1
				p1 = ((2*float64(j)-1)*z*p2 - (float64(j)-1)*p3) / float64(j)
			}
			pp := float64(n) * (z*p1 - p2) / (z*z - 1)
			z1 := z
			z = z1 - p1/pp
			if math.Abs(z-z1) < 1e-15 {
				w[i] = 2 / ((1 - z*z) * pp * pp)
				break
			}
		}
		x[i] = z
	}
	return
}

// logGrid returns n log-spaced radii in [rmin, rmax].
func logGrid(rmin, rmax float64, n int) []float64 {
	g := make([]float64, n)
	l := math.Log(rmax / rmin)
	for i := range g {
		g[i] = rmin * math.Exp(l*float64(i)/float64(n-1))
	}
	return g
}

// gridFunc is a function tabulated on a monotone grid.
type gridFunc struct {
	x, y []float64
}

func (g gridFunc) at(x float64) float64 {
	n := len(g.x)
	if x <= g.x[0] {
		// Enclosed fractions go to zero at the centre.
		if g.x[0] <= 0 {
			return g.y[0]
		}
		return g.y[0] * x / g.x[0]
	}
	if x >= g.x[n-1] {
		return g.y[n-1]
	}
	i := sort.SearchFloat64s(g.x, x)
	x0, x1 := g.x[i-1], g.x[i]
	return g.y[i-1] + (g.y[i]-g.y[i-1])*(x-x0)/(x1-x0)
}
