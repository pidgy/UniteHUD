package config

import (
	"encoding/json"
	"fmt"
	"image"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"gocv.io/x/gocv"

	"github.com/pidgy/unitehud/filter"
	"github.com/pidgy/unitehud/global"
	"github.com/pidgy/unitehud/notify"
	"github.com/pidgy/unitehud/state"
	"github.com/pidgy/unitehud/team"
	"github.com/pidgy/unitehud/template"
)

const (
	MainDisplay          = "Main Display"
	ProjectorWindow      = "UniteHUD Projector"
	NoVideoCaptureDevice = -1

	ProfilePlayer      = "player"
	ProfileBroadcaster = "broadcaster"

	PlatformSwitch     = "switch"
	PlatformMobile     = "mobile"
	PlatformBluestacks = "bluestacks"
)

type Config struct {
	Window                   string
	VideoCaptureDevice       int
	LostWindow               string `json:"-"`
	Record                   bool   `json:"-"` // Record all matched images and logs.
	Energy                   image.Rectangle
	Scores                   image.Rectangle
	Time                     image.Rectangle
	Objectives               image.Rectangle
	KOs                      image.Rectangle
	filenames                map[string]map[string][]filter.Filter      `json:"-"`
	templates                map[string]map[string][]*template.Template `json:"-"`
	Scale                    float64
	Shift                    Shift
	Acceptance               float32
	Profile                  string
	DisableBrowserFormatting bool
	Platform                 string
	HUDOverlay               bool

	DisableScoring, DisableTime, DisableObjectives, DisableEnergy, DisableDefeated, DisableKOs, DisablePreviews bool

	Crashed string

	load func()
}

type Shift struct {
	N, E, S, W int
}

var Current Config

func (c *Config) Assets() string {
	e, err := os.Executable()
	if err != nil {
		notify.Error("Failed to find profile directory (%v)", err)
		return ""
	}

	return fmt.Sprintf(`%s\assets`, filepath.Dir(e))
}

func (c *Config) Eq(c2 *Config) bool {
	return cmp.Equal(c, c2,
		cmpopts.IgnoreTypes(
			func() {},
			map[string]map[string][]filter.Filter{},
			map[string]map[string][]*template.Template{},
		),
	)
}

func (c *Config) File() string {
	return fmt.Sprintf("%s-config.unitehud.%s", strings.ReplaceAll(global.Version, ".", "-"), c.Profile)
}

func (c *Config) ProfileAssets() string {
	e, err := os.Executable()
	if err != nil {
		notify.Error("Failed to find profile directory (%v)", err)
		return ""
	}

	return path.Join(filepath.Dir(e), "assets", "profiles", c.Profile, c.Platform)
}

func (c *Config) Reload() {
	validate()
}

func (c *Config) Report(crash string) {
	c.Crashed = crash
}

func (c *Config) Reset() error {
	defer validate()

	err := os.Remove(c.File())
	if err != nil {
		return err
	}

	return Load(c.Profile)
}

func (c *Config) Save() error {
	notify.System("Saving configuration to %s", c.File())

	f, err := os.Create(c.File())
	if err != nil {
		return err
	}

	b, err := json.MarshalIndent(c, "", " ")
	if err != nil {
		return err
	}

	_, err = f.Write(b)
	if err != nil {
		return err
	}

	return nil
}

func (c *Config) Scoring() image.Rectangle {
	return image.Rectangle{
		Min: image.Pt(c.Energy.Min.X-50, c.Energy.Min.Y),
		Max: image.Pt(c.Energy.Max.X+50, c.Energy.Max.Y+100),
	}
}

func (c *Config) ScoringOption() image.Rectangle {
	return image.Rectangle{
		Min: image.Pt(c.Energy.Min.X-100, c.Energy.Min.Y-100),
		Max: image.Pt(c.Energy.Max.X+100, c.Energy.Max.Y-100),
	}
}

func (c *Config) SetDefaultAreas() {
	energy := image.Rect(908, 764, 1008, 864)
	scores := image.Rect(500, 50, 1500, 250)
	time := image.Rect(846, 0, 1046, 100)

	c.Energy = energy
	c.Scores = scores
	c.Time = time
	c.setKOArea()
	c.setObjectiveArea()
}

func (c *Config) SetProfile(p string) {
	switch p {
	case ProfileBroadcaster:
		c.setProfileBroadcaster()
	default:
		c.setProfilePlayer()
	}
}

