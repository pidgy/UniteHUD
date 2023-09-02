package cursor

import (
	"gioui.org/io/pointer"
	"gioui.org/layout"
)

var (
	is = pointer.CursorDefault
)

func Is(c pointer.Cursor) {
	is = c
}

func Draw(gtx layout.Context) {
	pointer.Cursor(is).Add(gtx.Ops)
	// is = pointer.CursorDefault
}
