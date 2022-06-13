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
	"github.com/pidgy/unitehud/state"
	"github.com/pidgy/unitehud/team"
	"github.com/pidgy/unitehud/template"
)

const (
	File        = "config.unitehud"
	MainDisplay = "Main Display"
)

type Config struct {
	Window     string
	Record     bool `json:"-"` // Record all matched images and logs.
	Balls      image.Rectangle
	Map        image.Rectangle
	Scores     image.Rectangle
	Time       image.Rectangle
	Filenames  map[string]map[string][]filter.Filter     `json:"-"`
	Templates  map[string]map[string][]template.Template `json:"-"`
	Scales     Scales
	Dir        string
	Acceptance float32
	load       func()
}

type Scales struct {
	Score float64
	Time  float64
	Balls float64
	Game  float64
}

var Current Config

func (s Scales) Is16x9() bool {
	return s.Balls+s.Game+s.Score+s.Time == 4
}

func (s *Scales) To16x9() {
	s.Balls, s.Game, s.Score, s.Time = 1, 1, 1, 1
}

func (s Scales) Is4x3() bool {
	return s.Balls == 0.4 && s.Game == 0.4 && s.Score == 0.4 && s.Time == 0.4
}

func (s *Scales) To4x3() {
	s.Balls, s.Game, s.Score, s.Time = 0.4, 0.4, 0.4, 0.4
}

func (c Config) Reload() {
	defer validate()
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

func Load() error {
	defer validate()

	if open() {
		Current.Record = Current.Record
		Current.Acceptance = .91
	} else {
		Current = Config{
			Window: MainDisplay,
			Balls:  image.Rect(0, 0, 100, 100), // Must be square (H==W).
			Map:    image.Rect(0, 0, 500, 460),
			Scores: image.Rect(0, 0, 1100, 500),
			Time:   image.Rect(0, 0, 200, 100),
			Scales: Scales{
				Game:  1,
				Score: 1,
				Balls: 1,
				Time:  1,
			},
			Dir:        "default",
			load:       loadDefault,
			Acceptance: .91,
		}
		Current.load()
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
				scaled := gocv.NewMat()

				transparent := false

				scale := float64(1)
				switch category {
				case "points":
					scale = Current.Scales.Score

					switch filter.Team.Name {
					case team.First.Name,
						team.Self.Name,
						team.Orange.Name,
						team.Purple.Name,
						team.Balls.Name,
						team.Game.Name:
						transparent = true
					}
				case "scored":
					scale = Current.Scales.Score
				case "balls":
					scale = Current.Scales.Balls
				case "time":
					scale = Current.Scales.Time
				case "game":
					scale = Current.Scales.Game
				}

				gocv.Resize(mat, &scaled, image.Pt(0, 0), scale, scale, gocv.InterpolationDefault)

				template := template.New(filter, scaled, category, subcategory, scale)
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
		for _, templates := range Current.Templates[category] {
			for _, t := range templates {
				if t.Empty() {
					log.Fatal().Msgf("invalid scored template: %s", t.File)
				}

				log.Debug().Object("template", t).Msg("template loaded")
			}
		}
	}
}

func (c Config) scoreFiles(t *team.Team) []filter.Filter {
	var files []string

	root := fmt.Sprintf("img/%s/%s/score/", c.Dir, t.Name)
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		files = append(files, path)
		return nil
	})
	if err != nil {
		log.Fatal().Err(err).Str("root", root).Msg("invalid point image path")
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
		files = append(files, path)
		return nil
	})
	if err != nil {
		log.Fatal().Err(err).Str("root", root).Msg("invalid point image path")
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
			log.Fatal().Err(err).Str("root", root).Msg("invalid point image filename")
			return nil
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
				filter.New(team.Game, "img/default/game/pre_scoring.png", state.PreScore.Int(), false),
				filter.New(team.Game, "img/default/game/post_scoring.png", state.PostScore.Int(), false),
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
		log.Warn().Err(err).Msg("previously saved config does not exist")
		return false
	}

	c := Config{
		load: loadDefault,
	}

	err = json.Unmarshal(b, &c)
	if err != nil {
		log.Error().Err(err).Msg("failed to parse previously saved config")
		return false
	}

	Current = c
	defer Current.load()

	return true
}
