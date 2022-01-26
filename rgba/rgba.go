package rgba

import "image/color"

var (
	Black       = color.RGBA{0, 0, 0, 255}
	Green       = color.RGBA{G: 0xFF, A: 0xFF}
	Orange      = color.RGBA{255, 165, 0, 255}
	Purple      = color.RGBA{83, 94, 255, 255}
	Yellow      = color.RGBA{R: 0xFF, G: 0xFF, A: 0xFF}
	DarkYellow  = color.RGBA{R: 0xFF, G: 0xFF, A: 0x3F}
	White       = color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}
	Red         = color.RGBA{R: 0xFF, A: 0xFF}
	ForestGreen = color.NRGBA{R: 0xF, G: 0xFF, B: 0xF, A: 0x3F}
	SlateGray   = color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0x3F}
	DarkGray    = color.NRGBA{A: 0x4F}
	DarkerGray  = color.NRGBA{A: 0xF}
	DarkRed     = color.NRGBA{R: 0xFF, G: 0xF, B: 0xF, A: 0x3F}
)

func Bool(b bool) color.RGBA {
	if b {
		return Green
	}

	return Yellow
}
