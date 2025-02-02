package exe

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	Title           = "UniteHUD"
	TitleAndVersion = Title + " " + Version
	Version         = "v" + VersionSemVer
	VersionSemVer   = "3.9.0"
	AssetDirectory  = `assets`
)

var (
	Debug  = strings.Contains(strings.ToLower(os.Args[0]), "debug")
	Uptime = time.Now()

	dir = ""
)

func Directory() string {
	if dir == "" {
		e, err := os.Executable()
		if err != nil {
			dir = "failed to locate executable directory"
		}
		dir = filepath.Dir(e)
	}
	return dir
}

func VersionDash() string {
	return strings.ReplaceAll(Version, ".", "-")
}
