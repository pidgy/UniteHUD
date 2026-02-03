package ux

import "gioui.org/layout"

// TODO: Add go style comments that reflect the purpose of each type, function, var, and const.

type (
	Then struct {
		Do func()
	}

	Thener interface {
		Then()
	}

	Widgeter interface {
		Layout(layout.Context) layout.Dimensions
	}
)

func (t Then) Then() { t.Do() }
