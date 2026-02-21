package jhobby

import (
	"fmt"
	"math"
	"testing"

	"github.com/npillmayer/arithm"
	"github.com/npillmayer/schuko/tracing"
	"github.com/npillmayer/schuko/tracing/gotestingadapter"
)

func testpath() (*Path, SplineControls) {
	path, controls := Nullpath().Knot(arithm.P(1, 1)).Curve().Knot(arithm.P(2, 2)).
		Curve().Knot(arithm.P(3, 1)).End()
	return path.(*Path), controls
}

func TestSliceEnlargement(t *testing.T) {
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	arr := make([]arithm.Pair, 0)
	arr = extendC(arr, 3, 2+1i)
	c := arr[3]
	if c != 2+1i {
		t.Fail()
	}
}

func TestCreatePath(t *testing.T) {
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	path, _ := testpath()
	if path.N() != 3 {
		t.Fail()
	}
}

func TestPadding(t *testing.T) {
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	path, _ := testpath()
	path.cycle = true
	if path.Z(1) != path.Z(path.N()+1) {
		t.Fail()
	}
}

func TestSetTension(t *testing.T) {
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	path, _ := Nullpath().Knot(arithm.P(1, 1)).TensionCurve(1.0, 2.0).Cycle()
	if path.PostTension(0) < 0.99 {
		t.Fail()
	}
	if path.PreTension(1) < 1.99 {
		t.Fail()
	}
}

func TestDir(t *testing.T) {
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	path, _ := Nullpath().DirKnot(arithm.P(1, 1), arithm.P(1, 0)).End()
	t.Logf("dir(0) = %g\n", path.PostDir(0))
	if angle(path.PostDir(0)) > 0.01 {
		t.Fail()
	}
}

/*
func TestCurl(t *testing.T) {
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	path, _ := Nullpath().Knot(arithm.P(1, 1)).Line().Cycle()
	t.Logf("curl(0) = %g\n", path.PostCurl(0))
	if path.PostCurl(0) > 0.09 {
		t.Fail()
	}
}
*/

func TestDelta(t *testing.T) {
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	path, _ := testpath()
	delta1 := delta(path, 1)
	t.Logf("delta [1->2] = %g\n", delta1)
	if delta1 != 1-1i {
		t.Fail()
	}
}

func TestD(t *testing.T) {
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	path, _ := testpath()
	d := d(path, 2)
	t.Logf("d [2->3] = %g\n", d)
	if d < 1.9 {
		t.Fail()
	}
}

func TestPsi(t *testing.T) {
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	path, _ := testpath()
	psi := psi(path, 1)
	t.Logf("psi [1->2] = %g\n", rad2deg(psi)) // -90.0000001
	if math.Abs(rad2deg(psi)+90.0) > 0.01 {
		t.Fail()
	}
}

func TestPsiCycle(t *testing.T) {
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	path, _ := testpath()
	path.cycle = true
	psi := psi(path, 2)
	t.Logf("psi [2->3] = %g\n", rad2deg(psi)) // -134.9999997
	if math.Abs(rad2deg(psi)+135.0) > 0.01 {
		t.Fail()
	}
}

func TestPsiCyclePadding(t *testing.T) {
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	path, _ := testpath()
	path.cycle = true
	psi1 := psi(path, 1)
	t.Logf("psi [1->2] = %g\n", rad2deg(psi1)) // -90
	if math.Abs(rad2deg(psi1)+90.0) > 0.01 {
		t.Fail()
	}
	psiN1 := psi(path, path.N()+1)
	t.Logf("psi [4->5] = %g\n", rad2deg(psiN1)) // -90
	if math.Abs(math.Abs(psi1)-math.Abs(psiN1)) > 0.0001 {
		t.Fail()
	}
}

func TestOpen(t *testing.T) {
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	path, controls := testpath()
	t.Log(AsString(path, nil))
	controls = FindHobbyControls(path, controls)
	t.Log(AsString(path, controls))
}

func TestCycle(t *testing.T) {
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	p, _ := testpath()
	path, controls := p.Knot(arithm.P(2, 0)).Curve().Cycle()
	t.Log(AsString(path, nil))
	controls = FindHobbyControls(path, controls)
	t.Log(AsString(path, controls))
}

// Draw a cicle with diameter 1 around (2,1). The builder statement returns
// a HobbyPath (type Path under the hood) and SplineControls. Type Path actually
// contains a link to its spline controls (field path.Controls). These controls
// are initially empty and then used for the call to FindHobbyControls(...),
// where they get filled.
func ExampleSplineControls_usage() {
	path, controls := Nullpath().Knot(arithm.P(1, 1)).Curve().Knot(arithm.P(2, 2)).Curve().Knot(arithm.P(3, 1)).
		Curve().Knot(arithm.P(2, 0)).Curve().Cycle()
	fmt.Printf("skeleton path = %s\n\n", AsString(path, nil))
	fmt.Printf("unknown path = \n%s\n\n", AsString(path, controls))
	controls = FindHobbyControls(path, controls)
	fmt.Printf("smooth path = \n%s\n\n", AsString(path, controls))

	// skeleton path = (1,1) .. (2,2) .. (3,1) .. (2,0) .. cycle

	// unknown path =
	// (1,1) .. controls (<unknown>) and (<unknown>)
	//  .. (2,2) .. controls (<unknown>) and (<unknown>)
	//  .. (3,1) .. controls (<unknown>) and (<unknown>)
	//  .. (2,0) .. controls (<unknown>) and (<unknown>)
	//  .. cycle

	// smooth path =
	// (1,1) .. controls (1.0000,1.5523) and (1.4477,2.0000)
	//  .. (2,2) .. controls (2.5523,2.0000) and (3.0000,1.5523)
	//  .. (3,1) .. controls (3.0000,0.4477) and (2.5523,0.0000)
	//  .. (2,0) .. controls (1.4477,0.0000) and (1.0000,0.4477)
	//  .. cycle
}

func TestSegmentProjection(t *testing.T) {
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	path, _ := Nullpath().Knot(arithm.P(1, 1)).Curve().Knot(arithm.P(2, 2)).Curve().Knot(arithm.P(3, 1)).End()
	seg := makePathSegment(path, 0, 1)
	if seg.N() != 2 {
		t.Fail()
	}
}

/*
func TestSegments(t *testing.T) {
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	path, _ := Nullpath().Knot(arithm.P(0, 0)).Curve().Knot(arithm.P(0, 3)).Curve().
		Knot(arithm.P(5, 3)).Line().DirKnot(arithm.P(3, -1), arithm.P(0, -1)).Curve().Cycle()
	segs := splitSegments(path)
	if len(segs) != 4 {
		t.Fail()
	}
}
*/

func TestSegmentedPath(t *testing.T) {
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	tracer().SetTraceLevel(tracing.LevelInfo)
	path, controls := Nullpath().Knot(arithm.P(1, 1)).Line().Knot(arithm.P(2, 2)).Line().Knot(arithm.P(3, 1)).End()
	controls = FindHobbyControls(path, controls)
}
