package rgba

import "image/color"

var (
	Black        = color.RGBA{0, 0, 0, 255}
	Background   = color.NRGBA{R: 75, G: 75, B: 75, A: 255}
	DarkGray     = color.NRGBA{A: 0x4F}
	DarkerGray   = color.NRGBA{A: 0xF}
	DarkRed      = color.NRGBA{R: 0xFF, G: 0xF, B: 0xF, A: 0x3F}
	DarkYellow   = color.RGBA{R: 204, G: 204, A: 0xFF}
	DarkerYellow = color.RGBA{R: 0xFF, G: 0xFF, A: 0x3F}
	ForestGreen  = color.NRGBA{R: 0xF, G: 0xFF, B: 0xF, A: 0x3F}
	Green        = color.RGBA{G: 0xFF, A: 0xFF}
	Highlight    = color.RGBA{255, 255, 255, 255}
	LightPurple  = color.RGBA{204, 0, 204, 255}
	Orange       = color.RGBA{255, 165, 0, 255}
	Purple       = color.RGBA{83, 94, 255, 255}
	PaleRed      = color.RGBA{168, 49, 49, 255}
	Red          = color.RGBA{R: 0xFF, A: 0xFF}
	SlateGray    = color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0x3F}
	Yellow       = color.RGBA{R: 0xFF, G: 0xFF, A: 0xFF}
	White        = color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}
)

func Bool(b bool) color.RGBA {
	if b {
		return Green
	}

	return color.RGBA(SlateGray)
}

func Alpha(c color.RGBA, a uint8) color.RGBA {
	c.A = a
	return c
}
