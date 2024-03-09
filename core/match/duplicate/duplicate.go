package duplicate

import (
	"time"

	"gocv.io/x/gocv"

	"github.com/pidgy/unitehud/core/notify"
)

const delay = time.Millisecond * 2000

type Duplicate struct {
	Value int
	time.Time
	gocv.Mat
	region  gocv.Mat
	Counted bool

	Captured bool
	Replaces int
}

func New(value int, mat, region gocv.Mat) *Duplicate {
	return &Duplicate{
		Value:  value,
		Time:   time.Now(),
		Mat:    mat.Clone(),
		region: region,
	}
}

func (d *Duplicate) Region() gocv.Mat {
	return d.region.Clone()
}

func (d *Duplicate) Close() {
	if d == nil {
		return
	}

	err := d.Mat.Close()
	if err != nil {
		notify.Warn("Duplicate: Failed to close duplicate matrix")
	}

	err = d.region.Close()
	if err != nil {
		notify.Warn("Duplicate: Failed to close duplicate region")
	}
}

func (d *Duplicate) Of(d2 *Duplicate) (bool, string) {
	if d2.Value == 0 {
		return false, "zero-value"
	}

	if d == nil || d2 == nil {
		return false, "nil-equality"
	}

	if d.Empty() || d2.Empty() {
		return false, "empty-equality"
	}

	// Fallacy to think we'll capture the same values everytime... maybe one day?
	if d.Value != d2.Value {
		return false, "inequality"
	}

	delta := d2.Time.Sub(d.Time)
	if delta > delay*2 {
		return false, "long-delay"
	}
	if delta < delay && d.Value != -1 && d.Value == d2.Value && d.Counted {
		return true, "short-delay,positive-equality,counted"
	}

	return d.Pixels(d2)
}

func (d *Duplicate) Overrides(prev *Duplicate) bool {
	switch {
	case d.Time.Sub(prev.Time) >= delay:
		// Too much time has passed.
		return false
	case !prev.Counted:
		// Last match was not counted.
		return false
	case d.Value <= prev.Value:
		// Unlikely we match a smaller number after.
		return false
	case d.Value/prev.Value != 10 && d.Value/prev.Value != 100:
		// Likely the first digit will match, and follow-on digits did not.
		return false
	default:
		prev.Replaces = prev.Value
		d.Replaces = prev.Value
		notify.Warn("Duplicate: Potential duplicate override detected (-%d)/(+%d)", prev.Value, d.Value)
		return true
	}
}

func (d *Duplicate) Pixels(d2 *Duplicate) (bool, string) {
	if d == nil || d2 == nil {
		return false, "nil-equality"
	}

	if d.Value == 0 || d2.Value == 0 {
		return false, "zero-equality"
	}

	mat := gocv.NewMat()
	defer mat.Close()

	gocv.MatchTemplate(d.region, d2.region, &mat, gocv.TmCcoeffNormed, gocv.NewMat())

	_, maxc, _, _ := gocv.MinMaxLoc(mat)

	return maxc > 0.91, "max-gt-91"
}
