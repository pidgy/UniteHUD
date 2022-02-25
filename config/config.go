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
	Stats            bool // Log image template match statistics.
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

func Load(acceptance float32, record, missed, dup, stats bool) error {
	defer validate()

	if open() {
		Current.Acceptance = acceptance
		Current.Record = record
		Current.RecordMissed = missed
		Current.RecordDuplicates = dup
		Current.Stats = stats
	} else {
		Current = Config{
			Acceptance:       acceptance,
			Record:           record,
			RecordMissed:     missed,
			RecordDuplicates: dup,
			Stats:            stats,
			Scores:           image.Rect(0, 0, 1100, 500),
			Time:             image.Rect(0, 0, 300, 200),
			Balls:            image.Rect(0, 0, 150, 100),
			Scales: Scales{
				Game:  1,
				Score: 1,
				Balls: 1,
				Time:  1,
			},
			Dir:  "default",
			load: loadDefault,
		}

		Current.load()
	}

	return Current.Save()
}

func validate() {
	Current.Templates = map[string]map[string][]template.Template{
		"game": {
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
						team.Balls.Name:
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

				template := template.New(filter, scaled, category, subcategory)
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

				log.Debug().Object("template", t).Msg("score template loaded")
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
		if !strings.Contains(file, ".png") && !strings.Contains(file, ".PNG") {
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
			notify.Feed(rgba.Yellow, "Invalid file in points directory: %s", file)
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

	err := os.Remove("config.unitehud")
	if err != nil {
		return err
	}

	return Load(Current.Acceptance, Current.Record, Current.RecordMissed, Current.RecordDuplicates, Current.Stats)
}

func loadDefault() {
	Current.Filenames = map[string]map[string][]filter.Filter{
		"game": {
			"vs": {
				filter.New(team.Game, "img/default/game/vs.png", 0, false),
			},
			"end": {
				filter.New(team.Game, "img/default/game/end.png", 0, false),
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
	b, err := os.ReadFile("config.unitehud")
	if err != nil {
		notify.Feed(rgba.DarkerYellow, "Creating default config.unitehud file. Select \"Configure\" from the main screen to customize your HUD.")
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
