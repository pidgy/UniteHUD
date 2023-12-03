package ui

import (
	"image"
	"strings"
	"time"

	"gioui.org/unit"
	"gioui.org/widget"

	"github.com/pidgy/unitehud/core/config"
	"github.com/pidgy/unitehud/core/fonts"
	"github.com/pidgy/unitehud/core/match"
	"github.com/pidgy/unitehud/core/notify"
	"github.com/pidgy/unitehud/core/team"
	"github.com/pidgy/unitehud/gui/visual/area"
	"github.com/pidgy/unitehud/gui/visual/dropdown"
	"github.com/pidgy/unitehud/media/audio"
	"github.com/pidgy/unitehud/media/video"
	"github.com/pidgy/unitehud/media/video/device"
)

type areas struct {
	energy    *area.Widget
	ko        *area.Widget
	objective *area.Widget
	score     *area.Widget
	state     *area.Widget
	time      *area.Widget

	onevent func()
}

type audios struct {
	in  capture
	out capture
}

type capture struct {
	list     *dropdown.Widget
	populate func(bool)
	len      int
}

type videos struct {
	device   capture
	window   capture
	monitor  capture
	platform capture
	profile  capture
	apis     capture

	onevent func()
}

func (g *GUI) audios(text float32, session *audio.Session) *audios {
	a := &audios{
		in: capture{
			list: &dropdown.Widget{
				Theme:         g.header.Collection.Calibri().Theme,
				WidthModifier: 1,
				TextSize:      text,
				Items: []*dropdown.Item{
					{
						Text:    audio.Disabled,
						Checked: widget.Bool{Value: true},
						Callback: func(i *dropdown.Item) {
							err := session.Input(audio.Disabled)
							if err != nil {
								g.ToastError(err)
								return
							}
							i.Checked.Value = true
						},
					},
					{
						Text: audio.Default,
						Callback: func(i *dropdown.Item) {
							err := session.Input(audio.Default)
							if err != nil {
								g.ToastError(err)
								return
							}
							i.Checked.Value = true
						},
					},
				},
				Callback: func(i *dropdown.Item, d *dropdown.Widget) bool {
					for _, item := range d.Items {
						item.Checked.Value = false
						if item == i {
							item.Checked.Value = true
						}
					}
					return true
				},
			},
		},
		out: capture{
			list: &dropdown.Widget{
				Theme:         g.header.Collection.Calibri().Theme,
				WidthModifier: 1,
				TextSize:      text,
				Items: []*dropdown.Item{
					{
						Text: audio.Disabled,
						Callback: func(i *dropdown.Item) {
							err := session.Output(audio.Disabled)
							if err != nil {
								g.ToastError(err)
								return
							}
							i.Checked.Value = true
						},
					},
					{
						Text:    audio.Default,
						Checked: widget.Bool{Value: true},
						Callback: func(i *dropdown.Item) {
							err := session.Output(audio.Default)
							if err != nil {
								g.ToastError(err)
								return
							}
							i.Checked.Value = true
						},
					},
				},
				Callback: func(i *dropdown.Item, d *dropdown.Widget) bool {
					for _, item := range d.Items {
						item.Checked.Value = false
						if item == i {
							item.Checked.Value = true
						}
					}
					return true
				},
			},
		},
	}

	for _, d := range session.Inputs() {
		if d.Is(device.ActiveName()) {
			err := session.Input(d.Name())
			if err != nil {
				g.ToastError(err)
			}

			a.in.list.Enabled()
		}

		a.in.list.Items = append(a.in.list.Items, &dropdown.Item{
			Text:    d.Name(),
			Checked: widget.Bool{Value: d.Is(device.ActiveName())},
			Callback: func(i *dropdown.Item) {
				err := session.Input(i.Text)
				if err != nil {
					g.ToastError(err)
					return
				}

				i.Checked.Value = true

				err = session.Start()
				if err != nil {
					g.ToastError(err)
					return
				}
			},
		})
	}

	for _, d := range session.Outputs() {
		a.out.list.Items = append(a.out.list.Items, &dropdown.Item{
			Text:    d.Name(),
			Checked: widget.Bool{Value: d.Is(device.ActiveName())},
			Callback: func(i *dropdown.Item) {
				err := session.Output(i.Text)
				if err != nil {
					g.ToastError(err)
					return
				}

				i.Checked.Value = true

				err = session.Start()
				if err != nil {
					g.ToastError(err)
					return
				}
			},
		})
	}

	return a
}

