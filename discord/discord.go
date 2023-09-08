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
	for ; reconnect(rpc.error()); time.Sleep(time.Second * 15) {
		if config.Current.Advanced.Discord.Disabled {
			continue
		}

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
}

func reconnect(err error) bool {
	if err != nil {
		notify.Feed(nrgba.Discord, "Discord disconnected (%v)", err)
	}

	for ; !rpc.connected; time.Sleep(time.Second * 10) {
		if config.Current.Advanced.Discord.Disabled {
			continue
		}

		notify.Feed(nrgba.Discord, "Discord connecting...")

		rpc, err = connect()
		if err != nil {
			continue
		}

		rpc.handshake(id)

		err = rpc.error()
		if err != nil {
			continue
		}

		notify.Feed(nrgba.Discord, "Discord connected")
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
		if server.Match() {
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
			case state.Nothing:
			case state.MatchStarting:
				a.Timestamps = timestamps{}
			case state.Killed, state.KilledWithPoints, state.KilledWithoutPoints:
				a.State = fmt.Sprintf("Died %s ago", time.Since(event.Time))
			case state.MatchEnding:
				a.Details = "UniteHUD - Match Ended"

				won := "Purple"
				if last.score.orange > last.score.purple {
					won = "Orange"
				}
				a.State = fmt.Sprintf("%s team won %d - %d", won, last.score.purple, last.score.orange)
			default:
				if time.Since(event.Time) > time.Second*30 {
					break
				}
				a.Details = fmt.Sprintf("UniteHUD - %s", event.EventType.String())
			}

			last.event.EventType = event.EventType
			last.event.count++
		}
	}

	return a
}
