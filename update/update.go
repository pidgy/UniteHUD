package update

import (
	"encoding/json"
	"net/http"

	"github.com/pidgy/unitehud/global"
	"github.com/pidgy/unitehud/notify"
)

var (
	Available = false
)

type query struct {
	Message string `json:"message"`
	Version string `json:"version"`
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

	Available = (q.Version != global.Version)

	if Available {
		notify.Announce("UniteHUD %s is now available for download at UniteHUD.dev", q.Version)
	} else {
		notify.System("Running the latest version of UniteHUD (%s)", global.Version)
	}

	if q.Message != "" {
		notify.Announce("[UniteHUD.dev] %s", q.Message)
	}
}
