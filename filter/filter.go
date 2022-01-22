package filter

import (
	"github.com/pidgy/unitehud/team"
	"github.com/rs/zerolog"
)

type Filter struct {
	*team.Team
	File  string
	Value int
}

func (f Filter) MarshalZerologObject(e *zerolog.Event) {
	e.Str("file", f.File).Str("team", f.Team.Name)
}
