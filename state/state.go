package state

import (
	"fmt"
	"time"

	"github.com/pidgy/unitehud/team"
)

type Event struct {
	EventType
	time.Time
	Clock  string
	Value  int
	Vetoed bool

	Verified bool
}

type EventType int

const (
	Nothing                 = EventType(-1)
	PreScore                = EventType(0)
	PostScore               = EventType(1)
	Killed                  = EventType(2)
	KilledWithPoints        = EventType(3)
	KilledWithoutPoints     = EventType(4)
	MatchStarting           = EventType(5)
	MatchEnding             = EventType(6)
	HoldingEnergy           = EventType(7)
	PurpleBaseOpen          = EventType(8)
	OrangeBaseOpen          = EventType(9)
	PurpleBaseClosed        = EventType(10)
	OrangeBaseClosed        = EventType(11)
	OrangeScore             = EventType(12)
	PurpleScore             = EventType(13)
	FirstScored             = EventType(14)
	OrangeScoreMissed       = EventType(15)
	PurpleScoreMissed       = EventType(16)
	RegielekiAdvancingAlly  = EventType(17)
	RegielekiAdvancingEnemy = EventType(18)
	PressButtonToScore      = EventType(19)
	ScoreOverride           = EventType(20)
	ObjectivePresent        = EventType(21)
	ObjectiveReachedOrange  = EventType(22)
	ObjectiveReachedPurple  = EventType(23)
	ServerStarted           = EventType(24)
)

var (
	Events = []*Event{}
)

func (e EventType) Int() int {
	return int(e)
}

func (e EventType) String() string {
	switch e {
	case Nothing:
		return "Nothing"
	case PreScore:
		return "Pre score"
	case PostScore:
		return "Post score"
	case Killed:
		return "Killed"
	case KilledWithPoints:
		return "Killed with points"
	case KilledWithoutPoints:
		return "Killed without points"
	case MatchStarting:
		return "Match starting"
	case MatchEnding:
		return "Match ending"
	case HoldingEnergy:
		return "Holding energy"
	case PurpleBaseOpen:
		return "Purple base open"
	case OrangeBaseOpen:
		return "Orange base open"
	case PurpleBaseClosed:
		return "Purple base closed"
	case OrangeBaseClosed:
		return "Orange base closed"
	case PurpleScore:
		return "Purple scored"
	case OrangeScore:
		return "Orange scored"
	case FirstScored:
		return "First scored"
	case OrangeScoreMissed:
		return "Orange score missed"
	case PurpleScoreMissed:
		return "Purple score missed"
	case RegielekiAdvancingAlly:
		return "Regieleki enemy secure"
	case RegielekiAdvancingEnemy:
		return "Regieleki ally secure"
	case PressButtonToScore:
		return "Press button to score"
	case ScoreOverride:
		return "Override"
	case ObjectivePresent:
		return "Objective present"
	case ObjectiveReachedOrange:
		return "Objective reached orange base"
	case ObjectiveReachedPurple:
		return "Objective reached purple base"
	case ServerStarted:
		return "Server Started"
	default:
		return fmt.Sprintf("Unknown (%d)", e.Int())
	}
}

func Add(e EventType, clock string, points int) {
	event := &Event{
		EventType: e,
		Time:      time.Now(),
		Clock:     clock,
		Value:     points,
	}

	Events = append([]*Event{event}, Events...)
}

func Clear() {
	Events = []*Event{}
}

func Dump() (string, bool) {
	if len(Events) == 0 {
		return "No event data is available to display...", false
	}

	str := "Matched Events"
	for i := len(Events) - 1; i >= 0; i-- {
		e := Events[i]

		str += fmt.Sprintf("\n[%02d:%02d:%02d] [Event] [%s] %s", e.Time.Hour(), e.Time.Minute(), e.Time.Second(), e.Clock, e.EventType)
		if e.Value != -1 {
			str += fmt.Sprintf(" (%d)", e.Value)
		}
		if e.Vetoed {
			str += " (Vetoed)"
		}
		if e.Verified {
			str += " (Verified)"
		}
	}

	return str, true
}

func (e *Event) Eq(e2 *Event) bool {
	if e2 == nil {
		return e == nil
	}

	return e.EventType == e2.EventType &&
		e.Value == e2.Value &&
		e.Vetoed == e2.Vetoed &&
		e.Verified == e2.Verified
}

func First(e EventType, since time.Duration) *Event {
	events := Past(e, since)
	if len(events) > 0 {
		return events[len(events)-1]
	}

	return nil
}

func Last(e EventType, since time.Duration) *Event {
	for _, event := range Events {
		// Have we gone too far?
		if time.Since(event.Time) > since {
			return nil
		}

		if event.EventType == e {
			return event
		}
	}

	return nil
}

func Since() time.Duration {
	if len(Events) == 0 {
		return 0
	}

	return time.Since(Events[0].Time)
}

func LastAny(since time.Duration, any ...EventType) *Event {
	for _, event := range Events {
		// Have we gone too far?
		if time.Since(event.Time) > since {
			return nil
		}

		for _, a := range any {
			if event.EventType == a {
				if time.Since(event.Time) < since {
					return event
				}
			}
		}
	}

	return nil
}

func Past(e EventType, since time.Duration) []*Event {
	events := []*Event{}

	for _, event := range Events {
		// Have we gone too far?
		if time.Since(event.Time) > since {
			return events
		}

		if event.EventType == e {
			if time.Since(event.Time) > since {
				return events
			}

			events = append(events, event)
		}
	}

	return events
}

func ScoredBy(name string) EventType {
	switch name {
	case team.Purple.Name:
		return PurpleScore
	case team.Orange.Name:
		return OrangeScore
	case team.Self.Name:
		return Nothing
	case team.First.Name:
		return FirstScored
	}
	return Nothing
}

func ScoreMissedBy(name string) EventType {
	switch name {
	case team.Purple.Name:
		return PurpleScoreMissed
	case team.Orange.Name:
		return OrangeScoreMissed
	case team.Self.Name:
		return Nothing
	case team.First.Name:
		return PurpleScoreMissed
	}
	return Nothing
}
