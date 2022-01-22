package template

import (
	"gocv.io/x/gocv"

	"github.com/pidgy/unitehud/filter"
	"github.com/rs/zerolog"
)

type Template struct {
	filter.Filter
	gocv.Mat
	Category    string
	Subcategory string
}

func (t Template) MarshalZerologObject(e *zerolog.Event) {
	e.Object("filter", t.Filter).
		Int("mrows", t.Mat.Rows()).
		Int("mcols", t.Mat.Cols())
}
