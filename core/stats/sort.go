package stats

import "sort"

// sortable collects stat rows and provides sort behavior for them.
type sortable []struct {
	Name      string
	Matches   int
	Average   int
	Frequency float32
}

// Len returns the number of rows in the sortable collection.
func (s sortable) Len() int { return len(s) }

// Less orders by Matches, then Average, then Frequency in descending order.
func (s sortable) Less(i, j int) bool {
	if s[i].Matches == s[j].Matches {
		if s[i].Average == s[j].Average {
			return s[i].Frequency > s[j].Frequency
		}

		return s[i].Average > s[j].Average
	}

	return s[i].Matches >= s[j].Matches
}

// Swap exchanges two rows in the sortable collection.
func (s sortable) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

// Sort orders the sortable collection in place.
func (s sortable) Sort() {
	sort.Sort(s)
}

// add appends a new row to the sortable collection.
func (s *sortable) add(name string, m, a int, f float32) {
	*s = append(*s, struct {
		Name      string
		Matches   int
		Average   int
		Frequency float32
	}{name, m, a, f})
}
