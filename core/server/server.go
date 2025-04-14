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

	"github.com/pidgy/unitehud/avi/img"
	"github.com/pidgy/unitehud/avi/video"
	"github.com/pidgy/unitehud/avi/video/fps"
	"github.com/pidgy/unitehud/core/config"
	"github.com/pidgy/unitehud/core/notify"
	"github.com/pidgy/unitehud/core/rgba/nrgba"
	"github.com/pidgy/unitehud/core/state"
	"github.com/pidgy/unitehud/core/team"
	"github.com/pidgy/unitehud/exe"
	"github.com/pidgy/unitehud/system/lang"
)

const Address = "127.0.0.1:17069"

type State struct {
	Bottom    []objective `json:"bottom"`
	Config    bool        `json:"config"`
	Debug     bool        `json:"debug"`
	Defeated  []int64     `json:"defeated"`
	Energy    int         `json:"balls"`
	Events    []string    `json:"events"`
	InMatch   bool        `json:"match"`
	Orange    *score      `json:"orange"`
	Purple    *score      `json:"purple"`
	Platform  string      `json:"platform"`
	Rayquaza  string      `json:"rayquaza"`
	Regilekis []string    `json:"regis"`
	Seconds   int64       `json:"seconds"`
	Self      *score      `json:"self"`
	Stacks    int         `json:"stacks"`
	Ready     bool        `json:"ready"`
	Version   string      `json:"version"`

	lastSecondsUpdate time.Time
}

type info struct {
	*State

	tx       int
	requests int
	duration time.Duration

	clients map[string]time.Time

	mutex *sync.Mutex
}

type objective struct {
	Name        string `json:"name"`
	Team        string `json:"team"`
	Time        int64  `json:"time"`
	Surrendered bool   `json:"surrendered"`
}

type score struct {
	Team        string `json:"team"`
	Value       int    `json:"value"`
	KOs         int    `json:"kos"`
	Surrendered bool   `json:"surrendered"`
}

var current = &info{
	State:   reset(),
	clients: map[string]time.Time{},
	mutex:   &sync.Mutex{},
}

var track struct {
	ws   []byte
	http []byte
}

func Bottom() []objective {
	return current.State.Bottom
}

func Clear() {
	started := current.State.Ready
	current.State = reset()
	current.State.Ready = started
}

func Clock() string {
	return fmt.Sprintf("%02d:%02d", current.State.Seconds/60, current.State.Seconds%60)
}

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

func Game() *State {
	return current.State
}

func Holding() int {
	return current.State.Energy
}

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
				if exe.Debug {
					defer fmt.Printf("HTTP /stream min=%s, max=%s, avg=%s\n", min, max, avg)
				}

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

func listen() {
	panic(http.ListenAndServe(Address, nil))
}

func Match() bool {
	return current.State.InMatch
}

func Objectives(t *team.Team) (regielekis, regices, regirocks, registeels, rayquazas int) {
	q := 0
	if current.State.Rayquaza == t.Name {
		q++
	}
	return RegielekisSecured(t), RegicesSecured(t), RegirocksSecured(t), RegisteelsSecured(t), q
}

func Rayquaza() string {
	return current.State.Rayquaza
}

func Ready() bool {
	return current.State.Ready
}
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

func RegicesSecured(t *team.Team) int {
	n := 0
	for _, b := range current.State.Bottom {
		if b.Name == "regice" && b.Team == t.Name {
			n++
		}
	}
	return n
}

func Regielekis() []string {
	return current.State.Regilekis
}

func RegielekisSecured(t *team.Team) int {
	n := 0
	for _, r := range current.State.Regilekis {
		if r == t.Name {
			n++
		}
	}
	return n
}

func RegirocksSecured(t *team.Team) int {
	n := 0
	for _, b := range current.State.Bottom {
		if b.Name == "regirock" && b.Team == t.Name {
			n++
		}
	}
	return n
}

func RegisteelsSecured(t *team.Team) int {
	n := 0
	for _, b := range current.State.Bottom {
		if b.Name == "registeel" && b.Team == t.Name {
			n++
		}
	}
	return n
}

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

func Scores() (orange, purple, self int) {
	return current.State.Orange.Value, current.State.Purple.Value, current.State.Self.Value
}

func Seconds() int64 {
	return current.State.Seconds
}

