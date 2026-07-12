package monitor

import (
	"fmt"
	"image"
	"testing"

	"github.com/pidgy/unitehud/core/config"
	"github.com/pidgy/unitehud/core/notify"
	"github.com/pidgy/unitehud/exe"
)

func init() {
	exe.Debug = true
	notify.Disabled.Debug = false
	config.Current.Video.Capture.Monitor.Name = config.MainDisplay
	Open()
}

func TestScreenshot(t *testing.T) {
	img, err := Capture()
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("%s, %s", img.Rect.Min, img.Rect.Max)

	img, err = CaptureRect(config.Current.XY.Scores)
	if err != nil {
		t.Fatal(err)
	}

	img = img.SubImage(config.Current.XY.Scores).(*image.RGBA)

	t.Logf("img: %s", img.Rect)
}

func BenchmarkLoadMap(b *testing.B) {
	m, ok := active()
	if !ok {
		b.Fatalf("%s: <ini:f:load> display", config.Current.Video.Capture.Monitor.Name)
	}

	s := fmt.Sprintf("%dx%d", m.bounds.Dx(), m.bounds.Dy())

	for i := 0; i < b.N; i++ {
		_ = s
	}
}
