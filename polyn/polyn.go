// Package polyn is for arithmetic with linear polynomials and linear equations.
/*
BSD 3-Clause License

Copyright (c) Norbert Pillmayer.

All rights reserved.

Please refer to the license file for more information.
*/
package polyn

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"sort"

	"github.com/npillmayer/arithm"
	"github.com/npillmayer/schuko/gtrace"
	"github.com/npillmayer/schuko/tracing"
)

var (
	// ErrNonConstantProduct is returned if neither multiplicand is constant.
	ErrNonConstantProduct = errors.New("cannot multiply two non-constant polynomials")
	// ErrIllegalDivisor is returned if divisor is zero or not constant.
	ErrIllegalDivisor = errors.New("illegal divisor polynomial")
)

// T traces to the equations tracer.
func T() tracing.Trace {
	return gtrace.EquationsTracer
}

// X is a helper for quick construction of polynomials.
// It denotes a term
//
//	C*x^I
//
// I > 0
//
// In LEQ use-cases, the same integer I is also used as a variable ID/key.
type X struct {
	I int     // exponent in C*x^I; also LEQ variable ID/key
	C float64 // coefficient
}

// New creates a polynomial from a constant term and a set of exponent terms.
//
// Use it as
//
//	polyn.New(8, polyn.X{2, 5}, polyn.X{1, 2.0/3.0})
//
// to get
//
//	P(x) = 8 + 5*x^2 + 2/3*x
func New(c float64, tms ...X) (Polynomial, error) { // construct a polynomial
	p := NewConstantPolynomial(c)
	var err error
	for _, t := range tms {
		if t.I < 1 {
			err = fmt.Errorf("term exponent must be at least 1, skipping it")
		} else {
			p.SetTerm(t.I, t.C)
		}
	}
	return p, err
}

// Polynomial is a type for linear polynomials
//
//	c + a_1*x^1 + a_2*x^2 + ... + a_n*x^n
//
// Mathematically this package models terms by exponent i. Implementation-wise,
// coefficients are stored sparsely in a map keyed by i, with key 0 for the
// constant term. In LEQ use-cases the same key i is interpreted as variable ID x.i.
type Polynomial struct {
	terms map[int]float64
}

// NewConstantPolynomial creates a Polynomial consisting of just a constant term.
func NewConstantPolynomial(c float64) Polynomial {
	p := Polynomial{}
	p.checkTerms()
	p.terms[0] = c // initialize with constant term (at position 0)
	return p.Zap()
}

func (p *Polynomial) checkTerms() {
	if p.terms == nil {
		p.terms = make(map[int]float64)
	}
}

// SetTerm sets the coefficient for a term a.i within a Polynomial.
// For i=0, sets the constant term.
func (p Polynomial) SetTerm(i int, scale float64) Polynomial {
	p.checkTerms()
	p.terms[i] = scale
	return p
}

// TermCount returns the number of stored terms, including the constant term.
func (p Polynomial) TermCount() int {
	return p.termMapSize()
}

func (p Polynomial) termMapSize() int {
	p.checkTerms()
	return len(p.terms)
}

// Exponents returns the currently stored term exponents in ascending order.
func (p Polynomial) Exponents() []int {
	p.checkTerms()
	exponents := make([]int, 0, len(p.terms))
	for key := range p.terms {
		exponents = append(exponents, key)
	}
	sort.Ints(exponents)
	return exponents
}

// Adapter helper for deterministic ascending iteration over terms.
func (p Polynomial) forEachTermAscending(fn func(int, float64)) {
	p.checkTerms()
	for _, key := range p.Exponents() {
		fn(key, p.terms[key])
	}
}

// Helper: for an equation [ 0 = p ] check if p is constant.
// The returned coefficient may be non-zero, signaling an inconsistent equation.
func (p Polynomial) isOff() (float64, bool) {
	if coeff, isconst := p.IsConstant(); isconst {
		return coeff, true
	}
	return 0.0, false
}

