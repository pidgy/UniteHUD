package monitor

// TODO: Add go style comments that reflect the purpose of each type, function, var, and const. Then remove this comment.

import (
	"fmt"
	"image"
	"slices"
	"strings"

	"github.com/pidgy/unitehud/core/config"
	"github.com/pidgy/unitehud/core/notify"
	"github.com/pidgy/unitehud/system/win32"
	"github.com/pidgy/unitehud/system/win32/d3d11"
)

type monitor struct {
	win32.Monitor
	bounds     image.Rectangle
	name       string
	resolution image.Point
}

var (
	DefaultResolution   = image.Rect(0, 0, 1920, 1080)
	DefaultResolution32 = win32.Rect{Left: 0, Top: 0, Right: 1920, Bottom: 1080}

	Sources  = []string{config.MainDisplay}
	displays = []monitor{{name: config.MainDisplay, Monitor: win32.Monitor{Index: 0}, resolution: DefaultResolution.Size()}}

	d3d *d3d11.Desktop
)

func ActiveName() string {
	m, _ := active()
	return m.name
}

func BoundsFromCoordinate(x int) image.Rectangle {
	for _, d := range displays {
		c := image.Pt(x, d.bounds.Min.Y)
		if c.In(d.bounds) {
			return d.bounds
		}
	}
	return DefaultResolution
}

func Bounds() image.Rectangle {
	m, ok := active()
	if !ok {
		return DefaultResolution
	}
	return m.bounds
}

func Capture() (*image.RGBA, error) {
	m, ok := active()
	if !ok {
		return nil, fmt.Errorf("%s: invalid display", m)
	}

	switch config.Current.Video.Capture.Monitor.Method {
	case config.CaptureMethodDefault:
		if d3d != nil {
			notify.Debug("[Monitor] Closing %s capture", config.Current.Video.Capture.Monitor.Method)

			d3d.Close()
			d3d = nil
		}
		return m.Capture(m.bounds, DefaultResolution.Size())
	case config.CaptureMethodDirectX11:
		if d3d == nil {
			notify.Debug("[Monitor] Starting %s capture", config.Current.Video.Capture.Monitor.Method)

			c, err := d3d11.NewDesktop(m.HWND)
			if err != nil {
				return nil, err
			}
			d3d = c
		}

		return d3d.Capture(m.bounds)
	default:
		return nil, fmt.Errorf("unknown window capture method")
	}
}

func CaptureRect(r image.Rectangle) (*image.RGBA, error) {
	img, err := Capture()
	if err != nil {
		return nil, err
	}

	return img.SubImage(r).(*image.RGBA), nil
}

func Close() {
	config.Current.SetDefaultMonitorCapture()

	if d3d != nil {
		d3d.Close()
		d3d = nil
	}
}

func IsActive() bool {
	_, ok := active()
	return ok
}

func Name(index int) string {
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

	ms, err := win32.FindMonitors()
	if err != nil {
		notify.Error("[Monitor] <ini:f:find> monitors (%v)", err)
		return
	}

	for i := 0; i < ms.Count; i++ {
		m := ms.Active[i]
		name := ""
		r := m.Rect.Image()
		switch {
		case i == 0:
			name = config.MainDisplay

			if !r.Eq(DefaultResolution) {
				notify.System("[Monitor] <ini:i:rescaling> display #%d from %s to %s", i, r, DefaultResolution)
			}
		case r.Min.X < DefaultResolution.Min.X:
			leftDisplays++
			name = display("Left Display", leftDisplays)
		case r.Min.X > DefaultResolution.Min.X:
			rightDisplays++
			name = display("Right Display", rightDisplays)
		case r.Min.Y < DefaultResolution.Min.Y:
			topDisplays++
			name = display("Top Display", topDisplays)
		case r.Min.Y > DefaultResolution.Min.Y:
			bottomDisplays++
			name = display("Bottom Display", bottomDisplays)
		default:
			notify.Error("[Monitor] <ini:f:locate> display #%d [%s] relative to %s [%s]", i, r, config.MainDisplay, DefaultResolution)
			continue
		}

		displaysTmp = append(displaysTmp, monitor{Monitor: ms.Active[i], bounds: r, name: name, resolution: r.Size()})
		sourcesTmp = append(sourcesTmp, name)
	}

	Sources = sourcesTmp
	displays = displaysTmp

	notify.Debug("[Monitor] Sources: [\"%s\"]", strings.Join(Sources, `", "`))
}

func Resolution() image.Point {
	m, ok := active()
	if !ok {
		return image.Pt(0, 0)
	}
	return m.resolution
}

func TaskbarHeight() int {
	r, err := win32.WorkArea()
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

func active() (m monitor, ok bool) {
	i := slices.Index(Sources, config.Current.Video.Capture.Monitor.Name)
	if i == -1 {
		return monitor{}, false
	}
	return displays[i], true
}

func display(name string, count int) string {
	if count <= 1 {
		return name
	}
	return fmt.Sprintf("%s #%d", name, count)
}

func (m monitor) String() string {
	return fmt.Sprintf("%s/%d/%d %s", m.name, m.Index, m.Monitor.HWND, m.bounds)
}
