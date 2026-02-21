package polyn

import (
	"errors"
	"fmt"
	"sort"

	"github.com/npillmayer/arithm"
)

var (
	// ErrEmptyEquationList indicates no equations were supplied to AddEqs.
	ErrEmptyEquationList = errors.New("empty list of equations")
	// ErrInconsistentEquation indicates an equation reduced to 0 = c with c != 0.
	ErrInconsistentEquation = errors.New("inconsistent equation")
)

/*
----------------------------------------------------------------------

Objects and interfaces for solving systems of linear equations (LEQ).

Inspired by Donald E. Knuth's MetaFont, John Hobby's MetaPost and by
a Lua project by John D. Ramsdell: http://luaforge.net/projects/lineqpp/
*/

// A VariableResolver links solver variable IDs to "real" variable names.
//
// In polynomial notation, terms are keyed by exponent i (a_i * x^i). In LEQ
// mode, the same key i is interpreted as an internal variable ID x.i.
// Example: variable "n[3].a" with ID=4711 is represented as x.4711
// internally. The resolver maps x.4711 to "n[3].a", i.e., IDs to names.
type VariableResolver interface {
	GetVariableName(int) string     // get real-life name of x.i
	SetVariableSolved(int, float64) // message: x.i is solved
	IsCapsule(int) bool             // x.i has gone out of scope
}

type EquationMap map[int]Polynomial
type SolvedMap map[int]Polynomial

// === System of linear equations =======================================

// LinEqSolver is a container for linear equations. Used to incrementally solve
// systems of linear equations.
//
// Inspired by Donald E. Knuth's MetaFont, John Hobby's MetaPost and by
// a Lua project by John D. Ramsdell: http://luaforge.net/projects/lineqpp/
type LinEqSolver struct {
	dependents       EquationMap      // dependent variable at position i has dependencies[i]
	solved           SolvedMap        // map x.i => numeric
	varresolver      VariableResolver // to resolve variable names from term positions
	showdependencies bool             // continuously show dependent variables
}

// NewLinEqSolver creates a new system of linear equations.
func NewLinEqSolver() *LinEqSolver {
	leq := LinEqSolver{
		dependents:       make(EquationMap),
		solved:           make(SolvedMap),
		showdependencies: false,
	}
	return &leq
}

