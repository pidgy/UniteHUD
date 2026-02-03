package update

// TODO: Add go style comments that reflect the purpose of each type, function, var, and const.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hashicorp/go-version"

	"github.com/pidgy/unitehud/exe"
	"github.com/pidgy/unitehud/system/desktop"
	"github.com/pidgy/unitehud/system/desktop/clicked"
)

const (
	Unknown Version = iota
	Latest
	Older
	Unstable
)

type (
	Version byte

	Query struct {
		Version `json:"-"`
		News    []string `json:"news"`
		Latest  string   `json:"latest"`
	}
)

func (v Version) String() string {
	switch v {
	case Latest:
		return "the latest"
	case Older:
		return "an older"
	case Unstable:
		return "an unstable"
	default:
		return "unknown"
	}
}

func Check() (Query, error) {
	b := &bytes.Buffer{}
	h, err := http.NewRequest(http.MethodGet, fmt.Sprintf("https://unitehud.dev/update.json?v=%s", exe.VersionSemVer), b)
	if err != nil {
		return Query{}, err
	}
	h.Header.Set("User-Agent", fmt.Sprintf("UniteHUD-Updater/%s", exe.VersionSemVer))

	r, err := http.DefaultClient.Do(h)
	if err != nil {
		return Query{}, err
	}
	defer r.Body.Close()

	q := Query{}

	err = json.NewDecoder(r.Body).Decode(&q)
	if err != nil {
		return Query{}, err
	}

	if q.Latest == "" {
		// notify.Warn("[Update] Failed to verify latest version")
		return Query{}, fmt.Errorf("unable to verify latest version, empty value")
	}

	local, err := version.NewVersion(exe.Version)
	if err != nil {
		// notify.Error("[Update] Failed to parse global version number (%v)", err)
		return Query{}, err
	}

	remote, err := version.NewVersion(q.Latest)
	if err != nil {
		// notify.Error("[Update] Failed to parse global version number (%v)", err)
		return Query{}, err
	}

	switch {
	case local.Equal(remote):
		q.Version = Latest
		return q, nil
		// notify.System("[Update] You are running the latest version of UniteHUD (%s)", local)
	case local.LessThan(remote):
		// notify.System("[Update] %s is now available for download (http://unitehud.dev)", remote)

		desktop.Notification("UniteHUD %s", remote).
			Says("An update is available for UniteHUD").
			When(clicked.VisitWebsite).
			Send()
		q.Version = Older
		return q, nil
	case remote.LessThan(local):
		// notify.System("[Update] You are running an unstable %s build", local)
		q.Version = Unstable
		return q, nil
	default:
		// notify.Warn("[Update] Unable to validate version %s ", remote)
		return q, nil
	}

	// for _, n := range q.News {
	// 	notify.Feed(nrgba.Highlight, "[News] %s", n)
	// }
}
