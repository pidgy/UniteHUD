package global

import (
	"os"
	"strings"
	"time"
)

const (
	Title        = "UniteHUD"
	Version      = "v2.3.0"
	TitleVersion = Title + " " + Version
	AssetsFolder = `assets`
)

var (
	DebugMode = strings.Contains(strings.ToLower(os.Args[0]), "debug")
	Uptime    = time.Now()
)
