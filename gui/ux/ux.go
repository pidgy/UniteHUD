package ux

import "gioui.org/layout"

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
