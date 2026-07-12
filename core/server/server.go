package server

import (
	"context"
	"encoding/json"
	"fmt"
	"image/png"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"nhooyr.io/websocket"

	"github.com/pidgy/unitehud/core/config"
	"github.com/pidgy/unitehud/core/notify"
	"github.com/pidgy/unitehud/core/rgba/nrgba"
	"github.com/pidgy/unitehud/core/state"
	"github.com/pidgy/unitehud/core/team"
	"github.com/pidgy/unitehud/exe"
	"github.com/pidgy/unitehud/media/img"
	"github.com/pidgy/unitehud/media/video"
	"github.com/pidgy/unitehud/media/video/fps"
	"github.com/pidgy/unitehud/system/lang"
)

const (
	// Address is the local bind address for the HTTP/WebSocket server.
	// Address = "127.0.0.1:17069"
	Address = "0.0.0.0:17069"

	// ObjectiveFinal marks the final objective placeholder name.
	ObjectiveFinal = "final" // "rayquaza", "groudon", etc.
	// ObjectiveRegice identifies a Regice objective.
	ObjectiveRegice = "regice"
	// ObjectiveRegidrago identifies a Regidrago objective.
	ObjectiveRegidrago = "regidrago"
	// ObjectiveRegieleki identifies a Regieleki objective.
	ObjectiveRegieleki = "regieleki"
	// ObjectiveRegirock identifies a Regirock objective.
	ObjectiveRegirock = "regirock"
	// ObjectiveRegisteel identifies a Registeel objective.
	ObjectiveRegisteel = "registeel"
)

// Objective is a single objective capture event and metadata.
type Objective struct {
	Name        string `json:"name"`
	Team        string `json:"team"`
	Time        int64  `json:"time"`
	Surrendered bool   `json:"surrendered"`
	IsFinal     bool   `json:"is_final"`
}

// State is the serialized server state exposed to clients.
type State struct {
	Objectives         []Objective `json:"objectives"`
	Config             bool        `json:"config"`
	Debug              bool        `json:"debug"`
	Defeated           []int64     `json:"defeated"`
	Energy             int         `json:"energy"`
	Events             []string    `json:"events"`
	FinalObjectiveTeam string      `json:"final_objective"`
	InMatch            bool        `json:"match"`
	Orange             *score      `json:"orange"`
	Purple             *score      `json:"purple"`
	Platform           string      `json:"platform"`
	Ready              bool        `json:"ready"`
	Regilekis          []string    `json:"regis"`
	Regidragos         []string    `json:"regidragos"`
	Seconds            int64       `json:"seconds"`
	Self               *score      `json:"self"`
	Stacks             int         `json:"stacks"`
	Version            string      `json:"version"`

	lastSecondsUpdate time.Time
}

// info holds server runtime stats and tracked clients.
type info struct {
	*State

	tx       int
	requests int
	duration time.Duration

	clients map[string]time.Time

	mutex *sync.Mutex
}

// score is a team's score snapshot.
type score struct {
	Team        string `json:"team"`
	Value       int    `json:"value"`
	KOs         int    `json:"kos"`
	Surrendered bool   `json:"surrendered"`
}

var (
	// current is the live server state and runtime tracker.
	current = &info{
		State:   reset(),
		clients: map[string]time.Time{},
		mutex:   &sync.Mutex{},
	}

	// track caches the last serialized payloads.
	track struct {
		ws   []byte
		http []byte
	}
)

// Clear resets server state while preserving readiness.
func Clear() {
	started := current.State.Ready
	current.State = reset()
	current.State.Ready = started
}

// Clients returns the number of active clients.
func Clients() int {
	current.mutex.Lock()
	defer current.mutex.Unlock()

	for c := range current.clients {
		if time.Since(current.clients[c]) > time.Second*5 {
			notify.FeedUnique(nrgba.Slate, "[Server] Client disconnected (%s)", c)
			delete(current.clients, c)
		}
	}

	return len(current.clients)
}

