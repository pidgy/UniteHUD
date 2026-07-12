package monitor

// TODO: Add go style comments that reflect the purpose of each type, function, var, and const. Then remove this comment.

import (
	"fmt"
	"image"
	"slices"
	"strings"

	"github.com/pidgy/unitehud/core/config"
	"github.com/pidgy/unitehud/core/notify"
	"github.com/pidgy/unitehud/system/wapi"
)

type monitor struct {
	wapi.Monitor
	bounds     image.Rectangle
	name       string
	resolution image.Point
}

var (
	DefaultResolution = image.Rect(0, 0, 1920, 1080)

	Sources  = []string{config.MainDisplay}
	displays = []monitor{{name: config.MainDisplay, Monitor: wapi.Monitor{Index: 0}, resolution: DefaultResolution.Size()}}
)

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

	return CaptureRect(m.bounds)
}

func CaptureRect(r image.Rectangle) (*image.RGBA, error) {
	m, ok := active()
	if !ok {
		return nil, fmt.Errorf("%s: invalid display", m)
	}

	return m.Capture(m.bounds, r)
}

// func CaptureRect2(r image.Rectangle) (*image.RGBA, error) {
// 	m, ok := active()
// 	if !ok {
// 		return nil, fmt.Errorf("%s: invalid display", m)
// 	}

// 	img, err := wapi.Window(m.hwnd).Capture(r, config.Current.Scale)
// 	if err != nil {
// 		return nil, fmt.Errorf("%s: %v", m, err)
// 	}

// 	if img.Rect.Max.X > DefaultResolution.Max.X && img.Rect.Max.Y > DefaultResolution.Max.Y {
// 		scaled := image.NewRGBA(DefaultResolution)
// 		draw.ApproxBiLinear.Scale(scaled, scaled.Rect, img, img.Bounds(), draw.Over, &draw.Options{})
// 		return scaled, nil
// 	}

// 	return img, nil
// }

// func CaptureRect3(rect image.Rectangle) (*image.RGBA, error) {
// 	m, ok := active()
// 	if !ok {
// 		return nil, fmt.Errorf("%s: invalid display", m)
// 	}

// 	rect.Min.X = m.bounds.Min.X + rect.Min.X
// 	rect.Max.X = m.bounds.Min.X + rect.Max.X

// 	rect.Min.Y = m.bounds.Min.Y + rect.Min.Y
// 	rect.Max.Y = m.bounds.Min.Y + rect.Max.Y

// 	src, _, _ := wapi.GetDC.Call(0)
// 	if src == 0 {
// 		return nil, fmt.Errorf("Failed to find primary display (%d)", wapi.GetLastError())
// 	}
// 	defer wapi.ReleaseDC.Call(m.hwnd, src)

// 	dst, _, _ := wapi.CreateCompatibleDC.Call(src)
// 	if dst == 0 {
// 		return nil, fmt.Errorf("Could not Create Compatible DC (%d)", wapi.GetLastError())
// 	}
// 	defer wapi.DeleteDC.Call(dst) // nolint

// 	x, y := rect.Dx(), rect.Dy()

// 	bt := wapi.BitmapInfo{}
// 	bt.BmiHeader = wapi.BitmapInfoHeader{
// 		BiSize:        uint32(reflect.TypeOf(bt.BmiHeader).Size()),
// 		BiWidth:       int32(x),
// 		BiHeight:      int32(-y),
// 		BiPlanes:      1,
// 		BiBitCount:    32,
// 		BiCompression: wapi.BitmapInfoHeaderCompression.RGB,
// 	}

// 	ptr := uintptr(0)

// 	dib, _, _ := wapi.CreateDIBSection.Call(uintptr(dst), uintptr(unsafe.Pointer(&bt)), uintptr(wapi.CreateDIBSectionUsage.RGBColors), uintptr(unsafe.Pointer(&ptr)), 0, 0)
// 	if dib == 0 {
// 		return nil, fmt.Errorf("Could not Create DIB Section err:%d.\n", wapi.GetLastError())
// 	}
// 	if dib == wapi.CreateDIBSectionError.InvalidParameter {
// 		return nil, fmt.Errorf("One or more of the input parameters is invalid while calling CreateDIBSection.\n")
// 	}
// 	defer wapi.DeleteObject.Call(dib)

// 	obj, _, _ := wapi.SelectObject.Call(dst, dib)
// 	if obj == 0 {
// 		return nil, fmt.Errorf("error occurred and the selected object is not a region err:%d.\n", wapi.GetLastError())
// 	}
// 	if obj == 0xffffffff { //GDI_ERROR
// 		return nil, fmt.Errorf("GDI_ERROR while calling SelectObject err:%d.\n", wapi.GetLastError())
// 	}
// 	defer wapi.DeleteObject.Call(obj)

// 	//if !bitBlt(mHDC, 0, 0, x, y, hdc, rect.Min.X, rect.Min.Y) {
// 	//	return nil, fmt.Errorf("BitBlt failed err:%d.\n", getLastError())
// 	//}

// 	width := rect.Dx()
// 	height := rect.Dy()

// 	var ret uintptr
// 	switch config.Current.Scale {
// 	case 1:
// 		ret, _, _ = wapi.BitBlt.Call(
// 			uintptr(dst),
// 			0,
// 			0,
// 			uintptr(width),
// 			uintptr(height),
// 			uintptr(src),
// 			uintptr(rect.Min.X),
// 			uintptr(rect.Min.Y),
// 			wapi.BitBltRasterOperations.CaptureBLT|wapi.BitBltRasterOperations.SrcCopy,
// 		)
// 	default: // Scaled.
// 		scaledW := int(float64(width) * config.Current.Scale)
// 		scaledH := int(float64(height) * config.Current.Scale)

// 		ret, _, _ = wapi.StretchBlt.Call(
// 			uintptr(dst),
// 			0,
// 			0,
// 			uintptr(scaledW),
// 			uintptr(scaledH),
// 			uintptr(src),
// 			uintptr(rect.Min.X),
// 			uintptr(rect.Min.Y),
// 			uintptr(width),
// 			uintptr(height),
// 			wapi.BitBltRasterOperations.CaptureBLT|wapi.BitBltRasterOperations.SrcCopy,
// 		)
// 	}
// 	if ret == 0 {
// 		notify.Error("Video: Failed to capture \"%s\"", config.Current.Video.Capture.Window.Name)
// 		return nil, fmt.Errorf("bitblt returned: %d", ret)
// 	}

// 	var slice []byte
// 	hdrp := (*reflect.SliceHeader)(unsafe.Pointer(&slice))
// 	hdrp.Data = uintptr(ptr)
// 	hdrp.Len = x * y * 4
// 	hdrp.Cap = x * y * 4

// 	imageBytes := make([]byte, len(slice))

// 	for i := 0; i < len(imageBytes); i += 4 {
// 		imageBytes[i], imageBytes[i+2], imageBytes[i+1], imageBytes[i+3] = slice[i+2], slice[i], slice[i+1], slice[i+3]
// 	}

// 	return &image.RGBA{
// 		Pix:    imageBytes,
// 		Stride: 4 * x,
// 		Rect:   image.Rect(0, 0, x, y),
// 	}, nil
// }

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
