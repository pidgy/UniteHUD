package exe

import (
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"
)

const (
	AssetDirectory  = `assets`
	Title           = "UniteHUD"
	TitleAndVersion = Title + " " + Version
	Version         = "v" + VersionSemVer
	VersionSemVer   = "4.3.0"
)

var (
	Debug  = strings.Contains(strings.ToLower(os.Args[0]), "debug")
	Uptime = time.Now()

	sigq = make(chan os.Signal, 1)

	dir = ""
)

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

func Exit() {
	os.Exit(0)
}

// func Sigq() chan os.Signal {
// 	return sigq
// }

func VersionDash() string {
	return strings.ReplaceAll(Version, ".", "-")
}

func WaitForSignal() os.Signal {
	signal.Notify(sigq, os.Interrupt)
	return <-sigq
}
