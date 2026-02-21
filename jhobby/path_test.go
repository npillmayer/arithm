package jhobby

import (
	"errors"
	"fmt"
	"math"
	"testing"

	"github.com/npillmayer/arithm"
	"github.com/npillmayer/schuko/tracing"
	"github.com/npillmayer/schuko/tracing/gotestingadapter"
)

func mustPanic(t *testing.T, f func()) {
	t.Helper()
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("expected panic, got none")
		}
	}()
	f()
}

func mustFindControls(t *testing.T, path *Path, controls *Controls) *Controls {
	t.Helper()
	c, err := FindHobbyControls(path, controls)
	if err != nil {
		t.Fatalf("FindHobbyControls failed: %v", err)
	}
	return c
}

func testpath() (*Path, *Controls) {
	path := Nullpath().Knot(arithm.P(1, 1)).Curve().Knot(arithm.P(2, 2)).
		Curve().Knot(arithm.P(3, 1)).End()
	controls := path.Controls
	return path, controls
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

func TestAsStringSnapshots(t *testing.T) {
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	openPath := Nullpath().
		Knot(arithm.P(1, 1)).Curve().
		Knot(arithm.P(2, 2)).Curve().
		Knot(arithm.P(3, 1)).End()
	if got, want := AsString(openPath, nil), "(1,1) .. (2,2) .. (3,1)"; got != want {
		t.Fatalf("open AsString mismatch:\n got: %s\nwant: %s", got, want)
	}
	cyclePath := Nullpath().
		Knot(arithm.P(1, 1)).Curve().
		Knot(arithm.P(2, 2)).Curve().
		Knot(arithm.P(3, 1)).Curve().
		Knot(arithm.P(2, 0)).Curve().Cycle()
	if got, want := AsString(cyclePath, nil), "(1,1) .. (2,2) .. (3,1) .. (2,0) .. cycle"; got != want {
		t.Fatalf("cycle AsString mismatch:\n got: %s\nwant: %s", got, want)
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
	path := Nullpath().Knot(arithm.P(1, 1)).TensionCurve(1.0, 2.0).Cycle()
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
	path := Nullpath().DirKnot(arithm.P(1, 1), arithm.P(1, 0)).End()
	t.Logf("dir(0) = %g\n", path.PostDir(0))
	if angle(path.PostDir(0)) > 0.01 {
		t.Fail()
	}
}

/*
func TestCurl(t *testing.T) {
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	path := Nullpath().Knot(arithm.P(1, 1)).Line().Cycle()
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
	controls = mustFindControls(t, path, controls)
	t.Log(AsString(path, controls))
}

func TestCycle(t *testing.T) {
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	p, _ := testpath()
	path := p.Knot(arithm.P(2, 0)).Curve().Cycle()
	controls := path.Controls
	t.Log(AsString(path, nil))
	controls = mustFindControls(t, path, controls)
	t.Log(AsString(path, controls))
}

func TestControlsDeterministicSnapshot(t *testing.T) {
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	path := Nullpath().
		Knot(arithm.P(1, 1)).Curve().
		Knot(arithm.P(2, 2)).Curve().
		Knot(arithm.P(3, 1)).Curve().
		Knot(arithm.P(2, 0)).Curve().Cycle()
	controls := path.Controls
	controls = mustFindControls(t, path, controls)
	p0post := controls.PostControl(0)
	if math.Abs(p0post.X()-1.0000) > 0.0002 || math.Abs(p0post.Y()-1.5523) > 0.0002 {
		t.Fatalf("unexpected post control[0]: %v", p0post)
	}
	p1pre := controls.PreControl(1)
	if math.Abs(p1pre.X()-1.4477) > 0.0002 || math.Abs(p1pre.Y()-2.0000) > 0.0002 {
		t.Fatalf("unexpected pre control[1]: %v", p1pre)
	}
	p2post := controls.PostControl(2)
	if math.Abs(p2post.X()-3.0000) > 0.0002 || math.Abs(p2post.Y()-0.4477) > 0.0002 {
		t.Fatalf("unexpected post control[2]: %v", p2post)
	}
}

// Draw a cicle with diameter 1 around (2,1). The builder statement returns
// a concrete Path and Controls. Type Path actually
// contains a link to its spline controls (field path.Controls). These controls
// are initially empty and then used for the call to FindHobbyControls(...),
// where they get filled.
func ExampleControls_usage() {
	path := Nullpath().Knot(arithm.P(1, 1)).Curve().Knot(arithm.P(2, 2)).Curve().Knot(arithm.P(3, 1)).
		Curve().Knot(arithm.P(2, 0)).Curve().Cycle()
	controls := path.Controls
	fmt.Printf("skeleton path = %s\n\n", AsString(path, nil))
	fmt.Printf("unknown path = \n%s\n\n", AsString(path, controls))
	controls = MustFindHobbyControls(path, controls)
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
	path := Nullpath().Knot(arithm.P(1, 1)).Curve().Knot(arithm.P(2, 2)).Curve().Knot(arithm.P(3, 1)).End()
	seg := makePathSegment(path, 0, 1)
	if seg.N() != 2 {
		t.Fail()
	}
}

/*
func TestSegments(t *testing.T) {
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	path := Nullpath().Knot(arithm.P(0, 0)).Curve().Knot(arithm.P(0, 3)).Curve().
		Knot(arithm.P(5, 3)).Line().DirKnot(arithm.P(3, -1), arithm.P(0, -1)).Curve().Cycle()
	segs := splitSegments(path)
	if len(segs) != 4 {
		t.Fail()
	}
}
*/

func TestSegmentsSplitBaseline(t *testing.T) {
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	path := Nullpath().
		Knot(arithm.P(0, 0)).Curve().
		Knot(arithm.P(0, 3)).Curve().
		Knot(arithm.P(5, 3)).Line().
		DirKnot(arithm.P(3, -1), arithm.P(0, -1)).
		Curve().Cycle()
	segs := splitSegments(path)
	if len(segs) != 1 {
		t.Fatalf("unexpected segment count for non-rough path: got %d, want 1", len(segs))
	}

	roughPath := Nullpath().
		Knot(arithm.P(0, 0)).Curve().
		Knot(arithm.P(1, 1)).Curve().
		Knot(arithm.P(2, 0)).End()
	roughPath.SetPreCurl(1, 2.0)
	segs = splitSegments(roughPath)
	if len(segs) != 2 {
		t.Fatalf("unexpected segment count for rough path: got %d, want 2", len(segs))
	}
	if segs[0].start != 0 || segs[0].end != 1 || segs[1].start != 1 || segs[1].end != 2 {
		t.Fatalf("unexpected rough segment bounds: [%d,%d] [%d,%d]",
			segs[0].start, segs[0].end, segs[1].start, segs[1].end)
	}
}

func TestSegmentedPath(t *testing.T) {
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	tracer().SetTraceLevel(tracing.LevelInfo)
	path := Nullpath().Knot(arithm.P(1, 1)).Line().Knot(arithm.P(2, 2)).Line().Knot(arithm.P(3, 1)).End()
	controls := path.Controls
	controls = mustFindControls(t, path, controls)
}

func TestFindHobbyControlsRejectsNilPath(t *testing.T) {
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	_, err := FindHobbyControls(nil, nil)
	if !errors.Is(err, ErrNilPath) {
		t.Fatalf("expected ErrNilPath, got %v", err)
	}
}

func TestFindHobbyControlsRejectsTooFewKnotsOpen(t *testing.T) {
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	path := Nullpath().Knot(arithm.P(0, 0)).End()
	_, err := FindHobbyControls(path, path.Controls)
	if !errors.Is(err, ErrTooFewKnots) {
		t.Fatalf("expected ErrTooFewKnots, got %v", err)
	}
}

func TestFindHobbyControlsRejectsTooFewKnotsCycle(t *testing.T) {
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	path := Nullpath().Knot(arithm.P(0, 0)).Curve().Knot(arithm.P(1, 0)).Curve().Cycle()
	_, err := FindHobbyControls(path, path.Controls)
	if !errors.Is(err, ErrTooFewKnots) {
		t.Fatalf("expected ErrTooFewKnots, got %v", err)
	}
}

func TestFindHobbyControlsRejectsDegenerateSegment(t *testing.T) {
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	path := Nullpath().Knot(arithm.P(0, 0)).Curve().Knot(arithm.P(0, 0)).End()
	_, err := FindHobbyControls(path, path.Controls)
	if !errors.Is(err, ErrDegenerateSegment) {
		t.Fatalf("expected ErrDegenerateSegment, got %v", err)
	}
}

func TestFindHobbyControlsRejectsInvalidKnot(t *testing.T) {
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	path := Nullpath().Knot(arithm.P(0, 0)).Curve().Knot(arithm.P(math.NaN(), 0)).End()
	_, err := FindHobbyControls(path, path.Controls)
	if !errors.Is(err, ErrInvalidKnot) {
		t.Fatalf("expected ErrInvalidKnot, got %v", err)
	}
}

func TestFindHobbyControlsRejectsCycleDuplicateTerminalKnot(t *testing.T) {
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	path := Nullpath().
		Knot(arithm.P(0, 0)).Curve().
		Knot(arithm.P(1, 0)).Curve().
		Knot(arithm.P(0, 0)).Curve().Cycle()
	_, err := FindHobbyControls(path, path.Controls)
	if !errors.Is(err, ErrCycleHasDuplicateTerminalKnot) {
		t.Fatalf("expected ErrCycleHasDuplicateTerminalKnot, got %v", err)
	}
}

func TestMustFindHobbyControlsPanicsOnInvalidPath(t *testing.T) {
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	path := Nullpath().Knot(arithm.P(0, 0)).End()
	mustPanic(t, func() { MustFindHobbyControls(path, path.Controls) })
}

func TestEmptyPathJoinPanics(t *testing.T) {
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	mustPanic(t, func() { Nullpath().Line() })
	mustPanic(t, func() { Nullpath().Curve() })
	mustPanic(t, func() { Nullpath().TensionCurve(1.2, 0.9) })
}
