package audio

import (
	"testing"
	"time"

	"github.com/pidgy/unitehud/avi/audio/ffmpeg"
)

func TestFFMPEG(t *testing.T) {
	err := Open()
	if err != nil {
		t.Fatal(err)
	}

	current.input = ffmpeg.New("Digital Audio Interface (2- USB Digital Audio)")
	// err = Input("Digital Audio Interface (2- USB Digital Audio)")
	// if err != nil {
	// 	t.Fatal(err)
	// }

	err = Output(Default)
	if err != nil {
		t.Fatal(err)
	}

	waitq := make(chan bool)
	time.AfterFunc(time.Second*5, func() { close(waitq) })
	<-waitq
	Close()
}

func TestUSBVideo(t *testing.T) {
	err := Open()
	if err != nil {
		t.Fatal(err)
	}

	err = Input("Digital Audio Interface (2- USB Digital Audio)")
	if err != nil {
		t.Fatal(err)
	}

	err = Output(Default)
	if err != nil {
		t.Fatal(err)
	}

	waitq := make(chan bool)
	time.AfterFunc(time.Second*5, func() { close(waitq) })
	<-waitq
	Close()
}
