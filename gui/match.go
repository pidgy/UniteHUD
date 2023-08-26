package gui

import (
	"fmt"
	"strings"

	"gocv.io/x/gocv"

	"github.com/pidgy/unitehud/config"
	"github.com/pidgy/unitehud/gui/visual/area"
	"github.com/pidgy/unitehud/match"
	"github.com/pidgy/unitehud/nrgba"
	"github.com/pidgy/unitehud/server"
	"github.com/pidgy/unitehud/state"
	"github.com/pidgy/unitehud/team"
	"github.com/pidgy/unitehud/video"
)

func (g *GUI) matchEnergy(a *area.Area) (bool, error) {
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

func (g *GUI) matchKOs(a *area.Area) (bool, error) {
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

	_, r, e := match.Matches(matrix, img, config.Current.Templates["ko"][team.Game.Name])
	if r != match.Found {
		a.NRGBA = area.Miss
		a.Subtext = strings.Title(r.String())
		return false, nil
	}
	a.NRGBA = area.Match
	a.Subtext = state.EventType(e).String()

	return r == match.Found, nil
}

func (g *GUI) matchObjectives(a *area.Area) (bool, error) {
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

	_, r, e := match.Matches(matrix, img, config.Current.Templates["secure"][team.Game.Name])
	if r != match.Found {
		a.NRGBA = area.Miss
		a.Subtext = strings.Title(r.String())
		return false, nil
	}
	a.NRGBA = area.Match
	a.Subtext = state.EventType(e).String()

	return r == match.Found, nil
}

func (g *GUI) matchScore(a *area.Area) (bool, error) {
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

	for _, t := range config.Current.Templates["scored"] {
		_, r, score := match.Matches(matrix, img, t)
		switch r {
		case match.Found, match.Duplicate:
			a.NRGBA = area.Match
			a.Subtext = fmt.Sprintf("%d", score)

			return true, nil
		case match.NotFound:
			a.NRGBA = area.Miss
			a.Subtext = fmt.Sprintf("%s", strings.Title(r.String()))
		case match.Missed:
			a.NRGBA = nrgba.DarkerYellow.Alpha(0x99)
			a.Subtext = fmt.Sprintf("%d?", score)
		case match.Invalid:
			a.NRGBA = area.Miss
			a.Subtext = fmt.Sprintf("%s", strings.Title(r.String()))
		}
	}

	return false, nil
}

func (g *GUI) matchState(a *area.Area) (bool, error) {
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

	_, r, e := match.Matches(matrix, img, config.Current.Templates["game"][team.Game.Name])
	if r == match.Found {
		a.Subtext = state.EventType(e).String()
		a.NRGBA = area.Match
		return true, nil
	}

	a.Subtext = strings.Title(r.String())
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

func (g *GUI) matchTime(a *area.Area) (bool, error) {
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

	s, k := match.Time(matrix, img)
	if s != 0 {
		a.NRGBA = area.Match
		a.Subtext = k
		return true, nil
	}

	a.NRGBA = area.Miss
	a.Subtext = "Not Found"

	return false, nil
}
