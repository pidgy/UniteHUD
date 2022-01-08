package pipe

import (
	"encoding/json"
	"net/http"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"golang.org/x/net/websocket"

	"github.com/pidgy/unitehud/team"
)

/*
	{
		"purple": {
			score: 10
		},
		"orange": {
			score: 10
		},
	}
*/

type Pipe struct {
	game Game
}

type Game struct {
	Purple  Score `json:"purple"`
	Orange  Score `json:"orange"`
	Self    Score `json:"self"`
	Seconds int   `json:"seconds"`
}

type Score struct {
	Team  string `json:"team"`
	Value int    `json:"value"`
}

func New(addr string) *Pipe {
	p := &Pipe{
		game: Game{
			Purple: Score{
				team.Purple.Name,
				0,
			},
			Orange: Score{
				team.Orange.Name,
				0,
			},
			Self: Score{
				team.Self.Name,
				0,
			},
		},
	}

	http.Handle("/ws", websocket.Handler(p.score))

	http.HandleFunc("/http", func(w http.ResponseWriter, r *http.Request) {
		log.Debug().Str("route", "/http").Str("remote", r.RemoteAddr).Msg("received")

		raw, err := json.Marshal(p.game)
		if err != nil {
			log.Error().Err(err).Str("route", "/http").Object("game", p.game).Msg("failed to marshal game update")
		}

		_, err = w.Write(raw)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Error().Err(err).Object("game", p.game).Str("route", "/http").Msg("failed to marshal game update")
		}

		log.Debug().Str("route", "/http").Str("remote", r.RemoteAddr).RawJSON("raw", raw).Msg("served")
	})

	go func() {
		err := http.ListenAndServe(addr, nil)
		if err != nil {
			log.Fatal().Err(err).Str("addr", addr).Msg("socket server encountered a fatal error")
		}
	}()

	return p
}

func (p *Pipe) Clear() {
	if p.game.Purple.Value == 0 && p.game.Orange.Value == 0 {
		return
	}

	log.Info().Object("game", p.game).Msg("clearing")

	p.game = Game{
		Purple: Score{
			team.Purple.Name,
			0,
		},
		Orange: Score{
			team.Orange.Name,
			0,
		},
		Self: Score{
			team.Self.Name,
			0,
		},
	}
}

func (p *Pipe) Publish(t *team.Team, value int) {
	s := Score{
		t.Name,
		value,
	}

	log.Info().Object("score", s).Object("game", p.game).Msg("publishing")

	switch t.Name {
	case team.Purple.Name:
		p.game.Purple.Value += s.Value
	case team.Orange.Name:
		p.game.Orange.Value += s.Value
	case team.Self.Name:
		p.game.Purple.Value += s.Value
		p.game.Self.Value += s.Value
	}
}

func (p *Pipe) Time(minutes, seconds int) {
	p.game.Seconds = minutes*60 + seconds
}

func (p *Pipe) score(ws *websocket.Conn) {
	log.Debug().Str("route", "/ws").Stringer("remote", ws.RemoteAddr()).Msg("request received")

	defer ws.Close()

	raw, err := json.Marshal(p.game)
	if err != nil {
		log.Error().Err(err).Str("route", "/ws").Object("game", p.game).Msg("failed to marshal game update")
	}

	err = websocket.JSON.Send(ws, raw)
	if err != nil {
		log.Error().Err(err).Str("route", "/ws").Object("game", p.game).Stringer("remote", ws.RemoteAddr()).Msg("failed to send game update")
	}

	log.Debug().Str("route", "/ws").Stringer("remote", ws.RemoteAddr()).RawJSON("raw", raw).Msg("request served")
}

func (g Game) MarshalZerologObject(e *zerolog.Event) {
	e.Object("purple", g.Purple).Object("orange", g.Orange)
}

func (s Score) MarshalZerologObject(e *zerolog.Event) {
	e.Str("team", s.Team).Int("value", s.Value)
}
