package detect

import (
	"fmt"
	"image"
	"regexp"
	"strings"
	"time"

	"gocv.io/x/gocv"

	"github.com/pidgy/unitehud/avi/img/splash"
	"github.com/pidgy/unitehud/avi/video"
	"github.com/pidgy/unitehud/avi/video/device"
	"github.com/pidgy/unitehud/avi/video/monitor"
	"github.com/pidgy/unitehud/avi/video/window"
	"github.com/pidgy/unitehud/core/config"
	"github.com/pidgy/unitehud/core/match"
	"github.com/pidgy/unitehud/core/notify"
	"github.com/pidgy/unitehud/core/server"
	"github.com/pidgy/unitehud/core/state"
	"github.com/pidgy/unitehud/core/stats/history"
	"github.com/pidgy/unitehud/core/team"
	"github.com/pidgy/unitehud/system/desktop"
	"github.com/pidgy/unitehud/system/desktop/clicked"
	"github.com/pidgy/unitehud/system/lang"
	"github.com/pidgy/unitehud/system/save"
)

var (
	Pause, Resume = func() { idling = true }, func() { idling = false }
	Images        = func(b bool) { images = b }

	idling = true
	images = false

	regexes = struct {
		defeated                   *regexp.Regexp
		defeatedWithUnscoredPoints *regexp.Regexp
		holdingPoints              *regexp.Regexp
		scoreOptionPresent         *regexp.Regexp
	}{
		defeated:                   regexp.MustCompile(`^\[Detect\] \[[0-9][0-9]:[0-9][0-9]\] \[Self\] Defeated$`),
		defeatedWithUnscoredPoints: regexp.MustCompile(`^\[Detect\] \[[0-9][0-9]:[0-9][0-9]\] \[Self\] Defeated with unscored points \([0-9]?[0-9]\)$`),
		holdingPoints:              regexp.MustCompile(`^\[Detect\] \[[0-9][0-9]:[0-9][0-9]\] \[Self\] Holding [0-9]?[0-9] point?[a-z]$`),
		scoreOptionPresent:         regexp.MustCompile(`^\[Detect\] \[[0-9][0-9]:[0-9][0-9]\] \[Self\] Score option present \([0-9]?[0-9]\)$`),
	}
)

func Clock() {
	for every(team.Time.Delay) {
		if config.Current.Advanced.Matching.Disabled.Time {
			continue
		}

		matrix, img, err := capture(config.Current.XY.Time)
		if err != nil {
			notify.Error("[Detect] Failed to capture clock area (%v)", err)
			matrix.Close()
			continue
		}

		min, sec, kitchen := match.Time(matrix, img)
		if min+sec == 0 {
			every(time.Second * 5) // Let's back off and save cpu cycles.
			matrix.Close()
			continue
		}

		server.SetTime(min, sec)

		if images {
			notify.Time, err = match.AsTimeImage(matrix, kitchen)
			if err != nil {
				notify.Error("[Detect] Failed to identify time (%v)", err)
				matrix.Close()
				continue
			}
		}

		matrix.Close()
	}
}

