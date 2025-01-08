package config

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"gocv.io/x/gocv"

	"github.com/pidgy/unitehud/app"
	"github.com/pidgy/unitehud/core/notify"
	"github.com/pidgy/unitehud/core/rgba/nrgba"
	"github.com/pidgy/unitehud/core/state"
	"github.com/pidgy/unitehud/core/team"
	"github.com/pidgy/unitehud/core/template"
	"github.com/pidgy/unitehud/core/template/filter"
	"github.com/pidgy/unitehud/system/sort"
)

const (
	MainDisplay              = "Main Display"
	ProjectorWindow          = "UniteHUD Projector"
	NoVideoCaptureDevice     = -1
	DefaultVideoCaptureAPI   = "Any"
	DefaultVideoCaptureCodec = "Any"

	DeviceBluestacks = "bluestacks"
	DeviceMobile     = "mobile"
	DeviceSwitch     = "switch"
)

var (
	first = false
)

type Config struct {
	Acceptance float32

	Advanced struct {
		Accessibility struct {
			ReducedFontColors   bool
			ReducedFontGraphics bool
		}

		Stats struct {
			Disabled bool
		}

		DecreasedCaptureLevel time.Duration

		Notifications struct {
			Muted    bool
			Disabled struct {
				All,
				Updates,
				MatchStarting,
				MatchStopped bool
			}
		}

		Matching struct {
			Disabled struct {
				Scoring,
				Time,
				Objectives,
				Energy,
				Defeated,
				KOs bool

				Previews bool `json:"-"`
			}
		}

		Discord struct {
			Disabled bool
		}
	}

	Audio struct {
		Capture struct {
			Device struct {
				Name string
			}
		}
		Playback struct {
			Device struct {
				Name string
			}
		}
	}

	Crashed string

	Gaming struct {
		Device string
	}

	Remember struct {
		Discord bool
	}

	Scale float64
	Shift Shift

	Theme  Theme
	Themes map[string]Theme

	Video struct {
		Capture struct {
			Device struct {
				Index int
				API   string
				FPS   int
				Name  string
				Codec string
			}
			Window struct {
				Name string
				Lost string `json:"-"`
			}
		}
	}

	XY struct {
		Energy     image.Rectangle
		Scores     image.Rectangle
		Time       image.Rectangle
		Objectives image.Rectangle
		KOs        image.Rectangle
	}

	// Unsaved configurations.

	Record bool `json:"-"` // Record all matched images and logs.

	filenames map[string]map[string][]filter.Filter      `json:"-"`
	templates map[string]map[string][]*template.Template `json:"-"`
}

type Shift struct {
	N, E, S, W int
}

type Theme struct {
	Background,
	BackgroundAlt,
	Foreground,
	ForegroundAlt,
	Splash,
	TitleBarBackground,
	TitleBarForeground,
	BordersIdle,
	BordersActive,
	ScrollbarBackground,
	ScrollbarForeground color.NRGBA
}

var (
	Current Config
	cached  Config
)

func Cached() Config {
	return cached
}

func (c *Config) AssetIcon(file string) string {
	return fmt.Sprintf("%s/icon/%s", c.Assets(), file)
}

func (c *Config) Assets() string {
	return filepath.Join(app.WorkingDirectory(), app.AssetDirectory)
}

func (c Config) Eq(c2 Config) bool {
	return cmp.Equal(c, c2,
		cmpopts.IgnoreTypes(
			func() {},
			map[string]map[string][]filter.Filter{},
			map[string]map[string][]*template.Template{},
			map[string]int{},
		),
	)
}

func (c *Config) File() string {
	if len(os.Args) > 1 && strings.HasSuffix(os.Args[1], ".unitehud") {
		return os.Args[1]
	}
	return fmt.Sprintf("config-%s.unitehud", strings.ReplaceAll(app.Version, ".", "-"))
}

func (c *Config) IsNew() bool {
	is := first
	first = false
	return is
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

	return openNotNew(c.Gaming.Device)
}

