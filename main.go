//go:generate go-winres make --product-version=git-tag

package main

import (
	"flag"
	"image"
	"image/color"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gocv.io/x/gocv"

	"github.com/pidgy/screenshot"
	"github.com/pidgy/unitehud/config"
	"github.com/pidgy/unitehud/dev"
	"github.com/pidgy/unitehud/gui"
	"github.com/pidgy/unitehud/match"
	"github.com/pidgy/unitehud/notify"
	"github.com/pidgy/unitehud/rgba"
	"github.com/pidgy/unitehud/server"
	"github.com/pidgy/unitehud/state"
	"github.com/pidgy/unitehud/team"
	"github.com/pidgy/unitehud/template"
)

// windows
// cls && go build && unitehud.exe
// go build -ldflags="-H windowsgui"
var (
	sigq = make(chan os.Signal, 1)
)

var (
	record = false
	addr   = ":17069"
)

var (
	imgq = map[string]chan image.Image{
		team.Game.Name: make(chan image.Image, 1),
		// team.Self.Name:   make(chan image.Image, 0),
		team.Purple.Name: make(chan image.Image, 1),
		team.Orange.Name: make(chan image.Image, 1),
		team.Balls.Name:  make(chan image.Image, 1),
		team.First.Name:  make(chan image.Image, 1),
	}
)

var (
	empty = gocv.NewMat()
)

func init() {
	notify.Feed(rgba.White, "Pokemon Unite HUD Server")

	flag.StringVar(&addr, "addr", addr, "http/websocket serve address")
	flag.BoolVar(&record, "record", record, "record data such as matched images and logs for developer-specific debugging")
	level := flag.String("v", zerolog.LevelErrorValue, "log level (panic, fatal, error, warn, info, debug)")
	flag.Parse()

	log.Logger = zerolog.New(
		zerolog.ConsoleWriter{
			Out:        os.Stderr,
			TimeFormat: time.Stamp,
		},
	).With().Timestamp().Logger()

	server.New(addr)

	lvl, err := zerolog.ParseLevel(*level)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	log.Logger = log.Logger.Level(lvl)

	err = config.Load(record)
	if err != nil {
		kill(err)
	}
}

func capture(name string, imgq chan image.Image, paused *bool) {
	for {
		time.Sleep(team.Delay(name))

		if *paused {
			time.Sleep(time.Second)
			continue
		}

		img, err := screenshot.CaptureRect(config.Current.Scores)
		if err != nil {
			kill(err)
			return
		}

		imgq <- img
	}
}

func first(t []template.Template, imgq chan image.Image) {
	for img := range imgq {
		matrix, err := gocv.ImageToMatRGB(img)
		if err != nil {
			kill(err)
		}

		m := &match.Match{}

		r, p := m.Matches(matrix, img, t)
		if r == match.NotFound {
			matrix.Close()
			continue
		}

		log.Info().Int("points", p).Object("team", m.Team).Msg(r.String())

		switch r {
		case match.Found:
			if p < 1 {
				break
			}

			go server.Publish(m.Team, p)

			if config.Current.Record {
				loc := dev.Capture(img, matrix, m.Team, m.Point, "capture", p)
				dev.Log("matched points for %s (%d)", m.Team.Name, p)
				dev.Log("Saved at %s", loc)

				if m.Team == team.Self {
					loc := dev.Capture(notify.Balls, empty, m.Team, m.Point, "capture", p)
					dev.Log("Saved at %s", loc)
				}
			}

			notify.Feed(m.Team.RGBA, "[%s] +%d %s", server.Clock(), p, m.Team.Alias)

			switch m.Team.Name {
			case team.Self.Name:
				notify.SelfScore, err = m.Identify(matrix, p)
				if err != nil {
					log.Error().Err(err).Send()
				}
			case team.First.Name:
				score, err := m.Identify(matrix, p)
				if err != nil {
					log.Error().Err(err).Send()
					break
				}

				if team.First.Alias == team.Purple.Name {
					notify.PurpleScore = score
				} else {
					notify.OrangeScore = score
				}
			case team.Purple.Name:
				notify.PurpleScore, err = m.Identify(matrix, p)
				if err != nil {
					log.Error().Err(err).Send()
				}
			case team.Orange.Name:
				notify.OrangeScore, err = m.Identify(matrix, p)
				if err != nil {
					log.Error().Err(err).Send()
				}
			}
		case match.Missed:
			notify.Feed(rgba.Red, "Missed points matched for %s! (%d?)", m.Team.Name, p)

			if config.Current.Record {
				dev.Capture(img, matrix, m.Team, m.Point, "missed", p)
			} else {
				notify.Feed(rgba.Red, "Select the \"Record\" button to view missed points in /tmp")
			}
		case match.Invalid:
			notify.Feed(rgba.Red, "Scored match is outside the configured selection area")
		case match.Duplicate:
			if config.Current.Record {
				dev.Capture(img, matrix, m.Team, m.Point, "duplicate", p)
				dev.Log("duplicate points matched for %s (%d)", m.Team.Name, p)
			}
		}

		matrix.Close()
	}
}

