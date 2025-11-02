package state

import (
	"testing"
	"time"
)

func TestPast(t *testing.T) {
	Add(FinalObjectiveSecureOrange, "10:00", 0)
	Add(RegiceSecureOrange, "10:00", 0)
	Add(RegirockSecureOrange, "10:00", 0)
	Add(RegisteelSecureOrange, "10:00", 0)
	Add(RegielekiSecureOrange, "10:00", 0)

	if len(Events) != 6 { // 5 + state.Nothing
		t.Fatal("expected 5 events")
	}

	if len(Past(time.Second, RegiceSecureOrange)) != 1 {
		t.Fatal("expected only 1 event")
	}

	if len(Past(time.Second, MatchStarting)) != 0 {
		t.Fatal("expected no events")
	}
}
