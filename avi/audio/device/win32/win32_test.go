package win32

import "testing"

func TestMain(t *testing.T) {
	TestNewAudioCaptureDevice(t)
	TestNewAudioCaptureRenderDevice(t)
	TestNewAudioRenderDevice(t)
}

func TestNewAudioCaptureDevice(t *testing.T) {
	for index := 0; index < 100; index++ {
		d, err := NewAudioCaptureDevice(index)
		if err != nil {
			continue
		}

		println(d.String())
		println("\t* Index:      ", d.Index)
		println("\t* Name:		  ", d.Name)
		println("\t* ID:		  ", d.ID)
		println("\t* GUID:		  ", d.GUID)
		println("\t* Format:	  ", d.Format)
		println("\t* Association: ", d.Association)
		println("\t* JackSubType: ", d.JackSubType)
		println("\t* Description: ", d.Description)
		println("\t* Flow:        ", d.Flow.String())
		println()
	}
}

func TestNewAudioCaptureRenderDevice(t *testing.T) {
	for index := 0; index < 100; index++ {
		d, err := NewAudioCaptureRenderDevice(index)
		if err != nil {
			continue
		}

		println(d.String())
		println("\t* Index:      ", d.Index)
		println("\t* Name:		  ", d.Name)
		println("\t* ID:		  ", d.ID)
		println("\t* GUID:		  ", d.GUID)
		println("\t* Format:	  ", d.Format)
		println("\t* Association: ", d.Association)
		println("\t* JackSubType: ", d.JackSubType)
		println("\t* Description: ", d.Description)
		println("\t* Flow:        ", d.Flow.String())
		println()
	}
}

func TestNewAudioRenderDevice(t *testing.T) {
	for index := 0; index < 100; index++ {
		d, err := NewAudioRenderDevice(index)
		if err != nil {
			continue
		}

		println(d.String())
		println("\t* Index:      ", d.Index)
		println("\t* Name:		  ", d.Name)
		println("\t* ID:		  ", d.ID)
		println("\t* GUID:		  ", d.GUID)
		println("\t* Format:	  ", d.Format)
		println("\t* Association: ", d.Association)
		println("\t* JackSubType: ", d.JackSubType)
		println("\t* Description: ", d.Description)
		println("\t* Flow:        ", d.Flow.String())
		println()
	}
}
