package polyn

import (
	"fmt"

	"github.com/npillmayer/arithm"

	"github.com/emirpasic/gods/maps"
	"github.com/emirpasic/gods/maps/treemap"
)

/*

BSD License

Copyright (c) 2017–21, Norbert Pillmayer

All rights reserved.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions
are met:

1. Redistributions of source code must retain the above copyright
notice, this list of conditions and the following disclaimer.

2. Redistributions in binary form must reproduce the above copyright
notice, this list of conditions and the following disclaimer in the
documentation and/or other materials provided with the distribution.

3. Neither the name of this software nor the names of its contributors
may be used to endorse or promote products derived from this software
without specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
"AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
HOLDER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
(INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.


----------------------------------------------------------------------

Objects and interfaces for solving systems of linear equations (LEQ).

Inspired by Donald E. Knuth's MetaFont, John Hobby's MetaPost and by
a Lua project by John D. Ramsdell: http://luaforge.net/projects/lineqpp/
*/

// A VariableResolver links to variables.
// We use an interface to resolve "real" variable names. Within the LEQ
// variables are encoded by their serial ID, which is used as their position
// within polynomias. Example: variable "n[3].a" with ID=4711 will become x.4711
// internally. The resolver maps x.4711 ⟼ "n[3].a", i.e., IDs to names.
type VariableResolver interface {
	GetVariableName(int) string     // get real-life name of x.i
	SetVariableSolved(int, float64) // message: x.i is solved
	IsCapsule(int) bool             // x.i has gone out of scope
}

// === System of linear equations =======================================

// LinEqSolver is a container for linear equations. Used to incrementally solve
// systems of linear equations.
//
// Inspired by Donald E. Knuth's MetaFont, John Hobby's MetaPost and by
// a Lua project by John D. Ramsdell: http://luaforge.net/projects/lineqpp/
//
type LinEqSolver struct {
	dependents       *treemap.Map     // dependent variable at position i has dependencies[i]
	solved           *treemap.Map     // map x.i => numeric
	varresolver      VariableResolver // to resolve variable names from term positions
	showdependencies bool             // continuously show dependent variables
}

// CreateLinEqSolver creates a new sytem of linear equations.
func CreateLinEqSolver() *LinEqSolver {
	leq := LinEqSolver{
		dependents:       treemap.NewWithIntComparator(), // sorted map
		solved:           treemap.NewWithIntComparator(), // sorted map
		showdependencies: false,
	}
	return &leq
}

// SetVariableResolver sets a variable resolver. Within the LEQ variables are
// encoded by their serial ID, which is used as their position within
// polynomias. Example: variable "n[3].a" with ID=4711 will become x.4711
// internally. The resolver maps "x.4711" to "n[3].a".
func (leq *LinEqSolver) SetVariableResolver(resolver VariableResolver) {
	leq.varresolver = resolver
}

// Collect all currently solved variables from a system of linear equations.
// Solved variables are returned as a map: i(var) -> numeric, where i(var) is an
// integer representing the position of variable var.
func (leq *LinEqSolver) getSolvedVars() maps.Map {
	setOfSolved := treemap.NewWithIntComparator() // return value
	it := leq.solved.Iterator()
	for it.Next() { // for every x.i = p[x.i = c] => put [x.i = c] into new set
		setOfSolved.Put(it.Key().(int), it.Value().(Polynomial).GetCoeffForTerm(0))
	}
	return setOfSolved
}

// AddEq adds a
// new equation 0 = p (p is Polynomial) to a system of linear equations.
// Immediately starts to solve the -- possibly incomplete -- system, as
// far as possible.
func (leq *LinEqSolver) AddEq(p Polynomial) *LinEqSolver {
	leq.addEq(p, false)
	if leq.showdependencies {
		leq.Dump(leq.varresolver)
	}
	return leq
}