func Defeated() {
	area := image.Rectangle{}
	modified := config.Current.TemplatesKilled(team.Game.Name)
	unmodified := config.Current.TemplatesKilled(team.Game.Name)

	// Frequent, used to invalidate Self score detection by justifying the held energy drop.
	for every(time.Second) {
		if config.Current.Advanced.Matching.Disabled.Defeated {
			modified = config.Current.TemplatesKilled(team.Game.Name)
			unmodified = config.Current.TemplatesKilled(team.Game.Name)
			continue
		}

		if area.Empty() {
			b := monitor.MainResolution
			area = image.Rect(b.Max.X/3, b.Max.Y/2, b.Max.X-b.Max.X/3, b.Max.Y-b.Max.Y/3)
		}

		matrix, img, err := capture(area)
		if err != nil {
			notify.Error("[Detect] Failed to capture area (%v)", err)
			matrix.Close()
			continue
		}

		m, r := match.Matches(matrix, img, modified)
		switch r {
		case match.Found:
			e := state.EventType(m.Template.Value)

			state.Add(e, server.Clock(), m.Numeric)

			switch e {
			case state.Killed:
				modified = modified[1:] // Remove killed templates for processing.
				team.Self.Killed = time.Now()
				team.Self.KilledWithPoints = false
			case state.KilledWithPoints:
				modified = modified[1:] // Remove killed templates for processing.
				team.Self.Killed = time.Now()
				team.Self.KilledWithPoints = true
			case state.KilledWithoutPoints:
				modified = modified[1:] // Remove killed templates for processing.
				team.Self.Killed = time.Now()
				team.Self.KilledWithPoints = false
			}

			str := "Defeated"
			regex := regexes.defeated
			if team.Self.KilledWithPoints {
				str = fmt.Sprintf("%s with unscored points (%d)", str, server.Holding())
				regex = regexes.defeatedWithUnscoredPoints
			}
			notify.FeedReplace(team.Self.NRGBA, regex, "[Detect] [%s] [Self] %s", server.Clock(), str)

			if state.Occured(time.Minute, state.Killed, state.KilledWithPoints, state.KilledWithoutPoints) {
				server.SetDefeated()
			}
		default:
			modified = unmodified
		}

		matrix.Close()
	}
}

func Energy() {
	assured := make(map[int]int)

	confirmScore := -1

	for every(team.Energy.Delay) {
		if config.Current.Advanced.Matching.Disabled.Energy {
			assured = make(map[int]int)
			confirmScore = -1
			continue
		}

		matrix, img, err := capture(config.Current.XY.Energy)
		if err != nil {
			notify.Error("[Detect] Failed to capture energy area (%v)", err)
			matrix.Close()
			continue
		}

		result, _, points := match.Energy(matrix, img)
		if result != match.Found {
			matrix.Close()
			continue
		}

		// TODO: Is it better to check if we have 0 points?
		if confirmScore != -1 {
			go confirmEnergyWasScored(confirmScore, points, time.Now())
			confirmScore = -1
		}

		assured[points]++

		threshold := 1
		if points != team.Energy.Holding {
			threshold = 2
		}

		if assured[points] == threshold {
			assured = make(map[int]int)
		}

		last := state.HoldingEnergy.Occured(time.Hour)
		if last == nil || last.Value != points {
			notify.FeedReplace(team.Self.NRGBA, regexes.holdingPoints, "[Detect] [%s] [Self] Holding %d %s", server.Clock(), points, plural("point", points))

			state.Add(state.HoldingEnergy, server.Clock(), points)

			server.SetEnergy(points)

			if images {
				notify.Energy, err = match.AsAeosImage(matrix, points)
				if err != nil {
					notify.Warn("[Detect] [Self] Failed to identify (%v)", err)
				}
			}

			// Can we assume change from n, where n > 0, to 0 means a goal without being defeated?
			if points == 0 || points < team.Energy.Holding {
				confirmScore = team.Energy.Holding
			}

			team.Energy.Holding = points
		}

		matrix.Close()
	}
}

func Objectives() {
	top, bottom, central := time.Time{}, time.Time{}, time.Time{}

	for every(time.Second) {
		if config.Current.Advanced.Matching.Disabled.Objectives {
			top, bottom, central = time.Time{}, time.Time{}, time.Time{}
			continue
		}

		matrix, img, err := capture(config.Current.XY.Objectives)
		if err != nil {
			notify.Error("[Detect] Failed to capture objective area (%v)", err)
			matrix.Close()
			continue
		}

		m, r := match.Matches(matrix, img, config.Current.TemplatesSecure(team.Game.Name))
		if r != match.Found {
			matrix.Close()
			continue
		}

		event := state.EventType(m.Numeric)
		team := event.Team()

		switch event {
		case state.RegielekiSecureOrange, state.RegielekiSecurePurple:
			if time.Since(top) < time.Minute {
				matrix.Close()
				continue
			}
			server.SetRegieleki(team)
			top = time.Now()
		case state.RayquazaSecureOrange, state.RayquazaSecurePurple:
			if time.Since(central) < time.Minute {
				matrix.Close()
				continue
			}
			server.SetRayquaza(team)
			central = time.Now()
		case state.RegiceSecureOrange, state.RegiceSecurePurple:
			if time.Since(bottom) < time.Minute {
				matrix.Close()
				continue
			}
			server.SetRegice(team)
			bottom = time.Now()
		case state.RegirockSecureOrange, state.RegirockSecurePurple:
			if time.Since(bottom) < time.Minute {
				matrix.Close()
				continue
			}
			server.SetRegirock(team)
			bottom = time.Now()
		case state.RegisteelSecureOrange, state.RegisteelSecurePurple:
			if time.Since(bottom) < time.Minute {
				matrix.Close()
				continue
			}
			server.SetRegisteel(team)
			bottom = time.Now()
		}

		state.Add(event, server.Clock(), 0)
		notify.Feed(team.NRGBA, "[Detect] [%s] %s", server.Clock(), event)
	}
}

