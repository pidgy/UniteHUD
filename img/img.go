package img

import (
	"image"
	"os"

	"gocv.io/x/gocv"

	"github.com/pidgy/unitehud/config"
	"github.com/pidgy/unitehud/notify"
)

var Empty = image.NewRGBA(image.Rect(0, 0, 50, 50))

var (
	cache = []image.Image{}
	names = []string{}
)

func RGBA(mat gocv.Mat) (*image.RGBA, error) {
	i, err := mat.ToImage()
	if err != nil {
		return nil, err
	}

	img, ok := i.(*image.RGBA)
	if !ok {
		return nil, err
	}

	return img, nil
}

func Icon(name string) image.Image {
	for i, n := range names {
		if n == name {
			return cache[i]
		}
	}

	f, err := os.Open(config.Current.AssetIcon(name))
	if err != nil {
		notify.Error("Failed to open image %s (%v)", name, err)
		return Empty
	}
	defer f.Close()

	i, format, err := image.Decode(f)
	if err != nil {
		notify.Error("Failed to decode %s image %s (%v)", format, name, err)
		return Empty
	}

	cache = append(cache, i)
	names = append(names, name)

	return i
}
