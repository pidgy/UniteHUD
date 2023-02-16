package global

import (
	"os"
	"strings"
)

const (
	Version = "v1.1beta"
)

var (
	DebugMode = strings.Contains(strings.ToLower(os.Args[0]), "debug")
)
