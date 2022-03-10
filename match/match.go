package match

import (
	"image"
	"image/color"
	"strconv"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gocv.io/x/gocv"

	"github.com/pidgy/unitehud/config"
	"github.com/pidgy/unitehud/dev"
	"github.com/pidgy/unitehud/notify"
	"github.com/pidgy/unitehud/rgba"
	"github.com/pidgy/unitehud/server"
	"github.com/pidgy/unitehud/team"
	"github.com/pidgy/unitehud/template"
)

type Match struct {
	image.Point
	template.Template
	Max image.Point
}

type Result int

const (
	Duplicate Result = -3
	Invalid   Result = -2
	Missed    Result = -1
	NotFound  Result = 0
	Found     Result = 1
)

var (
	mask = gocv.NewMat()
)

func (m *Match) Matches(matrix gocv.Mat, img image.Image, t []template.Template) (Result, int) {
	results := make([]gocv.Mat, len(t))

	m.Max = img.Bounds().Max

	for i, template := range t {
		results[i] = gocv.NewMat()
		defer results[i].Close()

		gocv.MatchTemplate(matrix, template.Mat, &results[i], gocv.TmCcoeffNormed, template.Mask)
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

			return m.process(matrix, img)
		}
	}

	return NotFound, 0
}

func (m *Match) Identify(mat gocv.Mat, points int) (image.Image, error) {
	clone := mat.Clone()
	defer clone.Close()

	gocv.Rectangle(&clone, m.Rectangle(), color.RGBA(rgba.Highlight), 2)

	region := clone.Region(m.Team.Crop(m.Point))

	p := image.Pt(10, region.Rows()-15)
	gocv.PutText(&region, strconv.Itoa(points), p, gocv.FontHersheyPlain, 2, color.RGBA(rgba.Highlight), 3)

	crop, err := region.ToImage()
	if err != nil {
		log.Error().Err(err).Msg("failed to convert image")
		return nil, err
	}

	return crop, nil
}

func (m *Match) Rectangle() image.Rectangle {
	return image.Rect(m.Point.X, m.Point.Y, m.Point.X+m.Template.Mat.Cols(), m.Point.Y+m.Template.Mat.Rows())
}

func (m *Match) process(matrix gocv.Mat, img image.Image) (Result, int) {
	log.Debug().Object("match", m).Int("cols", matrix.Cols()).Int("rows", matrix.Rows()).Msg("processing match")

	switch m.Template.Category {
	case "scored":
		crop := m.Team.Crop(m.Point)
		if crop.Min.X < 0 || crop.Min.Y < 0 || crop.Max.X > matrix.Cols() || crop.Max.Y > matrix.Rows() {
			log.Error().Object("match", m).Msg("cropped image is outside the legal selection")
			return Invalid, 0
		}

		return m.points(matrix.Region(crop))
	case "game":
		switch m.Template.Subcategory {
		case "vs":
			server.Clear()

			notify.Feed(team.Self.RGBA, "Match starting")

			if config.Current.Record || config.Current.RecordMissed || config.Current.RecordDuplicates {
				dev.Start()
			}

			return Found, 0
		case "end":
			p, o, s := server.Scores()
			if p+o+s > 0 {
				notify.Feed(team.Self.RGBA, "Match ended")
				notify.Feed(team.Self.RGBA, "Purple Score: %d", p)
				notify.Feed(team.Self.RGBA, "Orange Score: %d", o)
				notify.Feed(team.Self.RGBA, "Self Score: %d", s)
			}

			server.Clear()
			team.Clear()

			if config.Current.Record || config.Current.RecordMissed || config.Current.RecordDuplicates {
				dev.End()
			}

			return Found, 0
		}
	}

	return NotFound, 0
}

func (m *Match) MarshalZerologObject(e *zerolog.Event) {
	e.Object("template", m.Template).Stringer("point", m.Point).Object("duplicate", m.Duplicate).Object("team", m.Team)
}

func (r Result) String() string {
	switch r {
	case Duplicate:
		return "duplicate"
	case Invalid:
		return "invalid"
	case Missed:
		return "missed"
	case NotFound:
		return "not found"
	case Found:
		return "found"
	}
	return "unknown"
}
