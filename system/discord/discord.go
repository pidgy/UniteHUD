package discord

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pidgy/unitehud/core/config"
	"github.com/pidgy/unitehud/core/notify"
	"github.com/pidgy/unitehud/core/rgba/nrgba"
	"github.com/pidgy/unitehud/core/server"
	"github.com/pidgy/unitehud/core/state"
	"github.com/pidgy/unitehud/gui/is"
)

var (
	rpc client

	// previous struct {
	// 	score struct {
	// 		orange,
	// 		purple,
	// 		self int
	// 	}
	// }

	discard = false

	lastMatch time.Time
)

func Activity() activity {
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

	if is.Now == is.Configuring {
		a.State = "Configuring capture settings"
		return a
	}

	if is.Now == is.Loading {
		a.State = "Configuring capture settings"
		return a
	}

	game := server.Game()

	a.Instance = activityInstanceInMatch

	switch {
	case !game.InMatch && time.Since(lastMatch) > time.Second*10:
		return a
	case !game.InMatch && time.Since(lastMatch) < time.Second*10:
		fallthrough
	case state.Occured(time.Second*10, state.MatchEnding, state.SurrenderOrange, state.SurrenderPurple):
		discardFor(time.Since(lastMatch))

		a.Details = "UniteHUD - Match Ended"
		a.Timestamps = timestamps{}

		switch {
		case game.Orange.Surrendered:
			a.State = fmt.Sprintf("Won %d - %d (Surrendered)", game.Purple.Value, game.Orange.Value)
		case game.Purple.Surrendered:
			a.State = fmt.Sprintf("Lost %d - %d (Surrendered)", game.Purple.Value, game.Orange.Value)
		case game.Purple.Value > game.Orange.Value:
			a.State = fmt.Sprintf("Won %d - %d", game.Purple.Value, game.Orange.Value)
		case game.Purple.Value < game.Orange.Value:
			a.State = fmt.Sprintf("Lost %d - %d", game.Purple.Value, game.Orange.Value)
		case game.Purple.Value == game.Orange.Value:
			a.State = fmt.Sprintf("Tied %d - %d", game.Purple.Value, game.Orange.Value)
		}

		return a
	}
	lastMatch = time.Now()

	if state.Occured(time.Second*10, state.MatchStarting) && server.Clock() == "10:00" {
		discardFor(time.Second * 10)

		a.Details = "UniteHUD - Match Starting"
		a.State = "Loading..."
		a.Timestamps = timestamps{}

		return a
	}

	ten := (time.Minute * 10).Milliseconds()
	ms := int64(game.Seconds * 1000)

	started := ten - ms
	ending := ten - started
	if ms > 0 {
		a.Timestamps.Start = time.Now().UnixMilli() - started
		a.Timestamps.End = time.Now().UnixMilli() + ending
	}

	a.Details = "UniteHUD - In a Match"

	wl := "Tied"
	switch {
	case game.Purple.Value > game.Orange.Value:
		wl = "Winning"
	case game.Purple.Value < game.Orange.Value:
		wl = "Losing"
	}
	a.State = fmt.Sprintf("%s %d - %d", wl, game.Purple.Value, game.Orange.Value)

	if state.Occured(time.Second*10, state.RayquazaSecureOrange, state.RayquazaSecurePurple) {
		a.State = fmt.Sprintf("%s +Rayquaza %s", strings.Title(game.Rayquaza), a.State)
	}

	if server.IsFinalStretch() {
		a.State = fmt.Sprintf("Final Stretch - %s", a.State)
	}

	// switch is.Now {
	// case is.Configuring:
	// 	a.State = "Configuring capture settings"
	// case is.Loading:
	// 	a.State = "Starting"
	// case is.MainMenu:
	// 	if !server.Ready() {
	// 		a.State = "Ready to capture scores"
	// 	}

	// 	if !server.Match() {
	// 		return a, !state.Occured(time.Second*15, state.MatchEnding, state.SurrenderOrange, state.SurrenderPurple)
	// 	}

	// 	event := state.Last()

	// 	a.Instance = activityInstanceInMatch

	// 	ten := int64((time.Minute * 10).Milliseconds())
	// 	ms := int64(server.Seconds() * 1000)

	// 	case state.HoldingEnergy, state.OrangeScoreMissed, state.PurpleScoreMissed,
	// 		state.SelfScoreIndicator, state.PreScore, state.PostScore, state.Nothing:
	// 	case state.MatchStarting:
	// 		a.Timestamps = timestamps{}
	// 		a.Details = "UniteHUD - Match Starting"
	// 		a.State = "Loading..."
	// 	case state.Killed, state.KilledWithPoints, state.KilledWithoutPoints:
	// 		e := state.First(event.EventType, time.Second*30)
	// 		if e != nil {
	// 			event = e
	// 		}
	// 		a.State = fmt.Sprintf("KO'd %ds ago", int(time.Since(event.Time).Seconds()))

	// 		discardFor(time.Second * 5)
	// 	case state.PurpleScore, state.OrangeScore, state.FirstScored:
	// 		fallthrough
	// 	case state.RegiceSecureOrange:
	// 		a.State = "Regice Lost"
	// 	case state.RegirockSecureOrange:
	// 		a.State = "Regirock Lost"
	// 	case state.RegielekiSecureOrange:
	// 		a.State = "Regieleki Lost"
	// 	case state.RegisteelSecureOrange:
	// 		a.State = "Registeel Lost"
	// 	case state.RayquazaSecureOrange:
	// 		a.State = "Rayquaza Lost"
	// 	case state.RayquazaSecurePurple:
	// 		a.State = "Rayquaza Secured"
	// 	case state.RegiceSecurePurple:
	// 		a.State = "Regice Secured"
	// 	case state.RegielekiSecurePurple:
	// 		a.State = "Regieleki Secured"
	// 	case state.RegisteelSecurePurple:
	// 		a.State = "Registeel Secured"
	// 	case state.RegirockSecurePurple:
	// 		a.State = "Regirock Secured"
	// 	default:
	// 		if time.Since(event.Time) > time.Second*30 {
	// 			break
	// 		}

	// 		a.State = fmt.Sprintf("%s - %s", event.EventType, a.State)
	// 	}

	// 	if server.IsFinalStretch() {
	// 		a.State = fmt.Sprintf("Final Stretch - %s", a.State)
	// 	}
	// }

	return a
}

