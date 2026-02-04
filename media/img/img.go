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

// PNGPool reuses png.EncoderBuffer instances to reduce allocations.
type PNGPool struct {
	sync *sync.Pool
}

// cached stores icon data to avoid repeated loads.
type cached struct {
	name  string
	img   image.Image
	bytes []byte
}

var (
	// Empty is a fallback image when decoding fails.
	Empty = image.NewRGBA(image.Rect(0, 0, 128, 128))

	// cache stores icon images and bytes by name.
	cache = []*cached{}
)

// NewPNGPool returns a pool of reusable png encoder buffers.
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

// Icon loads and caches an icon image by name.
func Icon(name string) image.Image {
	for i, c := range cache {
		if c.name == name {
			return cache[i].img
		}
	}

	f, err := os.Open(config.Current.AssetIcon(name))
	if err != nil {
		notify.Error("[Image] <ini:f:open> image %s (%v)", name, err)
		return Empty
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		notify.Error("[Image] <ini:f:decode> %s (%v)", name, err)
		return Empty
	}

	c := &cached{
		name: name,
		img:  img,
	}
	cache = append(cache, c)

	return c.img
}

// IconBytes loads and caches an icon as ICO bytes.
func IconBytes(name string) []byte {
	for i, c := range cache {
		if c.name == name {
			return cache[i].bytes
		}
	}

	f, err := os.Open(config.Current.AssetIcon(name))
	if err != nil {
		notify.Error("[Image] <ini:f:open> image %s (%v)", name, err)
		return nil
	}
	defer f.Close()

	ico, err := winres.LoadICO(f)
	if err != nil {
		notify.Error("[Image] <ini:f:decode> %s (%v)", name, err)
		return nil
	}

	b := &bytes.Buffer{}

	err = ico.SaveICO(b)
	if err != nil {
		notify.Error("[Image] <ini:f:encode> %s (%v)", name, err)
		return nil
	}

	c := &cached{
		name:  name,
		bytes: b.Bytes(),
	}
	cache = append(cache, c)

	return c.bytes
}

// NRGBA converts a Mat to an *image.NRGBA.
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

// Get returns a buffer from the pool.
func (p *PNGPool) Get() *png.EncoderBuffer {
	return p.sync.Get().(*png.EncoderBuffer)
}

// Put returns a buffer to the pool.
func (p *PNGPool) Put(e *png.EncoderBuffer) {
	p.sync.Put(e)
}

// RGBA converts a Mat to an *image.RGBA.
func RGBA(mat gocv.Mat) (*image.RGBA, error) {
	i, err := mat.ToImage()
	if err != nil {
		return nil, err
	}

	switch img := i.(type) {
	case *image.RGBA:
		return img, nil
	default:
		return nil, fmt.Errorf("<ini:f:convert> %T to an RGBA image", i)
	}
}
