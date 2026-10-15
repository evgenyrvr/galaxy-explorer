package physics

import (
	"math"
	"math/rand"
	"runtime"
	"sync"

	"galaxy/internal/units"
)

// EnclosedMass returns the host's spherically enclosed mass M(<r) in M☉.
// Using the spherical approximation of the host's own mass model keeps the
// orbit integration consistent with the mass budget.
type EnclosedMass func(r float64) float64

func accel(menc EnclosedMass, x Vec3, soft float64) Vec3 {
	r2 := x.Dot(x) + soft*soft
	r := math.Sqrt(r2)
	return x.Scale(-units.G * menc(r) / (r2 * r))
}

// Phase is a position–velocity pair (kpc, km/s).
type Phase struct {
	X, V Vec3
}

// IntegrateOrbit advances p by duration t (internal units, kpc/(km/s)) with a
// kick-drift-kick leapfrog and an adaptive step dt = η r/|v| bounded by dtMax.
func IntegrateOrbit(menc EnclosedMass, p Phase, t float64, dtMax, soft float64) Phase {
	if t <= 0 {
		return p
	}
	const eta = 0.02
	elapsed := 0.0
	x, v := p.X, p.V
	for elapsed < t {
		r := math.Max(x.Norm(), soft)
		dt := math.Min(dtMax, eta*r/math.Max(v.Norm(), 1))
		if elapsed+dt > t {
			dt = t - elapsed
		}
		a := accel(menc, x, soft)
		v = v.Add(a.Scale(dt / 2))
		x = x.Add(v.Scale(dt))
		a = accel(menc, x, soft)
		v = v.Add(a.Scale(dt / 2))
		elapsed += dt
	}
	return Phase{x, v}
}

// SprayOptions configures a particle-spray tidal debris model in the spirit
// of Fardal, Huang & Weinberg (2015, MNRAS 452, 301): debris is released
// near the progenitor's Lagrange points over time and integrated to today.
type SprayOptions struct {
	Start        Phase   // progenitor phase at t = −Duration
	Duration     float64 // Gyr of integration
	ReleaseStart float64 // fraction of Duration at which release begins (0..1)
	ReleaseEnd   float64 // fraction of Duration at which release ends (0..1)
	ProgMass     float64 // progenitor mass for the Jacobi radius, M☉
	SigmaV       float64 // velocity dispersion of released debris, km/s
	N            int     // reference particles
	Seed         int64
	Soft         float64 // softening length, kpc
	RMax         float64 // debris that ends beyond RMax (unbound in this model) is discarded
}

// ParticleSpray returns present-day positions of tidal debris.
func ParticleSpray(menc EnclosedMass, o SprayOptions) []Vec3 {
	rng := rand.New(rand.NewSource(o.Seed))
	T := o.Duration / units.KpcPerKmsInGyr
	const nRelease = 400
	const dtMax = 0.004
	type release struct {
		p Phase
		t float64 // remaining integration time
	}
	var rel []release
	cur := o.Start
	tcur := 0.0
	per := o.N / nRelease
	if per < 1 {
		per = 1
	}
	for i := 0; i < nRelease; i++ {
		f := o.ReleaseStart + (o.ReleaseEnd-o.ReleaseStart)*float64(i)/float64(nRelease-1)
		tr := f * T
		cur = IntegrateOrbit(menc, cur, tr-tcur, dtMax, o.Soft)
		tcur = tr
		r := cur.X.Norm()
		rt := r * math.Cbrt(o.ProgMass/(3*math.Max(menc(r), o.ProgMass)))
		rhat := cur.X.Unit()
		vr := rhat.Scale(cur.V.Dot(rhat))
		vt := cur.V.Sub(vr)
		for j := 0; j < per; j++ {
			side := 1.0
			if j%2 == 1 {
				side = -1
			}
			off := side * rt * (1 + 0.3*rng.NormFloat64())
			x := cur.X.Add(rhat.Scale(off))
			// Match the progenitor's angular velocity: scale only the tangential part.
			v := vr.Add(vt.Scale(x.Norm() / math.Max(r, 1e-6)))
			v = v.Add(Vec3{rng.NormFloat64(), rng.NormFloat64(), rng.NormFloat64()}.Scale(o.SigmaV))
			rel = append(rel, release{Phase{x, v}, T - tr})
		}
	}
	out := make([]Vec3, len(rel))
	keep := make([]bool, len(rel))
	var wg sync.WaitGroup
	workers := runtime.NumCPU()
	ch := make(chan int, len(rel))
	for i := range rel {
		ch <- i
	}
	close(ch)
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range ch {
				p := IntegrateOrbit(menc, rel[i].p, rel[i].t, dtMax, o.Soft)
				out[i] = p.X
				keep[i] = o.RMax <= 0 || p.X.Norm() < o.RMax
			}
		}()
	}
	wg.Wait()
	res := out[:0]
	for i, k := range keep {
		if k {
			res = append(res, out[i])
		}
	}
	return res
}