func Preview() {
	notify.Preview = splash.Projector()

	tick := time.NewTicker(time.Second * 5)
	poll := time.NewTicker(time.Second * 1)

	window := ""
	device := config.NoVideoCaptureDevice

	preview := func() {
		img, err := video.Capture()
		if err != nil {
			notify.Error("[Detect] Failed to capture preview (%v)", err)
			return
		}
		notify.Preview = img

		if config.Current.Video.Capture.Window.Name != window && config.Current.Video.Capture.Device.Index != device {
			notify.System("[Detect] %dx%d input resolution calculated", img.Bounds().Max.X, img.Bounds().Max.Y)
		}

		window = config.Current.Video.Capture.Window.Name
		device = config.Current.Video.Capture.Device.Index
	}
	preview()

	for every(time.Second) {
		if !images || config.Current.Advanced.Matching.Disabled.Previews {
			continue
		}

		rgba, ok := notify.Preview.(*image.RGBA)
		if ok && rgba == nil {
			continue
		}

		if notify.Preview.Bounds().Max.X != 0 {
			select {
			case <-tick.C:
				preview()
			case <-poll.C:
			}
		}
	}
}

func Scores(by string) {
	for every(team.Delay(by)) {
		if config.Current.Advanced.Matching.Disabled.Scoring {
			continue
		}

		switch t := team.By(by); t {
		case team.Self:
			if team.Energy.Holding == 0 {
				continue
			}

			matrix, img, err := capture(config.Current.ScoringOption())
			if err != nil {
				notify.Error("[Detect] [%s] [Self] Failed to capture energy area (%v)", server.Clock(), err)
				matrix.Close()
				continue
			}

			_, r := match.SelfScoreIndicator(matrix, img)
			if r == match.Found {
				state.Add(state.SelfScoreIndicator, server.Clock(), team.Energy.Holding)

				notify.FeedReplace(t.NRGBA, regexes.scoreOptionPresent, "[Detect] [%s] [Self] Score option present (%d)", server.Clock(), team.Energy.Holding)

				// TODO: Should we sleep and save some resources?
				// time.Sleep(time.Second)
			}

			matrix.Close()
		case team.First:
			if t.Counted {
				continue
			}

			fallthrough
		case team.Purple, team.Orange:
			if by == team.First.Name && team.First.Counted {
				continue
			}

			matrix, img, err := capture(config.Current.XY.Scores)
			if err != nil {
				notify.Error("[Detect] Failed to capture score area (%v)", err)
				matrix.Close()
				continue
			}

			m, r := match.Matches(matrix, img, config.Current.TemplatesScored(t.Name))
			if r == match.NotFound {
				matrix.Close()
				continue
			}

			switch r {
			case match.Override:
				state.Add(state.ScoreOverride, server.Clock(), m.Numeric)
				server.SetScore(t, -t.Duplicate.Replaces)
				notify.Feed(t.NRGBA, "[Detect] [%s] [%s] -%d (override)", server.Clock(), lang.Title(t.Name), t.Duplicate.Replaces)

				fallthrough
			case match.Found:
				server.SetScore(t, m.Numeric)

				title := fmt.Sprintf("[%s]", lang.Title(t.Name))
				if t.Name == team.First.Name {
					title = fmt.Sprintf("[%s] [%s]", lang.Title(t.Alias), lang.Title(t.Name))
				}

				notify.Feed(t.NRGBA, "[Detect] [%s] %s +%d", server.Clock(), title, m.Numeric)

				state.Add(state.ScoredBy(t.Name), server.Clock(), m.Numeric)
				team.First.Counted = true

				if images {
					score, err := m.AsImage(matrix, m.Numeric)
					if err != nil {
						notify.Error("[Detect] [%s] [%s] Failed to identify score (%v)", server.Clock(), t, err)
						break
					}

					switch t {
					case team.First:
						if t.Alias == team.Purple.Name {
							notify.PurpleScore = score
						} else {
							notify.OrangeScore = score
						}
					case team.Purple:
						notify.PurpleScore = score
					case team.Orange:
						notify.OrangeScore = score
					}
				}
			case match.Missed:
				state.Add(state.ScoreMissedBy(t.Name), server.Clock(), m.Numeric)

				notify.Warn("[Detect] [%s] [%s] [Missed] +%d", server.Clock(), t, m.Numeric)
			case match.Invalid:
				notify.Error("[Detect] [%s] [%s] [Invalid] +%d", server.Clock(), t, m.Numeric)
			case match.Duplicate:
				notify.Warn("[Detect] [%s] [%s] [Duplicate] +%d", server.Clock(), t, m.Numeric)
			}

			if config.Current.Record {
				err = save.Image(img, matrix, t.Crop(m.Point), m.Numeric, t.Name, strings.ToLower(r.String()), server.Clock())
				if err != nil {
					notify.Warn("[Detect] Failed to save image (%v)", err)
				}
			}

			matrix.Close()
		}
	}
}

