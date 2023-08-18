package fonts

import (
	"os"

	"gioui.org/font"
	"gioui.org/font/gofont"
	"gioui.org/font/opentype"
	"gioui.org/widget/material"

	"github.com/pidgy/unitehud/notify"
)

type Style struct {
	Theme    *material.Theme
	Face     font.Face
	Typeface font.Typeface
	FontFace []font.FontFace
}

var (
	calibri          *Style
	cascadia         *Style
	cascadiaSemiBold *Style
	combo            *Style
	gio              *Style = &Style{Theme: material.NewTheme(gofont.Collection()), FontFace: gofont.Collection()}
	hack             *Style
	notoSans         *Style
	nishikiTeki      *Style
	roboto           *Style
)

func Calibri() *Style {
	if calibri == nil {
		calibri = load("CalibriRegular.ttf", "Calibri")
	}
	return calibri
}

func Cascadia() *Style {
	if cascadia == nil {
		cascadia = load("CascadiaCode-Regular.otf", "Cascadia")
	}
	return cascadia
}

func CascadiaSemiBold() *Style {
	if cascadiaSemiBold == nil {
		cascadiaSemiBold = load("CascadiaCodePL-SemiBold.otf", "Cascadia")
	}
	return cascadiaSemiBold
}

func Combo() *Style {
	if combo == nil {
		combo = load("Combo-Regular.ttf", "Combo")
	}
	return combo
}

func Default() *Style {
	return Calibri()
}

func GIO() *Style {
	return gio
}

func Hack() *Style {
	if hack == nil {
		hack = load("Hack-Regular.ttf", "Hack")
	}
	return hack
}

func NotoSans() *Style {
	if notoSans == nil {
		notoSans = load("NotoSansJP-Regular.otf", "NotoSansJP")
	}
	return notoSans
}

func NishikiTeki() *Style {
	if nishikiTeki == nil {
		nishikiTeki = load("NishikiTeki-MVxaJ.ttf", "NishikiTeki")
	}
	return nishikiTeki
}

func Roboto() *Style {
	if roboto == nil {
		roboto = load("Roboto-Regular.ttf", "Roboto")
	}
	return roboto
}

func load(path, typeface string) *Style {
	bytes, err := os.ReadFile("assets/font/" + path)
	if err != nil {
		notify.Warn("%v", err)
		return gio
	}

	custom, err := opentype.ParseCollection(bytes)
	if err != nil {
		notify.Warn("%v", err)
		return gio
	}

	face, err := opentype.Parse(bytes)
	if err != nil {
		notify.Warn("%v", err)
		return gio
	}

	return &Style{
		Theme:    material.NewTheme(custom),
		FontFace: custom,
		Face:     face,
		Typeface: font.Typeface(typeface),
	}
}
