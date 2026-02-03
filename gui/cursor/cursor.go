package cursor

// TODO: Add go style comments that reflect the purpose of each type, function, var, and const.

import (
	"gioui.org/io/pointer"
	"gioui.org/layout"
)

var (
	is = pointer.CursorDefault
)

// Draw draws the widget.
func Draw(gtx layout.Context) {
	pointer.Cursor(is).Add(gtx.Ops)
}

// Is reports whether the condition holds.
func Is(c pointer.Cursor) {
	is = c
}
