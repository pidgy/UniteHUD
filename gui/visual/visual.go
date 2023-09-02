package visual

import "gioui.org/layout"

type Widgeter interface {
	Layout(layout.Context) layout.Dimensions
}
