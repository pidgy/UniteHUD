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

var File = strings.ReplaceAll(global.Version, ".", "-") + "-config.unitehud"

const (
	MainDisplay          = "Main Display"
	NoVideoCaptureDevice = -1
)

type Config struct {
	Window             string
	VideoCaptureDevice int
	LostWindow         string `json:"-"`
	Record             bool   `json:"-"` // Record all matched images and logs.
	Balls              image.Rectangle
	Map                image.Rectangle
	Scores             image.Rectangle
	Time               image.Rectangle
	Filenames          map[string]map[string][]filter.Filter     `json:"-"`
	Templates          map[string]map[string][]template.Template `json:"-"`
	Scale              float64
	Shift              Shift
	Dir                string
	Acceptance         float32

	Crashed string

	load func()
}

type Shift struct {
	N, E, S, W int
}

var Current Config

func (c Config) Reload() {
	defer validate()
}

func (c *Config) Report(crash string) {
	c.Crashed = crash

	err := c.Save()
	if err != nil {
		log.Panic().Err(err).Msg("failed to save crash report")
	}
}
func (c Config) Save() error {
	f, err := os.Create(File)
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

func (c Config) Scoring() image.Rectangle {
	return image.Rectangle{
		Min: image.Pt(c.Balls.Min.X-50, c.Balls.Min.Y),
		Max: image.Pt(c.Balls.Max.X+50, c.Balls.Max.Y+100),
	}
}

func (c Config) ScoringOption() image.Rectangle {
	return image.Rectangle{
		Min: image.Pt(c.Balls.Min.X-100, c.Balls.Min.Y-100),
		Max: image.Pt(c.Balls.Max.X+100, c.Balls.Max.Y-100),
	}
}

func (c *Config) SetDefaultAreas() {
	c.Balls = image.Rect(910, 756, 1010, 856)
	c.Map = image.Rect(70, 100, 470, 250)
	c.Scores = image.Rect(500, 50, 1500, 250)
	c.Time = image.Rect(846, 0, 1046, 100)
}

func Load() error {
	defer validate()

	ok := open()
	if !ok {
		Current = Config{
			Window:             MainDisplay,
			VideoCaptureDevice: NoVideoCaptureDevice,
			Scale:              1,
			Shift:              Shift{},
			Dir:                "default",
			load:               loadDefault,
			Acceptance:         .91,
		}
		Current.SetDefaultAreas()
		Current.load()
	} else {
		Current.Acceptance = .91
	}

	if Current.Window == "" {
		Current.Window = MainDisplay
		Current.VideoCaptureDevice = NoVideoCaptureDevice
	}

	return Current.Save()
}

func validate() {
	Current.Templates = map[string]map[string][]template.Template{
		"goals": {
			team.Game.Name: {},
		},
		"game": {
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
			team.Balls.Name:  {},
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
						team.Balls.Name,
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

func (c Config) scoreFiles(t *team.Team) []filter.Filter {
	var files []string

	root := fmt.Sprintf("img/%s/%s/score/", c.Dir, t.Name)
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			if info.Name() != "score" {
				notify.SystemWarn("Skipping \"%s%s\"", root, info.Name())
				return filepath.SkipDir
			}
			notify.System("Loading templates from %s", path)
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

func (c Config) pointFiles(t *team.Team) []filter.Filter {
	var files []string

	root := fmt.Sprintf("img/%s/%s/points/", c.Dir, t.Name)
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			if info.Name() != "points" {
				notify.SystemWarn("Skipping templates from %s%s", root, info.Name())
				return filepath.SkipDir
			}
			notify.System("Loading templates from %s", path)
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

func Reset() error {
	defer validate()

	err := os.Remove(File)
	if err != nil {
		return err
	}

	return Load()
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

func loadDefault() {
	Current.Filenames = map[string]map[string][]filter.Filter{
		"goals": {
			team.Game.Name: {
				filter.New(team.Game, "img/default/game/purple_base_open.png", state.PurpleBaseOpen.Int(), false),
				filter.New(team.Game, "img/default/game/orange_base_open.png", state.OrangeBaseOpen.Int(), false),
				filter.New(team.Game, "img/default/game/purple_base_closed.png", state.PurpleBaseClosed.Int(), false),
				filter.New(team.Game, "img/default/game/orange_base_closed.png", state.OrangeBaseClosed.Int(), false),
			},
		},
		"killed": {
			team.Game.Name: {
				filter.New(team.Game, "img/default/game/killed.png", state.Killed.Int(), false),
				filter.New(team.Game, "img/default/game/killed_with_points.png", state.KilledWithPoints.Int(), false),
				filter.New(team.Game, "img/default/game/killed_without_points.png", state.KilledWithoutPoints.Int(), false),
			},
		},
		"secure": {
			team.Game.Name: {
				filter.New(team.Game, "img/default/game/regieleki_ally.png", state.RegielekiAdvancingAlly.Int(), false),
				filter.New(team.Game, "img/default/game/regieleki_enemy.png", state.RegielekiAdvancingEnemy.Int(), false),
			},
		},
		"objective": {
			team.Game.Name: {
				filter.New(team.Game, "img/default/game/objective.png", state.ObjectivePresent.Int(), false),
				filter.New(team.Game, "img/default/game/objective_half.png", state.ObjectivePresent.Int(), false),
				filter.New(team.Game, "img/default/game/objective_orange_base.png", state.ObjectiveReachedOrange.Int(), false),
			},
		},
		"game": {
			"vs": {
				filter.New(team.Game, "img/default/game/vs.png", state.MatchStarting.Int(), false),
			},
			"end": {
				filter.New(team.Game, "img/default/game/end.png", state.MatchEnding.Int(), false),
			},
		},
		"scoring": {
			team.Game.Name: {
				filter.New(team.Game, "img/default/game/pre_scoring_alt_alt.png", state.PreScore.Int(), false),
				filter.New(team.Game, "img/default/game/pre_scoring_alt.png", state.PreScore.Int(), false),
				filter.New(team.Game, "img/default/game/pre_scoring.png", state.PreScore.Int(), false),
				filter.New(team.Game, "img/default/game/post_scoring.png", state.PostScore.Int(), false),
				filter.New(team.Game, "img/default/game/press_button_to_score.png", state.PressButtonToScore.Int(), false),
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
			team.Balls.Name:  Current.pointFiles(team.Balls),
		},
		"time": {
			team.Time.Name: Current.pointFiles(team.Time),
		},
	}
}

func open() bool {
	b, err := os.ReadFile(File)
	if err != nil {
		return false
	}

	c := Config{
		load: loadDefault,
	}

	err = json.Unmarshal(b, &c)
	if err != nil {
		return false
	}

	Current = c
	defer Current.load()

	return true
}
