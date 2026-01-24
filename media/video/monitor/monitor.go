package monitor

import (
	"fmt"
	"image"

	"github.com/kbinani/screenshot"
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

func Active(name string) bool {
	return IsDisplay() && name == config.Current.Video.Capture.Window.Name
}

func NameFromIndex(index int) string {
	for i, m := range displays {
		if i == index {
			return m.name
		}
	}
	return "Unknown Monitor"
}

func Capture() (*image.RGBA, error) {
	m, ok := active()
	if !ok {
		return nil, fmt.Errorf("\"%s\": invalid display", config.Current.Video.Capture.Window.Name)
	}

	img, err := captureMonitor(m.bounds)
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

func CaptureRect(rect image.Rectangle) (*image.RGBA, error) {
	img, err := Capture()
	if err != nil {
		return nil, err
	}

	return img.SubImage(rect).(*image.RGBA), nil
}

// func CaptureRect(rect image.Rectangle) (*image.RGBA, error) {
// mutex.RLock()
// b := bounds[config.Current.Video.Capture.Window.Name]
// mutex.RUnlock()

// rect.Min.X = b.Min.X + rect.Min.X
// rect.Max.X = b.Min.X + rect.Max.X

// rect.Min.Y = b.Min.Y + rect.Min.Y
// rect.Max.Y = b.Min.Y + rect.Max.Y

// src, err := wapi.Window(0).Device()
// if err != nil {
// 	return nil, errors.Wrap(err, "device")
// }
// defer src.Release()

// dst, err := src.Compatible()
// if err != nil {
// 	return nil, fmt.Errorf("could not create compatible DC (%d)", lastError())
// }
// defer dst.Delete()

// size := rect.Size()

// info := wapi.BitmapInfo{
// 	BmiHeader: wapi.BitmapInfoHeader{
// 		BiSize:        wapi.BitmapInfoHeaderSize,
// 		BiWidth:       int32(size.X),
// 		BiHeight:      -int32(size.Y),
// 		BiPlanes:      1,
// 		BiBitCount:    32,
// 		BiCompression: wapi.BitmapInfoHeaderCompression.RGB,
// 	},
// }

// bitmap, raw, err := info.CreateRGBSection(&dst)
// if err != nil {
// 	return nil, errors.Wrap(err, "section")
// }
// defer bitmap.Delete()

// obj, err := dst.Select(bitmap)
// if err != nil {
// 	return nil, errors.Wrap(err, "bitmap select")
// }
// defer obj.Delete()

// err = dst.Copy(src, size, rect, config.Current.Scale)
// if err != nil {
// 	return nil, errors.Wrap(err, "bitmap copy")
// }

// data := raw.Slice(size)
// pix := make([]byte, len(data))

// for i := 0; i < len(pix); i += 4 {
// 	pix[i], pix[i+2], pix[i+1], pix[i+3] = byte(data[i+2]), byte(data[i]), byte(data[i+1]), byte(data[i+3])
// }

// return &image.RGBA{
// 	Pix:    pix,
// 	Stride: 4 * size.X,
// 	Rect:   image.Rect(0, 0, size.X, size.Y),
// }, nil
//}

func IsDisplay() bool {
	_, ok := active()
	return ok
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

	// for i := 0; i < screenshot.NumActiveDisplays(); i++ {
	// 	name := ""
	// 	r := screenshot.GetDisplayBounds(i)

	// 	switch {
	// 	case r.Eq(DefaultResolution):
	// 		name = config.MainDisplay
	// 	case i == 0 && r.Dx() > DefaultResolution.Dx() && r.Dy() > DefaultResolution.Dy():
	// 		notify.System("[Monitor] <ini:i:rescaling> display #%d from %s to %s", i, r, DefaultResolution)
	// 		name = config.MainDisplay
	// 	case r.Min.X < DefaultResolution.Min.X:
	// 		leftDisplays++
	// 		name = display("Left", leftDisplays)
	// 	case r.Min.X > DefaultResolution.Min.X:
	// 		rightDisplays++
	// 		name = display("Right", rightDisplays)
	// 	case r.Min.Y < DefaultResolution.Min.Y:
	// 		topDisplays++
	// 		name = display("Top", topDisplays)
	// 	case r.Min.Y > DefaultResolution.Min.Y:
	// 		bottomDisplays++
	// 		name = display("Bottom", bottomDisplays)
	// 	default:
	// 		notify.Error("[Monitor] <ini:f:locate> display #%d [%s] relative to %s [%s]", i, r, config.MainDisplay, DefaultResolution)
	// 		continue
	// 	}

	// 	displaysTmp = append(displaysTmp, monitor{index: i, bounds: r, name: name, resolution: fmt.Sprintf("%dx%d", r.Dx(), r.Dy())})
	// 	sourcesTmp = append(sourcesTmp, name)
	// }

	Sources = sourcesTmp
	displays = displaysTmp
}

func Open2() {
	sourcesTmp := []string{}
	displaysTmp := []monitor{}

	leftDisplays := 0
	rightDisplays := 0
	topDisplays := 0
	bottomDisplays := 0

	for i := 0; i < screenshot.NumActiveDisplays(); i++ {
		name := ""
		r := screenshot.GetDisplayBounds(i)

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

		displaysTmp = append(displaysTmp, monitor{index: i, bounds: r, name: name, resolution: fmt.Sprintf("%dx%d", r.Dx(), r.Dy())})
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

// func captureFullscreen() (*image.RGBA, error) {
// 	m, ok := active()
// 	if !ok {
// return nil, fmt.Errorf("%s: invalid display", config.Current.Video.Capture.Window.Name)
// 	}

// 	img, err := screenshot.CaptureDisplay(m.index)
// 	if err != nil {
// 		return nil, fmt.Errorf("%s/%s: %v", config.Current.Video.Capture.Window.Name, m, err)
// 	}

// 	if img.Rect.Max.X > DefaultResolution.Max.X && img.Rect.Max.Y > DefaultResolution.Max.Y {
// 		scaled := image.NewRGBA(DefaultResolution)
// 		draw.NearestNeighbor.Scale(scaled, scaled.Rect, img, img.Bounds(), draw.Over, &draw.Options{})
// 		return scaled, nil
// 	}

// 	return img, nil

// }

func active() (monitor, bool) {
	for i, name := range Sources {
		if name == config.Current.Video.Capture.Window.Name {
			return displays[i], true
		}
	}
	return monitor{}, false
}

func captureMonitor(rect image.Rectangle) (*image.RGBA, error) {
	m, ok := active()
	if !ok {
		return nil, fmt.Errorf("%s: invalid display", config.Current.Video.Capture.Window.Name)
	}

	// rect.Min.X = b.Min.X + rect.Min.X
	// rect.Max.X = b.Min.X + rect.Max.X

	// rect.Min.Y = b.Min.Y + rect.Min.Y
	// rect.Max.Y = b.Min.Y + rect.Max.Y

	src, err := wapi.Window(m.index).Device()
	if err != nil {
		return nil, fmt.Errorf("%s: %v", m, err)
	}
	defer src.Release()

	dst, err := src.Compatible()
	if err != nil {
		return nil, fmt.Errorf("%s: incompatible (%s)", m, wapi.GetLastError())
	}
	defer dst.Delete()

	size := rect.Size()

	info := wapi.BitmapInfo{
		BmiHeader: wapi.BitmapInfoHeader{
			BiSize:        wapi.BitmapInfoHeaderSize,
			BiWidth:       int32(size.X),
			BiHeight:      -int32(size.Y),
			BiPlanes:      1,
			BiBitCount:    32,
			BiCompression: wapi.BitmapInfoHeaderCompression.RGB,
		},
	}

	bitmap, raw, err := info.CreateRGBSection(&dst)
	if err != nil {
		return nil, fmt.Errorf("%s: RGB section %v", m, err)
	}
	defer bitmap.Delete()

	obj, err := dst.Select(bitmap)
	if err != nil {
		return nil, fmt.Errorf("%s: bitmap Select %v", m, err)
	}
	defer obj.Delete()

	err = dst.Copy(src, size, rect, config.Current.Scale)
	if err != nil {
		return nil, fmt.Errorf("%s: bitmap copy %v", m, err)
	}

	data := raw.Slice(size)

	pix := make([]byte, len(data))

	for i := 0; i < len(pix); i += 4 {
		pix[i], pix[i+2], pix[i+1], pix[i+3] = byte(data[i+2]), byte(data[i]), byte(data[i+1]), byte(data[i+3])
	}

	return &image.RGBA{
		Pix:    pix,
		Stride: 4 * size.X,
		Rect:   image.Rect(0, 0, size.X, size.Y),
	}, nil
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