// AddEqs adds a set of linear equations to the LEQ system.
// See AddEq.
func (leq *LinEqSolver) AddEqs(plist []Polynomial) *LinEqSolver {
	l := len(plist)
	if l == 0 {
		T().Errorf("given empty list of equations")
	} else {
		for i, p := range plist {
			T().Debugf("adding equation %d/%d: 0 = %s", i+1, l, p)
			leq.addEq(p, i+1 < l)
		}
	}
	if leq.showdependencies {
		leq.Dump(leq.varresolver)
	}
	return leq
}

// If parameter cont is true, expect another equation immediately after this
// one. This is necessary to suppress harvesting of capsules.
func (leq *LinEqSolver) addEq(p Polynomial, cont bool) *LinEqSolver {
	p = p.Zap()
	T().P("op", "new equation").Infof("0 = %s", leq.PolynString(p))
	// substitute solved in new equation
	p = leq.substituteSolved(0, p, leq.solved)
	if _, off := p.isOff(); !off { //  :-))  no pun intended
		// select x.i=p(i)
		i, _ := p.maxCoeff(leq.dependents)    // start with max (free) coefficient of p
		p = leq.activateEquationTowards(i, p) // now  x.i = -1/a * p(...).
		// Phase 1: substitute P(i) in every x.j=P(j)
		D := leq.updateDependentVariables(i, p)
		// done, now split solved x from D' off to S'
		S := treemap.NewWithIntComparator() // set up S' of solved
		itD := D.Iterator()
		for itD.Next() { // for every x.i=p(i) in D'
			i, p = itD.Key().(int), itD.Value().(Polynomial)
			if ok, rhs := solved(p); ok {
				S.Put(i, rhs) // add x.i to S'
				D.Remove(i)   // remove x.i from D'
			}
		}
		// substitute solved: subst s in S' into d in D'
		//T.Info("---------- subst solved -----------")
		itD = D.Iterator()
		for itD.Next() { // for every x.i=p(i) in D'
			i, p = itD.Key().(int), itD.Value().(Polynomial)
			p = leq.substituteSolved(i, p, S)
			if ok, rhs := solved(p); ok {
				S.Put(i, rhs) // add x.i to S'
				D.Remove(i)   // remove x.i from D'
			}
		}
		//T.Info("-----------------------------------")
		// done, update sets S and D
		S.Each(func(key interface{}, value interface{}) { // S = S + S'
			leq.setSolved(key.(int), value.(Polynomial))
		})
		leq.dependents = D // D = D'
	}
	if !cont { // if this equation is not part of an equation-pair
		leq.harvestCapsules()
	}
	return leq
}

// 1st pass of the LEQ algorithm: with a new equation x.i=P(i) walk
// through all dependent variables x.j=P(j) and substitute P(i) for x.i
// in every RHS.
// Return a new set D' of dependent variables.
func (leq *LinEqSolver) updateDependentVariables(i int, p Polynomial) *treemap.Map {
	D := treemap.NewWithIntComparator() // set up D' of dependents
	leq.updateDependency(i, p, D)
	// D -> D'
	it := leq.dependents.Iterator() // for all dependent x.j=q(j)
	savei := i
	T().Debugf("---------- subst dep --------------")
	for it.Next() { // iterate over all dependent variables
		i = savei // restore i
		tmp, _ := D.Get(i)
		p = tmp.(Polynomial).CopyPolynomial() // get current version of p(i)
		j, q := it.Key().(int), it.Value().(Polynomial)
		T().P("op", "substitute").Debugf("(1) p(%s) in %s = %s",
			leq.VarString(i), leq.VarString(j), leq.PolynString(q))
		if j == i { // x.j = x.i, i.e. equations with identical LHS
			k, _ := q.maxCoeff(D)                 // start with max (free) coefficient of q(j=i)
			lhs := NewConstantPolynomial(0.0)     // construct LHS as pp
			lhs.SetTerm(j, -1.0)                  // now LHS is { 0 - 1 x.j }
			q = q.Add(lhs, false)                 // move to RHS
			q = leq.activateEquationTowards(k, q) // now  x.k = -1/a.k * p(... x.j ...).
			j = k                                 // ride the new horse
		}
		T().P("op", "substitute").Debugf("(2) p(%s) in %s = %s",
			leq.VarString(i), leq.VarString(j), leq.PolynString(q))
		leq.updateDependency(j, q, D) // insert original dependency
		if !termContains(q, i) && termContains(p, j) {
			i, j = j, i
			p, q = q, p
		}
		T().P("op", "substitute").Debugf("(3) p(%s) in %s = %s",
			leq.VarString(i), leq.VarString(j), leq.PolynString(q))
		if termContains(q, i) {
			j, q = subst(i, p, j, q) // substitute new equation in x.j=q(j)
			T().P("op", "substitute").Debugf("result: %s = %s", leq.VarString(j), leq.PolynString(q))
			if j != 0 {
				leq.updateDependency(j, q, D) // insert substitution result
			} else { // j has been eliminated from q
				if _, off := q.isOff(); !off {
					k, _ := q.maxCoeff(D) // find max (free) coefficient of q(k)
					q = leq.activateEquationTowards(k, q)
					leq.updateDependency(k, q, D) // insert new equation
				}
			}
		}
	}
	T().Debugf("-----------------------------------")
	return D
}

