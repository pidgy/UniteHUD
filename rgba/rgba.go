package rgba

import (
	"image"
	"image/color"
	"math"
)

type RGBA color.RGBA

var (
	Announce       = RGBA{202, 222, 212, 255}
	Background     = RGBA{9, 8, 12, 255}
	BackgroundAlt  = RGBA{18, 17, 21, 255}
	Black          = RGBA{0, 0, 0, 255}
	BloodOrange    = RGBA{252, 72, 35, 255}
	CoolBlue       = RGBA{R: 71, G: 163, B: 255, A: 255}
	DarkRed        = RGBA{166, 43, 53, 255}
	DarkSeafoam    = RGBA{R: 46, G: 204, B: 113, A: 0xFF}
	DarkYellow     = RGBA{R: 204, G: 204, A: 0xFF}
	DarkBlue       = RGBA{R: 25, G: 25, B: 100, A: 50}
	DarkGray       = RGBA{R: 25, G: 25, B: 25, A: 255}
	DarkerYellow   = RGBA{R: 0xFF, G: 0xFF, A: 0x3F}
	DarkerRed      = RGBA{R: 0xFF, G: 0xF, B: 0xF, A: 0x0F}
	Denounce       = RGBA{222, 202, 206, 255}
	Disabled       = BackgroundAlt
	DreamyBlue     = RGBA{115, 119, 250, 255}
	DreamyPurple   = RGBA{143, 152, 247, 255}
	ForestGreen    = RGBA{R: 0xF, G: 0xFF, B: 0xF, A: 0x3F}
	PastelBabyBlue = RGBA{112, 170, 204, 255}
	PastelBlue     = RGBA{130, 130, 223, 255}
	PastelGreen    = RGBA{117, 199, 135, 255}
	PastelRed      = RGBA{245, 95, 95, 255}
	Gray           = RGBA{R: 75, G: 75, B: 75, A: 255}
	Green          = RGBA{G: 0xFF, A: 0xFF}
	Highlight      = RGBA{255, 255, 255, 255}
	LightGray      = RGBA{100, 100, 100, 50}
	LightPurple    = RGBA{204, 0, 204, 255}
	Night          = RGBA{50, 50, 0xFF, 0x3F}
	Orange         = RGBA{255, 165, 0, 255}
	Purple         = RGBA{161, 76, 252, 255}
	PurpleBlue     = RGBA{83, 94, 255, 255}
	PaleRed        = RGBA{168, 49, 49, 255}
	Pinkity        = RGBA{255, 112, 150, 255}
	Red            = RGBA{R: 0xFF, A: 0xFF}
	Regice         = SeaBlue
	Regieleki      = Yellow
	Regirock       = RGBA{R: 255, G: 102, B: 0, A: 0xFF}
	Registeel      = PaleRed
	SeaBlue        = RGBA{115, 165, 240, 255}
	Seafoam        = RGBA{R: 46, G: 204, B: 113, A: 0xFF}
	Slate          = RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0x3F}
	Splash         = RGBA{10, 8, 20, 255}
	System         = RGBA{R: 95, G: 95, B: 95, A: 255}
	Transparent30  = RGBA{A: 0x4F}
	Transparent    = RGBA{}
	User           = RGBA{166, 139, 224, 255}
	White          = RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}
	Yellow         = RGBA{R: 0xFF, G: 0xFF, A: 0xFF}
)

func Bool(b bool) RGBA {
	if b {
		return System
	}
	return System.Alpha(255 / 2)
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

func N(n color.NRGBA) RGBA {
	return RGBA(n)
}

func (r RGBA) Color() color.RGBA {
	return color.RGBA(r)
}

func (r RGBA) Alpha(a uint8) RGBA {
	r.A = a
	return r
}

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
