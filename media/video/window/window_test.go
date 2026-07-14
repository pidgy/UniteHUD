package window

import (
	"fmt"
	"image/png"
	"os"
	"testing"

	"github.com/pidgy/unitehud/core/config"
)

func TestList(t *testing.T) {
	err := background()
	if err != nil {
		t.Fatal(err)
	}

	if len(sources) == 0 {
		t.Fatal("failed to find windows")
	}

	config.Current.Video.Capture.Monitor.Name = sources[1].Title

	for i := range 5 {
		img, err := Capture()
		if err != nil {
			t.Fatal(err)
		}

		t.Logf("%s: %s", config.Current.Video.Capture.Monitor.Name, img.Bounds())

		f, err := os.Create(fmt.Sprintf("window_%d.png", i))
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()

		err = png.Encode(f, img)
		if err != nil {
			t.Fatal(err)
		}
	}
}