func (c *Config) Save() error {
	notify.System("[Config] Saving %s profile (%s)", c.Gaming.Device, c.File())

	f, err := os.Create(c.File())
	if err != nil {
		return err
	}

	b, err := json.Marshal(c)
	if err != nil {
		return err
	}

	_, err = f.Write(sort.JSON(b))
	if err != nil {
		return err
	}

	cached = Current

	return nil
}

func (c *Config) SaveTemp() (string, error) {
	path := filepath.Join(os.TempDir(), c.File())

	notify.Debug("[Config] Saving temporary %s profile (%s)", c.Gaming.Device, path)

	f, err := os.Create(path)
	if err != nil {
		return path, err
	}
	defer f.Close()

	b, err := json.Marshal(c)
	if err != nil {
		return path, err
	}

	_, err = f.Write(sort.JSON(b))
	if err != nil {
		return path, err
	}

	return path, nil
}

func (c *Config) ScoringOption() image.Rectangle {
	return image.Rectangle{
		Min: image.Pt(c.XY.Energy.Min.X, c.XY.Energy.Min.Y),
		Max: image.Pt(c.XY.Energy.Max.X, c.XY.Energy.Max.Y-75),
	}
}

func (c *Config) SetDefaultTheme() {
	c.Theme.Background = nrgba.Background.Color()
	c.Theme.BackgroundAlt = nrgba.BackgroundAlt.Color()
	c.Theme.ForegroundAlt = nrgba.White.Alpha(100).Color()
	c.Theme.Foreground = nrgba.White.Color()
	c.Theme.Splash = nrgba.Splash.Color()
	c.Theme.TitleBarBackground = nrgba.Background.Color()
	c.Theme.TitleBarForeground = nrgba.White.Color()
	c.Theme.BordersIdle = nrgba.Discord.Alpha(100).Color()
	c.Theme.BordersActive = nrgba.Active.Alpha(100).Color()
	c.Theme.ScrollbarBackground = nrgba.Transparent.Color()
	c.Theme.ScrollbarForeground = nrgba.Discord.Alpha(100).Color()
}

func (c *Config) Total() (total int) {
	for k := range c.templates {
		for _, v := range c.templates[k] {
			total += len(v)
		}
	}
	return
}

func (c *Config) TemplateMatchMap() map[string]int {
	m := make(map[string]int)

	for category := range c.templates {
		for _, templates := range c.templates[category] {
			for _, t := range templates {
				m[t.Truncated()] = 0
			}
		}
	}

	return m
}

func (c *Config) Templates(category string) map[string][]*template.Template {
	return c.templates[category]
}

func (c *Config) TemplatesByName(category, name string) []*template.Template {
	return c.templates[category][name]
}

func (c *Config) TemplateCategories() (categories []string) {
	for k := range c.templates {
		categories = append(categories, k)
	}
	sort.Strings(categories)
	return
}

func (c *Config) TemplatesStarting() []*template.Template {
	return c.templates["starting"][team.Game.Name]
}

func (c *Config) TemplatesEnding() []*template.Template {
	return c.templates["ending"][team.Game.Name]
}

