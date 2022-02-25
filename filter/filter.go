package filter

import (
	"strings"

	"github.com/pidgy/unitehud/team"
	"github.com/rs/zerolog"
)

type Filter struct {
	*team.Team
	File  string
	Value int
	Alias bool
}

func New(t *team.Team, file string, value int, alias bool) Filter {
	return Filter{t, file, value, alias}
}

func Strip(file string) string {
	return strings.ReplaceAll(
		strings.ReplaceAll(
			strings.ReplaceAll(
				strings.ReplaceAll(
					file,
					".png",
					"",
				),
				".PNG",
				"",
			),
			"_big",
			"",
		),
		"_alt",
		"",
	)
}

func (f Filter) MarshalZerologObject(e *zerolog.Event) {
	e.Str("file", f.File).Str("team", f.Team.Name)
}
