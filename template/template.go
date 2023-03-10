package template

import (
	"gocv.io/x/gocv"

	"github.com/pidgy/unitehud/filter"
)

type Template struct {
	filter.Filter
	gocv.Mat
	Category string
	Mask     gocv.Mat
}

func New(f filter.Filter, m gocv.Mat, category, subcategory string) *Template {
	return &Template{
		Filter:   f,
		Mat:      m,
		Category: category,
		Mask:     gocv.NewMat(),
	}
}

func (t *Template) AsTransparent() *Template {
	gocv.CvtColor(t.Mat, &t.Mask, gocv.ColorBGRAToBGR)
	return t
}
