//go:generate go-winres make --product-version=git-tag

package main

import (
	"os"
	"os/signal"

	"github.com/pidgy/unitehud/core/config"
	"github.com/pidgy/unitehud/core/detect"
	"github.com/pidgy/unitehud/core/global"
	"github.com/pidgy/unitehud/core/notify"
	"github.com/pidgy/unitehud/core/server"
	"github.com/pidgy/unitehud/core/state"
	"github.com/pidgy/unitehud/core/stats"
	"github.com/pidgy/unitehud/core/team"
	"github.com/pidgy/unitehud/gui/ui"
	"github.com/pidgy/unitehud/gui/visual/electron"
	"github.com/pidgy/unitehud/media/audio"
	"github.com/pidgy/unitehud/media/video"
	"github.com/pidgy/unitehud/system/discord"
	"github.com/pidgy/unitehud/system/process"
	"github.com/pidgy/unitehud/system/save"
	"github.com/pidgy/unitehud/system/update"
)

// windows
// cls && go build && unitehud.exe
// go build -ldflags="-H windowsgui"
var sigq = make(chan os.Signal, 1)

func init() {
	notify.Announce("UniteHUD: Initializing...")
}

func kill(errs ...error) {
	defer close(sigq)

	if len(errs) > 0 {
		config.Current.Report(errs[0].Error())

		err := config.Current.Save()
		if err != nil {
			notify.Error("UniteHUD: Failed to save crash log (%v)", err)
		}
	}

	for _, err := range errs {
		notify.Error("UniteHUD: Force Shutdown (%v)", err)
	}

	report := make(chan bool)

	ui.UI.ToastCrash(
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

	ui.UI.Close()
	video.Close()
	electron.Close()
	audio.Close()

	os.Exit(0)
}

func main() {
	ui.New()
	defer ui.UI.Open()

	err := process.Replace()
	if err != nil {
		notify.SystemWarn("UniteHUD: Failed to stop previous process (%v)", err)
	}

	err = config.Load(config.Current.Profile)
	if err != nil {
		kill(err)
	}

	err = video.Open()
	if err != nil {
		notify.Error("UniteHUD: Failed to open video (%v)", err)
	}

	err = audio.Open()
	if err != nil {
		notify.Error("UniteHUD: Failed to open audio session (%v)", err)
	}

	err = server.Listen()
	if err != nil {
		notify.Error("UniteHUD: Failed to start server (%v)", err)
	}

	go electron.Open()
	go discord.Connect()

	notify.Debug("UniteHUD: Server Address (%s)", server.Address)
	notify.Debug("UniteHUD: Recording (%t)", config.Current.Record)
	notify.Debug("UniteHUD: Profile (%s)", config.Current.Profile)
	notify.Debug("UniteHUD: Assets (%s)", config.Current.Assets())
	notify.Debug("UniteHUD: Match Threshold: (%.0f%%)", config.Current.Acceptance*100)

	go detect.Clock()
	go detect.Energy()
	go detect.PressButtonToScore()
	go detect.Preview()
	go detect.Defeated()
	go detect.Objectives()
	go detect.States()
	go detect.Scores(team.Purple.Name)
	go detect.Scores(team.Orange.Name)
	go detect.Scores(team.First.Name)
	go update.Check()

	go func() {
		for action := range ui.UI.Actions {
			switch action {
			case ui.Closing:
				signal.Reset()
				close(sigq)
			case ui.Config:
				server.SetConfig(true)
				fallthrough
			case ui.Start:
				detect.Resume()

				notify.Announce("UniteHUD: Starting %s...", global.Title)

				notify.Clear()
				server.Clear()
				team.Clear()
				stats.Clear()
				state.Clear()

				notify.Announce("UniteHUD: Started %s", global.Title)

				server.SetStarted()
			case ui.Stop:
				detect.Pause()

				notify.Announce("UniteHUD: Stopping %s...", global.Title)

				// Wait for the capture routines to go idle.
				// time.Sleep(time.Second * 2)

				notify.Announce("UniteHUD: Stopped %s", global.Title)

				server.Clear()
				team.Clear()

				server.SetStopped()

				if !config.Current.Record {
					continue
				}

				fallthrough
			case ui.Record:
				config.Current.Record = !config.Current.Record

				str := "Closing"
				if config.Current.Record {
					str = "Recording"
				}

				notify.System("UniteHUD: %s template match results in %s", str, save.Directory)

				if config.Current.Record {
					notify.System("UniteHUD: Record directory set to \"%s\"", save.Directory)

					err = config.Current.Save()
					if err != nil {
						notify.Error("Failed to save UniteHUD configuration (%v)", err)
					}

					err := save.Open()
					if err != nil {
						notify.Error("UniteHUD: Failed to open \"%s\" (%v)", save.Directory, err)
					}
				} else {
					notify.System("UniteHUD: Closing open files in %s", save.Directory)
				}
			case ui.Log:
				save.Logs()

				err := save.Open()
				if err != nil {
					notify.Error("UniteHUD: Failed to open \"%s\" (%v)", save.Directory, err)
				}
			case ui.Open:
				notify.System("UniteHUD: Opening \"%s\"", save.Directory)

				err := save.Open()
				if err != nil {
					notify.Error("UniteHUD: Failed to open \"%s\" (%v)", save.Directory, err)
				}
			case ui.Refresh:
				notify.Debug("UniteHUD: Action received (Refresh)")
			}
		}
	}()

	go signals()

	notify.Announce("UniteHUD: Initialized")
}