// Adapter helper to keep deterministic ascending iteration over equation maps.
// Keys are snapshotted so callbacks may remove entries from m safely.
func forEachEquationAscending(m map[int]Polynomial, fn func(int, Polynomial) error) error {
	if m == nil {
		return nil
	}
	keys := make([]int, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	for _, i := range keys {
		v, ok := m[i]
		if !ok { // key may have been removed by callback
			continue
		}
		if err := fn(i, v); err != nil {
			return err
		}
	}
	return nil
}

func equationMapSize(m map[int]Polynomial) int {
	if m == nil {
		return 0
	}
	return len(m)
}

// SetVariableResolver sets a variable resolver. Within the LEQ variables are
// encoded by their serial ID, i.e. by the term key/exponent i in this package's
// sparse term map. Example: variable "n[3].a" with ID=4711 will become x.4711
// internally. The resolver maps "x.4711" to "n[3].a".
func (leq *LinEqSolver) SetVariableResolver(resolver VariableResolver) {
	leq.varresolver = resolver
}

// Collect all currently solved variables from a system of linear equations.
// Solved variables are returned as a map: i(var) -> numeric, where i(var) is an
// integer representing the position of variable var.
func (leq *LinEqSolver) getSolvedVars() map[int]float64 {
	setOfSolved := make(map[int]float64)
	if equationMapSize(leq.solved) == 0 {
		return setOfSolved
	}
	_ = forEachEquationAscending(leq.solved, func(i int, p Polynomial) error {
		setOfSolved[i] = p.GetCoeffForTerm(0) // for every x.i = p[x.i = c]
		return nil
	})
	return setOfSolved
}

// AddEq adds a
// new equation 0 = p (p is Polynomial) to a system of linear equations.
// Immediately starts to solve the -- possibly incomplete -- system, as
// far as possible.
func (leq *LinEqSolver) AddEq(p Polynomial) (*LinEqSolver, error) {
	var err error
	leq, err = leq.addEq(p, false)
	if leq.showdependencies {
		leq.Dump(leq.varresolver)
	}
	return leq, err
}

// AddEqs adds a set of linear equations to the LEQ system.
// See AddEq.
func (leq *LinEqSolver) AddEqs(plist []Polynomial) (*LinEqSolver, error) {
	l := len(plist)
	if l == 0 {
		T().Errorf("given empty list of equations")
		return leq, ErrEmptyEquationList
	} else {
		for i, p := range plist {
			T().Debugf("adding equation %d/%d: 0 = %s", i+1, l, p)
			var err error
			leq, err = leq.addEq(p, i+1 < l)
			if err != nil {
				return leq, err
			}
		}
	}
	if leq.showdependencies {
		leq.Dump(leq.varresolver)
	}
	return leq, nil
}

// If parameter cont is true, expect another equation immediately after this
// one. This is necessary to suppress harvesting of capsules.
func (leq *LinEqSolver) addEq(p Polynomial, cont bool) (*LinEqSolver, error) {
	p = p.Zap()
	T().P("op", "new equation").Infof("0 = %s", leq.PolynString(p))
	// substitute solved in new equation
	p = leq.substituteSolved(0, p, leq.solved)
	if coeff, off := p.isOff(); !off { //  :-))  no pun intended
		// select x.i=p(i)
		i, _ := p.maxCoeff(leq.dependents) // start with max (free) coefficient of p
		var err error
		p, err = leq.activateEquationTowards(i, p) // now  x.i = -1/a * p(...).
		if err != nil {
			return leq, err
		}
		// Phase 1: substitute P(i) in every x.j=P(j)
		D, err := leq.updateDependentVariables(i, p)
		if err != nil {
			return leq, err
		}
		// done, now split solved x from D' off to S'
		S := make(SolvedMap)                                                    // set up S' of solved
		if err := forEachEquationAscending(D, func(i int, p Polynomial) error { // for every x.i=p(i) in D'
			if ok, rhs := solved(p); ok {
				S[i] = rhs   // add x.i to S'
				delete(D, i) // remove x.i from D'
			}
			return nil
		}); err != nil {
			return leq, err
		}
		// substitute solved: subst s in S' into d in D'
		//T.Info("---------- subst solved -----------")
		if err := forEachEquationAscending(D, func(i int, p Polynomial) error { // for every x.i=p(i) in D'
			p = leq.substituteSolved(i, p, S)
			if ok, rhs := solved(p); ok {
				S[i] = rhs   // add x.i to S'
				delete(D, i) // remove x.i from D'
			}
			return nil
		}); err != nil {
			return leq, err
		}
		//T.Info("-----------------------------------")
		// done, update sets S and D
		_ = forEachEquationAscending(S, func(i int, p Polynomial) error { // S = S + S'
			leq.setSolved(i, p)
			return nil
		})
		leq.dependents = D // D = D'
	} else if !arithm.Is0(coeff) {
		return leq, fmt.Errorf("%w: 0 = %s (off by %g)", ErrInconsistentEquation, leq.PolynString(p), coeff)
	}
	if !cont { // if this equation is not part of an equation-pair
		leq.harvestCapsules()
	}
	return leq, nil
}

// 1st pass of the LEQ algorithm: with a new equation x.i=P(i) walk
// through all dependent variables x.j=P(j) and substitute P(i) for x.i
// in every RHS.
// Return a new set D' of dependent variables.
func (leq *LinEqSolver) updateDependentVariables(i int, p Polynomial) (EquationMap, error) {
	D := make(EquationMap) // set up D' of dependents
	leq.updateDependency(i, p, D)
	// D -> D'
	savei := i
	T().Debugf("---------- subst dep --------------")
	err := forEachEquationAscending(leq.dependents, func(j int, q Polynomial) error { // iterate over all dependent variables
		i = savei // restore i
		tmp, ok := D[i]
		if !ok {
			return fmt.Errorf("internal solver state missing dependency for %s", leq.VarString(i))
		}
		p = tmp.CopyPolynomial() // get current version of p(i)
		T().P("op", "substitute").Debugf("(1) p(%s) in %s = %s",
			leq.VarString(i), leq.VarString(j), leq.PolynString(q))
		if j == i { // x.j = x.i, i.e. equations with identical LHS
			k, _ := q.maxCoeff(D)             // start with max (free) coefficient of q(j=i)
			lhs := NewConstantPolynomial(0.0) // construct LHS as pp
			lhs.SetTerm(j, -1.0)              // now LHS is { 0 - 1 x.j }
			q = q.Add(lhs, false)             // move to RHS
			var err error
			q, err = leq.activateEquationTowards(k, q) // now  x.k = -1/a.k * p(... x.j ...).
			if err != nil {
				return err
			}
			j = k // ride the new horse
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
			var err error
			j, q, err = subst(i, p, j, q) // substitute new equation in x.j=q(j)
			if err != nil {
				return err
			}
			T().P("op", "substitute").Debugf("result: %s = %s", leq.VarString(j), leq.PolynString(q))
			if j != 0 {
				leq.updateDependency(j, q, D) // insert substitution result
			} else { // j has been eliminated from q
				if coeff, off := q.isOff(); !off {
					k, _ := q.maxCoeff(D) // find max (free) coefficient of q(k)
					q, err = leq.activateEquationTowards(k, q)
					if err != nil {
						return err
					}
					leq.updateDependency(k, q, D) // insert new equation
				} else if !arithm.Is0(coeff) {
					return fmt.Errorf("%w: 0 = %s (off by %g)", ErrInconsistentEquation, leq.PolynString(q), coeff)
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	T().Debugf("-----------------------------------")
	return D, nil
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
func (leq *LinEqSolver) updateDependency(i int, p Polynomial, m map[int]Polynomial) {
	p = p.CopyPolynomial()
	//fmt.Printf("inserting x.%d = %v\n", i, p)
	if q, found := m[i]; found {
		//fmt.Printf("found     x.%d = %v\n", i, q)
		if termlength(p) < termlength(q) { // prefer shorter RHS terms
			varname := leq.VarString(i)
			T().P("var", varname).Infof("## %s = %s", varname, leq.PolynString(p))
			m[i] = p // replace equation x.i=p(i)
		}
	} else {
		m[i] = p // insert new equation x.i=p(i)
	}
	/*
		pp, ok := m[i]
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
func subst(i int, p Polynomial, j int, q Polynomial) (int, Polynomial, error) {
	ai := q.GetCoeffForTerm(i) // a.i in q
	if !arithm.Is0(ai) {       // if variable x.i exists in q
		delete(q.terms, i) // remove a.i*x.i in q (to be replaced)
		var err error
		p, err = p.Multiply(NewConstantPolynomial(ai), true) // scale p(i) by a.i of q
		if err != nil {
			return 0, Polynomial{}, err
		}
		q = q.Add(p, false).Zap()  // now insert p(i) into q(j)
		aj := q.GetCoeffForTerm(j) // results in a.j*x.j in q(j) ?
		if arithm.Is0(aj) {        // no => we're done
			// do nothing
		} else if arithm.Is1(aj) { // x.j = c + x.j + ...  => eliminate x.j and activate for free x.k
			delete(q.terms, j) // remove x.j from RHS q
			j = 0              // set LHS to 'impossible' variable x.0
		} else { // x.j = c + a.j*x.j + ...  => scale RHS by -1(a.j-1)
			a := -1.0 / (aj - 1.0)        // a = -1/(a.j-1)
			c := NewConstantPolynomial(a) //
			delete(q.terms, j)            // now remove a.j*x.j from RHS q
			q, err = q.Multiply(c, false) // and multiply RHS by -1/(a.j-1)
			if err != nil {
				return 0, Polynomial{}, err
			}
			q = q.Zap()
		}
	}
	return j, q, nil // return x.j = q'(j)

}

// Helper: number of variables in RHS of an equation.
func termlength(p Polynomial) int {
	return p.TermCount()
}

// In an equation, substitute all variables which are already known.
func (leq *LinEqSolver) substituteSolved(j int, p Polynomial, solved map[int]Polynomial) Polynomial {
	T().Debugf("---------- subst solved -----------")
	_ = forEachEquationAscending(solved, func(i int, rhs Polynomial) error { // iterate over all solved x.i = c
		c := rhs.GetConstantValue()
		coeff := p.GetCoeffForTerm(i)
		if !arithm.Is0(coeff) {
			coeff = coeff * c
			pc := p.GetConstantValue()
			p.SetTerm(0, pc+coeff)
			delete(p.terms, i)
			T().P("op", "subst-solved").Debugf("%s = %g  =>  RHS = %s",
				leq.VarString(i), c, leq.PolynString(p))
			if j > 0 {
				varname := leq.VarString(j)
				T().P("var", varname).Infof("## %s = %s", varname, leq.PolynString(p))
			} else {
				T().P("op", "subst known").Infof("# 0 = %s", leq.PolynString(p))
			}
		}
		return nil
	})
	T().Debugf("-----------------------------------")
	return p
}

// Transform an equation 0 = p(a x.i) to make x.i the dependent variable, i.e.
// x.i = -1/a * p(...).
func (leq *LinEqSolver) activateEquationTowards(i int, p Polynomial) (Polynomial, error) {
	coeff := p.GetCoeffForTerm(i)
	if arithm.Is0(coeff) {
		return Polynomial{}, fmt.Errorf("cannot activate equation towards %s: zero coefficient", leq.VarString(i))
	}
	delete(p.terms, i) // remove term x.i from RHS(p)
	pp := NewConstantPolynomial(-1.0 / coeff)
	var err error
	p, err = p.Multiply(pp, true)
	if err != nil {
		return Polynomial{}, err
	}
	p = p.Zap()
	//T.P("op", "activate").Infof("## %s = %s", leq.VarString(i), leq.PolynString(p))
	varname := leq.VarString(i)
	T().P("var", varname).Infof("## %s = %s", varname, leq.PolynString(p))
	return p, nil
}

// Mark a variable as solved. Sends a message to the variable resolver.
func (leq *LinEqSolver) setSolved(i int, p Polynomial) {
	c := p.GetConstantValue()
	varname := leq.VarString(i)
	T().P("var", varname).Infof("#### %s = %g", varname, c)
	leq.solved[i] = p // move x.i to set of solved variables
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
	_ = forEachEquationAscending(leq.dependents, func(w int, pw Polynomial) error { // iterate over all dependent x.w = p.w
		leq.checkAndCountCapsule(w, counts) // check LHS variable
		for _, i := range pw.Exponents() {  // get every term.i
			if i > 0 { // omit constant term
				leq.checkAndCountCapsule(i, counts)
			}
		}
		return nil
	})
	_ = forEachEquationAscending(leq.solved, func(j int, _ Polynomial) error { // count solved capsules
		leq.checkAndCountCapsule(j, counts)
		return nil
	})
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
	if _, ok := leq.solved[i]; ok {
		T().Debugf("unsolve %s", leq.VarString(i))
		delete(leq.solved, i)
	}
	delete(leq.dependents, i)                                                      // possibly remove from dependents
	eqs := make(EquationMap)                                                       // set of equation indices, i.e. int
	_ = forEachEquationAscending(leq.dependents, func(j int, p Polynomial) error { // iterate over dependents
		if a := p.GetCoeffForTerm(i); !arithm.Is0(a) { // yes, x.i in p
			eqs[j] = p // mark for deletion, as it is invalid now
		}
		return nil
	})
	_ = forEachEquationAscending(eqs, func(j int, _ Polynomial) error { // iterate over marked equations
		delete(leq.dependents, j)
		return nil
	})
}

// === Utilities =============================================================

// Dump is a debugging helper to dump all known equations.
func (leq *LinEqSolver) Dump(resolv VariableResolver) {
	fmt.Println("----------------------------------------------------------------------")
	fmt.Println("Dependents:                                                        LEQ")
	_ = forEachEquationAscending(leq.dependents, func(k int, p Polynomial) error { // for every x.i = p[x.i]
		fmt.Printf("\t%s = %s\n", TraceStringVar(k, resolv), p.TraceString(resolv))
		return nil
	})
	fmt.Println("Solved:")
	_ = forEachEquationAscending(leq.solved, func(k int, p Polynomial) error { // for every x.i = { c }
		fmt.Printf("\t%s = %g\n", TraceStringVar(k, resolv), p.GetConstantValue())
		return nil
	})
	fmt.Println("----------------------------------------------------------------------")
}
