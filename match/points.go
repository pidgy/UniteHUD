package match

import (
	"image"
	"math"

	"github.com/rs/zerolog/log"
	"gocv.io/x/gocv"

	"github.com/pidgy/unitehud/config"
	"github.com/pidgy/unitehud/duplicate"
	"github.com/pidgy/unitehud/sort"
	"github.com/pidgy/unitehud/stats"
	"github.com/pidgy/unitehud/team"
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

	mins := []int{math.MaxInt32, math.MaxInt32, math.MaxInt32}
	points := []int{-1, -1}

	templates := config.Current.Templates["points"][m.Team.Name]

	// Collect matched templates, exit early if we detect different images once.
	sorted := sort.NewTemplates()

	for round := 0; round < len(points); round++ {
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
				log.Warn().Str("filename", templates[i].File).Msg("empty result")
				continue
			}

			_, maxv, _, maxp := gocv.MinMaxLoc(results[i])
			if !math.IsInf(float64(maxv), 1) && maxv >= m.Team.Acceptance {
				sorted.Cache(templates[i], maxp, maxv)

				go stats.Average(templates[i].File, maxv)
				go stats.Count(templates[i].File)

				if maxp.X < mins[round] {
					mins[round] = maxp.X
					points[round] = templates[i].Value
				}
			}

			go stats.Frequency(templates[i].File, maxv)
		}

		// If the first round of matching justifies quick sorting, we can exit early.
		if sort.ByLocation(sorted) || sort.ByValues(sorted) {
			return m.validate(matrix, sorted.Value())
		}

		inset += mins[round] + 15
		if inset > matrix.Cols() {
			break
		}
	}

	r, p := pointSlice(points)
	if r != Found {
		return r, p
	}

	return m.validate(matrix, p)
}

func (m *Match) regular(matrix gocv.Mat) (Result, int) {
	inset := 0

	mins := []int{math.MaxInt32, math.MaxInt32, math.MaxInt32}
	points := []int{-1, -1, -1}

	templates := config.Current.Templates["points"][m.Team.Name]

	for round := 0; round < len(mins); round++ {
		region := matrix.Region(
			image.Rectangle{
				Min: image.Pt(inset, 0),
				Max: image.Pt(matrix.Cols(), matrix.Rows()),
			},
		)

		// gocv.IMWrite(fmt.Sprintf("region-%d.png", round), region)

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
				log.Warn().Str("filename", templates[i].File).Msg("empty result")
				continue
			}

			_, maxv, _, maxp := gocv.MinMaxLoc(results[i])
			if maxv >= m.Team.Acceptance {
				// fmt.Printf("#%d, %d%% %s, %s %d\n", round, int(maxv*100), templates[i].File, maxp, templates[i].Cols())

				if round > 0 && maxp.X > templates[i].Mat.Cols() {
					maxp.X = 0
				}

				go stats.Average(templates[i].File, maxv)
				go stats.Count(templates[i].File)

				if maxp.X < mins[round] {
					mins[round] = maxp.X + templates[i].Mat.Cols() - 1
					points[round] = templates[i].Value
				}
			}
		}

		inset += mins[round]
		if inset > matrix.Cols() {
			break
		}
	}

	r, p := pointSlice(points)
	if r != Found {
		return r, p
	}

	return m.validate(matrix, p)
}

func (m *Match) validate(matrix gocv.Mat, value int) (Result, int) {
	if value < 1 || value > 100 {
		return Missed, value
	}

	latest := duplicate.New(value, matrix, m.Team.Comparable(matrix))
	defer func() {
		if latest.Counted {
			m.Team.Duplicate = latest
		}
	}()

	dup := m.Team.Duplicate.Of(latest)
	if dup {
		return Duplicate, value
	}

	latest.Counted = true

	return Found, value
}

func pointSlice(points []int) (Result, int) {
	if len(points) == 2 {
		points = append(points, -1)
	}

	switch {
	case points[0]+points[1]+points[2] == -3:
		// Zero digits.
		return NotFound, 0
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