func (c *Config) TemplatesSurrender() []*template.Template {
	return c.templates["surrender"][team.Game.Name]
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

func (c *Config) UnsetHiddenThemes() {
	current := reflect.ValueOf(c.Theme)

	failed := []string{}
	applied := []string{}

	for i := 0; i < current.NumField(); i++ {
		name := reflect.Indirect(current).Type().Field(i).Name

		v, ok := current.Field(i).Interface().(color.NRGBA)
		if !ok {
			failed = append(failed, name)
			continue
		}

		if v != nrgba.Nothing.Color() {
			continue
		}

		want := Config{}
		want.SetDefaultTheme()

		e := reflect.ValueOf(&c.Theme).Field(i).Elem()
		if e.CanSet() {
			e.Set(reflect.ValueOf(&want.Theme).Field(i).Elem())
		}

		applied = append(applied, name)
	}

	if len(failed) > 0 {
		notify.Warn("[Config] Failed to apply default themes (%s)", strings.Join(failed, ", "))
	}

	if len(applied) > 0 {
		notify.System("[Config] Default themes applied to %s", strings.Join(applied, ", "))
	}
}
func (c *Config) deviceAsset(dir, file string) string {
	return filepath.Join(c.deviceAssets(), dir, file)
}

func (c *Config) deviceAssets() string {
	return filepath.Join(app.WorkingDirectory(), app.AssetDirectory, "device", c.Gaming.Device)
}

func (c *Config) loadDeviceAssets() {
	c.filenames = map[string]map[string][]filter.Filter{
		"goals": {
			team.Game.Name: {
				filter.New(team.Game, c.deviceAsset(team.Game.Name, "purple_base_open.png"), state.PurpleBaseOpen.Int(), false),
				filter.New(team.Game, c.deviceAsset(team.Game.Name, "orange_base_open.png"), state.OrangeBaseOpen.Int(), false),
				filter.New(team.Game, c.deviceAsset(team.Game.Name, "purple_base_closed.png"), state.PurpleBaseClosed.Int(), false),
				filter.New(team.Game, c.deviceAsset(team.Game.Name, "orange_base_closed.png"), state.OrangeBaseClosed.Int(), false),
			},
		},
		"killed": {
			team.Game.Name: {
				filter.New(team.Game, c.deviceAsset(team.Game.Name, "killed.png"), state.Killed.Int(), false),
				filter.New(team.Game, c.deviceAsset(team.Game.Name, "killed_with_points.png"), state.KilledWithPoints.Int(), false),
				filter.New(team.Game, c.deviceAsset(team.Game.Name, "killed_without_points.png"), state.KilledWithoutPoints.Int(), false),
			},
		},
		"secure": {
			team.Game.Name: {
				filter.New(team.Game, c.deviceAsset(team.Game.Name, "rayquaza_ally.png"), state.RayquazaSecurePurple.Int(), false),
				filter.New(team.Game, c.deviceAsset(team.Game.Name, "rayquaza_enemy.png"), state.RayquazaSecureOrange.Int(), false),
				filter.New(team.Game, c.deviceAsset(team.Game.Name, "regieleki_ally.png"), state.RegielekiSecurePurple.Int(), false),
				filter.New(team.Game, c.deviceAsset(team.Game.Name, "regieleki_enemy.png"), state.RegielekiSecureOrange.Int(), false),
				filter.New(team.Game, c.deviceAsset(team.Game.Name, "regice_ally.png"), state.RegiceSecurePurple.Int(), false),
				filter.New(team.Game, c.deviceAsset(team.Game.Name, "regice_enemy.png"), state.RegiceSecureOrange.Int(), false),
				filter.New(team.Game, c.deviceAsset(team.Game.Name, "regice_enemy_alt.png"), state.RegiceSecureOrange.Int(), false),
				filter.New(team.Game, c.deviceAsset(team.Game.Name, "regirock_ally.png"), state.RegirockSecurePurple.Int(), false),
				filter.New(team.Game, c.deviceAsset(team.Game.Name, "regirock_enemy.png"), state.RegirockSecureOrange.Int(), false),
				filter.New(team.Game, c.deviceAsset(team.Game.Name, "registeel_ally.png"), state.RegisteelSecurePurple.Int(), false),
				filter.New(team.Game, c.deviceAsset(team.Game.Name, "registeel_enemy.png"), state.RegisteelSecureOrange.Int(), false),
			},
		},
		"ko": {
			team.Game.Name: {
				// filter.New(team.Game, Current.ProfileAssets()+"/game/ko_ally.png", state.KOPurple.Int(), false),
				// filter.New(team.Game, Current.ProfileAssets()+"/game/ko_streak_ally.png", state.KOStreakPurple.Int(), false),
				// filter.New(team.Game, Current.ProfileAssets()+"/game/ko_enemy.png", state.KOOrange.Int(), false),
				// filter.New(team.Game, Current.ProfileAssets()+"/game/ko_streak_enemy.png", state.KOStreakOrange.Int(), false),
			},
		},
		"objective": {
			team.Game.Name: {
				filter.New(team.Game, c.deviceAsset(team.Game.Name, "objective.png"), state.ObjectivePresent.Int(), false),
				filter.New(team.Game, c.deviceAsset(team.Game.Name, "objective_half.png"), state.ObjectivePresent.Int(), false),
				filter.New(team.Game, c.deviceAsset(team.Game.Name, "objective_orange_base.png"), state.ObjectiveReachedOrange.Int(), false),
			},
		},
		"starting": {
			team.Game.Name: {
				// filter.New(team.Game, c.deviceAsset(team.Game.Name, "vs.png"), state.MatchStarting.Int(), false),
				filter.New(team.Game, c.deviceAsset(team.Game.Name, "vs_alt.png"), state.MatchStarting.Int(), false),
			},
		},
		"ending": {
			team.Game.Name: {
				filter.New(team.Game, c.deviceAsset(team.Game.Name, "end.png"), state.MatchEnding.Int(), false),
			},
		},
		"surrender": {
			team.Game.Name: {
				filter.New(team.Game, c.deviceAsset(team.Game.Name, "surrender_enemy.png"), state.SurrenderOrange.Int(), false),
				filter.New(team.Game, c.deviceAsset(team.Game.Name, "surrender_ally.png"), state.SurrenderPurple.Int(), false),
			},
		},
		"scoring": {
			team.Game.Name: {
				// filter.New(team.Game, c.deviceAsset(team.Game.Name, "pre_scoring_alt_alt.png"), state.PreScore.Int(), false),
				// filter.New(team.Game, c.deviceAsset(team.Game.Name, "pre_scoring_alt.png"), state.PreScore.Int(), false),
				// filter.New(team.Game, c.deviceAsset(team.Game.Name, "pre_scoring.png"), state.PreScore.Int(), false),
				// filter.New(team.Game, c.deviceAsset(team.Game.Name, "post_scoring.png"), state.PostScore.Int(), false),
				// filter.New(team.Game, c.deviceAsset(team.Game.Name, "press_button_to_score.png"), state.PressButtonToScore.Int(), false),
				filter.New(team.Game, c.deviceAsset(team.Game.Name, "press_button_to_score_alt.png"), state.SelfScoreIndicator.Int(), false),
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

func (c *Config) pointFiles(t *team.Team) []filter.Filter {
	var files []string

	root := fmt.Sprintf("%s/%s/points/", c.deviceAssets(), t.Name)
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if info == nil {
			return fmt.Errorf("directory does not exist")
		}
		if info.IsDir() {
			if info.Name() != "points" {
				notify.Warn("[Config] Skipping templates from %s", filepath.Join(root, info.Name()))
				return filepath.SkipDir
			}
		}

		files = append(files, path)

		return nil
	})
	if err != nil {
		notify.Error("[Config] Failed to read from point directory \"%s\" (%v)", root, err)
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
			notify.Warn("[Config] Failed to invalidate \"%s\" file \"%s\" (%v)", root, file, err)
			continue
		}

		alias := strings.Contains(file, "alt") || strings.Contains(file, "big")

		filters = append(filters, filter.New(t, file, value, alias))
	}

	return filters
}

func (c *Config) scoreFiles(t *team.Team) []filter.Filter {
	var files []string

	root := fmt.Sprintf("%s/%s/score/", c.deviceAssets(), t.Name)

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if info == nil {
			return fmt.Errorf("directory does not exist")
		}
		if info.IsDir() {
			if info.Name() != "score" {
				notify.Warn("[Config] Skipping %s", filepath.Join(root, info.Name()))
				return filepath.SkipDir
			}
		}

		files = append(files, path)

		return nil
	})
	if err != nil {
		notify.Error("[Config] Failed to read from \"score\" directory \"%s\" (%v)", root, err)
		return nil
	}

	filters := []filter.Filter{}

	for _, file := range files {
		if !strings.Contains(file, "score") {
			continue
		}

		if !strings.EqualFold(filepath.Ext(file), ".png") {
			continue
		}

		filters = append(filters, filter.New(t, file, state.Nothing.Int(), false))
	}

	return filters
}

