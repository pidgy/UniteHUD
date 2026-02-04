package state

// Package state tracks match events and provides helpers to query recent history.

import (
	"fmt"
	"time"

	"github.com/pidgy/unitehud/core/team"
	"github.com/pidgy/unitehud/exe"
)

// See [EventType.String] for literal descriptions.
const (
	// Custom represents a user-defined event type.
	Custom EventType = iota - 2
	// Nothing represents the absence of a meaningful event.
	Nothing
	// PreScore represents a pre-score state.
	PreScore
	// PostScore represents a post-score state.
	PostScore
	// Killed represents a defeat event.
	Killed
	// KilledWithPoints represents a defeat while holding points.
	KilledWithPoints
	// KilledWithoutPoints represents a defeat while holding no points.
	KilledWithoutPoints
	// MatchStarting represents the match starting.
	MatchStarting
	// MatchEnding represents the match ending.
	MatchEnding
	// HoldingEnergy represents the local player holding energy.
	HoldingEnergy
	// PurpleBaseOpen represents the purple base opening.
	PurpleBaseOpen
	// OrangeBaseOpen represents the orange base opening.
	OrangeBaseOpen
	// PurpleBaseClosed represents the purple base closing.
	PurpleBaseClosed
	// OrangeBaseClosed represents the orange base closing.
	OrangeBaseClosed
	// OrangeScore represents an orange team score.
	OrangeScore
	// PurpleScore represents a purple team score.
	PurpleScore
	// FirstScored represents the first score of the match.
	FirstScored
	// OrangeScoreMissed represents an orange team score miss.
	OrangeScoreMissed
	// PurpleScoreMissed represents a purple team score miss.
	PurpleScoreMissed
	// RegielekiSecureOrange represents orange securing Regieleki.
	RegielekiSecureOrange
	// RegielekiSecurePurple represents purple securing Regieleki.
	RegielekiSecurePurple
	// SelfScoreIndicator represents a self score indicator event.
	SelfScoreIndicator
	// ScoreOverride represents an override of the score.
	ScoreOverride
	// ObjectivePresent represents an objective being present.
	ObjectivePresent
	// ObjectiveReachedOrange represents an objective reaching the orange base.
	ObjectiveReachedOrange
	// ObjectiveReachedPurple represents an objective reaching the purple base.
	ObjectiveReachedPurple
	// ServerStarted represents the server starting.
	ServerStarted
	// ServerStopped represents the server stopping.
	ServerStopped
	// RegiceSecureOrange represents orange securing Regice.
	RegiceSecureOrange
	// RegiceSecurePurple represents purple securing Regice.
	RegiceSecurePurple
	// RegirockSecureOrange represents orange securing Regirock.
	RegirockSecureOrange
	// RegirockSecurePurple represents purple securing Regirock.
	RegirockSecurePurple
	// RegisteelSecureOrange represents orange securing Registeel.
	RegisteelSecureOrange
	// RegisteelSecurePurple represents purple securing Registeel.
	RegisteelSecurePurple
	// KOPurple represents a purple team KO.
	KOPurple
	// KOStreakPurple represents a purple KO streak.
	KOStreakPurple
	// KOOrange represents an orange team KO.
	KOOrange
	// KOStreakOrange represents an orange KO streak.
	KOStreakOrange
	// FinalObjectiveGroudonSecurePurple represents purple securing final Groudon.
	FinalObjectiveGroudonSecurePurple
	// FinalObjectiveGroudonSecureOrange represents orange securing final Groudon.
	FinalObjectiveGroudonSecureOrange
	// FinalObjectiveKyogreSecureKO represents a KO securing final Kyogre.
	FinalObjectiveKyogreSecureKO
	// FinalObjectiveKyogreSecurePurple represents purple securing final Kyogre.
	FinalObjectiveKyogreSecurePurple
	// FinalObjectiveKyogreSecureOrange represents orange securing final Kyogre.
	FinalObjectiveKyogreSecureOrange
	// FinalObjectiveRayquazaSecurePurple represents purple securing final Rayquaza.
	FinalObjectiveRayquazaSecurePurple
	// FinalObjectiveRayquazaSecureOrange represents orange securing final Rayquaza.
	FinalObjectiveRayquazaSecureOrange
	// SurrenderOrange represents orange surrendering.
	SurrenderOrange
	// SurrenderPurple represents purple surrendering.
	SurrenderPurple
	// RegidragoSecureKO represents a KO securing Regidrago.
	RegidragoSecureKO
	// RegidragoSecurePurple represents purple securing Regidrago.
	RegidragoSecurePurple
	// RegidragoSecureOrange represents orange securing Regidrago.
	RegidragoSecureOrange
)

