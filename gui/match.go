package gui

import (
	"fmt"
	"strings"

	"gocv.io/x/gocv"

	"github.com/pidgy/unitehud/config"
	"github.com/pidgy/unitehud/gui/visual/area"
	"github.com/pidgy/unitehud/match"
	"github.com/pidgy/unitehud/rgba"
	"github.com/pidgy/unitehud/state"
	"github.com/pidgy/unitehud/team"
	"github.com/pidgy/unitehud/video"
)

func (g *GUI) matchEnergy(a *area.Area) bool {
	if !g.Preview {
		a.NRGBA = area.Locked
		return false
	}

	a.NRGBA = area.Miss

	img, err := video.CaptureRect(a.Rectangle())
	if err != nil {
		g.ToastError(err)
		return false
	}

	matrix, err := gocv.ImageToMatRGB(img)
	if err != nil {
		g.ToastError(err)
		return false
	}
	defer matrix.Close()

	result, _, score := match.Energy(matrix, g.Image)
	switch result {
	case match.Found, match.Duplicate:
		a.NRGBA = area.Match
		a.Text = fmt.Sprintf("Aeos: %d", score)
	case match.NotFound:
		a.NRGBA = area.Miss
		a.Text = "Aeos"
	case match.Missed:
		a.NRGBA = rgba.N(rgba.Alpha(rgba.DarkerYellow, 0x99))
		a.Text = fmt.Sprintf("Aeos: %d?", score)
	case match.Invalid:
		a.NRGBA = area.Miss
		a.Text = "Aeos"
	}

	m, r := match.SelfScore(matrix, img)
	switch r {
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
		a.Text = "Invalid Aeos"
	}

	return r == match.Found
}

func (g *GUI) matchKOs(a *area.Area) bool {
	if !g.Preview {
		a.NRGBA = area.Locked
		return false
	}

	img, err := video.CaptureRect(a.Rectangle())
	if err != nil {
		g.ToastError(err)
		return false
	}

	matrix, err := gocv.ImageToMatRGB(img)
	if err != nil {
		g.ToastError(err)
		return false
	}
	defer matrix.Close()

	_, r, e := match.Matches(matrix, img, config.Current.Templates["ko"][team.Game.Name])
	if r != match.Found {
		a.NRGBA = area.Miss
		a.Text = fmt.Sprintf("KO: %s", strings.Title(r.String()))
		return false
	}
	a.NRGBA = area.Match
	a.Text = fmt.Sprintf("KO: %s", state.EventType(e))

	return r == match.Found
}

func (g *GUI) matchObjectives(a *area.Area) bool {
	if !g.Preview {
		a.NRGBA = area.Locked
		return false
	}

	img, err := video.CaptureRect(a.Rectangle())
	if err != nil {
		g.ToastError(err)
		return false
	}

	matrix, err := gocv.ImageToMatRGB(img)
	if err != nil {
		g.ToastError(err)
		return false
	}
	defer matrix.Close()

	_, r, e := match.Matches(matrix, img, config.Current.Templates["secure"][team.Game.Name])
	if r != match.Found {
		a.NRGBA = area.Miss
		a.Text = fmt.Sprintf("Objective: %s", strings.Title(r.String()))
		return false
	}
	a.NRGBA = area.Match
	a.Text = fmt.Sprintf("Objective: %s (%s)", strings.Title(r.String()), state.EventType(e))

	return r == match.Found
}

func (g *GUI) matchScore(a *area.Area) bool {
	if !g.Preview {
		a.NRGBA = area.Locked
		return false
	}

	img, err := video.CaptureRect(a.Rectangle())
	if err != nil {
		g.ToastError(err)
		return false
	}

	matrix, err := gocv.ImageToMatRGB(img)
	if err != nil {
		g.ToastError(err)
		return false
	}
	defer matrix.Close()

	for _, t := range config.Current.Templates["scored"] {
		_, r, score := match.Matches(matrix, g.Image, t)
		switch r {
		case match.Found, match.Duplicate:
			a.NRGBA = area.Match
			a.Text = fmt.Sprintf("Score: %d", score)

			return true
		case match.NotFound:
			a.NRGBA = area.Miss
			a.Text = fmt.Sprintf("Score: %s", strings.Title(r.String()))
		case match.Missed:
			a.NRGBA = rgba.N(rgba.Alpha(rgba.DarkerYellow, 0x99))
			a.Text = fmt.Sprintf("Score: %d?", score)
		case match.Invalid:
			a.NRGBA = area.Miss
			a.Text = fmt.Sprintf("Score: %s", strings.Title(r.String()))
		}
	}

	return false
}

func (g *GUI) matchTime(a *area.Area) bool {
	if !g.Preview {
		a.NRGBA = area.Locked
		return false
	}

	img, err := video.CaptureRect(a.Rectangle())
	if err != nil {
		g.ToastError(err)
		return false
	}

	matrix, err := gocv.ImageToMatRGB(img)
	if err != nil {
		g.ToastError(err)
		return false
	}
	defer matrix.Close()

	s, k := match.Time(matrix, img)
	if s != 0 {
		a.NRGBA = area.Match
		a.Text = fmt.Sprintf("Time: %s", k)
		return true
	}

	a.NRGBA = area.Miss
	a.Text = "Time: Not Found"
	return false
}
