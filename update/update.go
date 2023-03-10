package update

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/pidgy/unitehud/global"
	"github.com/pidgy/unitehud/notify"
)

var (
	Available = false
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

	Available = q.Latest != global.Version && strings.Contains(q.Latest, "beta") == strings.Contains(global.Version, "beta")

	if Available {
		notify.Announce("UniteHUD %s is now available for download at UniteHUD.dev", q.Latest)
	} else {
		notify.System("Running the latest version of UniteHUD (%s)", global.Version)
	}

	for _, n := range q.News {
		notify.Announce("[UniteHUD.dev] %s", n)
	}
}