func States() {
	area := image.Rectangle{}

	starting := config.Current.TemplatesStarting()
	ending := append(config.Current.TemplatesEnding(), config.Current.TemplatesSurrender()...)

	for every(time.Second * 2) {
		curr := starting
		if server.Seconds() != 0 {
			curr = ending
		}

		if area.Empty() {
			area = video.StateArea()
		}

		matrix, img, err := capture(area)
		if err != nil {
			notify.Error("[Detect] Failed to capture state area (%v)", err)
			matrix.Close()
			continue
		}

		m, r := match.Matches(matrix, img, curr)
		if r != match.Found {
			matrix.Close()
			continue
		}
		state.Add(state.EventType(m.Template.Value), server.Clock(), -1)

		switch e := state.EventType(m.Numeric); e {
		case state.MatchStarting:
			if server.Clock() == "10:00" {
				matrix.Close()
				continue
			}

			server.Clear()
			server.SetMatchStarted()

			team.Clear()
			state.Clear()

			d := config.Current.Video.Capture.Window.Name
			if device.IsActive() {
				d = device.ActiveName()
			}

			notify.Feed(team.Game.NRGBA, "[Detect] [%s] Match starting", team.Game)

			if !config.Current.Advanced.Notifications.Disabled.MatchStarting {
				desktop.Notification("Match Starting").
					Says("Capturing from %s", d).
					When(clicked.OpenUniteHUD).
					Send()
			}

			// Also tells javascript to turn on.
			server.SetTime(10, 0)
		case state.SurrenderOrange, state.SurrenderPurple:
			t := team.Purple
			if e == state.SurrenderOrange {
				t = team.Orange
			}
			notify.Feed(t.NRGBA, "[Detect] [%s] Surrendered", t)

			server.SetScoreSurrendered(t)

			fallthrough
		case state.MatchEnding:
			o, p, self := server.Scores()
			if o+p+self != 0 {
				notify.Feed(team.Game.NRGBA, "[Detect] [%s] Match ended", team.Game)

				// Purple score and objective results.

				regielekis, regices, regirocks, registeels, rayquazas := server.Objectives(team.Purple)
				notify.Feed(
					team.Purple.NRGBA,
					"[Detect] [%s] %s [+%d %s] [+%d %s] [+%d %s] [+%d %s] [+%d Rayquazas]",
					team.Purple,
					server.ScoreString(team.Purple),
					regielekis, plural("Regieleki", regielekis),
					regices, plural("Regice", regices),
					regirocks, plural("Regirock", regirocks),
					registeels, plural("Registeel", registeels),
					rayquazas,
				)

				// Orange score and objective results.

				regielekis, regices, regirocks, registeels, rayquazas = server.Objectives(team.Orange)
				notify.Feed(
					team.Orange.NRGBA,
					"[Detect] [%s] %s [+%d %s] [+%d %s] [+%d %s] [+%d %s] [+%d Rayquazas]",
					team.Orange,
					server.ScoreString(team.Orange),
					regielekis, plural("Regieleki", regielekis),
					regices, plural("Regice", regices),
					regirocks, plural("Regirock", regirocks),
					registeels, plural("Registeel", registeels),
					rayquazas,
				)

				// Self score and objective results.

				notify.Feed(team.Self.NRGBA, "[Detect] [%s] %d", team.Self, self)

				if !config.Current.Advanced.Notifications.Disabled.MatchStopped {
					pwin, owin := "", ""
					if p > o {
						pwin = "(Won)"
					} else if o > p {
						owin = "(Won)"
					}

					desktop.Notification("Match Ended").
						Says("Purple: %d %s\nOrange: %d %s\nYou scored %d points", p, pwin, o, owin, self).
						When(clicked.OpenUniteHUD).
						Send()
				}

				history.Add(p, o, self)
			}

			time.Sleep(time.Second * 10)

			server.Clear()
			team.Clear()
		}

		matrix.Close()
	}
}

