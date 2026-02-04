package nrgba

import (
	"fmt"
	"image/color"
	"math"

	"github.com/pidgy/unitehud/core/rgba"
)

// NRGBA is a color.NRGBA with convenience helpers.
type NRGBA color.NRGBA

var (
	// Any is a wildcard that matches any color in comparisons.
	Any = NRGBA{}

	// Active wraps the RGBA preset color.
	Active = NRGBA(rgba.Active)
	// Background wraps the RGBA preset color.
	Background = NRGBA(rgba.Background)
	// BackgroundAlt wraps the RGBA preset color.
	BackgroundAlt = NRGBA(rgba.BackgroundAlt)
	// Black wraps the RGBA preset color.
	Black = NRGBA(rgba.Black)
	// BloodOrange wraps the RGBA preset color.
	BloodOrange = NRGBA(rgba.BloodOrange)
	// CoolBlue wraps the RGBA preset color.
	CoolBlue = NRGBA(rgba.CoolBlue)
	// CoralRed wraps the RGBA preset color.
	CoralRed = NRGBA(rgba.CoralRed)
	// DarkRed wraps the RGBA preset color.
	DarkRed = NRGBA(rgba.DarkRed)
	// DarkSeafoam wraps the RGBA preset color.
	DarkSeafoam = NRGBA(rgba.DarkSeafoam)
	// DarkYellow wraps the RGBA preset color.
	DarkYellow = NRGBA(rgba.DarkYellow)
	// DarkBlue wraps the RGBA preset color.
	DarkBlue = NRGBA(rgba.DarkBlue)
	// DarkGray wraps the RGBA preset color.
	DarkGray = NRGBA(rgba.DarkGray)
	// DarkerYellow wraps the RGBA preset color.
	DarkerYellow = NRGBA(rgba.DarkerYellow)
	// DarkerRed wraps the RGBA preset color.
	DarkerRed = NRGBA(rgba.DarkerRed)
	// DeepBlue wraps the RGBA preset color.
	DeepBlue = NRGBA(rgba.DeepBlue)
	// Denounce wraps the RGBA preset color.
	Denounce = NRGBA(rgba.Denounce)
	// Disabled wraps the RGBA preset color.
	Disabled = NRGBA(rgba.Disabled)
	// Discord wraps the RGBA preset color.
	Discord = NRGBA(rgba.Discord)
	// DreamyBlue wraps the RGBA preset color.
	DreamyBlue = NRGBA(rgba.DreamyBlue)
	// DreamyPurple wraps the RGBA preset color.
	DreamyPurple = NRGBA(rgba.DreamyPurple)
	// ForestGreen wraps the RGBA preset color.
	ForestGreen = NRGBA(rgba.ForestGreen)
	// FullMoonBlue wraps the RGBA preset color.
	FullMoonBlue = NRGBA(rgba.FullMoonBlue)
	// Gold wraps the RGBA preset color.
	Gold = NRGBA(rgba.Gold)
	// Gray wraps the RGBA preset color.
	Gray = NRGBA(rgba.Gray)
	// Green wraps the RGBA preset color.
	Green = NRGBA(rgba.Green)
	// Highlight wraps the RGBA preset color.
	Highlight = NRGBA(rgba.Highlight)
	// Lemon wraps the RGBA preset color.
	Lemon = NRGBA(rgba.Lemon)
	// LightGray wraps the RGBA preset color.
	LightGray = NRGBA(rgba.LightGray)
	// LightPurple wraps the RGBA preset color.
	LightPurple = NRGBA(rgba.LightPurple)
	// Lilac wraps the RGBA preset color.
	Lilac = NRGBA(rgba.Lilac)
	// Night wraps the RGBA preset color.
	Night = NRGBA(rgba.Night)
	// Nothing wraps the RGBA preset color.
	Nothing = NRGBA(rgba.Nothing)
	// OfficeBlue wraps the RGBA preset color.
	OfficeBlue = NRGBA(rgba.OfficeBlue)
	// Orange wraps the RGBA preset color.
	Orange = NRGBA(rgba.Orange)
	// Purple wraps the RGBA preset color.
	Purple = NRGBA(rgba.Purple)
	// PurpleBlue wraps the RGBA preset color.
	PurpleBlue = NRGBA(rgba.PurpleBlue)
	// PaleRed wraps the RGBA preset color.
	PaleRed = NRGBA(rgba.PaleRed)
	// PastelBabyBlue wraps the RGBA preset color.
	PastelBabyBlue = NRGBA(rgba.PastelBabyBlue)
	// PastelBlue wraps the RGBA preset color.
	PastelBlue = NRGBA(rgba.PastelBlue)
	// PastelCoral wraps the RGBA preset color.
	PastelCoral = NRGBA(rgba.PastelCoral)
	// PastelGreen wraps the RGBA preset color.
	PastelGreen = NRGBA(rgba.PastelGreen)
	// PastelOrange wraps the RGBA preset color.
	PastelOrange = NRGBA(rgba.PastelOrange)
	// PastelRed wraps the RGBA preset color.
	PastelRed = NRGBA(rgba.PastelRed)
	// PastelYellow wraps the RGBA preset color.
	PastelYellow = NRGBA(rgba.PastelYellow)
	// Pinkity wraps the RGBA preset color.
	Pinkity = NRGBA(rgba.Pinkity)
	// PolarBlue wraps the RGBA preset color.
	PolarBlue = NRGBA(rgba.PolarBlue)
	// Red wraps the RGBA preset color.
	Red = NRGBA(rgba.Red)
	// Regice is an alias for SeaBlue.
	Regice = SeaBlue
	// Regieleki is an alias for Yellow.
	Regieleki = Yellow
	// Regirock wraps the RGBA preset color.
	Regirock = NRGBA(rgba.Regirock)
	// Registeel is an alias for PaleRed.
	Registeel = PaleRed
	// SeaBlue wraps the RGBA preset color.
	SeaBlue = NRGBA(rgba.SeaBlue)
	// Seafoam wraps the RGBA preset color.
	Seafoam = NRGBA(rgba.Seafoam)
	// SilverPurple wraps the RGBA preset color.
	SilverPurple = NRGBA(rgba.SilverPurple)
	// Slate wraps the RGBA preset color.
	Slate = NRGBA(rgba.Slate)
	// Splash wraps the RGBA preset color.
	Splash = NRGBA(rgba.Splash)
	// System wraps the RGBA preset color.
	System = NRGBA(rgba.System)
	// Transparent80 wraps the RGBA preset color.
	Transparent80 = NRGBA(rgba.Transparent80)
	// Transparent wraps the RGBA preset color.
	Transparent = NRGBA(rgba.Transparent)
	// User wraps the RGBA preset color.
	User = NRGBA(rgba.User)
	// White wraps the RGBA preset color.
	White = NRGBA(rgba.White)
	// Yellow wraps the RGBA preset color.
	Yellow = NRGBA(rgba.Yellow)
)

