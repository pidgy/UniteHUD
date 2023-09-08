package nrgba

import (
	"fmt"
	"image/color"

	"github.com/pidgy/unitehud/rgba"
)

type NRGBA color.NRGBA

var (
	Announce       = NRGBA(rgba.Announce)
	Background     = NRGBA(rgba.Background)
	BackgroundAlt  = NRGBA(rgba.BackgroundAlt)
	Black          = NRGBA(rgba.Black)
	BloodOrange    = NRGBA(rgba.BloodOrange)
	CoolBlue       = NRGBA(rgba.CoolBlue)
	DarkRed        = NRGBA(rgba.DarkRed)
	DarkSeafoam    = NRGBA(rgba.DarkSeafoam)
	DarkYellow     = NRGBA(rgba.DarkYellow)
	DarkBlue       = NRGBA(rgba.DarkBlue)
	DarkGray       = NRGBA(rgba.DarkGray)
	DarkerYellow   = NRGBA(rgba.DarkerYellow)
	DarkerRed      = NRGBA(rgba.DarkerRed)
	DeepBlue       = NRGBA(rgba.DeepBlue)
	Denounce       = NRGBA(rgba.Denounce)
	Disabled       = NRGBA(rgba.Disabled)
	Discord        = NRGBA(rgba.Discord)
	DreamyBlue     = NRGBA(rgba.DreamyBlue)
	DreamyPurple   = NRGBA(rgba.DreamyPurple)
	ForestGreen    = NRGBA(rgba.ForestGreen)
	FullMoonBlue   = NRGBA(rgba.FullMoonBlue)
	Gold           = NRGBA(rgba.Gold)
	Gray           = NRGBA(rgba.Gray)
	Green          = NRGBA(rgba.Green)
	Highlight      = NRGBA(rgba.Highlight)
	Lemon          = NRGBA(rgba.Lemon)
	LightGray      = NRGBA(rgba.LightGray)
	LightPurple    = NRGBA(rgba.LightPurple)
	Lilac          = NRGBA(rgba.Lilac)
	Night          = NRGBA(rgba.Night)
	Nothing        = NRGBA(rgba.Nothing)
	OfficeBlue     = NRGBA(rgba.OfficeBlue)
	Orange         = NRGBA(rgba.Orange)
	Purple         = NRGBA(rgba.Purple)
	PurpleBlue     = NRGBA(rgba.PurpleBlue)
	PaleRed        = NRGBA(rgba.PaleRed)
	PastelBabyBlue = NRGBA(rgba.PastelBabyBlue)
	PastelBlue     = NRGBA(rgba.PastelBlue)
	PastelGreen    = NRGBA(rgba.PastelGreen)
	PastelOrange   = NRGBA(rgba.PastelOrange)
	PastelRed      = NRGBA(rgba.PastelRed)
	PastelYellow   = NRGBA(rgba.PastelYellow)
	Pinkity        = NRGBA(rgba.Pinkity)
	PolarBlue      = NRGBA(rgba.PolarBlue)
	Red            = NRGBA(rgba.Red)
	Regice         = SeaBlue
	Regieleki      = Yellow
	Regirock       = NRGBA(rgba.Regirock)
	Registeel      = PaleRed
	SeaBlue        = NRGBA(rgba.SeaBlue)
	Seafoam        = NRGBA(rgba.Seafoam)
	SilverPurple   = NRGBA(rgba.SilverPurple)
	Slate          = NRGBA(rgba.Slate)
	Splash         = NRGBA(rgba.Splash)
	System         = NRGBA(rgba.System)
	Transparent80  = NRGBA(rgba.Transparent80)
	Transparent    = NRGBA(rgba.Transparent)
	User           = NRGBA(rgba.User)
	White          = NRGBA(rgba.White)
	Yellow         = NRGBA(rgba.Yellow)
)

func (n NRGBA) Alpha(a uint8) NRGBA {
	n.A = a
	return n
}

func Bool(b bool) NRGBA {
	if b {
		return System
	}

	return System.Alpha(255 / 2)
}

func (n NRGBA) Color() color.NRGBA {
	return color.NRGBA(n)
}

func (n NRGBA) Ref() *color.NRGBA {
	c := color.NRGBA(n)
	return &c
}

func (n NRGBA) String() string {
	return fmt.Sprintf("(%d,%d,%d,%d)", n.R, n.G, n.B, n.A)
}

func Objective(name string) NRGBA {
	return NRGBA(rgba.Objective(name))
}
