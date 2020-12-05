package arithm

import (
	"testing"

	"github.com/npillmayer/schuko/tracing/gotestingadapter"
)

// var two dec.Decimal
// var three dec.Decimal
// var six dec.Decimal
// var seven dec.Decimal

func TestNumericBasic(t *testing.T) {
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	a := 0.000000008
	if !Is0(a) {
		t.Errorf("Expected a to be zero, is not")
	}
}

func TestPairBasic(t *testing.T) {
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	p := P(3, 2)
	q := P(-3, -2)
	r := p + q
	if !r.IsOrigin() {
		t.Errorf("Expected p + q to be (0,0), is %v", r)
	}
}

func TestTranslation(t *testing.T) {
	teardown := gotestingadapter.RedirectTracing(t)
	defer teardown()
	if !P(1, 1).Shifted(P(-1, -1)).IsOrigin() {
		t.Errorf("Expected (1,1) shifted (-1,-1) to be origin, is not")
	}
	if !P(1, 0).Rotated(180 * Deg2Rad).Shifted(P(1, 0)).IsOrigin() {
		t.Errorf("Expected result to be origin, is not")
	}
}
