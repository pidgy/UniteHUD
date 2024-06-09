package ui

import (
	"fmt"
	"image"
	"strings"
	"time"

	"gioui.org/unit"
	"gioui.org/widget"

	"github.com/pidgy/unitehud/avi/audio"
	"github.com/pidgy/unitehud/avi/video"
	"github.com/pidgy/unitehud/avi/video/device"
	"github.com/pidgy/unitehud/core/config"
	"github.com/pidgy/unitehud/core/fonts"
	"github.com/pidgy/unitehud/core/match"
	"github.com/pidgy/unitehud/core/notify"
	"github.com/pidgy/unitehud/core/team"
	"github.com/pidgy/unitehud/gui/ux/area"
	"github.com/pidgy/unitehud/gui/ux/checklist"
	"github.com/pidgy/unitehud/system/lang"
)

type areas struct {
	energy *area.Widget
	// ko        *area.Widget
	objective          *area.Widget
	score              *area.Widget
	state              *area.Widget
	time               *area.Widget
	pressButtonToScore *area.Widget
}

type audios struct {
	in  capture
	out capture
}

type capture struct {
	list     *checklist.Widget
	populate func()
	len      int
}

type videos struct {
	device   capture
	window   capture
	monitor  capture
	platform capture
	apis     capture
	codecs   capture

	onevent func(bool)
}

func (g *GUI) audios(text float32) *audios {
	a := &audios{
		in: capture{
			list: &checklist.Widget{
				Theme:         g.nav.Collection.NotoSans().Theme,
				WidthModifier: 1,
				TextSize:      text,
				Radio:         true,
				Items: []*checklist.Item{
					{
						Text: audio.Disabled,
						Callback: func(i *checklist.Item) {
							err := audio.Input(audio.Disabled)
							if err != nil {
								g.ToastError(err)
								return
							}
						},
					},
					{
						Text: audio.Default,
						Callback: func(i *checklist.Item) {
							err := audio.Input(audio.Default)
							if err != nil {
								g.ToastError(err)
								return
							}
						},
					},
				},
			},
		},
		out: capture{
			list: &checklist.Widget{
				Theme:         g.nav.Collection.NotoSans().Theme,
				WidthModifier: 1,
				TextSize:      text,
				Radio:         true,
				Items: []*checklist.Item{
					{
						Text: audio.Disabled,
						Callback: func(i *checklist.Item) {
							err := audio.Output(audio.Disabled)
							if err != nil {
								g.ToastError(err)
								return
							}
							i.Checked.Value = false
						},
					},
					{
						Text: audio.Default,
						Callback: func(i *checklist.Item) {
							err := audio.Output(audio.Default)
							if err != nil {
								g.ToastError(err)
								return
							}
							i.Checked.Value = false
						},
					},
				},
			},
		},
	}

	for _, d := range audio.Inputs() {
		i := &checklist.Item{
			Text:    d.Name(),
			Checked: widget.Bool{Value: d.Is(config.Current.Audio.Capture.Device.Name)},
			Callback: func(i *checklist.Item) {
				err := audio.Input(i.Text)
				if err != nil {
					g.ToastError(err)
				}
				i.Checked.Value = err == nil
			},
		}
		a.in.list.Items = append(a.in.list.Items, i)
	}

	disabled := true
	for _, i := range a.in.list.Items {
		if i.Checked.Value {
			disabled = false
		}
	}
	if disabled {
		a.in.list.Items[0].Checked.Value = true
	}

	for _, d := range audio.Outputs() {
		i := &checklist.Item{
			Text:    d.Name(),
			Checked: widget.Bool{Value: d.Is(config.Current.Audio.Playback.Device.Name)},
			Callback: func(i *checklist.Item) {
				err := audio.Output(i.Text)
				if err != nil {
					g.ToastError(err)
				}
				i.Checked.Value = err == nil
			},
		}
		a.out.list.Items = append(a.out.list.Items, i)
	}
	disabled = true
	for _, i := range a.out.list.Items {
		if i.Checked.Value {
			disabled = false
		}
	}
	if disabled {
		a.out.list.Items[0].Checked.Value = true
	}

	return a
}

