package win32

import "testing"

func TestNewVideoCaptureDevice(t *testing.T) {
	index := 1

	d, err := NewVideoCaptureDevice(index)
	if err != nil {
		t.Fatal(err)
	}
	if d == nil {
		t.Fatalf("device %d is nil", index)
	}

	println("ID:", d.ID)
	println("Name:", d.Name)
	println("Path:", d.Path)
	println("WaveInID:", d.WaveInID)
}
