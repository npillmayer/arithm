package polyn

import (
	"bytes"
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/npillmayer/schuko/tracing/gotestingadapter"
	"github.com/stretchr/testify/assert"
)

type res map[int]float64 // a variable resolver for testing purposes

func newResolver() res {
	var r res
	r = make(map[int]float64)
	return r
}

func (r res) GetVariableName(n int) string { // get real-life name of x.i
	return string(rune(n + 96)) // 'a', 'b', ...
}

func (r res) SetVariableSolved(n int, v float64) { // message: x.i is solved
	//T.P("msg", "SOLVED").Infof("%s = %s", r.GetVariableName(n), v.String())
	r[n] = v // remember the value to assert test conditions
}

func (r res) IsCapsule(int) bool { // x.i has gone out of scope
	return false // no capsules
}

type capsuleRes struct {
	solved   res
	capsules map[int]bool
}

func newCapsuleResolver(capsules ...int) *capsuleRes {
	cr := &capsuleRes{
		solved:   newResolver(),
		capsules: make(map[int]bool),
	}
	for _, c := range capsules {
		cr.capsules[c] = true
	}
	return cr
}

func (cr *capsuleRes) GetVariableName(n int) string {
	return cr.solved.GetVariableName(n)
}

func (cr *capsuleRes) SetVariableSolved(n int, v float64) {
	cr.solved.SetVariableSolved(n, v)
}

func (cr *capsuleRes) IsCapsule(n int) bool {
	return cr.capsules[n]
}

func snapshotPolynomial(p Polynomial) map[int]float64 {
	snap := make(map[int]float64)
	for _, i := range p.Exponents() {
		snap[i] = p.GetCoeffForTerm(i)
	}
	return snap
}

func mustAddEq(t *testing.T, leq *LinEqSolver, p Polynomial) {
	t.Helper()
	_, err := leq.AddEq(p)
	assert.NoError(t, err)
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	assert.NoError(t, err)
	os.Stdout = w
	fn()
	_ = w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	_, err = io.Copy(&buf, r)
	assert.NoError(t, err)
	return buf.String()
}

func assertBefore(t *testing.T, s, first, second string) {
	t.Helper()
	iFirst := strings.Index(s, first)
	iSecond := strings.Index(s, second)
	assert.NotEqual(t, -1, iFirst, "missing substring %q", first)
	assert.NotEqual(t, -1, iSecond, "missing substring %q", second)
	assert.Less(t, iFirst, iSecond, "expected %q before %q", first, second)
}

// --- Tests -----------------------------------------------------------------

func TestPolynSimple1(t *testing.T) {
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	p := NewConstantPolynomial(1.0)
	if p.TermCount() != 1 {
		t.Fail()
	}
}

func TestPolynSimple2(t *testing.T) {
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	p := NewConstantPolynomial(0.5)
	p.SetTerm(1, 3)
	if p.TermCount() != 2 {
		t.Fail()
	}
}

func TestPolynConstant(t *testing.T) {
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	p := NewConstantPolynomial(0.5)
	_, isconst := p.IsConstant()
	if !isconst {
		t.Error("did not recognize constant polynomial as constant")
	}
	p.SetTerm(1, 2)
	_, isconst = p.IsConstant()
	if isconst {
		t.Error("did falsely recognize non-constant polynomial as constant")
	}
}

func TestZapPolyn(t *testing.T) {
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	p := NewConstantPolynomial(0.5)
	p.SetTerm(1, 0.0000000005)
	p.Zap()
	_, isconst := p.IsConstant()
	if !isconst {
		t.Error("Expected polynomial to be of constant type, isn't")
	}
}

func TestPolynAdd(t *testing.T) {
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	p, _ := New(5, X{1, 1}, X{2, 2})
	t.Logf("# p  = %s\n", p.String())
	p2, _ := New(4, X{1, 6}, X{5, 4})
	t.Logf("# p2 = %s\n", p2.String())
	pr := p.Add(p2, false)
	t.Logf("# pr = %s\n", pr.String())
	pr.Zap()
	if pr.GetCoeffForTerm(1) != 7.0 {
		t.Fail()
	}
}

