package jhobby

import (
	"math/cmplx"

	"github.com/npillmayer/arithm"
)

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
	path.Controls = &Controls{}
	return path
}

// Nullpath creates an empty path, to be extended by subsequent builder
// calls. The following example builds a closed path of three knots, which are
// connected by a curve, then a straight line, and a curve again.
//
//	var path *Path
//	var controls *Controls
//	path = Nullpath().Knot(0,0).Curve().Knot(3,2).Line().Knot(5,2.5).Curve().Cycle()
//	controls = path.Controls
//
// Calling Cycle() or End() returns a path. Its control point container
// (path.Controls) is empty and to be filled by calculating the Hobby spline
// control points.
func Nullpath() *Path {
	return newSkeletonPath(nil)
}

// End an open path. Part of builder functionality.
func (path *Path) End() *Path {
	return path
}

// Cycle closes a cyclic path. Part of builder functionality.
func (path *Path) Cycle() *Path {
	path.cycle = true
	return path
}

// Knot adds a standard smooth knot to a path. Part of builder functionality.
func (path *Path) Knot(pr arithm.Pair) *Path {
	return path.SmoothKnot(pr)
}

// SmoothKnot adds a standard smooth knot to a path (same as Knot(pr)).
// Part of builder functionality.
func (path *Path) SmoothKnot(p arithm.Pair) *Path {
	path.points = append(path.points, p)
	return path
}

// CurlKnot adds a path with curl information to a path. Callers may specify pre- and/or
// post-curl. A curl value of 1.0 is considered neutral.
// Part of builder functionality.
func (path *Path) CurlKnot(p arithm.Pair, precurl, postcurl float64) *Path {
	path.points = append(path.points, p)
	path.SetPreCurl(path.N()-1, precurl)
	path.SetPostCurl(path.N()-1, postcurl)
	return path
}

// DirKnot adds a knot with a given tangent direction.
// Part of builder functionality.
func (path *Path) DirKnot(p arithm.Pair, dir arithm.Pair) *Path {
	path.points = append(path.points, p)
	path.SetPreDir(path.N()-1, dir)
	path.SetPostDir(path.N()-1, dir)
	return path
}

// Line connects two knots with a straight line.
// Part of builder functionality.
func (path *Path) Line() *Path {
	if path.N() == 0 {
		panic("cannot add line to empty path")
	}
	path.SetPostCurl(path.N()-1, 1.0)
	path.SetPreCurl(path.N(), 1.0)
	return path
}

// Curve connects two knots with a smooth curve.
// Part of builder functionality.
func (path *Path) Curve() *Path {
	if path.N() == 0 {
		panic("cannot add curve to empty path")
	}
	path.TensionCurve(1.0, 1.0)
	return path
}

// TensionCurve connects two knots with a tense curve.
// Part of builder functionality.
//
// Tensions are adapted to lie between 3/4 and 4 (absolute). Negative tensions
// are interpreted as "at least" tensions to ensure the spline stays within
// the bounding box at its control point.
//
// BUG(norbert@pillmayer.com): Tension spec "at least" currently not completely implemented.
func (path *Path) TensionCurve(t1, t2 float64) *Path {
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
func (path *Path) AppendSubpath(sp *Path) *Path {
	tracer().Errorf("AppendSubpath not yet implemented")
	return path
}

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
	c := path.curls[i]
	pre := real(c)
	path.curls[i] = arithm.P(pre, curl)
	return path
}

// SetPreTension is a property setter.
//
// Tensions are adapted to lie between 3/4 and 4 (absolute). Negative tensions
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
// Tensions are adapted to lie between 3/4 and 4 (absolute). Negative tensions
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

// IsCycle is a predicate: is this path cyclic?
func (path *Path) IsCycle() bool {
	return path.cycle
}

// N returns the length of this path (knot count). For cyclic paths, the first and last knot
// should count as one.
func (path *Path) N() int {
	return len(path.points)
}

// Z returns the knot at position (i mod N).
func (path *Path) Z(i int) arithm.Pair {
	if i < 0 || i >= path.N() {
		i = i % path.N()
	}
	z := path.points[i]
	return z
}

// PreDir gets the incoming tangent / direction vector at z.i.
func (path *Path) PreDir(i int) arithm.Pair {
	return getC(path.predirs, i, arithm.Pair(cmplx.NaN()))
}

// PostDir gets the outgoing tangent / direction vector at z.i.
func (path *Path) PostDir(i int) arithm.Pair {
	return getC(path.postdirs, i, arithm.Pair(cmplx.NaN()))
}

// PreCurl gets the curl before z.i.
func (path *Path) PreCurl(i int) float64 {
	c := getC(path.curls, i, 1+1i)
	return real(c)
}

// PostCurl gets the curl after z.i.
func (path *Path) PostCurl(i int) float64 {
	c := getC(path.curls, i, 1+1i)
	return imag(c)
}

// PreTension returns the tension before z.i.
func (path *Path) PreTension(i int) float64 {
	t := getC(path.tensions, i, 1+1i)
	return real(t)
}

// PostTension returns the tension after z.i.
func (path *Path) PostTension(i int) float64 {
	t := getC(path.tensions, i, 1+1i)
	return imag(t)
}
