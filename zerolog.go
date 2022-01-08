package main

import (
	"strconv"

	"github.com/rs/zerolog"
)

func (f filter) MarshalZerologObject(e *zerolog.Event) {
	e.Str("file", f.file).Str("team", f.Team.Name)
}

func (m match) MarshalZerologObject(e *zerolog.Event) {
	e.Object("template", m.template).Stringer("point", m.Point).Object("duplicate", m.Duplicate)
}

func (p piece) MarshalZerologObject(e *zerolog.Event) {
	e.Stringer("point", p.Point).Str("file", p.file).Int("value", p.value)
}

func (p pieces) MarshalZerologObject(e *zerolog.Event) {
	e.Int("len", len(p))

	for i, piece := range p {
		e.Object(strconv.Itoa(i), piece)
	}
}

func (t template) MarshalZerologObject(e *zerolog.Event) {
	e.Object("filter", t.filter).
		Int("mrows", t.Mat.Rows()).
		Int("mcols", t.Mat.Cols()).
		Float64("scalar", t.scalar)
}
