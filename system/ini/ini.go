package ini

import (
	"fmt"
	"path"
	"regexp"
	"strings"

	"github.com/pkg/errors"
	"gopkg.in/ini.v1"

	"github.com/pidgy/unitehud/exe"
)

type Locale string

const (
	EnUS = "en-US"
	EsES = "es-ES"
	JpJP = "jp-JP"
	KrKR = "kr-KR"
)

var (
	file  *ini.File
	regex = regexp.MustCompile("<ini:[a-zA-Z0-9].*?>")
)

func Default() error {
	return errors.Wrap(Open(EnUS), "ini default")
}

func Find(s, k string) string { return find(s, k) }

func Format(format string) string {
	return regex.ReplaceAllStringFunc(format, replace)
}

func (l Locale) String() string {
	return string(l)
}

func Open(locale Locale) error {
	f := path.Join(exe.Directory(), "assets", "ini", locale.String())

	switch ext := path.Ext(f); ext {
	case ".ini":
	case "":
		f = fmt.Sprintf("%s.ini", f)
	default:
		return errors.Errorf("locale: invalid exension: %s", ext)
	}

	i, err := ini.Load(f)
	if err != nil {
		return errors.Wrapf(err, "locale: %s", locale)
	}
	file = i

	return nil
}

func find(s, k string) string {
	if file == nil {
		return fmt.Sprintf("%s-%s", s, k)
	}

	v := file.Section(s).Key(k).Value()
	if v == "" {
		return fmt.Sprintf("%s-%s", s, k)
	}

	return v
}

func replace(s string) string {
	px, ok := strings.CutPrefix(s, "<")
	if !ok {
		return s
	}

	sx, ok := strings.CutSuffix(px, ">")
	if !ok {
		return s
	}

	args := strings.Split(sx, ":")
	if args[0] != "ini" {
		return s
	}

	if len(args) != 3 {
		return s
	}

	return find(args[1], args[2])
}
