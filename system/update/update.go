package update

import (
	"encoding/json"
	"net/http"

	"github.com/hashicorp/go-version"

	"github.com/pidgy/unitehud/core/global"
	"github.com/pidgy/unitehud/core/notify"
	"github.com/pidgy/unitehud/system/desktop"
	"github.com/pidgy/unitehud/system/desktop/clicked"
)

type query struct {
	News   []string `json:"news"`
	Latest string   `json:"latest"`
}

func Check() {
	notify.Debug("📡 Validating %s", global.Version)

	r, err := http.Get("https://unitehud.dev/update.json")
	if err != nil {
		notify.Error("Failed to check for updates (%v)", err)
		return
	}
	defer r.Body.Close()

	q := query{}
	err = json.NewDecoder(r.Body).Decode(&q)
	if err != nil {
		notify.Error("📡 Failed to read update file (%v)", err)
		return
	}

	if q.Latest == "" {
		notify.SystemWarn("📡 Failed to verify latest version")
		return
	}

	notify.Debug("📡 Comparing %s against %s", q.Latest, global.Version)

	v1, err := version.NewVersion(global.Version)
	if err != nil {
		notify.Error("📡 Failed to parse global version number (%v)", err)
		return
	}

	v2, err := version.NewVersion(q.Latest)
	if err != nil {
		notify.Error("📡 Failed to parse global version number (%v)", err)
		return
	}

	switch {
	case v1.LessThan(v2):
		notify.System("📡 %s is available for download (http://unitehud.dev)", q.Latest)

		desktop.Notification("%s Update", q.Latest).
			Says("An update is available for UniteHUD").
			When(clicked.VisitWebsite).
			Send()
	case v2.LessThan(v1):
		notify.System("📡 You are running an unstable %s build", global.Version)
	case v1.Equal(v2):
		notify.System("📡 Running latest (%s)", global.Version)
	default:
		notify.Warn("📡 Unable to validate version %s ", q.Latest)
	}

	for _, n := range q.News {
		notify.Announce("📡 unitehud.dev: %s", n)
	}
}