func SetBottomObjective(t *team.Team, name string, n int) {
	o := objective{
		Team: t.Name,
		Name: name,
		Time: time.Now().Unix(),
	}

	op := fmt.Sprintf("[%s] %s #%d", lang.Title(t.Name), lang.Title(o.Name), n+1)

	switch {
	// Illegal.
	case len(current.Bottom) < n:
		notify.Warn("[Server] %s illegal operation (no index)", op)

	// Remove.
	case len(current.Bottom) == n+1 && current.Bottom[n].Team == t.Name && current.Bottom[n].Name == o.Name:
		// Remove last objective.
		current.Bottom = current.Bottom[:n]
		notify.Unique(t.NRGBA, "[Server] %s removed", op)

	// Add.
	case len(current.Bottom) == n:
		current.Bottom = append(current.Bottom, o)
		notify.Unique(t.NRGBA, "[Server] %s secured", op)
	case len(current.Bottom) > n+1 && current.Bottom[n].Team != t.Name:
		current.Bottom[n] = o
		notify.Unique(t.NRGBA, "[Server] %s secure replaced", op)

		// Overwrite.
	case len(current.Bottom) == n+1 && current.Bottom[n].Team == t.Name && current.Bottom[n].Name != o.Name:
		// Replace between first and last.
		fallthrough
	case len(current.Bottom) > n+1 && current.Bottom[n].Team == t.Name:
		fallthrough
	case len(current.Bottom) == n+1 && current.Bottom[n].Team != t.Name:
		// Overwrite last objective.
		current.Bottom[n] = o
		notify.Unique(t.NRGBA, "[Server] %s secure replaced", op)
	}
}

func SetConfig(c bool) {
	current.State.Config = c
}

func SetDefeated() {
	current.State.Defeated = append(current.State.Defeated, current.State.Seconds)
}

func SetEnergy(b int) {
	current.State.Energy = b
}

func SetKO(t *team.Team) {
	switch t.Name {
	case team.Purple.Name:
		current.State.Purple.KOs++
	case team.Orange.Name:
		current.State.Orange.KOs++
	}
}

func SetMatchStarted() {
	current.State.InMatch = true
}

func SetMatchStopped() {
	current.State.InMatch = false
}

func SetRayquaza(t *team.Team) {
	current.State.Rayquaza = t.Name
}

func SetRegice(t *team.Team) {
	current.Bottom = append(current.Bottom, objective{
		Team: t.Name,
		Name: "regice",
		Time: time.Now().Unix(),
	})
}

func SetRegieleki(t *team.Team) {
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

// SetRegielekiAt assumes n to be an index starting at 0.
func SetRegielekiAt(t *team.Team, n int) {
	op := fmt.Sprintf("[%s] Regieleki #%d", lang.Title(t.Name), n+1)

	switch {
	case n != 0 && current.State.Regilekis[n-1] == team.None.Name:
		notify.Warn("[Server] %s illegal operation (missing previous)", op)
	case current.State.Regilekis[n] != t.Name:
		notify.Unique(t.NRGBA, "[Server] %s secure replaced", op)
		current.State.Regilekis[n] = t.Name
	case n+1 == len(current.State.Regilekis) || current.State.Regilekis[n+1] == team.None.Name:
		notify.Unique(t.NRGBA, "[Server] %s reset", op)
		current.State.Regilekis[n] = team.None.Name
	default:
		notify.Warn("[Server] %s illegal operation", op)
	}
}

func SetRegirock(t *team.Team) {
	current.Bottom = append(current.Bottom, objective{
		Team: t.Name,
		Name: "regirock",
		Time: time.Now().Unix(),
	})
}

func SetRegisteel(t *team.Team) {
	current.Bottom = append(current.Bottom, objective{
		Team: t.Name,
		Name: "registeel",
		Time: time.Now().Unix(),
	})
}

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

func SetScoreSurrendered(t *team.Team) {
	switch t {
	case team.Purple:
		current.State.Purple.Surrendered = true
	case team.Orange:
		current.State.Orange.Surrendered = true
	}
}

func SetReady() {
	current.State.Ready = true
	state.Add(state.ServerStarted, Clock(), -1)
}

func SetNotReady() {
	current.State.Ready = false
	state.Add(state.ServerStopped, Clock(), -1)
}

func SetTime(minutes, seconds int64) {
	current.State.lastSecondsUpdate = time.Now()

	if minutes+seconds == 0 {
		current.State.InMatch = false
		return
	}

	current.State.InMatch = true

	current.State.Seconds = minutes*60 + seconds
}

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
		Seconds:   0,
		Energy:    0,
		Regilekis: []string{team.None.Name, team.None.Name, team.None.Name},
		Rayquaza:  "",
		Bottom:    []objective{},
		Version:   exe.Version,
		Defeated:  []int64{},
	}
}