// Check if a polynomial is constant, i.e. solves an equation.
func solved(p Polynomial) (bool, Polynomial) {
	if rhs, isconst := p.IsConstant(); isconst {
		rhs = arithm.Round(rhs) // round to epsilon
		p = p.SetTerm(0, rhs)   // replace const coeff by rounded value
		return true, p
	}
	return false, p
}

// Does this polynomial contain x.i ?
func termContains(p Polynomial, i int) bool {
	return !arithm.Is0(p.GetCoeffForTerm(i))
}

// Insert or replace x.i=p(i) in a set of equations.
func (leq *LinEqSolver) updateDependency(i int, p Polynomial, m *treemap.Map) {
	p = p.CopyPolynomial()
	//fmt.Printf("inserting x.%d = %v\n", i, p)
	if q, found := m.Get(i); found {
		//fmt.Printf("found     x.%d = %v\n", i, q)
		if termlength(p) < termlength(q.(Polynomial)) { // prefer shorter RHS terms
			varname := leq.VarString(i)
			T().P("var", varname).Infof("## %s = %s", varname, leq.PolynString(p))
			m.Put(i, p) // replace equation x.i=p(i)
		}
	} else {
		m.Put(i, p) // insert new equation x.i=p(i)
	}
	/*
		pp, ok := m.Get(i)
		if !ok {
			T.Errorf("not found: x.%d", i)
		}
		fmt.Printf("now       x.%d = %v\n", i, pp)
	*/
}

// Substitute term x.i=p(i) for x.i in q(j). p(i) may contain a.j*x.j,
// resulting in an equation x.j=q(j) with x.j in q(j). We then resolve
// for x.j. This may result in the elimination of x.j. We then return 0=q'.
//
// Returns the resulting - possibly new - equation.
func subst(i int, p Polynomial, j int, q Polynomial) (int, Polynomial) {
	ai := q.GetCoeffForTerm(i) // a.i in q
	if !arithm.Is0(ai) {       // if variable x.i exists in q
		q.Terms.Remove(i)                               // remove a.i*x.i in q (to be replaced)
		p = p.Multiply(NewConstantPolynomial(ai), true) // scale p(i) by a.i of q
		q = q.Add(p, false).Zap()                       // now insert p(i) into q(j)
		aj := q.GetCoeffForTerm(j)                      // results in a.j*x.j in q(j) ?
		if arithm.Is0(aj) {                             // no => we're done
			// do nothing
		} else if arithm.Is1(aj) { // x.j = c + x.j + ...  => eliminate x.j and activate for free x.k
			q.Terms.Remove(j) // remove x.j from RHS q
			j = 0             // set LHS to 'impossible' variable x.0
		} else { // x.j = c + a.j*x.j + ...  => scale RHS by -1(a.j-1)
			a := -1.0 / (aj - 1.0)         // a = -1/(a.j-1)
			c := NewConstantPolynomial(a)  //
			q.Terms.Remove(j)              // now remove a.j*x.j from RHS q
			q = q.Multiply(c, false).Zap() // and multiply RHS by -1/(a.j-1)
		}
	}
	return j, q // return x.j = q'(j)

}

