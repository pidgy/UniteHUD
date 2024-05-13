package win32

import "testing"

func TestNewAudioCaptureDevice(t *testing.T) {
	for index := 0; index < 100; index++ {
		d, err := NewAudioCaptureDevice(index)
		if err != nil {
			// println("* Error:", err.Error(), "\n")
			continue
		}

		println("Audio Capture Device", index)
		println(d.String())
		println("\t* Name:		 ", d.Name)
		println("\t* ID:		 ", d.ID)
		println("\t* GUID:		 ", d.GUID)
		println("\t* Format:	 ", d.Format)
		println("\t* Association:", d.Association)
		println("\t* JackSubType:", d.JackSubType)
		println("\t* Description:", d.Description)
		println()
	}
}
