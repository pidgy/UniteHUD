package template

import (
	"gocv.io/x/gocv"

	"github.com/pidgy/unitehud/core/template/filter"
)

// Template holds image data, mask state, and template metadata.
type Template struct {
	filter.Filter
	gocv.Mat
	Category string
	Mask     gocv.Mat
}

// New constructs a Template with the provided filter, image, and category.
func New(f filter.Filter, m gocv.Mat, category, subcategory string) *Template {
	return &Template{
		Filter:   f,
		Mat:      m,
		Category: category,
		Mask:     gocv.NewMat(),
	}
}

// Collection flattens template slices into a single slice.
func Collection(t ...[]*Template) []*Template {
	c := []*Template{}
	for _, p := range t {
		c = append(c, p...)
	}
	return c
}

// AsTransparent builds a transparency mask for the template image.
func (t *Template) AsTransparent() *Template {
	gocv.CvtColor(t.Mat, &t.Mask, gocv.ColorBGRAToBGR)
	return t
}
