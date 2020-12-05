package jhobby

import (
	"fmt"
	"math"
	"math/cmplx"

	"github.com/npillmayer/arithm"
	"github.com/npillmayer/schuko/gconf"
	"github.com/npillmayer/schuko/gtrace"
	"github.com/npillmayer/schuko/tracing"
)

// T is tracing to the graphics tracer.
func T() tracing.Trace {
	return gtrace.GraphicsTracer
}

const pi float64 = 3.14159265
const pi2 float64 = 6.28318530
const _epsilon = 0.0000001

// --- Interfaces ------------------------------------------------------------

// HobbyPath is the path type we're dealing with. We base the implementation
// on an interface, with numeric values and point values represented
// by Go's native float types.
//
// The interface is a "read only" interface in the sense that it provides
// the input path's parameters for the MetaFont spline interpolation.
// A path may have parameters provided for knots or for joins/curves. Possible
// knot parameters are: dir (= an explicit angle for the tangent at the knot) or
// curl (= the amount of curvature at the knot). Curves may be given tension
// parameters, which control the "tightness" of the line between knots. A
// negative value for a tension means "at least" the amount of tension and is
// used to prevent the spline from leaving the bounding box of its control
// point.
//
// Paths may be cyclic, i.e. closed. Knots, addressed by path.Z(i), must
// adhere to the following requirement: Z() must accept subscipts >= N, i.e.
// larger than the length of the path, and return knot[i mod N]. The last
// knot of a cyclic path is identical to the first one, but it must not be
// included twice! Instead, the algorithm relies on the modulo-subscripting
// mechanism for adressing all knots of the cycle.
//
// A HobbyPath does not contain information about spline control points (the
// path's properties are understood as "input parameters" for the Hobby
// algorithm). Control point information is handled using a separate interface.
//
// see type SplineControls.
type HobbyPath interface {
	IsCycle() bool           // is this a cyclic path?
	N() int                  // number of knots in the path
	Z(int) arithm.Pair       // knot #i modulo N
	PreDir(int) arithm.Pair  // explicit dir before knot #i
	PostDir(int) arithm.Pair // explicit dir after knot #i
	PreCurl(int) float64     // explicit curl before knot #i
	PostCurl(int) float64    // explicit curl after knot #i
	PreTension(int) float64  // explict tension before knot #i
	PostTension(int) float64 // explicit tension after knot #i
}

// SplineControls is used for gathering the spline control points
// // calculated by Hobby's algorithm.
//
// A HobbyPath starts out being void of spline control points (a skeleton
// path, which may be interpreted as a polygon). The path may include
// tension or tangent angle information (dir and/or curl). Clients then use
// FindHobbyControls(...) to fill in appropriate control point information
// for a curved path through the path's knots.
//
// see FindHobbyControls(...)
type SplineControls interface {
	PreControl(i int) arithm.Pair
	PostControl(i int) arithm.Pair
	SetPreControl(int, arithm.Pair)  // set control point (after calculation)
	SetPostControl(int, arithm.Pair) // set control point (after calculation)
}

// AsString returns
// a path -- optionally including spline control points -- as a (debugging)
// string. The string contains newlines if control point information is present.
// Otherwise it will include the knot coordinates in one line.
//
// Example, a circle of diameter 1 around (2,1):
//
//     (1,1) .. controls (1.0000,1.5523) and (1.4477,2.0000)
//       .. (2,2) .. controls (2.5523,2.0000) and (3.0000,1.5523)
//       .. (3,1) .. controls (3.0000,0.4477) and (2.5523,0.0000)
//       .. (2,0) .. controls (1.4477,0.0000) and (1.0000,0.4477)
//       .. cycle
//
// The format is not fully equivalent to MetaFont's, but close.
func AsString(path HobbyPath, contr SplineControls) string {
	var s string
	for i := 0; i < path.N(); i++ {
		pt := path.Z(i)
		if i > 0 {
			if contr != nil {
				s += fmt.Sprintf(" and %s\n  .. ", ptstring(contr.PreControl(i), true))
			} else {
				s += " .. "
			}
		}
		s += fmt.Sprintf("%s", ptstring(pt, false))
		if contr != nil && (i < path.N()-1 || path.IsCycle()) {
			s += fmt.Sprintf(" .. controls %s", ptstring(contr.PostControl(i), true))
		}
	}
	if path.IsCycle() {
		if contr != nil {
			s += fmt.Sprintf(" and %s\n ", ptstring(contr.PreControl(0), true))
		}
		s += " .. cycle"
	}
	return s
}