func (g *GUI) areas(collection fonts.Collection) *areas {
	a := &areas{
		onevent: func() { /*No-op.*/ },
	}

	a.ko = &area.Widget{
		Text:     "KO",
		TextSize: unit.Sp(13),
		Theme:    collection.Calibri().Theme,
		Min:      config.Current.KOs.Min,
		Max:      config.Current.KOs.Max,
		NRGBA:    area.Locked,
		Match:    g.matchKOs,
		Cooldown: time.Millisecond * 1500,

		Capture: &area.Capture{
			Option:      "KO",
			File:        "ko_area.png",
			Base:        config.Current.KOs,
			DefaultBase: config.Current.KOs,
		},
	}

	a.objective = &area.Widget{
		Text:     "Objectives",
		TextSize: unit.Sp(13),
		Theme:    collection.Calibri().Theme,
		Min:      config.Current.Objectives.Min,
		Max:      config.Current.Objectives.Max,
		NRGBA:    area.Locked,
		Match:    g.matchObjectives,
		Cooldown: time.Second,

		Capture: &area.Capture{
			Option:      "Objective",
			File:        "objective_area.png",
			Base:        config.Current.Objectives,
			DefaultBase: config.Current.Objectives,
		},
	}

	a.energy = &area.Widget{
		Text:     "Aeos",
		TextSize: unit.Sp(13),
		Theme:    collection.Calibri().Theme,
		Min:      config.Current.Energy.Min,
		Max:      config.Current.Energy.Max,
		NRGBA:    area.Locked,
		Match:    g.matchEnergy,
		Cooldown: team.Energy.Delay,

		Capture: &area.Capture{
			Option:      "Aeos",
			File:        "aeos_area.png",
			Base:        config.Current.Energy,
			DefaultBase: config.Current.Energy,
		},
	}

	a.time = &area.Widget{
		Text:     "Time",
		TextSize: unit.Sp(12),
		Theme:    collection.Calibri().Theme,
		Min:      config.Current.Time.Min,
		Max:      config.Current.Time.Max,
		NRGBA:    area.Locked,
		Match:    g.matchTime,
		Cooldown: team.Time.Delay,

		Capture: &area.Capture{
			Option:      "Time",
			File:        "time_area.png",
			Base:        config.Current.Time,
			DefaultBase: config.Current.Time,
		},
	}

	a.score = &area.Widget{
		Text:          "Score",
		TextAlignLeft: true,
		Theme:         collection.Calibri().Theme,
		Min:           config.Current.Scores.Min,
		Max:           config.Current.Scores.Max,
		NRGBA:         area.Locked,
		Match:         g.matchScore,
		Cooldown:      team.Purple.Delay,

		Capture: &area.Capture{
			Option:      "Score",
			File:        "score_area.png",
			Base:        config.Current.Scores,
			DefaultBase: config.Current.Scores,
		},
	}

	a.state = &area.Widget{
		Hidden: true,

		Text:    "State",
		Subtext: strings.Title(match.NotFound.String()),
		Theme:   collection.Calibri().Theme,
		NRGBA:   area.Locked.Alpha(0),
		Match:   g.matchState,
		Min:     image.Pt(0, 0),
		Max:     image.Pt(150, 25),

		Capture: &area.Capture{
			Option:      "State",
			File:        "state_area.png",
			Base:        video.StateArea(),
			DefaultBase: video.StateArea(),
		},
	}

	return a
}

