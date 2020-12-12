// Package polyn is for arithmetic with linear polynomials and linear equations.
/*
BSD 3-Clause License

Copyright (c) 2017–21, Norbert Pillmayer.

All rights reserved.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are met:

1. Redistributions of source code must retain the above copyright notice, this
   list of conditions and the following disclaimer.

2. Redistributions in binary form must reproduce the above copyright notice,
   this list of conditions and the following disclaimer in the documentation
   and/or other materials provided with the distribution.

3. Neither the name of the copyright holder nor the names of its
   contributors may be used to endorse or promote products derived from
   this software without specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE
FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL
DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR
SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER
CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY,
OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
*/
package polyn

import (
	"bytes"
	"fmt"
	"math"

	"github.com/emirpasic/gods/maps"
	"github.com/emirpasic/gods/maps/treemap"
	"github.com/npillmayer/arithm"
	"github.com/npillmayer/schuko/gtrace"
	"github.com/npillmayer/schuko/tracing"
)

// T traces to the equations tracer.
func T() tracing.Trace {
	return gtrace.EquationsTracer
}

// X is a helper for quick construction of polynomials.
// It denotes a term
//     C⋅x[I]
//
// I > 0
//
type X struct {
	I int     // exponent of x
	C float64 // coeffiencet
}

// New creates a polygon, given the term coefficients and exponents
//
// Use it as
//
//      polyn.New(8, polyn.X{2,5}, polyn.X{1,2/3} )
//
// to get
//
//      P(x) = 8 + 5a + 2/3b
//
func New(c float64, tms ...X) (Polynomial, error) { // construct a polynomial
	p := NewConstantPolynomial(c)
	var err error
	for _, t := range tms {
		if t.I < 1 {
			err = fmt.Errorf("Term coefficient must be at least 1, skipping it")
		} else {
			p.SetTerm(t.I, t.C)
		}
	}
	return p, err
}

// Polynomial is a type for linear polynomials
//
//     c + a.1 x.1 + a.2 x.2 + ... a.n x.n .
//
// We store the coefficients only. Index 0 is the constant term.
// We store the scales/coeff in a TreeMap (sorted map). Coefficients are of
// type float64.
type Polynomial struct {
	Terms *treemap.Map
}

// NewConstantPolynomial creates a Polynomial consisting of just a constant term.
func NewConstantPolynomial(c float64) Polynomial {
	//m := treemap.NewWithIntComparator()
	//p := Polynomial{m}
	p := Polynomial{}
	p.checkTerms()
	p.Terms.Put(0, c) // initialize with constant term (at position 0)
	return p.Zap()
}

func (p *Polynomial) checkTerms() {
	if p.Terms == nil {
		p.Terms = treemap.NewWithIntComparator()
	}
}

// SetTerm sets the coefficient for a term a.i within a Polynomial.
// For i=0, sets the constant term.
func (p Polynomial) SetTerm(i int, scale float64) Polynomial {
	p.checkTerms()
	p.Terms.Put(i, scale)
	return p
}

// Helper: for an equation [ 0 = p ] check if p is constant and != 0.
//
// Panics if true (for easier debugging).
func (p Polynomial) isOff() (float64, bool) {
	if coeff, isconst := p.IsConstant(); isconst {
		//coeff := p.getCoeffForTerm(0)
		if !arithm.Is0(coeff) {
			panic(fmt.Sprintf("equation off by %g", coeff))
		}
		return coeff, true
	}
	return 0.0, false
}

// Find coefficient of maximum absolute value.
// If parameter 'dependents' is given, first search for a.i * x.i, with
// x.i not in dependents (i.e., we're looking for free variables only:
// find free variable x.i in p, with abs(a.i) is max in p).
// If no free variable can be found, find max(dependent(a.j)).
//
func (p Polynomial) maxCoeff(dependents maps.Map) (int, float64) {
	p.checkTerms()
	it := p.Terms.Iterator()
	var maxp int      // variable position of max coeff
	var maxc = 0.0    // max coeff
	var coeff float64 // result coeff
	for it.Next() {
		i := it.Key().(int)
		var isdep = false
		if dependents != nil {
			_, isdep = dependents.Get(i) // could be better de-coupled by providing predicate func
		}
		if i == 0 || isdep {
			continue
		}
		c := p.GetCoeffForTerm(i)
		if math.Abs(c) > maxc {
			maxc, maxp, coeff = math.Abs(c), i, c
		}
	}
	if maxp == 0 && dependents != nil { // no free variable found
		maxp, coeff = p.maxCoeff(nil) //
	}
	if maxp == 0 {
		panic("I think this is an impossible error: seeing equation 0 = c")
	}
	return maxp, coeff
}

