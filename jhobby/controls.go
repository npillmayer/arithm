package jhobby

import (
	"math/cmplx"

	"github.com/npillmayer/arithm"
)

// BUG(norbert@pillmayer.com): Currently it isn't possible to explicitly set
// control points. This may or may not change in the future.
func (ctrls *Controls) SetPreControl(i int, c arithm.Pair) {
	ctrls.prec = extendC(ctrls.prec, i, arithm.Pair(cmplx.NaN()))
	ctrls.prec[i] = c
}

func (ctrls *Controls) SetPostControl(i int, c arithm.Pair) {
	ctrls.postc = extendC(ctrls.postc, i, arithm.Pair(cmplx.NaN()))
	ctrls.postc[i] = c
}

func (ctrls *Controls) PreControl(i int) arithm.Pair {
	return getC(ctrls.prec, i, arithm.Pair(cmplx.NaN()))
}

func (ctrls *Controls) PostControl(i int) arithm.Pair {
	return getC(ctrls.postc, i, arithm.Pair(cmplx.NaN()))
}
