package rgba

import (
	"image"
	"image/color"
	"math"
)

var (
	Black        = color.RGBA{0, 0, 0, 255}
	Background   = color.RGBA{R: 75, G: 75, B: 75, A: 255}
	CoolBlue     = color.RGBA{R: 71, G: 163, B: 255, A: 255}
	DarkBlue     = color.RGBA{R: 25, G: 25, B: 100, A: 50}
	DarkerGray   = color.RGBA{A: 0xF}
	DarkerYellow = color.RGBA{R: 0xFF, G: 0xFF, A: 0x3F}
	DarkerRed    = color.RGBA{R: 0xFF, G: 0xF, B: 0xF, A: 0x0F}
	DarkGray     = color.RGBA{A: 0x4F}
	DarkRed      = color.RGBA{R: 0xFF, G: 0xF, B: 0xF, A: 0x3F}
	DarkSeafoam  = color.RGBA{R: 46, G: 204, B: 113, A: 0xFF}
	DarkYellow   = color.RGBA{R: 204, G: 204, A: 0xFF}
	ForestGreen  = color.RGBA{R: 0xF, G: 0xFF, B: 0xF, A: 0x3F}
	Green        = color.RGBA{G: 0xFF, A: 0xFF}
	Seafoam      = color.RGBA{R: 46, G: 204, B: 113, A: 0xFF}
	Highlight    = color.RGBA{255, 255, 255, 255}
	LightPurple  = color.RGBA{204, 0, 204, 255}
	Orange       = color.RGBA{255, 165, 0, 255}
	Purple       = color.RGBA{83, 94, 255, 255}
	PaleRed      = color.RGBA{168, 49, 49, 255}
	Red          = color.RGBA{R: 0xFF, A: 0xFF}
	SlateGray    = color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0x3F}
	Yellow       = color.RGBA{R: 0xFF, G: 0xFF, A: 0xFF}
	White        = color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}
)

func Alpha(c color.RGBA, a uint8) color.RGBA {
	c.A = a
	return c
}

func Bool(b bool) color.RGBA {
	if b {
		return Green
	}

	return color.RGBA(SlateGray)
}

// Gray returns a new grayscale image
func Gray(img *image.RGBA) *image.Gray {
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