// Helper: number of variables in RHS of an equation.
func termlength(p Polynomial) int {
	return p.Terms.Size()
}

// In an equation, substitute all variables which are already known.
func (leq *LinEqSolver) substituteSolved(j int, p Polynomial, solved *treemap.Map) Polynomial {
	//it := leq.solved.Iterator()
	it := solved.Iterator()
	T().Debugf("---------- subst solved -----------")
	for it.Next() { // iterate over all solved x.i = c
		i := it.Key().(int)
		c := it.Value().(Polynomial).GetConstantValue()
		coeff := p.GetCoeffForTerm(i)
		if !arithm.Is0(coeff) {
			coeff = coeff * c
			pc := p.GetConstantValue()
			p.SetTerm(0, pc+coeff)
			p.Terms.Remove(i)
			T().P("op", "subst-solved").Debugf("%s = %g  =>  RHS = %s",
				leq.VarString(i), c, leq.PolynString(p))
			if j > 0 {
				varname := leq.VarString(j)
				T().P("var", varname).Infof("## %s = %s", varname, leq.PolynString(p))
			} else {
				T().P("op", "subst known").Infof("# 0 = %s", leq.PolynString(p))
			}
		}
	}
	T().Debugf("-----------------------------------")
	return p
}

// Transform an equation 0 = p(a x.i) to make x.i the dependent variable, i.e.
// x.i = -1/a * p(...).
//
func (leq *LinEqSolver) activateEquationTowards(i int, p Polynomial) Polynomial {
	coeff := p.GetCoeffForTerm(i)
	p.Terms.Remove(i) // remove term x.i from RHS(p)
	pp := NewConstantPolynomial(-1.0 / coeff)
	p = p.Multiply(pp, true).Zap()
	//T.P("op", "activate").Infof("## %s = %s", leq.VarString(i), leq.PolynString(p))
	varname := leq.VarString(i)
	T().P("var", varname).Infof("## %s = %s", varname, leq.PolynString(p))
	return p
}

// Mark a variable as solved. Sends a message to the variable resolver.
func (leq *LinEqSolver) setSolved(i int, p Polynomial) {
	c := p.GetConstantValue()
	varname := leq.VarString(i)
	T().P("var", varname).Infof("#### %s = %g", varname, c)
	leq.solved.Put(i, p) // move x.i to set of solved variables
	if leq.varresolver != nil {
		leq.varresolver.SetVariableSolved(i, c) // notify variable solver
	}
}

// VarString returns a readable variable name for an internal variable.
// Uses a VariableResolver, if present.
func (leq *LinEqSolver) VarString(i int) string {
	if leq.varresolver == nil {
		return fmt.Sprintf("x.%d", i)
	}
	return leq.varresolver.GetVariableName(i)
}

// PolynString outputs a polynomial as string. Uses VariableResolver, if present.
func (leq *LinEqSolver) PolynString(p Polynomial) string {
	if leq.varresolver != nil {
		return p.TraceString(leq.varresolver)
	}
	return p.String()
}

// === Capsules ==============================================================

/* 'Capsule' is a MetaFont terminus for variables in the LEQ, which have
 * fallen out of scope. This may happen on "endgroup", if a variable has
 * been "save"d, or with assignments, where the old incarnation of an
 * lvalue may still be entangled in the LEQ. Capsules may still be relevant
 * in the LEQ, but are of no further interest to the user.
 *
 * The typical case in MetaFont ist the use of "whatever", e.g. in the
 * equation z0 = whatever[z1,z2] (z0 is somewhere on the straight line
 * trough z1 and z2). "whatever" is defined as "begingroup save ?; ? endgroup".
 * The variable ? falls out of scope, but is still relevant for solving the
 * equations for z0 (the above command produces 2 equations).
 */

