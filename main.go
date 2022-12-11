//go:generate go-winres make --product-version=git-tag

package main

import (
	"flag"
	"fmt"
	"image"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/diode"
	"github.com/rs/zerolog/log"
	"gocv.io/x/gocv"

	"github.com/pidgy/unitehud/config"
	"github.com/pidgy/unitehud/debug"
	"github.com/pidgy/unitehud/global"
	"github.com/pidgy/unitehud/gui"
	"github.com/pidgy/unitehud/history"
	"github.com/pidgy/unitehud/match"
	"github.com/pidgy/unitehud/notify"
	"github.com/pidgy/unitehud/server"
	"github.com/pidgy/unitehud/state"
	"github.com/pidgy/unitehud/stats"
	"github.com/pidgy/unitehud/team"
	"github.com/pidgy/unitehud/template"
	"github.com/pidgy/unitehud/video"
	"github.com/pidgy/unitehud/video/window"
)

// windows
// cls && go build && unitehud.exe
// go build -ldflags="-H windowsgui"
var (
	sigq = make(chan os.Signal, 1)

	stopped = true

	imgq = map[string]chan image.Image{
		team.Game.Name: make(chan image.Image, 1),
		// team.Self.Name:   make(chan image.Image, 0),
		team.Purple.Name: make(chan image.Image, 1),
		team.Orange.Name: make(chan image.Image, 1),
		// team.Balls.Name:  make(chan image.Image, 1),
		team.First.Name: make(chan image.Image, 1),
	}
)

func init() {
	notify.System("Initializing...")

	log.Logger = zerolog.New(
		diode.NewWriter(zerolog.ConsoleWriter{
			Out:        os.Stderr,
			TimeFormat: time.Stamp,
		}, 4096, time.Nanosecond, func(missed int) {
			println("diode is falling behind")
		})).With().Timestamp().Logger()

	log.Logger = log.Logger.Level(zerolog.DebugLevel)

	profile := flag.Bool("profile", false, "start a memory/cpu profiler")
	flag.Parse()

	if *profile {
		log.Debug().Str("cpu", "cpu.prof").Str("mem", "mem.prof").Msg("starting profile")
		debug.ProfileStart()
		defer debug.ProfileStop()
	}
}

func captureClock() {
	for {
		time.Sleep(team.Delay(team.Time.Name))

		if stopped {
			continue
		}

		img, err := video.CaptureRect(config.Current.Time)
		if err != nil {
			kill(err)
		}

		matrix, err := gocv.ImageToMatRGB(img)
		if err != nil {
			kill(err)
		}

		rs, kitchen := match.Time(matrix, img)
		if rs == 0 {
			// Let's back off and not waste processing power.
			time.Sleep(time.Second * 5)
			continue
		}

		notify.Time, err = match.IdentifyTime(matrix, kitchen)
		if err != nil {
			log.Err(err).Send()
		}
	}
}

