package state

import (
	"time"

	"github.com/rs/zerolog/log"
)

type Event struct {
	EventType
	time.Time
	Clock  string
	Value  int
	Vetoed bool
}

type EventType int

const (
	Nothing             = EventType(-1)
	PreScore            = EventType(0)
	PostScore           = EventType(1)
	Killed              = EventType(2)
	KilledWithPoints    = EventType(3)
	KilledWithoutPoints = EventType(4)
	MatchStarting       = EventType(5)
	MatchEnding         = EventType(6)
	HoldingBalls        = EventType(7)
	PurpleBaseOpen      = EventType(8)
	OrangeBaseOpen      = EventType(9)
	PurpleBaseClosed    = EventType(10)
	OrangeBaseClosed    = EventType(11)
)

func (e EventType) String() string {
	switch e {
	case Nothing:
		return "Nothing"
	case PreScore:
		return "PreScore"
	case PostScore:
		return "PostScore"
	case Killed:
		return "Killed"
	case KilledWithPoints:
		return "KilledWithPoints"
	case KilledWithoutPoints:
		return "KilledWithoutPoints"
	case MatchStarting:
		return "MatchStarting"
	case MatchEnding:
		return "MatchEnding"
	case HoldingBalls:
		return "HoldingBalls"
	case PurpleBaseOpen:
		return "PurpleBaseOpen"
	case OrangeBaseOpen:
		return "OrangeBaseOpen"
	case PurpleBaseClosed:
		return "PurpleBaseClosed"
	case OrangeBaseClosed:
		return "OrangeBaseClosed"
	default:
		return "Unknown"
	}
}

var (
	Events = []*Event{}
)

func Add(e EventType, clock string, points int) {
	Events = append([]*Event{
		{
			EventType: e,
			Time:      time.Now(),
			Clock:     clock,
			Value:     points,
		},
	}, Events...)
}

func Clear() {
	Events = []*Event{}
}

func Dump() {
	for _, e := range Events {
		log.Debug().
			Stringer("type", e.EventType).
			Str("clock", e.Clock).
			Int("value", e.Value).
			Bool("veto", e.Vetoed).
			Msgf("%d:%d:%02d", e.Time.Hour(), e.Time.Minute(), e.Time.Second())
	}
}

func Past(e EventType, since time.Duration) []*Event {
	events := []*Event{}

	for _, event := range Events {
		if event.EventType == e {
			if time.Since(event.Time) > since {
				return events
			}

			events = append(events, event)
		}
	}

	return events
}

func LastScore() *Event {
	for _, event := range Events {
		if event.EventType == PostScore {
			return event
		}
	}

	return nil
}

func (e EventType) Int() int {
	return int(e)
}
