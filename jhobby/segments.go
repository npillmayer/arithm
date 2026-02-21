package jhobby

import (
	"fmt"
	"math/cmplx"

	"github.com/npillmayer/arithm"
)

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

func (pp *pathPartial) delta(i int) arithm.Pair {
	return pp.Z(i+1) - pp.Z(i)
}

func (pp *pathPartial) d(i int) float64 {
	r, _ := cmplx.Polar(pp.delta(i).C())
	return r
}

// Turning angle at z.i.
func (pp *pathPartial) psi(i int) float64 {
	psi := 0.0
	if pp.IsCycle() || (i > 0 && i < pp.N()-1) {
		psi = cmplx.Phase(pp.delta(i).C()) - cmplx.Phase(pp.delta(i-1).C())
	}
	return reduceAngle(psi)
}

func asStringPartial(path *pathPartial, contr *Controls) string {
	var s string
	for i := 0; i < path.N(); i++ {
		pt := path.Z(i)
		if i > 0 {
			if contr != nil {
				s += fmt.Sprintf(" and %s\n  .. ", ptstring(contr.PreControl(path.pmap(i)), true))
			} else {
				s += " .. "
			}
		}
		s += fmt.Sprintf("%s", ptstring(pt, false))
		if contr != nil && (i < path.N()-1 || path.IsCycle()) {
			s += fmt.Sprintf(" .. controls %s", ptstring(contr.PostControl(path.pmap(i)), true))
		}
	}
	if path.IsCycle() {
		if contr != nil {
			s += fmt.Sprintf(" and %s\n ", ptstring(contr.PreControl(path.pmap(0)), true))
		}
		s += " .. cycle"
	}
	return s
}

// Split a path into segments, breaking it up at "rough" knots.
// Rough knots are those with parameters creating a discontinuity.
func splitSegments(path *Path) []*pathPartial {
	var segments []*pathPartial
	segcnt, at := 0, 0
	for i := 1; i < path.N(); i++ {
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

// Create a path segment as a projection onto a parent path subset.
func makePathSegment(path *Path, from, to int) *pathPartial {
	partial := &pathPartial{
		whole: path,
		start: from,
		end:   to,
	}
	tracer().Debugf("breaking segment %d - %d of length %d, at %s and %s", from, to, partial.N(),
		ptstring(path.Z(from), false), ptstring(path.Z(to), false))
	tracer().Infof("partial = %s", asStringPartial(partial, nil))
	return partial
}

func validateSegment(seg *pathPartial) error {
	if seg == nil || seg.whole == nil {
		return ErrNilPath
	}
	if seg.N() < 2 {
		return fmt.Errorf("%w: segment has %d knots", ErrTooFewKnots, seg.N())
	}
	limit := seg.N() - 1
	if seg.IsCycle() {
		limit = seg.N()
	}
	for i := 0; i < limit; i++ {
		if cmplx.Abs(seg.delta(i).C()) <= _epsilon {
			j := i + 1
			if seg.IsCycle() {
				j = (i + 1) % seg.N()
			}
			return fmt.Errorf("%w in segment between %d and %d", ErrDegenerateSegment, i, j)
		}
	}
	return nil
}

func last(path *Path) int {
	return path.N() - 1
}

func delta(path *Path, i int) arithm.Pair {
	return path.Z(i+1) - path.Z(i)
}

func d(path *Path, i int) float64 {
	r, _ := cmplx.Polar(delta(path, i).C())
	return r
}

// Turning angle at z.i.
func psi(path *Path, i int) float64 {
	psi := 0.0
	if path.IsCycle() || (i > 0 && i < path.N()-1) {
		psi = cmplx.Phase(delta(path, i).C()) - cmplx.Phase(delta(path, i-1).C())
	}
	return reduceAngle(psi)
}

// Is a knot a breakpoint for splitting a path into segments?
func isrough(path *Path, i int) bool {
	lc, rc := path.PreCurl(i), path.PostCurl(i)
	hascurl := lc != 1 || rc != 1
	ld, rd := path.PreDir(i), path.PostDir(i)
	has2dirs := (!cmplx.IsNaN(ld.C()) && !cmplx.IsNaN(rd.C())) && !equal(ld, rd)
	if hascurl || has2dirs {
		return true
	}
	return false
}
