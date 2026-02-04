package spinner

import "time"

// Widget cycles through a set of glyphs for a spinner display.
type Widget struct {
	pos    int
	bytes  []string
	ticker *time.Ticker
	ready  bool
}

// Running returns the default running spinner.
func Running() *Widget {
	return defaultWithBytes([]string{"» ", " »", "  ", " «", "« ", "  "})
}

// Recording returns a slower blinking recording spinner.
func Recording() *Widget {
	return withDelayAndBytes(time.Millisecond*500, []string{"•", " "})
}

// Stopped returns the stopped-state spinner.
func Stopped() *Widget {
	return defaultWithBytes([]string{"×", "+"})
}

// Stop halts the spinner ticker.
func (s *Widget) Stop() {
	s.ticker.Stop()
}

// Next returns the current spinner glyph.
func (s *Widget) Next() string {
	s.ready = true
	return s.bytes[s.pos]
}

// withDelayAndBytes builds a spinner with a custom tick delay and glyphs.
func withDelayAndBytes(d time.Duration, b []string) *Widget {
	s := &Widget{
		bytes:  b,
		ticker: time.NewTicker(d),
	}

	go s.spin()

	return s
}

// defaultWithBytes builds a spinner with the default delay.
func defaultWithBytes(b []string) *Widget {
	return withDelayAndBytes(time.Millisecond*500, b)
}

// spin advances the spinner position on each tick.
func (s *Widget) spin() {
	// for range s.ticker.C {
	// 	if s.ready {
	// 		s.pos = (s.pos + 1) % len(s.bytes)
	// 	}
	// }
}
