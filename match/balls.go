package match

import (
	"image"
	"image/color"
	"math"
	"strconv"
	"time"

	"github.com/rs/zerolog/log"
	"gocv.io/x/gocv"

	"github.com/pidgy/unitehud/config"
	"github.com/pidgy/unitehud/rgba"
	"github.com/pidgy/unitehud/state"
	"github.com/pidgy/unitehud/stats"
	"github.com/pidgy/unitehud/team"
)

func Balls(matrix gocv.Mat, img image.Image) (Result, []int, int) {
	inset := 0

	mins := []int{math.MaxInt32, math.MaxInt32}
	maxs := []float32{0, 0}
	points := []int{-1, -1}

	templates := config.Current.Templates["points"][team.Balls.Name]

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
				return Invalid, points, -1
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
			if math.IsInf(float64(maxv), 1) {
				continue
			}

			go stats.Frequency(templates[i].File, maxv)

			if maxv >= team.Balls.Acceptance {
				go stats.Average(templates[i].File, maxv)
				go stats.Count(templates[i].File)

				// In the first round we care more about the smallest X value, 0 is trumped.
				// In the second round we care more about the highest acceptance value.

				switch round {
				case 0:
					if maxp.X > mins[round] || maxs[round] > maxv {
						continue
					}
					if templates[i].Value == 0 && maxs[round] > maxv {
						continue
					}
				case 1:
					if maxs[round] > maxv {
						continue
					}
				}

				maxs[round] = maxv
				mins[round] = maxp.X + templates[i].Cols() - 5
				points[round] = templates[i].Value
			}

			go stats.Frequency(templates[i].File, maxv)
		}

		if points[round] == -1 {
			continue
		}

		inset += mins[round]
		if inset > config.Current.Balls.Size().X {
			break
		}
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

// Balls2 avoids the walking inset method in Balls that fails for duplicate number values.
// Instead, it handles duplicate number values by drawing over matched areas.
func Balls2(matrix gocv.Mat, img image.Image) (Result, []int, int) {
	mins := []int{math.MaxInt32, math.MaxInt32}
	maxs := []float32{0, 0}
	points := []int{-1, -1}
	matched := []image.Rectangle{{}, {}}

	templates := config.Current.Templates["points"][team.Balls.Name]
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
				log.Warn().Str("filename", templates[i].File).Msg("empty result")
				continue
			}

			_, maxv, _, maxp := gocv.MinMaxLoc(results[i])
			if math.IsInf(float64(maxv), 1) {
				continue
			}

			go stats.Frequency(templates[i].File, maxv)

			if maxv >= team.Balls.Acceptance {
				go stats.Average(templates[i].File, maxv)
				go stats.Count(templates[i].File)

				// In the first round we care more about the smallest X value, 0 is trumped.
				// In the second round we care more about the highest acceptance value.

				switch round {
				case 0:
					if maxp.X > mins[round] || maxs[round] > maxv {
						continue
					}
					if templates[i].Value == 0 && maxs[round] > maxv {
						continue
					}
				case 1:
					if maxs[round] > maxv {
						continue
					}
				}

				maxs[round] = maxv
				mins[round] = maxp.X + templates[i].Cols() - 5
				points[round] = templates[i].Value
				matched[round] = image.Rect(maxp.X, maxp.Y, maxp.X+templates[i].Cols(), maxp.Y+templates[i].Rows())
			}

			go stats.Frequency(templates[i].File, maxv)
		}

		if points[round] == -1 {
			continue
		}

		gocv.Rectangle(&region, matched[round], rgba.Black, -1)
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

func IdentifyBalls(mat gocv.Mat, points int) (image.Image, error) {
	clone := mat.Clone()
	defer clone.Close()

	p := image.Pt(10, mat.Rows()-15)
	gocv.PutText(&clone, strconv.Itoa(points), p, gocv.FontHersheyPlain, 2, color.RGBA(rgba.Highlight), 3)

	crop, err := clone.ToImage()
	if err != nil {
		log.Error().Err(err).Msg("failed to convert image")
		return nil, err
	}

	return crop, nil
}

func SelfScore(matrix gocv.Mat, img image.Image) (Result, int) {
	_, r, n := Matches(matrix, img, config.Current.Templates["scoring"][team.Game.Name])
	if r != Found {
		return r, 0
	}

	// Verify the same event has not occured.
	past := state.Past(state.PostScore, time.Second*3)
	if len(past) > 2 {
		n = 0
		for _, event := range past[1:] {
			n -= event.Value
		}
		return Invalid, n
	}

	return Found, n
}
