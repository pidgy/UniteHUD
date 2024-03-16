package stats

import "testing"

func TestAppend(t *testing.T) {
	s := sortable{}

	if s.Len() != 0 {
		t.FailNow()
	}

	s.add("n", 1, 1, 1)
	if s.Len() != 1 {
		t.FailNow()
	}

	s.add("n", 1, 1, 1)
	s.add("n", 1, 1, 1)
	s.add("n", 1, 1, 1)
	s.add("n", 1, 1, 1)
	if s.Len() != 5 {
		t.FailNow()
	}
}
