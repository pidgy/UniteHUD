package team

import (
	"image"
	"time"

	"gocv.io/x/gocv"

	"github.com/pidgy/unitehud/core/match/duplicate"
	"github.com/pidgy/unitehud/core/rgba/nrgba"
)

// Team represents a team side in Pokémon Unite.
type Team struct {
	Name                 string `json:"name"`
	title                string
	Alias                string `json:"-"`
	nrgba.NRGBA          `json:"-"`
	*duplicate.Duplicate `json:"-"`

	Killed           time.Time
	KilledWithPoints bool
	Holding          int `json:"-"`
	HoldingMax       int `json:"-"`

	Acceptance float32
	Delay      time.Duration
}

var (
	// Energy represents the number of balls held by self.
	Energy = &Team{
		Name:  "balls",
		title: "Balls",

		NRGBA:     nrgba.Purple,
		Duplicate: duplicate.New(-1, gocv.NewMat(), gocv.NewMat()),

		HoldingMax: 50,

		Acceptance: .8,
		Delay:      time.Second,
	}
	First = &Team{
		Name:  "first",
		title: "First",

		NRGBA:     nrgba.LightPurple,
		Duplicate: duplicate.New(-1, gocv.NewMat(), gocv.NewMat()),

		Acceptance: .64,
		Delay:      time.Second,
	}
	Game = &Team{
		Name:  "game",
		title: "Game",

		NRGBA:     nrgba.White,
		Duplicate: duplicate.New(-1, gocv.NewMat(), gocv.NewMat()),

		Delay:      time.Second * 2,
		Acceptance: .8,
	}
	None = &Team{
		Name:  "none",
		title: "None",

		NRGBA: nrgba.Slate,

		Duplicate: duplicate.New(-1, gocv.NewMat(), gocv.NewMat()),
	}
	// Orange represents the standard Team for the Orange side.
	Orange = &Team{
		Name:  "orange",
		title: "Orange",

		NRGBA:     nrgba.Orange,
		Duplicate: duplicate.New(-1, gocv.NewMat(), gocv.NewMat()),

		Acceptance: .84,

		// Greater than 1s to reduce duplication errors.
		// Less than 2s to avoid missing difficult capture windows.
		Delay: time.Millisecond * 1750,
	}
	// Purple represents the standard Team for the Purple side.
	Purple = &Team{
		Name:  "purple",
		title: "Purple",

		NRGBA:     nrgba.Purple,
		Duplicate: duplicate.New(-1, gocv.NewMat(), gocv.NewMat()),

		Acceptance: Orange.Acceptance,
		Delay:      Orange.Delay,
	}
	// Self represents a wrapper Team for the Purple side.
	Self = &Team{
		Name:  "self",
		title: "Self",

		NRGBA:     nrgba.User,
		Duplicate: duplicate.New(-1, gocv.NewMat(), gocv.NewMat()),

		Acceptance: .75,
		Delay:      time.Second / 4,
	}
	Time = &Team{
		Name:  "time",
		title: "Time",

		NRGBA:     nrgba.White,
		Duplicate: duplicate.New(-1, gocv.NewMat(), gocv.NewMat()),

		Acceptance: .8,
		Delay:      time.Second,
	}

	Teams = []*Team{Orange, Purple, Self, Energy, Game, Time, First}

	nameOf = map[string]*Team{
		Orange.Name: Orange,
		Purple.Name: Purple,
		Self.Name:   Self,
		Energy.Name: Energy,
		Game.Name:   Game,
		Time.Name:   Time,
		First.Name:  First,
	}
)

func Clear() {
	for _, t := range Teams {
		t.Holding = 0
		t.Duplicate = duplicate.New(-1, gocv.NewMat(), gocv.NewMat())
		t.Killed = time.Time{}
		t.Counted = false
	}
}

func Color(name string) nrgba.NRGBA {
	switch name {
	case Self.Name:
		return Self.NRGBA
	case Orange.Name:
		return Orange.NRGBA
	case Purple.Name:
		return Purple.NRGBA
	default:
		return None.NRGBA
	}
}

// Comparable returns a smaller ROI to help increase duplication accuracy assurance.
func (t *Team) Comparable(mat gocv.Mat) gocv.Mat {
	switch t.Name {
	case Self.Name:
		return mat.Region(image.Rect(0, 20, 225, 60))
	case First.Name:
		return mat.Region(image.Rect(30, 20, 300, 60))
	case Time.Name:
		return mat.Region(image.Rect(15, 30, 100, 60))
	default:
		return mat.Region(image.Rect(0, 30, 120, 60))
	}
}

// Crop returns the dimensions for a cropped ROI for use with granular template matching.
func (t *Team) Crop(p image.Point) image.Rectangle {
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

func (t *Team) String() string {
	return t.title
}

func Delay(team string) time.Duration {
	return nameOf[team].Delay
}
