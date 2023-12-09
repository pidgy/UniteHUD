package match

import (
	"image"
	"math"

	"gocv.io/x/gocv"

	"github.com/pidgy/unitehud/core/config"
	"github.com/pidgy/unitehud/core/duplicate"
	"github.com/pidgy/unitehud/core/global"
	"github.com/pidgy/unitehud/core/notify"
	"github.com/pidgy/unitehud/core/server"
	"github.com/pidgy/unitehud/core/sort"
	"github.com/pidgy/unitehud/core/stats"
	"github.com/pidgy/unitehud/core/team"
	"github.com/pidgy/unitehud/core/template"
)

func (m *Match) points(matrix gocv.Mat) (Result, int) {
	switch m.Team.Name {
	case team.Purple.Name, team.Orange.Name:
		return m.regular(matrix)
	case team.First.Name:
		team.First.Alias = team.Purple.Name
		if m.Point.X > m.Max.X/2 {
			team.First.Alias = team.Orange.Name
		}
		return m.first(matrix)
	}

	return Invalid, 0
}

func (m *Match) first(matrix gocv.Mat) (Result, int) {
	if m.Team.Duplicate.Counted {
		return Duplicate, -1
	}

	inset := 0

	points := []int{-1, -1}
	mins := []int{math.MaxInt32, math.MaxInt32, math.MaxInt32}
	maxs := []float32{0, 0, 0}
	lefts := []int{math.MaxInt32, math.MaxInt32, math.MaxInt32}

	templatesWithZero := config.Current.TemplatesPoints(m.Team.Name)
	templatesWithoutZero := []*template.Template{}
	for _, t := range templatesWithZero {
		if t.Value == 0 {
			continue
		}
		templatesWithoutZero = append(templatesWithoutZero, t)
	}

	// Collect matched templates, exit early if we detect different images once.
	sorted := sort.NewTemplates()

	for round := 0; round < len(points); round++ {
		templates := templatesWithoutZero
		if round != 0 {
			templates = templatesWithZero
		}

		region := matrix.Region(
			image.Rectangle{
				Min: image.Pt(inset, 0),
				Max: image.Pt(matrix.Cols(), matrix.Rows()),
			},
		)

		results := make([]gocv.Mat, len(templates))

		for i, template := range templates {
			if template.Mat.Cols() > region.Cols() || template.Mat.Rows() > region.Rows() {
				return Invalid, -1
			}

			mat := gocv.NewMat()
			defer mat.Close()

			results[i] = mat

			gocv.MatchTemplate(region, template.Mat, &mat, gocv.TmCcoeffNormed, template.Mask)
		}

		for i := range results {
			if results[i].Empty() {
				notify.SystemWarn("Detect: Empty result for %s", templates[i].Truncated())
				continue
			}

			_, maxv, _, maxp := gocv.MinMaxLoc(results[i])
			if math.IsInf(float64(maxv), 1) {
				continue
			}

			go stats.Frequency(templates[i].Truncated(), maxv)

			if maxv >= m.Team.Acceptance {
				sorted.Cache(templates[i], maxp, maxv)

				go stats.Average(templates[i].Truncated(), maxv)
				go stats.Count(templates[i].Truncated())

				// Select the left-most image first, when the difference is small enough,
				// use the highest template-match value to break the tie.
				leftmost := maxp.X < lefts[round]
				if delta(maxp.X, lefts[round]) < 3 {
					leftmost = maxv > maxs[round]
				}

				if leftmost {
					lefts[round] = maxp.X
					maxs[round] = maxv
					mins[round] = maxp.X
					points[round] = templates[i].Value
				}
			}
		}

		inset += mins[round] + 15
		if inset > matrix.Cols() {
			break
		}
	}

	r, p := sliceToValue(points)
	if r != Found {
		return r, p
	}

	return m.validate(matrix, p)
}

