package state

import (
	"fmt"
	"time"

	"github.com/pidgy/unitehud/core/team"
	"github.com/pidgy/unitehud/global"
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
type Types []EventType

const (
	Custom EventType = iota - 2
	Nothing
	PreScore
	PostScore
	Killed
	KilledWithPoints
	KilledWithoutPoints
	MatchStarting
	MatchEnding
	HoldingEnergy
	PurpleBaseOpen
	OrangeBaseOpen
	PurpleBaseClosed
	OrangeBaseClosed
	OrangeScore
	PurpleScore
	FirstScored
	OrangeScoreMissed
	PurpleScoreMissed
	RegielekiSecureOrange
	RegielekiSecurePurple
	PressButtonToScore
	ScoreOverride
	ObjectivePresent
	ObjectiveReachedOrange
	ObjectiveReachedPurple
	ServerStarted
	ServerStopped
	RegiceSecureOrange
	RegiceSecurePurple
	RegirockSecureOrange
	RegirockSecurePurple
	RegisteelSecureOrange
	RegisteelSecurePurple
	KOPurple
	KOStreakPurple
	KOOrange
	KOStreakOrange
	RayquazaSecureOrange
	RayquazaSecurePurple
	SurrenderOrange
	SurrenderPurple
)

var (
	Events = []*Event{{EventType: Nothing, Time: global.Uptime}}
)

func (e EventType) Int() int {
	return int(e)
}

func (e EventType) String() string {
	switch e {
	case Custom:
		return "Custom"
	case Nothing:
		return "Nothing"
	case PreScore:
		return "Pre score"
	case PostScore:
		return "Post score"
	case Killed:
		return "Defeated"
	case KilledWithPoints:
		return "Defeated with points"
	case KilledWithoutPoints:
		return "Defeated without points"
	case MatchStarting:
		return "Match Starting"
	case MatchEnding:
		return "Match Ending"
	case HoldingEnergy:
		return "Holding Energy"
	case PurpleBaseOpen:
		return "Purple Open"
	case OrangeBaseOpen:
		return "Orange Open"
	case PurpleBaseClosed:
		return "Purple Closed"
	case OrangeBaseClosed:
		return "Orange Cclosed"
	case PurpleScore:
		return "Purple Scored"
	case OrangeScore:
		return "Orange Scored"
	case FirstScored:
		return "First score"
	case OrangeScoreMissed:
		return "Orange score missed"
	case PurpleScoreMissed:
		return "Purple score missed"
	case RegielekiSecurePurple:
		return "[Purple] Regieleki"
	case RegielekiSecureOrange:
		return "[Orange] Regieleki"
	case RegiceSecurePurple:
		return "[Purple] Regice"
	case RegiceSecureOrange:
		return "[Orange] Regice"
	case RegirockSecurePurple:
		return "[Purple] Regirock"
	case RegirockSecureOrange:
		return "[Orange] Regirock"
	case RegisteelSecurePurple:
		return "[Purple] Registeel"
	case RegisteelSecureOrange:
		return "[Orange] Registeel"
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
	case ServerStopped:
		return "Server Stopped"
	case KOPurple:
		return "[Purple] +1 KO"
	case KOOrange:
		return "[Orange] +1 KO"
	case KOStreakPurple:
		return "[Purple] KO Streak"
	case KOStreakOrange:
		return "[Orange] KO Streak"
	case RayquazaSecurePurple:
		return "[Purple] Rayquaza"
	case RayquazaSecureOrange:
		return "[Orange] Rayquaza"
	case SurrenderPurple:
		return "[Purple] Surrender"
	case SurrenderOrange:
		return "[Orange] Surrender"
	default:
		return fmt.Sprintf("Unknown State: %d", e.Int())
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
	Events = []*Event{{EventType: Nothing, Time: global.Uptime}}
}

func Dump() (string, bool) {
	if len(Events) == 0 {
		return "No event data is available to display...", false
	}

	str := "Event History"
	for i := len(Events) - 1; i >= 0; i-- {
		e := Events[i]

		str = fmt.Sprintf("%s\n%s", str, e.String())
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

func (e *Event) String() string {
	return fmt.Sprintf("[%02d:%02d:%02d] [Event] [%s] %s", e.Time.Hour(), e.Time.Minute(), e.Time.Second(), e.Clock, e.EventType)
}

func (e *Event) Strip() string {
	return fmt.Sprintf("[%s] %s", e.Clock, e.EventType)
}

func (this EventType) Before(that EventType) bool {
	for i := len(Events) - 1; i >= 0; i-- {
		switch {
		case Events[i].EventType == this:
			return true
		case Events[i].EventType == that:
			return false
		}
	}

	return false
}

func (this EventType) Either(those ...EventType) bool {
	for _, that := range those {
		if this == that {
			return true
		}
	}
	return false
}

func (this EventType) Occured(since time.Duration) *Event {
	for _, event := range Events {
		if time.Since(event.Time) > since {
			return nil
		}

		if this == event.EventType {
			return event
		}
	}

	return nil
}

func Start() *Event {
	if len(Events) == 0 {
		return &Event{}
	}
	return Events[0]
}

func First(e EventType, since time.Duration) *Event {
	events := Past(e, since)
	if len(events) > 0 {
		return events[len(events)-1]
	}
	return nil
}

func Last() *Event {
	return Events[0]
}

func Occured(since time.Duration, e ...EventType) bool {
	for _, e := range e {
		event := e.Occured(since)
		if event != nil {
			return true
		}
	}
	return false
}

func Past(e EventType, since time.Duration) []*Event {
	events := []*Event{}

	for _, event := range Events {
		if time.Since(event.Time) > since {
			return events
		}

		if event.EventType == e {
			events = append(events, event)
		}
	}

	return events
}

func Recent(e EventType) bool {
	for i := len(Events) - 1; i >= 0; i-- {
		if Events[i].EventType == e {
			return true
		}
	}
	return false
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

func Since(e EventType) time.Duration {
	for _, event := range Events {
		if event.EventType == e {
			return time.Since(event.Time)
		}
	}
	return 0
}

func Idle() time.Duration {
	if len(Events) < 2 {
		return 0
	}

	return time.Since(Events[0].Time)
}

func Strings(since time.Duration) []string {
	s := []string{}

	for _, event := range Events {
		if time.Since(event.Time) > since {
			return s
		}
		s = append(s, event.Strip())
	}

	return s
}

func (this EventType) Team() *team.Team {
	switch this {
	case PressButtonToScore, PreScore, PostScore, Killed, KilledWithPoints, KilledWithoutPoints, HoldingEnergy:
		return team.Self
	case OrangeScore, RegielekiSecureOrange, RegiceSecureOrange, RegirockSecureOrange, RegisteelSecureOrange, RayquazaSecureOrange:
		return team.Orange
	case FirstScored, PurpleScore, RegielekiSecurePurple, RegiceSecurePurple, RegirockSecurePurple, RegisteelSecurePurple, RayquazaSecurePurple:
		return team.Purple
	default:
		return team.Game
	}
}
