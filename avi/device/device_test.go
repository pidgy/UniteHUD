package device

import "testing"

func TestNewAudioCaptureDevice(t *testing.T) {
	for index := 0; index < 10; index++ {
		d, err := NewAudioCaptureDevice(index)
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

func TestNewAudioVideoCaptureDevice(t *testing.T) {
	for index := 0; index < 10; index++ {
		a, v, err := NewAudioVideoCaptureDevice(index)
		if err != nil {
			println(err.Error())
			continue
		}

		println(index, "how", a == nil, v == nil)

		println("Audio")
		println("\tName:", a.Name)
		println("\t\t* Index:       ", a.Index)
		println("\t\t* WaveInID:    ", a.WaveInID)
		println("\t\t* Description: ", a.Description)
		println("\t\t* Type:         ", a.Path.Type)
		println("\t\t* VendorID:     ", a.Path.VendorID)
		println("Video")
		println("\tName:", v.Name)
		println("\t\t* Index:       ", v.Index)
		println("\t\t* WaveInID:    ", v.WaveInID)
		println("\t\t* Description: ", v.Description)
		println("\t\t* Type:         ", v.Path.Type)
		println("\t\t* VendorID:     ", v.Path.VendorID)
		println()
	}
}

func TestNewVideoCaptureDevice(t *testing.T) {
	for index := 0; index < 10; index++ {
		d, err := NewVideoCaptureDevice(index)
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
