package textblock

import (
	"image"
	"image/color"
	"os"

	"gioui.org/font/gofont"
	"gioui.org/font/opentype"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	"github.com/pidgy/unitehud/notify"
	"github.com/pidgy/unitehud/rgba"
)

type TextBlock struct {
	Text string
	list *widget.List

	collection []text.FontFace
}

func (t *TextBlock) Collection() []text.FontFace {
	return t.collection
}

func NewCascadiaCode() (*TextBlock, error) {
	return New("fonts/CascadiaCode-Regular.otf", "Cascadia")
}

func NewCascadiaCodeSemiBold() (*TextBlock, error) {
	return New("fonts/CascadiaCodePL-SemiBold.otf", "Cascadia")
}

func NewCombo() (*TextBlock, error) {
	return New("fonts/Combo-Regular.ttf", "Combo")
}

func NewNotoSans() (*TextBlock, error) {
	return New("fonts/NotoSansJP-Regular.otf", "NotoSansJP")
}

func NewRoboto() (*TextBlock, error) {
	return New("fonts/Roboto-Regular.ttf", "Roboto")
}

func New(file, face string) (*TextBlock, error) {
	bytes, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}

	custom, err := opentype.Parse(bytes)
	if err != nil {
		return nil, err
	}

	return &TextBlock{
		collection: []text.FontFace{{Font: text.Font{Typeface: text.Typeface(face)}, Face: custom}},
	}, nil
}

func (t *TextBlock) Layout(gtx layout.Context, posts []notify.Post) layout.Dimensions {
	if t.list == nil {
		t.list = &widget.List{
			Scrollbar: widget.Scrollbar{},
			List: layout.List{
				Axis:        layout.Vertical,
				ScrollToEnd: true,
				Alignment:   layout.Baseline,
			},
		}
	}

	th := material.NewTheme(t.collection)
	if len(t.collection) == 0 {
		t.collection = gofont.Collection()
	}
	th.TextSize = unit.Sp(9)

	Fill(gtx,
		rgba.N(rgba.Background),
		func(gtx layout.Context) layout.Dimensions {
			return layout.Dimensions{Size: gtx.Constraints.Max}
		},
	)

	list := material.List(th, t.list)
	list.Track.Color = color.NRGBA(rgba.Slate)
	list.Track.Color.A = 0x0F
	layout.Inset{
		Bottom: unit.Px(5),
		Left:   unit.Px(5),
	}.Layout(
		gtx,
		func(gtx layout.Context) layout.Dimensions {
			return list.Layout(
				gtx,
				len(posts),
				func(gtx layout.Context, index int) layout.Dimensions {
					block := material.H5(th, posts[index].String())
					block.Color = color.NRGBA(posts[index].RGBA)
					block.Alignment = text.Alignment(text.Start)
					dim := block.Layout(gtx)
					dim.Size.X = gtx.Constraints.Max.X
					return dim
				},
			)
		},
	)

	return layout.Dimensions{Size: gtx.Constraints.Max}
}

// colorBox creates a widget with the specified dimensions and color.
func colorBox(gtx layout.Context, size image.Point, c color.NRGBA) layout.Dimensions {
	defer clip.Rect{Max: size}.Push(gtx.Ops).Pop()
	paint.ColorOp{Color: c}.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)

	return widget.Border{
		Color: color.NRGBA{R: 100, G: 100, B: 100, A: 50},
		Width: unit.Px(2),
	}.Layout(
		gtx,
		func(gtx layout.Context) layout.Dimensions {
			return layout.Dimensions{Size: size}
		})
}

func Fill(gtx layout.Context, backgroundColor color.NRGBA, w layout.Widget) layout.Dimensions {
	colorBox(gtx, gtx.Constraints.Max, backgroundColor)
	return layout.NW.Layout(gtx, w)
}
