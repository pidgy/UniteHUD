package template

import (
	"image"
	"math"
	"sort"

	"github.com/pidgy/unitehud/core/template/filter"
)

// Templates represents a sortable set of unique templates where set len > 1.
type (
	Sortable struct {
		templates []*Template
		cache     map[string]cached
		invalid   bool
	}

	byLocation Sortable
	byValues   Sortable

	cached struct {
		image.Point
		value float32
		seen  int
	}
)

func NewSortable() Sortable {
	return Sortable{
		cache: map[string]cached{},
	}
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

func ByLocation(t Sortable) bool {
	if t.invalid || len(t.cache) == 0 || len(t.cache) > 3 {
		return false
	}

	sort.Sort(byLocation(t))

	t.invalid = t.Value() < 1 || t.Value() > 100

	return !t.invalid
}

func ByValues(t Sortable) bool {
	for _, c := range t.cache {
		if c.seen > 1 {
			return false
		}
	}
	sort.Sort(byValues(t))

	return t.Value() > 0 && t.Value() < 100
}

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

func (b byLocation) Len() int {
	return len(b.templates)
}

func (b byLocation) Less(i, j int) bool {
	return b.cache[b.templates[i].File].X < b.cache[b.templates[j].File].X
}

func (b byLocation) Swap(i, j int) {
	b.templates[i], b.templates[j] = b.templates[j], b.templates[i]
}

func (b byValues) Len() int {
	return len(b.templates)
}

func (b byValues) Less(i, j int) bool {
	return b.cache[b.templates[i].File].value < b.cache[b.templates[j].File].value
}

func (b byValues) Swap(i, j int) {
	b.templates[i], b.templates[j] = b.templates[j], b.templates[i]
}
