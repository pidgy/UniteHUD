package duplicate

import (
	"time"

	"gocv.io/x/gocv"

	"github.com/pidgy/unitehud/core/notify"
)

const delay = time.Millisecond * 2200

type Duplicate struct {
	Value int
	time.Time
	gocv.Mat
	region  gocv.Mat
	Counted bool

	Captured  bool
	Replaces  int
	Potential bool
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
		notify.Warn("[Duplicate] Failed to close duplicate matrix")
	}

	err = d.region.Close()
	if err != nil {
		notify.Warn("[Duplicate] Failed to close duplicate region")
	}
}

func (d *Duplicate) Of(d2 *Duplicate) (bool, string) {
	if d.Value == 0 || d2.Value == 0 {
		return false, "zero-equality"
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
	if delta > delay {
		d2.Potential = true
		return false, "long-delay"
	}
	if delta < delay && d.Value != -1 && d.Value == d2.Value && d.Counted {
		d2.Potential = true
		return true, "short-delay,positive-equality,counted"
	}

	mat := gocv.NewMat()
	defer mat.Close()

	gocv.MatchTemplate(d.region, d2.region, &mat, gocv.TmCcoeffNormed, gocv.NewMat())

	_, maxc, _, _ := gocv.MinMaxLoc(mat)
	if maxc > 0.91 {
		d2.Potential = true
		return true, "max-gt-91"
	}

	return false, ""
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
		notify.Warn("[Duplicate] Potential duplicate override detected (-%d)/(+%d)", prev.Value, d.Value)
		return true
	}
}