// --- Implementation --------------------------------------------------------

// Path is a concrete implementation of interface HobbyPath.
// To construct a path, start with Nullpath(), which creates an empty
// path, and then extend it.
type Path struct {
	points   []arithm.Pair // point i
	cycle    bool          // is this path cyclic ?
	predirs  []arithm.Pair // explicit pre-direction at point i
	postdirs []arithm.Pair // explicit post-direction at point i
	curls    []arithm.Pair // explicit l and r curl at point i
	tensions []arithm.Pair // explicit pre- and post-tension at point i
	Controls *splcntrls    // control points to be calculated
}

// A segment of a path; will implement interface HobbyPath
type pathPartial struct {
	whole    HobbyPath      // parent path
	start    int            // first index within parent path
	end      int            // last index within parent path
	controls SplineControls // control points, shared with parent path
}

// Sub-type for collecting the calculated spline control points
type splcntrls struct {
	prec  []arithm.Pair // control point i-, to be calculated
	postc []arithm.Pair // control point i+, to be calculated
}

var _ HobbyPath = &Path{}
var _ SplineControls = &splcntrls{}
var _ SplineControls = &pathPartial{}

//var _ HobbyPath = &pathPartial{}

// === API ===================================================================

func newSkeletonPath(points []arithm.Pair) *Path {
	path := &Path{}
	path.points = make([]arithm.Pair, len(points), len(points)*2)
	path.predirs = make([]arithm.Pair, len(points), len(points)*2)
	path.postdirs = make([]arithm.Pair, len(points), len(points)*2)
	path.curls = make([]arithm.Pair, len(points), len(points)*2)
	path.tensions = make([]arithm.Pair, len(points), len(points)*2)
	for i, pt := range points {
		path.points[i] = pt // TODO: initialize all arrays
	}
	path.Controls = &splcntrls{}
	return path
}

// --- Building Paths --------------------------------------------------------

// KnotAdder is an interface for helping control the construction of a path. It is used
// to ensure that evey path-join (curve) is followed by a knot (or by a
// cycle statement).
//
// It is probably only useful in conjunction with type Path. It is made
// public to support source code editors with code completion.
type KnotAdder interface {
	Knot(arithm.Pair) JoinAdder
	SmoothKnot(arithm.Pair) JoinAdder
	CurlKnot(pr arithm.Pair, precurl, postcurl float64) JoinAdder
	DirKnot(pr arithm.Pair, dir arithm.Pair) JoinAdder
	AppendSubpath(sp *Path) JoinAdder
	Cycle() (HobbyPath, SplineControls)
}

// JoinAdder is an interface for helping control the construction of a path. It is used
// to ensure that every knot is followed by a join/curve (or ends the path).
//
// It is probably only useful in conjunction with type Path. It is made
// public to support source code editors with code completion.
type JoinAdder interface {
	Line() KnotAdder
	Curve() KnotAdder
	TensionCurve(t1, t2 float64) KnotAdder
	End() (HobbyPath, SplineControls)
}

var _ KnotAdder = &Path{}
var _ JoinAdder = &Path{}

// Nullpath creates an empty path, to be extended by subsequent builder
// calls. the following example builds a closed path of three knots, which are
// connected by a curve, then a straight line, and a curve again.
//
//     var path HobbyPath
//     var controls SplineControls
//     path, controls = Nullpath().Knot(0,0).Curve().Knot(3,2).Line().Knot(5,2.5).Curve().Cycle()
//
// Calling Cycle() or End() returns a path and a container for spline control point
// information. The latter is empty and to be filled by calculating the Hobby
// spline control points.
func Nullpath() *Path {
	return newSkeletonPath(nil)
}

// End an open path. Part of builder functionality.
func (path *Path) End() (HobbyPath, SplineControls) {
	return path, path.Controls
}

// Cycle closes a cyclic path. Part of builder functionality.
func (path *Path) Cycle() (HobbyPath, SplineControls) {
	path.cycle = true
	return path, path.Controls
}