func discardFor(d time.Duration) {
	discard = true
	time.AfterFunc(d, func() {
		discard = false
	})
}

func Close() {
	rpc.cleanup()
	notify.Feed(nrgba.Discord, "[Discord] Connection closed")
}

func Open() error {
	go func() {
		last := Activity()

		for ; ; time.Sleep(time.Second * 5) {
			reconnect()

			curr := Activity()
			if !discard {
				last = curr
			}

			rpc.send(frame{
				Cmd: commandSetActivity,
				Args: args{
					Pid:      os.Getpid(),
					Activity: last,
				},
				Nonce: uuid.New().String(),
			})
		}
	}()

	return nil
}

func reconnect() {
	err := rpc.error()
	if err != nil {
		notify.Warn("[Discord] Disconnected (%v)", err)
	}

	if config.Current.Advanced.Discord.Disabled && rpc.conn != nil {
		rpc.cleanup()
	}

	retries := 0

	for wait := time.Second; rpc.conn == nil; time.Sleep(wait) {
		if config.Current.Advanced.Discord.Disabled || config.Current.Remember.Discord == config.DiscordStandby {
			wait = time.Second
			continue
		}
		wait = wait << 1

		notify.Feed(nrgba.Discord, "[Discord] Connecting...")

		if retries++; retries == 5 {
			notify.Warn("[Discord] Exhausted connection attempts. RPC has been disabled")
			config.Current.Advanced.Discord.Disabled = true
			continue
		}

		rpc, err = connect()
		if err != nil {
			notify.Warn("[Discord] Failed to connect (%v)", err)
			continue
		}

		rpc.handshake(id)

		err = rpc.error()
		if err != nil {
			notify.Warn("[Discord] Handshake error (%v)", err)
			continue
		}

		notify.Feed(nrgba.Discord, "[Discord] Connected")
	}
}
