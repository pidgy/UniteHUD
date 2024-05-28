package device

import "testing"

func TestNewAudioCaptureDevice(t *testing.T) {
	for index := 0; index < 10; index++ {
		d, err := AudioCaptureDevice(index)
		if err != nil {
			continue
		}

		println("Name:", d.Name)
		println("\t* Index:       ", d.Index)
		println("\t* WaveInID:    ", d.WaveInID)
		println("\t* Description: ", d.Description)
		println("\t\t* Type:         ", d.Path.Type)
		println("\t\t* VendorID:     ", d.Path.VendorID)
		println()
	}
}

func TestNewVideoCaptureDevice(t *testing.T) {
	for index := 0; index < 10; index++ {
		d, err := VideoCaptureDevice(index)
		if err != nil {
			continue
		}

		println("Name:", d.Name)
		println("\t* Index:       ", d.Index)
		println("\t* WaveInID:    ", d.WaveInID)
		println("\t* Description: ", d.Description)
		println("\t\t* Type:         ", d.Path.Type)
		println("\t\t* VendorID:     ", d.Path.VendorID)
		println("\t* Path:         ", d.Path.String())
		println()
	}
}

func TestAudioCaptureDeviceName(t *testing.T) {
	for index := 0; index < 10; index++ {
		name, err := AudioCaptureDeviceName(index)
		if err != nil {
			continue
		}

		println("[", index, "]", name)
	}
}

func TestVideoCaptureDeviceName(t *testing.T) {
	for index := 0; index < 10; index++ {
		name, err := VideoCaptureDeviceName(index)
		if err != nil {
			continue
		}

		println("[", index, "]", name)
	}
}
