package gui

import (
	"fmt"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"gocv.io/x/gocv"

	"github.com/pidgy/unitehud/config"
	"github.com/pidgy/unitehud/gui/visual/area"
	"github.com/pidgy/unitehud/match"
	"github.com/pidgy/unitehud/rgba"
	"github.com/pidgy/unitehud/state"
	"github.com/pidgy/unitehud/team"
	"github.com/pidgy/unitehud/video"
)

func (g *GUI) matchEnergy(a *area.Area) {
	defer func() {
		r := recover()
		if r != nil {
			log.Error().Err(r.(error)).Msg("match balls failed")
		}
	}()

	if !g.Preview {
		a.NRGBA = area.Locked
		return
	}

	a.NRGBA = area.Miss

	img, err := video.CaptureRect(a.Rectangle())
	if err != nil {
		log.Err(err).Send()
		return
	}

	matrix, err := gocv.ImageToMatRGB(img)
	if err != nil {
		log.Err(err).Send()
		return
	}
	defer matrix.Close()

	result, _, score := match.Energy(matrix, g.Image)
	switch result {
	case match.Found, match.Duplicate:
		a.NRGBA = area.Match
		a.Text = fmt.Sprintf("\t %d", score)
	case match.NotFound:
		a.NRGBA = area.Miss
		a.Text = "Energy"
	case match.Missed:
		a.NRGBA = rgba.N(rgba.Alpha(rgba.DarkerYellow, 0x99))
		a.Text = fmt.Sprintf("\t %d?", score)
	case match.Invalid:
		a.NRGBA = area.Miss
		a.Text = "Energy"
	}

	m, result := match.SelfScore(matrix, img)
	switch result {
	case match.Found:
		if state.EventType(m.Template.Value) == state.PreScore {
			a.NRGBA = area.Match
			a.Text = "Scoring"
		} else {
			a.NRGBA = area.Match
			a.Text = "Scored"
		}
	case match.Invalid:
		a.NRGBA = area.Miss
		a.Text = "Invalid Energy"
	}
}

func (g *GUI) matchKOs(a *area.Area) {
	if !g.Preview {
		a.NRGBA = area.Locked
		return
	}

	img, err := video.CaptureRect(a.Rectangle())
	if err != nil {
		log.Err(err).Send()
		return
	}

	matrix, err := gocv.ImageToMatRGB(img)
	if err != nil {
		log.Err(err).Send()
		return
	}
	defer matrix.Close()

	_, r, e := match.Matches(matrix, img, config.Current.Templates["ko"][team.Game.Name])
	if r != match.Found {
		a.NRGBA = area.Miss
		a.Text = fmt.Sprintf("KO %s", strings.Title(r.String()))
		return
	}
	a.NRGBA = area.Match
	a.Text = fmt.Sprintf("KO %s (%s)", strings.Title(r.String()), state.EventType(e))
}

func (g *GUI) matchObjectives(a *area.Area) {
	if !g.Preview {
		a.NRGBA = area.Locked
		return
	}

	img, err := video.CaptureRect(a.Rectangle())
	if err != nil {
		log.Err(err).Send()
		return
	}

	matrix, err := gocv.ImageToMatRGB(img)
	if err != nil {
		log.Err(err).Send()
		return
	}
	defer matrix.Close()

	_, r, e := match.Matches(matrix, img, config.Current.Templates["secure"][team.Game.Name])
	if r != match.Found {
		a.NRGBA = area.Miss
		a.Text = fmt.Sprintf("Objective %s", strings.Title(r.String()))
		return
	}
	a.NRGBA = area.Match
	a.Text = fmt.Sprintf("Objective %s (%s)", strings.Title(r.String()), state.EventType(e))
}

func (g *GUI) matchScore(a *area.Area) {
	defer func() {
		r := recover()
		if r != nil {
			log.Error().Err(r.(error)).Msg("match score failed")
		}
	}()

	if !g.Preview {
		a.NRGBA = area.Locked
		return
	}

	// a.NRGBA = area.Miss
	// a.Subtext = ""

	img, err := video.CaptureRect(a.Rectangle())
	if err != nil {
		log.Err(err).Send()
		return
	}

	matrix, err := gocv.ImageToMatRGB(img)
	if err != nil {
		log.Err(err).Send()
		return
	}
	defer matrix.Close()

	for _, templates := range config.Current.Templates["scored"] {
		_, result, score := match.Matches(matrix, g.Image, templates)
		switch result {
		case match.Found, match.Duplicate:
			a.NRGBA = area.Match
			a.Subtext = fmt.Sprintf("(+%d)", score)
			return
		case match.NotFound:
			a.NRGBA = area.Miss
		case match.Missed:
			a.NRGBA = rgba.N(rgba.Alpha(rgba.DarkerYellow, 0x99))
			a.Subtext = fmt.Sprintf("(%d?)", score)
		case match.Invalid:
			a.NRGBA = area.Miss
		}

		a.Subtext = strings.Title(result.String())
	}
}

func (g *GUI) matchMap(a *area.Area) {
	defer func() {
		r := recover()
		if r != nil {
			log.Error().Err(r.(error)).Msg("match map failed")
		}
	}()

	if !g.Preview {
		a.NRGBA = area.Locked
		return
	}

	a.NRGBA = area.Miss
	a.Subtext = ""

	img, err := video.CaptureRect(a.Rectangle())
	if err != nil {
		log.Err(err).Send()
		return
	}

	matrix, err := gocv.ImageToMatRGB(img)
	if err != nil {
		log.Err(err).Send()
		return
	}
	defer matrix.Close()

	_, ok := match.MiniMap(matrix, img)
	if ok {
		a.NRGBA = area.Match
		a.Subtext = "(Found)"
	}
}

func (g *GUI) matchTime(a *area.Area) {
	defer func() {
		r := recover()
		if r != nil {
			log.Error().Err(r.(error)).Msg("match time failed")
		}
	}()

	if !g.Preview {
		a.NRGBA = area.Locked
		return
	}

	a.NRGBA = area.Miss
	a.Subtext = "(00:00)"

	img, err := video.CaptureRect(a.Rectangle())
	if err != nil {
		log.Err(err).Send()
		return
	}

	matrix, err := gocv.ImageToMatRGB(img)
	if err != nil {
		log.Err(err).Send()
		return
	}
	defer matrix.Close()

	s, k := match.Time(matrix, img)
	if s != 0 {
		a.NRGBA = area.Match
		a.Subtext = "(" + k + ")"
	}
}

func (g *GUI) while(fn func(), wait *bool) {
	for {
		time.Sleep(time.Second)

		if *wait {
			continue
		}

		fn()
	}
}