// Clock returns the match clock as MM:SS.
func Clock() string {
	return fmt.Sprintf("%02d:%02d", current.State.Seconds/60, current.State.Seconds%60)
}

// FinalObjectiveName returns the name of the final objective, if present.
func FinalObjectiveName() string {
	if current.State.FinalObjectiveTeam != "" {
		for _, o := range current.Objectives {
			if o.IsFinal {
				return o.Name
			}
		}
	}
	return "Final Objective"
}

// Holding returns the current energy held.
func Holding() int {
	return current.State.Energy
}

// IsFinalStretch reports whether the condition holds.
func IsFinalStretch() bool {
	if current.State.Seconds == 0 || current.State.Seconds >= 130 {
		return false
	}

	// Edge case to handle scoring at exactly 2:00 and missing time update.
	if time.Since(current.lastSecondsUpdate).Seconds() >= float64(current.State.Seconds-130) {
		return true
	}

	return current.State.Seconds > 0 && current.State.Seconds < 121
}

// KOs returns the KO count for the provided team.
func KOs(t *team.Team) int {
	switch t.Name {
	case team.Purple.Name:
		return current.State.Purple.KOs
	case team.Orange.Name:
		return current.State.Orange.KOs
	default:
		return 0
	}
}

// Open starts the HTTP/WebSocket server and registers handlers.
func Open() error {
	http.Handle("/stream", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		current.client(r)

		w.Header().Add(
			"Content-Type",
			"multipart/x-mixed-replace; boundary=frame",
		)

		boundary := "\r\n--frame\r\nContent-Type: image/png\r\n\r\n"

		enc := &png.Encoder{
			CompressionLevel: png.NoCompression,
			BufferPool:       img.NewPNGPool(),
		}

		defer fps.NewLoop(&fps.LoopOptions{
			FPS: 120,
			On: func(min, max, avg time.Duration) (close bool) {
				defer notify.Debug("HTTP /stream min=%s, max=%s, avg=%s\n", min, max, avg)

				img, err := video.Capture()
				if err != nil {
					notify.Error("[Server] /stream (%v)", err)
					return true
				}

				n, err := io.WriteString(w, boundary)
				if err != nil {
					notify.Error("[Server] /stream (%v)", err)
					return true
				}
				if n != len(boundary) {
					notify.Error("[Server] /stream (failed to add boundary header)")
					return true
				}

				err = enc.Encode(w, img)
				if err != nil {
					notify.Error("[Server] /stream (%v)", err)
					return true
				}

				n, err = io.WriteString(w, "\r\n")
				if err != nil || n != 2 {
					return true
				}

				return false
			},
		}).Stop()
	}))

	http.Handle("/ws", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error

		now := current.client(r)

		track.ws, err = json.Marshal(current.State)
		if err != nil {
			notify.Error("[Server] Failed to create WebSocket response (%v)", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		c, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			OriginPatterns:     []string{"127.0.0.1", "localhost", "0.0.0.0"},
			InsecureSkipVerify: true,
		})
		if err != nil {
			notify.Error("[Server] WebSocket connection failed (%v)", err)
			return
		}
		defer c.Close(websocket.StatusNormalClosure, "cross origin WebSocket accepted")

		current.State.Platform = config.Current.Gaming.Device
		current.State.Events = notify.LastNStrings(10)
		current.State.Debug = exe.Debug

		err = c.Write(context.Background(), websocket.MessageText, track.ws)
		if err != nil {
			notify.Error("[Server] Failed to send WebSocket response (%v)", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if current.State.Ready {
			current.tx += len(track.ws)
			current.duration += time.Since(now)
			current.requests++
		}
	}))

	http.HandleFunc("/http", func(w http.ResponseWriter, r *http.Request) {
		var err error

		now := current.client(r)

		track.http, err = json.Marshal(current.State)
		if err != nil {
			notify.Error("[Server] Failed to create HTTP response (%v)", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		current.State.Platform = config.Current.Gaming.Device
		current.State.Events = notify.LastNStrings(10)
		current.State.Debug = exe.Debug

		_, err = w.Write(track.http)
		if err != nil {
			notify.Error("[Server] Failed to send HTTP response (%v)", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if current.State.Ready {
			current.tx += len(track.http)
			current.duration += time.Since(now)
			current.requests++
		}
	})

	go listen()

	state.Add(state.ServerStarted, Clock(), -1)
	notify.System("[Server] Listening on %s", Address)

	go metrics()

	return nil
}

// Match reports whether a match is currently in progress.
func Match() bool {
	return current.State.InMatch
}

// Objectives returns secured objective counts for a team.
func Objectives(t *team.Team) (regielekis, regices, regirocks, registeels, regidragos, finals int) {
	if current.State.FinalObjectiveTeam == t.Name {
		finals = 1
	}
	return RegielekisSecured(t), RegicesSecured(t), RegirocksSecured(t), RegisteelsSecured(t), RegidragosSecured(t), finals
}

// ObjectivesSecured returns all recorded objectives.
func ObjectivesSecured() []Objective {
	return current.State.Objectives
}

// Ready reports whether the server is ready to serve state.
func Ready() bool {
	return current.State.Ready
}

// RegicesSecured returns the number of Regice secured by team.
func RegicesSecured(t *team.Team) int {
	n := 0
	for _, b := range current.State.Objectives {
		if b.Name == ObjectiveRegice && b.Team == t.Name {
			n++
		}
	}
	return n
}

// RegidragosSecured returns the number of Regidrago secured by team.
func RegidragosSecured(t *team.Team) int {
	n := 0
	for _, b := range current.State.Objectives {
		if b.Name == ObjectiveRegidrago && b.Team == t.Name {
			n++
		}
	}
	return n
}

// RegielekiAdv returns which team has Regieleki advantage.
func RegielekiAdv() *team.Team {
	p := 0
	o := 0

	for _, t := range current.State.Regilekis {
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

// Regielekis returns the list of Regieleki captures.
func Regielekis() []string {
	return current.State.Regilekis
}

// RegielekisSecured returns the number of Regieleki secured by team.
func RegielekisSecured(t *team.Team) int {
	n := 0
	for _, r := range current.State.Regilekis {
		if r == t.Name {
			n++
		}
	}
	return n
}

// RegirocksSecured returns the number of Regirock secured by team.
func RegirocksSecured(t *team.Team) int {
	n := 0
	for _, b := range current.State.Objectives {
		if b.Name == ObjectiveRegirock && b.Team == t.Name {
			n++
		}
	}
	return n
}

// RegisteelsSecured returns the number of Registeel secured by team.
func RegisteelsSecured(t *team.Team) int {
	n := 0
	for _, b := range current.State.Objectives {
		if b.Name == ObjectiveRegisteel && b.Team == t.Name {
			n++
		}
	}
	return n
}

// Score returns the current score for the provided team.
func Score(t *team.Team) int {
	switch t {
	case team.Purple:
		return current.State.Purple.Value
	case team.Orange:
		return current.State.Orange.Value
	case team.Self:
		return current.State.Self.Value
	default:
		return -1
	}
}

// ScoreString formats the team score and surrender status.
func ScoreString(t *team.Team) string {
	switch t {
	case team.Purple:
		if current.State.Purple.Surrendered {
			return fmt.Sprintf("%d [SND]", current.State.Purple.Value)
		}
		return fmt.Sprintf("%d", current.State.Purple.Value)
	case team.Orange:
		if current.State.Orange.Surrendered {
			return fmt.Sprintf("%d [SND]", current.State.Orange.Value)
		}
		return fmt.Sprintf("%d", current.State.Orange.Value)
	}
	return fmt.Sprintf("0 (Unknown Team %s)", t)
}

// Scores returns the orange, purple, and self scores.
func Scores() (orange, purple, self int) {
	return current.State.Orange.Value, current.State.Purple.Value, current.State.Self.Value
}

// SetBottomObjective updates a bottom-lane objective entry at index n.
func SetBottomObjective(t *team.Team, name string, n int) {
	o := Objective{
		Team: t.Name,
		Name: name,
		Time: time.Now().Unix(),
	}

	op := fmt.Sprintf("[%s] %s #%d", lang.Title(t.Name), lang.Title(o.Name), n+1)

	switch {
	// Illegal.
	case len(current.Objectives) < n:
		notify.Warn("[Server] %s illegal operation (no index)", op)

	// Remove.
	case len(current.Objectives) == n+1 && current.Objectives[n].Team == t.Name && current.Objectives[n].Name == o.Name:
		// Remove last objective.
		current.Objectives = current.Objectives[:n]
		notify.Unique(t.NRGBA, "[Server] %s removed", op)

	// Add.
	case len(current.Objectives) == n:
		current.Objectives = append(current.Objectives, o)
		notify.Unique(t.NRGBA, "[Server] %s secured", op)
	case len(current.Objectives) > n+1 && current.Objectives[n].Team != t.Name:
		current.Objectives[n] = o
		notify.Unique(t.NRGBA, "[Server] %s secure replaced", op)

		// Overwrite.
	case len(current.Objectives) == n+1 && current.Objectives[n].Team == t.Name && current.Objectives[n].Name != o.Name:
		// Replace between first and last.
		fallthrough
	case len(current.Objectives) > n+1 && current.Objectives[n].Team == t.Name:
		fallthrough
	case len(current.Objectives) == n+1 && current.Objectives[n].Team != t.Name:
		// Overwrite last objective.
		current.Objectives[n] = o
		notify.Unique(t.NRGBA, "[Server] %s secure replaced", op)
	}
}

// SetConfig toggles the config view flag in state.
func SetConfig(c bool) {
	current.State.Config = c
}

// SetDefeated records the current time as a defeat.
func SetDefeated() {
	current.State.Defeated = append(current.State.Defeated, current.State.Seconds)
}

// SetEnergy updates the current energy value.
func SetEnergy(b int) {
	current.State.Energy = b
}

// SetFinalObjective records the final objective capture for a team.
func SetFinalObjective(t *team.Team, e state.EventType) {
	current.State.FinalObjectiveTeam = t.Name

	current.Objectives = append(current.Objectives, Objective{
		Team:    t.Name,
		Name:    e.ObjectiveString(),
		Time:    time.Now().Unix(),
		IsFinal: true,
	})
}

// SetKO increments the KO count for a team.
func SetKO(t *team.Team) {
	switch t.Name {
	case team.Purple.Name:
		current.State.Purple.KOs++
	case team.Orange.Name:
		current.State.Orange.KOs++
	}
}

// SetMatchStarted marks the match as in progress.
func SetMatchStarted() {
	current.State.InMatch = true
}

// SetMatchStopped marks the match as not in progress.
func SetMatchStopped() {
	current.State.InMatch = false
}

// SetRegice records a Regice objective capture.
func SetRegice(t *team.Team) {
	current.Objectives = append(current.Objectives, Objective{
		Team: t.Name,
		Name: ObjectiveRegice,
		Time: time.Now().Unix(),
	})
}

// SetRegidrago records a Regidrago objective capture.
func SetRegidrago(t *team.Team) {
	current.Objectives = append(current.Objectives, Objective{
		Team: t.Name,
		Name: ObjectiveRegidrago,
		Time: time.Now().Unix(),
	})
}

// SetRegieleki records a Regieleki objective capture and rotation.
func SetRegieleki(t *team.Team) {
	current.Objectives = append(current.Objectives, Objective{
		Team: t.Name,
		Name: ObjectiveRegieleki,
		Time: time.Now().Unix(),
	})

	for i, t2 := range current.Regilekis {
		if t2 == team.None.Name {
			current.State.Regilekis[i] = t.Name
			return
		}
	}

	current.State.Regilekis[0] = t.Name
	current.State.Regilekis[1] = team.None.Name
	current.State.Regilekis[2] = team.None.Name
}

// SetRegirock records a Regirock objective capture.
func SetRegirock(t *team.Team) {
	current.Objectives = append(current.Objectives, Objective{
		Team: t.Name,
		Name: ObjectiveRegirock,
		Time: time.Now().Unix(),
	})
}

// SetRegisteel records a Registeel objective capture.
func SetRegisteel(t *team.Team) {
	current.Objectives = append(current.Objectives, Objective{
		Team: t.Name,
		Name: ObjectiveRegisteel,
		Time: time.Now().Unix(),
	})
}

// SetNotReady marks the server as not ready and emits a stop event.
func SetNotReady() {
	current.State.Ready = false
	state.Add(state.ServerStopped, Clock(), -1)
}

// SetReady marks the server as ready and emits a start event.
func SetReady() {
	current.State.Ready = true
	state.Add(state.ServerStarted, Clock(), -1)
}

// SetScore applies a score change for the provided team.
func SetScore(t *team.Team, v int) {
	s := score{
		Team:  t.Name,
		Value: v,
	}

	switch t.Name {
	case team.Purple.Name:
		current.State.Purple.Value += s.Value
	case team.Orange.Name:
		current.State.Orange.Value += s.Value
	case team.Self.Name:
		current.State.Purple.Value += s.Value
		current.State.Self.Value += s.Value
		current.State.Stacks++
	case team.First.Name:
		switch team.First.Alias {
		case team.Purple.Name:
			current.State.Purple.Value += s.Value
		case team.Orange.Name:
			current.State.Orange.Value += s.Value
		default:
			notify.Error("[Server] Received first goal from an unknown team")
		}
	}
}

// SetScoreSurrendered marks a team's score as surrendered.
func SetScoreSurrendered(t *team.Team) {
	switch t {
	case team.Purple:
		current.State.Purple.Surrendered = true
	case team.Orange:
		current.State.Orange.Surrendered = true
	}
}

// SetTime updates the match timer and in-match state.
func SetTime(minutes, seconds int64) {
	current.State.lastSecondsUpdate = time.Now()

	if minutes+seconds == 0 {
		current.State.InMatch = false
		return
	}

	current.State.InMatch = true

	current.State.Seconds = minutes*60 + seconds
}

// Seconds returns the current match seconds remaining.
func Seconds() int64 {
	return current.State.Seconds
}

// client registers or refreshes a client connection and returns its timestamp.
func (i *info) client(r *http.Request) time.Time {
	i.mutex.Lock()
	defer i.mutex.Unlock()

	key := fmt.Sprintf("%s -> %s", strings.Split(r.RemoteAddr, ":")[0], r.URL)

	_, ok := i.clients[key]
	if !ok {
		notify.FeedUnique(nrgba.Slate, "[Server] Client connected (%s)", key)
	}

	i.clients[key] = time.Now()

	return i.clients[key]
}

// listen starts the HTTP server and panics on failure.
func listen() {
	panic(http.ListenAndServe(Address, nil))
}

// metrics periodically reports server performance metrics.
func metrics() {
	for ; ; time.Sleep(time.Minute * 30) {
		if current.requests < 2 {
			notify.System("[Server] Awaiting connection...")
			continue
		}

		if current.State.Ready {
			notify.System(
				"[Detect] Averaging %s / %.1fkB latency",
				current.duration/time.Duration(current.requests),
				float64(current.tx)/float64(current.requests)/1000,
			)
		}
	}
}

// reset builds a fresh State with default values.
func reset() *State {
	return &State{
		Purple: &score{
			Team:        team.Purple.Name,
			Value:       0,
			Surrendered: false,
		},
		Orange: &score{
			Team:        team.Orange.Name,
			Value:       0,
			Surrendered: false,
		},
		Self: &score{
			Team:        team.Self.Name,
			Value:       0,
			Surrendered: false,
		},
		Seconds:            0,
		Energy:             0,
		Regilekis:          []string{team.None.Name, team.None.Name, team.None.Name},
		FinalObjectiveTeam: "",
		Objectives:         []Objective{},
		Version:            exe.Version,
		Defeated:           []int64{},
	}
}
