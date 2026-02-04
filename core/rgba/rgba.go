package rgba

import (
	"image"
	"image/color"
	"math"
)

// RGBA is a local alias of color.RGBA for the project's palette and helpers.
type RGBA color.RGBA

var (
	// Active is the primary accent color.
	Active = RGBA{R: 117, G: 230, B: 218, A: 255}
	// Background is the default background color.
	Background = RGBA{R: 9, G: 8, B: 12, A: 255}
	// BackgroundAlt is the alternate background color.
	BackgroundAlt = RGBA{R: 18, G: 17, B: 21, A: 255}
	// Black is solid black.
	Black = RGBA{R: 0, G: 0, B: 0, A: 255}
	// BloodOrange is a bright orange-red.
	BloodOrange = RGBA{R: 252, G: 72, B: 35, A: 255}
	// CoolBlue is a cool blue accent.
	CoolBlue = RGBA{R: 71, G: 163, B: 255, A: 255}
	// CoralRed is a soft red tone.
	CoralRed = RGBA{R: 196, G: 90, B: 90, A: 255}
	// DarkRed is a deep red tone.
	DarkRed = RGBA{R: 166, G: 43, B: 53, A: 255}
	// DarkSeafoam is a muted seafoam green.
	DarkSeafoam = RGBA{R: 46, G: 204, B: 113, A: 255}
	// DarkYellow is a muted yellow tone.
	DarkYellow = RGBA{R: 204, G: 204, B: 0, A: 255}
	// DarkBlue is a deep blue tone.
	DarkBlue = RGBA{R: 25, G: 25, B: 100, A: 255}
	// DarkGray is a dark gray tone.
	DarkGray = RGBA{R: 25, G: 25, B: 25, A: 255}
	// DarkerYellow is a saturated yellow tone.
	DarkerYellow = RGBA{R: 255, G: 255, B: 0, A: 255}
	// DarkerRed is a saturated red tone.
	DarkerRed = RGBA{R: 255, G: 15, B: 15, A: 255}
	// DeepBlue is a vivid deep blue.
	DeepBlue = RGBA{R: 51, G: 51, B: 255, A: 255}
	// Denounce is a muted red-gray tone.
	Denounce = RGBA{R: 222, G: 202, B: 206, A: 255}
	// Disabled is the disabled-state color.
	Disabled = BackgroundAlt
	// Discord is the Discord brand color.
	Discord = RGBA{R: 114, G: 137, B: 218, A: 255}
	// DreamyBlue is a soft blue tone.
	DreamyBlue = RGBA{R: 115, G: 119, B: 250, A: 255}
	// DreamyPurple is a soft purple tone.
	DreamyPurple = RGBA{R: 143, G: 152, B: 247, A: 255}
	// ForestGreen is a bright green tone.
	ForestGreen = RGBA{R: 15, G: 255, B: 15, A: 255}
	// FullMoonBlue is a deep blue tone.
	FullMoonBlue = RGBA{R: 20, G: 50, B: 71, A: 255}
	// Gold is a gold tone.
	Gold = RGBA{R: 255, G: 204, B: 102, A: 255}
	// Gray is a mid gray tone.
	Gray = RGBA{R: 75, G: 75, B: 75, A: 255}
	// Green is a bright green tone.
	Green = RGBA{R: 0, G: 255, B: 0, A: 255}
	// Highlight is the highlight color.
	Highlight = RGBA{R: 255, G: 255, B: 255, A: 255}
	// Lemon is a light yellow-green tone.
	Lemon = RGBA{R: 206, G: 242, B: 85, A: 255}
	// LightGray is a light gray tone.
	LightGray = RGBA{R: 100, G: 100, B: 100, A: 255}
	// LightPurple is a bright purple tone.
	LightPurple = RGBA{R: 204, G: 0, B: 204, A: 255}
	// Lilac is a muted purple tone.
	Lilac = RGBA{R: 121, G: 103, B: 181, A: 255}
	// Night is a vivid night blue tone.
	Night = RGBA{R: 50, G: 50, B: 255, A: 255}
	// Nothing is a fully transparent color.
	Nothing = RGBA{R: 0, G: 0, B: 0, A: 0}
	// OfficeBlue is a bright blue tone.
	OfficeBlue = RGBA{R: 51, G: 102, B: 255, A: 255}
	// Orange is a bright orange tone.
	Orange = RGBA{R: 255, G: 165, B: 0, A: 255}
	// PastelBabyBlue is a soft baby blue tone.
	PastelBabyBlue = RGBA{R: 112, G: 170, B: 204, A: 255}
	// PastelBlue is a soft blue tone.
	PastelBlue = RGBA{R: 130, G: 130, B: 223, A: 255}
	// PastelCoral is a soft coral tone.
	PastelCoral = RGBA{255, 173, 195, 255}
	// PastelGreen is a soft green tone.
	PastelGreen = RGBA{R: 117, G: 199, B: 135, A: 255}
	// PastelOrange is a soft orange tone.
	PastelOrange = RGBA{R: 231, G: 137, B: 99, A: 255}
	// PastelRed is a soft red tone.
	PastelRed = RGBA{R: 245, G: 95, B: 95, A: 255}
	// PastelYellow is a soft yellow tone.
	PastelYellow = RGBA{R: 242, G: 212, B: 143, A: 255}
	// Purple is a vivid purple tone.
	Purple = RGBA{R: 161, G: 76, B: 252, A: 255}
	// PurpleBlue is a purple-blue tone.
	PurpleBlue = RGBA{R: 83, G: 94, B: 255, A: 255}
	// PaleRed is a muted red tone.
	PaleRed = RGBA{R: 168, G: 49, B: 49, A: 255}
	// Pinkity is a bright pink tone.
	Pinkity = RGBA{R: 255, G: 112, B: 150, A: 255}
	// PolarBlue is a muted polar blue tone.
	PolarBlue = RGBA{R: 64, G: 75, B: 122, A: 255}
	// Red is a bright red tone.
	Red = RGBA{R: 255, G: 0, B: 0, A: 255}
	// Regice is the Regice objective color.
	Regice = SeaBlue
	// Regieleki is the Regieleki objective color.
	Regieleki = Yellow
	// Regirock is the Regirock objective color.
	Regirock = RGBA{R: 255, G: 102, B: 0, A: 255}
	// Registeel is the Registeel objective color.
	Registeel = PaleRed
	// SeaBlue is a soft sea blue tone.
	SeaBlue = RGBA{R: 115, G: 165, B: 240, A: 255}
	// Seafoam is a seafoam green tone.
	Seafoam = RGBA{R: 46, G: 204, B: 113, A: 255}
	// SilverPurple is a muted purple-gray tone.
	SilverPurple = RGBA{R: 102, G: 102, B: 153, A: 255}
	// Slate is a light slate gray tone.
	Slate = RGBA{R: 200, G: 200, B: 200, A: 255}
	// Splash is a deep splash background tone.
	Splash = RGBA{R: 10, G: 8, B: 20, A: 255}
	// System is the default system color.
	System = RGBA{R: 200, G: 200, B: 200, A: 255}
	// Transparent80 is an 80-alpha transparent black.
	Transparent80 = RGBA{R: 0, G: 0, B: 0, A: 80}
	// Transparent is fully transparent black.
	Transparent = RGBA{R: 0, G: 0, B: 0, A: 0}
	// User is the user accent color.
	User = RGBA{R: 166, G: 139, B: 224, A: 255}
	// White is solid white.
	White = RGBA{R: 255, G: 255, B: 255, A: 255}
	// Yellow is a bright yellow tone.
	Yellow = RGBA{R: 255, G: 255, B: 0, A: 255}
)

