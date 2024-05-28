package notify

import (
	"testing"
)

func TestLastStrings(t *testing.T) {
	// 0 Posts.
	expected := 0
	if got := len(LastNStrings(100)); got != expected {
		t.Fatalf("failed, got: %d, want: %d", got, expected)
	}
	if got := len(LastNStrings(1)); got != expected {
		t.Fatalf("failed, got: %d, want: %d", got, expected)
	}
	if got := len(LastNStrings(0)); got != expected {
		t.Fatalf("failed, got: %d, want: %d", got, expected)
	}

	// 1 Post.
	System("this is a test")

	expected = 1
	if got := len(LastNStrings(1)); got != expected {
		t.Fatalf("failed, got: %d, want: %d", got, expected)
	}
	if got := len(LastNStrings(5)); got != expected {
		t.Fatalf("failed, got: %d, want: %d", got, expected)
	}
	if got := len(LastNStrings(100)); got != expected {
		t.Fatalf("failed, got: %d, want: %d", got, expected)
	}

	// 5 Posts.
	for i := 0; i < 4; i++ {
		System("%d", i)
	}

	expected = 1
	if got := len(LastNStrings(1)); got != expected {
		t.Fatalf("failed, got: %d, want: %d", got, expected)
	}
	expected = 2
	if got := len(LastNStrings(2)); got != expected {
		t.Fatalf("failed, got: %d, want: %d", got, expected)
	}
	expected = 5
	if got := len(LastNStrings(5)); got != expected {
		t.Fatalf("failed, got: %d, want: %d", got, expected)
	}
	if got := len(LastNStrings(100)); got != expected {
		t.Fatalf("failed, got: %d, want: %d", got, expected)
	}

	// 100 Posts.
	for i := 0; i < 95; i++ {
		System("%d", i+10)
	}

	expected = 1
	if got := len(LastNStrings(1)); got != expected {
		t.Fatalf("failed, got: %d, want: %d", got, expected)
	}
	expected = 2
	if got := len(LastNStrings(2)); got != expected {
		t.Fatalf("failed, got: %d, want: %d", got, expected)
	}
	expected = 5
	if got := len(LastNStrings(5)); got != expected {
		t.Fatalf("failed, got: %d, want: %d", got, expected)
	}
	expected = 100
	if got := len(LastNStrings(100)); got != expected {
		t.Fatalf("failed, got: %d, want: %d", got, expected)
	}
}
