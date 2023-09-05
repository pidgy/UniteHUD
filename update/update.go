package update

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/pidgy/unitehud/desktop"
	"github.com/pidgy/unitehud/desktop/clicked"
	"github.com/pidgy/unitehud/global"
	"github.com/pidgy/unitehud/notify"
)

type query struct {
	News   []string `json:"news"`
	Latest string   `json:"latest"`
}

func Check() {
	notify.System("Checking for updates...")

	r, err := http.Get("https://unitehud.dev/update.json")
	if err != nil {
		notify.Error("Failed to check for updates (%v)", err)
		return
	}
	defer r.Body.Close()

	q := query{}
	err = json.NewDecoder(r.Body).Decode(&q)
	if err != nil {
		notify.Error("Failed to read update file (%v)", err)
		return
	}

	if q.Latest == "" {
		notify.SystemWarn("Failed to verify latest version")
		return
	}

	available := q.Latest != global.Version && strings.Contains(q.Latest, "beta") == strings.Contains(global.Version, "beta")

	if available {
		desktop.Notification("%s Update", q.Latest).
			Says("An update is available for UniteHUD").
			When(clicked.VisitWebsite).
			Send()
	} else {
		notify.System("Running the latest version of UniteHUD (%s)", global.Version)
	}

	for _, n := range q.News {
		notify.Announce("[UniteHUD.dev] %s", n)
	}
}
