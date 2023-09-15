package discord

import (
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"

	"github.com/pidgy/unitehud/config"
	"github.com/pidgy/unitehud/gui/is"
	"github.com/pidgy/unitehud/notify"
	"github.com/pidgy/unitehud/nrgba"
	"github.com/pidgy/unitehud/server"
	"github.com/pidgy/unitehud/state"
)

var (
	rpc client

	last struct {
		score struct {
			orange,
			purple,
			self int
		}

		event struct {
			state.EventType
			count int
		}
	}
)

var Activity = status()

func Connect() {
	for ; ; time.Sleep(time.Second * 15) {
		if !reconnect() {
			continue
		}

		Activity = status()

		notify.Debug("Discord: %s: \"%s\"", Activity.Details, Activity.State)

		rpc.send(frame{
			Cmd: commandSetActivity,
			Args: args{
				Pid:      os.Getpid(),
				Activity: status(),
			},
			Nonce: uuid.New().String(),
		})
	}
}

func reconnect() bool {
	if config.Current.Advanced.Discord.Disabled {
		return true
	}

	err := rpc.error()
	if err != nil {
		notify.Feed(nrgba.Discord, "Reconnecting to Discord (%v)", err)
	}

	for ; !rpc.connected; time.Sleep(time.Second * 15) {
		if config.Current.Advanced.Discord.Disabled {
			continue
		}

		notify.Feed(nrgba.Discord, "Connecting to Discord")

		rpc, err = connect()
		if err != nil {
			continue
		}

		rpc.handshake(id)

		err = rpc.error()
		if err != nil {
			continue
		}

		notify.Feed(nrgba.Discord, "Connected to Discord")
	}

	return rpc.connected
}

func status() activity {
	a := activity{
		State: "Waiting for match to start",

		Details: "UniteHUD - Main Menu",

		Assets: assets{
			LargeImage: "icon1024x1024",
			LargeText:  "UniteHUD",
			SmallImage: "asdasd",
			SmallText:  "unitehud.dev",
		},

		Timestamps: timestamps{
			Start: 0,
		},

		Type: activityTypeGame,

		Buttons: []button{
			{
				Label: "Download UniteHUD",
				URL:   "https://unitehud.dev",
			},
		},
	}

	a.Details = fmt.Sprintf("UniteHUD - %s", is.Now)

	switch is.Now {
	case is.Projecting:
		a.State = "Configuring capture settings"
	case is.Loading:
		a.State = "Starting"
	case is.MainMenu:
		if !server.Started() {
			a.State = "Ready to capture scores"
		}

		if !server.Match() {
			return a
		}

		last.score.orange, last.score.purple, last.score.self = server.Scores()
		a.Details = "UniteHUD - In a Match"
		a.State = fmt.Sprintf("[%s] %d - %d", server.Clock(), last.score.purple, last.score.orange)

		ten := int64((time.Minute * 10).Milliseconds())
		ms := int64(server.Seconds() * 1000)

		started := ten - ms
		ends := ten - started
		if ms > 0 {
			a.Timestamps.Start = time.Now().UnixMilli() - started
			a.Timestamps.End = time.Now().UnixMilli() + ends
		}

		event := state.Last()
		switch event.EventType {
		case state.HoldingEnergy,
			state.OrangeScoreMissed, state.PurpleScoreMissed,
			state.PressButtonToScore, state.PreScore, state.PostScore,
			state.Nothing:

		case state.MatchStarting:
			a.Timestamps = timestamps{}
		case state.Killed, state.KilledWithPoints, state.KilledWithoutPoints:
			a.State = fmt.Sprintf("Died %ds ago", int(time.Since(event.Time).Seconds()))
		case state.MatchEnding:
			a.Details = "UniteHUD - Match Ended"

			won := "Purple"
			if last.score.orange > last.score.purple {
				won = "Orange"
			}
			a.State = fmt.Sprintf("%s team won %d - %d", won, last.score.purple, last.score.orange)
		case state.PurpleScore, state.OrangeScore, state.FirstScored:
			fallthrough
		default:
			if time.Since(event.Time) > time.Second*30 {
				break
			}
			a.State = fmt.Sprintf("%s %s", event.EventType, a.State)
		}

		last.event.EventType = event.EventType
		last.event.count++
	}

	return a
}