func (c *Config) TemplatesGame(n string) []*template.Template {
	return c.templates["game"][n]
}

func (c *Config) TemplatesGoals(n string) []*template.Template {
	return c.templates["goals"][n]
}

func (c *Config) TemplatesKilled(n string) []*template.Template {
	return c.templates["killed"][n]
}

func (c *Config) TemplatesKO(n string) []*template.Template {
	return c.templates["ko"][n]
}

func (c *Config) TemplatesPoints(n string) []*template.Template {
	return c.templates["points"][n]
}

func (c *Config) TemplatesSecure(n string) []*template.Template {
	return c.templates["secure"][n]
}

func (c *Config) TemplatesScored(n string) []*template.Template {
	return c.templates["scored"][n]
}

func (c *Config) TemplatesScoredAll() map[string][]*template.Template {
	return c.templates["scored"]
}

func (c *Config) TemplatesScoring(n string) []*template.Template {
	return c.templates["scoring"][n]
}

func (c *Config) TemplatesTime(n string) []*template.Template {
	return c.templates["time"][n]
}

func (c *Config) pointFiles(t *team.Team) []filter.Filter {
	var files []string

	root := fmt.Sprintf("%s/%s/points/", c.ProfileAssets(), t.Name)
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if info == nil {
			return fmt.Errorf("Directory does not exist")
		}
		if info.IsDir() {
			if info.Name() != "points" {
				notify.SystemWarn("Skipping templates from %s%s", root, info.Name())
				return filepath.SkipDir
			}
		}

		files = append(files, path)

		return nil
	})
	if err != nil {
		notify.Error("Failed to read from \"point\" directory \"%s\" (%v)", root, err)
		return nil
	}

	filters := []filter.Filter{}
	for _, file := range files {
		if !strings.Contains(file, ".png") && !strings.Contains(file, ".PNG") {
			continue
		}
		b := strings.Split(file, "point_")
		if len(b) != 2 {
			continue
		}

		v := filter.Strip(b[1])
		if v == "" {
			continue
		}

		value, err := strconv.Atoi(v)
		if err != nil {
			notify.SystemWarn("Failed to invalidate \"%s\" file \"%s\" (%v)", root, file, err)
			continue
		}

		alias := strings.Contains(file, "alt") || strings.Contains(file, "big")

		filters = append(filters, filter.New(t, file, value, alias))
	}

	return filters
}

func (c *Config) scoreFiles(t *team.Team) []filter.Filter {
	var files []string

	root := fmt.Sprintf("%s/%s/score/", c.ProfileAssets(), t.Name)
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if info == nil {
			return fmt.Errorf("Directory does not exist")
		}
		if info.IsDir() {
			if info.Name() != "score" {
				notify.SystemWarn("Skipping \"%s%s\"", root, info.Name())
				return filepath.SkipDir
			}
		}

		files = append(files, path)

		return nil
	})
	if err != nil {
		notify.Error("Failed to read from \"score\" directory \"%s\" (%v)", root, err)
		return nil
	}

	filters := []filter.Filter{}
	for _, file := range files {
		if !strings.Contains(file, "score") {
			continue
		}

		if !strings.Contains(file, ".png") &&
			!strings.Contains(file, ".PNG") {
			continue
		}

		filters = append(filters, filter.New(t, file, -1, false))
	}

	return filters
}

func (c *Config) setKOArea() {
	switch c.Profile {
	case ProfileBroadcaster:
		c.KOs = image.Rect(730, 130, 1160, 310)
	case ProfilePlayer:
		c.KOs = image.Rect(730, 130, 1160, 310)
	}
}

func (c *Config) setObjectiveArea() {
	switch c.Profile {
	case ProfileBroadcaster:
		c.Objectives = image.Rect(350, 210, 1200, 310)
	case ProfilePlayer:
		c.Objectives = image.Rect(350, 210, 1200, 310)
	}
}

func (c *Config) setProfileBroadcaster() {
	c.Profile = ProfileBroadcaster

	c.load = loadProfileAssetsBroadcaster

	c.DisableEnergy = true
	c.DisableScoring = true
	c.DisableDefeated = true
}

func (c *Config) setProfilePlayer() {
	c.Profile = ProfilePlayer

	c.load = loadProfileAssetsPlayer

}