func (m *Match) regular(matrix gocv.Mat) (Result, int) {
	m.Points = []image.Point{image.Pt(0, 0), image.Pt(0, 0), image.Pt(0, 0)}

	inset := 0

	mins := []int{math.MaxInt32, math.MaxInt32}
	maxs := []float32{0, 0}
	lefts := []int{math.MaxInt32, math.MaxInt32}
	points := []int{-1, -1}

	if server.IsFinalStretch() || global.DebugMode {
		mins = []int{math.MaxInt32, math.MaxInt32, math.MaxInt32}
		maxs = []float32{0, 0, 0}
		lefts = []int{math.MaxInt32, math.MaxInt32, math.MaxInt32}
		points = []int{-1, -1, -1}
	}

	templates2ndRound := config.Current.TemplatesPoints(m.Team.Name)
	templates1stRound := config.TemplatesFirstRound(templates2ndRound)

	for round := 0; round < len(mins); round++ {
		templates := templates2ndRound
		if round == 0 {
			templates = templates1stRound
		}

		// Cut the image in half to prevent double numbers from matching the right-most first.
		max := image.Pt(matrix.Cols(), matrix.Rows())
		if round == 0 {
			max = image.Pt(matrix.Cols()/2+5, matrix.Rows())
		}

		region := matrix.Region(
			image.Rectangle{
				Min: image.Pt(inset, 0),
				Max: max,
			},
		)

		// gocv.IMWrite(fmt.Sprintf("round_%d.png", round), region)

		results := make([]gocv.Mat, len(templates))

		for i, template := range templates {
			if template.Mat.Cols() > region.Cols() || template.Mat.Rows() > region.Rows() {
				return Invalid, -1
			}

			mat := gocv.NewMat()
			defer mat.Close()

			results[i] = mat

			gocv.MatchTemplate(region, template.Mat, &mat, gocv.TmCcoeffNormed, template.Mask)
		}

		for i := range results {
			if results[i].Empty() {
				notify.SystemWarn("Detect: Empty result for %s", templates[i].Truncated())
				continue
			}

			_, maxv, _, maxp := gocv.MinMaxLoc(results[i])
			if math.IsInf(float64(maxv), 1) {
				continue
			}

			go stats.Frequency(templates[i].Truncated(), maxv)

			if maxv >= m.Team.Acceptance {
				if round > 0 && maxp.X > templates[i].Mat.Cols() {
					maxp.X = 0
				}

				// Select the left-most image first, when the difference is small enough,
				// use the highest template-match value to break the tie.
				leftmost := maxp.X < lefts[round]
				if delta(maxp.X, lefts[round]) < 5 {
					leftmost = maxv > maxs[round]
				}

				if leftmost {
					m.Points[round] = maxp
					lefts[round] = maxp.X
					maxs[round] = maxv
					mins[round] = maxp.X + templates[i].Mat.Cols() - 1
					points[round] = templates[i].Value
				}

				go stats.Average(templates[i].Truncated(), maxv)
				go stats.Count(templates[i].Truncated())
			}
		}

		inset += mins[round] - 5
		if inset > matrix.Cols() {
			break
		}
	}

	r, p := sliceToValue(points)
	if r != Found {
		return r, p
	}

	return m.validate(matrix, p)
}

func (m *Match) validate(matrix gocv.Mat, value int) (Result, int) {
	if value < 1 || value > 100 {
		return Invalid, value
	}

	latest := duplicate.New(value, matrix, m.Team.Comparable(matrix))
	defer func() {
		if latest.Counted {
			m.Team.Duplicate = latest
		}
	}()

	dup := m.Team.Duplicate.Of(latest)
	switch {
	case latest.Overrides(m.Team.Duplicate):
		latest.Counted = true
		return Override, value
	case dup:
		return Duplicate, value
	default:
		latest.Counted = true
		return Found, value
	}
}

func delta(a, b int) int {
	if a > b {
		return a - b
	}
	return b - a
}

func sliceToValue(points []int) (Result, int) {
	// Enforce a length 3 array to validate checks below.
	if len(points) == 2 {
		points = append(points, -1)
	}

	switch {
	case points[0]+points[1]+points[2] == -3:
		// Zero digits.
		return Missed, 0
	case points[1]+points[2] == -2:
		// Single digit only found at index 0.
		return Found, points[0]
	case points[2] == -1:
		// Double digits only found at index 0 and 1.
		return Found, points[0]*10 + points[1]
	default:
		// Triple digits found at index 0, 1, and 2.
		return Found, points[0]*100 + points[1]*10 + points[2]
	}
}
