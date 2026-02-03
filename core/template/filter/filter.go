package filter

// TODO: Add go style comments that reflect the purpose of each type, function, var, and const.

import (
	"fmt"
	"strings"

	"github.com/pidgy/unitehud/core/team"
)

// Filter defines Filter behavior and state.
type Filter struct {
	*team.Team
	File  string
	Value int
	Alias bool
}

// New returns a new instance.
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