func TestPolynSubtract(t *testing.T) {
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	p, _ := New(10, X{1, 7}, X{2, 2})
	q, _ := New(4, X{1, 2}, X{3, 9})
	r := p.Subtract(q, false).Zap()
	assert.InDelta(t, 6.0, r.GetCoeffForTerm(0), 1e-9)
	assert.InDelta(t, 5.0, r.GetCoeffForTerm(1), 1e-9)
	assert.InDelta(t, 2.0, r.GetCoeffForTerm(2), 1e-9)
	assert.InDelta(t, -9.0, r.GetCoeffForTerm(3), 1e-9)
}

func TestPolynMul(t *testing.T) {
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	p, _ := New(6, X{1, 4}, X{2, 2})
	t.Logf("T p  = %s\n", p.String())
	p2 := NewConstantPolynomial(-2.0)
	t.Logf("T p2 = %s\n", p2.String())
	pr, err := p.Multiply(p2, true)
	assert.NoError(t, err)
	t.Logf("T pr = %s\n", pr.String())
	pr.Zap()
	if pr.GetCoeffForTerm(1) != -8.0 {
		t.Fail()
	}
}

func TestPolynDiv(t *testing.T) {
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	p, _ := New(6, X{1, 4}, X{2, 2})
	t.Logf("T p  = %s\n", p.String())
	p2 := NewConstantPolynomial(2.0)
	t.Logf("T p2 = %s\n", p2.String())
	pr, err := p.Divide(p2, false)
	assert.NoError(t, err)
	t.Logf("T pr = %s\n", pr.String())
	pr.Zap()
	p2 = NewConstantPolynomial(0.0)
	t.Logf("T p2 = %s\n", p2.String())
	_, err = p.Divide(p2, false)
	assert.Error(t, err, "expected error for division by zero")
}

func TestPolynAddDoesNotMutateOperands(t *testing.T) {
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	p, _ := New(5, X{1, 1}, X{2, 2})
	p2, _ := New(4, X{1, 6}, X{5, 4})
	pBefore := snapshotPolynomial(p)
	p2Before := snapshotPolynomial(p2)
	_ = p.Add(p2, false)
	assert.Equal(t, pBefore, snapshotPolynomial(p), "Add mutated left operand")
	assert.Equal(t, p2Before, snapshotPolynomial(p2), "Add mutated right operand")
}

func TestPolynMultiplyDoesNotMutateOperands(t *testing.T) {
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	p := NewConstantPolynomial(2.0)
	p2, _ := New(3, X{1, 4}, X{2, -1})
	pBefore := snapshotPolynomial(p)
	p2Before := snapshotPolynomial(p2)
	_, err := p.Multiply(p2, false)
	assert.NoError(t, err)
	assert.Equal(t, pBefore, snapshotPolynomial(p), "Multiply mutated left operand")
	assert.Equal(t, p2Before, snapshotPolynomial(p2), "Multiply mutated right operand")
}

func TestPolynDivideDoesNotMutateDivisor(t *testing.T) {
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	p, _ := New(6, X{1, 4}, X{2, 2})
	div := NewConstantPolynomial(2.0)
	divBefore := snapshotPolynomial(div)
	_, err := p.Divide(div, false)
	assert.NoError(t, err)
	assert.Equal(t, divBefore, snapshotPolynomial(div), "Divide mutated divisor")
}

func TestPolynSubst(t *testing.T) {
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	p, _ := New(1, X{1, 10}, X{2, 20})
	t.Logf("T p  = %s\n", p.String())
	p2, _ := New(2, X{3, 30}, X{4, 40})
	t.Logf("T p2 = %s\n", p2.String())
	p = p.substitute(1, p2)
	t.Logf("T -> p = %s\n", p.String())
	if p.GetCoeffForTerm(3) != 300.0 {
		t.Fail()
	}
}

func TestPolynMaxCoeff(t *testing.T) {
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	p, _ := New(1, X{1, 8}, X{2, 2}, X{3, -2})
	t.Logf("T p  = %s\n", p.String())
	i, c := p.maxCoeff(nil)
	if i != 1 || c != 8.0 {
		t.Fail()
	}
	t.Logf("T ->max coeff @%d is %g, ok\n", i, c)
}

