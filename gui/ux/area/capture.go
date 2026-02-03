package area

// TODO: Add go style comments that reflect the purpose of each type, function, var, and const.

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"syscall"

	"github.com/pidgy/unitehud/core/notify"
	"github.com/pidgy/unitehud/exe"
	"github.com/pidgy/unitehud/media/video"
	"github.com/pidgy/unitehud/system/wapi"
)

// Capture defines Capture behavior and state.
type Capture struct {
	Option      string
	File        string
	Base        image.Rectangle
	DefaultBase image.Rectangle

	MatchedColor color.NRGBA
	MatchedText  string
}

// Open opens the target.
func (c *Capture) Open() error {
	img, err := video.CaptureRect(c.Base)
	if err != nil {
		return fmt.Errorf("<ini:f:capture> %s (%v)", c.File, err)
	}

	fd, err := os.Create(c.File)
	if err != nil {
		return err
	}
	defer fd.Close()

	err = png.Encode(fd, img)
	if err != nil {
		return fmt.Errorf("Failed to create %s (%v)", c.File, err)
	}

	argv, err := syscall.UTF16PtrFromString(os.Getenv("windir") + "\\system32\\cmd.exe /C " + fmt.Sprintf("\"%s\\%s\"", exe.Directory(), c.File))
	if err != nil {
		return fmt.Errorf("<ini:f:open> %s (%v)", c.File, err)
	}

	var sI syscall.StartupInfo
	var pI syscall.ProcessInformation

	err = syscall.CreateProcess(nil, argv, nil, nil, true, wapi.CreateProcessFlags.NoWindow, nil, nil, &sI, &pI)
	if err != nil {
		return fmt.Errorf("<ini:f:open> %s (%v)", c.File, err)
	}

	return nil
}

func (c *Capture) Rectangle() image.Rectangle {
	return c.Base
}

func (c *Capture) reset() {
	notify.Debug("[UI] Resetting %s capture area %s", c.Option, c.DefaultBase)
	c.Base = c.DefaultBase
}
