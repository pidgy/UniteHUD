package history

import (
	"time"

	"github.com/pidgy/unitehud/notify"
	"github.com/pidgy/unitehud/rgba"
)

type match struct {
	purple, orange, self int
	time.Time
}

var history = []match{}

func Add(purple, orange, self int) {
	history = append(history, match{
		purple: purple,
		orange: orange,
		self:   self,
		Time:   time.Now(),
	})
}

func Dump() {
	if len(history) == 0 {
		notify.Feed(rgba.White, "No match history currently exists...")
		return
	}

	notify.Feed(rgba.White, "Match History")

	for _, h := range history {
		color := rgba.Green
		result := ""
		switch {
		case h.purple > h.orange:
			result = "Win »"
			color = rgba.Green
		case h.orange > h.purple:
			result = "Loss «"
			color = rgba.DarkRed
		case h.orange == h.purple:
			result = "Tie ¤"
			color = rgba.Yellow
		}

		notify.Append(color, "(%s) %s %d - %d - %d", h.Time.Format(time.Kitchen), result, h.purple, h.orange, h.self)
	}
}