// Substitute variable i within p with Polynomial p2.
// If p does not contain a term.i, p is unchanged
// This routine is detructive!
//
func (p Polynomial) substitute(i int, p2 Polynomial) Polynomial {
	p.checkTerms()
	scale_i := p2.GetCoeffForTerm(i)
	if !arithm.Is0(scale_i) {
		panic(fmt.Sprintf("cyclic call to substitute term #%d: %s", i, p2.String()))
	}
	scale_i = p.GetCoeffForTerm(i)
	if !arithm.Is0(scale_i) { // variable i exists in p
		//log.Printf("# found x.%d scaled %s\n", i, scale_i.String())
		p.Terms.Remove(i)
		//log.Printf("# p/%d = %s\n", i, p)
		pp := p2.Multiply(NewConstantPolynomial(scale_i), true)
		//log.Printf("# p2 * %s = %s\n", scale_i, pp)
		p = p.Add(pp, true).Zap()
		//log.Printf("# p + p2 = %s\n", p)
	}
	return p
}

// CopyPolynomial makes a copy of a numeric Polynomial.
func (p Polynomial) CopyPolynomial() Polynomial {
	p1 := NewConstantPolynomial(0.0) // will become our return value
	p.checkTerms()
	it := p.Terms.Iterator()
	for it.Next() { // copy all terms of p into p1
		pos := it.Key().(int)
		scale := it.Value().(float64)
		p1.SetTerm(pos, scale)
	}
	return p1
}

// Internal method: add or subtract 2 polynomials. The high level methods
// are based on this one.
// Flag doAdd signals addition or subtraction.
//
func (p Polynomial) addOrSub(p2 Polynomial, doAdd bool, destructive bool) Polynomial {
	p.checkTerms()
	p1 := p.CopyPolynomial() // will become our return value
	it2 := p2.Terms.Iterator()
	for it2.Next() { // inspect all terms of p2
		pos2 := it2.Key().(int)
		scale2 := it2.Value().(float64)
		if !arithm.Is0(scale2) {
			scale1 := p1.GetCoeffForTerm(pos2)
			if doAdd {
				scale1 = scale1 + scale2 // if present, add a1 + a2
			} else {
				scale1 = scale1 - scale2 // if present, subtract a1 - a2
			}
			p1.SetTerm(pos2, scale1) // we operate on the copy p1
		}
	}
	if destructive {
		p.Terms = p1.Terms
	}
	return p1
}

// Add adds two Polynomials. Returns a new Polynomial, except when the
// 'destructive'-flag is set (then p is altered).
func (p Polynomial) Add(p2 Polynomial, destructive bool) Polynomial {
	/*
		if p.ispair {
			return p.AddPair(p2, destructive)
		} else {
			return p.addOrSub(p2, true, destructive)
		}
	*/
	p.checkTerms()
	return p.addOrSub(p2, true, destructive)
}

// Subtract subtracts two Polynomials. Returns a new Polynomial, except when the
// 'destructive'-flag is set (then p is altered).
func (p Polynomial) Subtract(p2 Polynomial, destructive bool) Polynomial {
	/*
		if p.ispair {
			return p.SubtractPair(p2, destructive)
		} else {
			return p.addOrSub(p2, false, destructive)
		}
	*/
	p.checkTerms()
	return p.addOrSub(p2, false, destructive)
}

// Multiply multiplys two Polynomials. One of both must be a constant.
// p2 will be destroyed.
func (p Polynomial) Multiply(p2 Polynomial, destructive bool) Polynomial {
	/*
		if p.ispair {
			return p.MultiplyPair(p2, destructive)
		} else {
	*/
	p.checkTerms()
	p1 := p.CopyPolynomial()      // will become our return value
	c, isconst := p2.IsConstant() // is p2 constant?
	if !isconst {
		c, isconst = p1.IsConstant() // is p1 constant?
		if !isconst {
			panic("not implemented: <unknown> * <unknown>")
		}
		p1 = p2 // swap to operate on p2
	}
	it := p1.Terms.Iterator()
	for it.Next() { // multiply all coefficients by c
		pos := it.Key().(int)
		scale := it.Value().(float64)
		p1.SetTerm(pos, arithm.Zap(scale*c))
	}
	if destructive {
		p.Terms = p1.Terms
	}
	p1 = p1.Zap()
	return p1
}

// Divide divides two polynomial by a numeric (not 0).
// p2 will be destroyed.
func (p Polynomial) Divide(p2 Polynomial, destructive bool) Polynomial {
	p.checkTerms()
	c, isconst := p2.IsConstant() // is p2 constant?
	if !isconst || arithm.Is0(c) {
		panic(fmt.Sprintf("illegal divisor: %s", p2.String()))
	} else {
		p2.Terms.Remove(0)
		p2.Terms.Put(0, 1.0/c) // now p2 = 1/c
	}
	return p.Multiply(p2, destructive)
}

// Zap eliminates all terms with coefficient=0 from a polynomial.
func (p Polynomial) Zap() Polynomial {
	p.checkTerms()
	positions := p.Terms.Keys()     // all non-Zero terms of p
	for _, pos := range positions { // inspect terms
		//if !(p.ispair && pos == 0) {
		if scale, _ := p.Terms.Get(pos); arithm.Is0(scale.(float64)) {
			p.Terms.Remove(pos) // may lose constant term c
		}
		//}
	}
	if _, ok := p.Terms.Get(0); !ok {
		p.Terms.Put(0, 0.0) // set p = 0: re-introduce c
	}
	//T.Debugf("# Zapped: %s", p.String())
	return p
}