func TestPolynMaxCoeffTieBehavior(t *testing.T) {
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	p, _ := New(0, X{1, 5}, X{2, -5}, X{4, 5})
	i, c := p.maxCoeff(nil)
	assert.Equal(t, 1, i, "tie should resolve to lowest exponent in ascending scan")
	assert.InDelta(t, 5.0, c, 1e-9)
	dependents := map[int]Polynomial{1: NewConstantPolynomial(0)}
	i, c = p.maxCoeff(dependents)
	assert.Equal(t, 2, i, "dependent variable should be skipped")
	assert.InDelta(t, -5.0, c, 1e-9)
}

func TestPolynVariableAndValidity(t *testing.T) {
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	var zero Polynomial
	assert.False(t, zero.IsValid())
	p, _ := New(0, X{3, 1})
	pos, ok := p.IsVariable()
	assert.True(t, ok)
	assert.Equal(t, 3, pos)
	assert.True(t, p.IsValid())
	q, _ := New(1, X{3, 1})
	_, ok = q.IsVariable()
	assert.False(t, ok)
}

func TestArityComparator(t *testing.T) {
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	p, _ := New(0, X{1, 1})
	q, _ := New(0, X{1, 1}, X{2, 1})
	assert.Less(t, ArityComparator(p, q), 0)
	assert.Greater(t, ArityComparator(q, p), 0)
}

func TestTraceStringVar(t *testing.T) {
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	assert.Equal(t, "x.3", TraceStringVar(3, nil))
	r := newResolver()
	assert.Equal(t, "c", TraceStringVar(3, r))
}

func TestTraceStringDeterministicOrdering(t *testing.T) {
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	r := newResolver()
	p, _ := New(0, X{8, 1}, X{2, 1}, X{5, 1})
	s := p.TraceString(r)
	assertBefore(t, s, "b", "e")
	assertBefore(t, s, "e", "h")
}

func TestSubst1(t *testing.T) {
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	p, _ := New(1, X{3, 3})
	q, _ := New(2, X{1, 3}, X{4, 4}, X{5, 5})
	t.Logf("   x.%d = %s\n", 1, p.String())
	t.Logf("   x.%d = %s\n", 2, q.String())
	var k int
	var err error
	k, q, err = subst(1, p, 2, q)
	assert.NoError(t, err)
	t.Logf("=> x.%d = %s\n", k, q.String())
	if q.GetCoeffForTerm(3) != 9.0 {
		t.Fail()
	}
}

func TestSubst2(t *testing.T) {
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	p, _ := New(1, X{3, 3})
	q, _ := New(2, X{1, 3}, X{3, 4}, X{5, 5})
	t.Logf("   x.%d = %s\n", 1, p.String())
	t.Logf("   x.%d = %s\n", 2, q.String())
	var k int
	var err error
	k, q, err = subst(1, p, 2, q)
	assert.NoError(t, err)
	t.Logf("=> x.%d = %s\n", k, q.String())
	if q.GetCoeffForTerm(3) != 13.0 {
		t.Fail()
	}
}

func TestSubst3(t *testing.T) {
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	p, _ := New(1, X{2, 1})
	q, _ := New(2, X{1, 1}, X{4, 4}, X{5, 5})
	t.Logf("   x.%d = %s\n", 1, p.String())
	t.Logf("   x.%d = %s\n", 2, q.String())
	var k int
	var err error
	k, q, err = subst(1, p, 2, q)
	assert.NoError(t, err)
	t.Logf("=> x.%d = %s\n", k, q.String())
	if k != 0 {
		t.Fail()
	}
}

func TestNewSolver(t *testing.T) {
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	leq := NewLinEqSolver()
	if leq == nil {
		t.Error("cannot create solver")
		t.Fail()
	}
}

func TestLinEqAddPolyn(t *testing.T) {
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	leq := NewLinEqSolver()
	r := newResolver()
	leq.SetVariableResolver(r)
	p, _ := New(1, X{1, 2})
	mustAddEq(t, leq, p)
	if _, found := r[1]; !found {
		t.Error("a still unsolved")
		t.Fail()
	}
}

func TestLinEqAddPolyn2(t *testing.T) {
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	leq := NewLinEqSolver()
	r := newResolver()
	leq.SetVariableResolver(r)
	p, _ := New(6, X{1, -1}, X{2, -1})
	mustAddEq(t, leq, p)
	q, _ := New(2, X{1, 3}, X{2, -1})
	mustAddEq(t, leq, q)
	if _, found := r[1]; !found {
		t.Error("a still unsolved")
		t.Fail()
	}
}

