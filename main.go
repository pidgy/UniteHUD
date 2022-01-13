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
	"github.com/vova616/screenshot"
	"gocv.io/x/gocv"

	"github.com/pidgy/unitehud/dev"
	"github.com/pidgy/unitehud/pipe"
	"github.com/pidgy/unitehud/team"
	"github.com/pidgy/unitehud/window"
)

// windows
// cls && go build && unitehud.exe

var (
	socket *pipe.Pipe
	screen = configs["default"]
	mask   = gocv.NewMat()
	sigq   = make(chan os.Signal, 1)
)

var imgq = map[string]chan *image.RGBA{
	team.None.Name:   make(chan *image.RGBA, 1),
	team.Self.Name:   make(chan *image.RGBA, 4),
	team.Purple.Name: make(chan *image.RGBA, 1),
	team.Orange.Name: make(chan *image.RGBA, 1),
	team.Balls.Name:  make(chan *image.RGBA, 1),
}

var (
	record     = false
	acceptance = float32(0.91)
	addr       = ":17069"
)

func init() {
	flag.BoolVar(&record, "record", record, "record data such as images and logs for developer-specific debugging")
	flag.StringVar(&addr, "addr", addr, "http/websocket serve address")
	custom := flag.Bool("custom", false, "configure a customized screen capture or use the default 1920x1080 setting")
	avg := flag.Float64("match", float64(acceptance)*100, `0-100% certainty when processing score values`)
	level := flag.String("v", zerolog.LevelInfoValue, "log level (panic, fatal, error, warn, info, debug)")
	flag.Parse()

	log.Logger = zerolog.New(
		zerolog.ConsoleWriter{
			Out:        os.Stderr,
			TimeFormat: time.Stamp,
		},
	).With().Timestamp().Logger()

	acceptance = float32(*avg) / 100
	socket = pipe.New(addr)

	go signals()

	lvl, err := zerolog.ParseLevel(*level)
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	log.Logger = log.Logger.Level(lvl)

	if *custom {
		screen = configs["custom"]
	}

	load()
}

func capture(name string) {
<<<<<<< HEAD
=======
	if game == "ios" {
		rect = image.Rect(0, 0, 1920, 1080)
	}
>>>>>>> a33b30429e1355c4b5e9c29fd10b084abeb48013
	for {
		img, err := screenshot.CaptureRect(screen.scores)
		if err != nil {
			kill(err)
		}

		select {
		case imgq[name] <- img:
		default:
		}

		time.Sleep(team.Delay(name))
	}
}

func loop(t []template, imgq chan *image.RGBA) {
	runtime.LockOSThread()

	for img := range imgq {
		matrix, err := gocv.ImageToMatRGB(img)
		if err != nil {
			kill(err)
		}

		matches(matrix, img, t)

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

func matches(matrix gocv.Mat, img *image.RGBA, t []template) {
	results := make([]gocv.Mat, len(t))

	for i, template := range t {
		results[i] = gocv.NewMat()
		defer results[i].Close()

		gocv.MatchTemplate(matrix, template.Mat, &results[i], gocv.TmCcoeffNormed, mask)
	}

	for i, mat := range results {
		if mat.Empty() {
			log.Warn().Str("filename", t[i].file).Msg("empty result")
			continue
		}

		_, maxc, _, maxp := gocv.MinMaxLoc(mat)
		if maxc >= acceptance {
			match{
				Point:    maxp,
				template: t[i],
			}.process(matrix, img)
		}
	}
}

func seconds() {
	m := match{}

	for {
		img, err := screenshot.CaptureRect(screen.time)
		if err != nil {
			kill(err)
		}

		matrix, err := gocv.ImageToMatRGB(img)
		if err != nil {
			kill(err)
		}

		if m.time(matrix, img, screen.regularTime) == 0 && m.time(matrix, img, screen.finalStretch) == 0 {
			// Let's back off and not waste processing power.
			time.Sleep(time.Second * 5)
		}

		time.Sleep(team.Delay(team.Time.Name))
	}
}

func signals() {
	signal.Notify(sigq, syscall.SIGINT, syscall.SIGTERM)
	s := <-sigq

	window.Close()
	log.Info().Stringer("signal", s).Msg("closing...")
	os.Exit(1)
}

func main() {
	log.Info().
		Bool("record", record).
		Str("match", strconv.Itoa(int(acceptance*100))+"%").
		Str("addr", addr).Msg("unitehud")

	if record {
		err := dev.New()
		if err != nil {
			kill(err)
		}
	}

	for category := range templates {
		if category == "points" || category == "time" {
			continue
		}

		for name := range templates[category] {
			for i := 0; i < cap(imgq[name]); i++ {
				go capture(name)
				go loop(templates[category][name], imgq[name])
			}
		}
	}

	go seconds()

	err := window.Init()
	if err != nil {
		kill(err)
	}

	window.Write(window.Default, "Started Pokemon Unite HUD Server... listening on", addr)

	window.Show()
}
