package help

import (
	"image"
	"image/png"
	"os"

	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/pidgy/unitehud/fonts"
	"github.com/pidgy/unitehud/nrgba"
	"golang.org/x/image/draw"
)

func Configuration() *configuration {
	return &configuration{
		Help: &Help{
			Page:  0,
			Pages: len(dialog),
		},
	}
}

type configuration struct {
	*Help
}

var dialog = []string{
	"Drag the appropriate shapes on your screen to assign selection areas for specific image processing.",
	"When the shapes turn green, they have successfully matched against UI updates.",
	"If the game resolution is less than 1920x1080, adjust the scale to match accordingly.",
	"Selection areas may safely overlap eachother and not interfere with the matching process.",
	"When you are finished configuring, select \"Save\" or \"Cancel\" to preserve or dismiss your setup.",
}

var images = []string{
	"img/help/config_score.png",
	"img/help/config_time.png",
	"img/help/config_score_scale.png",
	"img/help/config_overlap.png",
	"img/help/config_save_cancel.png",
}

func (c *configuration) Layout(gtx layout.Context) layout.Dimensions {
	collection := fonts.NewCollection()

	fill(gtx,
		nrgba.DarkBlue,
		func(gtx layout.Context) layout.Dimensions {
			return layout.Dimensions{Size: gtx.Constraints.Max}
		},
	)

	txt := material.H5(collection.Calibri().Theme, dialog[c.Page])
	txt.Color = nrgba.White.Color()
	txt.Alignment = text.Middle
	txt.TextSize = unit.Sp(14)

	layout.Inset{
		Top:  unit.Dp(10),
		Left: unit.Dp(10),
	}.Layout(
		gtx,
		func(gtx layout.Context) layout.Dimensions {
			return txt.Layout(gtx)
		},
	)

	img := img(images[c.Page], gtx.Constraints.Max)
	if img == nil {
		return layout.Dimensions{Size: gtx.Constraints.Max}
	}

	layout.Inset{
		Top:  unit.Dp(35),
		Left: unit.Dp(20),
	}.Layout(
		gtx,
		func(gtx layout.Context) layout.Dimensions {
			defer clip.Rect{Max: img.Bounds().Max}.Push(gtx.Ops).Pop()

			return widget.Border{
				Color: nrgba.White.Color(),
				Width: unit.Dp(2),
			}.Layout(
				gtx,
				func(gtx layout.Context) layout.Dimensions {
					paint.NewImageOp(img).Add(gtx.Ops)
					paint.PaintOp{}.Add(gtx.Ops)
					return layout.Dimensions{Size: img.Bounds().Max}
				},
			)
		},
	)

	return layout.Dimensions{Size: gtx.Constraints.Max}

}

func img(name string, max image.Point) image.Image {
	f, err := os.Open(name)
	if err != nil {
		return nil
	}

	img, err := png.Decode(f)
	if err != nil {
		return nil
	}

	if img.Bounds().Max.X > max.X || img.Bounds().Max.Y > max.Y {
		dst := image.NewRGBA(image.Rect(0, 0, int(float32(img.Bounds().Max.X)*0.75), int(float32(img.Bounds().Max.Y)*0.75)))
		draw.NearestNeighbor.Scale(dst, dst.Rect, img, img.Bounds(), draw.Over, nil)

		return dst
	}

	return img
}

func colorBox(gtx layout.Context, size image.Point, n nrgba.NRGBA) layout.Dimensions {
	defer clip.Rect{Max: size}.Push(gtx.Ops).Pop()
	paint.ColorOp{Color: n.Color()}.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)

	return widget.Border{
		Color: nrgba.LightGray.Color(),
		Width: unit.Dp(2),
	}.Layout(
		gtx,
		func(gtx layout.Context) layout.Dimensions {
			return layout.Dimensions{Size: size}
		})
}

func fill(gtx layout.Context, n nrgba.NRGBA, w layout.Widget) layout.Dimensions {
	colorBox(gtx, gtx.Constraints.Max, n)
	return layout.NW.Layout(gtx, w)
}
