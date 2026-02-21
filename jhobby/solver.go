package jhobby

import (
	"fmt"
	"math"
	"math/cmplx"
)

// ValidateForSolve checks if a path is solvable by Hobby interpolation.
func (path *Path) ValidateForSolve() error {
	if path == nil {
		return ErrNilPath
	}
	n := path.N()
	if path.IsCycle() {
		if n < 3 {
			return fmt.Errorf("%w: cycle needs at least 3 knots, got %d", ErrTooFewKnots, n)
		}
		if cmplx.Abs((path.points[0] - path.points[n-1]).C()) <= _epsilon {
			return ErrCycleHasDuplicateTerminalKnot
		}
	} else if n < 2 {
		return fmt.Errorf("%w: open path needs at least 2 knots, got %d", ErrTooFewKnots, n)
	}
	for i := 0; i < n; i++ {
		z := path.points[i]
		x, y := real(z), imag(z)
		if math.IsNaN(x) || math.IsNaN(y) || math.IsInf(x, 0) || math.IsInf(y, 0) {
			return fmt.Errorf("%w at knot %d", ErrInvalidKnot, i)
		}
	}
	limit := n - 1
	if path.IsCycle() {
		limit = n
	}
	for i := 0; i < limit; i++ {
		j := i + 1
		if path.IsCycle() {
			j = (i + 1) % n
		}
		if cmplx.Abs((path.points[j] - path.points[i]).C()) <= _epsilon {
			return fmt.Errorf("%w between knots %d and %d", ErrDegenerateSegment, i, j)
		}
	}
	return nil
}

// FindHobbyControls finds the parameters for Hobby-spline control points
// for a given skeleton path.
// It validates the path and returns an error for empty/invalid geometry.
//
// BUG(norbert@pillmayer.com): Currently there are slight deviations from
// MetaFont's calculation, probably due to different rounding. These are under
// investigation.
func FindHobbyControls(path *Path, controls *Controls) (*Controls, error) {
	if err := path.ValidateForSolve(); err != nil {
		return nil, err
	}
	if controls == nil {
		controls = &Controls{}
	}
	segments := splitSegments(path)
	if len(segments) > 0 {
		for _, segment := range segments {
			if err := validateSegment(segment); err != nil {
				return nil, err
			}
			segment.controls = controls
			tracer().Infof("find controls for segment %s", asStringPartial(segment, nil))
			findSegmentControls(segment, segment.controls)
		}
	}
	return controls, nil
}

// MustFindHobbyControls is a compatibility helper which panics on validation errors.
func MustFindHobbyControls(path *Path, controls *Controls) *Controls {
	c, err := FindHobbyControls(path, controls)
	if err != nil {
		panic(err)
	}
	return c
}

/*
Find the Control Points according to Hobby's Algorithm. This is the
central API function of this package.

Clients may provide a container for the spline control points. If none
is provided, i.e. controls == nil, this function will allocate one.

The function validates path geometry and returns an error if the path
cannot be solved safely.

FindHobbyControls(...) will trace the calculated final path using log-level
INFO, if tracingchoices=true (as MetaFont does).
*/
func findSegmentControls(path *pathPartial, controls *Controls) *Controls {
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

func solveOpenPath(path *pathPartial, theta, u, v []float64) {
	startOpen(path, theta, u, v)
	buildEqs(path, u, v, nil)
	endOpen(path, theta, u, v)
}

func solveCyclePath(path *pathPartial, theta, u, v, w []float64) {
	startCycle(path, theta, u, v, w)
	buildEqs(path, u, v, w)
	endCycle(path, theta, u, v, w)
}

func startOpen(path *pathPartial, theta, u, v []float64) {
	if cmplx.IsNaN(path.PostDir(0).C()) {
		a := recip(path.PostTension(0))
		b := recip(path.PreTension(1))
		tracer().Debugf("path.PostCurl(0) = %.4g", path.PostCurl(0))
		c := square(a) * path.PostCurl(0) / square(b)
		tracer().Debugf("a = %.4g, b = %.4g, c = %.4g", a, b, c)
		u[0] = ((3-a)*c + b) / (a*c + 3 - b)
		v[0] = -u[0] * path.psi(1)
	} else {
		u[0] = 0
		v[0] = reduceAngle(angle(path.PostDir(0)) - angle(path.delta(0)))
	}
	tracer().Debugf("u.0 = %.4g, v.0 = %.4g", u[0], v[0])
}

func endOpen(path *pathPartial, theta, u, v []float64) {
	last := path.N() - 1
	if cmplx.IsNaN(path.PreDir(last).C()) {
		a := recip(path.PostTension(last - 1))
		b := recip(path.PreTension(last))
		tracer().Debugf("path.PreCurl(%d) = %.4g", last, path.PostCurl(last))
		c := square(b) * path.PreCurl(last) / square(a)
		u[last] = (b*c + 3 - a) / ((3-b)*c + a)
		tracer().Debugf("u.%d = %g", last, u[last])
		theta[last] = v[last-1] / (u[last-1] - u[last])
	} else {
		theta[last] = reduceAngle(angle(path.PreDir(last)) - angle(path.delta(last-1)))
	}
	tracer().Debugf("theta.%d = %.4g", last, rad2deg(theta[last]))
	for i := last - 1; i >= 0; i-- {
		theta[i] = v[i] - u[i]*theta[i+1]
		tracer().Debugf("theta.%d = %.4g", i, rad2deg(theta[i]))
	}
}

func startCycle(path *pathPartial, theta, u, v, w []float64) {
	u[0], v[0], w[0] = 0, 0, 1
}

func endCycle(path *pathPartial, theta, u, v, w []float64) {
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
}

func buildEqs(path *pathPartial, u, v, w []float64) {
	n := path.N()
	for i := 1; i <= n; i++ {
		a0 := recip(path.PostTension(i - 1))
		a1 := recip(path.PostTension(i))
		b1 := recip(path.PreTension(i))
		b2 := recip(path.PreTension(i + 1))
		tracer().Debugf("1/tensions: %.4g, %.4g, %.4g, %.4g", a0, a1, b1, b2)
		A := a0 / (square(b1) * path.d(i-1))
		B := (3 - a0) / (square(b1) * path.d(i-1))
		C := (3 - b2) / (square(a1) * path.d(i))
		D := b2 / (square(a1) * path.d(i))
		tracer().Debugf("A, B, C, D: %.4g, %.4g, %.4g, %.4g", A, B, C, D)
		t := B - u[i-1]*A + C
		u[i] = D / t
		v[i] = (-B*path.psi(i) - D*path.psi(i+1) - A*v[i-1]) / t
		if path.IsCycle() {
			w[i] = -A * w[i-1] / t
		}
		tracer().Debugf("u.%d = %.4g, v.%d = %.4g", i, u[i], i, v[i])
	}
}

func setControls(path *pathPartial, theta []float64, controls *Controls) *Controls {
	n := path.N()
	for i := 0; i < n; i++ {
		phi := -path.psi(i+1) - theta[i+1]
		a := recip(path.PostTension(i))
		b := recip(path.PreTension(i + 1))
		dvec := path.delta(i)
		p2, p3 := controlPoints(i, phi, theta[i], a, b, dvec)
		controls.SetPostControl(i%n, path.Z(i)+p2)
		controls.SetPreControl((i+1)%n, path.Z(i+1)-p3)
	}
	tracer().Infof(asStringPartial(path, controls))
	return controls
}
