package stats

import (
	"encoding/json"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"testing"
)

// // TestAppend runs the routine.
// func TestAppend(t *testing.T) {
// 	s := sortable{}

// 	if s.Len() != 0 {
// 		t.FailNow()
// 	}

// 	s.add("n", 1, 1, 1)
// 	if s.Len() != 1 {
// 		t.FailNow()
// 	}

// 	s.add("n", 1, 1, 1)
// 	s.add("n", 1, 1, 1)
// 	s.add("n", 1, 1, 1)
// 	s.add("n", 1, 1, 1)
// 	if s.Len() != 5 {
// 		t.FailNow()
// 	}
// }

func TestCollectStats(t *testing.T) {
	dir := `C:\Users\trash\Documents\dev\go\src\github.com\pidgy\unitehud\saved`
	var files []string

	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		d.Name()
		if !d.IsDir() && d.Name() == "templates.json" {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}

	totals := map[string]int{}
	paths := map[string][]string{}

	for _, path := range files {
		file, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}

		cur := map[string]int{}
		err = json.Unmarshal(file, &cur)
		if err != nil {
			t.Fatalf("%s: %v", path, err)
		}

		for t, n := range cur {
			totals[t] += n
			paths[t] = append(paths[t], path)
		}
	}

	highestFile := ""
	highestFileCount := 0

	for t, n := range totals {
		if n > highestFileCount {
			highestFile = t
			highestFileCount = n
		}
		if n == 0 {
			println(t, n)
			for _, p := range paths[t] {
				println("->", p)
			}
		}
	}

	println("highest:", highestFile, highestFileCount)
	println("total files read:", len(files))
}
