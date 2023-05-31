package detect

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"strings"
	"time"

	"gocv.io/x/gocv"

	"github.com/pidgy/unitehud/config"
	"github.com/pidgy/unitehud/debug"
	"github.com/pidgy/unitehud/duplicate"
	"github.com/pidgy/unitehud/gui"
	"github.com/pidgy/unitehud/history"
	"github.com/pidgy/unitehud/match"
	"github.com/pidgy/unitehud/notify"
	"github.com/pidgy/unitehud/server"
	"github.com/pidgy/unitehud/state"
	"github.com/pidgy/unitehud/team"
	"github.com/pidgy/unitehud/video"
	"github.com/pidgy/unitehud/video/window"
)

var (
	Idle = true
)

func Clock() {
	for {
		time.Sleep(team.Delay(team.Time.Name))

		if Idle || config.Current.DisableTime {
			continue
		}

		matrix, img, err := capture(config.Current.Time)
		if err != nil {
			notify.Error("Failed to capture clock area (%v)", err)
			continue
		}

		rs, kitchen := match.Time(matrix, img)
		if rs == 0 {
			// Let's back off and not waste processing power.
			time.Sleep(time.Second * 5)
			continue
		}

		notify.Time, err = match.AsTimeImage(matrix, kitchen)
		if err != nil {
			notify.Error("Failed to identify time (%v)", err)
			continue
		}
	}
}

func Close() {
	video.Close()

	os.Exit(0)
}

