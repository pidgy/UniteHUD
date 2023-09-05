package fps

import (
	"fmt"
	"time"
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

// New will return a new FPS tracker.
func New() *FPS {
	return &FPS{
		window:  time.Second,
		counter: 0,
		start:   time.Now(),
	}
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
