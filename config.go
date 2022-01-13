package main

import (
	"fmt"
	"image"

	"github.com/rs/zerolog/log"
	"gocv.io/x/gocv"

	"github.com/pidgy/unitehud/team"
)

type config struct {
	scores       image.Rectangle
	time         image.Rectangle
	regularTime  [4]image.Rectangle
	finalStretch [4]image.Rectangle
	load         func()
}

type filter struct {
	*team.Team
	file  string
	value int
}

type template struct {
	filter
	gocv.Mat
	category    string
	subcategory string
}

var (
	filenames map[string]map[string][]filter
	templates map[string]map[string][]template

	configs = map[string]config{
		"default": {
			scores: image.Rect(640, 0, 1400, 500),
			time:   image.Rect(875, 0, 1025, 60),
			regularTime: [4]image.Rectangle{
				image.Rect(35, 15, 60, 50),
				image.Rect(55, 15, 75, 50),
				image.Rect(80, 15, 100, 50),
				image.Rect(95, 15, 120, 50),
			},
			finalStretch: [4]image.Rectangle{
				image.Rect(30, 20, 55, 60),
				image.Rect(54, 25, 72, 60),
				image.Rect(80, 20, 100, 60),
				image.Rect(104, 25, 122, 60),
			},
			load: loadSwitch,
		},
		"custom": {
			scores: image.Rect(480, 0, 1920, 1080),
			time:   image.Rect(1160, 15, 1228, 45),
			regularTime: [4]image.Rectangle{
				image.Rect(7, 0, 19, 20),
				image.Rect(19, 0, 31, 20),
				image.Rect(38, 0, 50, 20),
				image.Rect(50, 0, 62, 20),
			},
			finalStretch: [4]image.Rectangle{
				image.Rect(2, 7, 15, 29),
				image.Rect(17, 7, 30, 29),
				image.Rect(39, 7, 52, 29),
				image.Rect(54, 7, 67, 29),
			},
			load: loadIOS,
		},
	}
)

func load() {
	screen.load()

	for category := range filenames {
		for subcategory, filters := range filenames[category] {
			for _, filter := range filters {
				templates[category][filter.Team.Name] = append(templates[category][filter.Team.Name],
					template{
						filter,
						gocv.IMRead(filter.file, gocv.IMReadColor),
						category,
						subcategory,
					},
				)
			}
		}
	}

	for category := range templates {
		for _, templates := range templates[category] {
			for _, t := range templates {
				if t.Empty() {
					kill(fmt.Errorf("invalid scored template: %s", t.file))
				}

				log.Debug().Object("template", t).Msg("score template loaded")
			}
		}
	}
}

