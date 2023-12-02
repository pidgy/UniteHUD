package fonts

import (
	"fmt"
	"os"

	"gioui.org/font"
	"gioui.org/font/gofont"
	"gioui.org/font/opentype"
	"gioui.org/text"
	"gioui.org/widget/material"

	"github.com/pidgy/unitehud/global"
	"github.com/pidgy/unitehud/notify"
)

type Collection map[string]*Style

type Style struct {
	Theme    *material.Theme
	Face     font.Face
	Typeface font.Typeface
	FontFace []font.FontFace
}

var (
	cache = NewCollection()
)

func NewCollection() Collection {
	return Collection(make(map[string]*Style))
}

func (c Collection) Cascadia() *Style {
	return c.load("CascadiaCode-Regular.otf", "Cascadia")
}

func (c Collection) Calibri() *Style {
	return c.load("CalibriRegular.ttf", "Calibri")
}

func (c Collection) NishikiTeki() *Style {
	return c.load("NishikiTeki-MVxaJ.ttf", "NishikiTeki")
}

func (c Collection) CascadiaSemiBold() *Style {
	return c.load("CascadiaCodePL-SemiBold.otf", "Cascadia")
}

func (c Collection) Combo() *Style {
	return c.load("Combo-Regular.ttf", "Combo")
}

func (c Collection) Hack() *Style {
	return c.load("Hack-Regular.ttf", "Hack")
}

func (c Collection) NotoSans() *Style {
	return c.load("NotoSansJP-Regular.otf", "NotoSansJP")
}

func (c Collection) Roboto() *Style {
	return c.load("Roboto-Regular.ttf", "Roboto")
}

func cached(name string) *Style {
	if cache[name] != nil {
		s := cache[name]

		style := &Style{
			Theme:    material.NewTheme(),
			FontFace: s.FontFace,
			Face:     s.Face,
			Typeface: font.Typeface(s.Typeface),
		}
		style.Theme.Shaper = text.NewShaper(text.WithCollection(s.FontFace))
	}

	return nil
}

func (c Collection) load(path, typeface string) *Style {
	if c[path] != nil {
		return c[path]
	}

	s := cached(path)
	if s != nil {
		notify.Debug("Font: Cached \"%s\"", typeface)
		c[path] = s
		return c[path]
	}

	notify.Debug("Font: Loading \"%s\"", typeface)

	bytes, err := os.ReadFile(fmt.Sprintf("%s/font/%s", global.AssetDirectory, path))
	if err != nil {
		notify.Warn("Font: %v", err)
		return noStyle()
	}

	custom, err := opentype.ParseCollection(bytes)
	if err != nil {
		notify.Warn("Font: %v", err)
		return noStyle()
	}

	face, err := opentype.Parse(bytes)
	if err != nil {
		notify.Warn("Font: %v", err)
		return noStyle()
	}

	cache[path] = &Style{
		Theme:    material.NewTheme(),
		FontFace: custom,
		Face:     face,
		Typeface: font.Typeface(typeface),
	}
	cache[path].Theme.Shaper = text.NewShaper(text.WithCollection(custom))

	c[path] = cache[path]

	return c[path]
}

func noStyle() *Style {
	style := &Style{Theme: material.NewTheme(), FontFace: gofont.Collection()}
	style.Theme.Shaper = text.NewShaper(text.WithCollection(gofont.Collection()))
	return style
}