func Window() {
	for ; ; time.Sleep(time.Second * 2) {
		if config.Current.Video.Capture.Window.Lost == "" {
			continue
		}

		err := window.Reattach()
		if err != nil {
			notify.Error("[Detect] Failed to reattach window (%v)", err)
			continue
		}
	}
}

func capture(area image.Rectangle) (gocv.Mat, *image.RGBA, error) {
	img, err := video.CaptureRect(area)
	if err != nil {
		return gocv.Mat{}, nil, err
	}

	matrix, err := gocv.ImageToMatRGB(img)
	if err != nil {
		return gocv.Mat{}, nil, err
	}

	return matrix, img, nil
}

// confirmEnergyWasScored is another step to confirm a self-score event occured. This function
// handles multiple edge cases that can result in invalid detections, such as:
//   - Interrupted score attempts.
//   - Defeated while scoring.
//   - ...
//
// If a call is made to this function it is because UniteHUD has detected were holding 0 points
// after a confirmed score match.
func confirmEnergyWasScored(before, after int, at time.Time) {
	if before == after {
		return
	}

	notify.Feed(team.Self.NRGBA,
		"[Detect] [%s] [Self] +%d Confirming %s scored %s ago",
		server.Clock(),
		before,
		plural("point", before),
		time.Since(at),
	)

	// Confirm user was not defeated with points since the goal.
	if state.KilledWithPoints.Occured(time.Second*2) != nil {
		notify.Warn("[Detect] [%s] [Self] [Missed] +%d Defeated before scoring", server.Clock(), before)
		return
	}

	p := state.SelfScoreIndicator.Occured(time.Second * 5)
	if p == nil {
		notify.Warn("[Detect] [%s] [Self] [Missed] +%d Failed to find self-score option", server.Clock(), before)
		return
	}

	if p.Verified {
		return
	}
	p.Verified = true

	if server.IsFinalStretch() {
		before *= 2
	}

	if !team.First.Counted {
		team.First.Counted = true
	}

	go server.SetScore(team.Self, before)

	state.Add(state.PostScore, server.Clock(), before)

	notify.Feed(team.Self.NRGBA, "[Detect] [%s] [%s] [%s] +%d", server.Clock(), team.Purple, team.Self, before)
}

func plural(s string, size int) string {
	if size == 1 {
		return s
	}
	return s + "s"
}

func every(d time.Duration) bool {
	for {
		time.Sleep(d)
		if config.Current.Advanced.DecreasedCaptureLevel > 0 {
			time.Sleep(time.Second * config.Current.Advanced.DecreasedCaptureLevel)
		}

		if !idling {
			return true
		}
	}
}
