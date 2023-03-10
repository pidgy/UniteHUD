package rgba

import (
	"image"
	"image/color"
	"math"
)

var (
	Announce      = color.RGBA{202, 222, 212, 255}
	Background    = color.RGBA{9, 8, 12, 255}
	BackgroundAlt = color.RGBA{18, 17, 21, 255}
	Black         = color.RGBA{0, 0, 0, 255}
	BloodOrange   = color.RGBA{252, 72, 35, 255}
	CoolBlue      = color.RGBA{R: 71, G: 163, B: 255, A: 255}
	DarkBlue      = color.RGBA{R: 25, G: 25, B: 100, A: 50}
	DarkerYellow  = color.RGBA{R: 0xFF, G: 0xFF, A: 0x3F}
	DarkerRed     = color.RGBA{R: 0xFF, G: 0xF, B: 0xF, A: 0x0F}
	DarkGray      = color.RGBA{A: 0x4F}
	DarkRed       = color.RGBA{166, 43, 53, 255}
	DarkSeafoam   = color.RGBA{R: 46, G: 204, B: 113, A: 0xFF}
	DarkYellow    = color.RGBA{R: 204, G: 204, A: 0xFF}
	Denounce      = color.RGBA{222, 202, 206, 255}
	Disabled      = color.RGBA{8, 7, 7, 200}
	DreamyBlue    = color.RGBA{115, 119, 250, 255}
	DreamyPurple  = color.RGBA{143, 152, 247, 255}
	ForestGreen   = color.RGBA{R: 0xF, G: 0xFF, B: 0xF, A: 0x3F}
	Gray          = color.RGBA{R: 75, G: 75, B: 75, A: 255}
	Green         = color.RGBA{G: 0xFF, A: 0xFF}
	Highlight     = color.RGBA{255, 255, 255, 255}
	LightPurple   = color.RGBA{204, 0, 204, 255}
	Night         = color.RGBA{50, 50, 0xFF, 0x3F}
	Orange        = color.RGBA{255, 165, 0, 255}
	Purple        = color.RGBA{161, 76, 252, 255}
	PurpleBlue    = color.RGBA{83, 94, 255, 255}
	PaleRed       = color.RGBA{168, 49, 49, 255}
	Pinkity       = color.RGBA{255, 112, 150, 255}
	Red           = color.RGBA{R: 0xFF, A: 0xFF}
	Regice        = SeaBlue
	Regieleki     = Yellow
	Regirock      = color.RGBA{R: 255, G: 102, B: 0, A: 0xFF}
	Registeel     = PaleRed
	SeaBlue       = color.RGBA{115, 165, 240, 255}
	Seafoam       = color.RGBA{R: 46, G: 204, B: 113, A: 0xFF}
	Slate         = color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0x3F}
	System        = color.RGBA{R: 95, G: 95, B: 95, A: 255}
	User          = color.RGBA{166, 139, 224, 255}
	White         = color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}
	Yellow        = color.RGBA{R: 0xFF, G: 0xFF, A: 0xFF}
)

func Alpha(c color.RGBA, a uint8) color.RGBA {
	c.A = a
	return c
}

func Bool(b bool) color.RGBA {
	if b {
		return System
	}

	return Alpha(System, 255/2)
}

// Grayscale returns a new grayscale image
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

func N(c color.RGBA) color.NRGBA {
	return color.NRGBA(c)
}

func Objective(name string) color.RGBA {
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

func R(c color.NRGBA) color.RGBA {
	return color.RGBA(c)
}