func (c *Config) setDefaultAdvancedSettings() {
	c.Advanced.Notifications.Muted = false
	c.Advanced.Notifications.Disabled.All = false
	c.Advanced.Notifications.Disabled.Updates = false
	c.Advanced.Notifications.Disabled.MatchStarting = true
	c.Advanced.Notifications.Disabled.MatchStopped = true
	c.Advanced.Notifications.Muted = true

	c.Advanced.Discord.Disabled = false
}

func (c *Config) setDefaultAreas() {
	c.XY.Energy = image.Rect(908, 764, 1008, 864)
	c.XY.Scores = image.Rect(500, 50, 1500, 250)
	c.XY.Time = image.Rect(846, 0, 1046, 100)
	c.XY.Objectives = image.Rect(350, 200, 1200, 310)
	c.XY.KOs = image.Rect(730, 130, 1160, 310)
}

func Open(device string) error {
	defer func() {
		r := recover()
		if r != nil {
			notify.Warn("[Config] Corrupted .unitehud file (%s)", Current.File())
			recovered(r)
		}
	}()

	if device == "" {
		Current.Gaming.Device = DeviceSwitch
	}

	defer validate()

	notify.System("[Config] Loading %s profile (%s)", Current.Gaming.Device, Current.File())

	Current = Config{
		Scale:      1,
		Shift:      Shift{},
		Acceptance: .91,
	}
	defer Current.loadDeviceAssets()

	Current.Gaming.Device = DeviceSwitch
	Current.Video.Capture.Window.Name = MainDisplay
	Current.Video.Capture.Device.Index = NoVideoCaptureDevice
	Current.Video.Capture.Device.API = DefaultVideoCaptureAPI
	Current.Video.Capture.Device.Codec = DefaultVideoCaptureCodec
	Current.Gaming.Device = DeviceSwitch
	Current.SetDefaultTheme()
	Current.setDefaultAreas()
	Current.setDefaultAdvancedSettings()

	b, err := os.ReadFile(Current.File())
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}

		first = true

		b = json.RawMessage(`{}`)
	}

	err = json.Unmarshal(b, &Current)
	if err != nil {
		return err
	}

	if Current.Video.Capture.Window.Name == "" {
		Current.Video.Capture.Window.Name = MainDisplay
		Current.Video.Capture.Device.Index = NoVideoCaptureDevice
	}

	if Current.Themes == nil {
		Current.Themes = make(map[string]Theme)
	}

	if Current.Video.Capture.Device.FPS == 0 {
		Current.Video.Capture.Device.FPS = 60
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

func openNotNew(device string) error {
	err := Open(device)
	if err != nil {
		return err
	}

	first = false

	return nil
}

func recovered(r interface{}) {
	s := ""
	switch e := r.(type) {
	case error:
		s = e.Error()
	case string:
		s = e
	}
	notify.Warn("[Config] Recovered from %s", s)
}

func validate() {
	notify.System("[Config] Validating %s", Current.File())

	Current.templates = map[string]map[string][]*template.Template{
		"goals": {
			team.Game.Name: {},
		},
		"starting": {
			team.Game.Name: {},
		},
		"ending": {
			team.Game.Name: {},
		},
		"surrender": {
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
					notify.Error("[Config] Failed to read %s template from file \"%s\"", filepath.Join(category, subcategory), t.File)
					continue
				}
			}
		}
	}
}
