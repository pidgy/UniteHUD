//go:generate go-winres make --product-version=git-tag

package main

import (
	"os"
	"os/signal"

	"github.com/pidgy/unitehud/config"
	"github.com/pidgy/unitehud/detect"
	"github.com/pidgy/unitehud/global"
	"github.com/pidgy/unitehud/gui"
	"github.com/pidgy/unitehud/gui/visual/title"
	"github.com/pidgy/unitehud/notify"
	"github.com/pidgy/unitehud/process"
	"github.com/pidgy/unitehud/save"
	"github.com/pidgy/unitehud/server"
	"github.com/pidgy/unitehud/state"
	"github.com/pidgy/unitehud/stats"
	"github.com/pidgy/unitehud/team"
	"github.com/pidgy/unitehud/update"
	"github.com/pidgy/unitehud/video"
	"github.com/pidgy/unitehud/video/window/electron"
)

// windows
// cls && go build && unitehud.exe
// go build -ldflags="-H windowsgui"
var sigq = make(chan os.Signal, 1)

func init() {
	notify.System("Initializing...")
}

func kill(errs ...error) {
	defer close(sigq)

	if len(errs) > 0 {
		config.Current.Report(errs[0].Error())

		err := config.Current.Save()
		if err != nil {
			notify.Error("Failed to save crash log (%v)", err)
		}
	}

	for _, err := range errs {
		notify.Error("Force Shutdown: %v", err)
	}

	report := make(chan bool)

	gui.UI.ToastCrash(
		"UniteHUD has encountered an unrecoverable error",
		func() {
			close(report)
		},
		func() {
			save.Logs()

			err := save.OpenLogDirectory()
			if err != nil {
				println(err.Error())
			}
		},
	)
	<-report
}

func signals() {
	signal.Notify(sigq, os.Interrupt)
	<-sigq

	gui.UI.Close()
	video.Close()
	electron.Close()

	os.Exit(0)
}

func main() {
	gui.New()
	defer gui.UI.Open()

	err := process.Replace()
	if err != nil {
		notify.Error("Failed to kill previous UniteHUD (%v)", err)
	}

	err = config.Load(config.Current.Profile)
	if err != nil {
		kill(err)
	}

	err = video.Open()
	if err != nil {
		notify.Error("Failed to open Video Capture Device (%v)", err)
	}

	err = server.Listen()
	if err != nil {
		notify.Error("Failed to start UniteHUD server (%v)", err)
	}

	err = electron.Open()
	if err != nil {
		notify.Error("Failed to start UniteHUD overlay (%v)", err)
	}

	notify.System("Debug mode: %t", global.DebugMode)
	notify.System("Server address: \"%s\"", server.Address)
	notify.System("Recording: %t", config.Current.Record)
	notify.System("Profile: %s", config.Current.Profile)
	notify.System("Assets: %s", config.Current.Assets())
	notify.System("Default match threshold: %.0f%%", config.Current.Acceptance*100)

	go detect.Clock()
	go detect.Energy()
	go detect.PressButtonToScore()
	go detect.Preview()
	go detect.Defeated()
	go detect.KOs()
	go detect.Objectives()
	go detect.States()
	go detect.Scores(team.Purple.Name)
	go detect.Scores(team.Orange.Name)
	go detect.Scores(team.First.Name)
	go update.Check()

	go func() {
		lastWindow := config.Current.Window

		for action := range gui.UI.Actions {
			switch action {
			case gui.Closing:
				close(sigq)
			case gui.Config:
				server.SetConfig(true)
				fallthrough
			case gui.Start:
				detect.Resume()

				notify.Announce("Starting %s...", title.Default)

				notify.Clear()
				server.Clear()
				team.Clear()
				stats.Clear()
				state.Clear()

				notify.Announce("Started %s", title.Default)

				server.SetStarted()
			case gui.Stop:
				detect.Pause()

				notify.Announce("Stopping %s...", title.Default)

				// Wait for the capture routines to go idle.
				// time.Sleep(time.Second * 2)

				notify.Announce("Stopped %s", title.Default)

				server.Clear()
				team.Clear()

				server.SetStopped()

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

				notify.System("%s template match results in %s", str, save.Directory)

				if config.Current.Record {
					notify.System("Using \"%s\" directory for recording data", save.Directory)

					err = config.Current.Save()
					if err != nil {
						kill(err)
					}

					err := save.Open()
					if err != nil {
						notify.Error("Failed to open \"%s\" (%v)", save.Directory, err)
					}
				} else {
					notify.System("Closing open files in %s", save.Directory)
				}
			case gui.Log:
				save.Logs()

				err := save.Open()
				if err != nil {
					notify.Error("Failed to open \"%s\" (%v)", save.Directory, err)
				}
			case gui.Open:
				notify.System("Opening \"%s\"", save.Directory)

				err := save.Open()
				if err != nil {
					notify.Error("Failed to open \"%s\" (%v)", save.Directory, err)
				}
			case gui.Refresh:
				notify.Debug("GUI sent \"Refresh\"")

				err := video.Open()
				if err != nil {
					notify.Error("Failed to open Video Capture Device (%v)", err)
				}

				if lastWindow != config.Current.Window {
					notify.System("Capture window set to \"%s\"", lastWindow)
				}
			}
		}
	}()

	go signals()

	notify.System("Initialized")
}
