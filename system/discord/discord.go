package discord

import (
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"

	"github.com/pidgy/unitehud/core/config"
	"github.com/pidgy/unitehud/core/global"
	"github.com/pidgy/unitehud/core/notify"
	"github.com/pidgy/unitehud/core/nrgba"
	"github.com/pidgy/unitehud/core/server"
	"github.com/pidgy/unitehud/core/state"
	"github.com/pidgy/unitehud/gui/is"
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

	wait struct {
		activity
		time.Time
	}
)

var (
	Activity = status()
)

func Connect() {
	if global.DebugMode {
		return
	}

	update()

	t := time.NewTicker(time.Second * 5)
	for range t.C {
		update()
	}
}

func Disconnect() {
	rpc.cleanup()
	notify.Feed(nrgba.Discord, "Discord: Disconnected")
}

func reconnect() {
	err := rpc.error()
	if err != nil {
		notify.Warn("Discord: Disconnected (%v)", err)
	}

	if config.Current.Advanced.Discord.Disabled && rpc.conn != nil {
		rpc.cleanup()
	}

	retries := 0

	for wait := time.Second; rpc.conn == nil; time.Sleep(wait) {
		if config.Current.Advanced.Discord.Disabled {
			continue
		}
		wait = wait << 1

		retries++
		if retries == 10 {
			notify.Warn("Discord: Failed to connect, rpc disabled")
			config.Current.Advanced.Discord.Disabled = true
			continue
		}

		notify.Feed(nrgba.Discord, "Discord: Connecting...")

		rpc, err = connect()
		if err != nil {
			notify.Warn("Discord: Failed to connect (%v)", err)
			continue
		}

		rpc.handshake(id)

		err = rpc.error()
		if err != nil {
			notify.Warn("Discord: Failed to connect (%v)", err)
			continue
		}

		notify.Feed(nrgba.Discord, "Discord: Connected")
	}
}

func status() activity {
	if time.Now().Before(wait.Time) {
		return wait.activity
	}

	a := activity{
		State: "Waiting for match to start",

		Details: "UniteHUD - Main Menu",

		Assets: assets{
			LargeImage: "icon1024x1024",
			LargeText:  "UniteHUD",
			SmallImage: "asdasd",
			SmallText:  "unitehud.dev",
		},

		Timestamps: timestamps{},

		Type: activityTypePlaying,

		Buttons: []button{
			{
				Label: "Download UniteHUD",
				URL:   "https://unitehud.dev",
			},
		},

		Instance: activityInstanceIdle,

		Party: party{
			ID: partyID,
			Size: size{
				CurrentSize: 1,
				MaxSize:     5,
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

		// event := state.MatchEnding.Occured(time.Second * 5)
		// if event == nil {
		// event = state.Last()
		// }

		event := state.Last()

		if !server.Match() && event.EventType != state.MatchEnding {
			return a
		}

		a.Instance = activityInstanceInMatch

		score := ""
		switch {
		case last.score.purple > last.score.orange:
			score = "Winning"
		case last.score.purple < last.score.orange:
			score = "Behind"
		case last.score.purple == last.score.orange:
			score = "Tied"
		}

		last.score.orange, last.score.purple, last.score.self = server.Scores()
		a.Details = "UniteHUD - In a Match"
		a.State = fmt.Sprintf("%s %d - %d", score, last.score.purple, last.score.orange)

		ten := int64((time.Minute * 10).Milliseconds())
		ms := int64(server.Seconds() * 1000)

		started := ten - ms
		ends := ten - started
		if ms > 0 {
			a.Timestamps.Start = time.Now().UnixMilli() - started
			a.Timestamps.End = time.Now().UnixMilli() + ends
		}

		switch event.EventType {
		case state.HoldingEnergy, state.OrangeScoreMissed, state.PurpleScoreMissed,
			state.PressButtonToScore, state.PreScore, state.PostScore, state.Nothing:
		case state.MatchStarting:
			a.Timestamps = timestamps{}
			a.Details = "UniteHUD - Match Starting"
			a.State = "Loading..."
		case state.Killed, state.KilledWithPoints, state.KilledWithoutPoints:
			e := state.First(event.EventType, time.Second*30)
			if e != nil {
				event = e
			}
			a.State = fmt.Sprintf("Died %ds ago", int(time.Since(event.Time).Seconds()))
		case state.MatchEnding:
			a.Details = "UniteHUD - Match Ended"

			winner := "Tied"
			switch {
			case last.score.orange > last.score.purple:
				winner = "Orange"
			case last.score.purple > last.score.orange:
				winner = "Purple"
			}

			switch {
			case last.score.purple+last.score.orange == 0:
				a.State = "Viewing battle report"
			default:
				a.State = fmt.Sprintf("%s Team Won %d - %d", winner, last.score.purple, last.score.orange)
			}

			wait.activity = a
			wait.Time = time.Now().Add(time.Second * 10)
		case state.PurpleScore, state.OrangeScore, state.FirstScored:
			fallthrough
		default:
			if time.Since(event.Time) > time.Second*30 {
				break
			}

			a.State = fmt.Sprintf("%s - %s", event.EventType, a.State)
		}

		if server.IsFinalStretch() {
			a.State = fmt.Sprintf("Final Stretch - %s", a.State)
		}

		last.event.EventType = event.EventType
		last.event.count++
	}

	return a
}

func update() {
	reconnect()

	Activity = status()

	rpc.send(frame{
		Cmd: commandSetActivity,
		Args: args{
			Pid:      os.Getpid(),
			Activity: Activity,
		},
		Nonce: uuid.New().String(),
	})
}