// Knot adds a standard smooth knot to a path. Part of builder functionality.
func (path *Path) Knot(pr arithm.Pair) JoinAdder {
	return path.SmoothKnot(pr)
}

// SmoothKnot adds a standard smooth knot to a path (same as Knot(pr)).
// Part of builder functionality.
func (path *Path) SmoothKnot(p arithm.Pair) JoinAdder {
	path.points = append(path.points, p)
	return path
}

// CurlKnot adds a path with curl information to a path. Callers may specify pre- and/or
// post-curl. A curl value of 1.0 is considered neutral.
// Part of builder functionality.
func (path *Path) CurlKnot(p arithm.Pair, precurl, postcurl float64) JoinAdder {
	path.points = append(path.points, p)
	path.SetPreCurl(path.N()-1, precurl)
	path.SetPostCurl(path.N()-1, postcurl)
	return path
}

// DirKnot adds a knot with a given tangent direction.
// Part of builder functionality.
func (path *Path) DirKnot(p arithm.Pair, dir arithm.Pair) JoinAdder {
	path.points = append(path.points, p)
	path.SetPreDir(path.N()-1, dir)
	path.SetPostDir(path.N()-1, dir)
	return path
}

// Line connects two knots with a straight line.
// Part of builder functionality.
func (path *Path) Line() KnotAdder {
	if path.N() == 0 {
		panic("cannot add line to empty path")
	}
	path.SetPostCurl(path.N()-1, 1.0)
	path.SetPreCurl(path.N(), 1.0)
	return path
}

// Curve connects two knots with a smooth curve.
// Part of builder functionality.
func (path *Path) Curve() KnotAdder {
	if path.N() == 0 {
		panic("cannot add curve to empty path")
	}
	path.TensionCurve(1.0, 1.0)
	return path
}

// TensionCurve connects two knots with a tense curve.
// Part of builder functionality.
//
// Tensions are adapted to lie between 3/4 and 4 (absolute).  Negative tensions
// are interpreted as "at least" tensions to ensure the spline stays within
// the bounding box at its control point.
//
// BUG(norbert@pillmayer.com): Tension spec "at least" currently not completely implemented.
func (path *Path) TensionCurve(t1, t2 float64) KnotAdder {
	if path.N() == 0 {
		panic("cannot add curve to empty path")
	}
	if t1 != 1.0 {
		path.SetPostTension(path.N()-1, t1)
	}
	if t2 != 1.0 {
		path.SetPreTension(path.N(), t2)
	}
	return path
}

// AppendSubpath concatenates two paths at an overlapping knot.
// Part of builder functionality.
func (path *Path) AppendSubpath(sp *Path) JoinAdder {
	T().Errorf("AppendSubpath not yet implemented")
	return path
}

// --- Setting Path Properties -----------------------------------------------

// SetPreDir is a property setter.
func (path *Path) SetPreDir(i int, dir arithm.Pair) *Path {
	path.predirs = extendC(path.predirs, i, arithm.Pair(cmplx.NaN()))
	path.predirs[i] = dir
	return path
}

// SetPostDir is a property setter.
func (path *Path) SetPostDir(i int, dir arithm.Pair) *Path {
	path.postdirs = extendC(path.postdirs, i, arithm.Pair(cmplx.NaN()))
	path.postdirs[i] = dir
	return path
}

// SetPreCurl is a property setter.
func (path *Path) SetPreCurl(i int, curl float64) *Path {
	path.curls = extendC(path.curls, i, 1+1i)
	c := path.curls[i]
	post := imag(c)
	path.curls[i] = arithm.P(curl, post)
	return path
}

// SetPostCurl is a property setter.
func (path *Path) SetPostCurl(i int, curl float64) *Path {
	path.curls = extendC(path.curls, i, 1+1i)
	//fmt.Printf("i = %d, len(path.curls) = %d\n", i, len(path.curls))
	c := path.curls[i]
	pre := real(c)
	path.curls[i] = arithm.P(pre, curl)
	return path
}