// Bool returns System for true, and a half-alpha System for false.
func Bool(b bool) RGBA {
	if b {
		return System
	}
	return System.Alpha(255 / 2)
}

// Grayscale returns a new grayscale image.
func Grayscale(img *image.RGBA) *image.Gray {
	bounds := img.Bounds()
	w, h := bounds.Max.X, bounds.Max.Y
	gray := image.NewGray(image.Rectangle{image.Point{0, 0}, image.Point{w, h}})
	for x := 0; x < w; x++ {
		for y := 0; y < h; y++ {
			imageColor := img.At(x, y)
			rr, gg, bb, _ := imageColor.RGBA()
			r := math.Pow(float64(rr), 2.2)
			g := math.Pow(float64(gg), 2.2)
			b := math.Pow(float64(bb), 2.2)
			m := math.Pow(0.2125*r+0.7154*g+0.0721*b, 1/2.2)
			yy := uint16(m + 0.5)
			gray.Set(x, y, color.Gray{uint8(yy >> 8)})
		}
	}

	return gray
}

// N converts a color.NRGBA to RGBA.
func N(n color.NRGBA) RGBA {
	return RGBA(n)
}

// Objective returns an objective color by name.
func Objective(name string) RGBA {
	switch name {
	case "regice":
		return Regice
	case "regirock":
		return Regirock
	case "registeel":
		return Registeel
	case "regieleki":
		return Regieleki
	}
	return System
}

// Color converts RGBA to color.RGBA.
func (r RGBA) Color() color.RGBA {
	return color.RGBA(r)
}

// Alpha returns a copy of RGBA with the given alpha applied.
func (r RGBA) Alpha(a uint8) RGBA {
	r.A = a
	return r
}
