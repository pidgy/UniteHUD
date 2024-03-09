package save

import (
	"testing"

	"github.com/pidgy/unitehud/global"
)

func TestTemplateStatistics(t *testing.T) {
	global.DebugMode = true
	TemplateStatistics()
}
