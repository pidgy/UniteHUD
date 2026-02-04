package template

// Sortable tracks templates and cached data for stable, sortable grouping.

import (
	"image"
	"math"
	"sort"

	"github.com/pidgy/unitehud/core/template/filter"
)

type (
	// Sortable stores templates, cached sort metadata, and invalidation state.
	Sortable struct {
		templates []*Template
		cache     map[string]cached
		invalid   bool
	}

	// byLocation sorts templates by cached X coordinate.
	byLocation Sortable
	// byValues sorts templates by cached value.
	byValues Sortable

	// cached holds the cached location, value, and seen count for a template.
	cached struct {
		image.Point
		value float32
		seen  int
	}
)

// NewSortable returns a new Sortable with an initialized cache.
func NewSortable() Sortable {
	return Sortable{
		cache: map[string]cached{},
	}
}

// ByLocation sorts by cached location and validates the result.
func ByLocation(t Sortable) bool {
	if t.invalid || len(t.cache) == 0 || len(t.cache) > 3 {
		return false
	}

	sort.Sort(byLocation(t))

	t.invalid = t.Value() < 1 || t.Value() > 100

	return !t.invalid
}

// ByValues sorts by cached values and validates the result.
func ByValues(t Sortable) bool {
	for _, c := range t.cache {
		if c.seen > 1 {
			return false
		}
	}
	sort.Sort(byValues(t))

	return t.Value() > 0 && t.Value() < 100
}

func (t *Sortable) Cache(t2 *Template, p image.Point, value float32) {
	if t2.Value == 0 {
		p = image.Pt(math.MaxInt32, math.MaxInt32)
	}

	file := filter.Strip(t2.File)

	c, ok := t.cache[file]
	if !ok {
		t.cache[file] = cached{p, value, 1}
		t.templates = append(t.templates, t2)
	} else {
		c.seen++
		t.cache[file] = c
		t.invalid = true
	}
}

// Value returns a composite value for the sorted templates.
func (t *Sortable) Value() int {
	switch len(t.templates) {
	case 1:
		return -1
	case 2:
		return t.templates[0].Value*10 + t.templates[1].Value
	case 3:
		return t.templates[0].Value*100 + t.templates[1].Value*10 + t.templates[2].Value
	default:
		return -1
	}
}

// Len returns the number of templates.
func (b byLocation) Len() int {
	return len(b.templates)
}

// Less reports whether template i should sort before j by X coordinate.
func (b byLocation) Less(i, j int) bool {
	return b.cache[b.templates[i].File].X < b.cache[b.templates[j].File].X
}

// Swap swaps templates i and j.
func (b byLocation) Swap(i, j int) {
	b.templates[i], b.templates[j] = b.templates[j], b.templates[i]
}

// Len returns the number of templates.
func (b byValues) Len() int {
	return len(b.templates)
}

// Less reports whether template i should sort before j by cached value.
func (b byValues) Less(i, j int) bool {
	return b.cache[b.templates[i].File].value < b.cache[b.templates[j].File].value
}

// Swap swaps templates i and j.
func (b byValues) Swap(i, j int) {
	b.templates[i], b.templates[j] = b.templates[j], b.templates[i]
}
