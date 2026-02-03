package monitor

// TODO: Add go style comments that reflect the purpose of each type, function, var, and const.

import (
	"fmt"
	"image"

	"golang.org/x/image/draw"

	"github.com/pidgy/unitehud/core/config"
	"github.com/pidgy/unitehud/core/notify"
	"github.com/pidgy/unitehud/system/wapi"
)

type monitor struct {
	index      int
	bounds     image.Rectangle
	name       string
	resolution string
	hwnd       uintptr
}

var (
	DefaultResolution = image.Rect(0, 0, 1920, 1080)

	Sources  = []string{config.MainDisplay}
	displays = []monitor{{name: config.MainDisplay, index: 0, bounds: DefaultResolution, resolution: fmt.Sprintf("%dx%d", DefaultResolution.Dx(), DefaultResolution.Dy())}}
)

func Capture() (*image.RGBA, error) {
	m, ok := active()
	if !ok {
		return nil, fmt.Errorf("\"%s\": invalid display", config.Current.Video.Capture.Window.Name)
	}

	img, err := wapi.Window(m.index).Capture(m.bounds, config.Current.Scale)
	if err != nil {
		return nil, fmt.Errorf("%s: %v", config.Current.Video.Capture.Window.Name, err)
	}

	if img.Rect.Max.X > DefaultResolution.Max.X && img.Rect.Max.Y > DefaultResolution.Max.Y {
		scaled := image.NewRGBA(DefaultResolution)
		draw.ApproxBiLinear.Scale(scaled, scaled.Rect, img, img.Bounds(), draw.Over, &draw.Options{})
		return scaled, nil
	}

	return img, nil
}

func CaptureRect(r image.Rectangle) (*image.RGBA, error) {
	img, err := Capture()
	if err != nil {
		return nil, err
	}

	return img.SubImage(r).(*image.RGBA), nil
}

func IsActive() bool {
	_, ok := active()
	return ok
}

func NameFromIndex(index int) string {
	for i, m := range displays {
		if i == index {
			return m.name
		}
	}
	return "Unknown Monitor"
}

func Open() {
	sourcesTmp := []string{}
	displaysTmp := []monitor{}

	leftDisplays := 0
	rightDisplays := 0
	topDisplays := 0
	bottomDisplays := 0

	ms, err := wapi.GetAllMonitors()
	if err != nil {
		notify.Error("[Monitor] <ini:f:find> monitors (%v)", err)
		return
	}

	for i, m := range ms.Active {
		name := ""
		r := m.Rect.Image()

		switch {
		case r.Eq(DefaultResolution):
			name = config.MainDisplay
		case i == 0 && r.Dx() > DefaultResolution.Dx() && r.Dy() > DefaultResolution.Dy():
			notify.System("[Monitor] <ini:i:rescaling> display #%d from %s to %s", i, r, DefaultResolution)
			name = config.MainDisplay
		case r.Min.X < DefaultResolution.Min.X:
			leftDisplays++
			name = display("Left", leftDisplays)
		case r.Min.X > DefaultResolution.Min.X:
			rightDisplays++
			name = display("Right", rightDisplays)
		case r.Min.Y < DefaultResolution.Min.Y:
			topDisplays++
			name = display("Top", topDisplays)
		case r.Min.Y > DefaultResolution.Min.Y:
			bottomDisplays++
			name = display("Bottom", bottomDisplays)
		default:
			notify.Error("[Monitor] <ini:f:locate> display #%d [%s] relative to %s [%s]", i, r, config.MainDisplay, DefaultResolution)
			continue
		}

		displaysTmp = append(displaysTmp, monitor{index: m.Index, hwnd: m.Handle, bounds: r, name: name, resolution: fmt.Sprintf("%dx%d", r.Dx(), r.Dy())})
		sourcesTmp = append(sourcesTmp, name)
	}

	Sources = sourcesTmp
	displays = displaysTmp
}

func Resolution() string {
	m, ok := active()
	if !ok {
		return "0x0"
	}
	return m.resolution
}

func TaskbarHeight() int {
	r, err := wapi.WorkArea()
	if err != nil {
		notify.Error("[Monitor] <ini:f:find> monitor info: %v", err)
		return 0
	}

	m, ok := active()
	if !ok {
		return 0
	}

	return m.bounds.Max.Y - int(r.Bottom)
}

func active() (monitor, bool) {
	for i, name := range Sources {
		if name == config.Current.Video.Capture.Window.Name {
			return displays[i], true
		}
	}
	return monitor{}, false
}

func display(name string, count int) string {
	if count <= 1 {
		return name
	}
	return fmt.Sprintf("%s #%d", name, count)
}

func (m monitor) String() string {
	return fmt.Sprintf("%s/Index %d (%s)", m.name, m.index, m.bounds)
}