func Load(profile string) error {
	defer validate()

	if profile == "" {
		profile = ProfilePlayer
		Current.SetProfile(profile)
	}

	notify.System("Loading \"%s\" configuration from \"%s\"", profile, Current.File())

	ok := open()
	if !ok {
		Current = Config{
			Window:             MainDisplay,
			VideoCaptureDevice: NoVideoCaptureDevice,
			Scale:              1,
			Shift:              Shift{},
			Profile:            profile,
			Acceptance:         .91,
			Platform:           PlatformSwitch,
		}
		Current.SetProfile(profile)
		Current.SetDefaultAreas()
		Current.load()
	}

	if Current.Window == "" {
		Current.Window = MainDisplay
		Current.VideoCaptureDevice = NoVideoCaptureDevice
	}

	if Current.Platform == "" {
		Current.Platform = "Switch"
	}

	return Current.Save()
}

func TemplatesFirstRound(t1 []*template.Template) []*template.Template {
	t2 := []*template.Template{}
	for _, t := range t1 {
		if t.Value == 0 {
			continue
		}
		t2 = append(t2, t)
	}
	return t2
}

func loadProfileAssetsBroadcaster() {
	Current.filenames = map[string]map[string][]filter.Filter{
		"goals": {
			team.Game.Name: {
				filter.New(team.Game, Current.ProfileAssets()+"/game/purple_base_open.png", state.PurpleBaseOpen.Int(), false),
				filter.New(team.Game, Current.ProfileAssets()+"/game/orange_base_open.png", state.OrangeBaseOpen.Int(), false),
				filter.New(team.Game, Current.ProfileAssets()+"/game/purple_base_closed.png", state.PurpleBaseClosed.Int(), false),
				filter.New(team.Game, Current.ProfileAssets()+"/game/orange_base_closed.png", state.OrangeBaseClosed.Int(), false),
			},
		},
		"killed": {},
		"secure": {
			team.Game.Name: {
				filter.New(team.Game, Current.ProfileAssets()+"/game/rayquaza_ally.png", state.RayquazaSecurePurple.Int(), false),
				filter.New(team.Game, Current.ProfileAssets()+"/game/rayquaza_enemy.png", state.RayquazaSecureOrange.Int(), false),
				filter.New(team.Game, Current.ProfileAssets()+"/game/regice_ally.png", state.RegiceSecurePurple.Int(), false),
				filter.New(team.Game, Current.ProfileAssets()+"/game/regice_enemy.png", state.RegiceSecureOrange.Int(), false),
				filter.New(team.Game, Current.ProfileAssets()+"/game/regirock_ally.png", state.RegirockSecurePurple.Int(), false),
				filter.New(team.Game, Current.ProfileAssets()+"/game/regirock_enemy.png", state.RegirockSecureOrange.Int(), false),
				filter.New(team.Game, Current.ProfileAssets()+"/game/registeel_ally.png", state.RegisteelSecurePurple.Int(), false),
				filter.New(team.Game, Current.ProfileAssets()+"/game/registeel_enemy.png", state.RegisteelSecureOrange.Int(), false),
				filter.New(team.Game, Current.ProfileAssets()+"/game/regieleki_ally.png", state.RegielekiSecurePurple.Int(), false),
				filter.New(team.Game, Current.ProfileAssets()+"/game/regieleki_enemy.png", state.RegielekiSecureOrange.Int(), false),
			},
		},
		"ko": {
			team.Game.Name: {
				filter.New(team.Game, Current.ProfileAssets()+"/game/ko_ally.png", state.KOPurple.Int(), false),
				filter.New(team.Game, Current.ProfileAssets()+"/game/ko_streak_ally.png", state.KOStreakPurple.Int(), false),
				filter.New(team.Game, Current.ProfileAssets()+"/game/ko_enemy.png", state.KOOrange.Int(), false),
				filter.New(team.Game, Current.ProfileAssets()+"/game/ko_streak_enemy.png", state.KOStreakOrange.Int(), false),
			},
		},
		"objective": {
			team.Game.Name: {
				filter.New(team.Game, Current.ProfileAssets()+"/game/objective.png", state.ObjectivePresent.Int(), false),
				filter.New(team.Game, Current.ProfileAssets()+"/game/objective_half.png", state.ObjectivePresent.Int(), false),
				filter.New(team.Game, Current.ProfileAssets()+"/game/objective_orange_base.png", state.ObjectiveReachedOrange.Int(), false),
			},
		},
		"game": {
			"vs": {
				filter.New(team.Game, Current.ProfileAssets()+"/game/vs.png", state.MatchStarting.Int(), false),
				filter.New(team.Game, Current.ProfileAssets()+"/game/vs_alt.png", state.MatchStarting.Int(), false),
			},
			"end": {
				filter.New(team.Game, Current.ProfileAssets()+"/game/end.png", state.MatchEnding.Int(), false),
			},
		},
		"scoring": {},
		"scored":  {},
		"points":  {},
		"time": {
			team.Time.Name: Current.pointFiles(team.Time),
		},
	}
}

