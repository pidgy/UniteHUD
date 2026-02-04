package cursor

import (
	"gioui.org/io/pointer"
	"gioui.org/layout"
)

var (
	is = pointer.CursorDefault
)

// Draw applies the current cursor to the layout operations.
func Draw(gtx layout.Context) {
	pointer.Cursor(is).Add(gtx.Ops)
}

// Is sets the cursor that Draw will apply.
func Is(c pointer.Cursor) {
	is = c
}
