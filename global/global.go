package global

import (
	"os"
	"strings"
	"time"
)

const (
	Version = "v2.1"
)

var (
	DebugMode = strings.Contains(strings.ToLower(os.Args[0]), "debug")
	Uptime    = time.Now()
)
