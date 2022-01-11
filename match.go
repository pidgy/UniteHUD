package main

import (
	"fmt"
	"image"
	"time"

	"github.com/rs/zerolog/log"
	"gocv.io/x/gocv"

	"github.com/pidgy/unitehud/dev"
	"github.com/pidgy/unitehud/duplicate"
	"github.com/pidgy/unitehud/team"
	"github.com/pidgy/unitehud/window"
)

type match struct {
	image.Point
	template
}

func (m match) process(matrix gocv.Mat, img *image.RGBA) {
	log.Info().Object("match", m).Int("cols", matrix.Cols()).Int("rows", matrix.Rows()).Msg("match found")

	switch m.category {
	case "scored":
		m.points(matrix.Region(m.Team.Rectangle(m.Point)), img)
	case "game":
		switch m.subcategory {
		case "vs":
			socket.Clear()

			window.Write(window.Default, "Match starting")

			if record {
				dev.Start()
			}
		case "end":
			socket.Clear()

			window.Write(window.Default, "Match ended")

			if record {
				dev.End()
			}
		}
	}
}

func (m match) points(matrix2 gocv.Mat, img *image.RGBA) {
	results := make([]gocv.Mat, len(templates["points"][m.Team.Name]))

	for i, pt := range templates["points"][m.Team.Name] {
		mat := gocv.NewMat()
		defer mat.Close()

		results[i] = mat

		gocv.MatchTemplate(matrix2, pt.Mat, &mat, gocv.TmCcoeffNormed, mask)
	}

	pieces := pieces([]piece{})

	for i := range results {
		if results[i].Empty() {
			log.Warn().Str("filename", m.file).Msg("empty result")
			continue
		}

		_, maxc, _, maxp := gocv.MinMaxLoc(results[i])
		if maxc >= acceptance {
			pieces = append(pieces,
				piece{
					maxp,
					templates["points"][m.Team.Name][i].filter,
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
	if !dup && m.Team.Name == team.Self.Name && time.Now().Sub(m.Team.Duplicate.Time) < time.Second*3 {
		dup = true
	}

	if dup {
		log.Warn().Object("latest", latest).Object("match", m).Msg("duplicate match")
	}

	m.Team.Duplicate.Close()
	m.Team.Duplicate = latest

	if !dup && value > 0 {
		go socket.Publish(m.Team, value)
	}

	if record {
		dev.Capture(img, matrix2, m.Team.Name, order, dup, value)
		dev.Log(fmt.Sprintf("%s %d (duplicate: %t)", m.Team.Name, value, dup))
	}
}

func (m match) time(matrix gocv.Mat, img *image.RGBA) {
	clock := []int{-1, -1, -1, -1}

	hands := []image.Rectangle{
		image.Rect(35, 15, 60, 50),
		image.Rect(55, 15, 75, 50),
		image.Rect(80, 15, 100, 50),
		image.Rect(95, 15, 120, 50),
	}

	for i := range clock {
		mat := gocv.NewMat()
		defer mat.Close()

		results := make([]gocv.Mat, len(templates["time"][team.Time.Name]))

		for j, template := range templates["time"][team.Time.Name] {
			mat := gocv.NewMat()
			defer mat.Close()

			results[j] = mat

			gocv.MatchTemplate(matrix.Region(hands[i]), template.Mat, &mat, gocv.TmCcoeffNormed, mask)
		}

		for j := range results {
			if results[j].Empty() {
				log.Warn().Str("filename", m.file).Msg("empty result")
				continue
			}

			_, maxc, _, _ := gocv.MinMaxLoc(results[j])
			if maxc >= .9 {
				clock[i] = templates["time"][team.Time.Name][j].value
			}
		}
	}

	for i := range clock {
		if clock[i] == -1 {
			return
		}
	}

	mins := clock[0]*10 + clock[1]
	secs := clock[2]*10 + clock[3]

	socket.Time(mins, secs)
}
