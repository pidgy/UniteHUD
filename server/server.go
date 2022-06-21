package server

import (
	"context"
	"encoding/json"
	"fmt"
	"image/color"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"nhooyr.io/websocket"

	"github.com/pidgy/unitehud/notify"
	"github.com/pidgy/unitehud/rgba"
	"github.com/pidgy/unitehud/team"
)

const Address = "127.0.0.1:17069"

type Pipe struct {
	game
	tx       int
	requests int

	clients map[string]time.Time
	mutex   *sync.Mutex
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

func Clients() int {
	pipe.mutex.Lock()
	defer pipe.mutex.Unlock()

	for c := range pipe.clients {
		if time.Since(pipe.clients[c]) > time.Second*5 {
			notify.Feed(rgba.SlateGray, "%s has disconnected", c)
			delete(pipe.clients, c)
		}
	}

	return len(pipe.clients)
}

func IsFinalStretch() bool {
	return pipe.game.Seconds != 0 && pipe.game.Seconds <= 120
}

func Start() {
	pipe = &Pipe{
		game:    newGame(),
		clients: map[string]time.Time{},
		mutex:   &sync.Mutex{},
	}

	http.Handle("/ws", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pipe.client(r, fmt.Sprintf("%s -> %s", strings.Split(r.RemoteAddr, ":")[0], r.URL), "/ws")

		c, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			OriginPatterns:     []string{"127.0.0.1", "localhost", "0.0.0.0"},
			InsecureSkipVerify: true,
		})
		if err != nil {
			notify.Feed(rgba.Red, err.Error())
			return
		}
		defer c.Close(websocket.StatusNormalClosure, "cross origin WebSocket accepted")

		raw, err := json.Marshal(pipe.game)
		if err != nil {
			log.Error().Err(err).Str("route", "/ws").Object("game", pipe.game).Msg("failed to marshal game update")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		err = c.Write(context.Background(), websocket.MessageText, raw)
		if err != nil {
			log.Error().Err(err).Str("route", "/ws").Object("game", pipe.game).Msg("failed to write game update")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		pipe.tx += len(raw)
		pipe.requests++
	}))

	http.HandleFunc("/http", func(w http.ResponseWriter, r *http.Request) {
		pipe.client(r, fmt.Sprintf("%s -> %s", r.RemoteAddr, r.URL), "/http")

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
	})

	go func() {
		for range time.NewTicker(time.Minute * 5).C {
			if pipe.requests < 1 {
				continue
			}

			notify.Feed(color.RGBA(rgba.SlateGray), "Server is sending ~%d bytes per request", pipe.tx/pipe.requests)
		}
	}()

	err := http.ListenAndServe(Address, nil)
	if err != nil {
		log.Fatal().Err(err).Str("addr", Address).Msg("socket server encountered a fatal error")
	}
}

func (p *Pipe) client(r *http.Request, key, route string) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	_, ok := p.clients[key]
	if !ok {
		notify.Feed(rgba.White, "Accepting new %s connection from %s", route, key)
		log.Debug().Str("route", route).Str("remote", r.RemoteAddr).Msg("received")
	}

	p.clients[key] = time.Now()
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

func Seconds() int {
	return pipe.game.Seconds
}

func Scores() (purple, orange, self int) {
	return pipe.game.Purple.Value, pipe.game.Orange.Value, pipe.game.Self.Value
}

func Time(minutes, seconds int) {
	if minutes+seconds == 0 {
		return
	}

	pipe.game.Seconds = minutes*60 + seconds
}

/*
func score(ws *websocket.Conn) {
	defer ws.Close()

	log.Debug().Str("route", "/ws").Stringer("remote", ws.RemoteAddr()).Msg("request received")

	p.mutex.Lock()
	req := fmt.Sprintf("%s -> %s", ws.RemoteAddr(), ws.Request().URL)
	ok := p.clients[req]
	if !ok {
		p.clients[req] = true
		notify.Feed(rgba.White, "Accepting new websocket connection from %s", req)
	}
	p.mutex.Unlock()

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
*/
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
		Balls:   0,
	}
}

// Zerolog.

func (g game) MarshalZerologObject(e *zerolog.Event) {
	e.Object("purple", g.Purple).Object("orange", g.Orange).Int("seconds", g.Seconds)
}

func (s Score) MarshalZerologObject(e *zerolog.Event) {
	e.Str("team", s.Team).Int("value", s.Value)
}
