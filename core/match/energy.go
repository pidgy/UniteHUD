package match

import (
	"image"
	"image/color"
	"math"
	"strconv"

	"gocv.io/x/gocv"

	"github.com/pidgy/unitehud/core/config"
	"github.com/pidgy/unitehud/core/rgba"
	"github.com/pidgy/unitehud/core/state"
	"github.com/pidgy/unitehud/core/stats"
	"github.com/pidgy/unitehud/core/team"
	"github.com/pidgy/unitehud/core/template"
)

func AsAeosImage(mat gocv.Mat, points int) (image.Image, error) {
	if config.Current.Advanced.Matching.Disabled.Previews {
		return nil, nil
	}

	clone := mat.Clone()
	defer clone.Close()

	p := image.Pt(10, mat.Rows()-15)
	gocv.PutText(&clone, strconv.Itoa(points), p, gocv.FontHersheyPlain, 2, color.RGBA(rgba.Highlight), 3)

	crop, err := clone.ToImage()
	if err != nil {
		return nil, err
	}

	return crop, nil
}

// Energy avoids the walking inset method in previous versions which fails for duplicate values.
// Instead, Energy handles duplicate values by removing matched areas to avoid detection.
func Energy(matrix gocv.Mat, img image.Image) (Result, []int, int) {
	mins := []int{math.MaxInt32, math.MaxInt32}
	maxs := []float32{0, 0}
	points := []int{-1, -1}
	matched := []image.Rectangle{{}, {}}

	templates := config.Current.TemplatesPoints(team.Energy.Name)

	region := matrix.Clone()

	for round := 0; round < len(points); round++ {
		results := make([]gocv.Mat, len(templates))

		for i, template := range templates {
			if template.Mat.Cols() > region.Cols() || template.Mat.Rows() > region.Rows() {
				return Invalid, points, -1
			}

			mat := gocv.NewMat()
			defer mat.Close()

			results[i] = mat

			gocv.MatchTemplate(region, template.Mat, &mat, gocv.TmCcoeffNormed, template.Mask)
		}

		for i := range results {
			if results[i].Empty() {
				continue
			}

			_, maxv, _, maxp := gocv.MinMaxLoc(results[i])
			if math.IsInf(float64(maxv), 1) {
				continue
			}

			if maxv < team.Energy.Acceptance {
				continue
			}

			go stats.Collect(templates[i].Truncated(), maxv)

			// No sorting comparison exists yet, proceed.
			if mins[round] == 0 {
				break
			}

			// Keep the leftmost value, always.
			if maxp.X > mins[round] {
				continue
			}
			// Keep the best match for locations.
			if maxp.X == mins[round] && maxv < maxs[round] {
				continue
			}

			// First round we care more about the smallest X value, 0 is trumped.
			// Second round we care more about the highest acceptance value.
			switch round {
			case 0:
			case 1:
				if maxv < maxs[round] {
					continue
				}
			}

			// Once were here we should have the smallest X value (leftmost).
			maxs[round] = maxv
			mins[round] = maxp.X
			points[round] = templates[i].Value
			matched[round] = image.Rect(maxp.X, maxp.Y, maxp.X+templates[i].Cols(), maxp.Y+templates[i].Rows())
		}

		if points[round] == -1 {
			continue
		}

		gocv.Rectangle(&region, matched[round], rgba.Black.Color(), -1)
	}

	switch {
	case points[0] == -1 && points[1] == 0:
		return Found, points, points[1]
	case points[0]+points[1] == -2: // Zero digits.
		return NotFound, points, -1
	case points[0]+points[1] == -1: // Single digit, can only be zero.
		return Found, points, 0
	case points[1] < 0: // Single digit only found at index 0.
		return Found, points, points[0]
	default: // Double digits found at index 0, and 1.
		return Found, points, points[0]*10 + points[1]
	}
}

func SelfScore(matrix gocv.Mat, img image.Image) (*Match, Result) {
	templates := []*template.Template{}
	for _, t := range config.Current.TemplatesScoring(team.Game.Name) {
		if state.EventType(t.Value) == state.PreScore || state.EventType(t.Value) == state.PostScore {
			templates = append(templates, t)
		}
	}
	m, r, _ := Matches(matrix, img, templates)
	return m, r
}

func SelfScored(matrix gocv.Mat, img image.Image) (*Match, Result) {
	templates := []*template.Template{}
	for _, t := range config.Current.TemplatesScoring(team.Game.Name) {
		if state.EventType(t.Value) == state.PostScore {
			templates = append(templates, t)
		}
	}
	m, r, _ := Matches(matrix, img, templates)
	return m, r
}

func SelfScoring(matrix gocv.Mat, img image.Image) (*Match, Result) {
	templates := []*template.Template{}
	for _, t := range config.Current.TemplatesScoring(team.Game.Name) {
		e := state.EventType(t.Value)
		if e == state.PreScore || e == state.PressButtonToScore {
			templates = append(templates, t)
		}
	}
	m, r, _ := Matches(matrix, img, templates)
	return m, r
}

func SelfScoreOption(matrix gocv.Mat, img image.Image) (*Match, Result) {
	templates := []*template.Template{}
	for _, t := range config.Current.TemplatesScoring(team.Game.Name) {
		if state.EventType(t.Value) == state.PressButtonToScore {
			templates = append(templates, t)
		}
	}
	m, r, _ := MatchesWithAcceptance(matrix, img, templates, .85)
	return m, r
}