// Bool returns System for true and a half-alpha System for false.
func Bool(b bool) NRGBA {
	if b {
		return System
	}
	return System.Alpha(255 / 2)
}

// Alpha returns a copy with the provided alpha value.
func (n NRGBA) Alpha(a uint8) NRGBA {
	n.A = a
	return n
}

// Color converts the value to a standard library color.NRGBA.
func (n NRGBA) Color() color.NRGBA {
	return color.NRGBA(n)
}

// Div divides RGB components by f, preserving alpha.
func (n NRGBA) Div(f float64) NRGBA {
	return NRGBA{R: uint8(float64(float64(n.R) / f)), G: uint8(float64(float64(n.G) / f)), B: uint8(float64(float64(n.B) / f)), A: n.A}
}

// Eq reports whether n matches n2, honoring the Any wildcard.
func (n NRGBA) Eq(n2 NRGBA) bool {
	if n == Any {
		return true
	}

	return n.R == n2.R && n.G == n2.G && n.B == n2.B && n.A == n2.A
}

// Hex formats the color as an 8-digit RGBA hex string.
func (n NRGBA) Hex() string {
	return fmt.Sprintf("%02x%02x%02x%02x", n.R, n.G, n.B, n.A)
}

// Ref returns a pointer to a copy of the color.NRGBA value.
func (n NRGBA) Ref() *color.NRGBA {
	c := color.NRGBA(n)
	return &c
}

// String formats the color as a comma-separated RGBA tuple.
func (n NRGBA) String() string {
	return fmt.Sprintf("(%d,%d,%d,%d)", n.R, n.G, n.B, n.A)
}

// Mul multiplies RGB components by f, preserving alpha.
func (n NRGBA) Mul(f float64) NRGBA {
	return NRGBA{R: uint8(float64(float64(n.R) * f)), G: uint8(float64(float64(n.G) * f)), B: uint8(float64(float64(n.B) * f)), A: n.A}
}

// colorDistance computes a simple RGB distance metric.
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
	// cache stores ANSI color codes by RGB for Windows formatting.
	cache = [256][256][256]string{}
)

// Windows formats s with an ANSI color code approximating the color for Windows terminals.
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
		return f("91m")
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

// Objective returns the objective color by name.
func Objective(name string) NRGBA {
	return NRGBA(rgba.Objective(name))
}

// Percent maps a percent to a pastel status color.
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

// Status maps a status score to a pastel status color.
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
