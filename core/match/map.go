package match

import (
	"fmt"
	"image"

	"gocv.io/x/gocv"

	"github.com/pidgy/unitehud/core/config"
	"github.com/pidgy/unitehud/core/notify"
	"github.com/pidgy/unitehud/core/rgba"
	"github.com/pidgy/unitehud/core/state"
	"github.com/pidgy/unitehud/core/stats"
	"github.com/pidgy/unitehud/core/team"
)

type Tier struct {
	Destroyed bool
	image.Point
	Match float32
}

type Tier1 struct {
	Top    Tier
	Bottom Tier
}

type Tier2 struct {
	Top    Tier
	Bottom Tier
}

type Tier3 struct {
	Middle Tier
}

type Tiers struct {
	Tier1
	Tier2
	Tier3
}

type Goals struct {
	Purple Tiers
	Orange Tiers
}

type Objectives struct {
	Top    bool
	Bottom bool
}

type Map struct {
	Goals
	Objectives
}

func MiniMap(matrix gocv.Mat, img *image.RGBA) (Map, bool) {
	return Map{}, false

	g, ok := goals(matrix, img)
	if !ok {
		return Map{}, false
	}

	obj, ok := objectives(matrix, img)
	if !ok {
		return Map{}, false
	}

	return Map{
		Goals:      g,
		Objectives: obj,
	}, true
}

func objectives(matrix gocv.Mat, img *image.RGBA) (Objectives, bool) {
	return Objectives{}, true
}

func goals(matrix gocv.Mat, img *image.RGBA) (Goals, bool) {
	templates := config.Current.TemplatesGoals(team.Game.Name)

	purple := []Tier{}
	orange := []Tier{}

	for x := 0; x < 5; x++ {
		results := []gocv.Mat{}

		for _, template := range templates {
			mat := gocv.NewMat()
			defer mat.Close()

			results = append(results, mat)

			gocv.MatchTemplate(matrix, template.Mat, &mat, gocv.TmCcoeffNormed, mask)
		}

		for i := range results {
			if results[i].Empty() {
				notify.Warn("Detect: Empty result for %s", templates[i].Truncated())
				continue
			}

			_, maxv, _, maxp := gocv.MinMaxLoc(results[i])
			if maxv >= .9 {
				go stats.Average(templates[i].Truncated(), maxv)
				go stats.Count(templates[i].Truncated())

				switch e := state.EventType(templates[i].Value); e {
				case state.PurpleBaseOpen:
					println("purple base open: ", maxp.String())
					purple = append(purple, Tier{Point: maxp, Match: maxv})
					gocv.Rectangle(&matrix, image.Rectangle{maxp, maxp.Add(image.Pt(25, 25))}, rgba.Black.Color(), -1)
					gocv.PutText(&matrix, fmt.Sprintf("%.1f%%", maxv*100), maxp, gocv.FontHersheyPlain, 1, rgba.White.Color(), 2)
				case state.PurpleBaseClosed:
					println("purple base closed: ", maxp.String())
					purple = append(purple, Tier{Destroyed: true, Point: maxp, Match: maxv})
					gocv.Rectangle(&matrix, image.Rectangle{maxp, maxp.Add(image.Pt(25, 25))}, rgba.Black.Color(), -1)
					gocv.PutText(&matrix, fmt.Sprintf("%.1f%%", maxv*100), maxp, gocv.FontHersheyPlain, 1, rgba.White.Color(), 2)
				case state.OrangeBaseOpen:
					println("orange base open: ", maxp.String())
					orange = append(orange, Tier{Point: maxp, Match: maxv})
					gocv.Rectangle(&matrix, image.Rectangle{maxp, maxp.Add(image.Pt(25, 25))}, rgba.Black.Color(), -1)
					gocv.PutText(&matrix, fmt.Sprintf("%.1f%%", maxv*100), maxp, gocv.FontHersheyPlain, 1, rgba.White.Color(), 2)
				case state.OrangeBaseClosed:
					println("orange base closed: ", maxp.String(), templates[i].File)
					orange = append(orange, Tier{Destroyed: true, Point: maxp, Match: maxv})
					gocv.Rectangle(&matrix, image.Rectangle{maxp, maxp.Add(image.Pt(25, 25))}, rgba.Black.Color(), -1)
					gocv.PutText(&matrix, fmt.Sprintf("%.1f%%", maxv*100), maxp, gocv.FontHersheyPlain, 1, rgba.White.Color(), 2)
				}
			}

			go stats.Frequency(templates[i].Truncated(), 1)
		}
	}

	return Goals{}, true
}