func Defeated() {
	area := image.Rectangle{}
	modified := config.Current.Templates["killed"][team.Game.Name]
	unmodified := config.Current.Templates["killed"][team.Game.Name]

	for {
		time.Sleep(time.Second)

		if Idle || gui.Window.Screen == nil || config.Current.DisableDefeated {
			modified = config.Current.Templates["killed"][team.Game.Name]
			unmodified = config.Current.Templates["killed"][team.Game.Name]
			continue
		}

		if area.Empty() {
			b := gui.Window.Screen.Bounds()
			area = image.Rect(b.Max.X/3, b.Max.Y/2, b.Max.X-b.Max.X/3, b.Max.Y-b.Max.Y/3)
		}

		matrix, img, err := capture(area)
		if err != nil {
			notify.Error("Failed to capture defeated area (%v)", err)
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

			notify.Feed(team.Self.NRGBA, "[%s] [Self] %s", server.Clock(), str)

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

	for {
		time.Sleep(team.Energy.Delay)

		if Idle || config.Current.DisableEnergy {
			assured = make(map[int]int)
			confirmScore = -1
			continue
		}

		matrix, img, err := capture(config.Current.Energy)
		if err != nil {
			notify.Error("Failed to capture energy area (%v)", err)
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
			notify.Feed(team.Self.NRGBA, "[%s] [Self] Holding %d point%s", server.Clock(), points, s(points))
			state.Add(state.HoldingEnergy, server.Clock(), points)

			server.SetEnergy(points)

			notify.Energy, err = match.AsAeosImage(matrix, points)
			if err != nil {
				notify.Warn("[Self] Failed to identify energy (%v)", err)
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

func KOs() {
	var last *duplicate.Duplicate

	for {
		time.Sleep(time.Millisecond * 1500)

		if Idle || gui.Window.Screen == nil || config.Current.DisableKOs {
			last = nil
			continue
		}

		matrix, img, err := capture(config.Current.KOs)
		if err != nil {
			notify.Error("Failed to capture objective area (%v)", err)
			continue
		}

		_, r, e := match.Matches(matrix, img, config.Current.Templates["ko"][team.Game.Name])
		if r != match.Found {
			matrix.Close()
			continue
		}

		dup := duplicate.New(-1, matrix, matrix.Region(image.Rect(10, 10, matrix.Cols()-10, matrix.Rows()-10)))
		if dup.Pixels(last) {
			if time.Since(last.Time) < time.Second*10 {
				matrix.Close()
				continue
			}
		}

		last.Close()
		last = dup

		switch e := state.EventType(e); e {
		case state.KOPurple, state.KOStreakPurple:
			notify.Unique(team.Purple.NRGBA, "[%s] [%s] %s", server.Clock(), team.Purple, e)
			server.SetKO(team.Purple)
		case state.KOOrange, state.KOStreakOrange:
			notify.Unique(team.Orange.NRGBA, "[%s] [%s] %s", server.Clock(), team.Orange, e)
			server.SetKO(team.Orange)
		}
	}
}

func Objectives() {
	top, bottom, middle := time.Time{}, time.Time{}, time.Time{}

	for {
		time.Sleep(time.Second)

		if Idle || gui.Window.Screen == nil || config.Current.DisableObjectives {
			top, bottom, middle = time.Time{}, time.Time{}, time.Time{}
			continue
		}

		matrix, img, err := capture(config.Current.Objectives)
		if err != nil {
			notify.Error("Failed to capture objective area (%v)", err)
			continue
		}

		_, r, e := match.Matches(matrix, img, config.Current.Templates["secure"][team.Game.Name])
		if r != match.Found {
			matrix.Close()
			continue
		}

		done := false

		if time.Since(top) > time.Minute {
			switch e := state.EventType(e); e {
			case state.RegielekiSecureOrange:
				state.Add(e, server.Clock(), 0)
				notify.Feed(team.Orange.NRGBA, "[%s] [%s] Regieleki secured", server.Clock(), strings.Title(team.Orange.Name))
				server.SetRegieleki(team.Orange)
				top = time.Now()

				done = true
			case state.RegielekiSecurePurple:
				state.Add(e, server.Clock(), 0)
				notify.Feed(team.Purple.NRGBA, "[%s] [%s] Regieleki secured", server.Clock(), strings.Title(team.Purple.Name))
				server.SetRegieleki(team.Purple)
				top = time.Now()

				done = true
			}
		}

		if !done && time.Since(bottom) > time.Minute {
			switch e := state.EventType(e); e {
			case state.RegiceSecureOrange:
				state.Add(e, server.Clock(), 0)
				notify.Feed(team.Orange.NRGBA, "[%s] [%s] Regice secured", server.Clock(), strings.Title(team.Orange.Name))
				server.SetRegice(team.Orange)
				bottom = time.Now()
			case state.RegiceSecurePurple:
				state.Add(e, server.Clock(), 0)
				notify.Feed(team.Purple.NRGBA, "[%s] [%s] Regice secured", server.Clock(), strings.Title(team.Purple.Name))
				server.SetRegice(team.Purple)
				bottom = time.Now()

			case state.RegirockSecureOrange:
				state.Add(e, server.Clock(), 0)
				notify.Feed(team.Orange.NRGBA, "[%s] [%s] Regirock secured", server.Clock(), strings.Title(team.Orange.Name))
				server.SetRegirock(team.Orange)
				bottom = time.Now()
			case state.RegirockSecurePurple:
				state.Add(e, server.Clock(), 0)
				notify.Feed(team.Purple.NRGBA, "[%s] [%s] Regirock secured", server.Clock(), strings.Title(team.Purple.Name))
				server.SetRegirock(team.Purple)
				bottom = time.Now()

			case state.RegisteelSecureOrange:
				state.Add(e, server.Clock(), 0)
				notify.Feed(team.Orange.NRGBA, "[%s] [%s] Registeel secured", server.Clock(), strings.Title(team.Orange.Name))
				server.SetRegisteel(team.Orange)
				bottom = time.Now()
			case state.RegisteelSecurePurple:
				state.Add(e, server.Clock(), 0)
				notify.Feed(team.Purple.NRGBA, "[%s] [%s] Registeel secured", server.Clock(), strings.Title(team.Purple.Name))
				server.SetRegisteel(team.Purple)
				bottom = time.Now()
			}
		}

		if !done && time.Since(middle) > time.Minute {
			switch e := state.EventType(e); e {
			case state.RayquazaSecureOrange:
				state.Add(e, server.Clock(), 0)
				notify.Feed(team.Orange.NRGBA, "[%s] [%s] Rayquaza secured", server.Clock(), strings.Title(team.Orange.Name))
				server.SetRayquaza(team.Orange)
				middle = time.Now()

			case state.RayquazaSecurePurple:
				state.Add(e, server.Clock(), 0)
				notify.Feed(team.Purple.NRGBA, "[%s] [%s] Rayquaza secured", server.Clock(), strings.Title(team.Purple.Name))
				server.SetRayquaza(team.Purple)
				middle = time.Now()
			}
		}

		matrix.Close()
	}
}

func PressButtonToScore() {
	for {
		time.Sleep(time.Millisecond * 500)

		if Idle {
			continue
		}

		matrix, img, err := capture(config.Current.ScoringOption())
		if err != nil {
			notify.Error("Failed to capture energy area (%v)", err)
			continue
		}

		_, r := match.SelfScoreOption(matrix, img)
		if r != match.Found {
			matrix.Close()
			continue
		}

		state.Add(state.PressButtonToScore, server.Clock(), team.Energy.Holding)

		notify.Feed(team.Self.NRGBA, "[%s] [Self] Score option present (%d)", server.Clock(), team.Energy.Holding)

		matrix.Close()

		// Save some resources,
		time.Sleep(time.Second * 2)
	}
}

func Preview() {
	f, err := os.Open(`assets\splash\projector.png`)
	if err != nil {
		notify.Error("Failed to open splash capture screen (%v)", err)
	}
	if f != nil {
		notify.Preview, err = png.Decode(f)
		if err != nil {
			notify.Error("Failed to decode splash capture screen (%v)", err)
		}
	}

	tick := time.NewTicker(time.Second * 5)
	poll := time.NewTicker(time.Second * 5)

	window := ""
	device := config.NoVideoCaptureDevice

	for {
		if config.Current.DisablePreviews {
			time.Sleep(time.Second)
			continue
		}

		if notify.Preview.Bounds().Max.X != 0 {
			select {
			case <-tick.C:
			case <-poll.C:
				if config.Current.Window == window && config.Current.VideoCaptureDevice == device {
					continue
				}
			}
		}

		return

		img, err := video.Capture()
		if err != nil {
			notify.Error("Failed to capture preview (%v)", err)
			continue
		}
		notify.Preview = img

		if config.Current.Window != window && config.Current.VideoCaptureDevice != device {
			notify.System("Input resolution calculated %s", img.Bounds().Max)
		}

		window = config.Current.Window
		device = config.Current.VideoCaptureDevice
	}
}

func Scores(name string) {
	for {
		time.Sleep(team.Delay(name))

		if Idle || config.Current.DisableScoring {
			continue
		}

		if name == team.First.Name && team.First.Counted {
			continue
		}

		matrix, img, err := capture(config.Current.Scores)
		if err != nil {
			notify.Error("Failed to capture score area (%v)", err)
			continue
		}

		m, r, p := match.Matches(matrix, img, config.Current.Templates["scored"][name])
		if r == match.NotFound {
			matrix.Close()
			continue
		}

		switch r {
		case match.Override:
			state.Add(state.ScoreOverride, server.Clock(), p)

			server.SetScore(m.Team, -m.Team.Duplicate.Replaces)

			notify.Feed(m.Team.NRGBA, "[%s] [%s] -%d (override)", server.Clock(), strings.Title(m.Team.Name), m.Team.Duplicate.Replaces)

			fallthrough
		case match.Found:
			server.SetScore(m.Team, p)

			title := fmt.Sprintf("[%s]", strings.Title(m.Team.Name))
			if m.Team.Name == team.First.Name {
				title = fmt.Sprintf("[%s] [%s]", strings.Title(m.Team.Alias), strings.Title(m.Team.Name))
			}

			notify.Feed(m.Team.NRGBA, "[%s] %s +%d", server.Clock(), title, p)

			state.Add(state.ScoredBy(m.Team.Name), server.Clock(), p)

			score, err := m.AsImage(matrix, p)
			if err != nil {
				notify.Error("[%s] [%s] Failed to identify score (%v)", server.Clock(), strings.Title(m.Team.Name), err)
				break
			}

			switch m.Team.Name {
			case team.First.Name:
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

			notify.Error("[%s] [%s] +%d (missed)", server.Clock(), strings.Title(m.Team.Name), p)
		case match.Invalid:
			notify.Error("[%s] [%s] +%d (invalid)", server.Clock(), strings.Title(m.Team.Name), p)
		case match.Duplicate:
			notify.Warn("[%s] [%s] +%d (duplicate)", server.Clock(), strings.Title(m.Team.Name), p)
		}

		if config.Current.Record {
			debug.Capture(img, matrix, m.Team, m.Point, p, r)
		}

		matrix.Close()
	}
}

func States() {
	area := image.Rectangle{}

	for {
		time.Sleep(time.Second * 2)

		if Idle || gui.Window.Screen == nil {
			continue
		}

		if area.Empty() {
			area = gui.StateArea()
		}

		matrix, img, err := capture(area)
		if err != nil {
			notify.Error("Failed to capture state area (%v)", err)
			matrix.Close()
			continue
		}

		m, r, e := match.Matches(matrix, img, config.Current.Templates["game"][team.Game.Name])
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

			notify.Feed(team.Game.NRGBA, "[%s] Match starting", strings.Title(team.Game.Name))

			// Also tells javascript to turn on.
			server.SetTime(10, 0)
		case state.MatchEnding:
			switch config.Current.Profile {
			case config.ProfileBroadcaster:
				if !server.Match() {
					break
				}

				notify.Feed(team.Game.NRGBA, "[%s] Match ended", strings.Title(team.Game.Name))

				// Purple score and objective results.
				regielekis, regices, regirocks, registeels := server.Objectives(team.Purple)
				notify.Feed(team.Purple.NRGBA,
					"[%s] [+%d KO%s] [+%d Regieleki%s] [+%d Regice%s] [+%d Regirock%s] [+%d Registeel%s]",
					strings.Title(team.Purple.Name),
					server.KOs(team.Purple), s(server.KOs(team.Purple)),
					regielekis, s(regielekis),
					regices, s(regices),
					regirocks, s(regirocks),
					registeels, s(registeels),
				)

				// Orange score and objective results.
				regielekis, regices, regirocks, registeels = server.Objectives(team.Orange)
				notify.Feed(team.Orange.NRGBA,
					"[%s] [+%d KO%s] [+%d Regieleki%s] [+%d Regice%s] [+%d Regirock%s] [+%d Registeel%s]",
					strings.Title(team.Orange.Name),
					server.KOs(team.Orange), s(server.KOs(team.Orange)),
					regielekis, s(regielekis),
					regices, s(regices),
					regirocks, s(regirocks),
					registeels, s(registeels),
				)
			case config.ProfilePlayer:
				o, p, self := server.Scores()
				if o+p+self > 0 {
					notify.Feed(team.Game.NRGBA, "[%s] Match ended", strings.Title(team.Game.Name))

					// Purple score and objective results.
					regielekis, regices, regirocks, registeels := server.Objectives(team.Purple)
					notify.Feed(team.Purple.NRGBA,
						"[%s] %d [+%d KO%s] [+%d Regieleki%s] [+%d Regice%s] [+%d Regirock%s] [+%d Registeel%s]",
						strings.Title(team.Purple.Name),
						p,
						server.KOs(team.Purple), s(server.KOs(team.Purple)),
						regielekis, s(regielekis),
						regices, s(regices),
						regirocks, s(regirocks),
						registeels, s(registeels),
					)

					// Orange score and objective results.
					regielekis, regices, regirocks, registeels = server.Objectives(team.Orange)
					notify.Feed(team.Orange.NRGBA,
						"[%s] %d [+%d KO%s] [+%d Regieleki%s] [+%d Regice%s] [+%d Regirock%s] [+%d Registeel%s]",
						strings.Title(team.Orange.Name),
						o,
						server.KOs(team.Orange), s(server.KOs(team.Orange)),
						regielekis, s(regielekis),
						regices, s(regices),
						regirocks, s(regirocks),
						registeels, s(registeels),
					)

					// Self score and objective results.
					notify.Feed(team.Self.NRGBA, "[%s] %d", strings.Title(team.Self.Name), self)

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
	for {
		time.Sleep(time.Second * 2)

		if config.Current.LostWindow == "" {
			continue
		}

		err := window.Reattach()
		if err != nil {
			notify.Error("Failed to reattach window (%v)", err)
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
		"[%s] [Self] Confirming %d point%s scored %s ago",
		server.Clock(),
		before,
		s(before),
		time.Since(at),
	)

	// Confirm user was not defeated with points since the goal.
	if state.KilledWithPoints.Occured(time.Second*2) != nil {
		notify.Warn("[%s] Failed to score because you were defeated (-%d)", server.Clock(), before)
		return
	}

	p := state.PressButtonToScore.Occured(time.Second * 5)
	if p != nil && !p.Verified {
		p.Verified = true
	} else {
		notify.Warn("[%s] [Self] Failed to score because the score option was not present (-%d)", server.Clock(), before)
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
