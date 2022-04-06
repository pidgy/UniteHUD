package template

import (
	"gocv.io/x/gocv"

	"github.com/rs/zerolog"

	"github.com/pidgy/unitehud/filter"
)

type Template struct {
	filter.Filter
	gocv.Mat
	Category string
	Mask     gocv.Mat
}

func New(f filter.Filter, m gocv.Mat, category, subcategory string) Template {
	t := Template{
		Filter:   f,
		Mat:      m,
		Category: category,
		Mask:     gocv.NewMat(),
	}

	return t
}

func (t Template) AsTransparent() Template {
	gocv.CvtColor(t.Mat, &t.Mask, gocv.ColorBGRAToBGR)
	return t
}

func (t Template) MarshalZerologObject(e *zerolog.Event) {
	e.Object("filter", t.Filter).
		Int("mrows", t.Mat.Rows()).
		Int("mcols", t.Mat.Cols())
}
