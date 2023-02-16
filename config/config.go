package config

import (
	"encoding/json"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
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
	NoVideoCaptureDevice = -1

	ProfilePlayer      = "player"
	ProfileBroadcaster = "broadcaster"
)

type Config struct {
	Window             string
	VideoCaptureDevice int
	LostWindow         string `json:"-"`
	Record             bool   `json:"-"` // Record all matched images and logs.
	Energy             image.Rectangle
	Map                image.Rectangle
	Scores             image.Rectangle
	Time               image.Rectangle
	Objectives         image.Rectangle
	KOs                image.Rectangle
	Filenames          map[string]map[string][]filter.Filter     `json:"-"`
	Templates          map[string]map[string][]template.Template `json:"-"`
	Scale              float64
	Shift              Shift
	Acceptance         float32
	Profile            string

	DisableScoring, DisableTime, DisableObjectives, DisableEnergy, DisableDefeated, DisableKOs bool

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

func (c *Config) File() string {
	return fmt.Sprintf("%s-config.unitehud.%s", strings.ReplaceAll(global.Version, ".", "-"), c.Profile)
}

func (c *Config) ProfileAssets() string {
	e, err := os.Executable()
	if err != nil {
		notify.Error("Failed to find profile directory (%v)", err)
		return ""
	}

	return fmt.Sprintf(`%s\assets\profiles\%s`, filepath.Dir(e), c.Profile)
}

func (c *Config) Reload() {
	defer validate()
}

func (c *Config) Report(crash string) {
	c.Crashed = crash

	err := c.Save()
	if err != nil {
		log.Panic().Err(err).Msg("failed to save crash report")
	}
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
	minimap := image.Rect(70, 100, 470, 250)
	scores := image.Rect(500, 50, 1500, 250)
	time := image.Rect(846, 0, 1046, 100)

	c.Energy = energy
	c.Map = minimap
	c.Scores = scores
	c.Time = time
	c.setKOArea()
	c.setObjectiveArea()
}

func (c *Config) SetProfile() {
	switch c.Profile {
	case ProfileBroadcaster:
		c.setProfileBroadcaster()
	default:
		c.setProfilePlayer()
	}
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
			log.Warn().Str("file", file).Msg("invalid file in points directory")
			continue
		}

		value, err := strconv.Atoi(v)
		if err != nil {
			notify.SystemWarn("Failed to invalid \"point\" file \"%s\" (%v)", root, file, err)
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
		c.Objectives = image.Rect(350, 200, 1350, 315)
	case ProfilePlayer:
		c.Objectives = image.Rect(600, 200, 1350, 315)
	}
}

func (c *Config) setProfileBroadcaster() {
	c.Profile = ProfileBroadcaster

	c.load = loadProfileAssetsBroadcaster

	c.DisableEnergy = true
	c.DisableScoring = true
	c.DisableDefeated = true

	c.DisableObjectives = false
	c.DisableTime = false
	c.DisableKOs = false

}

func (c *Config) setProfilePlayer() {
	c.Profile = ProfilePlayer

	c.load = loadProfileAssetsPlayer

	c.DisableDefeated = false
	c.DisableEnergy = false
	c.DisableObjectives = false
	c.DisableScoring = false
	c.DisableTime = false
	c.DisableKOs = false
}

func Load(profile string) error {
	defer validate()

	if profile == "" {
		profile = ProfilePlayer
	}

	notify.System("Loading configuration from %s", Current.File())

	ok := open()
	if !ok {
		Current = Config{
			Window:             MainDisplay,
			VideoCaptureDevice: NoVideoCaptureDevice,
			Scale:              1,
			Shift:              Shift{},
			Profile:            profile,
			Acceptance:         .91,
		}
		Current.SetProfile()
		Current.SetDefaultAreas()
		Current.load()
	}

	if Current.Window == "" {
		Current.Window = MainDisplay
		Current.VideoCaptureDevice = NoVideoCaptureDevice
	}

	return Current.Save()
}

func TemplatesFirstRound(t1 []template.Template) []template.Template {
	t2 := []template.Template{}
	for _, t := range t1 {
		if t.Value == 0 {
			continue
		}
		t2 = append(t2, t)
	}
	return t2
}

