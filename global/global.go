package global

import (
	"os"
	"strings"
)

const (
	Version = "v1.1"
)

var (
	DebugMode = strings.Contains(strings.ToLower(os.Args[0]), "debug")
)
