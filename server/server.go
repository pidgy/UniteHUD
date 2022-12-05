package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"nhooyr.io/websocket"

	"github.com/pidgy/unitehud/global"
	"github.com/pidgy/unitehud/notify"
	"github.com/pidgy/unitehud/rgba"
	"github.com/pidgy/unitehud/state"
	"github.com/pidgy/unitehud/team"
)

const Address = "127.0.0.1:17069"

type Pipe struct {
	game
	tx       int
	requests int

	clients map[string]time.Time

	mutex *sync.Mutex
}

type Score struct {
	Team  string `json:"team"`
	Value int    `json:"value"`
}

type game struct {
	Purple    Score    `json:"purple"`
	Orange    Score    `json:"orange"`
	Self      Score    `json:"self"`
	Seconds   int      `json:"seconds"`
	Balls     int      `json:"balls"`
	Regilekis []string `json:"regis"`
	Started   bool     `json:"started"`
	Stacks    int      `json:"stacks"`

	Version string `json:"version"`
}

var pipe *Pipe

func Balls(b int) {
	pipe.game.Balls = b
}

func Clear() {
	log.Debug().Object("game", pipe.game).Msg("clearing")

	started := pipe.game.Started
	pipe.game = newGame()
	pipe.game.Started = started
}

func Clock() string {
	return fmt.Sprintf("%02d:%02d", pipe.game.Seconds/60, pipe.game.Seconds%60)
}

func Clients() int {
	pipe.mutex.Lock()
	defer pipe.mutex.Unlock()

	for c := range pipe.clients {
		if time.Since(pipe.clients[c]) > time.Second*5 {
			notify.Feed(rgba.SlateGray, "Client %s has disconnected", c)
			delete(pipe.clients, c)
		}
	}

	return len(pipe.clients)
}

func IsFinalStretch() bool {
	return pipe.game.Seconds != 0 && pipe.game.Seconds <= 120
}

func (p *Pipe) client(r *http.Request, route string, raw []byte) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	ip := strings.Split(r.RemoteAddr, ":")[0]
	key := fmt.Sprintf("%s -> %s", ip, r.URL)

	_, ok := p.clients[key]
	if !ok {
		notify.System("Server accepted a new %s connection from %s", route, key)
		log.Debug().RawJSON("response", raw).Str("client", key).Msg("json response")
	}

	p.clients[key] = time.Now()
}

func Publish(t *team.Team, value int) {
	s := Score{
		Team:  t.Name,
		Value: value,
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
		pipe.game.Stacks++
	case team.First.Name:
		switch team.First.Alias {
		case team.Purple.Name:
			pipe.game.Purple.Value += s.Value
		case team.Orange.Name:
			pipe.game.Orange.Value += s.Value
		default:
			notify.Error("Server received first goal from an unknown team")
		}
	}
}

func PublishRegieleki(t *team.Team) {
	for i, t2 := range pipe.Regilekis {
		if t2 == team.None.Name {
			pipe.game.Regilekis[i] = t.Name
			return
		}
	}

	pipe.game.Regilekis[0] = t.Name
	pipe.game.Regilekis[1] = team.None.Name
	pipe.game.Regilekis[2] = team.None.Name
}

func Regielekis() []string {
	return pipe.game.Regilekis
}

func RegielekiAdv() *team.Team {
	p := 0
	o := 0

	for _, t := range pipe.game.Regilekis {
		switch t {
		case team.Purple.Name:
			p++
		case team.Orange.Name:
			o++
		}
	}

	switch {
	case p > o:
		return team.Purple
	case o > p:
		return team.Orange
	default:
		return team.None
	}
}

func RegielekisSecured(t *team.Team) int {
	n := 0
	for _, r := range pipe.game.Regilekis {
		if r == t.Name {
			n++
		}
	}
	return n
}

func Seconds() int {
	return pipe.game.Seconds
}

func Scores() (orange, purple, self int) {
	return pipe.game.Orange.Value, pipe.game.Purple.Value, pipe.game.Self.Value
}

func Start() error {
	pipe = &Pipe{
		game:    newGame(),
		clients: map[string]time.Time{},
		mutex:   &sync.Mutex{},
	}

	http.Handle("/ws", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			OriginPatterns:     []string{"127.0.0.1", "localhost", "0.0.0.0"},
			InsecureSkipVerify: true,
		})
		if err != nil {
			notify.Error("Server failed to accept websocket connection (%v)", err)
			return
		}
		defer c.Close(websocket.StatusNormalClosure, "cross origin WebSocket accepted")

		raw, err := json.Marshal(pipe.game)
		if err != nil {
			notify.Error("Server failed to create server response (%v)", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		err = c.Write(context.Background(), websocket.MessageText, raw)
		if err != nil {
			notify.Error("Server failed to send server response (%v)", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		pipe.client(r, "/ws", raw)

		pipe.tx += len(raw)
		pipe.requests++
	}))

	http.HandleFunc("/http", func(w http.ResponseWriter, r *http.Request) {
		raw, err := json.Marshal(pipe.game)
		if err != nil {
			notify.Error("Server failed to create server response (%v)", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		log.Debug().
			RawJSON("response", raw).
			Str("client", r.RemoteAddr).
			Msg("http response")

		_, err = w.Write(raw)
		if err != nil {
			notify.Error("Server failed to send server response (%v)", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		pipe.client(r, "/http", raw)
		pipe.tx += len(raw)
		pipe.requests++
	})

	go func() {
		for range time.NewTicker(time.Minute * 5).C {
			if pipe.requests < 1 {
				continue
			}

			notify.System("Server is sending an average of %d bytes per request", pipe.tx/pipe.requests)
		}
	}()

	state.Add(state.ServerStarted, Clock(), -1)

	errq := make(chan error)
	go func() {
		errq <- http.ListenAndServe(Address, nil)
	}()

	time.AfterFunc(time.Second, func() { errq <- nil })

	return <-errq
}

func Started(s bool) {
	pipe.game.Started = s
}

func Time(minutes, seconds int) {
	if minutes+seconds == 0 {
		return
	}

	pipe.game.Seconds = minutes*60 + seconds
}

func newGame() game {
	g := game{
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
		Seconds:   0,
		Balls:     0,
		Regilekis: []string{team.None.Name, team.None.Name, team.None.Name},
		Version:   global.Version,
	}

	return g
}

// Zerolog.

func (g game) MarshalZerologObject(e *zerolog.Event) {
	e.Object("purple", g.Purple).Object("orange", g.Orange).Int("seconds", g.Seconds)
}

func (s Score) MarshalZerologObject(e *zerolog.Event) {
	e.Str("team", s.Team).Int("value", s.Value)
}
