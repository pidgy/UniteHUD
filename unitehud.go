//go:generate go-winres make --product-version=git-tag
package main

import (
	"fmt"
	"os"
	"os/signal"

	"github.com/pidgy/unitehud/avi/audio"
	"github.com/pidgy/unitehud/avi/video"
	"github.com/pidgy/unitehud/core/config"
	"github.com/pidgy/unitehud/core/detect"
	"github.com/pidgy/unitehud/core/global"
	"github.com/pidgy/unitehud/core/notify"
	"github.com/pidgy/unitehud/core/server"
	"github.com/pidgy/unitehud/core/state"
	"github.com/pidgy/unitehud/core/stats"
	"github.com/pidgy/unitehud/core/team"
	"github.com/pidgy/unitehud/gui/ui"
	"github.com/pidgy/unitehud/system/discord"
	"github.com/pidgy/unitehud/system/process"
	"github.com/pidgy/unitehud/system/save"
	"github.com/pidgy/unitehud/system/update"
)

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
		notify.Error("UniteHUD: Crashed (%v)", err)
	}

	report := make(chan bool)

	ui.UI.ToastYesNo(
		"Crashed",
		"UniteHUD has crashed. Open log directory?",
		ui.OnToastYes(
			func() {
				close(report)
			},
		),
		ui.OnToastNo(
			func() {
				defer close(report)

				save.Logs()
				err := save.OpenLogDirectory()
				if err != nil {
					notify.Error("UniteHUD: Failed to open log directory (%v)", err)
				}
			},
		),
	)

	<-report
}

func signals() {
	signal.Notify(sigq, os.Interrupt)
	<-sigq

	video.Close()
	audio.Close()
	ui.UI.Close()

	save.TemplateStatistics()

	os.Exit(0)
}

func main() {
	ui.New()
	defer ui.UI.Open()

	defer func() {
		if r := recover(); r != nil {
			kill(fmt.Errorf("%v", r))
		}
	}()

	err := process.Start()
	if err != nil {
		notify.SystemWarn("UniteHUD: Failed to stop previous process (%v)", err)
	}

	err = config.Load(config.Current.Profile)
	if err != nil {
		notify.Error("UniteHUD: Failed to load %s (%v)", config.Current.File(), err)
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
			case ui.Start:
				server.SetConfig(true)

				detect.Resume()

				notify.Clear()
				server.Clear()
				state.Clear()
				stats.Clear()
				team.Clear()

				server.SetStarted()

				notify.Announce("UniteHUD: Started %s", global.Title)
			case ui.Stop:
				detect.Pause()

				server.Clear()
				team.Clear()

				server.SetStopped()

				save.TemplateStatistics()

				notify.Announce("UniteHUD: Stopped %s", global.Title)
			case ui.Refresh:
				notify.Debug("UniteHUD: Action received (Refresh)")
			}
		}
	}()

	go signals()

	notify.Announce("UniteHUD: Initialized")
}
