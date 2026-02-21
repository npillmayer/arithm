package polygon

import (
	"testing"

	"github.com/npillmayer/arithm"
	"github.com/npillmayer/schuko/tracing/gotestingadapter"
)

func TestBuilder(t *testing.T) {
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	pg := NullPolygon().Knot(arithm.P(0, 0)).Knot(arithm.P(1, 3)).Knot(arithm.P(3, 0)).Cycle()
	L().Infof("pg = %s", AsString(pg))
	if pg.N() != 3 {
		t.Fail()
	}
}

func TestBox(t *testing.T) {
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	box := Box(arithm.P(0, 5), arithm.P(4, 1))
	L().Infof("box = %s", AsString(box))
	if box.N() != 4 {
		t.Fail()
	}
}