func captureEnergy() {
	assured := make(map[int]int)

	confirmScore := -1

	for {
		time.Sleep(team.Balls.Delay)

		if stopped {
			confirmScore = -1
			continue
		}

		img, err := video.CaptureRect(config.Current.Balls)
		if err != nil {
			kill(err)
		}

		matrix, err := gocv.ImageToMatRGB(img)
		if err != nil {
			kill(err)
		}

		result, _, points := match.Energy(matrix, img)
		if result != match.Found {
			matrix.Close()
			continue
		}

		// TODO: Is it better to check if we have 0 points?
		if confirmScore != -1 {
			go captureEnergyScoredConfirm(confirmScore, points, time.Now())
			confirmScore = -1
		}

		assured[points]++

		threshold := 1
		if points != team.Balls.Holding {
			threshold = 2
		}
		/*
			if team.Balls.Holding == 0 {
				threshold = 3
			} else if team.Balls.Holding != 0 && points != team.Balls.Holding {
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

			server.Balls(points)

			notify.Balls, err = match.IdentifyBalls(matrix, points)
			if err != nil {
				notify.Warn("[Self] Failed to identify energy (%v)", err)
			}

			// Can we assume change from n, where n > 0, to 0 means a goal without being defeated?
			if points == 0 || points < team.Balls.Holding {
				confirmScore = team.Balls.Holding
			}

			team.Balls.Holding = points
		}

		matrix.Close()
	}
}

func capturePressButtonToScore() {
	for {
		time.Sleep(time.Millisecond * 500)

		if stopped {
			continue
		}

		img, err := video.CaptureRect(config.Current.ScoringOption())
		if err != nil {
			kill(err)
		}

		matrix, err := gocv.ImageToMatRGB(img)
		if err != nil {
			kill(err)
		}

		m, r := match.SelfScoreOption(matrix, img)
		if r != match.Found {
			matrix.Close()
			continue
		}
		println(fmt.Sprintf("%.5f\n", m.Accepted*100))

		state.Add(state.PressButtonToScore, server.Clock(), team.Balls.Holding)

		notify.Feed(team.Self.RGBA, "[%s] [Self] Score option present (%d)", server.Clock(), team.Balls.Holding)

		matrix.Close()
	}
}

// Another step to confirm a self-score event occured. This function handles multiple edge cases
// that can result in invalid detections, such as:
//  - Interrupted score attempts.
//  - Defeated while scoring.
//  - ...
// If a call is made to this function it is because UniteHUD has detected were holding 0 points
// after a confirmed score match.
func captureEnergyScoredConfirm(before, after int, at time.Time) {
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

	go server.Publish(team.Self, before)

	state.Add(state.PostScore, server.Clock(), before)

	notify.Feed(team.Self.RGBA, "[%s] [%s] [%s] +%d",
		server.Clock(),
		strings.Title(team.Purple.Name),
		strings.Title(team.Self.Name),
		before,
	)
}

func captureKilled() {
	area := image.Rectangle{}
	modified := config.Current.Templates["killed"][team.Game.Name]
	unmodified := config.Current.Templates["killed"][team.Game.Name]

	for {
		time.Sleep(time.Second)

		if stopped || gui.Window.Screen == nil {
			continue
		}

		if area.Empty() {
			b := gui.Window.Screen.Bounds()
			area = image.Rect(b.Max.X/3, b.Max.Y/2, b.Max.X-b.Max.X/3, b.Max.Y-b.Max.Y/3)
		}

		img, err := video.CaptureRect(area)
		if err != nil {
			kill(err)
		}

		matrix, err := gocv.ImageToMatRGB(img)
		if err != nil {
			kill(err)
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
				str += " with unscored points"
			}
			notify.Feed(team.Self.RGBA, "[%s] [Self] %s", server.Clock(), str)
		default:
			modified = unmodified
		}

		matrix.Close()
	}
}

func captureMinimap() {
	return
	templates := config.Current.Templates["objective"][team.Game.Name]

	for {
		time.Sleep(time.Second * 2)

		if stopped {
			time.Sleep(time.Second)
			continue
		}

		img, err := video.CaptureRect(config.Current.Map)
		if err != nil {
			kill(err)
		}

		matrix, err := gocv.ImageToMatRGB(img)
		if err != nil {
			kill(err)
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

func captureObjectives() {
	area := image.Rectangle{}
	templates := config.Current.Templates["secure"][team.Game.Name]
	found := time.Time{}

	for {
		time.Sleep(time.Millisecond * 500)

		if stopped || gui.Window.Screen == nil {
			found = time.Time{}
			continue
		}

		if time.Since(found) < time.Minute {
			continue
		}

		if area.Empty() {
			b := gui.Window.Screen.Bounds()
			area = image.Rect(b.Max.X/2, 0, b.Max.X-b.Max.X/3, b.Max.Y/3)
		}

		img, err := video.CaptureRect(area)
		if err != nil {
			kill(err)
		}

		matrix, err := gocv.ImageToMatRGB(img)
		if err != nil {
			kill(err)
		}

		_, r, e := match.Matches(matrix, img, templates)
		if r != match.Found {
			matrix.Close()
			continue
		}

		switch e := state.EventType(e); e {
		case state.RegielekiAdvancingAlly:
			state.Add(state.RegielekiAdvancingAlly, server.Clock(), 20)
			notify.Feed(team.Orange.RGBA, "[%s] [%s] Regieleki secured", server.Clock(), strings.Title(team.Orange.Name))
			server.PublishRegieleki(team.Orange)
		case state.RegielekiAdvancingEnemy:
			state.Add(state.RegielekiAdvancingEnemy, server.Clock(), 20)
			notify.Feed(team.Purple.RGBA, "[%s] [%s] Regieleki secured", server.Clock(), strings.Title(team.Purple.Name))
			server.PublishRegieleki(team.Purple)
		default:
			notify.Warn("Unknown objective secured for event (%s)", e)
		}

		matrix.Close()

		found = time.Now()
	}
}

func capturePreview() {
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

func captureScores(name string, m map[string][]template.Template) {
	t, ok := m[name]
	if !ok {
		notify.Error("Failed to start score captures for unknown team \"%s\"", name)
		return
	}

	withFirst := t
	withoutFirst := []template.Template{}
	for _, temp := range t {
		if temp.Team != team.First {
			withoutFirst = append(withoutFirst, temp)
		}
	}

	for {
		time.Sleep(team.Delay(name))

		if stopped {
			continue
		}

		img, err := video.CaptureRect(config.Current.Scores)
		if err != nil {
			kill(err)
		}

		matrix, err := gocv.ImageToMatRGB(img)
		if err != nil {
			kill(err)
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

			go server.Publish(m.Team, -m.Team.Duplicate.Replaces)

			notify.Feed(m.Team.RGBA, "[%s] [%s] -%d (override)", server.Clock(), strings.Title(m.Team.Name), m.Team.Duplicate.Replaces)

			fallthrough
		case match.Found:
			go server.Publish(m.Team, p)

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

func captureStates() {
	area := image.Rectangle{}
	templates := config.Current.Templates["game"][team.Game.Name]

	for {
		time.Sleep(time.Second * 2)

		if stopped || gui.Window.Screen == nil {
			continue
		}

		if area.Empty() {
			b := gui.Window.Screen.Bounds()
			area = image.Rect(b.Max.X/3, 0, b.Max.X-b.Max.X/3, b.Max.Y)
		}

		img, err := video.CaptureRect(area)
		if err != nil {
			kill(err)
		}

		matrix, err := gocv.ImageToMatRGB(img)
		if err != nil {
			kill(err)
		}

		m, r, e := match.Matches(matrix, img, templates)
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
			team.Clear()
			state.Clear()

			notify.Feed(team.Game.RGBA, "[%s] Match starting", strings.Title(team.Game.Name))
			// Also tells javascript to turn on.
			server.Time(10, 0)
		case state.MatchEnding:
			o, p, self := server.Scores()
			if o+p+self > 0 {
				notify.Feed(team.Game.RGBA, "[%s] Match ended", strings.Title(team.Game.Name))
				notify.Feed(team.Purple.RGBA, "[%s] %d (+%d Regieleki%s)", strings.Title(team.Purple.Name), p, server.RegielekisSecured(team.Purple), s(server.RegielekisSecured(team.Purple)))
				notify.Feed(team.Orange.RGBA, "[%s] %d (+%d Regieleki%s)", strings.Title(team.Orange.Name), o, server.RegielekisSecured(team.Orange), s(server.RegielekisSecured(team.Orange)))
				notify.Feed(team.Self.RGBA, "[%s] %d", strings.Title(team.Self.Name), self)

				str := "Regielekis:"
				objs := server.Regielekis()
				for i, t := range objs {
					str += fmt.Sprintf(" %s", strings.Title(t))
					if i < len(objs)-1 {
						str += ","
					}
				}
				notify.Append(server.RegielekiAdv().RGBA, str)

				history.Add(p, o, self)
			}

			server.Clear()
			team.Clear()
		}

		matrix.Close()
	}
}

func captureWindows() {
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

var closed = false

func handleClosing() {
	closed = true

	debug.Close()
	video.Close()
	os.Exit(0)
}

func handleCrash() {
	for range time.NewTicker(time.Second * 5).C {
		if closed {
			return
		}

		err := window.StartingWith(gui.Title)
		if err != nil {
			kill(err)
		}
	}
}

func kill(errs ...error) {
	if len(errs) > 0 {
		config.Current.Report(errs[0].Error())
	}

	for _, err := range errs {
		log.Err(err).Msg(gui.Title)
	}

	time.Sleep(time.Second)

	sig := os.Kill
	if len(errs) == 0 {
		sig = os.Interrupt
	}

	sigq <- sig
}

func s(size int) string {
	if size == 1 {
		return ""
	}
	return "s"
}

func signals() {
	signal.Notify(sigq, syscall.SIGINT, syscall.SIGTERM)
	s := <-sigq

	log.Info().Stringer("signal", s).Msg("closing...")

	handleClosing()

	os.Exit(1)
}

func main() {
	runtime.LockOSThread()

	go signals()

	err := config.Load()
	if err != nil {
		kill(err)
	}

	err = video.Load()
	if err != nil {
		notify.Error("Failed to load windows (%v)", err)
	}

	err = server.Start()
	if err != nil {
		kill(err)
	}

	log.Info().
		Bool("record", config.Current.Record).
		Str("imgs", "img/"+config.Current.Dir+"/").
		Msg("unitehud")

	notify.System("Debug Mode: %t", global.DebugMode)
	notify.System("Server address: \"%s\"", server.Address)
	notify.System("Recording: %t", config.Current.Record)
	notify.System("Image directory: img/%s/", config.Current.Dir)

	// Detection routines.
	go captureClock()
	go captureEnergy()
	go captureKilled()
	go captureMinimap()
	go captureObjectives()
	go capturePressButtonToScore()
	go capturePreview()
	go captureStates()
	go captureWindows()
	go captureScores(team.Purple.Name, config.Current.Templates["scored"])
	go captureScores(team.Orange.Name, config.Current.Templates["scored"])
	go captureScores(team.First.Name, config.Current.Templates["scored"])
	go handleCrash()

	lastWindow := ""

	gui.New()
	defer gui.Window.Open()

	go func() {
		for action := range gui.Window.Actions {
			switch action {
			case gui.Closing:
				handleClosing()
				return
			case gui.Start:
				if !stopped {
					continue
				}

				notify.Announce("Starting %s...", gui.Title)

				notify.Clear()
				server.Clear()
				team.Clear()
				stats.Clear()
				state.Clear()

				stopped = false

				notify.Announce("Started %s", gui.Title)

				state.Add(state.Nothing, server.Clock(), -1)

				server.Started(true)
			case gui.Stop:
				if stopped {
					continue
				}
				stopped = true

				notify.Denounce("Stopping %s...", gui.Title)

				// Wait for the capture routines to go idle.
				time.Sleep(time.Second * 2)

				notify.Denounce("Stopped %s", gui.Title)

				server.Clear()
				team.Clear()

				server.Started(false)

				if !config.Current.Record {
					continue
				}

				fallthrough
			case gui.Record:
				config.Current.Record = !config.Current.Record

				str := "Closing"
				if config.Current.Record {
					str = "Recording"
				}

				notify.System("%s template match results in %s", str, debug.Dir)
				switch config.Current.Record {
				case true:
					err := debug.LoggingStart()
					if err != nil {
						kill(err)
					}

					notify.System("Using \"%s\" directory for recording data", debug.Dir)

					err = config.Current.Save()
					if err != nil {
						kill(err)
					}
				case false:
					notify.System("Closing open files in %s", debug.Dir)

					debug.LoggingStop()
				}
			case gui.Open:
				notify.System("Opening \"%s\"", debug.Dir)

				err := debug.Open()
				if err != nil {
					notify.Error("Failed to open \"%s\" (%v)", debug.Dir, err)
				}
			case gui.Refresh:
				err := video.Load()
				if err != nil {
					notify.Error("Failed to load windows (%v)", err)
				}

				if lastWindow != config.Current.Window {
					lastWindow = config.Current.Window
					notify.System("Capture window set to \"%s\"", lastWindow)
				}
			case gui.Debug:
				was := stopped

				stopped = true

				notify.Announce("Reloading image templates...")

				time.Sleep(time.Second * 3)

				err := config.Load()
				if err != nil {
					notify.Error("Failed to reload config (%v)", err)
					continue
				}

				notify.Announce("Successfully reloaded image templates")

				stopped = was
			}
		}
	}()

	notify.System("Initialized")
}
