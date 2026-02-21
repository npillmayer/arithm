package jhobby

import (
	"errors"

	"github.com/npillmayer/arithm"
	"github.com/npillmayer/schuko/tracing"
)

// tracer writes to trace with key 'graphics'
func tracer() tracing.Trace {
	return tracing.Select("graphics")
}

const pi float64 = 3.14159265
const pi2 float64 = 6.28318530
const _epsilon = 0.0000001

var (
	// ErrNilPath indicates a nil path pointer.
	ErrNilPath = errors.New("path must not be nil")
	// ErrTooFewKnots indicates path knot count is insufficient for solving.
	ErrTooFewKnots = errors.New("path has too few knots")
	// ErrInvalidKnot indicates a knot coordinate contains NaN/Inf.
	ErrInvalidKnot = errors.New("path has invalid knot coordinate")
	// ErrDegenerateSegment indicates two consecutive knots collapse to one point.
	ErrDegenerateSegment = errors.New("path has degenerate segment")
	// ErrCycleHasDuplicateTerminalKnot indicates cyclic path redundantly repeats first knot as last knot.
	ErrCycleHasDuplicateTerminalKnot = errors.New("cycle path must not repeat first knot as terminal knot")
)

// Path is the concrete type for building and solving Hobby splines.
// To construct a path, start with Nullpath(), which creates an empty
// path, and then extend it.
type Path struct {
	points   []arithm.Pair // point i
	cycle    bool          // is this path cyclic ?
	predirs  []arithm.Pair // explicit pre-direction at point i
	postdirs []arithm.Pair // explicit post-direction at point i
	curls    []arithm.Pair // explicit l and r curl at point i
	tensions []arithm.Pair // explicit pre- and post-tension at point i
	Controls *Controls     // control points to be calculated
}

// A segment view onto a parent path.
type pathPartial struct {
	whole    *Path     // parent path
	start    int       // first index within parent path
	end      int       // last index within parent path
	controls *Controls // control points, shared with parent path
}

// Controls collects calculated spline control points.
type Controls struct {
	prec  []arithm.Pair // control point i-, to be calculated
	postc []arithm.Pair // control point i+, to be calculated
}