// IsConstant checks wether
// a Polynomial is a constant, i.e. p = { c }? Returns the constant and a flag.
func (p Polynomial) IsConstant() (float64, bool) {
	/*
		if p.ispair {
			return p.GetConstantPair().x, p.Terms.Size() == 1
		} else {
			return p.GetCoeffForTerm(0), p.Terms.Size() == 1
		}
	*/
	return p.GetCoeffForTerm(0), p.Terms.Size() == 1
}

// IsVariable checks wether
// a Polynomial is a variable?, i.e. a single term with coefficient = 1.
// Returns the position of the term and a flag.
func (p Polynomial) IsVariable() (int, bool) {
	p.checkTerms()
	if p.Terms.Size() == 2 { // ok: p = a*x.i + c
		if arithm.Is0(p.GetCoeffForTerm(0)) { // if c == 0
			positions := p.Terms.Keys() // all non-Zero Terms of p, ordered
			pos := positions[1].(int)
			a := p.GetCoeffForTerm(pos)
			if arithm.Is1(a) { // if a.i = 0
				return pos, true
			}
		}
	}
	return -77777, false
}

// IsValid checks if this a correctly initialized polynomial.
func (p Polynomial) IsValid() bool {
	return (p.Terms != nil)
}

// GetConstantValue returns the constant term of a polynomial.
func (p Polynomial) GetConstantValue() float64 {
	return p.GetCoeffForTerm(0)
}

// GetCoeffForTerm gets the coefficient for term # i.
//
// Example:
//     p = x + 3x.2
// ⇒
//    coeff(2) = 3
//
func (p Polynomial) GetCoeffForTerm(i int) float64 {
	var sc interface{}
	var found bool
	p.checkTerms()
	sc, found = p.Terms.Get(i)
	if found {
		return sc.(float64)
	}
	return 0.0
}

// ArityComparator is a
// Comparator for polynomials. Polynomials are "smaller" if their arity
// is smaller, i.e. they have less unknown variables.
//
func ArityComparator(polyn1, polyn2 interface{}) int {
	p1, _ := polyn1.(Polynomial)
	p2, _ := polyn2.(Polynomial)
	if p1.Terms == nil {
		if p1.Terms == nil {
			return 0
		}
		return -1
	} else if p2.Terms == nil {
		return 1
	}
	T().Debugf("|p1| = %d, |p2| = %d", p1.Terms.Size(), p2.Terms.Size())
	return p1.Terms.Size() - p2.Terms.Size()
}

// String creates a readable string representation for a Polynomial.
// Uses internal variable representations x.<n> where n corresponds to
// the variable's real life ID.
func (p Polynomial) String() string {
	return p.TraceString(nil)
}

// TraceString creates a string representation for a Polynomial. Uses a variable name
// resolver to print 'real' variable identifiers. If no resolver is
// present, variables are printed in a generic form: +/- a.i x.i, where i is
// the position of the term. Coefficients are rounded to the 3rd place.
func (p Polynomial) TraceString(resolv VariableResolver) string {
	var buffer bytes.Buffer
	p.checkTerms()
	it := p.Terms.Iterator()
	var indent = false // no space before first term (usually constant)
	for it.Next() {
		pos := it.Key().(int)
		if pos == 0 { // constant term
			/*
				if p.ispair {
					pc := it.Value().(Pair)
				} else {
					pc := it.Value().(float64).Round(3)
				}
			*/
			pc := it.Value().(float64)
			if resolv == nil {
				buffer.WriteString(fmt.Sprintf("{ %g } ", arithm.Round(pc)))
			} else {
				if !arithm.Is0(pc) {
					buffer.WriteString(fmt.Sprintf("%g", arithm.Round(pc)))
					indent = true
				}
			}
		} else { // variable term
			scale := it.Value().(float64)
			if resolv == nil {
				buffer.WriteString(fmt.Sprintf("{ %g x.%d } ",
					arithm.Round(scale), pos))
			} else {
				if indent {
					if scale < 0.0 {
						buffer.WriteString(" - ")
					} else if scale > 0.0 {
						buffer.WriteString(" + ")
					}
				} else {
					indent = true
					if scale < 0.0 {
						buffer.WriteString("-")
					}
				}
				if !arithm.Is0(math.Abs(scale) - 1.0) {
					buffer.WriteString(fmt.Sprintf("%g", scale))
				}
				buffer.WriteString(resolv.GetVariableName(pos))
			}
		}
	}
	return buffer.String()
}

// TraceStringVar is a helper for tracing output. Parameter resolv may be nil.
func TraceStringVar(i int, resolv VariableResolver) string {
	if resolv == nil {
		return fmt.Sprintf("x.%d", i)
	}
	return resolv.GetVariableName(i)
}
