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

	v1, err := version.NewVersion(app.Version)
	if err != nil {
		notify.Error("[Update] Failed to parse global version number (%v)", err)
		return
	}

	v2, err := version.NewVersion(q.Latest)
	if err != nil {
		notify.Error("[Update] Failed to parse global version number (%v)", err)
		return
	}

	switch {
	case v1.LessThan(v2):
		notify.System("[Update] %s is now available for download (http://unitehud.dev)", q.Latest)

		desktop.Notification("UniteHUD %s", q.Latest).
			Says("An update is available for UniteHUD").
			When(clicked.VisitWebsite).
			Send()
	case v2.LessThan(v1):
		notify.System("[Update] You are running an unstable %s build", app.Version)
	case v1.Equal(v2):
		notify.System("[Update] You are running the latest version of UniteHUD (%s)", app.Version)
	default:
		notify.Warn("[Update] Unable to validate version %s ", q.Latest)
	}

	for _, n := range q.News {
		notify.Announce("[News] %s", n)
	}
}
