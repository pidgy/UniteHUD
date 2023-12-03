package discord

import "github.com/google/uuid"

const (
	activityInstanceInMatch = true
	activityInstanceIdle    = false
)

const (
	// https://discord.com/developers/docs/topics/gateway-events#activity-object-activity-types
	activityTypePlaying activityType = iota
	activityTypeStreaming
	activityTypeListening
	activityTypeWatching
	activityTypeCustom
	activityTypeCompeting
)

const (
	commandSetActivity = "SET_ACTIVITY"
)

type (
	// https://discord.com/developers/docs/topics/gateway#activity-object
	activity struct {
		State      string           `json:"state"`
		Details    string           `json:"details,omitempty"`
		Timestamps timestamps       `json:"timestamps,omitempty"`
		Assets     assets           `json:"assets,omitempty"`
		Type       activityType     `json:"type,omitempty"`
		Buttons    []button         `json:"buttons,omitempty"`
		Instance   activityInstance `json:"instance"`
		Party      party            `json:"party,omitempty"`
	}

	activityType     int
	activityInstance bool

	// https://github.com/discord/discord-rpc/blob/master/documentation/hard-mode.md
	args struct {
		Pid      int      `json:"pid"`
		Activity activity `json:"activity"`
	}

	assets struct {
		LargeImage string `json:"large_image"`
		LargeText  string `json:"large_text"`
		SmallImage string `json:"small_image"`
		SmallText  string `json:"small_text"`
	}

	button struct {
		Label string `json:"label"`
		URL   string `json:"url"`
	}

	// Emoji struct holds data related to Emoji's
	// https://discord.com/developers/docs/topics/gateway-events#activity-object-activity-emoji
	emoji struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		// Roles         []string `json:"roles"`
		// RequireColons bool     `json:"require_colons"`
		// Managed       bool     `json:"managed"`
		Animated bool `json:"animated"`
		// Available     bool     `json:"available"`
	}

	// frame contains the generic outer fields for Discord JSON requests.
	frame struct {
		Nonce string      `json:"nonce"`
		Args  interface{} `json:"args"`
		Cmd   string      `json:"cmd"`
	}

	// handshake represents the Discord handshake JSON.
	// It doesn't appear to be officially documented.
	handshake struct {
		// Undocumented API version, 1 is the only known value.
		Version int `json:"v"`
		// Application ID from the Discord developer portal.
		ClientId string `json:"client_id"`
		// A token or sequence number for matching the response.
		Nonce string `json:"nonce,omitempty"`
	}

	timestamps struct {
		Start int64 `json:"start,omitempty"`
		End   int64 `json:"end,omitempty"`
	}

	party struct {
		ID   string `json:"id,omitempty"`
		Size size
	}

	size struct {
		CurrentSize int `json:"currentsize,omitempty"`
		MaxSize     int `json:"maxsize,omitempty"`
	}
)

var (
	partyID = uuid.NewString()

	unitehudEmoji = emoji{
		ID:       "1149174424361238619",
		Name:     "unitehud",
		Animated: false,
	}
)

func (f frame) opcode() opcode     { return opFrame }
func (f frame) opreceived() opcode { return opFrame }

func (h handshake) opcode() opcode     { return opHandshake }
func (h handshake) opreceived() opcode { return opFrame }
