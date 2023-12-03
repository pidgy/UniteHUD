package fps

import (
	"fmt"
	"time"

	"github.com/pidgy/unitehud/core/notify"
)

// FPS handles frames-per-second arithmetic using interval ticks.
type FPS struct {
	window  time.Duration
	counter int
	start   time.Time

	frames struct {
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

// New will return a new FPS tracker.
func New() *FPS {
	return &FPS{
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

// FPS returns the average number of ticks per-second count.
func (f *FPS) FPS() float64 {
	return f.frames.ps
}

// Frames returns the number of ticks that occured in the active window.
func (f *FPS) Frames() int {
	return f.frames.count
}

// String returns the average number of ticks per-second count as a string.
func (f *FPS) String() string {
	return fmt.Sprintf("%d", int(f.frames.ps))
}

// Increment and adjust the total frame count and per-second count respectively.
func (f *FPS) Tick(t time.Time) {
	defer f.count()

	interval := t.Sub(f.start)
	if interval <= f.window {
		return
	}
	defer f.restart(t)

	f.frames.start = f.start
	f.frames.end = t
	f.frames.count = f.counter
	f.frames.ps = float64(f.counter) / interval.Seconds()
}

func (f *FPS) count() {
	f.counter++
}

func (f *FPS) restart(t time.Time) {
	f.start = t
	f.counter = 0
}

func (l *Loop) Stop() { l.stop = true }

func (l *Loop) start() {
	go func() {
		defer close(l.syncq)

		notify.Debug("FPS: Starting at %d FPS/%dms", l.FPS, l.interval.Milliseconds())
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