func minimap(paused *bool) {
}

func orbs(paused *bool) {
	assured := make(map[int]int)

	for {
		time.Sleep(team.Delay(team.Balls.Name))

		if *paused {
			time.Sleep(time.Second)
			continue
		}

		img, err := screenshot.CaptureRect(config.Current.Balls)
		if err != nil {
			kill(err)
		}

		matrix, err := gocv.ImageToMatRGB(img)
		if err != nil {
			kill(err)
		}

		result, order, points := match.Balls2(matrix, img)
		if result != match.Found {
			continue
		}

		assured[points]++

		threshold := 1
		if points == 0 {
			threshold = 2
		}

		// TODO: touching pad
		if team.Balls.Holding != 0 && team.Balls.Holding/10 == points {
			continue
		}

		if assured[points] < threshold {
			continue
		}
		assured = make(map[int]int)

		if points != team.Balls.Holding {
			log.Info().Int("points", points).Int("prev", team.Balls.Holding).Object("team", team.Balls).Ints("read", order).Msg(result.String())
			s := "s"
			if points == 1 {
				s = ""
			}

			notify.Feed(team.Self.RGBA, "[%s] Holding %d point%s", server.Clock(), points, s)

			state.AddEvent(state.HoldingBalls, server.Clock(), points)
		}

		notify.Balls, err = match.IdentifyBalls(matrix, points)
		if err != nil {
			log.Error().Err(err).Send()
			continue
		}

		team.Balls.HoldingReset = false
		team.Balls.Holding = points

		server.Balls(points)
	}
}

func scoring(paused *bool) {
	for {
		time.Sleep(time.Millisecond * 1500)

		if *paused {
			time.Sleep(time.Second)
			continue
		}

		img, err := screenshot.CaptureRect(config.Current.Scoring())
		if err != nil {
			kill(err)
		}

		matrix, err := gocv.ImageToMatRGB(img)
		if err != nil {
			kill(err)
		}

		m := &match.Match{}
		r, n := m.Matches(matrix, img, config.Current.Templates["scoring"][team.Game.Name])
		if r != match.Found {
			matrix.Close()
			continue
		}

		past := state.PastEvents(state.PostScore, time.Second*3)
		if len(past) < 2 {
			go server.Publish(team.Self, n)
			notify.Feed(team.Self.RGBA, "[%s] +%d (self)", server.Clock(), n)
		} else {
			total := 0
			for _, event := range past[1:] {
				total -= event.Value
			}

			go server.Publish(team.Self, total)
			notify.Feed(team.Self.RGBA, "[%s] %d (invalid self)", server.Clock(), total)
		}

		matrix.Close()
	}
}

func killed(paused *bool) {
	area := image.Rectangle{}
	modified := config.Current.Templates["killed"][team.Game.Name]
	unmodified := config.Current.Templates["killed"][team.Game.Name]

	for {
		time.Sleep(time.Second)

		if *paused || gui.Window.Screen == nil {
			time.Sleep(time.Second)
			continue
		}

		if area.Empty() {
			b := gui.Window.Screen.Bounds()
			area = image.Rect(b.Max.X/3, b.Max.Y/2, b.Max.X-b.Max.X/3, b.Max.Y-b.Max.Y/3)
		}

		img, err := screenshot.CaptureRect(area)
		if err != nil {
			kill(err)
		}

		matrix, err := gocv.ImageToMatRGB(img)
		if err != nil {
			kill(err)
		}

		gocv.IMWrite("killed2.png", matrix)

		m := &match.Match{}

		r, e := m.Matches(matrix, img, modified)
		if r == match.Found {
			state.AddEvent(state.EventType(e), server.Clock(), -1)

			switch e := state.EventType(e); e {
			case state.Killed:
				modified = modified[1:] // Remove killed event.

				team.Self.Killed = time.Now()

				notify.Feed(color.RGBA(rgba.DarkRed), "[%s] Killed", server.Clock())
			case state.KilledWithPoints:
				modified = modified[1:]

				notify.Feed(color.RGBA(rgba.DarkRed), "[%s] Killed with %d points", server.Clock(), team.Balls.Holding)
			case state.KilledWithoutPoints:
				modified = modified[1:]

				notify.Feed(color.RGBA(rgba.DarkRed), "[%s] Killed without points", server.Clock())
			}
		} else {
			if time.Since(team.Self.Killed) > time.Minute {
				modified = unmodified
			}
		}

		matrix.Close()
	}
}

