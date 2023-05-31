package textblock

import (
	"image"
	"os"

	"gioui.org/font"
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
	"github.com/pidgy/unitehud/nrgba"
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
		collection: []text.FontFace{{Font: font.Font{Typeface: font.Typeface(face)}, Face: custom}},
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

	fill(gtx,
		nrgba.Background,
		func(gtx layout.Context) layout.Dimensions {
			return layout.Dimensions{Size: gtx.Constraints.Max}
		},
	)

	list := material.List(th, t.list)
	list.Track.Color = nrgba.Slate.Color()
	list.Track.Color.A = 0x0F
	layout.Inset{
		Bottom: unit.Dp(5),
		Left:   unit.Dp(5),
	}.Layout(
		gtx,
		func(gtx layout.Context) layout.Dimensions {
			return list.Layout(
				gtx,
				len(posts),
				func(gtx layout.Context, index int) layout.Dimensions {
					block := material.H5(th, posts[index].String())
					block.Color = posts[index].Color()
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
func colorBox(gtx layout.Context, size image.Point, bg nrgba.NRGBA) layout.Dimensions {
	defer clip.Rect{Max: size}.Push(gtx.Ops).Pop()
	paint.ColorOp{Color: bg.Color()}.Add(gtx.Ops)
	/*
		// XXX: Do we want gradient?
		paint.LinearGradientOp{
			Stop1:  f32.Pt(0, 0),
			Color1: bg.N(),
			Stop2:  image.Pt((size.X), float32(size.Y)),
			Color2: rgba.DarkBlue.N(),
		}.Add(gtx.Ops)
	*/
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

func fill(gtx layout.Context, bg nrgba.NRGBA, w layout.Widget) layout.Dimensions {
	colorBox(gtx, gtx.Constraints.Max, bg)
	return layout.NW.Layout(gtx, w)
}
