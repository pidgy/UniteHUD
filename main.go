//go:generate go-winres make --product-version=git-tag

package main

import (
	"flag"
	"image"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gocv.io/x/gocv"

	"github.com/pidgy/screenshot"
	"github.com/pidgy/unitehud/config"
	"github.com/pidgy/unitehud/dev"
	"github.com/pidgy/unitehud/match"
	"github.com/pidgy/unitehud/notify"
	"github.com/pidgy/unitehud/pipe"
	"github.com/pidgy/unitehud/rgba"
	"github.com/pidgy/unitehud/team"
	"github.com/pidgy/unitehud/template"
	"github.com/pidgy/unitehud/window/gui"
	"github.com/pidgy/unitehud/window/terminal"
)

// windows
// cls && go build && unitehud.exe
// go build -ldflags="-H windowsgui"
var (
	sigq = make(chan os.Signal, 1)
)

var (
	dup    = false
	record = false
	missed = false
	addr   = ":17069"
	term   = false
)

func init() {
	flag.BoolVar(&record, "record", record, "record data such as matched images and logs for developer-specific debugging")
	flag.BoolVar(&dup, "dup", dup, "record duplicate image matching")
	flag.BoolVar(&missed, "missed", missed, "record missed image matching")
	flag.BoolVar(&term, "terminal", term, "use a custom terminal style window for debugging")
	flag.StringVar(&addr, "addr", addr, "http/websocket serve address")
	avg := flag.Float64("match", 91, `0-100% certainty when processing score values`)
	level := flag.String("v", zerolog.LevelErrorValue, "log level (panic, fatal, error, warn, info, debug)")
	flag.Parse()

	log.Logger = zerolog.New(
		zerolog.ConsoleWriter{
			Out:        os.Stderr,
			TimeFormat: time.Stamp,
		},
	).With().Timestamp().Logger()

	pipe.Socket = pipe.New(addr)

	go signals()

	lvl, err := zerolog.ParseLevel(*level)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	log.Logger = log.Logger.Level(lvl)

	err = config.Load(float32(*avg)/100, record, missed, dup)
	if err != nil {
		kill(err)
	}
}

func capture(name string, imgq chan *image.RGBA, paused *bool) {
	for {
		if *paused {
			time.Sleep(team.Delay(name))
			continue
		}

		img, err := screenshot.CaptureRect(config.Current.Scores)
		if err != nil {
			kill(err)
		}

		imgq <- img

		time.Sleep(team.Delay(name))
	}
}

func loop(t []template.Template, imgq chan *image.RGBA) {
	runtime.LockOSThread()

	for img := range imgq {
		matrix, err := gocv.ImageToMatRGB(img)
		if err != nil {
			kill(err)
		}

		m := match.Match{}

		m.Matches(matrix, img, t)

		matrix.Close()
	}
}

func kill(errs ...error) {
	for _, err := range errs {
		log.Error().Err(err).Send()
		time.Sleep(time.Millisecond)
	}

	sig := os.Kill
	if len(errs) == 0 {
		sig = os.Interrupt
	}

	sigq <- sig
}

func seconds(paused *bool) {
	m := match.Match{}

	for {
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

		rs, _ := m.Time(matrix, img)
		if rs == 0 {
			// Let's back off and not waste processing power.
			time.Sleep(time.Second * 5)
		}

		time.Sleep(team.Delay(team.Time.Name))
	}
}

func signals() {
	signal.Notify(sigq, syscall.SIGINT, syscall.SIGTERM)
	s := <-sigq

	terminal.Close()
	log.Info().Stringer("signal", s).Msg("closing...")
	os.Exit(1)
}

func main() {
	gui.New()
	defer gui.Window.Open()

	log.Info().
		Bool("duplicates", config.Current.RecordDuplicates).
		Bool("missed", config.Current.RecordMissed).
		Bool("record", config.Current.Record).
		Str("imgs", "img/"+config.Current.Dir+"/").
		Str("match", strconv.Itoa(int(config.Current.Acceptance*100))+"%").
		Str("addr", addr).Msg("unitehud")

	notify.Feed(rgba.White, "Pokemon Unite HUD Server")
	notify.Feed(rgba.Green, "Options")
	notify.Feed(rgba.Green, "Server address: \"%s\"", addr)
	notify.Feed(rgba.Bool(config.Current.Record), "Record matched points: %t", config.Current.Record)
	notify.Feed(rgba.Bool(config.Current.RecordMissed), "Record missed points: %t", config.Current.RecordMissed)
	notify.Feed(rgba.Bool(config.Current.RecordDuplicates), "Record duplicate points: %t", config.Current.RecordDuplicates)
	notify.Feed(rgba.Green, "Image directory: img/%s/", config.Current.Dir)
	notify.Feed(rgba.Green, "Match acceptance: %d%s", int(config.Current.Acceptance*100), "%")

	if config.Current.Record || config.Current.RecordMissed {
		err := dev.New()
		if err != nil {
			kill(err)
		}

		notify.Feed(rgba.Green, "Created tmp/ directory for match recording")
	}

	imgq := map[string]chan *image.RGBA{
		team.None.Name:   make(chan *image.RGBA, 1),
		team.Self.Name:   make(chan *image.RGBA, 4),
		team.Purple.Name: make(chan *image.RGBA, 1),
		team.Orange.Name: make(chan *image.RGBA, 1),
		team.Balls.Name:  make(chan *image.RGBA, 1),
	}

	paused := true

	for category := range config.Current.Templates {
		if category == "points" || category == "time" {
			continue
		}

		for name := range config.Current.Templates[category] {
			for i := 0; i < cap(imgq[name]); i++ {
				go capture(name, imgq[name], &paused)
				go loop(config.Current.Templates[category][name], imgq[name])
			}
		}
	}

	go seconds(&paused)

	go func() {
		for {
			switch <-gui.Window.Actions {
			case gui.Start:
				notify.Feed(rgba.Green, "Starting...")
				paused = false
			case gui.Stop:
				notify.Feed(rgba.Green, "Stopping...")
				paused = true
			case gui.Record:
				config.Current.Record = !config.Current.Record

				switch config.Current.Record {
				case true:
					err := dev.New()
					if err != nil {
						kill(err)
					}

					notify.Feed(rgba.White, "Using tmp/ directory for recording matches")

					err = config.Current.Save()
					if err != nil {
						kill(err)
					}
				case false:
					notify.Feed(rgba.White, "Closing open files in tmp/")

					dev.End()
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
