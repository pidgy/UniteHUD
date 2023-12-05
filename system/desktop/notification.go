package desktop

import (
	"fmt"

	"git.sr.ht/~jackmordaunt/go-toast"

	"github.com/pidgy/unitehud/core/config"
	"github.com/pidgy/unitehud/core/notify"
	"github.com/pidgy/unitehud/system/desktop/clicked"
)

type Factory struct {
	toast toast.Notification
}

func Notification(format string, args ...interface{}) *Factory {
	a := toast.Mail
	if config.Current.Advanced.Notifications.Muted {
		a = toast.Silent
	}
	return &Factory{
		toast: toast.Notification{
			AppID:               "UniteHUD",
			Title:               fmt.Sprintf(format, args...),
			Body:                "Notification",
			Icon:                config.Current.AssetIcon("icon256x256.png"),
			ActivationArguments: "https://unitehud.dev",
			Audio:               a,
		},
	}
}

func (n *Factory) Says(format string, args ...interface{}) *Factory {
	n.toast.Body = fmt.Sprintf(format, args...)
	return n
}

func (n *Factory) Send() {
	if config.Current.Advanced.Notifications.Disabled.All {
		return
	}

	err := n.toast.Push()
	if err != nil {
		notify.SystemWarn("🔔 Failed to send desktop notification (%v)", err)
	}
}

func (n *Factory) When(clicked ...clicked.Action) *Factory {
	for _, clicked := range clicked {
		n.toast.Actions = append(n.toast.Actions,
			clicked.Then(),
		)
	}
	return n
}
