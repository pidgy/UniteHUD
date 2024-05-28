package global

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	Title          = "UniteHUD"
	Version        = "v" + VersionNoV
	VersionNoV     = "3.3.0"
	TitleVersion   = Title + " " + Version
	AssetDirectory = `assets`
)

var (
	DebugMode = strings.Contains(strings.ToLower(os.Args[0]), "debug")
	Uptime    = time.Now()

	dir = ""
)

func WorkingDirectory() string {
	if dir == "" {
		e, err := os.Executable()
		if err != nil {
			dir = "failed to locate executable directory"
		}
		dir = filepath.Dir(e)
	}
	return dir
}