// SetPreTension is a property setter.
//
// Tensions are adapted to lie between 3/4 and 4 (absolute).  Negative tensions
// are interpreted as "at least" tensions to ensure the spline stays within
// the bounding box at its control point.
func (path *Path) SetPreTension(i int, tension float64) *Path {
	path.tensions = extendC(path.tensions, i, 1+1i)
	t := path.tensions[i]
	post := imag(t)
	pretension := tension
	if pretension < 0.75 {
		pretension = 0.75
	} else if pretension > 4.0 {
		pretension = 4.0
	}
	path.tensions[i] = arithm.P(pretension, post)
	return path
}

// SetPostTension is a property setter.
//
// Tensions are adapted to lie between 3/4 and 4 (absolute).  Negative tensions
// are interpreted as "at least" tensions to ensure the spline stays within
// the bounding box at its control point.
func (path *Path) SetPostTension(i int, tension float64) *Path {
	path.tensions = extendC(path.tensions, i, 1+1i)
	t := path.tensions[i]
	pre := real(t)
	posttension := tension
	if posttension < 0.75 {
		posttension = 0.75
	} else if posttension > 4.0 {
		posttension = 4.0
	}
	path.tensions[i] = arithm.P(pre, posttension)
	return path
}

// === Interface Implementation ==============================================

// IsCycle is a predicate: is this path cyclic?
//
// Interface HobbyPath.
func (path *Path) IsCycle() bool {
	return path.cycle
}

// N returns the length of this path (knot count). For cyclic paths, the first and last knot
// should count as one.
//
// Interface HobbyPath.
func (path *Path) N() int {
	return len(path.points)
}

// Z returns the knot at position (i mod N).
//
// Interface HobbyPath.
func (path *Path) Z(i int) arithm.Pair {
	if i < 0 || i >= path.N() {
		i = i % path.N()
	}
	z := path.points[i]
	return z
}

// PreDir gets the incoming tangent / direction vector at z.i .
//
// Interface HobbyPath.
func (path *Path) PreDir(i int) arithm.Pair {
	return getC(path.predirs, i, arithm.Pair(cmplx.NaN()))
}

// PostDir gets the outgoing tangent / direction vector at z.i .
//
// Interface HobbyPath.
func (path *Path) PostDir(i int) arithm.Pair {
	return getC(path.postdirs, i, arithm.Pair(cmplx.NaN()))
}

// PreCurl gets the curl before z.i.
//
// Interface HobbyPath.
func (path *Path) PreCurl(i int) float64 {
	c := getC(path.curls, i, 1+1i)
	return real(c)
}

// PostCurl gets the curl after z.i.
//
// Interface HobbyPath.
func (path *Path) PostCurl(i int) float64 {
	c := getC(path.curls, i, 1+1i)
	return imag(c)
}

// PreTension returns the tension before z.i.
//
// Interface HobbyPath.
func (path *Path) PreTension(i int) float64 {
	t := getC(path.tensions, i, 1+1i)
	return real(t)
}

// PostTension returns the tension after z.i.
//
// Interface HobbyPath.
func (path *Path) PostTension(i int) float64 {
	t := getC(path.tensions, i, 1+1i)
	return imag(t)
}

// --- Segments --------------------------------------------------------------

func (pp *pathPartial) IsCycle() bool {
	return pp.whole.IsCycle() && pp.whole.N() == pp.N()
}

func (pp *pathPartial) N() int {
	return pp.end - pp.start + 1
}

func (pp *pathPartial) pmap(i int) int {
	i = i%pp.N() + pp.start
	return i
}

func (pp *pathPartial) Z(i int) arithm.Pair {
	if pp.IsCycle() {
		return pp.whole.Z(i)
	}
	return pp.whole.Z(pp.pmap(i))
}

func (pp *pathPartial) PreDir(i int) arithm.Pair {
	return pp.whole.PreDir(pp.pmap(i))
}

func (pp *pathPartial) PostDir(i int) arithm.Pair {
	return pp.whole.PostDir(pp.pmap(i))
}

func (pp *pathPartial) PreCurl(i int) float64 {
	return pp.whole.PreCurl(pp.pmap(i))
}

func (pp *pathPartial) PostCurl(i int) float64 {
	return pp.whole.PostCurl(pp.pmap(i))
}

func (pp *pathPartial) PreTension(i int) float64 {
	return pp.whole.PreTension(pp.pmap(i))
}

func (pp *pathPartial) PostTension(i int) float64 {
	return pp.whole.PostTension(pp.pmap(i))
}

