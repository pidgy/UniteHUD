package ux

import "gioui.org/layout"

type (
	// Then wraps a callback to satisfy the Thener interface.
	Then struct {
		Do func()
	}

	// Thener represents a type that can execute a deferred action.
	Thener interface {
		Then()
	}

	// Widgeter abstracts a layout-capable widget.
	Widgeter interface {
		Layout(layout.Context) layout.Dimensions
	}
)

// Then invokes the stored callback.
func (t Then) Then() { t.Do() }
