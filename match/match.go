package match

import (
	"image"
	"image/color"
	"strconv"

	"gocv.io/x/gocv"

	"github.com/pidgy/unitehud/config"
	"github.com/pidgy/unitehud/notify"
	"github.com/pidgy/unitehud/rgba"
	"github.com/pidgy/unitehud/state"
	"github.com/pidgy/unitehud/stats"
	"github.com/pidgy/unitehud/team"
	"github.com/pidgy/unitehud/template"
)

type Match struct {
	image.Point
	*template.Template
	Max      image.Point
	Accepted float32

	Points []image.Point
}

const (
	Duplicate Result = -3
	Invalid   Result = -2
	Missed    Result = -1
	NotFound  Result = 0
	Found     Result = 1
	Override  Result = 2
)

func (m *Match) AsImage(mat gocv.Mat, points int) (image.Image, error) {
	if config.Current.DisablePreviews {
		return nil, nil
	}

	clone := mat.Clone()
	defer clone.Close()

	gocv.Rectangle(&clone, m.rectangle(), color.RGBA(rgba.Highlight), 2)

	region := clone.Region(m.Team.Crop(m.Point))

	gocv.PutText(
		&region,
		strconv.Itoa(points),
		image.Pt(region.Cols()/3*2-25, region.Rows()/2+7),
		gocv.FontHersheySimplex,
		1,
		rgba.White.Color(),
		4,
	)

	crop, err := region.ToImage()
	if err != nil {

		return nil, err
	}

	return crop, nil
}

func Matches(matrix gocv.Mat, img image.Image, templates []*template.Template) (*Match, Result, int) {
	return MatchesWithAcceptance(matrix, img, templates, config.Current.Acceptance)
}

func MatchesWithAcceptance(matrix gocv.Mat, img image.Image, templates []*template.Template, acceptance float32) (*Match, Result, int) {
	results := make([]gocv.Mat, len(templates))

	m := &Match{
		Max: img.Bounds().Max,
	}

	for i, template := range templates {
		results[i] = gocv.NewMat()
		defer results[i].Close()

		if template.Mat.Rows() > matrix.Rows() || template.Mat.Cols() > matrix.Cols() {
			notify.Error("Match is outside the configured selection area")

			if config.Current.Record {
				// dev.Capture(img, region, team.Time.Name, "missed-"+template.Name, false, template.Value)
			}

			continue
		}

		gocv.MatchTemplate(matrix, template.Mat, &results[i], gocv.TmCcoeffNormed, template.Mask)
	}

	for i, mat := range results {
		if mat.Empty() {
			notify.SystemWarn("Empty result for %s", templates[i].Truncated())

			continue
		}

		_, maxv, _, maxp := gocv.MinMaxLoc(mat)

		go stats.Frequency(templates[i].Truncated(), maxv)

		if maxv >= acceptance {
			m.Template = templates[i]
			m.Point = maxp
			m.Accepted = maxv

			go stats.Average(m.Template.Truncated(), maxv)
			go stats.Count(m.Template.Truncated())

			r, p := m.process(matrix, img)

			return m, r, p
		}
	}

	return m, NotFound, 0
}

func (m *Match) process(matrix gocv.Mat, img image.Image) (Result, int) {
	switch m.Template.Category {
	case "killed":
		return Found, team.Energy.Holding
	case "scored": // Orange, Purple scoring.
		crop := m.Team.Crop(m.Point)
		if crop.Min.X < 0 || crop.Min.Y < 0 || crop.Max.X > matrix.Cols() || crop.Max.Y > matrix.Rows() {
			return Invalid, 0
		}

		return m.points(matrix.Region(crop))
	case "scoring": // Self scoring.
		return Found, m.Template.Value // Use team.Energy.Holding.
	case "game":
		return Found, state.EventType(m.Template.Value).Int()
	default:
		return Found, m.Template.Value
	}
}

func (m *Match) rectangle() image.Rectangle {
	return image.Rect(m.Point.X, m.Point.Y, m.Point.X+m.Template.Mat.Cols(), m.Point.Y+m.Template.Mat.Rows())
}