func loadProfileAssetsPlayer() {
	Current.filenames = map[string]map[string][]filter.Filter{
		"goals": {
			team.Game.Name: {
				filter.New(team.Game, Current.ProfileAssets()+"/game/purple_base_open.png", state.PurpleBaseOpen.Int(), false),
				filter.New(team.Game, Current.ProfileAssets()+"/game/orange_base_open.png", state.OrangeBaseOpen.Int(), false),
				filter.New(team.Game, Current.ProfileAssets()+"/game/purple_base_closed.png", state.PurpleBaseClosed.Int(), false),
				filter.New(team.Game, Current.ProfileAssets()+"/game/orange_base_closed.png", state.OrangeBaseClosed.Int(), false),
			},
		},
		"killed": {
			team.Game.Name: {
				filter.New(team.Game, Current.ProfileAssets()+"/game/killed.png", state.Killed.Int(), false),
				filter.New(team.Game, Current.ProfileAssets()+"/game/killed_with_points.png", state.KilledWithPoints.Int(), false),
				filter.New(team.Game, Current.ProfileAssets()+"/game/killed_without_points.png", state.KilledWithoutPoints.Int(), false),
			},
		},
		"secure": {
			team.Game.Name: {
				filter.New(team.Game, Current.ProfileAssets()+"/game/rayquaza_ally.png", state.RayquazaSecurePurple.Int(), false),
				filter.New(team.Game, Current.ProfileAssets()+"/game/rayquaza_enemy.png", state.RayquazaSecureOrange.Int(), false),
				filter.New(team.Game, Current.ProfileAssets()+"/game/regieleki_ally.png", state.RegielekiSecurePurple.Int(), false),
				filter.New(team.Game, Current.ProfileAssets()+"/game/regieleki_enemy.png", state.RegielekiSecureOrange.Int(), false),
				filter.New(team.Game, Current.ProfileAssets()+"/game/regice_ally.png", state.RegiceSecurePurple.Int(), false),
				filter.New(team.Game, Current.ProfileAssets()+"/game/regice_enemy.png", state.RegiceSecureOrange.Int(), false),
				filter.New(team.Game, Current.ProfileAssets()+"/game/regirock_ally.png", state.RegirockSecurePurple.Int(), false),
				filter.New(team.Game, Current.ProfileAssets()+"/game/regirock_enemy.png", state.RegirockSecureOrange.Int(), false),
				filter.New(team.Game, Current.ProfileAssets()+"/game/registeel_ally.png", state.RegisteelSecurePurple.Int(), false),
				filter.New(team.Game, Current.ProfileAssets()+"/game/registeel_enemy.png", state.RegisteelSecureOrange.Int(), false),
			},
		},
		"ko": {
			team.Game.Name: {
				filter.New(team.Game, Current.ProfileAssets()+"/game/ko_ally.png", state.KOPurple.Int(), false),
				filter.New(team.Game, Current.ProfileAssets()+"/game/ko_streak_ally.png", state.KOStreakPurple.Int(), false),
				filter.New(team.Game, Current.ProfileAssets()+"/game/ko_enemy.png", state.KOOrange.Int(), false),
				filter.New(team.Game, Current.ProfileAssets()+"/game/ko_streak_enemy.png", state.KOStreakOrange.Int(), false),
			},
		},
		"objective": {
			team.Game.Name: {
				filter.New(team.Game, Current.ProfileAssets()+"/game/objective.png", state.ObjectivePresent.Int(), false),
				filter.New(team.Game, Current.ProfileAssets()+"/game/objective_half.png", state.ObjectivePresent.Int(), false),
				filter.New(team.Game, Current.ProfileAssets()+"/game/objective_orange_base.png", state.ObjectiveReachedOrange.Int(), false),
			},
		},
		"game": {
			"vs": {
				filter.New(team.Game, Current.ProfileAssets()+"/game/vs.png", state.MatchStarting.Int(), false),
				filter.New(team.Game, Current.ProfileAssets()+"/game/vs_alt.png", state.MatchStarting.Int(), false),
			},
			"end": {
				filter.New(team.Game, Current.ProfileAssets()+"/game/end.png", state.MatchEnding.Int(), false),
			},
		},
		"scoring": {
			team.Game.Name: {
				filter.New(team.Game, Current.ProfileAssets()+"/game/pre_scoring_alt_alt.png", state.PreScore.Int(), false),
				filter.New(team.Game, Current.ProfileAssets()+"/game/pre_scoring_alt.png", state.PreScore.Int(), false),
				filter.New(team.Game, Current.ProfileAssets()+"/game/pre_scoring.png", state.PreScore.Int(), false),
				filter.New(team.Game, Current.ProfileAssets()+"/game/post_scoring.png", state.PostScore.Int(), false),
				filter.New(team.Game, Current.ProfileAssets()+"/game/press_button_to_score.png", state.PressButtonToScore.Int(), false),
			},
		},
		"scored": {
			team.Purple.Name: Current.scoreFiles(team.Purple),
			team.Orange.Name: Current.scoreFiles(team.Orange),
			team.Self.Name:   Current.scoreFiles(team.Self),
			team.First.Name:  Current.scoreFiles(team.First),
		},
		"points": {
			team.Purple.Name: Current.pointFiles(team.Purple),
			team.Orange.Name: Current.pointFiles(team.Orange),
			team.Self.Name:   Current.pointFiles(team.Self),
			team.First.Name:  Current.pointFiles(team.First),
			team.Energy.Name: Current.pointFiles(team.Energy),
		},
		"time": {
			team.Time.Name: Current.pointFiles(team.Time),
		},
	}
}

