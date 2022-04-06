package server

import (
	"encoding/json"
	"fmt"
	"image/color"
	"net/http"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"golang.org/x/net/websocket"

	"github.com/pidgy/unitehud/notify"
	"github.com/pidgy/unitehud/rgba"
	"github.com/pidgy/unitehud/team"
)

type Pipe struct {
	game
	tx       int
	requests int
}

type game struct {
	Purple  Score `json:"purple"`
	Orange  Score `json:"orange"`
	Self    Score `json:"self"`
	Seconds int   `json:"seconds"`
	Balls   int   `json:"balls"`
}

type Score struct {
	Team  string `json:"team"`
	Value int    `json:"value"`
}

var pipe *Pipe

var (
	clients   = map[string]bool{}
	clientTex = &sync.Mutex{}
)

func New(addr string) {
	pipe = &Pipe{
		game: newGame(),
	}

	http.Handle("/ws", websocket.Handler(score))

	http.HandleFunc("/http", func(w http.ResponseWriter, r *http.Request) {
		clientTex.Lock()
		req := fmt.Sprintf("%s -> %s", r.RemoteAddr, r.URL)
		ok := clients[req]
		if !ok {
			clients[req] = true
			notify.Feed(rgba.White, "Accepting new http connection from %s", req)
		}
		clientTex.Unlock()

		log.Debug().Str("route", "/http").Str("remote", r.RemoteAddr).Msg("received")

		raw, err := json.Marshal(pipe.game)
		if err != nil {
			log.Error().Err(err).Str("route", "/http").Object("game", pipe.game).Msg("failed to marshal game update")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		_, err = w.Write(raw)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Error().Err(err).Object("game", pipe.game).Str("route", "/http").Msg("failed to marshal game update")
			return
		}

		pipe.tx += len(raw)
		pipe.requests++

		log.Debug().Str("route", "/http").Str("remote", r.RemoteAddr).RawJSON("raw", raw).Msg("served")
	})

	go func() {
		err := http.ListenAndServe(addr, nil)
		if err != nil {
			log.Fatal().Err(err).Str("addr", addr).Msg("socket server encountered a fatal error")
		}
	}()

	go func() {
		for range time.NewTicker(time.Minute * 5).C {
			if pipe.requests < 1 {
				continue
			}

			notify.Feed(color.RGBA(rgba.SlateGray), "Server is sending ~%d bytes per request", pipe.tx/pipe.requests)
		}
	}()
}

func Balls(b int) {
	pipe.game.Balls = b
}

func Clear() {
	log.Debug().Object("game", pipe.game).Msg("clearing")
	pipe.game = newGame()
}

func Clock() string {
	return fmt.Sprintf("%02d:%02d", pipe.game.Seconds/60, pipe.game.Seconds%60)
}

func IsFinalStretch() bool {
	return pipe.game.Seconds != 0 && pipe.game.Seconds <= 120
}

func Publish(t *team.Team, value int) {
	s := Score{
		t.Name,
		value,
	}

	log.Debug().Object("score", s).Object("game", pipe.game).Msg("publishing")

	switch t.Name {
	case team.Purple.Name:
		pipe.game.Purple.Value += s.Value
	case team.Orange.Name:
		pipe.game.Orange.Value += s.Value
	case team.Self.Name:
		pipe.game.Purple.Value += s.Value
		pipe.game.Self.Value += s.Value
	case team.First.Name:
		switch team.First.Alias {
		case team.Purple.Name:
			pipe.game.Purple.Value += s.Value
		case team.Orange.Name:
			pipe.game.Orange.Value += s.Value
		default:
			notify.Feed(rgba.Red, "Unknown team scored first goal")
		}
	}
}

func Time(minutes, seconds int) {
	if minutes+seconds == 0 {
		return
	}

	pipe.game.Seconds = minutes*60 + seconds
}

func Scores() (purple, orange, self int) {
	return pipe.game.Purple.Value, pipe.game.Orange.Value, pipe.game.Self.Value
}

func score(ws *websocket.Conn) {
	defer ws.Close()

	log.Debug().Str("route", "/ws").Stringer("remote", ws.RemoteAddr()).Msg("request received")

	clientTex.Lock()
	req := fmt.Sprintf("%s -> %s", ws.RemoteAddr(), ws.Request().URL)
	ok := clients[req]
	if !ok {
		clients[req] = true
		notify.Feed(rgba.White, "Accepting new websocket connection from %s", req)
	}
	clientTex.Unlock()

	raw, err := json.Marshal(pipe.game)
	if err != nil {
		log.Error().Err(err).Str("route", "/ws").Object("game", pipe.game).Msg("failed to marshal game update")
	}

	err = websocket.JSON.Send(ws, raw)
	if err != nil {
		log.Error().Err(err).Str("route", "/ws").Object("game", pipe.game).Stringer("remote", ws.RemoteAddr()).Msg("failed to send game update")
	}

	pipe.tx += len(raw)

	log.Debug().Str("route", "/ws").Stringer("remote", ws.RemoteAddr()).RawJSON("raw", raw).Msg("request served")
}

func newGame() game {
	return game{
		Purple: Score{
			Team:  team.Purple.Name,
			Value: 0,
		},
		Orange: Score{
			Team:  team.Orange.Name,
			Value: 0,
		},
		Self: Score{
			Team:  team.Self.Name,
			Value: 0,
		},
		Seconds: 0,
	}
}

// Zerolog.

func (g game) MarshalZerologObject(e *zerolog.Event) {
	e.Object("purple", g.Purple).Object("orange", g.Orange).Int("seconds", g.Seconds)
}

func (s Score) MarshalZerologObject(e *zerolog.Event) {
	e.Str("team", s.Team).Int("value", s.Value)
}