// Remove all equations which are dependent on a capsule, but only if the
// capsule is a loner. If a capsule occurs in at least 2 equations, it
// is still relevant for solving the LEQ.
func (leq *LinEqSolver) harvestCapsules() {
	var counts = make(map[int]int)
	it := leq.dependents.Iterator()
	for it.Next() { // iterate over all dependent x.w = p.w ( c ... { a x.v } ... )
		w := it.Key().(int)
		pw := it.Value().(Polynomial)
		leq.checkAndCountCapsule(w, counts) // check LHS variable
		pit := pw.Terms.Iterator()          // for all terms in polynomial
		for pit.Next() {
			i := pit.Key().(int) // get every term.i
			if i > 0 {           // omit constant term
				leq.checkAndCountCapsule(i, counts)
			}
		}
	}
	itsolv := leq.solved.Iterator() // count solved capsules
	for itsolv.Next() {
		j := itsolv.Key().(int)
		leq.checkAndCountCapsule(j, counts)
	}
	for pos, count := range counts { // now remove capsules with count == 1
		if count == 1 { // only remove loners
			T().P("capsule", pos).Debugf("capsule removed")
			leq.retractVariable(pos)
		}
	}
}

// Helper for counting capsule references. Updates the count for a capsule.
func (leq *LinEqSolver) checkAndCountCapsule(i int, counts map[int]int) {
	if leq.varresolver != nil && leq.varresolver.IsCapsule(i) {
		counts[i]++
		//T.P("capsule", i).Debugf("capsule counted, #=%d", counts[i])
	}
}

/* If a capsule is removed, all equations containing the capsule must
 * be deleted from the LEQ.
 *
 * TODO: The whole procedure for removing capsules is awfully inefficient:
 * lots of set iterations (some nested loops) and set creations. But for
 * my use cases the number of simultaneous equations is small, therefore
 * I'll clean this up sometime later... :-)
 */
func (leq *LinEqSolver) retractVariable(i int) {
	if _, ok := leq.solved.Get(i); ok {
		T().Debugf("unsolve %s", leq.VarString(i))
		leq.solved.Remove(i)
	}
	leq.dependents.Remove(i)              // possibly remove from dependents
	eqs := treemap.NewWithIntComparator() // set of equation indices, i.e. int
	it := leq.dependents.Iterator()
	for it.Next() { // iterate over all dependent x.j = p.i ( c ... { a x.i } ... )
		j := it.Key().(int)
		p := it.Value().(Polynomial)
		if a := p.GetCoeffForTerm(i); !arithm.Is0(a) { // yes, x.i in p
			eqs.Put(j, p) // mark for deletion, as it is invalid now
		}
	}
	it = eqs.Iterator()
	for it.Next() { // iterate over marked equations
		leq.dependents.Remove(it.Key().(int))
	}
}

// === Utilities =============================================================

// Dump is a debugging helper to dump all known equations.
func (leq *LinEqSolver) Dump(resolv VariableResolver) {
	fmt.Println("----------------------------------------------------------------------")
	fmt.Println("Dependents:                                                        LEQ")
	it := leq.dependents.Iterator()
	for it.Next() { // for every x.i = p[x.i]
		k := it.Key().(int)
		p := it.Value().(Polynomial)
		fmt.Printf("\t%s = %s\n", TraceStringVar(k, resolv), p.TraceString(resolv))
	}
	fmt.Println("Solved:")
	it = leq.solved.Iterator()
	for it.Next() { // for every x.i = { c }
		k := it.Key().(int)
		p := it.Value().(Polynomial)
		fmt.Printf("\t%s = %g\n", TraceStringVar(k, resolv), p.GetConstantValue())
	}
	fmt.Println("----------------------------------------------------------------------")
}
