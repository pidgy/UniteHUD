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
	"github.com/pidgy/unitehud/notify"
	"github.com/pidgy/unitehud/rgba"
	"github.com/pidgy/unitehud/team"
	"github.com/pidgy/unitehud/template"
)

type Config struct {
	Acceptance       float32
	Record           bool // Record all matched images and logs.
	RecordMissed     bool // Record missed images.
	RecordDuplicates bool // Record duplicate matched images.
	Scores           image.Rectangle
	Time             image.Rectangle
	Balls            image.Rectangle
	Filenames        map[string]map[string][]filter.Filter     `json:"-"`
	Templates        map[string]map[string][]template.Template `json:"-"`
	Scales           Scales
	Dir              string

	load func()
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

func Reset() error {
	defer validate()

	os.Remove("config.unitehud")

	return Load("default", Current.Acceptance, Current.Record, Current.RecordMissed, Current.RecordDuplicates)
}

func (c Config) Save() error {
	f, err := os.Create("config.unitehud")
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

func Load(dir string, acceptance float32, record, missed, dup bool) error {
	defer validate()

	if open(dir) {
		Current.Acceptance = acceptance
		Current.Dir = dir
		Current.Record = record
		Current.RecordMissed = missed
		Current.RecordDuplicates = dup
		return Current.Save()
	}

	configs := map[string]Config{
		"default": {
			Acceptance:   acceptance,
			Record:       record,
			RecordMissed: missed,
			Scores:       image.Rect(400, 0, 1300, 300),
			Time:         image.Rect(800, 0, 1100, 200),
			Balls:        image.Rect(0, 0, 200, 200),
			Scales: Scales{
				Game:  1,
				Score: 1,
				Balls: 1,
				Time:  1,
			},
			Dir:  "default",
			load: loadDefault,
		},
		"custom": {
			Acceptance:   acceptance,
			Record:       record,
			RecordMissed: missed,
			Scores:       image.Rect(400, 0, 1100, 400),
			Time:         image.Rect(800, 0, 1100, 150),
			Balls:        image.Rect(0, 0, 200, 200),
			Scales: Scales{
				Game:  1,
				Score: 1,
				Balls: 1,
				Time:  1,
			},
			Dir:  "custom",
			load: loadCustom,
		},
	}

	c, ok := configs[dir]
	if !ok {
		return fmt.Errorf("unknown configuration: %s", dir)
	}

	Current = c
	Current.load()

	return Current.Save()
}

func validate() {
	Current.Templates = map[string]map[string][]template.Template{
		"game": {
			team.None.Name: {},
		},
		"scored": {
			team.Orange.Name: {},
			team.Purple.Name: {},
			team.Self.Name:   {},
		},
		"points": {
			team.Orange.Name: {},
			team.Purple.Name: {},
			team.Self.Name:   {},
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

				scale := float64(1)
				switch category {
				case "scored", "points":
					scale = Current.Scales.Score
				case "balls":
					scale = Current.Scales.Balls
				case "time":
					scale = Current.Scales.Time
				case "game":
					scale = Current.Scales.Game
				}

				gocv.Resize(mat, &scaled, image.Pt(0, 0), scale, scale, gocv.InterpolationDefault)

				Current.Templates[category][filter.Team.Name] = append(Current.Templates[category][filter.Team.Name],
					template.Template{
						Filter:      filter,
						Mat:         scaled,
						Category:    category,
						Subcategory: subcategory,
					},
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

				log.Debug().Object("template", t).Msg("score template loaded")
			}
		}
	}
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
		if !strings.Contains(file, ".png") {
			continue
		}
		b := strings.Split(file, "point_")
		if len(b) != 2 {
			continue
		}

		v := strings.ReplaceAll(
			strings.ReplaceAll(
				strings.ReplaceAll(
					b[1],
					".png",
					"",
				),
				"_big",
				"",
			),
			"_alt",
			"",
		)
		if v == "" {
			log.Warn().Str("file", file).Msg("invalid file in points directory")
			notify.Feed(rgba.Yellow, "Invalid file in points directory: %s", file)
			continue
		}

		value, err := strconv.Atoi(v)
		if err != nil {
			log.Fatal().Err(err).Str("root", root).Msg("invalid point image filename")
			return nil
		}

		filters = append(filters, filter.Filter{
			Team:  t,
			File:  file,
			Value: value,
		})
	}

	return filters
}

func loadDefault() {
	Current.Filenames = map[string]map[string][]filter.Filter{
		"game": {
			"vs": {
				filter.Filter{team.None, "img/default/game/vs.png", -0},
			},
			"end": {
				filter.Filter{team.None, "img/default/game/end.png", -0},
			},
		},
		"scored": {
			team.Purple.Name: {
				filter.Filter{team.Purple, "img/default/purple/score/score.png", -0},
				filter.Filter{team.Purple, "img/default/purple/score/score_alt.png", -0},
			},
			team.Orange.Name: {
				filter.Filter{team.Orange, "img/default/orange/score/score.png", -0},
				filter.Filter{team.Orange, "img/default/orange/score/score_alt.png", -0},
			},
			team.Self.Name: {
				//filter.Filter{team.Self, "img/default/self/score/score.png", -0},
				filter.Filter{team.Self, "img/default/self/score/score_alt.png", -0},
				/*
					filter.Filter{team.Self, "img/default/self/score/score_alt_alt.png", -0},
					filter.Filter{team.Self, "img/default/self/score/score_alt_alt_alt.png", -0},
					filter.Filter{team.Self, "img/default/self/score/score_alt_alt_alt_alt.png", -0},
					filter.Filter{team.Self, "img/default/self/score/score_alt_alt.png", -0},
					filter.Filter{team.Self, "img/default/self/score/score_big_alt.png", -0},
				*/
			},
		},
		"points": {
			team.Purple.Name: Current.pointFiles(team.Purple),
			team.Orange.Name: Current.pointFiles(team.Orange),
			team.Self.Name:   Current.pointFiles(team.Self),
		},
		"time": {
			team.Time.Name: Current.pointFiles(team.Time),
		},
	}
}

func loadCustom() {
	Current.Filenames = map[string]map[string][]filter.Filter{
		"game": {
			"vs":  {},
			"end": {},
		},
		"scored": {
			team.Purple.Name: {},
			team.Orange.Name: {},
			team.Self.Name:   {},
		},
		"points": {
			team.Purple.Name: {},
			team.Orange.Name: {},
			team.Self.Name:   {},
		},
		"time": {
			team.Time.Name: {
				filter.Filter{team.Time, "img/custom/time/points/point_0.png", 0},
				filter.Filter{team.Time, "img/custom/time/points/point_0_alt.png", 0},

				filter.Filter{team.Time, "img/custom/time/points/point_1.png", 1},
				filter.Filter{team.Time, "img/custom/time/points/point_1_alt.png", 1},

				filter.Filter{team.Time, "img/custom/time/points/point_2.png", 2},
				filter.Filter{team.Time, "img/custom/time/points/point_2_alt.png", 2},

				filter.Filter{team.Time, "img/custom/time/points/point_3.png", 3},
				filter.Filter{team.Time, "img/custom/time/points/point_3_alt.png", 3},

				filter.Filter{team.Time, "img/custom/time/points/point_4.png", 4},
				filter.Filter{team.Time, "img/custom/time/points/point_4_alt.png", 4},

				filter.Filter{team.Time, "img/custom/time/points/point_5.png", 5},
				filter.Filter{team.Time, "img/custom/time/points/point_5_alt.png", 5},

				filter.Filter{team.Time, "img/custom/time/points/point_6.png", 6},
				filter.Filter{team.Time, "img/custom/time/points/point_6_alt.png", 6},

				filter.Filter{team.Time, "img/custom/time/points/point_7.png", 7},
				filter.Filter{team.Time, "img/custom/time/points/point_7_alt.png", 7},

				filter.Filter{team.Time, "img/custom/time/points/point_8.png", 8},
				filter.Filter{team.Time, "img/custom/time/points/point_8_alt.png", 8},

				filter.Filter{team.Time, "img/custom/time/points/point_9.png", 9},
				filter.Filter{team.Time, "img/custom/time/points/point_9_alt.png", 9},
			},
		},
	}

	Current.Templates = map[string]map[string][]template.Template{
		"game": {
			team.None.Name: {},
		},
		"scored": {
			team.Orange.Name: {},
			team.Purple.Name: {},
			team.Self.Name:   {},
		},
		"points": {
			team.Orange.Name: {},
			team.Purple.Name: {},
			team.Self.Name:   {},
		},
		"time": {
			team.Time.Name: {},
		},
	}
}

func open(dir string) bool {
	b, err := os.ReadFile("config.unitehud")
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
	if Current.Dir == "custom" {
		Current.load = loadCustom
	}
	Current.load()

	return true
}
