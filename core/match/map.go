package match

// TODO: Add go style comments that reflect the purpose of each type, function, var, and const.

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

// Tier defines Tier behavior and state.
type Tier struct {
	Destroyed bool
	image.Point
	Match float32
}

// Tier1 defines Tier1 behavior and state.
type Tier1 struct {
	Top    Tier
	Bottom Tier
}

// Tier2 defines Tier2 behavior and state.
type Tier2 struct {
	Top    Tier
	Bottom Tier
}

// Tier3 defines Tier3 behavior and state.
type Tier3 struct {
	Middle Tier
}

// Tiers defines Tiers behavior and state.
type Tiers struct {
	Tier1
	Tier2
	Tier3
}

// Goals defines Goals behavior and state.
type Goals struct {
	Purple Tiers
	Orange Tiers
}

// Objectives defines Objectives behavior and state.
type Objectives struct {
	Top    bool
	Bottom bool
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

			gocv.MatchTemplate(matrix, template.Mat, &mat, config.DefaultMatchMethod, mask)
		}

		for i := range results {
			if results[i].Empty() {
				notify.Warn("[Detect] Empty result for %s", templates[i].Truncated())
				continue
			}

			_, maxv, _, maxp := gocv.MinMaxLoc(results[i])
			if maxv < .9 {
				continue
			}

			go stats.Collect(templates[i].Truncated(), maxv)

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
	}

	return Goals{}, true
}


func objectives(matrix gocv.Mat, img *image.RGBA) (Objectives, bool) {
	return Objectives{}, true
}