func TestLEQCharacterizationSimpleSystem(t *testing.T) {
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	leq := NewLinEqSolver()
	r := newResolver()
	leq.SetVariableResolver(r)
	p, _ := New(6, X{1, -1}, X{2, -1}) // a+b=6
	q, _ := New(2, X{1, 3}, X{2, -1})  // b=2+3a
	mustAddEq(t, leq, p)
	mustAddEq(t, leq, q)
	assert.InDelta(t, 1.0, r[1], 1e-9, "unexpected solution for a")
	assert.InDelta(t, 5.0, r[2], 1e-9, "unexpected solution for b")
}

func TestAddEqsAndGetSolvedVars(t *testing.T) {
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	leq := NewLinEqSolver()
	r := newResolver()
	leq.SetVariableResolver(r)
	p, _ := New(6, X{1, -1}, X{2, -1}) // a+b=6
	q, _ := New(2, X{1, 3}, X{2, -1})  // b=2+3a
	_, err := leq.AddEqs([]Polynomial{p, q})
	assert.NoError(t, err)
	solved := leq.getSolvedVars()
	v, ok := solved[1]
	assert.True(t, ok)
	assert.InDelta(t, 1.0, v, 1e-9)
}

func TestAddEqsEmptyList(t *testing.T) {
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	leq := NewLinEqSolver()
	_, err := leq.AddEqs(nil)
	assert.True(t, errors.Is(err, ErrEmptyEquationList))
}

func TestRetractVariable(t *testing.T) {
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	leq := NewLinEqSolver()
	leq.solved[1] = NewConstantPolynomial(7)
	pKeep, _ := New(0, X{3, 1})
	pDrop, _ := New(0, X{1, 1}, X{2, 1})
	leq.dependents[3] = pKeep
	leq.dependents[2] = pDrop
	leq.retractVariable(1)
	_, ok := leq.solved[1]
	assert.False(t, ok)
	_, ok = leq.dependents[2]
	assert.False(t, ok)
	_, ok = leq.dependents[3]
	assert.True(t, ok)
}

func TestHarvestCapsulesAndDump(t *testing.T) {
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	leq := NewLinEqSolver()
	cr := newCapsuleResolver(5, 6)
	leq.SetVariableResolver(cr)
	pA, _ := New(0, X{5, 1}) // capsule 5 appears twice, should stay
	pB, _ := New(1, X{5, 1})
	pC, _ := New(0, X{6, 1}) // capsule 6 appears once, should be removed
	leq.dependents[2] = pA
	leq.dependents[3] = pB
	leq.dependents[4] = pC
	leq.harvestCapsules()
	_, ok := leq.dependents[2]
	assert.True(t, ok)
	_, ok = leq.dependents[3]
	assert.True(t, ok)
	_, ok = leq.dependents[4]
	assert.False(t, ok)
	out := captureStdout(t, func() { leq.Dump(cr) })
	assert.Contains(t, out, "Dependents:")
	assert.Contains(t, out, "Solved:")
}

func TestDumpDeterministicOrdering(t *testing.T) {
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	leq := NewLinEqSolver()
	p2, _ := New(1, X{2, 1})
	p5, _ := New(1, X{5, 1})
	p9, _ := New(1, X{9, 1})
	leq.dependents[9] = p9
	leq.dependents[2] = p2
	leq.dependents[5] = p5
	leq.solved[8] = NewConstantPolynomial(8)
	leq.solved[1] = NewConstantPolynomial(1)
	out := captureStdout(t, func() { leq.Dump(nil) })
	assertBefore(t, out, "\tx.2 =", "\tx.5 =")
	assertBefore(t, out, "\tx.5 =", "\tx.9 =")
	assertBefore(t, out, "\tx.1 =", "\tx.8 =")
}

func TestHarvestCapsuleInSolvedGetsRetracted(t *testing.T) {
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	leq := NewLinEqSolver()
	cr := newCapsuleResolver(8)
	leq.SetVariableResolver(cr)
	leq.solved[8] = NewConstantPolynomial(42)
	leq.harvestCapsules()
	_, ok := leq.solved[8]
	assert.False(t, ok, "capsule in solved map should be retracted")
}

