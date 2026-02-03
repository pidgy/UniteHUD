package template

// TODO: Add go style comments that reflect the purpose of each type, function, var, and const.

import (
	"gocv.io/x/gocv"

	"github.com/pidgy/unitehud/core/template/filter"
)

// Template defines Template behavior and state.
type Template struct {
	filter.Filter
	gocv.Mat
	Category string
	Mask     gocv.Mat
}

// New returns a new instance.
func New(f filter.Filter, m gocv.Mat, category, subcategory string) *Template {
	return &Template{
		Filter:   f,
		Mat:      m,
		Category: category,
		Mask:     gocv.NewMat(),
	}
}

func Collection(t ...[]*Template) []*Template {
	c := []*Template{}
	for _, p := range t {
		c = append(c, p...)
	}
	return c
}

func (t *Template) AsTransparent() *Template {
	gocv.CvtColor(t.Mat, &t.Mask, gocv.ColorBGRAToBGR)
	return t
}
