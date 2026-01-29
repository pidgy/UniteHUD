package window

import (
	"image/png"
	"os"
	"testing"

	"github.com/pidgy/unitehud/core/config"
)

func TestList(t *testing.T) {
	config.Current.Scale = 1

	err := list()
	if err != nil {
		t.Fatal(err)
	}

	if len(Sources) == 0 {
		t.Fatal("failed to find windows")
	}

	config.Current.Video.Capture.Window.Name = Sources[1]
	img, err := Capture()
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("%s: %s", config.Current.Video.Capture.Window.Name, img.Bounds())

	f, err := os.Create("window.png")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	err = png.Encode(f, img)
	if err != nil {
		t.Fatal(err)
	}
}
