package spinner

import "time"

// TODO: Add go style comments that reflect the purpose of each type, function, var, and const.

// Widget defines Widget behavior and state.
type Widget struct {
	pos    int
	bytes  []string
	ticker *time.Ticker
	ready  bool
}

func Running() *Widget {
	return defaultWithBytes([]string{"» ", " »", "  ", " «", "« ", "  "})
}

func Recording() *Widget {
	return withDelayAndBytes(time.Millisecond*500, []string{"•", " "})
}

// Stopped stops the operation.
func Stopped() *Widget {
	return defaultWithBytes([]string{"×", "+"})
}

// Stop stops the operation.
func (s *Widget) Stop() {
	s.ticker.Stop()
}

func (s *Widget) Next() string {
	s.ready = true
	return s.bytes[s.pos]
}

func withDelayAndBytes(d time.Duration, b []string) *Widget {
	s := &Widget{
		bytes:  b,
		ticker: time.NewTicker(d),
	}

	go s.spin()

	return s
}

func defaultWithBytes(b []string) *Widget {
	return withDelayAndBytes(time.Millisecond*500, b)
}

func (s *Widget) spin() {
	// for range s.ticker.C {
	// 	if s.ready {
	// 		s.pos = (s.pos + 1) % len(s.bytes)
	// 	}
	// }
}
