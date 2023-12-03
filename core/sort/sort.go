package sort

import (
	"image"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/pidgy/unitehud/core/notify"
	"github.com/pidgy/unitehud/core/template"
)

const minDistance = 6

type Piece struct {
	image.Point
	template.Template
}

type Pieces []Piece

type StringInt struct {
	Int    int
	String string
}

type StringInts []StringInt

func (p Piece) Eq(p2 Piece) bool {
	f := strings.ReplaceAll(strings.ReplaceAll(p.File, "_alt", ""), "_big", "")
	f2 := strings.ReplaceAll(strings.ReplaceAll(p2.File, "_alt", ""), "_big", "")
	if f != f2 {
		return false
	}

	return math.Abs(float64(p.X-p2.X)) < minDistance
}

func (p Pieces) Len() int {
	return len(p)
}

func (p Pieces) Less(i, j int) bool {
	return p[i].X < p[j].X
}

func (p Pieces) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

func (p Pieces) Sort(hack bool) (int, string) {
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

	// TODO: replace/remove hack once image processing stabilizes.
	if hack {
		switch len(order) {
		case 3:
			if order == "100" {
				break
			}

			o := ""

			if order[0] == order[1] {
				o = order[1:]
			} else if order[0] == order[2] {
				o = order[:2]
			}

			v, err := strconv.Atoi(o)
			if err != nil {
				notify.SystemWarn("Failed to convert 3 sortable pieces to an integer")
			}

			return v, order
		case 4:
			if order[:2] == order[2:] {
				v, err := strconv.Atoi(order[:2])
				if err != nil {
					notify.SystemWarn("Failed to convert 4 sortable pieces to an integer")
				}

				return v, order
			}
		}
	}

	v, err := strconv.Atoi(order)
	if err != nil {
		notify.SystemWarn("Failed to convert sortable pieces to an integer")
	}

	return v, order
}

func (s StringInts) Len() int {
	return len(s)
}

func (s StringInts) Less(i, j int) bool {
	return s[i].Int > s[j].Int
}

func (s StringInts) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func StringIntMap(m map[string]int) StringInts {
	is := StringInts{}

	for s, i := range m {
		is = append(is, StringInt{Int: i, String: s})
	}

	sort.Sort(is)

	return is
}
