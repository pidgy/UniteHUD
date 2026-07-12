package desktop

// TODO: Add go style comments that reflect the purpose of each type, function, var, and const. Then remove this comment.

import (
	"fmt"

	"git.sr.ht/~jackmordaunt/go-toast"

	"github.com/pidgy/unitehud/core/config"
)

type Factory struct {
	toast  toast.Notification
	errors func(string, ...any)
}

var (
	OpenUniteHUD = toast.Action{
		Type:      toast.Foreground,
		Content:   "Open",
		Arguments: "",
	}

	ViewDetails = toast.Action{
		Type:      toast.Foreground,
		Content:   "View Details",
		Arguments: "UniteHUD.exe",
	}

	VisitWebsite = toast.Action{
		Type:      toast.Protocol,
		Content:   "Download",
		Arguments: "https://unitehud.dev",
	}
)

func (f *Factory) Logs(fn func(string, ...any)) *Factory {
	f.errors = fn
	return f
}

func (f *Factory) Says(format string, args ...any) *Factory {
	f.toast.Body = fmt.Sprintf(format, args...)
	return f
}

func (f *Factory) Send() {
	if config.Current.Advanced.Notifications.Disabled.All {
		return
	}

	err := f.toast.Push()
	if err != nil {
		if f.errors != nil {
			f.errors("[Desktop] Failed to send notification :%v", err)
		}
	}
}

func (f *Factory) When(clicked ...toast.Action) *Factory {
	f.toast.Actions = append(f.toast.Actions, clicked...)
	return f
}

func Notification(format string, args ...any) *Factory {
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