// Event is a recorded match event with timing and metadata.
type Event struct {
	EventType
	time.Time
	Clock  string
	Value  int
	Vetoed bool

	Verified bool
}

// EventType indicates the kind of event that occurred.
type EventType int

var (
	// Events holds the current event history with the newest event first.
	Events = []*Event{{EventType: Nothing, Time: exe.Uptime}}

	// past is a copy of all events recorded, newest first.
	past = Events
)

// Add records an event and prepends it to the history.
func Add(e EventType, clock string, points int) {
	event := &Event{
		EventType: e,
		Time:      time.Now(),
		Clock:     clock,
		Value:     points,
	}

	Events = append([]*Event{event}, Events...)
	past = append([]*Event{event}, past...)
}

// Clear resets event history to the initial nothing event.
func Clear() {
	Events = []*Event{{EventType: Nothing, Time: exe.Uptime}}
}

// Dump returns a formatted event history and whether any data exists.
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

// Eq reports whether two events are equal in type and metadata.
func (e *Event) Eq(e2 *Event) bool {
	if e2 == nil {
		return e == nil
	}

	return e.EventType == e2.EventType &&
		e.Value == e2.Value &&
		e.Vetoed == e2.Vetoed &&
		e.Verified == e2.Verified
}

// String formats the event with its timestamp, clock, and type.
func (e *Event) String() string {
	s := fmt.Sprintf("[%02d:%02d:%02d] [Event] [%s] %s", e.Time.Hour(), e.Time.Minute(), e.Time.Second(), e.Clock, e.EventType)
	if e.Value > 0 {
		s = fmt.Sprintf("%s %d", s, e.Value)
	}
	return s
}

// Strip formats a compact event string without timestamp.
func (e *Event) Strip() string {
	s := fmt.Sprintf("[%s] %s", e.Clock, e.EventType)
	if e.Value > 0 {
		s = fmt.Sprintf("%s %d", s, e.Value)
	}
	return s
}

// Before reports whether this event type appears earlier than the other in history.
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

// Either reports whether this event type equals any of the provided types.
func (this EventType) Either(those ...EventType) bool {
	for _, that := range those {
		if this == that {
			return true
		}
	}
	return false
}

// Int returns the integer value of the event type.
func (e EventType) Int() int {
	return int(e)
}

// ObjectiveString returns a short objective identifier for final objective events.
func (e EventType) ObjectiveString() string {
	switch e {
	case FinalObjectiveGroudonSecurePurple, FinalObjectiveGroudonSecureOrange:
		return "groudon"
	case FinalObjectiveKyogreSecurePurple, FinalObjectiveKyogreSecureOrange:
		return "kyogre"
	case FinalObjectiveRayquazaSecureOrange, FinalObjectiveRayquazaSecurePurple:
		return "rayquaza"
	default:
		return fmt.Sprintf("unknown_objective_string_%d", e.Int())
	}
}

// Occured returns the most recent event of this type within the duration.
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

