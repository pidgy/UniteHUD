package fps

import (
	"fmt"
	"time"

	"github.com/pidgy/unitehud/core/notify"
)

const Sixty = time.Duration(16.7 * float64(time.Millisecond))

// Hz handles frames-per-second arithmetic using interval ticks.
type Hz struct {
	window  time.Duration
	counter int
	start   time.Time

	ticks struct {
		start, end time.Time
		count      int
		ps         float64
	}
}

type Loop struct {
	stop bool
	*LoopOptions
}

type LoopOptions struct {
	Async  bool
	FPS    int
	Render func(min, max, avg time.Duration) (close bool)

	stats struct {
		min, max, avg,
		total,
		unique time.Duration
	}

	interval time.Duration
	syncq    chan bool
}

// NewHz will return a new FPS tracker.
func NewHz() *Hz {
	return &Hz{
		window:  time.Second,
		counter: 0,
		start:   time.Now(),
	}
}

func NewLoop(o *LoopOptions) *Loop {
	l := &Loop{
		LoopOptions: o,
	}
	if l.FPS == 0 {
		l.FPS = 60
	}
	if l.Render == nil {
		l.Render = func(min, max, avg time.Duration) (close bool) { return true }
	}

	l.interval = time.Second / time.Duration(l.FPS)
	l.syncq = make(chan bool)

	l.start()

	return l
}

// PS returns the average number of ticks per-second count.
func (h *Hz) PS() float64 {
	return h.ticks.ps
}

// Ticks returns the number of ticks that occured in the active window.
func (h *Hz) Ticks() int {
	return h.ticks.count
}

// String returns the average number of ticks per-second count as a string.
func (h *Hz) String() string {
	return fmt.Sprintf("%d", int(h.ticks.ps))
}

// Increment and adjust the total frame count and per-second count respectively.
func (h *Hz) Tick(t time.Time) {
	defer h.count()

	interval := t.Sub(h.start)
	if interval <= h.window {
		return
	}
	defer h.restart(t)

	h.ticks.start = h.start
	h.ticks.end = t
	h.ticks.count = h.counter
	h.ticks.ps = float64(h.counter) / interval.Seconds()
}

func (h *Hz) count() {
	h.counter++
}

func (h *Hz) restart(t time.Time) {
	h.start = t
	h.counter = 0
}

func (l *Loop) Stop() { l.stop = true }

func (l *Loop) start() {
	go func() {
		defer close(l.syncq)

		notify.Debug("FPS: Loop starting at %d FPS/%dms", l.FPS, l.interval.Milliseconds())
		defer notify.Debug("FPS: Stopping loop at %d FPS/%dms", l.FPS, l.interval.Milliseconds())

		for ; !l.stop; time.Sleep(l.interval) {
			close := l.render()
			if close {
				return
			}
		}
	}()

	if !l.Async {
		<-l.syncq
	}
}

func (l *Loop) render() (close bool) {
	defer l.track(time.Now())

	close = l.Render(l.stats.min, l.stats.max, l.stats.avg)

	return
}

func (l *Loop) track(now time.Time) {
	d := time.Since(now)

	if d > l.stats.max {
		l.stats.max = d
	}
	if d < l.stats.min || l.stats.min == 0 {
		l.stats.min = d
	}

	l.stats.total += d
	l.stats.unique++
	l.stats.avg =
		l.stats.total /
			l.stats.unique
}
