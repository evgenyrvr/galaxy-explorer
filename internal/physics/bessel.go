package physics

import "math"

// Modified Bessel functions via the polynomial approximations of
// Abramowitz & Stegun (1964), §9.8.1–9.8.8 (|ε| ≲ 2×10⁻⁷).

func besselI0(x float64) float64 {
	ax := math.Abs(x)
	if ax <= 3.75 {
		t := x / 3.75
		t *= t
		return 1 + t*(3.5156229+t*(3.0899424+t*(1.2067492+t*(0.2659732+t*(0.0360768+t*0.0045813)))))
	}
	t := 3.75 / ax
	return math.Exp(ax) / math.Sqrt(ax) * (0.39894228 + t*(0.01328592+t*(0.00225319+t*(-0.00157565+t*(0.00916281+
		t*(-0.02057706+t*(0.02635537+t*(-0.01647633+t*0.00392377))))))))
}

func besselI1(x float64) float64 {
	ax := math.Abs(x)
	var r float64
	if ax <= 3.75 {
		t := x / 3.75
		t *= t
		r = ax * (0.5 + t*(0.87890594+t*(0.51498869+t*(0.15084934+t*(0.02658733+t*(0.00301532+t*0.00032411))))))
	} else {
		t := 3.75 / ax
		r = math.Exp(ax) / math.Sqrt(ax) * (0.39894228 + t*(-0.03988024+t*(-0.00362018+t*(0.00163801+t*(-0.01031555+
			t*(0.02282967+t*(-0.02895312+t*(0.01787654-t*0.00420059))))))))
	}
	if x < 0 {
		return -r
	}
	return r
}

func besselK0(x float64) float64 {
	if x <= 2 {
		t := x * x / 4
		return -math.Log(x/2)*besselI0(x) + (-0.57721566 + t*(0.42278420+t*(0.23069756+t*(0.03488590+t*(0.00262698+t*(0.00010750+t*0.0000074))))))
	}
	t := 2 / x
	return math.Exp(-x) / math.Sqrt(x) * (1.25331414 + t*(-0.07832358+t*(0.02189568+t*(-0.01062446+t*(0.00587872+t*(-0.00251540+t*0.00053208))))))
}

func besselK1(x float64) float64 {
	if x <= 2 {
		t := x * x / 4
		return (x*math.Log(x/2)*besselI1(x) + (1 + t*(0.15443144+t*(-0.67278579+t*(-0.18156897+t*(-0.01919402+t*(-0.00110404-t*0.00004686))))))) / x
	}
	t := 2 / x
	return math.Exp(-x) / math.Sqrt(x) * (1.25331414 + t*(0.23498619+t*(-0.03655620+t*(0.01504268+t*(-0.00780353+t*(0.00325614-t*0.00068245))))))
}