// String returns a human-readable description of the event type.
func (e EventType) String() string {
	switch e {
	case Custom:
		return "Custom"
	case Nothing:
		return "Nothing"
	case PreScore:
		return "Pre-score"
	case PostScore:
		return "Post-score"
	case Killed:
		return "Defeated"
	case KilledWithPoints:
		return "Defeated with points"
	case KilledWithoutPoints:
		return "Defeated without points"
	case MatchStarting:
		return "Match starting"
	case MatchEnding:
		return "Match Ending"
	case HoldingEnergy:
		return "[Self] Holding energy"
	case PurpleBaseOpen:
		return "[Purple] Base open"
	case OrangeBaseOpen:
		return "[Orange] Base open"
	case PurpleBaseClosed:
		return "[Purple] Closed"
	case OrangeBaseClosed:
		return "[Orange] Closed"
	case PurpleScore:
		return "[Purple] Scored"
	case OrangeScore:
		return "[Orange] Scored"
	case FirstScored:
		return "First score"
	case OrangeScoreMissed:
		return "[Orange] Score missed"
	case PurpleScoreMissed:
		return "[Purple] Score missed"
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
	case RegidragoSecureKO:
		return "Regidrago KO"
	case RegidragoSecurePurple:
		return "[Purple] Regidrago"
	case RegidragoSecureOrange:
		return "[Orange] Regidrago"
	case SelfScoreIndicator:
		return `Self-Score Indicator"`
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
	case FinalObjectiveGroudonSecurePurple:
		return "[Purple] Groudon"
	case FinalObjectiveGroudonSecureOrange:
		return "[Orange] Groudon"
	case FinalObjectiveKyogreSecureKO:
		return "Kyogre KO"
	case FinalObjectiveKyogreSecurePurple:
		return "[Purple] Kyogre"
	case FinalObjectiveKyogreSecureOrange:
		return "[Orange] Kyogre"
	case FinalObjectiveRayquazaSecurePurple:
		return "[Purple] Rayquaza"
	case FinalObjectiveRayquazaSecureOrange:
		return "[Orange] Rayquaza"
	case SurrenderPurple:
		return "[Purple] Surrendered"
	case SurrenderOrange:
		return "[Orange] Surrendered"
	default:
		return fmt.Sprintf("Unknown State: %d", e.Int())
	}
}

// Team returns the team associated with the event type.
func (this EventType) Team() *team.Team {
	switch this {
	case SelfScoreIndicator, PreScore, PostScore, Killed, KilledWithPoints, KilledWithoutPoints, HoldingEnergy:
		return team.Self
	case OrangeScore, RegielekiSecureOrange, RegiceSecureOrange, RegirockSecureOrange, RegisteelSecureOrange, RegidragoSecureOrange, FinalObjectiveGroudonSecureOrange, FinalObjectiveKyogreSecureOrange, FinalObjectiveRayquazaSecureOrange:
		return team.Orange
	case FirstScored, PurpleScore, RegielekiSecurePurple, RegiceSecurePurple, RegirockSecurePurple, RegisteelSecurePurple, RegidragoSecurePurple, FinalObjectiveGroudonSecurePurple, FinalObjectiveKyogreSecurePurple, FinalObjectiveRayquazaSecurePurple:
		return team.Purple
	default:
		return team.Game
	}
}

// First returns the earliest event of the given type within the duration.
func First(e EventType, since time.Duration) *Event {
	events := []*Event{}

	for _, event := range Events {
		if time.Since(event.Time) > since {
			break
		}

		if event.EventType == e {
			events = append(events, event)
		}
	}

	if len(events) > 0 {
		return events[len(events)-1]
	}
	return nil
}

// Idle returns the duration since the most recent event.
func Idle() time.Duration {
	if len(Events) < 2 {
		return 0
	}

	return time.Since(Events[0].Time)
}

// Last returns the most recent event.
func Last() *Event {
	return Events[0]
}

// Occured reports whether any of the given event types occurred recently.
func Occured(since time.Duration, e ...EventType) bool {
	for _, e := range e {
		event := e.Occured(since)
		if event != nil {
			return true
		}
	}
	return false
}

// Past returns events of the given types within the duration from the full history.
func Past(since time.Duration, es ...EventType) []*Event {
	events := []*Event{}

	for _, event := range past {
		if time.Since(event.Time) > since {
			return events
		}

		for _, e := range es {
			if e == event.EventType {
				events = append(events, event)
			}
		}
	}

	return events
}

// Recent reports whether the given event type exists in history.
func Recent(e EventType) bool {
	for i := len(Events) - 1; i >= 0; i-- {
		if Events[i].EventType == e {
			return true
		}
	}
	return false
}

// ScoredBy maps a team name to the corresponding score event type.
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

// ScoreMissedBy maps a team name to the corresponding score-missed event type.
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

// Since returns the duration since the most recent event of the given type.
func Since(e EventType) time.Duration {
	for _, event := range Events {
		if event.EventType == e {
			return time.Since(event.Time)
		}
	}
	return 0
}

// Start returns the first event in history.
func Start() *Event {
	if len(Events) == 0 {
		return &Event{}
	}
	return Events[0]
}

// Strings returns compact event strings within the duration.
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
