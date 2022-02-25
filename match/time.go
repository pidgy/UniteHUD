package match

import (
	"fmt"
	"image"
	"image/color"

	"github.com/rs/zerolog/log"
	"gocv.io/x/gocv"

	"github.com/pidgy/unitehud/config"
	"github.com/pidgy/unitehud/notify"
	"github.com/pidgy/unitehud/rgba"
	"github.com/pidgy/unitehud/server"
	"github.com/pidgy/unitehud/sort"
	"github.com/pidgy/unitehud/team"
)

func (m *Match) Time(matrix gocv.Mat, img *image.RGBA) (seconds int, kitchen string) {
	clock := [4]int{0, -1, -1, -1}

	region := matrix

	for i := range clock {
		results := []gocv.Mat{}

		for _, template := range config.Current.Templates["time"][team.Time.Name] {
			if template.Mat.Cols() > region.Cols() || template.Mat.Rows() > region.Rows() {
				log.Warn().Str("type", "time").Msg("match is outside the legal selection")
				notify.Feed(rgba.Red, "Time match is outside the configured selection area")

				if config.Current.RecordMissed {
					// dev.Capture(img, region, team.Time.Name, "missed-"+template.Name, false, template.Value)
				}

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

		_, order := pieces.Sort(false)
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
	kitchen = fmt.Sprintf("%d%d:%d%d", clock[0], clock[1], clock[2], clock[3])

	if clock[0] != 0 || mins > 9 {
		notify.Feed(rgba.Red, "Invalid time detected %s", kitchen)
		return 0, "00:00"
	}

	server.Time(mins, secs)

	return mins*60 + secs, kitchen
}

func IdentifyTime(mat gocv.Mat, kitchen string) (image.Image, error) {
	clone := mat.Clone()
	defer clone.Close()

	region := clone.Region(image.Rect(clone.Cols()/4, 0, clone.Cols()-(clone.Cols()/4), clone.Rows()/2))

	p := image.Pt(10, region.Rows()-15)
	gocv.PutText(&region, kitchen, p, gocv.FontHersheyPlain, 2, color.RGBA(rgba.Highlight), 3)

	crop, err := region.ToImage()
	if err != nil {
		log.Error().Err(err).Msg("failed to convert image")
		return nil, err
	}

	return crop, nil
}