func TestHarvestDoesNotRemoveNonCapsules(t *testing.T) {
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	leq := NewLinEqSolver()
	cr := newCapsuleResolver(9)
	leq.SetVariableResolver(cr)
	pNonCapsule, _ := New(0, X{7, 1}) // variable 7 is not a capsule
	leq.dependents[7] = pNonCapsule
	leq.harvestCapsules()
	_, ok := leq.dependents[7]
	assert.True(t, ok, "non-capsule variable should not be harvested")
}

func TestLEQ1(t *testing.T) {
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	leq := NewLinEqSolver()
	r := newResolver()
	leq.SetVariableResolver(r)
	p1, _ := New(100, X{1, -2})           // 2a=100   =>  0=100-2a
	p2, _ := New(100, X{2, -1}, X{3, -1}) // 100=b+c  =>  0=100-b-c
	mustAddEq(t, leq, p1)
	mustAddEq(t, leq, p2)
	if _, found := r[1]; !found {
		t.Error("a still unsolved")
		t.Fail()
	}
}

func TestLEQ2(t *testing.T) {
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	leq := NewLinEqSolver()
	r := newResolver()
	leq.SetVariableResolver(r)
	p1, _ := New(100, X{2, -1}, X{3, -1})        // b+c=100 =>  0=100-b-c
	p2, _ := New(0, X{1, 2}, X{2, -1}, X{3, -1}) // 2a=b+c  =>  0=2a-b-c
	mustAddEq(t, leq, p1)
	mustAddEq(t, leq, p2)
	if _, found := r[1]; !found {
		t.Error("a still unsolved")
		t.Fail()
	}
}

func TestLEQ3(t *testing.T) {
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	leq := NewLinEqSolver()
	r := newResolver()
	leq.SetVariableResolver(r)
	p1, _ := New(100, X{1, -1}) // a = 100
	p2, _ := New(99, X{1, -2})  // 2a = 99
	mustAddEq(t, leq, p1)
	_, err := leq.AddEq(p2)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrInconsistentEquation))
}

func TestLEQ4(t *testing.T) {
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	leq := NewLinEqSolver()
	r := newResolver()
	leq.SetVariableResolver(r)
	p1, _ := New(100, X{1, -1})                          // a=100
	p2, _ := New(0, X{1, 2}, X{2, -1}, X{3, 1}, X{4, 4}) // 2a=b-c-e
	p3, _ := New(0, X{2, 1}, X{3, -1})                   // b=c
	mustAddEq(t, leq, p1)
	mustAddEq(t, leq, p2)
	mustAddEq(t, leq, p3) // eliminates b and c from p2 => d solved
	if _, found := r[4]; !found {
		for i, v := range r {
			t.Logf("r[%d] = %v", i, v)
		}
		leq.Dump(r)
		t.Error("d still unsolved")
		t.Fail()
	}
}

func TestLEQ5(t *testing.T) {
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	leq := NewLinEqSolver()
	//leq.showdependencies = true
	r := newResolver()
	leq.SetVariableResolver(r)
	p1, _ := New(0, X{2, -1}, X{3, 1}) // b=c
	p2, _ := New(0, X{3, -1}, X{4, 1}) // c=d
	p3, _ := New(0, X{4, -1}, X{2, 1}) // d=b
	mustAddEq(t, leq, p1)
	mustAddEq(t, leq, p2)
	mustAddEq(t, leq, p3)
	p4, _ := New(0, X{1, -1}, X{2, 1}, X{3, 1}, X{4, 1}) // a=b+c+d
	mustAddEq(t, leq, p4)                                // now a=3d (or b or c)
	p := leq.dependents[1]
	if termlength(p) != 2 { // a = 0 + 3d
		t.Fail()
	}
}

// Example for solving linear equations. We use a variable resolver, which
// maps a numeric value of 0..<n> to lowercase letters 'a'..'z'.
func TestExampleLinEqSolver_usage(t *testing.T) {
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	//func TestExampleLinEqSolver_usage() {
	leq := NewLinEqSolver()
	r := newResolver() // clients have to provide their own
	leq.SetVariableResolver(r)
	p, _ := New(6, X{1, -1}, X{2, -1})
	mustAddEq(t, leq, p)
	q, _ := New(2, X{1, 3}, X{2, -1})
	mustAddEq(t, leq, q)
}
