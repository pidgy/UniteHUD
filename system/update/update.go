package update

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hashicorp/go-version"

	"github.com/pidgy/unitehud/app"
	"github.com/pidgy/unitehud/core/notify"
	"github.com/pidgy/unitehud/system/desktop"
	"github.com/pidgy/unitehud/system/desktop/clicked"
)

type query struct {
	News   []string `json:"news"`
	Latest string   `json:"latest"`
}

func Check() {
	notify.Debug("[Update] Validating %s", app.Version)

	b := &bytes.Buffer{}
	h, err := http.NewRequest(http.MethodGet, fmt.Sprintf("https://unitehud.dev/update.json?v=%s", app.VersionNoV), b)
	if err != nil {
		notify.Error("[Update] Failed to check for updates (%v)", err)
		return
	}
	h.Header.Set("User-Agent", fmt.Sprintf("UniteHUD-Updater/%s", app.VersionNoV))

	r, err := http.DefaultClient.Do(h)
	if err != nil {
		notify.Error("[Update] Failed to check for updates (%v)", err)
		return
	}
	defer r.Body.Close()

	q := query{}

	err = json.NewDecoder(r.Body).Decode(&q)
	if err != nil {
		notify.Error("[Update] Failed to read update file (%v)", err)
		return
	}

	if q.Latest == "" {
		notify.Warn("[Update] Failed to verify latest version")
		return
	}

	notify.Debug("[Update] Comparing %s against %s", q.Latest, app.Version)

	local, err := version.NewVersion(app.Version)
	if err != nil {
		notify.Error("[Update] Failed to parse global version number (%v)", err)
		return
	}

	remote, err := version.NewVersion(q.Latest)
	if err != nil {
		notify.Error("[Update] Failed to parse global version number (%v)", err)
		return
	}

	switch {
	case local.Equal(remote):
		notify.System("[Update] You are running the latest version of UniteHUD (%s)", local)
	case local.LessThan(remote):
		notify.System("[Update] %s is now available for download (http://unitehud.dev)", remote)

		desktop.Notification("UniteHUD %s", remote).
			Says("An update is available for UniteHUD").
			When(clicked.VisitWebsite).
			Send()
	case remote.LessThan(local):
		notify.System("[Update] You are running an unstable %s build", local)
	default:
		notify.Warn("[Update] Unable to validate version %s ", remote)
	}

	for _, n := range q.News {
		notify.Announce("[News] %s", n)
	}
}
