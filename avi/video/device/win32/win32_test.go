package win32

import "testing"

func TestNewVideoCaptureDevice(t *testing.T) {
	for index := 0; index < 100; index++ {
		d, err := NewVideoCaptureDevice(index)
		if err != nil {
			continue
		}

		println(d.Name)
		println("\t* Index:    ", d.Index)
		println("\t* WaveInID: ", d.WaveInID)
		println()
	}
}
