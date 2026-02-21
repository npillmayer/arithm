package jhobby

import (
	"fmt"
	"math"
	"math/cmplx"

	"github.com/npillmayer/arithm"
)

func hobbyParamsAlphaBeta(theta, phi float64) (float64, float64) {
	constA := 1.41421356     // sqrt(2) -- empiric constants, as explained by J.Hobby
	constB := 0.0625         // 1/16
	constC := 0.38196601125  // (3 - sqrt(5)) / 2
	constCC := 0.61803398875 // 1 - c
	st := math.Sin(theta)    // in-angle
	ct := math.Cos(theta)
	sf := math.Sin(phi) // out-angle
	cf := math.Cos(phi)
	alpha := constA * (st - constB*sf) * (sf - constB*st) * (ct - cf)
	beta := 1 + constCC*ct + constC*cf
	return alpha, beta
}

func hobbyParamsRhoSigma(alpha, beta float64) (float64, float64) {
	rho := (2 + alpha) / beta
	sigma := (2 - alpha) / beta
	return rho, sigma
}

func cunitvecs(i int, theta, phi float64, dvec arithm.Pair) (arithm.Pair, arithm.Pair) {
	st := math.Sin(theta)
	ct := math.Cos(theta)
	sf := math.Sin(phi)
	cf := math.Cos(phi)
	dx, dy := real(dvec), imag(dvec)
	uv1 := arithm.P(dx*ct-dy*st, dx*st+dy*ct)
	uv2 := arithm.P(dx*cf+dy*sf, -dx*sf+dy*cf)
	return uv1, uv2
}

// Calculate control points between z.i and z.[i+1].
func controlPoints(i int, phi, theta, a, b float64, dvec arithm.Pair) (arithm.Pair, arithm.Pair) {
	alpha, beta := hobbyParamsAlphaBeta(theta, phi)
	rho, sigma := hobbyParamsRhoSigma(alpha, beta)
	uv1, uv2 := cunitvecs(i, theta, phi, dvec)
	crho := arithm.P(a/3*rho, 0)
	csigma := arithm.P(b/3*sigma, 0)
	p2 := crho * uv1
	p3 := csigma * uv2
	return p2, p3
}

// Extend an array/slice of pairs to make room for index i.
// Will do nothing if the array is already large enough.
func extendC(arr []arithm.Pair, i int, deflt arithm.Pair) []arithm.Pair {
	l := len(arr)
	if i >= l {
		arr = append(arr, make([]arithm.Pair, i-l+1)...)
		for ; i >= l; i-- {
			arr[i] = deflt
		}
	}
	return arr
}

// Get a value from an array/slice if present, default value deflt otherwise.
func getC(arr []arithm.Pair, i int, deflt arithm.Pair) arithm.Pair {
	if i >= len(arr) {
		return deflt
	}
	return arr[i]
}

func angle(pr arithm.Pair) float64 {
	if cmplx.IsNaN(pr.C()) {
		return 0.0
	}
	return cmplx.Phase(pr.C())
}

// Reduce an angle to fit into -pi .. pi.
func reduceAngle(a float64) float64 {
	if math.Abs(a) > pi {
		if a > 0 {
			a -= pi2
		} else {
			a += pi2
		}
	}
	return a
}

// Return 1/a for a.
func recip(a float64) float64 {
	if math.IsNaN(a) {
		return 1.0
	}
	return 1.0 / a
}

// Return a^2 for a.
func square(a float64) float64 {
	return math.Pow(a, 2.0)
}

func rad2deg(a float64) float64 {
	return a * 180 / pi
}

func ptstring(p arithm.Pair, iscontrol bool) string {
	if cmplx.IsNaN(p.C()) {
		return "(<unknown>)"
	}
	if iscontrol {
		return fmt.Sprintf("(%.4f,%.4f)", round(p.X()), round(p.Y()))
	}
	return fmt.Sprintf("(%.4g,%.4g)", round(p.X()), round(p.Y()))
}

func round(x float64) float64 {
	if x >= 0 {
		return float64(int64(x*10000.0+0.5)) / 10000.0
	}
	return float64(int64(x*10000.0-0.5)) / 10000.0
}

func equal(c1, c2 arithm.Pair) bool {
	return math.Abs(cmplx.Phase(c1.C()-c2.C())) < _epsilon
}