func (pp *pathPartial) SetPreControl(i int, c arithm.Pair) {
	pp.controls.SetPreControl(pp.pmap(i), c)
}

func (pp *pathPartial) SetPostControl(i int, c arithm.Pair) {
	pp.controls.SetPostControl(pp.pmap(i), c)
}

func (pp *pathPartial) PreControl(i int) arithm.Pair {
	return pp.controls.PreControl(pp.pmap(i))
}

func (pp *pathPartial) PostControl(i int) arithm.Pair {
	return pp.controls.PostControl(pp.pmap(i))
}

// --- Control Points --------------------------------------------------------

// BUG(norbert@pillmayer.com): Currently it isn't possible to explicitly set
// control points. This may or may not change in the future.
func (ctrls *splcntrls) SetPreControl(i int, c arithm.Pair) {
	ctrls.prec = extendC(ctrls.prec, i, arithm.Pair(cmplx.NaN()))
	ctrls.prec[i] = c
}

func (ctrls *splcntrls) SetPostControl(i int, c arithm.Pair) {
	//if ctrls.prec == nil {
	//  ctrls.postc = make([]arithm.Pair, path.N()+2) // postcontrol points to find
	//}
	ctrls.postc = extendC(ctrls.postc, i, arithm.Pair(cmplx.NaN()))
	ctrls.postc[i] = c
}

func (ctrls *splcntrls) PreControl(i int) arithm.Pair {
	return getC(ctrls.prec, i, arithm.Pair(cmplx.NaN()))
}

func (ctrls *splcntrls) PostControl(i int) arithm.Pair {
	return getC(ctrls.postc, i, arithm.Pair(cmplx.NaN()))
}

// === Calculation API =======================================================

// FindHobbyControls finds the parameters for Hobby-spline control points
// for a given skeletion path.
//
// BUG(norbert@pillmayer.com): Currently there are slight deviations from
// MetaFont's calculation, probably due to different rounding. These are under
// investigation.
func FindHobbyControls(path HobbyPath, controls SplineControls) SplineControls {
	if controls == nil {
		controls = &splcntrls{}
	}
	segments := splitSegments(path)
	if len(segments) > 0 {
		for _, segment := range segments {
			segment.controls = controls
			T().Infof("find controls for segment %s", AsString(segment, nil))
			findSegmentControls(segment, segment)
		}
	}
	return controls
}

/*
Find the Control Points according to Hobby's Algorithm. This is the
central API function of this package.

Clients may proved a container for the spline control points. If none
is provided, i.e. controls == nil, this function will allocate one.

FindHobbyControls(...) will trace the calculated final path using log-level
INFO, if tracingchoices=true (as MetaFont does).
*/
func findSegmentControls(path HobbyPath, controls SplineControls) SplineControls {
	var u = make([]float64, path.N()+2)
	var v = make([]float64, path.N()+2)
	var theta = make([]float64, path.N()+2)
	if path.IsCycle() {
		var w = make([]float64, path.N()+2)
		solveCyclePath(path, theta, u, v, w)
	} else {
		solveOpenPath(path, theta, u, v)
	}
	setControls(path, theta, controls) // set control points from theta angles
	return controls
}

func solveOpenPath(path HobbyPath, theta, u, v []float64) {
	startOpen(path, theta, u, v)
	buildEqs(path, u, v, nil)
	endOpen(path, theta, u, v)
}

func solveCyclePath(path HobbyPath, theta, u, v, w []float64) {
	startCycle(path, theta, u, v, w)
	buildEqs(path, u, v, w)
	endCycle(path, theta, u, v, w)
}

func startOpen(path HobbyPath, theta, u, v []float64) {
	if cmplx.IsNaN(path.PostDir(0).C()) {
		a := recip(path.PostTension(0))
		b := recip(path.PreTension(1))
		T().Debugf("path.PostCurl(0) = %.4g", path.PostCurl(0))
		c := square(a) * path.PostCurl(0) / square(b)
		T().Debugf("a = %.4g, b = %.4g, c = %.4g", a, b, c)
		u[0] = ((3-a)*c + b) / (a*c + 3 - b)
		v[0] = -u[0] * psi(path, 1)
	} else {
		u[0] = 0
		v[0] = reduceAngle(angle(path.PostDir(0)) - angle(delta(path, 0)))
	}
	T().Debugf("u.0 = %.4g, v.0 = %.4g", u[0], v[0])
}

