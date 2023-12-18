package ffmpeg

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	"github.com/gen2brain/malgo"
	"github.com/hajimehoshi/go-mp3"
	"github.com/hajimehoshi/oto/v2"
)

func TestDevice(t *testing.T) {
	f := New("USB Video")
	defer f.Close()

	buf := bytes.NewBuffer(make([]byte, 2646))

	d, err := mp3.NewDecoder(buf)
	if err != nil {
		t.Fatal(err)
	}

	c, ready, err := oto.NewContext(d.SampleRate(), 2, 2)
	if err != nil {
		t.Fatal(err)
	}
	<-ready

	err = f.Start(malgo.Context{}, buf)
	if err != nil {
		t.Fatal(err)
	}

	p := c.NewPlayer(d)
	defer p.Close()
	p.Play()

	fmt.Printf("Length: %d[bytes]\n", d.Length())
	for {
		time.Sleep(time.Second)
		if !p.IsPlaying() {
			break
		}
	}
}
