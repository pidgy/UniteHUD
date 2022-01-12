package team

import (
	"image"
	"image/color"
	"time"

	"github.com/rs/zerolog"
	"gocv.io/x/gocv"

	"github.com/pidgy/unitehud/duplicate"
)

var (
	black  = color.RGBA{0, 0, 0, 255}
	orange = color.RGBA{255, 165, 0, 255}
	purple = color.RGBA{83, 94, 255, 255}
	// white  = color.RGBA{255, 255, 255, 255}
)

// Team represents a team side in Pokemon Unite.
type Team struct {
	Name                 string `json:"name"`
	color.RGBA           `json:"-"`
	*duplicate.Duplicate `json:"-"`
}

var (
	// Orange represents the standard Team for the Orange side.
	Orange = &Team{
		Name:      "orange",
		RGBA:      orange,
		Duplicate: duplicate.New(-1, gocv.NewMat(), gocv.NewMat()),
	}

	// Purple represents the standard Team for the Purple side.
	Purple = &Team{
		Name:      "purple",
		RGBA:      purple,
		Duplicate: duplicate.New(-1, gocv.NewMat(), gocv.NewMat()),
	}

	// Self represents a wrapper Team for the Purple side.
	Self = &Team{
		Name:      "self",
		RGBA:      purple,
		Duplicate: duplicate.New(-1, gocv.NewMat(), gocv.NewMat()),
	}

	//Balls represents the number of balls held by self.
	Balls = &Team{
		Name:      "balls",
		RGBA:      purple,
		Duplicate: duplicate.New(-1, gocv.NewMat(), gocv.NewMat()),
	}

	// None represents the default case for an unidentifiable side.
	None = &Team{
		Name:      "none",
		RGBA:      black,
		Duplicate: duplicate.New(-1, gocv.NewMat(), gocv.NewMat()),
	}

	Time = &Team{
		Name:      "time",
		RGBA:      black,
		Duplicate: duplicate.New(-1, gocv.NewMat(), gocv.NewMat()),
	}
)

func Delay(name string) time.Duration {
	switch name {
	case None.Name:
		return time.Second * 5
	case Self.Name:
		return time.Second / 4
	case Balls.Name:
		return time.Second
	default:
		return time.Second * 2
	}
}

func (t Team) Rectangle(p image.Point) image.Rectangle {
	if t.Name == Self.Name {
		return image.Rect(p.X-200, p.Y-50, p.X+250, p.Y+100)
	}
	return image.Rect(p.X-100, p.Y-30, p.X+150, p.Y+75)
}

func (t Team) Region(mat gocv.Mat) gocv.Mat {
	if t.Name == Self.Name {
		return mat.Region(image.Rect(30, 20, 225, 60))
	}

	return mat.Region(image.Rect(15, 30, 150, 60))
}

func (t Team) MarshalZerologObject(e *zerolog.Event) {
	e.Str("name", t.Name)
}
