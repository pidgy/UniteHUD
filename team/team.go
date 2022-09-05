package team

import (
	"image"
	"image/color"
	"time"

	"github.com/rs/zerolog"
	"gocv.io/x/gocv"

	"github.com/pidgy/unitehud/duplicate"
	"github.com/pidgy/unitehud/rgba"
)

// Team represents a team side in Pokémon Unite.
type Team struct {
	Name                 string `json:"name"`
	Alias                string `json:"-"`
	color.RGBA           `json:"-"`
	*duplicate.Duplicate `json:"-"`

	Killed       time.Time
	Holding      int  `json:"-"`
	HoldingMax   int  `json:"-"`
	HoldingReset bool `json:"-"`

	Acceptance float32
	Delay      time.Duration
}

var (
	// Balls represents the number of balls held by self.
	Balls = &Team{
		Name:      "balls",
		RGBA:      rgba.Purple,
		Duplicate: duplicate.New(-1, gocv.NewMat(), gocv.NewMat()),

		HoldingMax:   50,
		HoldingReset: true,

		Acceptance: .7,
		Delay:      time.Second,
	}

	First = &Team{
		Name:      "first",
		RGBA:      color.RGBA(rgba.LightPurple),
		Duplicate: duplicate.New(-1, gocv.NewMat(), gocv.NewMat()),

		Acceptance: .8,
		Delay:      time.Second / 4,
	}

	Game = &Team{
		Name:      "game",
		RGBA:      rgba.White,
		Duplicate: duplicate.New(-1, gocv.NewMat(), gocv.NewMat()),

		Delay:      time.Second * 5,
		Acceptance: .8,
	}

	// Orange represents the standard Team for the Orange side.
	Orange = &Team{
		Name:      "orange",
		RGBA:      rgba.Orange,
		Duplicate: duplicate.New(-1, gocv.NewMat(), gocv.NewMat()),

		Acceptance: .8,
		Delay:      time.Second,
	}

	// Purple represents the standard Team for the Purple side.
	Purple = &Team{
		Name:      "purple",
		RGBA:      rgba.Purple,
		Duplicate: duplicate.New(-1, gocv.NewMat(), gocv.NewMat()),

		Acceptance: .8,
		Delay:      time.Second,
	}

	// Self represents a wrapper Team for the Purple side.
	Self = &Team{
		Name:      "self",
		RGBA:      rgba.Yellow,
		Duplicate: duplicate.New(-1, gocv.NewMat(), gocv.NewMat()),

		Acceptance: .75,
		Delay:      time.Second / 4,
	}

	Time = &Team{
		Name:      "time",
		RGBA:      rgba.White,
		Duplicate: duplicate.New(-1, gocv.NewMat(), gocv.NewMat()),

		Acceptance: .8,
		Delay:      time.Second,
	}

	Teams = []*Team{Orange, Purple, Self, Balls, Game, Time, First}

	cache = map[string]*Team{
		Orange.Name: Orange,
		Purple.Name: Purple,
		Self.Name:   Self,
		Balls.Name:  Balls,
		Game.Name:   Game,
		Time.Name:   Time,
		First.Name:  First,
	}
)

func Clear() {
	for _, t := range Teams {
		t.Holding = 0
		t.HoldingReset = true
		t.Duplicate = duplicate.New(-1, gocv.NewMat(), gocv.NewMat())
		t.Killed = time.Time{}
	}
}

// Comparable returns a smaller ROI to help increase duplication accuracy assurance.
func (t Team) Comparable(mat gocv.Mat) gocv.Mat {
	switch t.Name {
	case Self.Name:
		return mat.Region(image.Rect(0, 20, 225, 60))
	case First.Name:
		return mat.Region(image.Rect(30, 20, 300, 60))
	case Time.Name:
		return mat.Region(image.Rect(15, 30, 100, 60))
	default:
		return mat.Region(image.Rect(0, 30, 100, 60))
	}
}

// Crop returns the dimensions for a cropped ROI for use with granular template matching.
func (t Team) Crop(p image.Point) image.Rectangle {
	switch t.Name {
	case Self.Name:
		return image.Rect(p.X, p.Y-100, p.X+300, p.Y+100)
	case First.Name:
		return image.Rect(p.X, 0, p.X+350, p.Y+100)
	case Purple.Name, Orange.Name:
		return image.Rect(p.X-50, p.Y-30, p.X+200, p.Y+75)
	default:
		return image.Rect(p.X-50, p.Y-30, p.X+200, p.Y+75)
	}
}

func Delay(team string) time.Duration {
	return cache[team].Delay
}

func (t Team) MarshalZerologObject(e *zerolog.Event) {
	e.Str("name", t.Name)
}