func endOpen(path HobbyPath, theta, u, v []float64) {
	last := path.N() - 1
	if cmplx.IsNaN(path.PreDir(last).C()) {
		a := recip(path.PostTension(last - 1))
		b := recip(path.PreTension(last))
		T().Debugf("path.PreCurl(%d) = %.4g", last, path.PostCurl(last))
		c := square(b) * path.PreCurl(last) / square(a)
		u[last] = (b*c + 3 - a) / ((3-b)*c + a)
		T().Debugf("u.%d = %g", last, u[last])
		theta[last] = v[last-1] / (u[last-1] - u[last])
	} else {
		theta[last] = reduceAngle(angle(path.PreDir(last)) - angle(delta(path, last-1)))
	}
	T().Debugf("theta.%d = %.4g", last, rad2deg(theta[last]))
	for i := last - 1; i >= 0; i-- {
		theta[i] = v[i] - u[i]*theta[i+1]
		T().Debugf("theta.%d = %.4g", i, rad2deg(theta[i]))
	}
}

func startCycle(path HobbyPath, theta, u, v, w []float64) {
	u[0], v[0], w[0] = 0, 0, 1
}

func endCycle(path HobbyPath, theta, u, v, w []float64) {
	n := path.N()
	var a, b float64 = 0, 1
	for i := n; i > 0; i-- {
		a = v[i] - a*u[i]
		b = w[i] - b*u[i]
	}
	t0 := (v[n] - a*u[n]) / (1 - (w[n] - b*u[n]))
	v[0] = t0
	for i := 1; i <= n; i++ {
		v[i] += w[i] * t0
	}
	theta[0], theta[n] = t0, t0
	for i := n - 1; i > 0; i-- {
		theta[i] = v[i] - u[i]*theta[i+1]
	}
	/*
	   for i := 0; i <= n; i++ {
	       fmt.Printf("theta.%d = %.2g\n", i, rad2deg(theta[i]))
	   }
	*/
}

func buildEqs(path HobbyPath, u, v, w []float64) {
	n := path.N()
	for i := 1; i <= n; i++ {
		a0 := recip(path.PostTension(i - 1))
		a1 := recip(path.PostTension(i))
		b1 := recip(path.PreTension(i))
		b2 := recip(path.PreTension(i + 1))
		T().Debugf("1/tensions: %.4g, %.4g, %.4g, %.4g", a0, a1, b1, b2)
		A := a0 / (square(b1) * d(path, i-1))
		B := (3 - a0) / (square(b1) * d(path, i-1))
		C := (3 - b2) / (square(a1) * d(path, i))
		D := b2 / (square(a1) * d(path, i))
		T().Debugf("A, B, C, D: %.4g, %.4g, %.4g, %.4g", A, B, C, D)
		t := B - u[i-1]*A + C
		u[i] = D / t
		v[i] = (-B*psi(path, i) - D*psi(path, i+1) - A*v[i-1]) / t
		if path.IsCycle() {
			w[i] = -A * w[i-1] / t
		}
		T().Debugf("u.%d = %.4g, v.%d = %.4g", i, u[i], i, v[i])
	}
}

