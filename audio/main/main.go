package main

import (
	"time"

	"github.com/pidgy/unitehud/audio"
)

func main() {
	// in := audio.DefaultCapture()
	// out := audio.DefaultPlayback()

	session, err := audio.New("HD60", audio.DeviceDefault)
	if err != nil {
		panic(err)
	}

	time.AfterFunc(time.Second*5, func() {
		err = session.Close()
		if err != nil {
			panic(err)
		}
	})
	//defer session.Close()

	for ; ; time.Sleep(time.Second) {
		err := session.Error()
		if err != nil {
			if err == audio.SessionClosed {
				return
			}

			panic(err)
		}
	}
}
