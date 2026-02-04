package filter

import (
	"fmt"
	"strings"

	"github.com/pidgy/unitehud/core/team"
)

// Filter captures the file-backed filter metadata for a team template entry.
type Filter struct {
	*team.Team
	File  string
	Value int
	Alias bool
}

// New builds a Filter for a given team and filter file with its value and alias flag.
func New(t *team.Team, file string, value int, alias bool) Filter {
	return Filter{t, file, value, alias}
}

// Truncated collapses repeated "_alt" suffixes into a single "_alt_xN" filename.
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

// Strip removes known suffixes and extensions from a filter filename for comparisons.
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