func (g *GUI) areas(collection fonts.Collection) *areas {
	return &areas{
		objective: &area.Widget{
			Text:     "Objectives",
			TextSize: unit.Sp(13),
			Theme:    collection.Calibri().Theme,
			Min:      config.Current.XY.Objectives.Min,
			Max:      config.Current.XY.Objectives.Max,
			NRGBA:    area.Locked,
			Match:    g.matchObjectives,
			Cooldown: time.Second,

			Capture: &area.Capture{
				Option:      "Objective",
				File:        "objective_area.png",
				Base:        config.Current.XY.Objectives,
				DefaultBase: config.Current.XY.Objectives,
			},
		},

		energy: &area.Widget{
			Text:     "Aeos",
			TextSize: unit.Sp(13),
			Theme:    collection.Calibri().Theme,
			Min:      config.Current.XY.Energy.Min,
			Max:      config.Current.XY.Energy.Max,
			NRGBA:    area.Locked,
			Match:    g.matchEnergy,
			Cooldown: team.Energy.Delay,

			Capture: &area.Capture{
				Option:      "Aeos",
				File:        "aeos_area.png",
				Base:        config.Current.XY.Energy,
				DefaultBase: config.Current.XY.Energy,
			},
		},

		time: &area.Widget{
			Text:     "Time",
			TextSize: unit.Sp(12),
			Theme:    collection.Calibri().Theme,
			Min:      config.Current.XY.Time.Min,
			Max:      config.Current.XY.Time.Max,
			NRGBA:    area.Locked,
			Match:    g.matchTime,
			Cooldown: team.Time.Delay,

			Capture: &area.Capture{
				Option:      "Time",
				File:        "time_area.png",
				Base:        config.Current.XY.Time,
				DefaultBase: config.Current.XY.Time,
			},
		},

		score: &area.Widget{
			Text:          "Score",
			TextAlignLeft: true,
			Theme:         collection.Calibri().Theme,
			Min:           config.Current.XY.Scores.Min,
			Max:           config.Current.XY.Scores.Max,
			NRGBA:         area.Locked,
			Match:         g.matchScore,
			Cooldown:      team.Purple.Delay,

			Capture: &area.Capture{
				Option:      "Score",
				File:        "score_area.png",
				Base:        config.Current.XY.Scores,
				DefaultBase: config.Current.XY.Scores,
			},
		},

		state: &area.Widget{
			Hidden: true,

			Text:    "State",
			Subtext: match.NotFound.String(),
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
		},

		pressButtonToScore: &area.Widget{
			Hidden: true,

			Text:          "Self-Score",
			TextAlignLeft: true,
			Theme:         collection.Calibri().Theme,
			Min:           config.Current.ScoringOption().Min,
			Max:           config.Current.ScoringOption().Max,
			NRGBA:         area.Locked,
			Match:         g.matchPressButtonToScore,
			Cooldown:      team.Purple.Delay,

			Capture: &area.Capture{
				Option:      "Self-Score",
				File:        "self_score_area.png",
				Base:        config.Current.ScoringOption(),
				DefaultBase: config.Current.ScoringOption(),
			},
		},
	}
}

