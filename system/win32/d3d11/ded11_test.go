package d3d11

import (
	"image"
	"strings"
	"testing"
	"time"

	"github.com/pidgy/unitehud/system/save"
	"github.com/pidgy/unitehud/system/win32"
)

/*
	cd system\win32\d3d11
	clear && go test -v
*/
func TestCaptureWindow(t *testing.T) {
	ws, err := win32.FindWindows()
	if err != nil {
		t.Fatal(err)
	}

	ex := win32.WindowInfoEx{}

	for _, ex = range ws.Infos {
		if strings.Contains(ex.Title, "Chrome") {
			break
		}
	}

	m, err := ex.Monitor()
	if err != nil {
		t.Fatal(err)
	}

	println("Windows:", len(ws.Infos))
	println("Window:", ex.String(), "Rect:", ex.WindowInfo.Window.String())
	println("Monitor:", m.String())
	println("----------------------------------------")

	c, err := NewCapture(m.HWND)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	var img image.Image

	for range 5 {
		time.Sleep(time.Second)

		now := time.Now()
		img2, err := c.CaptureWindow(ex.WindowInfo.Window.Image())
		if err != nil {
			t.Log(err)
			continue
		}
		ms := time.Since(now)
		println("took:", ms.String())
		img = img2
	}

	if img == nil {
		t.Fatal("img nil")
	}

	err = save.PNG(img, "./d3d11.png")
	if err != nil {
		t.Fatal(err)
	}
}
