package device

import (
	"testing"
	"time"

	"github.com/pidgy/unitehud/avi/img/splash"
	"github.com/pidgy/unitehud/core/config"
	"gocv.io/x/gocv"
)

func BenchmarkTimeNow(b *testing.B) {
	for i := 0; i < b.N; i++ {
		time.Now()
	}
}

func BenchmarkCapture(b *testing.B) {
	active.reset()

	var err error

	mat, err = gocv.ImageToMatRGB(splash.Device())
	if err != nil {
		b.Fatal(err)
	}
	if mat.Empty() {
		b.Fatal("mat is empty")
	}

	config.Current.Video.Capture.Device.Index = 1

	err = Open()
	if err != nil {
		b.Fatal(err)
	}
	defer Close()

	defer since(time.Now())

	img, err := Capture()
	if err != nil {
		b.Fatal(err)
	}
	if img == nil {
		b.Fatal("image is nil")
	}
}

func since(now time.Time) {
	println(time.Since(now).Milliseconds(), "ms")
}