func open() bool {
	if Current.Profile == "" {
		Current.Profile = ProfilePlayer
	}

	b, err := os.ReadFile(Current.File())
	if err != nil {
		return false
	}

	c := Config{
		load: loadProfileAssetsPlayer,
	}

	err = json.Unmarshal(b, &c)
	if err != nil {
		return false
	}

	c.SetProfile(Current.Profile)

	Current = c

	Current.load()

	return true
}

func validate() {
	Current.templates = map[string]map[string][]*template.Template{
		"goals": {
			team.Game.Name: {},
		},
		"game": {
			team.Game.Name: {},
		},
		"ko": {
			team.Game.Name: {},
		},
		"objective": {
			team.Game.Name: {},
		},
		"killed": {
			team.Game.Name: {},
		},
		"scoring": {
			team.Game.Name: {},
		},
		"scored": {
			team.Orange.Name: {},
			team.Purple.Name: {},
			team.Self.Name:   {},
			team.First.Name:  {},
		},
		"secure": {
			team.Game.Name: {},
		},
		"points": {
			team.Orange.Name: {},
			team.Purple.Name: {},
			team.Self.Name:   {},
			team.First.Name:  {},
			team.Energy.Name: {},
		},
		"time": {
			team.Time.Name: {},
		},
	}

	for category := range Current.filenames {
		for subcategory, filters := range Current.filenames[category] {
			for _, filter := range filters {
				mat := gocv.IMRead(filter.File, gocv.IMReadColor)

				transparent := false

				switch category {
				case "points":
					switch filter.Team.Name {
					case team.First.Name,
						team.Self.Name,
						team.Orange.Name,
						team.Purple.Name,
						team.Energy.Name,
						team.Game.Name:
						transparent = true
					}
				}

				template := template.New(filter, mat, category, subcategory)
				if transparent {
					template = template.AsTransparent()
				}

				Current.templates[category][filter.Team.Name] = append(
					Current.templates[category][filter.Team.Name],
					template,
				)
			}
		}
	}

	for category := range Current.templates {
		for subcategory, templates := range Current.templates[category] {
			for _, t := range templates {
				if t.Empty() {
					notify.Error("Failed to read %s/%s template from file \"%s\"", category, subcategory, t.File)
					continue
				}

			}
		}
	}
}
