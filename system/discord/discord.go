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
	"github.com/pidgy/unitehud/core/team"
	"github.com/pidgy/unitehud/exe"
	"github.com/pidgy/unitehud/gui/is"
)

var (
	Asked   = config.Current.Remember.Discord != config.DiscordRememberStandby
	Current = def

	def = activity{
		State: "Waiting for match to start",

		Details: "UniteHUD - Main Menu",

		Assets: assets{
			LargeImage: "icon1024x1024",
			LargeText:  "UniteHUD",
			SmallImage: "-",
			SmallText:  "UniteHUD.dev",
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

	rpc client

	replace = true
)

func current() activity {
	a := Current
	if !replace {
		return a
	}

	switch is.Now {
	case is.Loading:
		a.Details = fmt.Sprintf("UniteHUD - %s", is.Now)
		a.State = "Loading..."
		return a
	case is.Configuring:
		a.Details = fmt.Sprintf("UniteHUD - %s", is.Now)
		a.State = "Configuring capture settings"
		return a
	}

	if !server.Ready() {
		a.Details = "UniteHUD - Main Menu / Paused"
		a.State = "Waiting for match to start"
		return a
	}

	// Confirmed server is running, looking for matches.
	// Determine if a game is ongoing.

	a.Details = "UniteHUD - In a Match"

	a.Instance = activityInstanceInMatch

	orange, purple, _ := server.Scores()

	wl := "Tied"
	switch {
	case purple > orange:
		wl = "Winning"
	case purple < orange:
		wl = "Losing"
	}
	a.State = fmt.Sprintf("%s %d - %d", wl, purple, orange)

	ten, ms := (time.Minute * 10).Milliseconds(), server.Seconds()*1000
	started := ten - ms
	if ms > 0 {
		a.Timestamps = timestamps{
			Start: time.Now().UnixMilli() - started,
			End:   time.Now().UnixMilli() + (ten - started),
		}
	}

	// If were not in a match, confirm it hasnt ended recently.
	events := state.Past(time.Second*10,
		state.MatchStarting,
		state.MatchEnding,
		state.SurrenderOrange,
		state.SurrenderPurple,
		state.RegiceSecureOrange,
		state.RegiceSecurePurple,
		state.RegisteelSecureOrange,
		state.RegisteelSecurePurple,
		state.RegirockSecureOrange,
		state.RegirockSecurePurple,
		state.RegielekiSecureOrange,
		state.RegielekiSecurePurple,
		state.FinalObjectiveGroudonSecureOrange,
		state.FinalObjectiveGroudonSecurePurple,
	)

	ignoreFinalStretch := false
	defer func() {
		if ignoreFinalStretch {
			return
		}
		if server.IsFinalStretch() {
			a.State = fmt.Sprintf("Final Stretch - %s", a.State)
		}
	}()

	// Determine if match is starting or ending.
	for _, event := range events {
		switch event.EventType {
		case state.MatchStarting:
			dontReplaceFor(time.Second * 30)

			a.Details = "UniteHUD - Match Starting"
			a.State = "Loading..."
			a.Timestamps = timestamps{}

			return a
		case state.SurrenderOrange:
			dontReplaceFor(time.Second * 30)

			a.Details = "UniteHUD - Match Ending"
			a.State = fmt.Sprintf("Won %d - %d (Surrender)", purple, orange)

			return a
		case state.SurrenderPurple:
			dontReplaceFor(time.Second * 30)

			a.Details = "UniteHUD - Match Ending"
			a.State = fmt.Sprintf("Lost %d - %d (Surrender)", purple, orange)

			return a
		case state.MatchEnding:
			dontReplaceFor(time.Second * 30)

			wl := "Tied"
			switch {
			case purple > orange:
				wl = "Won"
			case purple < orange:
				wl = "Lost"
			}

			a.Details = "UniteHUD - Match Ended"
			a.State = fmt.Sprintf("%s %d - %d", wl, purple, orange)

			a.Timestamps = timestamps{}

			return a
		}
	}

	// Could be middle of a match, check for objectives.
	for _, event := range events {
		switch e := event.EventType; e {
		case state.FinalObjectiveGroudonSecureOrange, state.RegisteelSecureOrange, state.RegiceSecureOrange, state.RegirockSecureOrange, state.RegielekiSecureOrange,
			state.FinalObjectiveGroudonSecurePurple, state.RegisteelSecurePurple, state.RegiceSecurePurple, state.RegirockSecurePurple, state.RegielekiSecurePurple:

			dontReplaceFor(time.Second * 10)

			obj := "Objective"
			strs := strings.Split(e.String(), " ")
			if len(strs) > 1 {
				obj = strs[1]
			}

			wl := "lost"
			if e.Team() == team.Purple {
				wl = "secured"
			}

			a.State = fmt.Sprintf("%d - %d / %s %s", purple, orange, obj, wl)
			return a
		}
	}

	if server.Seconds() == 0 {
		a.Details = "UniteHUD - Main Menu"
		a.State = "Waiting for next match to start"
		a.Timestamps = timestamps{
			Start: exe.Uptime.UnixMilli(),
			End:   0,
		}
		a.Instance = activityInstanceIdle

		ignoreFinalStretch = true
	}

	return a
}

func dontReplaceFor(d time.Duration) {
	replace = false
	time.AfterFunc(d, func() {
		replace = true
	})
}

func Connected() bool {
	return rpc.conn != nil
}

func Close() {
	rpc.cleanup()
	notify.Feed(nrgba.Discord, "[Discord] Connection closed")
}

func Open() error {

	go func() {
		for ; ; time.Sleep(time.Second * 5) {
			reconnect()

			Current = current()

			rpc.send(frame{
				Cmd: commandSetActivity,
				Args: args{
					Pid:      os.Getpid(),
					Activity: Current,
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

	for wait := time.Second * 5; rpc.conn == nil; time.Sleep(wait) {
		if config.Current.Advanced.Discord.Disabled || !Asked {
			continue
		}

		notify.Feed(nrgba.Discord, "[Discord] Connecting...")

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
