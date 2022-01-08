package main

import (
	"image"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
)

type piece struct {
	image.Point
	filter
}

func (p piece) Eq(p2 piece) bool {
	f := strings.ReplaceAll(strings.ReplaceAll(p.file, "_alt", ""), "_big", "")
	f2 := strings.ReplaceAll(strings.ReplaceAll(p2.file, "_alt", ""), "_big", "")
	if f != f2 {
		return false
	}

	return math.Abs(float64(p.X-p2.X)) < 6
}

type pieces []piece

func (p pieces) Len() int {
	return len(p)
}

func (p pieces) Less(i, j int) bool {
	return p[i].X < p[j].X
}

// Swap swaps the elements with indexes i and j.
func (p pieces) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

func (p pieces) Int() (int, string) {
	if len(p) == 0 {
		return 0, ""
	}

	sort.Sort(p)

	unique := pieces{}
	removed := pieces{}

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
		order += strconv.Itoa(piece.value)
	}

	log.Info().Object("pieces", p).Str("order", order).Object("removed", removed).Msg("sorted")

	v, err := strconv.Atoi(order)
	if err != nil {
		log.Warn().Err(err).Object("pieces", p).Msg("failed to convert sortable pieces to an integer")
	}

	return v, order
}
