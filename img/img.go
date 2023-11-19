package img

import (
	"bytes"
	"image"
	"image/png"
	"os"
	"sync"

	"gocv.io/x/gocv"

	"github.com/tc-hib/winres"

	"github.com/pidgy/unitehud/config"
	"github.com/pidgy/unitehud/notify"
)

var Empty = image.NewRGBA(image.Rect(0, 0, 128, 128))

var (
	cache = []*cached{}
)

type cached struct {
	name  string
	img   image.Image
	bytes []byte
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

	img, ok := i.(*image.RGBA)
	if !ok {
		return nil, err
	}

	return img, nil
}

func Icon(name string) image.Image {
	for i, c := range cache {
		if c.name == name {
			return cache[i].img
		}
	}

	f, err := os.Open(config.Current.AssetIcon(name))
	if err != nil {
		notify.Error("Failed to open image %s (%v)", name, err)
		return Empty
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		notify.Error("Failed to decode %s (%v)", name, err)
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
		notify.Error("Failed to open image %s (%v)", name, err)
		return []byte{}
	}
	defer f.Close()

	ico, err := winres.LoadICO(f)
	if err != nil {
		notify.Error("Failed to decode %s (%v)", name, err)
		return []byte{}
	}

	b := &bytes.Buffer{}

	err = ico.SaveICO(b)
	if err != nil {
		notify.SystemWarn("Failed to encode %s (%v)", name, err)
	}

	c := &cached{
		name:  name,
		bytes: b.Bytes(),
	}
	cache = append(cache, c)

	return c.bytes
}

type PNGPool struct {
	sync *sync.Pool
}

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

func (p *PNGPool) Get() *png.EncoderBuffer {
	return p.sync.Get().(*png.EncoderBuffer)
}

func (p *PNGPool) Put(e *png.EncoderBuffer) {
	p.sync.Put(e)
}
