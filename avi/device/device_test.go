package device

import "testing"

func TestNewAudioCaptureDevice(t *testing.T) {
	for index := 0; index < 10; index++ {
		d, err := NewAudioCaptureDevice(index)
		if err != nil {
			continue
		}

		println(d.Name)
		println("\t* Index:       ", d.Index)
		println("\t* WaveInID:    ", d.WaveInID)
		println("\t* Path:        ", d.Path)
		println("\t* Description: ", d.Description)
		println()
	}
}

func TestNewVideoCaptureDevice(t *testing.T) {
	for index := 0; index < 10; index++ {
		d, err := NewVideoCaptureDevice(index)
		if err != nil {
			continue
		}

		println(d.Name)
		println("\t* Index:       ", d.Index)
		println("\t* WaveInID:    ", d.WaveInID)
		println("\t* Path:        ", d.Path)
		println("\t* Description: ", d.Description)
		println()
	}
}

func TestNewVideoCaptureDeviceName(t *testing.T) {
	for index := 0; index < 10; index++ {
		name, err := VideoCaptureDeviceName(index)
		if err != nil {
			continue
		}

		println("[", index, "]", name)
	}
}
