package rgba

import (
	"image"
	"image/color"
	"math"
)

type RGBA color.RGBA

var (
	Announce       = RGBA{R: 202, G: 222, B: 212, A: 255}
	Background     = RGBA{R: 9, G: 8, B: 12, A: 255}
	BackgroundAlt  = RGBA{R: 18, G: 17, B: 21, A: 255}
	Black          = RGBA{R: 0, G: 0, B: 0, A: 255}
	BloodOrange    = RGBA{R: 252, G: 72, B: 35, A: 255}
	CoolBlue       = RGBA{R: 71, G: 163, B: 255, A: 255}
	DarkRed        = RGBA{R: 166, G: 43, B: 53, A: 255}
	DarkSeafoam    = RGBA{R: 46, G: 204, B: 113, A: 255}
	DarkYellow     = RGBA{R: 204, G: 204, B: 0, A: 255}
	DarkBlue       = RGBA{R: 25, G: 25, B: 100, A: 50}
	DarkGray       = RGBA{R: 25, G: 25, B: 25, A: 255}
	DarkerYellow   = RGBA{R: 255, G: 255, B: 0, A: 63}
	DarkerRed      = RGBA{R: 255, G: 15, B: 15, A: 15}
	DeepBlue       = RGBA{R: 51, G: 51, B: 255, A: 255}
	Denounce       = RGBA{R: 222, G: 202, B: 206, A: 255}
	Disabled       = BackgroundAlt
	DreamyBlue     = RGBA{R: 115, G: 119, B: 250, A: 255}
	DreamyPurple   = RGBA{R: 143, G: 152, B: 247, A: 255}
	ForestGreen    = RGBA{R: 15, G: 255, B: 15, A: 63}
	Gold           = RGBA{R: 255, G: 204, B: 102, A: 255}
	Gray           = RGBA{R: 75, G: 75, B: 75, A: 255}
	Green          = RGBA{R: 0, G: 255, B: 0, A: 255}
	Highlight      = RGBA{R: 255, G: 255, B: 255, A: 255}
	LightGray      = RGBA{R: 100, G: 100, B: 100, A: 50}
	LightPurple    = RGBA{R: 204, G: 0, B: 204, A: 255}
	Night          = RGBA{R: 50, G: 50, B: 255, A: 63}
	Nothing        = RGBA{R: 0, G: 0, B: 0, A: 0}
	OfficeBlue     = RGBA{R: 51, G: 102, B: 255, A: 255}
	Orange         = RGBA{R: 255, G: 165, B: 0, A: 255}
	PastelBabyBlue = RGBA{R: 112, G: 170, B: 204, A: 255}
	PastelBlue     = RGBA{R: 130, G: 130, B: 223, A: 255}
	PastelGreen    = RGBA{R: 117, G: 199, B: 135, A: 255}
	PastelRed      = RGBA{R: 245, G: 95, B: 95, A: 255}
	Purple         = RGBA{R: 161, G: 76, B: 252, A: 255}
	PurpleBlue     = RGBA{R: 83, G: 94, B: 255, A: 255}
	PaleRed        = RGBA{R: 168, G: 49, B: 49, A: 255}
	Pinkity        = RGBA{R: 255, G: 112, B: 150, A: 255}
	Red            = RGBA{R: 255, G: 0, B: 0, A: 255}
	Regice         = SeaBlue
	Regieleki      = Yellow
	Regirock       = RGBA{R: 255, G: 102, B: 0, A: 255}
	Registeel      = PaleRed
	SeaBlue        = RGBA{R: 115, G: 165, B: 240, A: 255}
	Seafoam        = RGBA{R: 46, G: 204, B: 113, A: 255}
	SilverPurple   = RGBA{R: 102, G: 102, B: 153, A: 255}
	Slate          = RGBA{R: 255, G: 255, B: 255, A: 63}
	Splash         = RGBA{R: 10, G: 8, B: 20, A: 255}
	System         = RGBA{R: 95, G: 95, B: 95, A: 255}
	Transparent30  = RGBA{R: 0, G: 0, B: 0, A: 79}
	Transparent    = RGBA{R: 0, G: 0, B: 0, A: 0}
	User           = RGBA{R: 166, G: 139, B: 224, A: 255}
	White          = RGBA{R: 255, G: 255, B: 255, A: 255}
	Yellow         = RGBA{R: 255, G: 255, B: 0, A: 255}
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
