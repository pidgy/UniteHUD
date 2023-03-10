package match

import (
	"fmt"
	"image"
	"math"

	"gocv.io/x/gocv"

	"github.com/pidgy/unitehud/config"
	"github.com/pidgy/unitehud/notify"
	"github.com/pidgy/unitehud/rgba"
	"github.com/pidgy/unitehud/server"
	"github.com/pidgy/unitehud/stats"
	"github.com/pidgy/unitehud/team"
)

func Time(matrix gocv.Mat, img *image.RGBA) (seconds int, kitchen string) {
	clock := [4]int{-1, -1, -1, -1}
	locs := []int{math.MaxInt32, math.MaxInt32, math.MaxInt32, math.MaxInt32}
	cols := []int{0, 0, 0, 0}
	templates := config.Current.Templates["time"][team.Time.Name]

	inset := 0

	for c := range clock {
		region := matrix.Region(
			image.Rectangle{
				Min: image.Pt(inset, 0),
				Max: image.Pt(matrix.Cols(), matrix.Rows()),
			},
		)

		results := []gocv.Mat{}

		for _, template := range templates {
			if template.Mat.Cols() > region.Cols() || template.Mat.Rows() > region.Rows() {
				notify.Error("Time match is outside the configured selection area")

				if config.Current.Record {
					// dev.Capture(img, region, team.Time.Name, "missed-"+template.Name, false, template.Value)
				}

				return 0, ""
			}

			mat := gocv.NewMat()
			defer mat.Close()

			results = append(results, mat)

			gocv.MatchTemplate(region, template.Mat, &mat, gocv.TmCcoeffNormed, mask)
		}

		for i := range results {
			if results[i].Empty() {
				notify.SystemWarn("Empty result for %s", templates[i].Truncated())
				continue
			}

			_, maxv, _, maxp := gocv.MinMaxLoc(results[i])
			if math.IsInf(float64(maxv), 1) {
				continue
			}

			go stats.Frequency(templates[i].Truncated(), maxv)

			if maxv >= team.Time.Acceptance {
				go stats.Average(templates[i].Truncated(), maxv)
				go stats.Count(templates[i].Truncated())

				if maxp.X < locs[c] {
					locs[c] = maxp.X
					cols[c] = templates[i].Cols() - 2
					clock[c] = templates[i].Value
				}
			}

			go stats.Frequency(templates[i].Truncated(), 1)
		}

		if clock[c] == -1 {
			return 0, "00:00"
		}

		// Crop the left side of the selection area via the first <x,y> point matched.
		inset += locs[c] + cols[c]
		if inset > matrix.Cols() {
			break
		}
	}

	minutes := clock[0]*10 + clock[1]
	secs := clock[2]*10 + clock[3]
	kitchen = fmt.Sprintf("%d%d:%d%d", clock[0], clock[1], clock[2], clock[3])

	if clock[0] != 0 || minutes > 9 {
		notify.Error("Invalid time detected %s", kitchen)
		return 0, "00:00"
	}

	server.SetTime(minutes, secs)

	return minutes*60 + secs, kitchen
}

func IdentifyTime(mat gocv.Mat, kitchen string) (image.Image, error) {
	clone := mat.Clone()
	defer clone.Close()

	rect := image.Rect(clone.Cols()/4, 0, clone.Cols()-25, clone.Rows())
	region := clone.Region(rect)

	gocv.PutText(
		&region,
		kitchen,
		image.Pt(15, 75),
		gocv.FontHersheySimplex,
		1,
		rgba.White,
		4,
	)

	crop, err := region.ToImage()
	if err != nil {

		return nil, err
	}

	return crop, nil
}
