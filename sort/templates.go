package sort

import (
	"image"
	"math"
	"sort"

	"github.com/pidgy/unitehud/filter"
	"github.com/pidgy/unitehud/template"
)

// Templates represents a sortable set of unique templates where set len > 1.
type Templates struct {
	templates []template.Template
	cache     map[string]cached
	invalid   bool
}

type cached struct {
	image.Point
	value float32
	seen  int
}

func NewTemplates() Templates {
	return Templates{
		cache: map[string]cached{},
	}
}

func (t *Templates) Cache(t2 template.Template, p image.Point, value float32) {
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

func ByLocation(t Templates) bool {
	if t.invalid || len(t.cache) == 0 || len(t.cache) > 3 {
		return false
	}

	sort.Sort(byLocation(t))

	t.invalid = t.Value() < 1 || t.Value() > 100

	return !t.invalid
}

func ByValues(t Templates) bool {
	for _, c := range t.cache {
		if c.seen > 1 {
			return false
		}
	}
	sort.Sort(byValues(t))

	return t.Value() > 0 && t.Value() < 100
}

func (t *Templates) Value() int {
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

type byLocation Templates

func (b byLocation) Len() int {
	return len(b.templates)
}

func (b byLocation) Less(i, j int) bool {
	return b.cache[b.templates[i].File].X < b.cache[b.templates[j].File].X
}

func (b byLocation) Swap(i, j int) {
	b.templates[i], b.templates[j] = b.templates[j], b.templates[i]
}

type byValues Templates

func (b byValues) Len() int {
	return len(b.templates)
}

func (b byValues) Less(i, j int) bool {
	return b.cache[b.templates[i].File].value < b.cache[b.templates[j].File].value
}

func (b byValues) Swap(i, j int) {
	b.templates[i], b.templates[j] = b.templates[j], b.templates[i]
}
