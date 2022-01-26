package duplicate

import (
	"time"

	"gocv.io/x/gocv"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type Duplicate struct {
	Value int
	time.Time
	gocv.Mat
	region  gocv.Mat
	Counted bool
}

func New(value int, mat, region gocv.Mat) *Duplicate {
	return &Duplicate{
		value,
		time.Now(),
		mat.Clone(),
		region,
		false,
	}
}

func (d *Duplicate) Close() {
	err := d.Mat.Close()
	if err != nil {
		log.Warn().Err(err).Object("duplicate", d).Msg("failed to close duplicate matrix")
	}

	err = d.region.Close()
	if err != nil {
		log.Warn().Err(err).Object("duplicate", d).Msg("failed to close duplicate region matrix")
	}
}

func (d *Duplicate) Of(d2 *Duplicate) bool {
	if d == nil || d2 == nil {
		return false
	}

	if d2.Empty() || d.Empty() {
		return false
	}

	// if d.Value != d2.Value {
	//	return false
	//}

	if d.Time.Sub(d2.Time) > time.Second*3 {
		return false
	}

	m2 := gocv.NewMat()

	gocv.MatchTemplate(d.Mat, d2.region, &m2, gocv.TmCcoeffNormed, gocv.NewMat())
	_, maxc, _, _ := gocv.MinMaxLoc(m2)

	return maxc >= 0.85
}

func (d *Duplicate) MarshalZerologObject(e *zerolog.Event) {
	e.Time("time", d.Time).Int("value", d.Value)
}