func loadProfileAssetsBroadcaster() {
	Current.Filenames = map[string]map[string][]filter.Filter{
		"goals": {
			team.Game.Name: {
				filter.New(team.Game, "assets/profiles/broadcaster/game/purple_base_open.png", state.PurpleBaseOpen.Int(), false),
				filter.New(team.Game, "assets/profiles/broadcaster/game/orange_base_open.png", state.OrangeBaseOpen.Int(), false),
				filter.New(team.Game, "assets/profiles/broadcaster/game/purple_base_closed.png", state.PurpleBaseClosed.Int(), false),
				filter.New(team.Game, "assets/profiles/broadcaster/game/orange_base_closed.png", state.OrangeBaseClosed.Int(), false),
			},
		},
		"killed": {},
		"secure": {
			team.Game.Name: {
				filter.New(team.Game, "assets/profiles/broadcaster/game/regice_ally.png", state.RegiceSecureAlly.Int(), false),
				filter.New(team.Game, "assets/profiles/broadcaster/game/regice_enemy.png", state.RegiceSecureEnemy.Int(), false),
				filter.New(team.Game, "assets/profiles/broadcaster/game/regirock_ally.png", state.RegirockSecureAlly.Int(), false),
				filter.New(team.Game, "assets/profiles/broadcaster/game/regirock_enemy.png", state.RegirockSecureEnemy.Int(), false),
				filter.New(team.Game, "assets/profiles/broadcaster/game/registeel_ally.png", state.RegisteelSecureAlly.Int(), false),
				filter.New(team.Game, "assets/profiles/broadcaster/game/registeel_enemy.png", state.RegisteelSecureEnemy.Int(), false),
				filter.New(team.Game, "assets/profiles/broadcaster/game/regieleki_ally.png", state.RegielekiSecureAlly.Int(), false),
				filter.New(team.Game, "assets/profiles/broadcaster/game/regieleki_enemy.png", state.RegielekiSecureEnemy.Int(), false),
				filter.New(team.Game, "assets/profiles/broadcaster/game/rayquaza_ally.png", state.RayquazaSecureAlly.Int(), false),
				filter.New(team.Game, "assets/profiles/broadcaster/game/rayquaza_enemy.png", state.RayquazaSecureEnemy.Int(), false),
			},
		},
		"ko": {
			team.Game.Name: {
				filter.New(team.Game, "assets/profiles/broadcaster/game/ko_ally.png", state.KOAlly.Int(), false),
				filter.New(team.Game, "assets/profiles/broadcaster/game/ko_streak_ally.png", state.KOStreakAlly.Int(), false),
				filter.New(team.Game, "assets/profiles/broadcaster/game/ko_enemy.png", state.KOEnemy.Int(), false),
				filter.New(team.Game, "assets/profiles/broadcaster/game/ko_streak_enemy.png", state.KOStreakEnemy.Int(), false),
			},
		},
		"objective": {
			team.Game.Name: {
				filter.New(team.Game, "assets/profiles/broadcaster/game/objective.png", state.ObjectivePresent.Int(), false),
				filter.New(team.Game, "assets/profiles/broadcaster/game/objective_half.png", state.ObjectivePresent.Int(), false),
				filter.New(team.Game, "assets/profiles/broadcaster/game/objective_orange_base.png", state.ObjectiveReachedOrange.Int(), false),
			},
		},
		"game": {
			"vs": {
				filter.New(team.Game, "assets/profiles/broadcaster/game/vs.png", state.MatchStarting.Int(), false),
				filter.New(team.Game, "assets/profiles/broadcaster/game/vs_alt.png", state.MatchStarting.Int(), false),
			},
			"end": {
				filter.New(team.Game, "assets/profiles/broadcaster/game/end.png", state.MatchEnding.Int(), false),
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
	Current.Filenames = map[string]map[string][]filter.Filter{
		"goals": {
			team.Game.Name: {
				filter.New(team.Game, "assets/profiles/player/game/purple_base_open.png", state.PurpleBaseOpen.Int(), false),
				filter.New(team.Game, "assets/profiles/player/game/orange_base_open.png", state.OrangeBaseOpen.Int(), false),
				filter.New(team.Game, "assets/profiles/player/game/purple_base_closed.png", state.PurpleBaseClosed.Int(), false),
				filter.New(team.Game, "assets/profiles/player/game/orange_base_closed.png", state.OrangeBaseClosed.Int(), false),
			},
		},
		"killed": {
			team.Game.Name: {
				filter.New(team.Game, "assets/profiles/player/game/killed.png", state.Killed.Int(), false),
				filter.New(team.Game, "assets/profiles/player/game/killed_with_points.png", state.KilledWithPoints.Int(), false),
				filter.New(team.Game, "assets/profiles/player/game/killed_without_points.png", state.KilledWithoutPoints.Int(), false),
			},
		},
		"secure": {
			team.Game.Name: {
				filter.New(team.Game, "assets/profiles/player/game/regieleki_ally.png", state.RegielekiSecureAlly.Int(), false),
				filter.New(team.Game, "assets/profiles/player/game/regieleki_enemy.png", state.RegielekiSecureEnemy.Int(), false),
				filter.New(team.Game, "assets/profiles/player/game/regice_ally.png", state.RegiceSecureAlly.Int(), false),
				filter.New(team.Game, "assets/profiles/player/game/regice_enemy.png", state.RegiceSecureEnemy.Int(), false),
				filter.New(team.Game, "assets/profiles/player/game/regirock_ally.png", state.RegirockSecureAlly.Int(), false),
				filter.New(team.Game, "assets/profiles/player/game/regirock_enemy.png", state.RegirockSecureEnemy.Int(), false),
				filter.New(team.Game, "assets/profiles/player/game/registeel_ally.png", state.RegisteelSecureAlly.Int(), false),
				filter.New(team.Game, "assets/profiles/player/game/registeel_enemy.png", state.RegisteelSecureEnemy.Int(), false),
			},
		},
		"ko": {
			team.Game.Name: {
				filter.New(team.Game, "assets/profiles/player/game/ko_ally.png", state.KOAlly.Int(), false),
				filter.New(team.Game, "assets/profiles/player/game/ko_streak_ally.png", state.KOStreakAlly.Int(), false),
				filter.New(team.Game, "assets/profiles/player/game/ko_enemy.png", state.KOEnemy.Int(), false),
				filter.New(team.Game, "assets/profiles/player/game/ko_streak_enemy.png", state.KOStreakEnemy.Int(), false),
			},
		},
		"objective": {
			team.Game.Name: {
				filter.New(team.Game, "assets/profiles/player/game/objective.png", state.ObjectivePresent.Int(), false),
				filter.New(team.Game, "assets/profiles/player/game/objective_half.png", state.ObjectivePresent.Int(), false),
				filter.New(team.Game, "assets/profiles/player/game/objective_orange_base.png", state.ObjectiveReachedOrange.Int(), false),
			},
		},
		"game": {
			"vs": {
				filter.New(team.Game, "assets/profiles/player/game/vs.png", state.MatchStarting.Int(), false),
				filter.New(team.Game, "assets/profiles/player/game/vs_alt.png", state.MatchStarting.Int(), false),
			},
			"end": {
				filter.New(team.Game, "assets/profiles/player/game/end.png", state.MatchEnding.Int(), false),
			},
		},
		"scoring": {
			team.Game.Name: {
				filter.New(team.Game, "assets/profiles/player/game/pre_scoring_alt_alt.png", state.PreScore.Int(), false),
				filter.New(team.Game, "assets/profiles/player/game/pre_scoring_alt.png", state.PreScore.Int(), false),
				filter.New(team.Game, "assets/profiles/player/game/pre_scoring.png", state.PreScore.Int(), false),
				filter.New(team.Game, "assets/profiles/player/game/post_scoring.png", state.PostScore.Int(), false),
				filter.New(team.Game, "assets/profiles/player/game/press_button_to_score.png", state.PressButtonToScore.Int(), false),
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

	c.SetProfile()

	Current = c

	Current.load()

	return true
}

func validate() {
	Current.Templates = map[string]map[string][]template.Template{
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

	for category := range Current.Filenames {
		for subcategory, filters := range Current.Filenames[category] {
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

				Current.Templates[category][filter.Team.Name] = append(
					Current.Templates[category][filter.Team.Name],
					template,
				)
			}
		}
	}

	for category := range Current.Templates {
		for subcategory, templates := range Current.Templates[category] {
			for _, t := range templates {
				if t.Empty() {
					notify.Error("Failed to read %s/%s template from file \"%s\"", category, subcategory, t.File)
					continue
				}

				log.Debug().Str("category", category).Str("subcategory", subcategory).Object("template", t).Msg("template loaded")
			}
		}
	}
}
