package win32

import "testing"

func TestNewVideoCaptureDevice(t *testing.T) {
	for index := 0; index < 10; index++ {
		println("Video Capture Device", index)

		d, err := NewVideoCaptureDevice(index)
		if err != nil {
			println("* Error:", err.Error(), "\n")
			continue
		}

		println("\t* Name:", d.Name)
		println("\t* Path:", d.Path)
		println("\t* ID:", d.ID)
		println("\t* WaveInID:", d.WaveInID)
		println()
	}
}
