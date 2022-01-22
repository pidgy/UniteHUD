package match

import (
	"fmt"
	"image"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gocv.io/x/gocv"

	"github.com/pidgy/unitehud/config"
	"github.com/pidgy/unitehud/dev"
	"github.com/pidgy/unitehud/duplicate"
	"github.com/pidgy/unitehud/pipe"
	"github.com/pidgy/unitehud/sort"
	"github.com/pidgy/unitehud/team"
	"github.com/pidgy/unitehud/template"
	"github.com/pidgy/unitehud/window/terminal"
)

type Match struct {
	image.Point
	template.Template
}

var (
	mask = gocv.NewMat()
)

func (m Match) Matches(matrix gocv.Mat, img image.Image, t []template.Template) bool {
	results := make([]gocv.Mat, len(t))

	for i, template := range t {
		results[i] = gocv.NewMat()
		defer results[i].Close()

		gocv.MatchTemplate(matrix, template.Mat, &results[i], gocv.TmCcoeffNormed, mask)
	}

	for i, mat := range results {
		if mat.Empty() {
			log.Warn().Str("filename", t[i].File).Msg("empty result")
			continue
		}

		_, maxc, _, maxp := gocv.MinMaxLoc(mat)
		if maxc >= config.Current.Acceptance {
			m.Template = t[i]
			m.Point = maxp

			return m.Process(matrix, img)
		}
	}

	return false
}

func (m Match) Process(matrix gocv.Mat, img image.Image) bool {
	log.Info().Object("match", m).Int("cols", matrix.Cols()).Int("rows", matrix.Rows()).Msg("match found")

	switch m.Category {
	case "scored":
		rect := m.Team.Rectangle(m.Point)
		if rect.Min.X < 0 || rect.Min.Y < 0 || rect.Max.X > matrix.Cols() || rect.Max.Y > matrix.Rows() {
			log.Warn().Object("match", m).Msg("match is outside the legal selection")
			return false
		}

		return m.Points(matrix.Region(rect), img)
	case "game":
		switch m.Subcategory {
		case "vs":
			pipe.Socket.Clear()

			terminal.Write(terminal.White, "Match starting")

			if config.Current.Record {
				dev.Start()
			}

			return true
		case "end":
			pipe.Socket.Clear()

			terminal.Write(terminal.White, "Match ended")

			if config.Current.Record {
				dev.End()
			}

			return true
		}
	}

	return false
}

func (m Match) Points(matrix2 gocv.Mat, img image.Image) bool {
	results := make([]gocv.Mat, len(config.Current.Templates["points"][m.Team.Name]))

	for i, pt := range config.Current.Templates["points"][m.Team.Name] {
		mat := gocv.NewMat()
		defer mat.Close()

		results[i] = mat

		gocv.MatchTemplate(matrix2, pt.Mat, &mat, gocv.TmCcoeffNormed, mask)
	}

	pieces := sort.Pieces([]sort.Piece{})

	for i := range results {
		if results[i].Empty() {
			log.Warn().Str("filename", m.File).Msg("empty result")
			continue
		}

		_, maxc, _, maxp := gocv.MinMaxLoc(results[i])
		if maxc >= config.Current.Acceptance {
			pieces = append(pieces,
				sort.Piece{
					Point:  maxp,
					Filter: config.Current.Templates["points"][m.Team.Name][i].Filter,
				},
			)
		}
	}

	value, order := pieces.Int()
	if value == 0 {
		log.Warn().Object("team", m.Team).Str("order", order).Msg("no value extracted")
	}

	region := m.Team.Region(matrix2)

	latest := duplicate.New(value, matrix2, region)

	dup := m.Team.Duplicate.Of(latest)
	//TODO simplify this
	if !dup && m.Team.Name == team.Self.Name && time.Since(m.Team.Duplicate.Time) < time.Second {
		dup = true
	}

	if dup {
		log.Warn().Object("latest", latest).Object("match", m).Msg("duplicate match")
	}

	m.Team.Duplicate.Close()
	m.Team.Duplicate = latest

	if !dup && value > 0 {
		go pipe.Socket.Publish(m.Team, value)
	}

	if config.Current.Record {
		dev.Capture(img, matrix2, m.Team.Name, order, dup, value)
		dev.Log(fmt.Sprintf("%s %d (duplicate: %t)", m.Team.Name, value, dup))
	}

	return value > 0
}

func (m Match) Time(matrix gocv.Mat, img *image.RGBA, hands [4]image.Rectangle) int {
	clock := [4]int{-1, -1, -1, -1}

	for i := range clock {
		mat := gocv.NewMat()
		defer mat.Close()

		results := make([]gocv.Mat, len(config.Current.Templates["time"][team.Time.Name]))

		for j, template := range config.Current.Templates["time"][team.Time.Name] {

			mat := gocv.NewMat()
			defer mat.Close()

			results[j] = mat

			gocv.MatchTemplate(matrix.Region(hands[i]), template.Mat, &mat, gocv.TmCcoeffNormed, mask)
		}

		for j := range results {
			if results[j].Empty() {
				log.Warn().Str("filename", m.File).Msg("empty result")
				continue
			}

			_, maxc, _, _ := gocv.MinMaxLoc(results[j])
			if maxc >= .9 {
				clock[i] = config.Current.Templates["time"][team.Time.Name][j].Value
			}
		}
	}

	for i := range clock {
		if clock[i] == -1 {
			return 0
		}
	}

	mins := clock[0]*10 + clock[1]
	secs := clock[2]*10 + clock[3]

	pipe.Socket.Time(mins, secs)

	return mins*60 + secs
}

func (m Match) MarshalZerologObject(e *zerolog.Event) {
	e.Object("template", m.Template).Stringer("point", m.Point).Object("duplicate", m.Duplicate)
}
