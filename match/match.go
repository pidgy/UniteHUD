package match

import (
	"fmt"
	"image"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gocv.io/x/gocv"

	"github.com/pidgy/unitehud/config"
	"github.com/pidgy/unitehud/dev"
	"github.com/pidgy/unitehud/duplicate"
	"github.com/pidgy/unitehud/notify"
	"github.com/pidgy/unitehud/pipe"
	"github.com/pidgy/unitehud/rgba"
	"github.com/pidgy/unitehud/sort"
	"github.com/pidgy/unitehud/team"
	"github.com/pidgy/unitehud/template"
)

type Match struct {
	image.Point
	template.Template
}

var (
	mask = gocv.NewMat()
)

func (m *Match) Matches(matrix gocv.Mat, img image.Image, t []template.Template) (bool, int) {
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

	return false, 0
}

func (m *Match) Process(matrix gocv.Mat, img image.Image) (bool, int) {
	log.Info().Object("match", m).Int("cols", matrix.Cols()).Int("rows", matrix.Rows()).Msg("match found")

	switch m.Category {
	case "scored":
		rect := m.Team.Rectangle(m.Point)
		if rect.Min.X < 0 || rect.Min.Y < 0 || rect.Max.X > matrix.Cols() || rect.Max.Y > matrix.Rows() {
			log.Warn().Object("match", m).Msg("match is outside the legal selection")
			notify.Feed(rgba.Red, "Scored match is outside the configured selection area")
			return false, 0
		}

		return m.Points(matrix.Region(rect), img)
	case "game":
		switch m.Subcategory {
		case "vs":
			pipe.Socket.Clear()

			notify.Feed(team.Self.RGBA, "Match starting")

			if config.Current.Record || config.Current.RecordMissed || config.Current.RecordDuplicates {
				dev.Start()
			}

			return true, 0
		case "end":
			p, o, s := pipe.Socket.Score()
			if p+o+s > 0 {
				notify.Feed(team.Self.RGBA, "Match ended")
				notify.Feed(team.Self.RGBA, "Purple Score: %d", p)
				notify.Feed(team.Self.RGBA, "Orange Score: %d", o)
				notify.Feed(team.Self.RGBA, "Self Score:   %d", s)
			}

			pipe.Socket.Clear()

			if config.Current.Record || config.Current.RecordMissed || config.Current.RecordDuplicates {
				dev.End()
			}

			return true, 0
		}
	}

	return false, 0
}

func (m *Match) Points(matrix gocv.Mat, img image.Image) (bool, int) {
	results := make([]gocv.Mat, len(config.Current.Templates["points"][m.Team.Name]))

	for i, pt := range config.Current.Templates["points"][m.Team.Name] {
		mat := gocv.NewMat()
		defer mat.Close()

		results[i] = mat

		gocv.MatchTemplate(matrix, pt.Mat, &mat, gocv.TmCcoeffNormed, mask)
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
					Point:    maxp,
					Template: config.Current.Templates["points"][m.Team.Name][i],
				},
			)
		}
	}

	value, order := pieces.Sort()
	if value == 0 {
		log.Warn().Object("team", m.Team).Str("order", order).Msg("no value extracted")

		if config.Current.RecordMissed {
			dev.Capture(img, matrix, m.Team.Name, "missed", false, value)
		}

		notify.Feed(rgba.Red, "[%s] Potential score miss for %s", pipe.Socket.Clock(), strings.Title(m.Template.Team.Name))

		return false, -1
	}

	region := m.Team.Region(matrix)

	latest := duplicate.New(value, matrix, region)

	dup := m.Team.Duplicate.Of(latest)
	//TODO simplify this
	if !dup && m.Team.Name == team.Self.Name && time.Since(m.Team.Duplicate.Time) < time.Second {
		dup = true
	}

	if dup {
		log.Warn().Object("latest", latest).Object("previous", m.Duplicate).Msg("duplicate match")

		if latest.Value > m.Team.Duplicate.Value {
			if !m.Team.Duplicate.Counted {
				notify.Feed(m.Team.RGBA, "Ignorning previous duplicate score %d", m.Team.Duplicate.Value)
			} else {
				log.Warn().Object("latest", latest).Object("match", m).Msg("overwriting previous value")
				notify.Feed(m.Team.RGBA, "[%s] -%d", pipe.Socket.Clock(), m.Team.Duplicate.Value)
				notify.Feed(m.Team.RGBA, "[%s] +%d", pipe.Socket.Clock(), value)

				pipe.Socket.Publish(m.Team, latest.Value)
				pipe.Socket.Publish(m.Team, -m.Team.Duplicate.Value)
			}
		}
	}

	m.Team.Duplicate.Close()
	m.Team.Duplicate = latest

	if !dup && value > 0 {
		go pipe.Socket.Publish(m.Team, value)
		m.Team.Duplicate.Counted = true
		notify.Feed(m.Team.RGBA, "[%s] +%d", pipe.Socket.Clock(), value)
	}

	if config.Current.Record {
		dev.Capture(img, matrix, m.Team.Name, order, dup, value)
		dev.Log(fmt.Sprintf("%s %d (duplicate: %t)", m.Team.Name, value, dup))
	}

	return value > 0, value
}

func (m *Match) Time(matrix gocv.Mat, img *image.RGBA) (seconds int, kitchen string) {
	clock := [4]int{-1, -1, -1, -1}

	region := matrix

	for i := range clock {
		results := []gocv.Mat{}

		for _, template := range config.Current.Templates["time"][team.Time.Name] {
			if template.Mat.Cols() > region.Cols() || template.Mat.Rows() > region.Rows() {
				log.Warn().Str("type", "time").Msg("match is outside the legal selection")
				notify.Feed(rgba.Red, "Time match is outside the configured selection area")
				return 0, ""
			}

			mat := gocv.NewMat()
			defer mat.Close()

			results = append(results, mat)

			gocv.MatchTemplate(region, template.Mat, &mat, gocv.TmCcoeffNormed, mask)
		}

		pieces := sort.Pieces([]sort.Piece{})

		for j := range results {
			if results[j].Empty() {
				log.Warn().Str("filename", m.File).Msg("empty result")
				continue
			}

			_, maxc, _, maxp := gocv.MinMaxLoc(results[j])
			if maxc >= config.Current.Acceptance {
				if len(config.Current.Templates["time"][team.Time.Name]) <= j {
					return 0, ""
				}

				pieces = append(pieces,
					sort.Piece{
						Point:    maxp,
						Template: config.Current.Templates["time"][team.Time.Name][j],
					},
				)
			}
		}

		_, order := pieces.Sort()
		if len(order) == 0 {
			return 0, ""
		}

		clock[i] = pieces[0].Value

		// Crop the left side of the selection area via the  first <x-5,y> point matched.
		rect := image.Rect(
			pieces[0].Point.X+pieces[0].Cols()-5,
			0,
			region.Cols(),
			region.Rows(),
		)

		if rect.Min.X < 0 || rect.Min.Y < 0 || rect.Max.X > region.Cols() || rect.Max.Y > region.Rows() {
			log.Warn().Object("match", m).Msg("match is outside the legal selection")
			break
		}

		region = region.Region(rect)
	}

	mins := clock[0]*10 + clock[1]
	secs := clock[2]*10 + clock[3]

	pipe.Socket.Time(mins, secs)

	return mins*60 + secs, fmt.Sprintf("%d%d:%d%d", clock[0], clock[1], clock[2], clock[3])
}

func (m *Match) MarshalZerologObject(e *zerolog.Event) {
	e.Object("template", m.Template).Stringer("point", m.Point).Object("duplicate", m.Duplicate)
}