func setControls(path HobbyPath, theta []float64, controls SplineControls) SplineControls {
	/*
	   const_a := 1.41421356     // sqrt(2) -- empiric constants, as explained by J.Hobby
	   const_b := 0.0625         // 1/16
	   const_c := 0.38196601125  // (3 - sqrt(5)) / 2
	   const_cc := 0.61803398875 // 1 - c
	*/
	n := path.N()
	for i := 0; i < n; i++ {
		phi := -psi(path, i+1) - theta[i+1]
		//fmt.Printf("#### phi(%d) = %.2g\n", i, rad2deg(phi))
		//fmt.Printf("phi.%d = %.4g - %.4g = %.4g\n", i, rad2deg(-path.psi(i+1)),
		//  rad2deg(theta[i+1]), rad2deg(phi))
		/*
		   a := recip(path.posttension(i))
		   b := recip(path.pretension(i + 1))
		   //
		       st := math.Sin(theta[i])
		       ct := math.Cos(theta[i])
		       sf := math.Sin(phi)
		       cf := math.Cos(phi)
		           alpha := const_a * (st - const_b*sf) * (sf - const_b*st) * (ct - cf)
		           beta := 1 + const_cc*ct + const_c*cf
		           //rho := (2 + alpha) / beta
		           //sigma := (2 - alpha) / beta
		*/
		/*
		   alpha, beta := hobbyParamsAlphaBeta(theta[i], phi)
		   rho, sigma := hobbyParamsRhoSigma(alpha, beta)
		          rho := complex(a/3*((2+alpha)/beta), 0)
		          sigma := complex(b/3*((2-alpha)/beta), 0)
		       xpart, ypart := real(path.delta(i)), imag(path.delta(i))
		       pci := complex(xpart*ct-ypart*st, xpart*st+ypart*ct) * rho
		       pcii := complex(xpart*cf+ypart*sf, -xpart*sf+ypart*cf) * sigma
		       path.postc[i%n] = path.z(i) + pci
		       path.prec[(i+1)%n] = path.z(i+1) - pcii
		*/
		a := recip(path.PostTension(i))
		b := recip(path.PreTension(i + 1))
		dvec := delta(path, i)
		p2, p3 := controlPoints(i, phi, theta[i], a, b, dvec)
		controls.SetPostControl(i%n, path.Z(i)+p2)
		controls.SetPreControl((i+1)%n, path.Z(i+1)-p3)
	}
	if gconf.IsSet("tracingchoices") {
		T().Infof(AsString(path, controls))
	}
	return controls
}

func hobbyParamsAlphaBeta(theta, phi float64) (float64, float64) {
	const_a := 1.41421356     // sqrt(2) -- empiric constants, as explained by J.Hobby
	const_b := 0.0625         // 1/16
	const_c := 0.38196601125  // (3 - sqrt(5)) / 2
	const_cc := 0.61803398875 // 1 - c
	st := math.Sin(theta)     // in-angle
	ct := math.Cos(theta)
	sf := math.Sin(phi) // out-angle
	cf := math.Cos(phi)
	alpha := const_a * (st - const_b*sf) * (sf - const_b*st) * (ct - cf)
	beta := 1 + const_cc*ct + const_c*cf
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
	//dx, dy := real(path.delta(i)), imag(path.delta(i))
	dx, dy := real(dvec), imag(dvec)
	uv1 := arithm.P(dx*ct-dy*st, dx*st+dy*ct)
	uv2 := arithm.P(dx*cf+dy*sf, -dx*sf+dy*cf)
	return uv1, uv2
}

/* Calculate control points between z.i and z.[i+1]
 */
//func (path *Path) controlPoints(i int, phi, theta, rho, sigma float64) {
func controlPoints(i int, phi, theta, a, b float64, dvec arithm.Pair) (arithm.Pair, arithm.Pair) {
	/*
	   n := path.n()
	   a := recip(path.posttension(i))
	   b := recip(path.pretension(i + 1))
	       crho := complex(a/3*rho, 0)
	       csigma := complex(b/3*sigma, 0)
	           dx, dy := real(path.delta(i)), imag(path.delta(i))
	           pci := complex(dx*ct-dy*st, dx*st+dy*ct) * crho
	           pcii := complex(dx*cf+dy*sf, -dx*sf+dy*cf) * csigma
	           path.postc[i%n] = path.z(i) + pci
	           path.prec[(i+1)%n] = path.z(i+1) - pcii
	*/
	alpha, beta := hobbyParamsAlphaBeta(theta, phi)
	rho, sigma := hobbyParamsRhoSigma(alpha, beta)
	uv1, uv2 := cunitvecs(i, theta, phi, dvec)
	crho := arithm.P(a/3*rho, 0)
	csigma := arithm.P(b/3*sigma, 0)
	p2 := crho * uv1
	p3 := csigma * uv2
	/*
	   path.postc[i%n] = path.z(i) + complex(a/3, 0)*uv1
	   path.prec[(i+1)%n] = path.z(i+1) - complex(b/3, 0)*uv2
	   fmt.Printf("#### post-control of %d = %.1f\n", i, path.postc[i%n])
	   fmt.Printf("#### pre-control of %d  = %.1f\n", i+1, path.prec[(i+1)%n])
	*/
	return p2, p3
}

