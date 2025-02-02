package ui

import (
	"fmt"

	"gocv.io/x/gocv"

	"github.com/pidgy/unitehud/avi/video"
	"github.com/pidgy/unitehud/core/config"
	"github.com/pidgy/unitehud/core/match"
	"github.com/pidgy/unitehud/core/rgba/nrgba"
	"github.com/pidgy/unitehud/core/server"
	"github.com/pidgy/unitehud/core/state"
	"github.com/pidgy/unitehud/core/team"
	"github.com/pidgy/unitehud/gui/ux/area"
)

func (g *GUI) matchEnergy(a *area.Widget) (bool, error) {
	if !g.Preview {
		a.NRGBA = area.Locked
		return false, nil
	}

	a.NRGBA = area.Miss

	img, err := video.CaptureRect(a.Rectangle())
	if err != nil {
		return false, err
	}

	matrix, err := gocv.ImageToMatRGB(img)
	if err != nil {
		return false, err
	}
	defer matrix.Close()

	result, _, score := match.Energy(matrix, img)
	switch result {
	case match.Found, match.Duplicate:
		a.NRGBA = area.Match
		a.Subtext = fmt.Sprintf("%d", score)

	case match.NotFound:
		a.NRGBA = area.Miss
	case match.Missed:
		a.NRGBA = nrgba.DarkerYellow.Alpha(0x99)
		a.Subtext = fmt.Sprintf("%d?", score)
	case match.Invalid:
		a.NRGBA = area.Miss
	}

	m, r := match.SelfScore(matrix, img)
	switch r {
	case match.Found:
		if state.EventType(m.Template.Value) == state.PreScore {
			a.NRGBA = area.Match
			a.Subtext = "Scoring"
		} else {
			a.NRGBA = area.Match
			a.Subtext = "Scored"
		}
	case match.Invalid:
		a.NRGBA = area.Miss
		a.Subtext = "Invalid Aeos"
	}

	return r == match.Found || result == match.Found, nil
}

func (g *GUI) matchKOs(a *area.Widget) (bool, error) {
	if !g.Preview {
		a.NRGBA = area.Locked
		return false, nil
	}

	img, err := video.CaptureRect(a.Rectangle())
	if err != nil {
		return false, nil
	}

	matrix, err := gocv.ImageToMatRGB(img)
	if err != nil {
		return false, nil
	}
	defer matrix.Close()

	m, r := match.Matches(matrix, img, config.Current.TemplatesKO(team.Game.Name))
	if r != match.Found {
		a.NRGBA = area.Miss
		a.Subtext = r.String()
		return false, nil
	}
	a.NRGBA = area.Match
	a.Subtext = state.EventType(m.Value).String()

	return r == match.Found, nil
}

func (g *GUI) matchObjectives(a *area.Widget) (bool, error) {
	if !g.Preview {
		a.NRGBA = area.Locked
		return false, nil
	}

	img, err := video.CaptureRect(a.Rectangle())
	if err != nil {
		return false, err
	}

	matrix, err := gocv.ImageToMatRGB(img)
	if err != nil {
		return false, err
	}
	defer matrix.Close()

	m, r := match.Matches(matrix, img, config.Current.TemplatesSecure(team.Game.Name))
	if r != match.Found {
		a.NRGBA = area.Miss
		a.Subtext = r.String()
		return false, nil
	}
	a.NRGBA = area.Match
	a.Subtext = state.EventType(m.Value).String()

	return r == match.Found, nil
}

func (g *GUI) matchPressButtonToScore(a *area.Widget) (bool, error) {
	if !g.Preview {
		a.NRGBA = area.Locked
		return false, nil
	}

	img, err := video.CaptureRect(a.Rectangle())
	if err != nil {
		return false, err
	}

	matrix, err := gocv.ImageToMatRGB(img)
	if err != nil {
		return false, err
	}
	defer matrix.Close()

	a.NRGBA = area.Miss

	_, r := match.SelfScoreIndicator(matrix, img)
	if r == match.Found {
		a.NRGBA = area.Match
	}

	a.Subtext = r.String()

	return r == match.Found, nil
}

func (g *GUI) matchScore(a *area.Widget) (bool, error) {
	if !g.Preview {
		a.NRGBA = area.Locked
		return false, nil
	}

	img, err := video.CaptureRect(a.Rectangle())
	if err != nil {
		return false, err
	}

	matrix, err := gocv.ImageToMatRGB(img)
	if err != nil {
		return false, err
	}
	defer matrix.Close()

	for _, t := range config.Current.TemplatesScoredAll() {
		m, r := match.Matches(matrix, img, t)
		switch r {
		case match.Found, match.Duplicate:
			a.NRGBA = area.Match
			a.Subtext = fmt.Sprintf("%d", m.Value)

			return true, nil
		case match.NotFound:
			a.NRGBA = area.Miss
			a.Subtext = fmt.Sprintf("%s", r.String())
		case match.Missed:
			a.NRGBA = nrgba.DarkerYellow.Alpha(0x99)
			a.Subtext = fmt.Sprintf("%d?", m.Value)
		case match.Invalid:
			a.NRGBA = area.Miss
			a.Subtext = fmt.Sprintf("%s", r.String())
		}
	}

	return false, nil
}

func (g *GUI) matchState(a *area.Widget) (bool, error) {
	if !g.Preview {
		a.NRGBA = area.Locked
		return false, nil
	}

	img, err := video.CaptureRect(a.Rectangle())
	if err != nil {
		return false, err
	}

	matrix, err := gocv.ImageToMatRGB(img)
	if err != nil {
		return false, err
	}
	defer matrix.Close()

	templates := append(config.Current.TemplatesStarting(), append(config.Current.TemplatesEnding(), config.Current.TemplatesSurrender()...)...)

	m, r := match.Matches(matrix, img, templates)
	if r == match.Found {
		a.Subtext = state.EventType(m.Value).String()
		a.NRGBA = area.Match
		return true, nil
	}

	a.Subtext = r.String()
	a.NRGBA = area.Miss

	switch {
	case server.IsFinalStretch():
		a.Subtext = "Final Stretch"
		a.NRGBA = area.Match

		return true, nil
	case server.Clock() != "00:00":
		a.Subtext = "In Match"
		a.NRGBA = area.Match

		return true, nil
	}

	return false, nil
}

func (g *GUI) matchTime(a *area.Widget) (bool, error) {
	if !g.Preview {
		a.NRGBA = area.Locked
		return false, nil
	}

	img, err := video.CaptureRect(a.Rectangle())
	if err != nil {
		return false, err
	}

	matrix, err := gocv.ImageToMatRGB(img)
	if err != nil {
		return false, err
	}
	defer matrix.Close()

	m, s, k := match.Time(matrix, img)
	if m+s != 0 {
		a.NRGBA = area.Match
		a.Subtext = k
		return true, nil
	}

	a.NRGBA = area.Miss
	a.Subtext = "Not Found"

	return false, nil
}
