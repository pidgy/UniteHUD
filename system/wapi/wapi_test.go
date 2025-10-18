package wapi

import (
	"fmt"
	"testing"
	"time"
)

func TestEnumerateWindows(t *testing.T) {
	time.Sleep(time.Second * 5)

	err := EnumerateWindows(func(w Window) bool {
		title, err := w.Title()
		if err != nil {
			return false
		}

		if w.Visible() {
			fmt.Printf("%s is visible: %t\n", title, w.Visible())
		}

		return false
	})
	if err != nil {
		t.Fatal(err)
	}
}
