package stats

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"
)

type stat struct {
	path     string
	file     string
	matches  int
	category string
}

type stats struct {
	all []stat

	categories map[string]*category
}

type category struct {
	hits   float64
	misses float64
	usage  float64
}

func load(f string) (m map[string]int) {
	b, err := os.ReadFile(f)
	if err != nil {
		panic(err)
	}

	err = json.Unmarshal(b, &m)
	if err != nil {
		panic(err)
	}

	return
}

func TestTemplates(t *testing.T) {
	s := stats{
		categories: map[string]*category{},
	}

	for k, v := range load("../../templates.json") {
		args := strings.Split(k, "/")

		c := strings.Join(args[3:len(args)-1], "/")

		if s.categories[c] == nil {
			s.categories[c] = &category{}
		}
		if v > 0 {
			s.categories[c].hits++
		} else {
			s.categories[c].misses++
		}
		s.all = append(s.all, stat{
			path:     k,
			file:     args[len(args)-1],
			matches:  v,
			category: c,
		})
	}

	for _, c := range s.categories {
		c.usage = c.hits / (c.hits + c.misses)
	}

	sort.Slice(s.all, func(i, j int) bool {
		d := strings.Compare(s.all[i].category, s.all[j].category)
		if d == 0 {
			return s.all[i].matches > s.all[j].matches
		}
		return d > 0
	})

	c := ""
	for _, stat := range s.all {
		if stat.category != c {
			println()
			println(stat.category, "usage:", fmt.Sprintf("%.1f%%", s.categories[stat.category].usage*100))
			c = stat.category
		}

		println(stat.file, stat.matches)
	}
}
