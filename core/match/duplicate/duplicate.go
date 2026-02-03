package duplicate

// TODO: Add go style comments that reflect the purpose of each type, function, var, and const.

import (
	"time"

	"gocv.io/x/gocv"

	"github.com/pidgy/unitehud/core/notify"
)

const delay = time.Millisecond * 4000

// Duplicate defines Duplicate behavior and state.
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

// New returns a new instance.
func New(value int, mat, region gocv.Mat) *Duplicate {
	return &Duplicate{
		Value:  value,
		Time:   time.Now(),
		Mat:    mat.Clone(),
		region: region,
	}
}

// Close closes the target.
func (d *Duplicate) Close() {
	if d == nil {
		return
	}

	err := d.Mat.Close()
	if err != nil {
		notify.Warn("[Duplicate] <ini:f:close> duplicate matrix")
	}

	err = d.region.Close()
	if err != nil {
		notify.Warn("[Duplicate] <ini:f:close> duplicate region")
	}
}


func (d *Duplicate) Of(d2 *Duplicate) (bool, string) {
	if d.Value == 0 || d2.Value == 0 {
		return false, "Zero Equality"
	}

	if d.Value == -1 {
		return false, "Negative Comparison"
	}

	if d == nil || d2 == nil {
		return false, "Nil Equality"
	}

	if d.Empty() || d2.Empty() {
		return false, "Empty Equality"
	}

	// Fallacy to think we'll capture the same values everytime... maybe one day?
	if d.Value != d2.Value {
		return false, "Inequality"
	}

	delta := d2.Time.Sub(d.Time)
	if delta > delay {
		d2.Potential = true
		return false, "Long Delay"
	}

	if d.Counted {
		d2.Potential = true
		return true, "Counted"
	}

	mat := gocv.NewMat()
	defer mat.Close()

	gocv.MatchTemplate(d.region, d2.region, &mat, gocv.TmCcoeffNormed, gocv.NewMat())

	_, maxc, _, _ := gocv.MinMaxLoc(mat)
	if maxc > 0.91 {
		d2.Potential = true
		return true, "Min. Acceptance"
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
