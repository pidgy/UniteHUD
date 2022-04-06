package spinner

import "time"

type Spinner struct {
	pos    int
	bytes  []string
	ticker *time.Ticker
	ready  bool
}

func (s *Spinner) Stop() {
	s.ticker.Stop()
}

func (s *Spinner) Next() string {
	s.ready = true
	return s.bytes[s.pos]
}

func Running() *Spinner {
	return defaultWithBytes([]string{"« ", " «", "  ", " »", "» ", "  "})
}

func Recording() *Spinner {
	return withDelayAndBytes(time.Millisecond*200, []string{"•", " "})
}

func Stopped() *Spinner {
	return defaultWithBytes([]string{"×", "+"})
}

func withDelayAndBytes(d time.Duration, b []string) *Spinner {
	s := &Spinner{
		bytes:  b,
		ticker: time.NewTicker(d),
	}

	go s.spin()

	return s
}

func defaultWithBytes(b []string) *Spinner {
	return withDelayAndBytes(time.Millisecond*100, b)
}

func (s *Spinner) spin() {
	for range s.ticker.C {
		if s.ready {
			s.pos = (s.pos + 1) % len(s.bytes)
		}
	}
}
