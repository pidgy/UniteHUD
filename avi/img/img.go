package img

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"os"
	"sync"

	"github.com/tc-hib/winres"
	"gocv.io/x/gocv"

	"github.com/pidgy/unitehud/core/config"
	"github.com/pidgy/unitehud/core/notify"
)

type PNGPool struct {
	sync *sync.Pool
}

type cached struct {
	name  string
	img   image.Image
	bytes []byte
}

var (
	Empty = image.NewRGBA(image.Rect(0, 0, 128, 128))

	cache = []*cached{}
)

func NewPNGPool() *PNGPool {
	p := &PNGPool{
		sync: &sync.Pool{
			New: func() any {
				return new(png.EncoderBuffer)
			},
		},
	}
	for i := 0; i < 4096; i++ {
		p.Put(new(png.EncoderBuffer))
	}
	return p
}

func Icon(name string) image.Image {
	for i, c := range cache {
		if c.name == name {
			return cache[i].img
		}
	}

	f, err := os.Open(config.Current.AssetIcon(name))
	if err != nil {
		notify.Error("[Image] Failed to open image %s (%v)", name, err)
		return Empty
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		notify.Error("[Image] Failed to decode %s (%v)", name, err)
		return Empty
	}

	c := &cached{
		name: name,
		img:  img,
	}
	cache = append(cache, c)

	return c.img
}

func IconBytes(name string) []byte {
	for i, c := range cache {
		if c.name == name {
			return cache[i].bytes
		}
	}

	f, err := os.Open(config.Current.AssetIcon(name))
	if err != nil {
		notify.Error("[Image] Failed to open image %s (%v)", name, err)
		return nil
	}
	defer f.Close()

	ico, err := winres.LoadICO(f)
	if err != nil {
		notify.Error("[Image] Failed to decode %s (%v)", name, err)
		return nil
	}

	b := &bytes.Buffer{}

	err = ico.SaveICO(b)
	if err != nil {
		notify.Error("[Image] Failed to encode %s (%v)", name, err)
		return nil
	}

	c := &cached{
		name:  name,
		bytes: b.Bytes(),
	}
	cache = append(cache, c)

	return c.bytes
}

func NRGBA(mat gocv.Mat) (*image.NRGBA, error) {
	i, err := mat.ToImage()
	if err != nil {
		return nil, err
	}

	img, ok := i.(*image.NRGBA)
	if !ok {
		return nil, err
	}

	return img, nil
}

func RGBA(mat gocv.Mat) (*image.RGBA, error) {
	i, err := mat.ToImage()
	if err != nil {
		return nil, err
	}

	switch img := i.(type) {
	case *image.RGBA:
		return img, nil
	default:
		return nil, fmt.Errorf("failed to convert %T to an rgba image", i)
	}
}

func (p *PNGPool) Get() *png.EncoderBuffer {
	return p.sync.Get().(*png.EncoderBuffer)
}

func (p *PNGPool) Put(e *png.EncoderBuffer) {
	p.sync.Put(e)
}
