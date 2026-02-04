package history

import (
	"time"

	"github.com/pidgy/unitehud/core/notify"
	"github.com/pidgy/unitehud/core/rgba/nrgba"
)

// match stores the scoreline and timestamp for a single match.
type match struct {
	purple, orange, self int
	time.Time
}

// history holds the in-memory list of recent matches.
// history holds the in-memory list of recent match results.
var history = []match{}

// Add records a new match result in the history.
// Add records a match result in history with the current time.
func Add(purple, orange, self int) {
	history = append(history, match{
		orange: orange,
		purple: purple,
		self:   self,
		Time:   time.Now(),
	})
}

// Dump prints recent match history to the notification system.
// Dump emits the recent match history to the notification system.
func Dump() {
	if len(history) == 0 {
		notify.Warn("+/- No recent game history to display...")
		return
	}

	notify.System("+/- Match History")

	for _, h := range history {
		color := nrgba.Green
		result := ""
		symbol := "+/-"
		switch {
		case h.purple > h.orange:
			result = "Won"
			color = nrgba.Green
			symbol = "+/ "
		case h.orange > h.purple:
			result = "Lost"
			color = nrgba.DarkRed
			symbol = " /-"
		case h.orange == h.purple:
			result = "Tied"
			color = nrgba.Yellow
			symbol = " / "
		}

		notify.Append(color, "%s (%s) %s %d - %d - %d", symbol, h.Time.Format(time.Kitchen), result, h.purple, h.orange, h.self)
	}
}
