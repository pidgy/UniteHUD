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

// Team represents a team side in Pokemon Unite.
type Team struct {
	Name                 string `json:"name"`
	color.RGBA           `json:"-"`
	*duplicate.Duplicate `json:"-"`

	Holding      int
	HoldingMax   int
	HoldingReset bool

	acceptance float32
}

var (
	// Orange represents the standard Team for the Orange side.
	Orange = &Team{
		Name:      "orange",
		RGBA:      rgba.Orange,
		Duplicate: duplicate.New(-1, gocv.NewMat(), gocv.NewMat()),

		acceptance: .8,
	}

	// Purple represents the standard Team for the Purple side.
	Purple = &Team{
		Name:      "purple",
		RGBA:      rgba.Purple,
		Duplicate: duplicate.New(-1, gocv.NewMat(), gocv.NewMat()),

		acceptance: .8,
	}

	// Self represents a wrapper Team for the Purple side.
	Self = &Team{
		Name:      "self",
		RGBA:      rgba.Yellow,
		Duplicate: duplicate.New(-1, gocv.NewMat(), gocv.NewMat()),

		acceptance: .75,
	}

	//Balls represents the number of balls held by self.
	Balls = &Team{
		Name:      "balls",
		RGBA:      rgba.Purple,
		Duplicate: duplicate.New(-1, gocv.NewMat(), gocv.NewMat()),

		HoldingMax:   50,
		HoldingReset: true,

		acceptance: .7,
	}

	// None represents the default case for an unidentifiable side.
	Game = &Team{
		Name:      "game",
		RGBA:      rgba.White,
		Duplicate: duplicate.New(-1, gocv.NewMat(), gocv.NewMat()),
	}

	Time = &Team{
		Name:      "time",
		RGBA:      rgba.White,
		Duplicate: duplicate.New(-1, gocv.NewMat(), gocv.NewMat()),
	}

	First = &Team{
		Name:      "first",
		RGBA:      color.RGBA(rgba.LightPurple),
		Duplicate: duplicate.New(-1, gocv.NewMat(), gocv.NewMat()),

		acceptance: .8,
	}
)

func (t *Team) Acceptance(base float32) float32 {
	if t.acceptance != 0 {
		return t.acceptance
	}

	return base
}

func Clear() {
	for _, t := range []*Team{Orange, Purple, Self, Balls, Game, Time, First} {
		t.Holding = 0
		t.HoldingReset = true
		t.Duplicate = duplicate.New(-1, gocv.NewMat(), gocv.NewMat())
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
		return mat.Region(image.Rect(15, 30, 150, 60))
	}
}

// Crop returns the dimensions for a cropped ROI for use with granular template matching.
func (t Team) Crop(p image.Point) image.Rectangle {
	switch t.Name {
	case Self.Name:
		return image.Rect(p.X, p.Y-100, p.X+300, p.Y+100)
	case First.Name:
		return image.Rect(p.X, 0, p.X+300, p.Y+100)
	case Purple.Name, Orange.Name:
		return image.Rect(p.X-50, p.Y-30, p.X+200, p.Y+75)
	default:
		return image.Rect(p.X-50, p.Y-30, p.X+200, p.Y+75)
	}
}

func Delay(name string) time.Duration {
	switch name {
	case Game.Name:
		return time.Second * 3
	case Self.Name:
		return time.Second / 4
	case Balls.Name:
		return time.Second
	case First.Name:
		return time.Second / 4
	case Purple.Name, Orange.Name:
		return time.Second
	default:
		return time.Second
	}
}

func (t Team) MarshalZerologObject(e *zerolog.Event) {
	e.Str("name", t.Name)
}
