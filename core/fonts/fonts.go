package fonts

import (
	"os"
	"path/filepath"

	"gioui.org/font"
	"gioui.org/font/gofont"
	"gioui.org/font/opentype"
	"gioui.org/text"
	"gioui.org/widget/material"

	"github.com/google/uuid"
	"github.com/pidgy/unitehud/core/notify"
	"github.com/pidgy/unitehud/exe"
)

type Collection struct {
	id     string
	styles map[string]*Style
	copied map[string]bool
}

type Style struct {
	Theme    *material.Theme
	Face     font.Face
	Typeface font.Typeface
	FontFace []font.FontFace
}

var (
	cache = NewCollection()
)

func NewCollection() *Collection {
	return &Collection{
		id:     uuid.New().String(),
		styles: make(map[string]*Style),
		copied: make(map[string]bool),
	}
}

func (c *Collection) Cascadia() *Style {
	return c.load("CascadiaCode-Regular.otf", "Cascadia")
}

func (c *Collection) Calibri() *Style {
	return c.load("CalibriRegular.ttf", "Calibri")
}

func (c *Collection) NishikiTeki() *Style {
	return c.load("NishikiTeki-MVxaJ.ttf", "NishikiTeki")
}

func (c *Collection) CascadiaSemiBold() *Style {
	return c.load("CascadiaCodePL-SemiBold.otf", "Cascadia")
}

func (c *Collection) Combo() *Style {
	return c.load("Combo-Regular.ttf", "Combo")
}

func (c *Collection) Hack() *Style {
	return c.load("Hack-Regular.ttf", "Hack")
}

func (c *Collection) NotoSans() *Style {
	return c.load("NotoSansJP-Regular.otf", "NotoSansJP")
}

func (c *Collection) Roboto() *Style {
	return c.load("Roboto-Regular.ttf", "Roboto")
}

func (s *Style) copy() *Style {
	// if c.copied[string(s.Typeface)] {
	// 	return s
	// }

	s2 := &Style{
		Theme:    material.NewTheme(),
		FontFace: s.FontFace,
		Face:     s.Face,
		Typeface: s.Typeface,
	}
	s2.Theme.Shaper = text.NewShaper(text.WithCollection(s2.FontFace))

	// c.styles[string(s.Typeface)] = s2

	return s2
}

func cached(name string) *Style {
	if cache.styles[name] != nil {
		return cache.styles[name]
	}

	return nil
}

func (c *Collection) load(path, typeface string) *Style {
	if c.styles[path] != nil {
		return c.styles[path]
	}

loaded:
	s := cached(path)
	if s != nil {
		notify.Debug("[Font] Cached: %s", typeface)

		if c.styles[path] == nil {
			c.styles[path] = s.copy()
		}

		return c.styles[path]
	}

	notify.Debug("[Font] Loading: %s", typeface)

	bytes, err := os.ReadFile(filepath.Join(exe.AssetDirectory, "font", path))
	if err != nil {
		notify.Warn("[Font] %v", err)
		return noStyle()
	}

	fontFace, err := opentype.ParseCollection(bytes)
	if err != nil {
		notify.Warn("[Font] %v", err)
		return noStyle()
	}

	face, err := opentype.Parse(bytes)
	if err != nil {
		notify.Warn("[Font] %v", err)
		return noStyle()
	}

	cache.styles[path] = &Style{
		Theme:    material.NewTheme(),
		FontFace: fontFace,
		Face:     face,
		Typeface: font.Typeface(typeface),
	}
	cache.styles[path].Theme.Shaper = text.NewShaper(text.WithCollection(fontFace))

	goto loaded
}

func noStyle() *Style {
	style := &Style{Theme: material.NewTheme(), FontFace: gofont.Collection()}
	style.Theme.Shaper = text.NewShaper(text.WithCollection(gofont.Collection()))
	return style
}