func loadSwitch() {
	filenames = map[string]map[string][]filter{
		"game": {
			"vs": {
				filter{team.None, "img/default/game/vs.png", -0},
			},
			"end": {
				filter{team.None, "img/default/game/end.png", -0},
			},
		},
		"scored": {
			team.Purple.Name: {
				filter{team.Purple, "img/default/purple/score/score.png", -0},
				filter{team.Purple, "img/default/purple/score/score_alt.png", -0},
			},
			team.Orange.Name: {
				filter{team.Orange, "img/default/orange/score/score.png", -0},
				filter{team.Orange, "img/default/orange/score/score_alt.png", -0},
			},
			team.Self.Name: {
				//filter{team.Self, "img/default/self/score/score.png", -0},
				filter{team.Self, "img/default/self/score/score_alt.png", -0},
				/*
					filter{team.Self, "img/default/self/score/score_alt_alt.png", -0},
					filter{team.Self, "img/default/self/score/score_alt_alt_alt.png", -0},
					filter{team.Self, "img/default/self/score/score_alt_alt_alt_alt.png", -0},
					filter{team.Self, "img/default/self/score/score_alt_alt.png", -0},
					filter{team.Self, "img/default/self/score/score_big_alt.png", -0},
				*/
			},
		},
		"points": {
			team.Purple.Name: {
				filter{team.Purple, "img/default/purple/points/point_0.png", 0},
				filter{team.Purple, "img/default/purple/points/point_0_alt.png", 0},
				filter{team.Purple, "img/default/purple/points/point_0_alt_alt.png", 0},
				filter{team.Purple, "img/default/purple/points/point_0_alt_alt_alt.png", 0},
				filter{team.Purple, "img/default/purple/points/point_0_alt_alt_alt_alt.png", 0},
				filter{team.Purple, "img/default/purple/points/point_0_alt_alt_alt_alt_alt.png", 0},
				filter{team.Purple, "img/default/purple/points/point_0_alt_alt_alt_alt_alt_alt.png", 0},

				filter{team.Purple, "img/default/purple/points/point_0_big.png", 0},
				filter{team.Purple, "img/default/purple/points/point_0_big_alt.png", 0},
				filter{team.Purple, "img/default/purple/points/point_0_big_alt_alt.png", 0},
				filter{team.Purple, "img/default/purple/points/point_0_big_alt_alt_alt.png", 0},
				filter{team.Purple, "img/default/purple/points/point_0_big_alt_alt_alt_alt.png", 0},

				filter{team.Purple, "img/default/purple/points/point_1.png", 1},
				filter{team.Purple, "img/default/purple/points/point_1_alt.png", 1},
				filter{team.Purple, "img/default/purple/points/point_1_alt_alt.png", 1},
				filter{team.Purple, "img/default/purple/points/point_1_big.png", 1},
				filter{team.Purple, "img/default/purple/points/point_1_big_alt.png", 1},
				filter{team.Purple, "img/default/purple/points/point_1_big_alt_alt.png", 1},

				filter{team.Purple, "img/default/purple/points/point_2.png", 2},
				filter{team.Purple, "img/default/purple/points/point_2_alt.png", 2},
				filter{team.Purple, "img/default/purple/points/point_2_alt_alt.png", 2},
				filter{team.Purple, "img/default/purple/points/point_2_alt_alt_alt.png", 2},
				filter{team.Purple, "img/default/purple/points/point_2_big_alt.png", 2},

				filter{team.Purple, "img/default/purple/points/point_3.png", 3},
				filter{team.Purple, "img/default/purple/points/point_3_alt.png", 3},

				filter{team.Purple, "img/default/purple/points/point_4.png", 4},
				filter{team.Purple, "img/default/purple/points/point_4_alt.png", 4},
				filter{team.Purple, "img/default/purple/points/point_4_alt_alt.png", 4},
				filter{team.Purple, "img/default/purple/points/point_4_big.png", 4},
				filter{team.Purple, "img/default/purple/points/point_4_big_alt.png", 4},
				filter{team.Purple, "img/default/purple/points/point_4_big_alt_alt.png", 4},
				filter{team.Purple, "img/default/purple/points/point_4_big_alt_alt_alt.png", 4},

				filter{team.Purple, "img/default/purple/points/point_5_alt.png", 5},
				filter{team.Purple, "img/default/purple/points/point_5_big.png", 5},

				filter{team.Purple, "img/default/purple/points/point_6.png", 6},
				filter{team.Purple, "img/default/purple/points/point_6_alt.png", 6},
				filter{team.Purple, "img/default/purple/points/point_6_big.png", 6},
				filter{team.Purple, "img/default/purple/points/point_6_big_alt.png", 6},

				filter{team.Purple, "img/default/purple/points/point_7.png", 7},
				filter{team.Purple, "img/default/purple/points/point_7_big.png", 7},

				filter{team.Purple, "img/default/purple/points/point_8.png", 8},
				filter{team.Purple, "img/default/purple/points/point_8_big.png", 8},
				filter{team.Purple, "img/default/purple/points/point_8_big_alt.png", 8},
				filter{team.Purple, "img/default/purple/points/point_8_big_alt_alt.png", 8},

				filter{team.Purple, "img/default/purple/points/point_9.png", 9},
				filter{team.Purple, "img/default/purple/points/point_9_alt.png", 9},
				filter{team.Purple, "img/default/purple/points/point_9_big.png", 9},
			},
			team.Orange.Name: {
				filter{team.Orange, "img/default/orange/points/point_0.png", 0},
				filter{team.Orange, "img/default/orange/points/point_0_alt.png", 0},
				filter{team.Orange, "img/default/orange/points/point_0_big.png", 0},
				filter{team.Orange, "img/default/orange/points/point_0_big_alt.png", 0},
				filter{team.Orange, "img/default/orange/points/point_0_big_alt_alt.png", 0},
				filter{team.Orange, "img/default/orange/points/point_0_big_alt_alt_alt.png", 0},
				filter{team.Orange, "img/default/orange/points/point_0_big_alt_alt_alt_alt.png", 0},

				filter{team.Orange, "img/default/orange/points/point_1.png", 1},
				filter{team.Orange, "img/default/orange/points/point_1_alt.png", 1},
				filter{team.Orange, "img/default/orange/points/point_1_big.png", 1},
				filter{team.Orange, "img/default/orange/points/point_1_big_alt.png", 1},

				filter{team.Orange, "img/default/orange/points/point_2.png", 2},
				filter{team.Orange, "img/default/orange/points/point_2_alt.png", 2},
				filter{team.Orange, "img/default/orange/points/point_2_big_alt.png", 2},

				filter{team.Orange, "img/default/orange/points/point_3.png", 3},
				filter{team.Orange, "img/default/orange/points/point_3_alt.png", 3},

				filter{team.Orange, "img/default/orange/points/point_4.png", 4},
				filter{team.Orange, "img/default/orange/points/point_4_alt.png", 4},
				filter{team.Orange, "img/default/orange/points/point_4_alt_alt.png", 4},
				filter{team.Orange, "img/default/orange/points/point_4_alt_alt_alt.png", 4},
				filter{team.Orange, "img/default/orange/points/point_4_big_alt.png", 4},

				filter{team.Orange, "img/default/orange/points/point_5.png", 5},
				filter{team.Orange, "img/default/orange/points/point_5_alt.png", 5},

				filter{team.Orange, "img/default/orange/points/point_6.png", 6},
				filter{team.Orange, "img/default/orange/points/point_6_alt.png", 6},
				filter{team.Orange, "img/default/orange/points/point_6_alt_alt.png", 6},
				filter{team.Orange, "img/default/orange/points/point_6_big_alt.png", 6},
				filter{team.Orange, "img/default/orange/points/point_6_big_alt_alt.png", 6},

				filter{team.Orange, "img/default/orange/points/point_7.png", 7},
				filter{team.Orange, "img/default/orange/points/point_7_big.png", 7},

				filter{team.Orange, "img/default/orange/points/point_8.png", 8},
				filter{team.Orange, "img/default/orange/points/point_8_alt.png", 8},
				filter{team.Orange, "img/default/orange/points/point_8_alt_alt.png", 8},
				filter{team.Orange, "img/default/orange/points/point_8_big_alt.png", 8},

				filter{team.Orange, "img/default/orange/points/point_9.png", 9},
				filter{team.Orange, "img/default/orange/points/point_9_alt.png", 9},
				filter{team.Orange, "img/default/orange/points/point_9_big.png", 9},
			},
			team.Self.Name: {
				filter{team.Self, "img/default/self/points/point_0.png", 0},
				filter{team.Self, "img/default/self/points/point_0_alt.png", 0},
				filter{team.Self, "img/default/self/points/point_0_alt_alt.png", 0},
				filter{team.Self, "img/default/self/points/point_0_alt_alt_alt.png", 0},
				filter{team.Self, "img/default/self/points/point_1.png", 1},
				filter{team.Self, "img/default/self/points/point_1_alt.png", 1},
				filter{team.Self, "img/default/self/points/point_2.png", 2},
				filter{team.Self, "img/default/self/points/point_2_alt.png", 2},
				filter{team.Self, "img/default/self/points/point_5.png", 5},
				filter{team.Self, "img/default/self/points/point_5_alt.png", 5},
				filter{team.Self, "img/default/self/points/point_5_alt_alt.png", 5},
				filter{team.Self, "img/default/self/points/point_5_alt_alt_alt.png", 5},
				filter{team.Self, "img/default/self/points/point_5_alt_alt_alt_alt.png", 5},
				filter{team.Self, "img/default/self/points/point_6.png", 6},
				filter{team.Self, "img/default/self/points/point_6_alt.png", 6},
				filter{team.Self, "img/default/self/points/point_7.png", 7},
				filter{team.Self, "img/default/self/points/point_7_alt.png", 7},
				filter{team.Self, "img/default/self/points/point_7_alt_alt.png", 7},
				filter{team.Self, "img/default/self/points/point_8_alt.png", 8},
			},
		},
		"time": {
			team.Time.Name: {
				filter{team.Time, "img/default/time/points/point_0.png", 0},
				filter{team.Time, "img/default/time/points/point_0_alt.png", 0},
				filter{team.Time, "img/default/time/points/point_1.png", 1},
				filter{team.Time, "img/default/time/points/point_1_alt.png", 1},
				filter{team.Time, "img/default/time/points/point_2.png", 2},
				filter{team.Time, "img/default/time/points/point_2_alt.png", 2},
				filter{team.Time, "img/default/time/points/point_3.png", 3},
				filter{team.Time, "img/default/time/points/point_3_alt.png", 3},
				filter{team.Time, "img/default/time/points/point_4.png", 4},
				filter{team.Time, "img/default/time/points/point_4_alt.png", 4},
				filter{team.Time, "img/default/time/points/point_5.png", 5},
				filter{team.Time, "img/default/time/points/point_5_alt.png", 5},
				filter{team.Time, "img/default/time/points/point_6.png", 6},
				filter{team.Time, "img/default/time/points/point_6_alt.png", 6},
				filter{team.Time, "img/default/time/points/point_7.png", 7},
				filter{team.Time, "img/default/time/points/point_7_alt.png", 7},
				filter{team.Time, "img/default/time/points/point_8.png", 8},
				filter{team.Time, "img/default/time/points/point_8_alt.png", 8},
				filter{team.Time, "img/default/time/points/point_9.png", 9},
				filter{team.Time, "img/default/time/points/point_9_alt.png", 9},
			},
		},
	}

	templates = map[string]map[string][]template{
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

func loadIOS() {
	filenames = map[string]map[string][]filter{
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
				filter{team.Time, "img/ios/time/points/point_0.png", 0},
				filter{team.Time, "img/ios/time/points/point_1.png", 1},
				filter{team.Time, "img/ios/time/points/point_2.png", 2},
				filter{team.Time, "img/ios/time/points/point_3.png", 3},
				filter{team.Time, "img/ios/time/points/point_4.png", 4},
				filter{team.Time, "img/ios/time/points/point_5.png", 5},
				filter{team.Time, "img/ios/time/points/point_6.png", 6},
				filter{team.Time, "img/ios/time/points/point_7.png", 7},
				filter{team.Time, "img/ios/time/points/point_8.png", 8},
				filter{team.Time, "img/ios/time/points/point_9.png", 9},
			},
		},
	}

	templates = map[string]map[string][]template{
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