func (g *GUI) videos(text float32) *videos {
	v := &videos{
		onevent: func() { /*No-op.*/ },
	}

	v.monitor = capture{
		list: &dropdown.Widget{
			Theme:    g.header.Collection.Calibri().Theme,
			TextSize: text,
			Items:    []*dropdown.Item{},
			Callback: func(i *dropdown.Item, _ *dropdown.Widget) bool {
				defer v.onevent()

				video.Close()

				defer v.monitor.populate(true)
				defer v.window.populate(true)
				defer v.device.populate(true)

				config.Current.VideoCaptureWindow = i.Text
				if config.Current.VideoCaptureWindow == "" {
					config.Current.VideoCaptureWindow = config.MainDisplay
				}

				return true
			},
		},
		populate: func(videoCaptureDisabledEvent bool) {
			if videoCaptureDisabledEvent {
				for _, item := range v.monitor.list.Items {
					item.Checked.Value = false
					if item.Text == config.Current.VideoCaptureWindow {
						item.Checked.Value = true
					}
				}
			}

			screens := video.Screens()
			if len(screens) == v.monitor.len && !videoCaptureDisabledEvent {
				return
			}
			v.monitor.len = len(screens)

			items := []*dropdown.Item{}

			if videoCaptureDisabledEvent && config.Current.VideoCaptureWindow == "" {
				config.Current.VideoCaptureWindow = config.MainDisplay
			}

			for _, screen := range screens {
				items = append(items,
					&dropdown.Item{
						Text:    screen,
						Checked: widget.Bool{Value: screen == config.Current.VideoCaptureWindow},
					},
				)
			}

			v.monitor.list.Items = items
		},
	}

	v.window = capture{
		list: &dropdown.Widget{
			Theme:    g.header.Collection.Calibri().Theme,
			TextSize: text,
			Items:    []*dropdown.Item{},
			Callback: func(i *dropdown.Item, _ *dropdown.Widget) bool {
				defer v.onevent()

				video.Close()

				defer v.window.populate(true)
				defer v.monitor.populate(true)
				defer v.device.populate(true)

				config.Current.VideoCaptureWindow = i.Text
				if config.Current.VideoCaptureWindow == "" {
					config.Current.VideoCaptureWindow = config.MainDisplay
				}
				return true
			},
		},
		populate: func(videoCaptureDisabledEvent bool) {
			if videoCaptureDisabledEvent && config.Current.VideoCaptureWindow == "" {
				config.Current.VideoCaptureWindow = config.MainDisplay
			}

			for _, item := range v.window.list.Items {
				item.Checked.Value = config.Current.VideoCaptureWindow == item.Text
			}

			items := []*dropdown.Item{}

			windows := video.Windows()
			if len(windows) == len(v.window.list.Items) && !videoCaptureDisabledEvent {
				if len(v.window.list.Items) == 0 {
					return
				}

				if v.window.list.Default().Checked.Value {
					return
				}

				for _, item := range v.window.list.Items {
					if item.Checked.Value {
						items = append([]*dropdown.Item{item}, items...)
					} else {
						items = append(items, item)
					}
				}
			} else {
				for _, win := range windows {
					item := &dropdown.Item{
						Text:    win,
						Checked: widget.Bool{Value: win == config.Current.VideoCaptureWindow},
					}
					if item.Checked.Value {
						items = append([]*dropdown.Item{item}, items...)
					} else {
						items = append(items, item)
					}
				}
			}

			v.window.list.Items = items
		},
	}

	v.device = capture{
		list: &dropdown.Widget{
			Theme:    g.header.Collection.Calibri().Theme,
			TextSize: text,
			Items: []*dropdown.Item{
				{
					Text:  "Disabled",
					Value: config.NoVideoCaptureDevice,
					Checked: widget.Bool{
						Value: device.IsActive(),
					},
				},
			},
			Callback: func(i *dropdown.Item, _ *dropdown.Widget) bool {
				defer v.onevent()

				video.Close()

				if i.Text == "Disabled" {
					i.Checked = widget.Bool{Value: true}
				}

				defer v.device.populate(i.Text == "Disabled")
				defer v.window.populate(true)
				defer v.monitor.populate(true)

				go func() {
					config.Current.VideoCaptureDevice = i.Value

					err := video.Open()
					if err != nil {
						g.ToastError(err)

						config.Current.VideoCaptureWindow = config.MainDisplay
						config.Current.VideoCaptureDevice = config.NoVideoCaptureDevice
					}
				}()

				return true
			},
		},
		populate: func(videoCaptureDisabledEvent bool) {
			devices := video.Devices()

			// Set the "Disabled" checkbox when device is not active.
			if len(devices)+1 == len(v.device.list.Items) && !videoCaptureDisabledEvent {
				v.device.list.Default().Checked.Value = !device.IsActive()

				for _, item := range v.device.list.Items {
					item.Checked.Value = false
					if config.Current.VideoCaptureDevice == item.Value {
						item.Checked.Value = true
					}
				}

				return
			}

			v.device.list.Items = []*dropdown.Item{
				{
					Text:  "Disabled",
					Value: config.NoVideoCaptureDevice,
					Checked: widget.Bool{
						Value: device.IsActive(),
					},
				},
			}
			for _, d := range devices {
				v.device.list.Items = append(v.device.list.Items, &dropdown.Item{
					Text:  device.Name(d),
					Value: d,
				},
				)
			}

			for _, i := range v.device.list.Items {
				i.Checked.Value = false
				if i.Value == config.Current.VideoCaptureDevice {
					i.Checked.Value = true
				}
			}
		},
	}

	v.apis = capture{
		list: &dropdown.Widget{
			Theme:    g.header.Collection.Calibri().Theme,
			TextSize: text,
			Items:    []*dropdown.Item{},
			Callback: func(i *dropdown.Item, this *dropdown.Widget) bool {
				for _, item := range this.Items {
					item.Checked.Value = false
				}
				i.Checked.Value = true

				config.Current.VideoCaptureAPI = i.Text

				for _, item := range v.device.list.Items {
					if item.Checked.Value {
						v.device.list.Callback(item, v.device.list)
						return true
					}
				}

				return true
			},
		},
		populate: func(videoCaptureDisabledEvent bool) {
			apis := device.APIs()
			if len(apis) == 0 {
				return
			}

			for i, api := range device.APIs() {
				v.apis.list.Items = append(v.apis.list.Items,
					&dropdown.Item{
						Text:  api,
						Value: device.API(api),
						Checked: widget.Bool{
							Value: api == config.Current.VideoCaptureAPI || (i == 0 && config.Current.VideoCaptureAPI == ""),
						},
					},
				)
			}
		},
	}

	v.platform = capture{
		list: &dropdown.Widget{
			Theme: g.header.Collection.Calibri().Theme,
			Items: []*dropdown.Item{
				{
					Text:    strings.Title(config.PlatformSwitch),
					Checked: widget.Bool{Value: config.Current.Platform == config.PlatformSwitch},
				},
				{
					Text:    strings.Title(config.PlatformMobile),
					Checked: widget.Bool{Value: config.Current.Platform == config.PlatformMobile},
				},
				{
					Text:    strings.Title(config.PlatformBluestacks),
					Checked: widget.Bool{Value: config.Current.Platform == config.PlatformBluestacks},
				},
			},
			Callback: func(i *dropdown.Item, l *dropdown.Widget) bool {
				defer v.onevent()

				for _, item := range l.Items {
					if item != i {
						item.Checked.Value = false
						continue
					}
					item.Checked.Value = true

					config.Current.Platform = strings.ToLower(item.Text)

					err := config.Current.Save()
					if err != nil {
						notify.Error("Failed to load %s profile configuration", config.Current.Profile)
						return false
					}

					err = config.Load(config.Current.Profile)
					if err != nil {
						notify.Error("Failed to load %s profile configuration", config.Current.Profile)
						return false
					}

					time.AfterFunc(time.Second, func() {
						err := config.Current.Save()
						if err != nil {
							notify.Error("Failed to save %s profile configuration", config.Current.Profile)
						}
					})
				}
				return true
			},
		},
	}

	v.profile = capture{
		list: &dropdown.Widget{
			Theme: g.header.Collection.Calibri().Theme,
			Radio: true,
			Items: []*dropdown.Item{
				{
					Text: strings.Title(config.ProfilePlayer),
					Checked: widget.Bool{
						Value: config.Current.Profile == config.ProfilePlayer,
					},
				},
				{
					Text: strings.Title(config.ProfileBroadcaster),
					Checked: widget.Bool{
						Value: config.Current.Profile == config.ProfileBroadcaster,
					},
				},
			},
			Callback: func(i *dropdown.Item, _ *dropdown.Widget) bool {
				defer v.onevent()

				if config.Current.Profile == strings.ToLower(i.Text) {
					return true
				}

				config.Current.Profile = strings.ToLower(i.Text)
				err := config.Load(config.Current.Profile)
				if err != nil {
					notify.Error("Failed to load %s profile configuration", config.Current.Profile)
					return false
				}

				v.window.populate(true)
				v.device.populate(true)
				v.monitor.populate(true)

				notify.System("Profile set to %s mode", i.Text)

				time.AfterFunc(time.Second, func() {
					err := config.Current.Save()
					if err != nil {
						notify.Error("Failed to save %s profile configuration", config.Current.Profile)
					}
				})
				return true
			},
		},
	}

	return v
}
