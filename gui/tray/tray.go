package tray

import (
	"github.com/rupor-github/win-gpg-agent/systray"
	"github.com/skratchdot/open-golang/open"

	"github.com/pidgy/unitehud/config"
	"github.com/pidgy/unitehud/discord"
	"github.com/pidgy/unitehud/global"
	"github.com/pidgy/unitehud/img"
	"github.com/pidgy/unitehud/notify"
)

var hidden = false

type menu struct {
	title *systray.MenuItem

	settings struct {
		opens

		discord struct {
			opens
			toggle
		}

		notifications struct {
			opens

			all,
			muted,
			updates,
			matchStarted,
			matchStopped toggle
		}

		tray struct {
			opens
			toggle
		}
	}

	website toggle
	exit    toggle
}

type opens struct {
	*systray.MenuItem
}

type toggle struct {
	*systray.MenuItem
	event func()
}

func Close() {
	notify.Debug("Tray: Closing")
	systray.Quit()
}

func Open(close func()) {
	notify.Debug("Tray: Opening")

	go systray.Run(ready, exit(close), func(se systray.SessionEvent) {
		notify.System("Tray: Session Event \"%s\"", se)
	})
}

func exit(close func()) func() {
	return func() {
		if !hidden {
			close()
		}
	}
}

func ready() {
	m := menu{}

	m.addTitle()
	m.addSettings()
	m.addWebsite()
	m.addExit()

	for {
		select {
		case <-m.settings.tray.toggle.ClickedCh:
			m.settings.tray.toggle.event()
			return
		case <-m.website.ClickedCh:
			m.website.event()
		case <-m.settings.discord.toggle.ClickedCh:
			m.settings.discord.toggle.event()
		case <-m.settings.notifications.all.ClickedCh:
			m.settings.notifications.all.event()
		case <-m.settings.notifications.muted.ClickedCh:
			m.settings.notifications.muted.event()
		case <-m.settings.notifications.updates.ClickedCh:
			m.settings.notifications.updates.event()
		case <-m.settings.notifications.matchStarted.ClickedCh:
			m.settings.notifications.matchStarted.event()
		case <-m.settings.notifications.matchStopped.ClickedCh:
			m.settings.notifications.matchStopped.event()
		case <-m.exit.ClickedCh:
			m.exit.event()
		}
	}
}

func (m *menu) addExit() {
	notify.Debug("Tray: Adding Exit")

	systray.AddSeparator()

	m.exit = toggle{
		MenuItem: systray.AddMenuItem("Exit", "Close this application"),
		event:    Close,
	}
	// m.exit.SetIcon(img.IconBytes("exit-bg.ico"))
}

