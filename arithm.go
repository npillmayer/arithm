/*
Package arithm implements points, affine transformations,
arithmetic for polynomials, and a linear equations solver.

# BSD License

# Copyright (c) Norbert Pillmayer

All rights reserved.

Please refer to the license file for more information.
*/
package arithm

import (
	"fmt"
	"math"
	"math/cmplx"

	"github.com/npillmayer/schuko/tracing"
)

// tracer writes to trace with key 'arithm'
func tracer() tracing.Trace {
	return tracing.Select("arithm")
}

// === Numeric Data Type =====================================================

// Deg2Rad is a constant for converting from DEG to RAD or vice versa
var Deg2Rad float64 = 0.01745329251

// Epsilon : numbers below ε are considered 0
var Epsilon float64 = 0.0000001

// Is0 is a predicate: is n = 0 ?
func Is0(n float64) bool {
	return math.Abs(n) <= Epsilon
}

// Is1 is a predicate: is n = 1.0 ?
func Is1(n float64) bool {
	return math.Abs(1-n) <= Epsilon
}

// Zap makes n = 0 if n "means" to be zero
func Zap(n float64) float64 {
	if Is0(n) {
		n = 0
	}
	return n
}

// Round to ε.
func Round(n float64) float64 {
	return math.Round(n/Epsilon) * Epsilon
}

// === Pair Data Type ========================================================

// Pair is an interface for pairs / 2D-points
type Pair complex128

// Origin represents the frequently used constant (0,0).
var Origin = P(float64(0), float64(0))

// Pretty Stringer for simple pairs.
func (p Pair) String() string {
	return fmt.Sprintf("(%g,%g)", real(p), imag(p))
}

// C returns a Pair as a complex number.
func (p Pair) C() complex128 {
	return complex128(p)
}

// C2P returns a Pair from a complex number.
func C2P(c complex128) Pair {
	if cmplx.IsNaN(c) || cmplx.IsInf(c) {
		tracer().Errorf("created pair for complex.NaN")
		return P(0, 0)
	}
	return P(real(c), imag(c))
}

// P is a quick notation for contructing a pair from floats.
func P(x, y float64) Pair {
	return Pair(complex(x, y))
}

// F is a quick notation for getting float values from a pair.
func (p Pair) F() (float64, float64) {
	px := real(p.C())
	py := imag(p.C())
	return px, py
}

// X is the x-part of a pair.
func (p Pair) X() float64 {
	return real(p.C())
}

// Y is the y-part of a pair.
func (p Pair) Y() float64 {
	return imag(p.C())
}

// Zap rounds x-part and y-part to Epsilon.
func (p Pair) Zap() Pair {
	p = P(Zap(p.X()), Zap(p.Y()))
	return p
}

// IsOrigin is a predicate: is this pair origin?
func (p Pair) IsOrigin() bool {
	return p.Equal(Origin)
}

// Equal compares two pairs.
func (p Pair) Equal(p2 Pair) bool {
	p2 = p2.Zap()
	return Is0(p.X()-p2.X()) && Is0(p.Y()-p2.Y())
}

// Scaled returns a new pair scaled by factor a.
func (p Pair) Scaled(a float64) Pair {
	return P(p.X()*a, p.Y()*a).Zap()
}

// XScaled returns a new pair x-scaled by factor a.
func (p Pair) XScaled(a float64) Pair {
	return P(p.X()*a, p.Y()).Zap()
}

// YScaled returns a new pair y-scaled by factor a.
func (p Pair) YScaled(a float64) Pair {
	return P(p.X(), p.Y()*a).Zap()
}

// Shifted returns a new pair translated by v.
func (p Pair) Shifted(v Pair) Pair {
	T := Translation(v)
	return T.Transform(p).Zap()
}

// Rotated returns a new pair rotated around origin by theta (counterclockwise).
func (p Pair) Rotated(theta float64) Pair {
	T := Rotation(theta)
	return T.Transform(p).Zap()
}

// Rotatedaround returns a new pair rotated around v by theta (counterclockwise).
func (p Pair) Rotatedaround(v Pair, theta float64) Pair {
	return p.Shifted(-v).Rotated(theta).Shifted(v).Zap()
}

// === Affine Transformations ================================================

// AT is an affine transform, a matrix type used for transforming vectors.
type AT []float64 // a 3x3 matrix, flattened by rows

// Internal constructor. Clients implicitely use this as a starting point for
// transform combinations.
func newAT() AT {
	m := make([]float64, 9)
	return m
}

func (m AT) get(row, col int) float64 {
	return m[row*3+col]
}

func (m AT) set(row, col int, value float64) {
	m[row*3+col] = value
}

func (m AT) row(row int) []float64 {
	return m[row*3 : (row+1)*3]
}

func (m AT) col(col int) []float64 {
	c := make([]float64, 3)
	c[0] = m[col]
	c[1] = m[3+col]
	c[2] = m[6+col]
	return c
}

// Identity transform. Will transform a point onto itself.
func Identity() AT {
	m := newAT()
	m.set(0, 0, 1.0)
	m.set(1, 1, 1.0)
	m.set(2, 2, 1.0)
	return m
}

// Translation transform. Translate a point by (dx,dy).
func Translation(p Pair) AT {
	m := Identity()
	m.set(0, 2, p.X())
	m.set(1, 2, p.Y())
	return m
}

// Rotation transform. Rotate a point counter-clockwise around the origin.
// Argument is in radians.
func Rotation(theta float64) AT {
	m := newAT()
	sin := math.Sin(theta)
	cos := math.Cos(theta)
	m.set(0, 0, cos)
	m.set(0, 1, -sin)
	m.set(1, 0, sin)
	m.set(1, 1, cos)
	m.set(2, 2, 1.0)
	return m
}

// Debug Stringer for an affine transform.
func (m AT) String() string {
	s := fmt.Sprintf("[%g,%g,%g|%g,%g,%g|%g,%g,%g]",
		m[0], m[1], m[2], m[3], m[4], m[5], m[6], m[7], m[8])
	return s
}

// v1 × v2, v.n = [a,b,c]
func dotProd(vec1, vec2 []float64) float64 {
	p1 := vec1[0] * vec2[0]
	p2 := vec1[1] * vec2[1]
	p3 := vec1[2] * vec2[2]
	return p1 + p2 + p3
}

// Combine 2 affine transformation to a new one. Returns a new transformation
// without changing the argument(s).
func (m AT) Combine(n AT) AT {
	o := newAT()
	for row := 0; row < 3; row++ {
		for col := 0; col < 3; col++ {
			o.set(row, col, dotProd(n.row(row), m.col(col)))
		}
	}
	return o
}

func (m *AT) multiplyVector(v []float64) []float64 {
	c := make([]float64, 3)
	c[0] = dotProd(m.row(0), v)
	c[1] = dotProd(m.row(1), v)
	c[2] = dotProd(m.row(2), v)
	return c
}

// Transform a 2D-point. The argument is unchanged and a new pair is returned.
func (m AT) Transform(p Pair) Pair {
	c := make([]float64, 3)
	c[0] = p.X()
	c[1] = p.Y()
	c[2] = 1.0
	c = m.multiplyVector(c)
	return P(c[0], c[1])
}
