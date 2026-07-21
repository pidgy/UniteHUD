package d3d11

import (
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
		if strings.Contains(ex.Title, "Projector") {
			break
		}
	}

	println("Windows:", len(ws.Infos))
	println("Window:", ex.String(), "Rect:", ex.WindowInfo.Window.String())
	println("----------------------------------------")

	w, err := NewWindow(ex.HWND())
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()

	for range 5 {
		now := time.Now()
		_, err := w.Capture()
		if err != nil {
			t.Fatal(err)
		}
		took := time.Since(now)
		println("took:", took.String())

		time.Sleep(time.Second)
	}

	img, err := w.Capture()
	if err != nil {
		t.Fatal(err)
	}
	err = save.PNG(img, "./d3d11.png")
	if err != nil {
		t.Fatal(err)
	}
}