// --- Splitting Paths into Segments -----------------------------------------

/* Split a path into segments, breaking it up at "rough" knots. Rough knots
 * are those with parameters which create a discontinuity.
 */
func splitSegments(path HobbyPath) []*pathPartial {
	var segments []*pathPartial
	segcnt, at := 0, 0
	for i := 1; i < path.N(); i++ {
		//T().Debugf("analyzing z.%d = %s\n", i, ptstring(path.Z(i), false))
		if isrough(path, i) {
			segments = append(segments, makePathSegment(path, at, i))
			segcnt++
			at = i
		}
	}
	if path.IsCycle() {
		if segcnt == 0 {
			segments = append(segments, makePathSegment(path, 0, last(path)))
		} else {
			segments = append(segments, makePathSegment(path, at, path.N()))
		}
	} else if at != last(path) {
		segments = append(segments, makePathSegment(path, at, last(path)))
	}
	return segments
}

/* Create a path segment at a breakpoint of a parent path.
 * This will create a kind of "projection" onto a subset of knots of
 * the parent path.
 */
func makePathSegment(path HobbyPath, from, to int) *pathPartial {
	partial := &pathPartial{
		whole: path, // parent path
		start: from, // first index within parent path
		end:   to,   // last index within parent path
	}
	if gconf.IsSet("tracingchoices") {
		T().Debugf("breaking segment %d - %d of length %d, at %s and %s", from, to, partial.N(),
			ptstring(path.Z(from), false), ptstring(path.Z(to), false))
		T().Infof("partial = %s", AsString(partial, nil))
	}
	return partial
}

// === Utilities =============================================================

func last(path HobbyPath) int {
	return path.N() - 1
}

func delta(path HobbyPath, i int) arithm.Pair {
	delta := path.Z(i+1) - path.Z(i)
	return delta
}

func d(path HobbyPath, i int) float64 {
	r, _ := cmplx.Polar(delta(path, i).C())
	return r
}

/* Turning angle at z.i
 */
func psi(path HobbyPath, i int) float64 {
	psi := 0.0
	if path.IsCycle() || (i > 0 && i < path.N()-1) {
		psi = cmplx.Phase(delta(path, i).C()) - cmplx.Phase(delta(path, i-1).C())
	}
	return reduceAngle(psi)
}

// Is a knot a breakpoint for splitting a path into segments?
func isrough(path HobbyPath, i int) bool {
	lc, rc := path.PreCurl(i), path.PostCurl(i)
	hascurl := lc != 1 || rc != 1
	ld, rd := path.PreDir(i), path.PostDir(i)
	has2dirs := (!cmplx.IsNaN(ld.C()) && !cmplx.IsNaN(rd.C())) && !equal(ld, rd)
	if hascurl || has2dirs {
		return true
	}
	return false
}

// --- Helpers ---------------------------------------------------------------

/* Extend an array/slice of complex numbers to make room for index i.
 * Will do nothing if the array is already large enough. Added entries
 * are assigned a default value of deflt.
 */
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

/* Get a complex number from an array/slice if present, default value
 * deflt otherwise.
 */
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

/* Reduce an angle to fit int -pi .. pi.
 */
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

/* Return 1/a for a.
 */
func recip(a float64) float64 {
	if math.IsNaN(a) {
		return 1.0
	}
	return 1.0 / a
}

/* Return a^2 for a.
 */
func square(a float64) float64 {
	return math.Pow(a, 2.0)
}

// P is a quick notation for contructing a pair, i.e. knot coordinates.
// Use it during path creation:
//   Knot(P(0,0))  // knot at origin
// func P(x, y float64) arithm.Pair {
// 	return arithm.Pair(y, y)
// }

func rad2deg(a float64) float64 {
	return a * 180 / pi
}

func ptstring(p arithm.Pair, iscontrol bool) string {
	if cmplx.IsNaN(p.C()) {
		return "(<unknown>)"
	} else if iscontrol {
		return fmt.Sprintf("(%.4f,%.4f)", round(p.X()), round(p.Y()))
	} else {
		return fmt.Sprintf("(%.4g,%.4g)", round(p.X()), round(p.Y()))
	}
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
