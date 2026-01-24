//go:generate go-winres make --product-version=git-tag
package main

import (
	"github.com/pidgy/unitehud/core/config"
	"github.com/pidgy/unitehud/core/detect"
	"github.com/pidgy/unitehud/core/notify"
	"github.com/pidgy/unitehud/core/rgba/nrgba"
	"github.com/pidgy/unitehud/core/server"
	"github.com/pidgy/unitehud/core/stats"
	"github.com/pidgy/unitehud/core/team"
	"github.com/pidgy/unitehud/exe"
	"github.com/pidgy/unitehud/gui/ui"
	"github.com/pidgy/unitehud/media/audio"
	"github.com/pidgy/unitehud/media/video"
	"github.com/pidgy/unitehud/system/discord"
	"github.com/pidgy/unitehud/system/ini"
	"github.com/pidgy/unitehud/system/process"
	"github.com/pidgy/unitehud/system/save"
	"github.com/pidgy/unitehud/system/tray"
	"github.com/pidgy/unitehud/system/update"
)

func init() {
	notify.Announce("[UniteHUD] Initializing...")

	err := ini.Default()
	if err != nil {
		notify.Error("<ini:f:set> default locale (%v)", err)
	}

	discord.InfoLog = func(format string, a ...any) { notify.Feed(nrgba.Discord, format, a...) }
	discord.WarnLog = notify.Warn
	discord.ErrorLog = notify.Error
}

func close() {
	exe.WaitForSignal()

	notify.Announce("[UniteHUD] Closing...")

	video.Close()
	audio.Close()
	ui.Close()
	tray.Close()

	err := save.Logs(notify.FeedStrings(), stats.Lines(), stats.Counts())
	if err != nil {
		notify.Warn("[UniteHUD] <ini:f:save> logs (%v)", err)
	}

	exe.Exit()
}

func main() {
	defer ui.New().OnClose(exe.Close).Open()

	err := process.Open()
	if err != nil {
		notify.Warn("[UniteHUD] <ini:f:stop> <ini:i:previous_process> (%v)", err)
	}

	err = config.Open()
	if err != nil {
		notify.Warn("[UniteHUD] <ini:f:load> %s (%v)", config.Current.File(), err)
	}

	err = ini.Open(config.Current.Advanced.Locale)
	if err != nil {
		notify.Warn("[UniteHUD] <ini:f:set> %s (%v)", config.Current.Advanced.Locale, err)
	}

	err = video.Open()
	if err != nil {
		notify.Warn("[UniteHUD] <ini:f:open> video (%v)", err)
	}

	err = audio.Open()
	if err != nil {
		notify.Warn("[UniteHUD] <ini:f:open> audio session (%v)", err)
	}

	err = server.Open()
	if err != nil {
		notify.Warn("[UniteHUD] <ini:f:start> server (%v)", err)
	}

	err = tray.Open()
	if err != nil {
		notify.Warn("[UniteHUD] <ini:f:open> system tray (%v)", err)
	}

	err = discord.Open()
	if err != nil {
		notify.Warn("[UniteHUD] <ini:f:open> Discord RPC (%v)", err)
	}

	q, err := update.Check()
	if err != nil {
		notify.Warn("[UniteHUD] <ini:f:check> UniteHUD updates (%v)", err)
	}
	for _, n := range q.News {
		notify.Feed(nrgba.Highlight, "[UniteHUD] [News] %s", n)
	}
	notify.Feed(nrgba.Highlight, "[UniteHUD] [Update] You are running %s, %s version of UniteHUD (Latest: %s)", exe.TitleAndVersion, q.Version, q.Latest)

	notify.Debug("[UniteHUD] Server Address (%s)", server.Address)
	notify.Debug("[UniteHUD] Recording (%t)", config.Current.Record)
	notify.Debug("[UniteHUD] Platform (%s)", config.Current.Gaming.Device)
	notify.Debug("[UniteHUD] Assets (%s)", config.Current.Assets())
	notify.Debug("[UniteHUD] Match Threshold: (%.0f%%)", config.Current.Acceptance*100)

	go detect.Clock()
	go detect.Energy()
	go detect.Preview()
	go detect.Defeated()
	go detect.Objectives()
	go detect.States()
	go detect.Scores(team.Self.Name)
	go detect.Scores(team.Purple.Name)
	go detect.Scores(team.Orange.Name)
	go detect.Scores(team.First.Name)

	go close()

	notify.Announce("[UniteHUD] Initialized")
}