func (g *GUI) videos(text float32) *videos {
	v := &videos{}

	v.monitor = capture{
		list: &checklist.Widget{
			Theme:    g.nav.Collection.NotoSans().Theme,
			TextSize: text,
			Items:    []*checklist.Item{},
			Callback: func(i *checklist.Item, _ *checklist.Widget) (check bool) {
				video.Close()

				config.Current.Video.Capture.Window.Name = i.Text
				if config.Current.Video.Capture.Window.Name == "" {
					config.Current.Video.Capture.Window.Name = config.MainDisplay
				}

				v.populate()

				return true
			},
		},
		populate: func() {
			// if videoCaptureDisabledEvent {
			// 	for _, item := range v.monitor.list.Items {
			// 		item.Checked.Value = false
			// 		if item.Text == config.Current.Video.Capture.Window.Name {
			// 			item.Checked.Value = true
			// 		}
			// 	}
			// }

			screens := video.Screens()
			if len(screens) == v.monitor.len {
				return
			}
			v.monitor.len = len(screens)

			items := []*checklist.Item{}

			if config.Current.Video.Capture.Window.Name == "" {
				config.Current.Video.Capture.Window.Name = config.MainDisplay
			}

			for _, screen := range screens {
				items = append(items,
					&checklist.Item{
						Text:    screen,
						Checked: widget.Bool{Value: video.Active(video.Monitor, screen)},
					},
				)
			}

			v.monitor.list.Items = items
		},
	}

	v.window = capture{
		list: &checklist.Widget{
			Theme:    g.nav.Collection.NotoSans().Theme,
			TextSize: text,
			Items:    []*checklist.Item{},
			Callback: func(i *checklist.Item, _ *checklist.Widget) (check bool) {
				video.Close()

				defer v.window.populate()
				defer v.monitor.populate()
				defer v.device.populate()
				defer v.apis.populate()
				defer v.codecs.populate()

				config.Current.Video.Capture.Window.Name = i.Text
				if config.Current.Video.Capture.Window.Name == "" {
					config.Current.Video.Capture.Window.Name = config.MainDisplay
				}
				return true
			},
		},
		populate: func() {
			if config.Current.Video.Capture.Window.Name == "" {
				config.Current.Video.Capture.Window.Name = config.MainDisplay
			}

			for _, item := range v.window.list.Items {
				item.Checked.Value = config.Current.Video.Capture.Window.Name == item.Text
			}

			items := []*checklist.Item{}

			windows := video.Windows()
			if len(windows) == len(v.window.list.Items) {
				if len(v.window.list.Items) == 0 {
					return
				}

				if v.window.list.Default().Checked.Value {
					return
				}

				for _, item := range v.window.list.Items {
					if item.Checked.Value {
						items = append([]*checklist.Item{item}, items...)
					} else {
						items = append(items, item)
					}
				}
			} else {
				for _, win := range windows {
					item := &checklist.Item{
						Text:    win,
						Checked: widget.Bool{Value: win == config.Current.Video.Capture.Window.Name},
					}
					if item.Checked.Value {
						items = append([]*checklist.Item{item}, items...)
					} else {
						items = append(items, item)
					}
				}
			}

			v.window.list.Items = items

		},
	}

	v.device = capture{
		list: &checklist.Widget{
			Theme:    g.nav.Collection.NotoSans().Theme,
			TextSize: text,
			Items: []*checklist.Item{
				{
					Text:  "Disabled",
					Value: config.NoVideoCaptureDevice,
					Checked: widget.Bool{
						Value: device.IsActive(),
					},
				},
			},
			Callback: func(i *checklist.Item, _ *checklist.Widget) (check bool) {
				video.Close()

				if i.Text == "Disabled" {
					i.Checked.Value = true
				}

				go func() {
					config.Current.Video.Capture.Device.Index = i.Value
					config.Current.Video.Capture.Device.Name = i.Text
					config.Current.Video.Capture.Device.API = config.DefaultVideoCaptureAPI
					config.Current.Video.Capture.Device.Codec = config.DefaultVideoCaptureCodec

					err := video.Open()
					if err != nil {
						g.ToastError(err)
					}

				}()

				return true
			},
		},
		populate: func() {
			devices := video.Devices()

			// Set the "Disabled" checkbox when device is not active.
			if len(devices)+1 == len(v.device.list.Items) {
				v.device.list.Default().Checked.Value = !device.IsActive()

				for _, item := range v.device.list.Items {
					item.Checked.Value = false
					if config.Current.Video.Capture.Device.Index == item.Value {
						item.Checked.Value = true
					}
				}

				return
			}

			v.device.list.Items = []*checklist.Item{
				{
					Text:  "Disabled",
					Value: config.NoVideoCaptureDevice,
					Checked: widget.Bool{
						Value: device.IsActive(),
					},
				},
			}
			for _, d := range devices {
				v.device.list.Items = append(v.device.list.Items, &checklist.Item{
					Text:  device.Name(d),
					Value: d,
				},
				)
			}

			for _, i := range v.device.list.Items {
				i.Checked.Value = false
				if i.Value == config.Current.Video.Capture.Device.Index {
					i.Checked.Value = true
				}
			}
		},
	}

	v.apis = capture{
		list: &checklist.Widget{
			Theme:    g.nav.Collection.NotoSans().Theme,
			TextSize: text,
			Items:    []*checklist.Item{},
			Callback: func(i *checklist.Item, this *checklist.Widget) (check bool) {
				if i.Text == config.Current.Video.Capture.Device.API {
					return true
				}

				if config.Current.Video.Capture.Device.Index == config.NoVideoCaptureDevice {
					return false
				}

				defer v.populate()

				for _, item := range this.Items {
					item.Checked.Value = false
				}
				i.Checked.Value = true

				// Set the API, restart the capture device, and verify application.
				prev := config.Current.Video.Capture.Device
				config.Current.Video.Capture.Device.API = i.Text

				v.onevent(true) // Hide preview.

				err := device.Restart()
				if err != nil {
					g.ToastOK(
						config.Current.Video.Capture.Device.Name,
						err.Error(),
						OnToastOK(func() {
							defer v.apis.populate()

							config.Current.Video.Capture.Device = prev

							err = device.Restart()
							if err != nil {
								g.ToastOK(
									config.Current.Video.Capture.Device.Name,
									lang.Title(err.Error()),
									OnToastOK(func() {
										defer v.apis.populate()

										v.onevent(false) // Show preview.
									}),
								)
								return
							}

							v.onevent(false) // Show preview.
						}))

					return false
				}

				if config.Current.Video.Capture.Device.API != i.Text {
					g.ToastOK(
						config.Current.Video.Capture.Device.Name,
						fmt.Sprintf("Using default API for this device (%s)", config.Current.Video.Capture.Device.API),
						OnToastOK(func() {
							defer v.apis.populate()

							v.onevent(false) // Show preview.
						}),
					)

					return false
				}

				v.onevent(false) // Show preview.
				return true
			},
		},
		populate: func() {
			v.apis.list.Items = []*checklist.Item{}

			for _, api := range device.APIs() {
				v.apis.list.Items = append(v.apis.list.Items,
					&checklist.Item{
						Text:  api,
						Value: device.API(api).Value(),
						Checked: widget.Bool{
							Value: api == config.Current.Video.Capture.Device.API,
						},
					},
				)
			}
		},
	}

	v.codecs = capture{
		list: &checklist.Widget{
			Theme:    g.nav.Collection.NotoSans().Theme,
			TextSize: text,
			Items:    []*checklist.Item{},
			Callback: func(i *checklist.Item, this *checklist.Widget) (check bool) {
				if config.Current.Video.Capture.Device.Index == config.NoVideoCaptureDevice {
					return false
				}

				if i.Text == config.Current.Video.Capture.Device.Codec {
					return true
				}

				defer v.device.populate()
				defer v.window.populate()
				defer v.monitor.populate()
				defer v.apis.populate()
				defer v.codecs.populate()

				for _, item := range this.Items {
					item.Checked.Value = false
				}
				i.Checked.Value = true

				// Set the Codec, restart the capture device, and verify application.
				prev := config.Current.Video.Capture.Device
				config.Current.Video.Capture.Device.Codec = i.Text

				v.onevent(true) // Hide preview.

				err := device.Restart()
				if err != nil {
					g.ToastOK(
						config.Current.Video.Capture.Device.Name,
						err.Error(),
						OnToastOK(func() {
							defer v.codecs.populate()

							config.Current.Video.Capture.Device = prev

							err = device.Restart()
							if err != nil {
								g.ToastOK(
									config.Current.Video.Capture.Device.Name,
									err.Error(),
									OnToastOK(func() {
										defer v.device.populate()
										defer v.window.populate()
										defer v.monitor.populate()
										defer v.apis.populate()
										defer v.codecs.populate()

										v.onevent(false) // Show preview.
									}),
								)
								return
							}

							v.onevent(false) // Show preview.
						}))

					return false
				}

				if config.Current.Video.Capture.Device.Codec != i.Text {
					g.ToastOK(
						config.Current.Video.Capture.Device.Name,
						fmt.Sprintf("Using default codec for this device (%s)", config.Current.Video.Capture.Device.Codec),
						OnToastOK(func() {
							defer v.codecs.populate()
							v.onevent(false) // Show preview.
						}),
					)

					return false
				}

				v.onevent(false) // Show preview.

				return true
			},
		},
		populate: func() {
			v.codecs.list.Items = []*checklist.Item{}

			for _, c := range device.Codecs() {
				s := c.String()

				v.codecs.list.Items = append(v.codecs.list.Items,
					&checklist.Item{
						Text: s,
						Checked: widget.Bool{
							Value: s == config.Current.Video.Capture.Device.Codec,
						},
					},
				)
			}
		},
	}

	v.platform = capture{
		list: &checklist.Widget{
			Theme: g.nav.Collection.NotoSans().Theme,
			Items: []*checklist.Item{
				{
					Text:    lang.Title(config.DeviceSwitch),
					Checked: widget.Bool{Value: config.Current.Gaming.Device == config.DeviceSwitch},
				},
				{
					Text:    lang.Title(config.DeviceMobile),
					Checked: widget.Bool{Value: config.Current.Gaming.Device == config.DeviceMobile},
				},
				{
					Text:    lang.Title(config.DeviceBluestacks),
					Checked: widget.Bool{Value: config.Current.Gaming.Device == config.DeviceBluestacks},
				},
			},
			Callback: func(i *checklist.Item, l *checklist.Widget) (check bool) {
				for _, item := range l.Items {
					if item != i {
						item.Checked.Value = false
						continue
					}
					item.Checked.Value = true

					config.Current.Gaming.Device = strings.ToLower(item.Text)

					err := config.Current.Save()
					if err != nil {
						notify.Error("[UI] Failed to load %s configuration", config.Current.Gaming.Device)
						return false
					}

					err = config.Open(config.Current.Gaming.Device)
					if err != nil {
						notify.Error("[UI] Failed to load %s configuration", config.Current.Gaming.Device)
						return false
					}

					time.AfterFunc(time.Second, func() {
						err := config.Current.Save()
						if err != nil {
							notify.Error("[UI] Failed to save %s configuration", config.Current.Gaming.Device)
						}
					})
				}
				return true
			},
		},
	}

	return v
}

func (v *videos) populate() {
	v.device.populate()
	v.window.populate()
	v.monitor.populate()
	v.apis.populate()
	v.codecs.populate()
}
