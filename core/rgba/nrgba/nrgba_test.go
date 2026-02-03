package nrgba

import (
	"fmt"
	"testing"
)

// TestWindows runs the routine.
func TestWindows(t *testing.T) {
	m := map[string]NRGBA{
		"Red":          {R: 255},
		"Green":        {G: 255},
		"Blue":         {B: 255},
		"Pinkity":      Pinkity,
		"System":       System,
		"Discord":      Discord,
		"PastelBlue":   PastelBlue,
		"PastelCoral":  PastelCoral,
		"LightPurple":  LightPurple,
		"Orange":       Orange,
		"Purple":       Purple,
		"User":         User,
		"DarkerYellow": DarkerYellow,
		"PastelYellow": PastelYellow,
		"Yellow":       Yellow,
		"Regice":       Regice,
	}

	for k, v := range m {
		fmt.Print(v.Windows(k))
	}
}
