package detect

import (
	"fmt"
	"image"
	"strings"
	"time"

	"gocv.io/x/gocv"

	"github.com/pidgy/unitehud/config"
	"github.com/pidgy/unitehud/desktop"
	"github.com/pidgy/unitehud/desktop/clicked"
	"github.com/pidgy/unitehud/history"
	"github.com/pidgy/unitehud/img/splash"
	"github.com/pidgy/unitehud/match"
	"github.com/pidgy/unitehud/notify"
	"github.com/pidgy/unitehud/save"
	"github.com/pidgy/unitehud/server"
	"github.com/pidgy/unitehud/state"
	"github.com/pidgy/unitehud/team"
	"github.com/pidgy/unitehud/video"
	"github.com/pidgy/unitehud/video/device"
	"github.com/pidgy/unitehud/video/monitor"
	"github.com/pidgy/unitehud/video/window"
)

var (
	idle = true

	Pause  = func() { idle = true }
	Resume = func() { idle = false }
)

func Clock() {
	for ; ; sleep(team.Delay(team.Time.Name)) {
		if idle || config.Current.Advanced.Matching.Disabled.Time {
			continue
		}

		matrix, img, err := capture(config.Current.Time)
		if err != nil {
			notify.Error("Detect: Failed to capture clock area (%v)", err)
			continue
		}

		rs, kitchen := match.Time(matrix, img)
		if rs == 0 {
			// Let's back off and not waste processing power.
			sleep(time.Second * 5)
			continue
		}

		notify.Time, err = match.AsTimeImage(matrix, kitchen)
		if err != nil {
			notify.Error("Detect: Failed to identify time (%v)", err)
			continue
		}
	}
}

