package stats

import "sort"

type (
	sortable []struct {
		Name      string
		Matches   int
		Average   int
		Frequency float32
	}
)

func (s *sortable) add(name string, m, a int, f float32) {
	*s = append(*s, struct {
		Name      string
		Matches   int
		Average   int
		Frequency float32
	}{name, m, a, f})
}

func (s sortable) Len() int { return len(s) }

func (s sortable) Less(i, j int) bool {
	if s[i].Matches == s[j].Matches {
		if s[i].Average == s[j].Average {
			return s[i].Frequency > s[j].Frequency
		}

		return s[i].Average > s[j].Average
	}

	return s[i].Matches >= s[j].Matches
}

func (s sortable) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s sortable) Sort() {
	sort.Sort(s)
}
