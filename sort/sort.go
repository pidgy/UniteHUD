package sort

import (
	"image"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/pidgy/unitehud/filter"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type Piece struct {
	image.Point
	filter.Filter
}

type Pieces []Piece

func (p Piece) Eq(p2 Piece) bool {
	f := strings.ReplaceAll(strings.ReplaceAll(p.File, "_alt", ""), "_big", "")
	f2 := strings.ReplaceAll(strings.ReplaceAll(p2.File, "_alt", ""), "_big", "")
	if f != f2 {
		return false
	}

	return math.Abs(float64(p.X-p2.X)) < 6
}

func (p Pieces) Len() int {
	return len(p)
}

func (p Pieces) Less(i, j int) bool {
	return p[i].X < p[j].X
}

// Swap swaps the elements with indexes i and j.
func (p Pieces) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

func (p Pieces) Int() (int, string) {
	if len(p) == 0 {
		return 0, ""
	}

	sort.Sort(p)

	unique := Pieces{}
	removed := Pieces{}

	for i := 0; i < len(p); i++ {
		if i+1 == len(p) {
			unique = append(unique, p[i])
			break
		}

		if p[i].Eq(p[i+1]) {
			removed = append(removed, p[i+1])
			continue
		}

		unique = append(unique, p[i])
	}

	p = unique

	order := ""
	for _, piece := range unique {
		order += strconv.Itoa(piece.Value)
	}

	log.Info().Object("pieces", p).Str("order", order).Object("removed", removed).Msg("sorted")

	v, err := strconv.Atoi(order)
	if err != nil {
		log.Warn().Err(err).Object("pieces", p).Msg("failed to convert sortable pieces to an integer")
	}

	return v, order
}

func (p Piece) MarshalZerologObject(e *zerolog.Event) {
	e.Stringer("point", p.Point).Str("file", p.File).Int("value", p.Value)
}

func (p Pieces) MarshalZerologObject(e *zerolog.Event) {
	e.Int("len", len(p))

	for i, piece := range p {
		e.Object(strconv.Itoa(i), piece)
	}
}
