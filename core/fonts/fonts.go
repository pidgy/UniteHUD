package fonts

// TODO: Add go style comments that reflect the purpose of each type, function, var, and const.

import (
	"os"
	"path/filepath"

	"gioui.org/font"
	"gioui.org/font/gofont"
	"gioui.org/font/opentype"
	"gioui.org/text"
	"gioui.org/widget/material"
	"github.com/google/uuid"

	"github.com/pidgy/unitehud/core/config"
	"github.com/pidgy/unitehud/core/notify"
	"github.com/pidgy/unitehud/exe"
)

// Collection manages a set of loaded font styles.
type Collection struct {
	id     string
	styles map[string]*Style
	copied map[string]bool
}

// Style holds a themed font face and related metadata.
type Style struct {
	Theme    *material.Theme
	Face     font.Face
	Typeface font.Typeface
	FontFace []font.FontFace
}

var (
	// cache is the shared font collection used for reuse.
	cache = NewCollection()
)

// NewCollection returns a new font collection.
func NewCollection() *Collection {
	return &Collection{
		id:     uuid.New().String(),
		styles: make(map[string]*Style),
		copied: make(map[string]bool),
	}
}

// Cascadia returns the Cascadia Code font style.
func (c *Collection) Cascadia() *Style {
	return c.load("CascadiaCode-Regular.otf", "Cascadia")
}

// Calibri returns the Calibri font style.
func (c *Collection) Calibri() *Style {
	return c.load("CalibriRegular.ttf", "Calibri")
}

// NishikiTeki returns the NishikiTeki font style.
func (c *Collection) NishikiTeki() *Style {
	return c.load("NishikiTeki-MVxaJ.ttf", "NishikiTeki")
}

// CascadiaSemiBold returns the Cascadia Code SemiBold font style.
func (c *Collection) CascadiaSemiBold() *Style {
	return c.load("CascadiaCodePL-SemiBold.otf", "Cascadia")
}

// Combo returns the Combo font style.
func (c *Collection) Combo() *Style {
	return c.load("Combo-Regular.ttf", "Combo")
}

// Hack returns the Hack font style.
func (c *Collection) Hack() *Style {
	return c.load("Hack-Regular.ttf", "Hack")
}

// NotoSans returns the Noto Sans font style.
func (c *Collection) NotoSans() *Style {
	return c.load("NotoSansJP-Regular.otf", "NotoSansJP")
}

// Roboto returns the Roboto font style.
func (c *Collection) Roboto() *Style {
	return c.load("Roboto-Regular.ttf", "Roboto")
}

// load loads the requested data.
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

	cache.styles[path] = (&Style{
		Theme:    material.NewTheme(),
		FontFace: fontFace,
		Face:     face,
		Typeface: font.Typeface(typeface),
	}).WithTheme()
	cache.styles[path].Theme.Shaper = text.NewShaper(text.WithCollection(fontFace))

	goto loaded
}

// WithTheme applies the current config theme to the style.
func (s *Style) WithTheme() *Style {
	s.Theme.Bg = config.Current.Theme.Background
	s.Theme.ContrastBg = config.Current.Theme.BackgroundAlt
	s.Theme.Fg = config.Current.Theme.Foreground
	s.Theme.ContrastFg = config.Current.Theme.ForegroundAlt
	return s
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

// cached returns a cached style by name if it exists.
func cached(name string) *Style {
	if cache.styles[name] != nil {
		return cache.styles[name]
	}

	return nil
}

// noStyle returns a fallback style based on Go's default fonts.
func noStyle() *Style {
	s := &Style{Theme: material.NewTheme(), FontFace: gofont.Collection()}
	s.Theme.Shaper = text.NewShaper(text.WithCollection(gofont.Collection()))
	return s.WithTheme()
}