func Defeated() {
	area := image.Rectangle{}
	modified := config.Current.TemplatesKilled(team.Game.Name)
	unmodified := config.Current.TemplatesKilled(team.Game.Name)

	for ; ; sleep(time.Second) {
		if idle || config.Current.Advanced.Matching.Disabled.Defeated {
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
			notify.Error("Detect: Failed to capture defeated area (%v)", err)
			continue
		}

		m, r, p := match.Matches(matrix, img, modified)
		switch r {
		case match.Found:
			e := state.EventType(m.Template.Value)

			state.Add(e, server.Clock(), p)

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
			if team.Self.KilledWithPoints {
				str = fmt.Sprintf("%s with unscored points (%d)", str, server.Holding())
			}

			notify.Feed(team.Self.NRGBA, "Detect: [%s] [Self] %s", server.Clock(), str)

			if state.Occured(time.Minute, state.Killed, state.KilledWithPoints, state.KilledWithoutPoints) != nil {
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

	for ; ; sleep(team.Energy.Delay) {
		if idle || config.Current.Advanced.Matching.Disabled.Energy {
			assured = make(map[int]int)
			confirmScore = -1
			continue
		}

		matrix, img, err := capture(config.Current.Energy)
		if err != nil {
			notify.Error("Detect: Failed to capture energy area (%v)", err)
			continue
		}

		result, _, points := match.Energy(matrix, img)
		if result != match.Found {
			matrix.Close()
			continue
		}

		// TODO: Is it better to check if we have 0 points?
		if confirmScore != -1 {
			go energyScoredConfirm(confirmScore, points, time.Now())
			confirmScore = -1
		}

		assured[points]++

		threshold := 1
		if points != team.Energy.Holding {
			threshold = 2
		}
		/*
			if team.Energy.Holding == 0 {
				threshold = 3
			} else if team.Energy.Holding != 0 && points != team.Energy.Holding {
				threshold = 2
			}
		*/
		if assured[points] == threshold {
			assured = make(map[int]int)
		}

		last := state.HoldingEnergy.Occured(time.Hour)
		if last == nil || last.Value != points {
			notify.Feed(team.Self.NRGBA, "Detect: [%s] [Self] Holding %d point%s", server.Clock(), points, s(points))
			state.Add(state.HoldingEnergy, server.Clock(), points)

			server.SetEnergy(points)

			notify.Energy, err = match.AsAeosImage(matrix, points)
			if err != nil {
				notify.Warn("Detect: [Self] Failed to identify energy (%v)", err)
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
	top, bottom, middle := time.Time{}, time.Time{}, time.Time{}

	for ; ; sleep(time.Second) {
		if idle || config.Current.Advanced.Matching.Disabled.Objectives {
			top, bottom, middle = time.Time{}, time.Time{}, time.Time{}
			continue
		}

		matrix, img, err := capture(config.Current.Objectives)
		if err != nil {
			notify.Error("Detect: Failed to capture objective area (%v)", err)
			continue
		}

		_, r, e := match.Matches(matrix, img, config.Current.TemplatesSecure(team.Game.Name))
		if r != match.Found {
			matrix.Close()
			continue
		}

		early := false

		if time.Since(top) > time.Minute {
			switch e := state.EventType(e); e {
			case state.RegielekiSecureOrange:
				state.Add(e, server.Clock(), 0)
				notify.Feed(team.Orange.NRGBA, "Detect: [%s] [%s] Regieleki secured", server.Clock(), strings.Title(team.Orange.Name))
				server.SetRegieleki(team.Orange)
				top = time.Now()

				early = true
			case state.RegielekiSecurePurple:
				state.Add(e, server.Clock(), 0)
				notify.Feed(team.Purple.NRGBA, "Detect: [%s] [%s] Regieleki secured", server.Clock(), strings.Title(team.Purple.Name))
				server.SetRegieleki(team.Purple)
				top = time.Now()

				early = true
			}
		}

		if !early && time.Since(bottom) > time.Minute {
			switch e := state.EventType(e); e {
			case state.RegiceSecureOrange:
				state.Add(e, server.Clock(), 0)
				notify.Feed(team.Orange.NRGBA, "Detect: [%s] [%s] Regice secured", server.Clock(), strings.Title(team.Orange.Name))
				server.SetRegice(team.Orange)
				bottom = time.Now()

				early = true
			case state.RegiceSecurePurple:
				state.Add(e, server.Clock(), 0)
				notify.Feed(team.Purple.NRGBA, "Detect: [%s] [%s] Regice secured", server.Clock(), strings.Title(team.Purple.Name))
				server.SetRegice(team.Purple)
				bottom = time.Now()

				early = true
			case state.RegirockSecureOrange:
				state.Add(e, server.Clock(), 0)
				notify.Feed(team.Orange.NRGBA, "Detect: [%s] [%s] Regirock secured", server.Clock(), strings.Title(team.Orange.Name))
				server.SetRegirock(team.Orange)
				bottom = time.Now()

				early = true
			case state.RegirockSecurePurple:
				state.Add(e, server.Clock(), 0)
				notify.Feed(team.Purple.NRGBA, "Detect:[%s] [%s] Regirock secured", server.Clock(), strings.Title(team.Purple.Name))
				server.SetRegirock(team.Purple)
				bottom = time.Now()

				early = true
			case state.RegisteelSecureOrange:
				state.Add(e, server.Clock(), 0)
				notify.Feed(team.Orange.NRGBA, "Detect:[%s] [%s] Registeel secured", server.Clock(), strings.Title(team.Orange.Name))
				server.SetRegisteel(team.Orange)
				bottom = time.Now()

				early = true
			case state.RegisteelSecurePurple:
				state.Add(e, server.Clock(), 0)
				notify.Feed(team.Purple.NRGBA, "Detect:[%s] [%s] Registeel secured", server.Clock(), strings.Title(team.Purple.Name))
				server.SetRegisteel(team.Purple)
				bottom = time.Now()

				early = true
			}
		}

		if !early && time.Since(middle) > time.Minute {
			switch e := state.EventType(e); e {
			case state.RayquazaSecureOrange:
				state.Add(e, server.Clock(), 0)
				notify.Feed(team.Orange.NRGBA, "Detect: [%s] [%s] Rayquaza secured", server.Clock(), strings.Title(team.Orange.Name))
				server.SetRayquaza(team.Orange)
				middle = time.Now()

				early = true
			case state.RayquazaSecurePurple:
				state.Add(e, server.Clock(), 0)
				notify.Feed(team.Purple.NRGBA, "Detect: [%s] [%s] Rayquaza secured", server.Clock(), strings.Title(team.Purple.Name))
				server.SetRayquaza(team.Purple)
				middle = time.Now()

				early = true
			}
		}

		matrix.Close()
	}
}

func PressButtonToScore() {
	for ; ; sleep(time.Millisecond * 500) {
		if idle {
			continue
		}

		if team.Self.Holding == 0 {
			continue
		}

		matrix, img, err := capture(config.Current.ScoringOption())
		if err != nil {
			notify.Error("Detect: [%s] [Self] Failed to capture energy area (%v)", server.Clock(), err)
			continue
		}

		_, r := match.SelfScoreOption(matrix, img)
		if r != match.Found {
			matrix.Close()
			continue
		}

		state.Add(state.PressButtonToScore, server.Clock(), team.Energy.Holding)

		notify.Feed(team.Self.NRGBA, "Detect: [%s] [Self] Score option present (%d)", server.Clock(), team.Energy.Holding)

		matrix.Close()

		// Save some resources,
		time.Sleep(time.Second * 2)
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
			notify.Error("Detect: [%s] [Preview] Failed to capture preview (%v)", server.Clock(), err)
			return
		}
		notify.Preview = img

		if config.Current.VideoCaptureWindow != window && config.Current.VideoCaptureDevice != device {
			notify.System("Detect: [Preview] Input resolution calculated (%dpx, %dpx)", img.Bounds().Max.X, img.Bounds().Max.Y)
		}

		window = config.Current.VideoCaptureWindow
		device = config.Current.VideoCaptureDevice
	}

	preview()

	for {
		if config.Current.Advanced.Matching.Disabled.Previews {
			time.Sleep(time.Second)
			continue
		}

		if notify.Preview.Bounds().Max.X != 0 {
			select {
			case <-tick.C:
				preview()
			case <-poll.C:
				if config.Current.VideoCaptureWindow == window && config.Current.VideoCaptureDevice == device {
					continue
				}
			}
		}
	}
}

func Scores(name string) {
	for ; ; sleep(team.Delay(name)) {
		if idle || config.Current.Advanced.Matching.Disabled.Scoring {
			continue
		}

		if name == team.First.Name && team.First.Counted {
			continue
		}

		matrix, img, err := capture(config.Current.Scores)
		if err != nil {
			notify.Error("Detect: Failed to capture score area (%v)", err)
			continue
		}

		m, r, p := match.Matches(matrix, img, config.Current.TemplatesScored(name))
		if r == match.NotFound {
			matrix.Close()
			continue
		}

		switch r {
		case match.Override:
			state.Add(state.ScoreOverride, server.Clock(), p)

			server.SetScore(m.Team, -m.Team.Duplicate.Replaces)

			notify.Feed(m.Team.NRGBA, "Detect: [%s] [%s] -%d (override)", server.Clock(), strings.Title(m.Team.Name), m.Team.Duplicate.Replaces)

			fallthrough
		case match.Found:
			server.SetScore(m.Team, p)

			title := fmt.Sprintf("[%s]", strings.Title(m.Team.Name))
			if m.Team.Name == team.First.Name {
				title = fmt.Sprintf("[%s] [%s]", strings.Title(m.Team.Alias), strings.Title(m.Team.Name))
			}

			notify.Feed(m.Team.NRGBA, "Detect:[%s] %s +%d", server.Clock(), title, p)

			state.Add(state.ScoredBy(m.Team.Name), server.Clock(), p)

			score, err := m.AsImage(matrix, p)
			if err != nil {
				notify.Error("Detect: [%s] [%s] Failed to identify score (%v)", server.Clock(), strings.Title(m.Team.Name), err)
				break
			}

			switch m.Team.Name {
			case team.First.Name:
				team.First.Counted = true

				if team.First.Alias == team.Purple.Name {
					notify.PurpleScore = score
				} else {
					notify.OrangeScore = score
				}
			case team.Purple.Name:
				notify.PurpleScore = score
			case team.Orange.Name:
				notify.OrangeScore = score
			}
		case match.Missed:
			state.Add(state.ScoreMissedBy(m.Team.Name), server.Clock(), p)

			notify.Error("Detect: [%s] [%s] +%d (missed)", server.Clock(), strings.Title(m.Team.Name), p)
		case match.Invalid:
			notify.Error("Detect: [%s] [%s] +%d (invalid)", server.Clock(), strings.Title(m.Team.Name), p)
		case match.Duplicate:
			notify.Warn("[%s] [%s] +%d (duplicate)", server.Clock(), strings.Title(m.Team.Name), p)
		}

		if config.Current.Record {
			save.Image(img, matrix, m.Team, m.Point, p, r)
		}

		matrix.Close()
	}
}

func States() {
	area := image.Rectangle{}

	for ; ; sleep(time.Second * 2) {
		if idle {
			continue
		}

		if area.Empty() {
			area = video.StateArea()
		}

		matrix, img, err := capture(area)
		if err != nil {
			notify.Error("Detect: Failed to capture state area (%v)", err)
			matrix.Close()
			continue
		}

		m, r, e := match.Matches(matrix, img, config.Current.TemplatesGame(team.Game.Name))
		if r != match.Found {
			matrix.Close()
			continue
		}

		state.Add(state.EventType(m.Template.Value), server.Clock(), -1)

		switch e := state.EventType(e); e {
		case state.MatchStarting:
			if server.Clock() == "10:00" {
				matrix.Close()
				continue
			}

			server.Clear()
			server.SetMatchStarted()

			team.Clear()
			state.Clear()

			d := config.Current.VideoCaptureWindow
			if device.IsActive() {
				d = device.ActiveName()
			}

			desktop.Notification("Match is starting").
				Says("Capturing from %s", d).
				When(clicked.OpenUniteHUD).
				Send()

			// Also tells javascript to turn on.
			server.SetTime(10, 0)
		case state.MatchEnding:
			switch config.Current.Profile {
			case config.ProfileBroadcaster:
				if !server.Match() {
					break
				}

				// Purple score and objective results.
				regielekis, regices, regirocks, registeels, rayquazas := server.Objectives(team.Purple)
				purpleResult := fmt.Sprintf(
					"[%s] [+%d KO%s] [+%d Regieleki%s] [+%d Regice%s] [+%d Regirock%s] [+%d Registeel%s] [+%d Rayquazas]",
					strings.Title(team.Purple.Name),
					server.KOs(team.Purple), s(server.KOs(team.Purple)),
					regielekis, s(regielekis),
					regices, s(regices),
					regirocks, s(regirocks),
					registeels, s(registeels),
					rayquazas,
				)
				notify.Feed(team.Purple.NRGBA, purpleResult)

				// Orange score and objective results.
				regielekis, regices, regirocks, registeels, rayquazas = server.Objectives(team.Orange)
				orangeResult := fmt.Sprintf(
					"[%s] [+%d KO%s] [+%d Regieleki%s] [+%d Regice%s] [+%d Regirock%s] [+%d Registeel%s] [+%d Rayquazas]",
					strings.Title(team.Orange.Name),
					server.KOs(team.Orange), s(server.KOs(team.Orange)),
					regielekis, s(regielekis),
					regices, s(regices),
					regirocks, s(regirocks),
					registeels, s(registeels),
					rayquazas,
				)
				notify.Feed(team.Orange.NRGBA, orangeResult)

			case config.ProfilePlayer:
				o, p, self := server.Scores()
				if o+p+self > 0 {
					notify.Feed(team.Game.NRGBA, "Detect:[%s] Match ended", strings.Title(team.Game.Name))

					// Purple score and objective results.
					regielekis, regices, regirocks, registeels, rayquazas := server.Objectives(team.Purple)
					purpleResult := fmt.Sprintf(
						"[%s] %d [+%d KO%s] [+%d Regieleki%s] [+%d Regice%s] [+%d Regirock%s] [+%d Registeel%s] [+%d Rayquazas]",
						strings.Title(team.Purple.Name),
						p,
						server.KOs(team.Purple), s(server.KOs(team.Purple)),
						regielekis, s(regielekis),
						regices, s(regices),
						regirocks, s(regirocks),
						registeels, s(registeels),
						rayquazas,
					)

					notify.Feed(team.Purple.NRGBA, purpleResult)

					// Orange score and objective results.
					regielekis, regices, regirocks, registeels, rayquazas = server.Objectives(team.Orange)
					orangeResult := fmt.Sprintf(
						"[%s] %d [+%d KO%s] [+%d Regieleki%s] [+%d Regice%s] [+%d Regirock%s] [+%d Registeel%s] [+%d Rayquazas]",
						strings.Title(team.Orange.Name),
						o,
						server.KOs(team.Orange), s(server.KOs(team.Orange)),
						regielekis, s(regielekis),
						regices, s(regices),
						regirocks, s(regirocks),
						registeels, s(registeels),
						rayquazas,
					)

					notify.Feed(team.Orange.NRGBA, orangeResult)

					// Self score and objective results.
					notify.Feed(team.Self.NRGBA, "Detect:[%s] %d", strings.Title(team.Self.Name), self)

					pwin := ""
					owin := ""
					if p > o {
						pwin = "(Won)"
					} else if o > p {
						owin = "(Won)"
					}

					desktop.Notification("Match has ended").
						Says("Purple: %d %s\nOrange: %d %s\nYou scored %d points", p, pwin, o, owin, self).
						When(clicked.OpenUniteHUD).
						Send()

					history.Add(p, o, self)
				}
			}

			// If time since match started is greater thaaaan 2 mins lets wait for 10 seconds...
			cooldown := time.Second * 0
			start := state.MatchStarting.Occured(time.Since(state.Start().Time))
			if start != nil {
				end := state.MatchEnding.Occured(time.Since(state.Start().Time))
				if end != nil && start.After(end.Time) {
					cooldown = time.Second * 10
				}
			}

			time.Sleep(cooldown)

			server.Clear()
			team.Clear()
		}

		matrix.Close()
	}
}

func Window() {
	for ; ; time.Sleep(time.Second * 2) {
		if config.Current.LostWindow == "" {
			continue
		}

		err := window.Reattach()
		if err != nil {
			notify.Error("Detect: Failed to reattach window (%v)", err)
		}
	}
}

func capture(area image.Rectangle) (gocv.Mat, *image.RGBA, error) {
	img, err := video.CaptureRect(area)
	if err != nil {
		return gocv.Mat{}, nil, err
	}

	m, err := gocv.ImageToMatRGB(img)
	if err != nil {
		return gocv.Mat{}, nil, err
	}

	return m, img, nil
}

// energyScoredConfirm is another step to confirm a self-score event occured. This function
// handles multiple edge cases that can result in invalid detections, such as:
//   - Interrupted score attempts.
//   - Defeated while scoring.
//   - ...
//
// If a call is made to this function it is because UniteHUD has detected were holding 0 points
// after a confirmed score match.
func energyScoredConfirm(before, after int, at time.Time) {
	if before == after {
		return
	}

	notify.Feed(team.Self.NRGBA,
		"Detect: [%s] [Self] Confirming %d point%s scored %s ago",
		server.Clock(),
		before,
		s(before),
		time.Since(at),
	)

	// Confirm user was not defeated with points since the goal.
	if state.KilledWithPoints.Occured(time.Second*2) != nil {
		notify.Warn("[%s] Defeated before scoring", server.Clock())
		return
	}

	delay := time.Second * time.Duration(config.Current.ConfirmScoreDelay)
	if delay == 0 {
		delay = time.Second * 5
	}

	p := state.PressButtonToScore.Occured(delay)
	if p != nil && !p.Verified {
		p.Verified = true
	} else {
		notify.Warn(
			"Detect: [%s] [Self] Failed to score because the score option was not present within %s (-%d)",
			server.Clock(),
			delay,
			before,
		)

		return
	}

	if server.IsFinalStretch() {
		before *= 2
	}

	go server.SetScore(team.Self, before)

	state.Add(state.PostScore, server.Clock(), before)

	notify.Feed(team.Self.NRGBA,
		"[%s] [%s] [%s] +%d",
		server.Clock(),
		strings.Title(team.Purple.Name),
		strings.Title(team.Self.Name),
		before,
	)
}

func s(size int) string {
	if size == 1 {
		return ""
	}
	return "s"
}

func sleep(d time.Duration) {
	delta := time.Duration(float64(d) * (float64(config.Current.Advanced.IncreasedCaptureRate) / 100))
	if delta < 0 {
		time.Sleep(d - delta)
	} else {
		time.Sleep(d + delta)
	}
}
