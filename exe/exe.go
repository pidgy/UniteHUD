package exe

import (
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

const (
	// AssetDirectory is the default folder name for bundled app assets.
	AssetDirectory = `assets`
	// Title is the human-facing app name.
	Title = "UniteHUD"
	// TitleAndVersion is the full title string with the current version appended.
	TitleAndVersion = Title + " " + Version
	// Version is the semantic version prefixed with "v".
	Version = "v" + VersionSemVer
	// VersionSemVer is the raw semantic version string.
	VersionSemVer = "4.6.0"
)

var (
	// Debug reports whether the executable name suggests a debug build.
	Debug = strings.Contains(strings.ToLower(os.Args[0]), "debug")
	// Uptime records the process start time.
	Uptime = time.Now()

	// Caser provides consistent title casing for UI strings.
	Caser = cases.Title(language.English)

	sigq = make(chan os.Signal, 1)

	dir = ""
)

// Close shuts down the signal channel used for orderly exits.
func Close() {
	if sigq == nil {
		return
	}
	close(sigq)
	sigq = nil
}

// Directory returns the executable's directory, caching the result.
func Directory() string {
	if dir == "" {
		e, err := os.Executable()
		if err != nil {
			dir = "<ini:f:locate> executable directory"
		}
		dir = filepath.Dir(e)
	}
	return dir
}

// Exit terminates the process immediately with a zero status.
func Exit() {
	os.Exit(0)
}

// VersionDash returns Version with dots replaced by dashes.
func VersionDash() string {
	return strings.ReplaceAll(Version, ".", "-")
}

// WaitForSignal blocks until an interrupt signal is received.
func WaitForSignal() os.Signal {
	signal.Notify(sigq, os.Interrupt)
	return <-sigq
}
