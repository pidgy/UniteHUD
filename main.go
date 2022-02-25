//go:generate go-winres make --product-version=git-tag

package main

import (
	"flag"
	"image"
	"image/color"
	"math"
	"os"
	"os/signal"
	"strconv"
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
	"github.com/pidgy/unitehud/stats"
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
	dup    = false
	record = false
	missed = false
	addr   = ":17069"
)

var (
	imgq = map[string]chan image.Image{
		team.Game.Name:   make(chan image.Image, 1),
		team.Self.Name:   make(chan image.Image, 2),
		team.Purple.Name: make(chan image.Image, 1),
		team.Orange.Name: make(chan image.Image, 1),
		team.Balls.Name:  make(chan image.Image, 1),
		team.First.Name:  make(chan image.Image, 1),
	}
)

func init() {
	notify.Feed(rgba.White, "Pokemon Unite HUD Server")

	flag.StringVar(&addr, "addr", addr, "http/websocket serve address")
	flag.BoolVar(&dup, "dup", dup, "record duplicate image matching")
	flag.BoolVar(&missed, "missed", missed, "record missed image matching")
	flag.BoolVar(&record, "record", record, "record data such as matched images and logs for developer-specific debugging")

	avg := flag.Float64("match", 91, `0-100% certainty when processing score values`)
	stats := flag.Bool("stats", false, "record image template stats in logs and print data to UI")

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

	err = config.Load(float32(*avg)/100, record, missed, dup, *stats)
	if err != nil {
		kill(err)
	}
}

func capture(name string, imgq chan image.Image, paused *bool) {
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

func gray(img *image.RGBA) *image.Gray {
	// Create a new grayscale image
	bounds := img.Bounds()
	w, h := bounds.Max.X, bounds.Max.Y
	gray := image.NewGray(image.Rectangle{image.Point{0, 0}, image.Point{w, h}})
	for x := 0; x < w; x++ {
		for y := 0; y < h; y++ {
			imageColor := img.At(x, y)
			rr, gg, bb, _ := imageColor.RGBA()
			r := math.Pow(float64(rr), 2.2)
			g := math.Pow(float64(gg), 2.2)
			b := math.Pow(float64(bb), 2.2)
			m := math.Pow(0.2125*r+0.7154*g+0.0721*b, 1/2.2)
			Y := uint16(m + 0.5)
			c := color.Gray{uint8(Y >> 8)}
			gray.Set(x, y, c)
		}
	}

	return gray
}

func process(t []template.Template, imgq chan image.Image) {
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
				dev.Capture(img, matrix, m.Team, m.Point, "capture", false, p)
			}

			notify.Feed(m.Team.RGBA, "[%s] +%d", server.Clock(), p)

			switch m.Team.Name {
			case team.Self.Name:
				notify.SelfScore, err = m.Identify(matrix, p)
				if err != nil {
					log.Error().Err(err).Send()
				}
			case team.Purple.Name, team.First.Name:
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
			if config.Current.RecordMissed {
				dev.Capture(img, matrix, m.Team, m.Point, "missed", false, p)
			} else {
				notify.Feed(color.RGBA(rgba.SlateGray), "Set \"RecordMissed: true\" to view missed points in /tmp")
			}
		case match.Invalid:
			notify.Feed(rgba.Red, "Scored match is outside the configured selection area")

			if config.Current.RecordMissed {
				dev.Capture(img, matrix, m.Team, m.Point, "invalid", false, 0)
			}
		case match.Duplicate:
			if config.Current.RecordDuplicates {
				dev.Capture(img, matrix, m.Team, m.Point, "", true, p)
			}
		}

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

func balls(paused *bool) {
	assured := make(map[int]int)

	for {
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

		result, order, points := match.Balls(matrix, img)
		if result == match.Found {
			_, ok := assured[points]
			assured = make(map[int]int)
			if !ok {
				assured[points]++
				goto sleep
			}

			if points != team.Balls.Holding {
				log.Info().Int("points", points).Int("prev", team.Balls.Holding).Object("team", team.Balls).Ints("read", order).Msg(result.String())
			}

			notify.Balls, err = match.IdentifyBalls(matrix, points)
			if err != nil {
				log.Error().Err(err).Send()
				goto sleep
			}

			team.Balls.HoldingReset = false
			team.Balls.Holding = points
		}

	sleep:
		time.Sleep(team.Delay(team.Balls.Name))
	}
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

		time.Sleep(team.Delay(team.Time.Name))
	}
}

func signals() {
	signal.Notify(sigq, syscall.SIGINT, syscall.SIGTERM)
	s := <-sigq

	log.Info().Stringer("signal", s).Msg("closing...")
	os.Exit(1)
}

func main() {
	go signals()

	gui.New()
	defer gui.Window.Open()

	log.Info().
		Bool("duplicates", config.Current.RecordDuplicates).
		Bool("missed", config.Current.RecordMissed).
		Bool("record", config.Current.Record).
		Bool("stats", config.Current.Stats).
		Str("imgs", "img/"+config.Current.Dir+"/").
		Str("match", strconv.Itoa(int(config.Current.Acceptance*100))+"%").
		Str("addr", addr).Msg("unitehud")

	notify.Append(rgba.Green, "Server address: \"%s\"", addr)
	notify.Append(rgba.Bool(config.Current.Record), "Record matched points: %t", config.Current.Record)
	notify.Append(rgba.Bool(config.Current.RecordMissed), "Record missed points: %t", config.Current.RecordMissed)
	notify.Append(rgba.Bool(config.Current.RecordDuplicates), "Record duplicate points: %t", config.Current.RecordDuplicates)
	notify.Append(rgba.Green, "Image directory: img/%s/", config.Current.Dir)
	notify.Append(rgba.Green, "Match acceptance: %d%s", int(config.Current.Acceptance*100), "%")

	if config.Current.Record || config.Current.RecordMissed {
		err := dev.New()
		if err != nil {
			kill(err)
		}

		notify.Feed(rgba.Green, "Created tmp/ directory for match recording")
	}

	paused := true

	for category := range config.Current.Templates {
		switch category {
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
				go process(config.Current.Templates[category][name], imgq[name])

				time.Sleep(time.Millisecond * 250) // .
			}
		}
	}

	go seconds(&paused)
	go balls(&paused)

	go func() {
		for {
			switch <-gui.Window.Actions {
			case gui.Start:
				notify.Feed(rgba.Green, "Starting...")

				server.Clear()
				team.Clear()
				stats.Clear()

				paused = false
			case gui.Stop:
				paused = true

				server.Clear()
				team.Clear()
				if config.Current.Stats {
					stats.Data()
				}

				notify.Feed(rgba.Green, "Stopping...")
			case gui.Record:
				config.Current.Record = !config.Current.Record

				switch config.Current.Record {
				case true:
					err := dev.New()
					if err != nil {
						kill(err)
					}

					notify.Feed(rgba.White, "Using tmp/ directory for recording match captures")

					err = config.Current.Save()
					if err != nil {
						kill(err)
					}
				case false:
					notify.Feed(rgba.White, "Closing any open match capture files in tmp/")

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