func states(paused *bool) {
	area := image.Rectangle{}

	for {
		time.Sleep(team.Game.Delay)

		if *paused || gui.Window.Screen == nil {
			time.Sleep(time.Second)
			continue
		}

		if area.Empty() {
			b := gui.Window.Screen.Bounds()
			area = image.Rect(b.Max.X/3, 0, b.Max.X-b.Max.X/3, b.Max.Y)
		}

		img, err := screenshot.CaptureRect(area)
		if err != nil {
			kill(err)
		}

		matrix, err := gocv.ImageToMatRGB(img)
		if err != nil {
			kill(err)
		}

		(&match.Match{}).Matches(matrix, img, config.Current.Templates["game"][team.Game.Name])

		matrix.Close()
	}
}

func seconds(paused *bool) {
	m := match.Match{}

	for {
		time.Sleep(team.Delay(team.Time.Name))

		if *paused {
			time.Sleep(time.Second)
			continue
		}

		img, err := screenshot.CaptureRect(config.Current.Time)
		if err != nil {
			kill(err)
		}

		matrix, err := gocv.ImageToMatRGB(img)
		if err != nil {
			kill(err)
		}

		rs, kitchen := m.Time(matrix, img)
		if rs == 0 {
			// Let's back off and not waste processing power.
			time.Sleep(time.Second * 5)
		} else {
			notify.Time, err = match.IdentifyTime(matrix, kitchen)
			if err != nil {
				log.Error().Err(err).Send()
			}
		}
	}
}

func main() {
	go signals()

	gui.New()
	defer gui.Window.Open()

	log.Info().
		Bool("record", config.Current.Record).
		Str("imgs", "img/"+config.Current.Dir+"/").
		Str("addr", addr).
		Msg("unitehud")

	notify.Append(rgba.Green, "Server address: \"%s\"", addr)
	notify.Append(rgba.Bool(config.Current.Record), "Recording: %t", config.Current.Record)
	notify.Append(rgba.Green, "Image directory: img/%s/", config.Current.Dir)

	paused := true

	for category := range config.Current.Templates {
		switch category {
		case "game":
			// Ignore first-stage matching for game.
			continue
		case "points":
			// Ignore first-stage matching for points.
			continue
		case "time":
			// Ignore first-stage matching for time in this context.
			continue
		}

		for name := range config.Current.Templates[category] {
			for i := 0; i < cap(imgq[name]); i++ {
				go capture(name, imgq[name], &paused)
				go first(config.Current.Templates[category][name], imgq[name])

				// Stagger processing for workers by sleeping.
				time.Sleep(time.Millisecond * 250)
			}
		}
	}

	go killed(&paused)
	go minimap(&paused)
	go orbs(&paused)
	go seconds(&paused)
	go states(&paused)
	go scoring(&paused)

	go func() {
		for action := range gui.Window.Actions {
			switch action {
			case gui.Closing:
				dev.Close()
				os.Exit(0)
				return
			case gui.Start:
				if !paused {
					continue
				}

				notify.Feed(rgba.Green, "Starting...")

				notify.Clear()
				server.Clear()
				team.Clear()
				// stats.Clear()
				state.Clear()

				paused = false
			case gui.Stop:
				if paused {
					continue
				}
				paused = true

				server.Clear()
				team.Clear()

				notify.Feed(rgba.Green, "Stopping...")

				if !config.Current.Record {
					continue
				}

				fallthrough
			case gui.Record:
				config.Current.Record = !config.Current.Record
				notify.Feed(rgba.Bool(config.Current.Record), "Recording images: %t", config.Current.Record)

				switch config.Current.Record {
				case true:
					err := dev.Start()
					if err != nil {
						kill(err)
					}

					notify.Feed(rgba.White, "Using tmp/ directory for recording data")

					err = config.Current.Save()
					if err != nil {
						kill(err)
					}
				case false:
					notify.Feed(rgba.White, "Closing open files in tmp/")

					dev.Stop()
				}
			case gui.Open:
				err := dev.Open()
				if err != nil {
					notify.Feed(rgba.Red, "%s", err.Error())
				}
			}
		}
	}()

	notify.Feed(rgba.White, "Not started")
}

func signals() {
	signal.Notify(sigq, syscall.SIGINT, syscall.SIGTERM)
	s := <-sigq

	log.Info().Stringer("signal", s).Msg("closing...")
	os.Exit(1)
}

func kill(errs ...error) {
	for _, err := range errs {
		log.Error().Err(err).Msg("killing unitehud")
	}

	time.Sleep(time.Second)

	sig := os.Kill
	if len(errs) == 0 {
		sig = os.Interrupt
	}

	sigq <- sig
}
