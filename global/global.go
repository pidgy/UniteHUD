package global

import (
	"os"
	"strings"
)

const (
	Version = "v2.0"
)

var (
	DebugMode = strings.Contains(strings.ToLower(os.Args[0]), "debug")
)
