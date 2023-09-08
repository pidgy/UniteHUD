package history

import (
	"time"

	"github.com/pidgy/unitehud/notify"
	"github.com/pidgy/unitehud/nrgba"
)

type match struct {
	purple, orange, self int
	time.Time
}

var history = []match{}

func Add(purple, orange, self int) {
	history = append(history, match{
		orange: orange,
		purple: purple,
		self:   self,
		Time:   time.Now(),
	})
}

func Dump() {
	if len(history) == 0 {
		notify.Warn("No recent game history to display...")
		return
	}

	notify.System("Match History")

	for _, h := range history {
		color := nrgba.Green
		result := ""
		switch {
		case h.purple > h.orange:
			result = "Won"
			color = nrgba.Green
		case h.orange > h.purple:
			result = "Lost"
			color = nrgba.DarkRed
		case h.orange == h.purple:
			result = "Tied"
			color = nrgba.Yellow
		}

		notify.Append(color, "(%s) %s %d - %d - %d", h.Time.Format(time.Kitchen), result, h.purple, h.orange, h.self)
	}
}

func Record() {
}