// Find coefficient of maximum absolute value.
// If parameter 'dependents' is given, first search for a.i * x.i, with
// x.i not in dependents (i.e., we're looking for free variables only:
// find free variable x.i in p, with abs(a.i) is max in p).
// If no free variable can be found, find max(dependent(a.j)).
func (p Polynomial) maxCoeff(dependents map[int]Polynomial) (int, float64) {
	p.checkTerms()
	var maxp int      // variable position of max coeff
	var maxc = 0.0    // max coeff
	var coeff float64 // result coeff
	p.forEachTermAscending(func(i int, c float64) {
		var isdep = false
		if dependents != nil {
			_, isdep = dependents[i] // could be better de-coupled by providing predicate func
		}
		if i == 0 || isdep {
			return
		}
		if math.Abs(c) > maxc {
			maxc, maxp, coeff = math.Abs(c), i, c
		}
	})
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
func (p Polynomial) substitute(i int, p2 Polynomial) Polynomial {
	p.checkTerms()
	scale_i := p2.GetCoeffForTerm(i)
	if !arithm.Is0(scale_i) {
		panic(fmt.Sprintf("cyclic call to substitute term #%d: %s", i, p2.String()))
	}
	scale_i = p.GetCoeffForTerm(i)
	if !arithm.Is0(scale_i) { // variable i exists in p
		//log.Printf("# found x.%d scaled %s\n", i, scale_i.String())
		delete(p.terms, i)
		//log.Printf("# p/%d = %s\n", i, p)
		pp, err := p2.Multiply(NewConstantPolynomial(scale_i), true)
		if err != nil {
			panic(err) // internal invariant: second factor is constant
		}
		//log.Printf("# p2 * %s = %s\n", scale_i, pp)
		p = p.Add(pp, true).Zap()
		//log.Printf("# p + p2 = %s\n", p)
	}
	return p
}

// CopyPolynomial makes a copy of a numeric Polynomial.
func (p Polynomial) CopyPolynomial() Polynomial {
	p1 := NewConstantPolynomial(0.0)                      // will become our return value
	p.forEachTermAscending(func(pos int, scale float64) { // copy all terms of p into p1
		p1.SetTerm(pos, scale)
	})
	return p1
}

// Internal method: add or subtract 2 polynomials. The high level methods
// are based on this one.
// Flag doAdd signals addition or subtraction.
func (p Polynomial) addOrSub(p2 Polynomial, doAdd bool, destructive bool) Polynomial {
	p.checkTerms()
	p2.checkTerms()
	p1 := p.CopyPolynomial()                                 // will become our return value
	p2.forEachTermAscending(func(pos2 int, scale2 float64) { // inspect all terms of p2
		if !arithm.Is0(scale2) {
			scale1 := p1.GetCoeffForTerm(pos2)
			if doAdd {
				scale1 = scale1 + scale2 // if present, add a1 + a2
			} else {
				scale1 = scale1 - scale2 // if present, subtract a1 - a2
			}
			p1.SetTerm(pos2, scale1) // we operate on the copy p1
		}
	})
	if destructive {
		p.terms = p1.terms
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

// Multiply multiplies two Polynomials. One of both must be a constant.
func (p Polynomial) Multiply(p2 Polynomial, destructive bool) (Polynomial, error) {
	/*
		if p.ispair {
			return p.MultiplyPair(p2, destructive)
		} else {
	*/
	p.checkTerms()
	p1 := p.CopyPolynomial() // will become our return value
	p2c := p2.CopyPolynomial()
	c, isconst := p2c.IsConstant() // is p2 constant?
	if !isconst {
		c, isconst = p1.IsConstant() // is p1 constant?
		if !isconst {
			return Polynomial{}, fmt.Errorf("%w: %s * %s", ErrNonConstantProduct, p.String(), p2.String())
		}
		p1 = p2c // swap to operate on a copy of p2
	}
	p1.forEachTermAscending(func(pos int, scale float64) { // multiply all coefficients by c
		p1.SetTerm(pos, arithm.Zap(scale*c))
	})
	if destructive {
		p.terms = p1.terms
	}
	p1 = p1.Zap()
	return p1, nil
}

// Divide divides a polynomial by a non-zero numeric polynomial.
func (p Polynomial) Divide(p2 Polynomial, destructive bool) (Polynomial, error) {
	p.checkTerms()
	c, isconst := p2.IsConstant() // is p2 constant?
	if !isconst || arithm.Is0(c) {
		return Polynomial{}, fmt.Errorf("%w: %s", ErrIllegalDivisor, p2.String())
	}
	return p.Multiply(NewConstantPolynomial(1.0/c), destructive)
}

// Zap eliminates all terms with coefficient=0 from a polynomial.
func (p Polynomial) Zap() Polynomial {
	p.checkTerms()
	positions := p.Exponents()      // all non-Zero terms of p
	for _, pos := range positions { // inspect terms
		//if !(p.ispair && pos == 0) {
		if scale := p.GetCoeffForTerm(pos); arithm.Is0(scale) {
			delete(p.terms, pos) // may lose constant term c
		}
		//}
	}
	if _, ok := p.terms[0]; !ok {
		p.terms[0] = 0.0 // set p = 0: re-introduce c
	}
	//T.Debugf("# Zapped: %s", p.String())
	return p
}

// IsConstant checks wether
// a Polynomial is a constant, i.e. p = { c }? Returns the constant and a flag.
func (p Polynomial) IsConstant() (float64, bool) {
	/*
		if p.ispair {
			return p.GetConstantPair().x, p.terms.Size() == 1
		} else {
			return p.GetCoeffForTerm(0), p.terms.Size() == 1
		}
	*/
	return p.GetCoeffForTerm(0), p.termMapSize() == 1
}

// IsVariable checks wether
// a Polynomial is a variable?, i.e. a single term with coefficient = 1.
// Returns the position of the term and a flag.
func (p Polynomial) IsVariable() (int, bool) {
	p.checkTerms()
	if p.termMapSize() == 2 { // ok: p = a*x.i + c
		if arithm.Is0(p.GetCoeffForTerm(0)) { // if c == 0
			positions := p.Exponents() // all non-Zero terms of p, ordered
			pos := positions[1]
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
	return (p.terms != nil)
}

// GetConstantValue returns the constant term of a polynomial.
func (p Polynomial) GetConstantValue() float64 {
	return p.GetCoeffForTerm(0)
}

// GetCoeffForTerm gets the coefficient for term # i.
//
// Example:
//
//	p = x + 3x.2
//
// ⇒
//
//	coeff(2) = 3
func (p Polynomial) GetCoeffForTerm(i int) float64 {
	p.checkTerms()
	if sc, found := p.terms[i]; found {
		return sc
	}
	return 0.0
}

// ArityComparator is a
// Comparator for polynomials. Polynomials are "smaller" if their arity
// is smaller, i.e. they have less unknown variables.
func ArityComparator(polyn1, polyn2 interface{}) int {
	p1, _ := polyn1.(Polynomial)
	p2, _ := polyn2.(Polynomial)
	if p1.terms == nil {
		if p1.terms == nil {
			return 0
		}
		return -1
	} else if p2.terms == nil {
		return 1
	}
	T().Debugf("|p1| = %d, |p2| = %d", p1.termMapSize(), p2.termMapSize())
	return p1.termMapSize() - p2.termMapSize()
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
	var indent = false // no space before first term (usually constant)
	p.forEachTermAscending(func(pos int, scale float64) {
		if pos == 0 { // constant term
			/*
				if p.ispair {
					pc := it.Value().(Pair)
				} else {
					pc := it.Value().(float64).Round(3)
				}
			*/
			pc := scale
			if resolv == nil {
				buffer.WriteString(fmt.Sprintf("{ %g } ", arithm.Round(pc)))
			} else {
				if !arithm.Is0(pc) {
					buffer.WriteString(fmt.Sprintf("%g", arithm.Round(pc)))
					indent = true
				}
			}
		} else { // variable term
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
	})
	return buffer.String()
}

// TraceStringVar is a helper for tracing output. Parameter resolv may be nil.
func TraceStringVar(i int, resolv VariableResolver) string {
	if resolv == nil {
		return fmt.Sprintf("x.%d", i)
	}
	return resolv.GetVariableName(i)
}
