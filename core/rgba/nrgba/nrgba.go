package nrgba

import (
	"fmt"
	"image/color"
	"math"

	"github.com/pidgy/unitehud/core/rgba"
)

type NRGBA color.NRGBA

var (
	Any = NRGBA{}

	Active         = NRGBA(rgba.Active)
	Background     = NRGBA(rgba.Background)
	BackgroundAlt  = NRGBA(rgba.BackgroundAlt)
	Black          = NRGBA(rgba.Black)
	BloodOrange    = NRGBA(rgba.BloodOrange)
	CoolBlue       = NRGBA(rgba.CoolBlue)
	CoralRed       = NRGBA(rgba.CoralRed)
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
	PastelCoral    = NRGBA(rgba.PastelCoral)
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

func Bool(b bool) NRGBA {
	if b {
		return System
	}
	return System.Alpha(255 / 2)
}

func (n NRGBA) Alpha(a uint8) NRGBA {
	n.A = a
	return n
}

func (n NRGBA) Color() color.NRGBA {
	return color.NRGBA(n)
}

func (n NRGBA) Div(f float64) NRGBA {
	return NRGBA{R: uint8(float64(float64(n.R) / f)), G: uint8(float64(float64(n.G) / f)), B: uint8(float64(float64(n.B) / f)), A: n.A}
}

func (n NRGBA) Eq(n2 NRGBA) bool {
	if n == Any {
		return true
	}

	return n.R == n2.R && n.G == n2.G && n.B == n2.B && n.A == n2.A
}

func (n NRGBA) Hex() string {
	return fmt.Sprintf("%02x%02x%02x%02x", n.R, n.G, n.B, n.A)
}

func (n NRGBA) Ref() *color.NRGBA {
	c := color.NRGBA(n)
	return &c
}

func (n NRGBA) String() string {
	return fmt.Sprintf("(%d,%d,%d,%d)", n.R, n.G, n.B, n.A)
}

func (n NRGBA) Mul(f float64) NRGBA {
	return NRGBA{R: uint8(float64(float64(n.R) * f)), G: uint8(float64(float64(n.G) * f)), B: uint8(float64(float64(n.B) * f)), A: n.A}
}

func colorDistance(n1, n2 NRGBA) float64 {
	dr := float64(n1.R - n2.R)
	dg := float64(n1.G - n2.G)
	db := float64(n1.B - n2.B)
	return dr + dg + db
}

// var cmdColors = map[string]NRGBA{
// 	// "90m": {R: 128, G: 128, B: 128, A: 255}, // Bright black (gray)
// 	"91m": {R: 255, G: 0, B: 0, A: 255},     // Bright red
// 	"92m": {R: 0, G: 255, B: 0, A: 255},     // Bright green
// 	"93m": {R: 255, G: 255, B: 0, A: 255},   // Bright yellow
// 	"94m": {R: 0, G: 0, B: 255, A: 255},     // Bright blue
// 	"95m": {R: 255, G: 0, B: 255, A: 255},   // Bright magenta
// 	"96m": {R: 0, G: 255, B: 255, A: 255},   // Bright cyan
// 	"97m": {R: 255, G: 255, B: 255, A: 255}, // Bright white
// }

var (
	cache = [256][256][256]string{}
)

func (n NRGBA) Windows(s string) string {
	f := func(c string) string {
		cache[n.R][n.G][n.B] = c
		return fmt.Sprintf("\033[0;%s%s\033[0m\n\033[0m", c, s)
	}

	w := cache[n.R][n.G][n.B]
	if w != "" {
		return f(w)
	}

	r := float64(n.R)
	g := float64(n.G)
	b := float64(n.B)

	max := math.Max(r, math.Max(g, b))
	min := math.Min(r, math.Min(g, b))
	delta := max - min

	// Handle grayscale.
	if delta < 0.05 {
		switch {
		case max < 0.1:
			return f("90m")
		case max > 0.9:
			return f("97m")
		default:
			return f("90m")
		}
	}

	// Compute hue.
	var hue float64
	switch max {
	case r:
		hue = math.Mod((g-b)/delta, 6)
	case g:
		hue = (b-r)/delta + 2
	case b:
		hue = (r-g)/delta + 4
	}
	hue *= 60
	if hue < 0 {
		hue += 360
	}

	// Bucket hue into color names.
	switch {
	case hue < 15 || hue >= 345:
		return f("97m")
	case hue < 45:
		return f("31m")
	case hue < 75:
		return f("93m")
	case hue < 150:
		return f("92m")
	case hue < 210:
		return f("36m")
	case hue < 270:
		return f("94m")
	case hue < 345:
		return f("91m")
	default:
		return f("97m")
	}
}

func Objective(name string) NRGBA {
	return NRGBA(rgba.Objective(name))
}

func Percent(p float64) NRGBA {
	switch {
	case p >= .9:
		return PastelGreen
	case p >= .69:
		return PastelYellow
	case p >= .49:
		return PastelOrange
	default:
		return PastelRed
	}
}

func Status(s float64) NRGBA {
	switch {
	case s >= 50:
		return PastelGreen
	case s >= 30:
		return PastelOrange
	case s >= 15:
		return PastelYellow
	default:
		return PastelRed
	}
}
