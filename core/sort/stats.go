package sort

import "sort"

type Stats []stat

type stat struct {
	Name      string
	Matches   int
	Average   int
	Frequency float32
}

func (s Stats) Append(name string, m, a int, f float32) Stats {
	return append(s, stat{name, m, a, f})
}

func (s Stats) Len() int { return len(s) }

func (s Stats) Less(i, j int) bool {
	if s[i].Matches == s[j].Matches {
		if s[i].Average == s[j].Average {
			return s[i].Frequency > s[j].Frequency
		}

		return s[i].Average > s[j].Average
	}

	return s[i].Matches >= s[j].Matches
}

func (s Stats) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s Stats) Sort() {
	sort.Sort(s)
}
