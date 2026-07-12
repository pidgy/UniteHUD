package area

import (
	"fmt"
	"image"
	"image/color"

	"github.com/pidgy/unitehud/core/notify"
	"github.com/pidgy/unitehud/media/video"
	"github.com/pidgy/unitehud/system/save"
)

// Capture describes a screen region and its saved preview metadata.
type Capture struct {
	Option      string
	File        string
	Base        image.Rectangle
	DefaultBase image.Rectangle

	MatchedColor color.NRGBA
	MatchedText  string
}

// Open captures the region to a PNG and opens it with the shell.
func (c *Capture) Open() error {
	img, err := video.CaptureRect(c.Base)
	if err != nil {
		return fmt.Errorf("<ini:f:capture> %s (%v)", c.File, err)
	}

	err = save.PNG(img, c.File)
	if err != nil {
		return fmt.Errorf("<ini:f:save> %s (%v)", c.File, err)
	}

	err = save.OpenImage(c.File)
	if err != nil {
		return fmt.Errorf("<ini:f:open> %s (%v)", c.File, err)
	}

	return nil
}

// Rectangle returns the capture region in screen coordinates.
func (c *Capture) Rectangle() image.Rectangle {
	return c.Base
}

// reset restores the capture region to its default bounds.
func (c *Capture) reset() {
	notify.Debug("[UI] Resetting %s capture area %s", c.Option, c.DefaultBase)
	c.Base = c.DefaultBase
}
