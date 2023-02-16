//go:generate go-winres make --product-version=git-tag

package main

import (
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/diode"
	"github.com/rs/zerolog/log"

	"github.com/pidgy/unitehud/config"
	"github.com/pidgy/unitehud/debug"
	"github.com/pidgy/unitehud/detect"
	"github.com/pidgy/unitehud/global"
	"github.com/pidgy/unitehud/gui"
	"github.com/pidgy/unitehud/notify"
	"github.com/pidgy/unitehud/process"
	"github.com/pidgy/unitehud/server"
	"github.com/pidgy/unitehud/state"
	"github.com/pidgy/unitehud/stats"
	"github.com/pidgy/unitehud/team"
	"github.com/pidgy/unitehud/video"
)

// windows
// cls && go build && unitehud.exe
// go build -ldflags="-H windowsgui"
var sigq = make(chan os.Signal, 1)

func init() {
	notify.System("Initializing...")

	log.Logger = zerolog.New(
		diode.NewWriter(zerolog.ConsoleWriter{
			Out:        os.Stderr,
			TimeFormat: time.Stamp,
		}, 4096, time.Nanosecond, func(missed int) {
			println("diode is falling behind")
		})).With().Timestamp().Logger().Level(zerolog.DebugLevel)
}

func kill(errs ...error) {
	if len(errs) > 0 {
		config.Current.Report(errs[0].Error())
	}

	for _, err := range errs {
		log.Err(err).Msg(gui.Title())
	}

	time.Sleep(time.Second)

	sig := os.Kill
	if len(errs) == 0 {
		sig = os.Interrupt
	}
	sigq <- sig
}

func signals() {
	signal.Notify(sigq, syscall.SIGINT, syscall.SIGTERM)
	s := <-sigq

	log.Info().Stringer("signal", s).Msg("closing...")

	detect.Close()

	os.Exit(1)
}

func main() {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	go signals()

	err := process.Replace()
	if err != nil {
		notify.Error("Failed to kill previous UniteHUD (%v)", err)
	}

	err = config.Load(config.Current.Profile)
	if err != nil {
		kill(err)
	}

	err = video.Load()
	if err != nil {
		notify.Error("Failed to load video input (%v)", err)
	}

	err = server.Listen()
	if err != nil {
		notify.Error("Failed to start UniteHUD server (%v)", err)
	}

	log.Info().
		Bool("record", config.Current.Record).
		Str("assets", config.Current.Assets()).
		Str("profile", config.Current.Profile).
		Msg("unitehud")

	notify.System("Debug Mode: %t", global.DebugMode)
	notify.System("Server address: \"%s\"", server.Address)
	notify.System("Recording: %t", config.Current.Record)
	notify.System("Profile: %s", config.Current.Profile)
	notify.System("Assets: %s", config.Current.Assets())
	notify.System("Match Threshold: %.0f%%", config.Current.Acceptance*100)

	go detect.Clock()
	// go detect.Crash()
	go detect.Energy()
	go detect.Defeated()
	go detect.KOs()
	go detect.Minimap()
	go detect.Objectives()
	go detect.PressButtonToScore()
	go detect.Preview()
	go detect.States()
	go detect.Window()
	go detect.Scores(team.Purple.Name)
	go detect.Scores(team.Orange.Name)
	go detect.Scores(team.First.Name)

	gui.New()
	defer gui.Window.Open()

	go func() {
		lastWindow := ""

		for action := range gui.Window.Actions {
			switch action {
			case gui.Closing:
				detect.Close()
				return
			case gui.Config:
				server.SetConfig(true)
				fallthrough
			case gui.Start:
				if !detect.Stopped {
					continue
				}

				notify.Announce("Starting %s...", gui.Title)

				notify.Clear()
				server.Clear()
				team.Clear()
				stats.Clear()
				state.Clear()

				detect.Stopped = false

				notify.Announce("Started %s", gui.Title)

				server.SetStarted()
				state.Add(state.ServerStarted, server.Clock(), -1)
			case gui.Stop:
				if detect.Stopped {
					continue
				}
				detect.Stopped = true

				notify.Denounce("Stopping %s...", gui.Title)

				// Wait for the capture routines to go idle.
				// time.Sleep(time.Second * 2)

				notify.Denounce("Stopped %s", gui.Title)

				server.Clear()
				team.Clear()

				server.SetStopped()
				state.Add(state.ServerStopped, server.Clock(), -1)

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
				was := detect.Stopped
				detect.Stopped = true

				err := config.Load(config.Current.Profile)
				if err != nil {
					notify.Error("Failed to reload config (%v)", err)
					continue
				}

				detect.Stopped = was

			}
		}
	}()

	notify.System("Launched")
}
