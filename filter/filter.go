package filter

import (
	"fmt"
	"strings"

	"github.com/pidgy/unitehud/team"
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

func (f *Filter) Truncated() string {
	count := strings.Count(f.File, "_alt")

	if count > 0 {
		return fmt.Sprintf("%s_alt_x%d.png",
			strings.ReplaceAll(
				strings.ReplaceAll(
					strings.ReplaceAll(
						f.File,
						".PNG",
						"",
					),
					".png",
					"",
				),
				"_alt",
				"",
			),
			count)
	}

	return f.File
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
