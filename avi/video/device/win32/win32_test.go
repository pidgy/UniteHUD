package win32

import "testing"

func TestNewVideoCaptureDevice(t *testing.T) {
	for index := 0; index < 10; index++ {
		println("Index:", index)

		d, err := NewVideoCaptureDevice(index)
		if err != nil {
			println("\tError:", err.Error())
			continue
		}
		println("\tID:", d.ID)
		println("\tName:", d.Name)
		println("\tPath:", d.Path)
		println("\tWaveInID:", d.WaveInID)
	}
}
