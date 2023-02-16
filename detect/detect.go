package detect

import (
	"fmt"
	"image"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
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
	"github.com/pidgy/unitehud/template"
	"github.com/pidgy/unitehud/video"
	"github.com/pidgy/unitehud/video/window"
)

var (
	Stopped = true
	closed  = false
)

func Clock() {
	for {
		time.Sleep(team.Delay(team.Time.Name))

		if Stopped || config.Current.DisableTime {
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

		notify.Time, err = match.IdentifyTime(matrix, kitchen)
		if err != nil {
			notify.Error("Failed to identify time (%v)", err)
			continue
		}
	}
}

func Close() {
	closed = true

	debug.Close()
	video.Close()

	os.Exit(0)
}

func Crash() {
	for range time.NewTicker(time.Second * 5).C {
		if closed {
			return
		}

		err := window.StartingWith(gui.Title)
		if err != nil {
			notify.Error("Failed to detect window (%v)", err)
			continue
		}
	}
}

func Defeated() {
	area := image.Rectangle{}
	modified := config.Current.Templates["killed"][team.Game.Name]
	unmodified := config.Current.Templates["killed"][team.Game.Name]

	for {
		time.Sleep(time.Second)

		if Stopped || gui.Window.Screen == nil || config.Current.DisableDefeated {
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

			notify.Feed(team.Self.RGBA, "[%s] [Self] %s", server.Clock(), str)

			if state.LastAny(time.Minute, state.Killed, state.KilledWithoutPoints, state.KilledWithPoints) != nil {
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

		if Stopped || config.Current.DisableEnergy {
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

		last := state.Last(state.HoldingEnergy, time.Hour)
		if last == nil || last.Value != points {
			notify.Feed(team.Self.RGBA, "[%s] [Self] Holding %d point%s", server.Clock(), points, s(points))
			state.Add(state.HoldingEnergy, server.Clock(), points)

			server.SetEnergy(points)

			notify.Energy, err = match.IdentifyEnergy(matrix, points)
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
	area := image.Rect(
		config.Current.Scores.Min.X+230,
		config.Current.Scores.Min.Y+90,
		config.Current.Scores.Max.X-340,
		config.Current.Scores.Max.Y-45,
	)

	var last *duplicate.Duplicate

	for {
		time.Sleep(time.Second)

		if Stopped || gui.Window.Screen == nil || config.Current.DisableKOs {
			last = nil
			continue
		}

		matrix, img, err := capture(area)
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
		case state.KOAlly, state.KOStreakAlly:
			notify.Feed(team.Purple.RGBA, "[%s] %s", server.Clock(), e)
			server.SetKO(team.Purple)
		case state.KOEnemy, state.KOStreakEnemy:
			notify.Feed(team.Orange.RGBA, "[%s] %s", server.Clock(), e)
			server.SetKO(team.Orange)
		}
	}
}

func Minimap() {
	return
	templates := config.Current.Templates["objective"][team.Game.Name]

	for {
		time.Sleep(time.Second * 2)

		if Stopped {
			time.Sleep(time.Second)
			continue
		}

		matrix, img, err := capture(config.Current.Map)
		if err != nil {
			notify.Error("Failed to capture minimap area (%v)", err)
			continue
		}

		_, r, e := match.MatchesWithAcceptance(matrix, img, templates, .7)
		if r != match.Found {
			matrix.Close()
			continue
		}

		state.Add(state.EventType(e), server.Clock(), 0)

		switch state.EventType(e) {
		case state.ObjectivePresent:
			notify.Feed(team.Game.RGBA, "[%s] Regieleki heading towards base", server.Clock())
		case state.ObjectiveReachedOrange:
			notify.Feed(team.Purple.RGBA, "[%s] Regieleki reached orange base", server.Clock())
		case state.ObjectiveReachedPurple:
			notify.Feed(team.Orange.RGBA, "[%s] Regieleki reached purple base", server.Clock())
		}

		matrix.Close()
	}
}

func Objectives() {
	area := image.Rect(
		config.Current.Scores.Min.X+100,
		config.Current.Scores.Min.Y+150,
		config.Current.Scores.Max.X-200,
		config.Current.Scores.Max.Y,
	)
	templates := config.Current.Templates["secure"][team.Game.Name]

	top, bottom := time.Time{}, time.Time{}

	for {
		time.Sleep(time.Second)

		if Stopped || gui.Window.Screen == nil || config.Current.DisableObjectives {
			top, bottom = time.Time{}, time.Time{}
			continue
		}

		matrix, img, err := capture(area)
		if err != nil {
			notify.Error("Failed to capture objective area (%v)", err)
			continue
		}

		_, r, e := match.Matches(matrix, img, templates)
		if r != match.Found {
			matrix.Close()
			continue
		}

		if time.Since(top) > time.Minute {
			switch e := state.EventType(e); e {
			case state.RegielekiSecureEnemy:
				state.Add(e, server.Clock(), 0)
				notify.Feed(team.Orange.RGBA, "[%s] [%s] Regieleki secured", server.Clock(), strings.Title(team.Orange.Name))
				server.SetRegieleki(team.Orange)
				top = time.Now()

				matrix.Close()
				continue
			case state.RegielekiSecureAlly:
				state.Add(e, server.Clock(), 0)
				notify.Feed(team.Purple.RGBA, "[%s] [%s] Regieleki secured", server.Clock(), strings.Title(team.Purple.Name))
				server.SetRegieleki(team.Purple)
				top = time.Now()

				matrix.Close()
				continue
			}
		}

		if time.Since(bottom) > time.Minute {
			switch e := state.EventType(e); e {
			case state.RegiceSecureEnemy:
				state.Add(e, server.Clock(), 0)
				notify.Feed(team.Orange.RGBA, "[%s] [%s] Regice secured", server.Clock(), strings.Title(team.Orange.Name))
				server.SetRegice(team.Orange)
				bottom = time.Now()

			case state.RegiceSecureAlly:
				state.Add(e, server.Clock(), 0)
				notify.Feed(team.Purple.RGBA, "[%s] [%s] Regice secured", server.Clock(), strings.Title(team.Purple.Name))
				server.SetRegice(team.Purple)
				bottom = time.Now()

			case state.RegirockSecureEnemy:
				state.Add(e, server.Clock(), 0)
				notify.Feed(team.Orange.RGBA, "[%s] [%s] Regirock secured", server.Clock(), strings.Title(team.Orange.Name))
				server.SetRegirock(team.Orange)
				bottom = time.Now()
			case state.RegirockSecureAlly:
				state.Add(e, server.Clock(), 0)
				notify.Feed(team.Purple.RGBA, "[%s] [%s] Regirock secured", server.Clock(), strings.Title(team.Purple.Name))
				server.SetRegirock(team.Purple)
				bottom = time.Now()

			case state.RegisteelSecureEnemy:
				state.Add(e, server.Clock(), 0)
				notify.Feed(team.Orange.RGBA, "[%s] [%s] Registeel secured", server.Clock(), strings.Title(team.Orange.Name))
				server.SetRegisteel(team.Orange)
				bottom = time.Now()
			case state.RegisteelSecureAlly:
				state.Add(e, server.Clock(), 0)
				notify.Feed(team.Purple.RGBA, "[%s] [%s] Registeel secured", server.Clock(), strings.Title(team.Purple.Name))
				server.SetRegisteel(team.Purple)
				bottom = time.Now()
			}
		}

		matrix.Close()
	}
}

func PressButtonToScore() {
	for {
		time.Sleep(time.Millisecond * 500)

		if Stopped {
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

		notify.Feed(team.Self.RGBA, "[%s] [Self] Score option present (%d)", server.Clock(), team.Energy.Holding)

		matrix.Close()
	}
}

func Preview() {
	for {
		time.Sleep(time.Second)

		img, err := video.Capture()
		if err != nil {
			notify.Error("Failed to capture preview (%v)", err)
			continue
		}

		notify.Preview = img
	}
}

func Scores(name string) {
	t := config.Current.Templates["scored"][name]

	withFirst := t
	withoutFirst := []template.Template{}
	for _, temp := range t {
		if temp.Team != team.First {
			withoutFirst = append(withoutFirst, temp)
		}
	}

	for {
		time.Sleep(team.Delay(name))

		if Stopped || config.Current.DisableScoring {
			continue
		}

		matrix, img, err := capture(config.Current.Scores)
		if err != nil {
			notify.Error("Failed to capture score area (%v)", err)
			continue
		}

		t = withFirst
		if team.First.Counted {
			t = withoutFirst
		}

		m, r, p := match.Matches(matrix, img, t)
		if r == match.NotFound {
			matrix.Close()
			continue
		}

		log.Debug().Int("points", p).Object("team", m.Team).Msg(r.String())

		switch r {
		case match.Override:
			state.Add(state.ScoreOverride, server.Clock(), p)

			server.SetScore(m.Team, -m.Team.Duplicate.Replaces)

			notify.Feed(m.Team.RGBA, "[%s] [%s] -%d (override)", server.Clock(), strings.Title(m.Team.Name), m.Team.Duplicate.Replaces)

			fallthrough
		case match.Found:
			server.SetScore(m.Team, p)

			title := fmt.Sprintf("[%s]", strings.Title(m.Team.Name))
			if m.Team.Name == team.First.Name {
				title = fmt.Sprintf("[%s] [%s]", strings.Title(m.Team.Alias), strings.Title(m.Team.Name))
			}

			notify.Feed(m.Team.RGBA, "[%s] %s +%d", server.Clock(), title, p)

			state.Add(state.ScoredBy(m.Team.Name), server.Clock(), p)

			score, err := m.Identify(matrix, p)
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

		if Stopped || gui.Window.Screen == nil {
			continue
		}

		if area.Empty() {
			b := gui.Window.Screen.Bounds()
			area = image.Rect(b.Max.X/3, 0, b.Max.X-b.Max.X/3, b.Max.Y)
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

			notify.Feed(team.Game.RGBA, "[%s] Match starting", strings.Title(team.Game.Name))

			// Also tells javascript to turn on.
			server.SetTime(10, 0)
		case state.MatchEnding:
			o, p, self := server.Scores()
			if o+p+self > 0 {
				notify.Feed(team.Game.RGBA, "[%s] Match ended", strings.Title(team.Game.Name))

				// Purple score and objective results.
				regielekis, regices, regirocks, registeels := server.Objectives(team.Purple)
				result := fmt.Sprintf("[%s] %d [+%d KO%s] [+%d Regieleki%s] [+%d Regice%s] [+%d Regirock%s] [+%d Registeel%s]",
					strings.Title(team.Purple.Name),
					p,
					server.KOs(team.Purple), s(server.KOs(team.Purple)),
					regielekis, s(regielekis),
					regices, s(regices),
					regirocks, s(regirocks),
					registeels, s(registeels),
				)
				notify.Feed(team.Purple.RGBA, result)

				// Orange score and objective results.
				regielekis, regices, regirocks, registeels = server.Objectives(team.Orange)
				result = fmt.Sprintf("[%s] %d [+%d KO%s] [+%d Regieleki%s] [+%d Regice%s] [+%d Regirock%s] [+%d Registeel%s]",
					strings.Title(team.Orange.Name),
					o,
					server.KOs(team.Orange), s(server.KOs(team.Orange)),
					regielekis, s(regielekis),
					regices, s(regices),
					regirocks, s(regirocks),
					registeels, s(registeels),
				)
				notify.Feed(team.Orange.RGBA, result)

				// Self score and objective results.
				notify.Feed(team.Self.RGBA, "[%s] %d", strings.Title(team.Self.Name), self)

				history.Add(p, o, self)
			}

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
//  - Interrupted score attempts.
//  - Defeated while scoring.
//  - ...
// If a call is made to this function it is because UniteHUD has detected were holding 0 points
// after a confirmed score match.
func energyScoredConfirm(before, after int, at time.Time) {
	if before == after {
		return
	}

	notify.Feed(team.Self.RGBA, "[%s] [Self] Confirming %d point%s scored %s ago",
		server.Clock(),
		before,
		s(before),
		time.Since(at),
	)

	scored := false
	defeated := false
	for i := 0; i < 1 && !scored && !defeated; i++ {
		// Confirm user was not defeated with points since the goal.
		lastDefeat := state.Last(state.KilledWithPoints, time.Second*2)
		if lastDefeat != nil {
			defeated = true
			continue
		}

		lastPress := state.Last(state.PressButtonToScore, time.Second*5)
		if lastPress != nil && !lastPress.Verified {
			scored = true
			lastPress.Verified = true
			continue
		}

		time.Sleep(time.Second)
	}

	if defeated {
		notify.Warn("[%s] Failed to score because you were defeated (-%d)", server.Clock(), before)
		return
	}

	if !scored {
		notify.Warn("[%s] [Self] Failed to score because the score option was not present (-%d)", server.Clock(), before)
		return
	}

	// notify.Feed(team.Self.RGBA, "[%s] Last defeated %s", server.Clock(), time.Since(team.Self.Killed))

	if server.IsFinalStretch() {
		before *= 2
	}

	go server.SetScore(team.Self, before)

	state.Add(state.PostScore, server.Clock(), before)

	notify.Feed(team.Self.RGBA, "[%s] [%s] [%s] +%d",
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
