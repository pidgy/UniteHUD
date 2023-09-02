package gui

import (
	"image"
	"strings"
	"time"

	"gioui.org/unit"
	"gioui.org/widget"
	"github.com/pidgy/unitehud/audio"
	"github.com/pidgy/unitehud/config"
	"github.com/pidgy/unitehud/fonts"
	"github.com/pidgy/unitehud/gui/visual/area"
	"github.com/pidgy/unitehud/gui/visual/dropdown"
	"github.com/pidgy/unitehud/match"
	"github.com/pidgy/unitehud/notify"
	"github.com/pidgy/unitehud/team"
	"github.com/pidgy/unitehud/video"
	"github.com/pidgy/unitehud/video/device"
	"github.com/pidgy/unitehud/video/window/electron"
)

type areas struct {
	energy    *area.Area
	ko        *area.Area
	objective *area.Area
	score     *area.Area
	state     *area.Area
	time      *area.Area

	onevent func()
}

type audios struct {
	in  capture
	out capture
}

type capture struct {
	list     *dropdown.List
	populate func(bool)
	len      int
}

type videos struct {
	device   capture
	window   capture
	monitor  capture
	platform capture
	profile  capture

	onevent func()
}

func (g *GUI) audios(text float32, session *audio.Session) *audios {
	a := &audios{
		in: capture{
			list: &dropdown.List{
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
				Callback: func(i *dropdown.Item, d *dropdown.List) {
					for _, item := range d.Items {
						item.Checked.Value = false
						if item == i {
							item.Checked.Value = true
						}
					}
				},
			},
		},
		out: capture{
			list: &dropdown.List{
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
				Callback: func(i *dropdown.Item, d *dropdown.List) {
					for _, item := range d.Items {
						item.Checked.Value = false
						if item == i {
							item.Checked.Value = true
						}
					}
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

	a.ko = &area.Area{
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

	a.objective = &area.Area{
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

	a.energy = &area.Area{
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

	a.time = &area.Area{
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

	a.score = &area.Area{
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

	a.state = &area.Area{
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
		list: &dropdown.List{
			TextSize: text,
			Items:    []*dropdown.Item{},
			Callback: func(i *dropdown.Item, _ *dropdown.List) {
				defer v.onevent()

				device.Close()
				electron.Close()

				defer v.monitor.populate(true)
				defer v.window.populate(true)
				defer v.device.populate(true)

				config.Current.Window = i.Text
				if config.Current.Window == "" {
					config.Current.Window = config.MainDisplay
					return
				}
			},
		},
		populate: func(videoCaptureDisabledEvent bool) {
			if videoCaptureDisabledEvent {
				for _, item := range v.monitor.list.Items {
					item.Checked.Value = false
					if item.Text == config.Current.Window && !device.IsActive() {
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

			if videoCaptureDisabledEvent && config.Current.Window == "" {
				config.Current.Window = config.MainDisplay
			}

			for _, screen := range screens {
				items = append(items,
					&dropdown.Item{
						Text:    screen,
						Checked: widget.Bool{Value: screen == config.Current.Window && !device.IsActive()},
					},
				)
			}

			v.monitor.list.Items = items
		},
	}

	v.window = capture{
		list: &dropdown.List{
			TextSize: text,
			Items:    []*dropdown.Item{},
			Callback: func(i *dropdown.Item, _ *dropdown.List) {
				defer v.onevent()

				device.Close()
				electron.Close()

				defer v.window.populate(true)
				defer v.monitor.populate(true)
				defer v.device.populate(true)

				config.Current.Window = i.Text
				if config.Current.Window == "" {
					config.Current.Window = config.MainDisplay
					return
				}
			},
		},
		populate: func(videoCaptureDisabledEvent bool) {
			if videoCaptureDisabledEvent && config.Current.Window == "" {
				config.Current.Window = config.MainDisplay
			}

			for _, item := range v.window.list.Items {
				item.Checked.Value = config.Current.Window == item.Text && config.Current.VideoCaptureDevice == config.NoVideoCaptureDevice
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
						Checked: widget.Bool{Value: win == config.Current.Window},
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
		list: &dropdown.List{
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
			Callback: func(i *dropdown.Item, _ *dropdown.List) {
				defer v.onevent()

				electron.Close()
				video.Close()
				// Can this be Disabled? Fixes concurrency error in device.go Close.
				// time.Sleep(time.Second)

				config.Current.VideoCaptureDevice = i.Value

				if i.Text == "Disabled" {
					i.Checked = widget.Bool{Value: true}
				}

				defer v.device.populate(i.Text == "Disabled")
				defer v.window.populate(true)
				defer v.monitor.populate(true)

				go func() {
					err := video.Open()
					if err != nil {
						g.ToastErrorForce(err)

						config.Current.Window = config.MainDisplay
						config.Current.VideoCaptureDevice = config.NoVideoCaptureDevice

						defer v.window.populate(true)
						defer v.device.populate(true)
						defer v.monitor.populate(true)

						return
					}

					config.Current.LostWindow = ""
				}()
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

	v.platform = capture{
		list: &dropdown.List{
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
			Callback: func(i *dropdown.Item, l *dropdown.List) {
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
						return
					}

					err = config.Load(config.Current.Profile)
					if err != nil {
						notify.Error("Failed to load %s profile configuration", config.Current.Profile)
						return
					}

					time.AfterFunc(time.Second, func() {
						err := config.Current.Save()
						if err != nil {
							notify.Error("Failed to save %s profile configuration", config.Current.Profile)
							return
						}
					})
				}
			},
		},
	}

	v.profile = capture{
		list: &dropdown.List{
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
			Callback: func(i *dropdown.Item, _ *dropdown.List) {
				defer v.onevent()

				if config.Current.Profile == strings.ToLower(i.Text) {
					return
				}

				electron.Close()

				config.Current.Profile = strings.ToLower(i.Text)

				err := config.Load(config.Current.Profile)
				if err != nil {
					notify.Error("Failed to load %s profile configuration", config.Current.Profile)
					return
				}

				v.window.populate(true)
				v.device.populate(true)
				v.monitor.populate(true)

				notify.System("Profile set to %s mode", i.Text)

				time.AfterFunc(time.Second, func() {
					err := config.Current.Save()
					if err != nil {
						notify.Error("Failed to save %s profile configuration", config.Current.Profile)
						return
					}
				})
			},
		},
	}

	return v
}