func (m *menu) addSettings() {
	notify.Debug("Tray: Adding Settings")

	systray.AddSeparator()

	m.settings.opens = opens{MenuItem: systray.AddMenuItem("Settings", "Change settings")}
	m.settings.opens.SetIcon(img.IconBytes("settings.ico"))

	m.settings.discord.opens = opens{MenuItem: m.settings.AddSubMenuItem("Discord", "Change Discord settings")}
	m.settings.discord.toggle = toggle{
		MenuItem: m.settings.discord.opens.AddSubMenuItemCheckbox("Disabled", "Toggle Discord RPC", !config.Current.Advanced.Discord.Disabled),
		event: func() {
			config.Current.Advanced.Discord.Disabled = !config.Current.Advanced.Discord.Disabled
			err := config.Current.Save()
			if err != nil {
				notify.Warn("Tray: Failed to save UniteHUD configuration (%v)", err)
			}

			m.settings.discord.toggle.Check()
			m.settings.discord.toggle.SetTitle("Enabled")

			if config.Current.Advanced.Discord.Disabled {
				defer discord.Disconnect()

				m.settings.discord.toggle.Uncheck()
				m.settings.discord.toggle.SetTitle("Disabled")
			}
		},
	}
	if !config.Current.Advanced.Discord.Disabled {
		m.settings.discord.toggle.Check()
		m.settings.discord.toggle.SetTitle("Enabled")
	}
	m.settings.discord.opens.SetIcon(img.IconBytes("discord-f2.ico"))

	m.settings.notifications.opens = opens{m.settings.AddSubMenuItem("Notifications", "Change Desktop notification settings")}
	m.settings.notifications.opens.SetIcon(img.IconBytes("notifications-bg.ico"))
	m.settings.notifications.all = toggle{
		MenuItem: m.settings.notifications.AddSubMenuItemCheckbox(
			"Disabled", "Toggle Desktop notifications", !config.Current.Advanced.Notifications.Disabled.All,
		),
		event: func() {
			config.Current.Advanced.Notifications.Disabled.All = !config.Current.Advanced.Notifications.Disabled.All
			err := config.Current.Save()
			if err != nil {
				notify.Warn("Tray: Failed to save UniteHUD configuration (%v)", err)
			}

			m.settings.notifications.all.Uncheck()
			if config.Current.Advanced.Notifications.Disabled.All {
				m.settings.notifications.all.Check()
			}
		},
	}
	if config.Current.Advanced.Notifications.Disabled.All {
		m.settings.notifications.all.Check()
	}

	m.settings.notifications.muted = toggle{
		MenuItem: m.settings.notifications.AddSubMenuItemCheckbox(
			"Muted", "Toggle Desktop notification volume", !config.Current.Advanced.Notifications.Muted,
		),
		event: func() {
			config.Current.Advanced.Notifications.Muted = !config.Current.Advanced.Notifications.Muted
			err := config.Current.Save()
			if err != nil {
				notify.Warn("Tray: Failed to save UniteHUD configuration (%v)", err)
			}

			m.settings.notifications.muted.Uncheck()
			if config.Current.Advanced.Notifications.Muted {
				m.settings.notifications.muted.Check()
			}
		},
	}
	if config.Current.Advanced.Notifications.Muted {
		m.settings.notifications.muted.Check()
	}

	m.settings.notifications.updates = toggle{
		MenuItem: m.settings.notifications.AddSubMenuItemCheckbox(
			"Updates", "Toggle notifcations for newer versions of UniteHUD", !config.Current.Advanced.Notifications.Disabled.Updates,
		),
		event: func() {
			config.Current.Advanced.Notifications.Disabled.Updates = !config.Current.Advanced.Notifications.Disabled.Updates
			err := config.Current.Save()
			if err != nil {
				notify.Warn("Tray: Failed to save UniteHUD configuration (%v)", err)
			}

			m.settings.notifications.updates.Check()
			if config.Current.Advanced.Notifications.Disabled.Updates {
				m.settings.notifications.updates.Uncheck()
			}
		},
	}
	if !config.Current.Advanced.Notifications.Disabled.Updates {
		m.settings.notifications.updates.Check()
	}

	m.settings.notifications.matchStarted = toggle{
		MenuItem: m.settings.notifications.AddSubMenuItemCheckbox(
			"Match Started", "Toggle notifcations for when a match is starting", !config.Current.Advanced.Notifications.Disabled.MatchStarting,
		),
		event: func() {
			config.Current.Advanced.Notifications.Disabled.MatchStarting = !config.Current.Advanced.Notifications.Disabled.MatchStarting
			err := config.Current.Save()
			if err != nil {
				notify.Warn("Tray: Failed to save UniteHUD configuration (%v)", err)
			}

			m.settings.notifications.matchStarted.Check()
			if config.Current.Advanced.Notifications.Disabled.MatchStarting {
				m.settings.notifications.matchStarted.Uncheck()
			}
		},
	}
	if !config.Current.Advanced.Notifications.Disabled.MatchStarting {
		m.settings.notifications.matchStarted.Check()
	}

	m.settings.notifications.matchStopped = toggle{
		MenuItem: m.settings.notifications.AddSubMenuItemCheckbox(
			"Match Ended", "Toggle notifcations for when a match is ending", !config.Current.Advanced.Notifications.Disabled.MatchStarting,
		),
		event: func() {
			config.Current.Advanced.Notifications.Disabled.MatchStopped = !config.Current.Advanced.Notifications.Disabled.MatchStopped
			err := config.Current.Save()
			if err != nil {
				notify.Warn("Tray: Failed to save UniteHUD configuration (%v)", err)
			}

			m.settings.notifications.matchStopped.Check()

			if config.Current.Advanced.Notifications.Disabled.MatchStopped {
				m.settings.notifications.matchStopped.Uncheck()
			}
		},
	}
	if !config.Current.Advanced.Notifications.Disabled.MatchStopped {
		m.settings.notifications.matchStopped.Check()
	}

	m.settings.tray.opens = opens{m.settings.AddSubMenuItem("System Tray", "Modify visibilty of this tray")}
	m.settings.tray.toggle = toggle{
		MenuItem: m.settings.tray.opens.AddSubMenuItem(
			"Hide", "Hide this Tray",
		),
		event: func() {
			hidden = true
			systray.Quit()
		},
	}
}

func (m *menu) addTitle() {
	notify.Debug("Tray: Adding Title")

	systray.SetIcon(img.IconBytes("icon.ico"))
	systray.SetTitle(global.Title)
	systray.SetTooltip(global.TitleVersion)

	m.title = systray.AddMenuItem(global.TitleVersion, "")
	m.title.SetIcon(img.IconBytes("icon-bg.ico"))
	m.title.Disable()
}

func (m *menu) addWebsite() {
	notify.Debug("Tray: Adding Website")

	systray.AddSeparator()

	m.website = toggle{
		MenuItem: systray.AddMenuItem("Website", "Browse to unitehud.dev"),
		event: func() {
			err := open.Run("https://unitehud.dev")
			if err != nil {
				notify.Error("Tray: Failed to open unitehud.dev (%v)", err)
			}
		},
	}
	m.website.SetIcon(img.IconBytes("website-bg.ico"))
}
