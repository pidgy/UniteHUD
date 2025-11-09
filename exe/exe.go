package exe

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	AssetDirectory  = `assets`
	Title           = "UniteHUD"
	TitleAndVersion = Title + " " + Version
	Version         = "v" + VersionSemVer
	VersionSemVer   = "4.1.2"
)

var (
	Debug  = strings.Contains(strings.ToLower(os.Args[0]), "debug")
	Uptime = time.Now()

	sigq = make(chan os.Signal, 1)

	dir = ""
)

func Chan() chan os.Signal {
	return sigq
}

func Close() {
	if sigq == nil {
		return
	}
	close(sigq)
	sigq = nil
}

func Directory() string {
	if dir == "" {
		e, err := os.Executable()
		if err != nil {
			dir = "<ini:failed:locate> executable directory"
		}
		dir = filepath.Dir(e)
	}
	return dir
}

func VersionDash() string {
	return strings.ReplaceAll(Version, ".", "-")
}
