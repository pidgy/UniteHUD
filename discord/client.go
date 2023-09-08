package discord

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/Microsoft/go-winio"

	"github.com/pidgy/unitehud/notify"
)

type client struct {
	connected bool

	socket int
	addr   string

	inq  chan message
	outq chan payload
	errq chan error

	conn io.ReadWriteCloser
}

const (
	id       = "1148787786816696350"
	min, max = 0, 9
)

var (
	timeout = time.Second
)

func connect() (client, error) {
	prefix := ""

	c := client{
		connected: false,
		socket:    min,

		outq: make(chan payload, 1),
		inq:  make(chan message),
		errq: make(chan error, 1),
	}

	for ; !c.connected && c.socket <= max; c.socket++ {
		c.addr = fmt.Sprintf(`\\?\pipe\%sdiscord-ipc-%d`, prefix, c.socket)

		conn, err := winio.DialPipe(c.addr, &timeout)
		if err != nil {
			continue
		}

		c.connected = true
		c.conn = conn
		c.loop()

		return c, nil
	}

	return c, fmt.Errorf("max socket connections attempted")
}

func (c *client) close() {
	if c.outq != nil {
		close(c.outq)
	}
}

func (c *client) cleanup() {
	close(c.inq)
	c.conn.Close()
	c.conn = nil
}

func (c *client) error() error {
	select {
	case err := <-c.errq:
		return err
	default:
		return nil
	}
}

func (c *client) handshake(clientId string) {
	c.send(handshake{
		Version:  1,
		ClientId: clientId,
		Nonce:    "0",
	})
}

func (c *client) loop() {
	go func() {
		defer c.cleanup()

		next := opHandshake

		for c.connected {
			resultq := make(chan result, 1)
			go func() {
				msg, err := c.receive()
				resultq <- result{msg, err}
			}()

			select {
			case payload, ok := <-c.outq:
				if !ok {
					continue
				}

				m := message{
					Opcode:  next,
					Payload: payload,
				}

				err := m.writeMessage(c.conn)
				if err != nil {
					c.errq <- err
					continue
				}

				next = opFrame

				// Block on an answer from Discord.
				r := <-resultq
				if r.err != nil {
					c.errq <- r.err
					continue
				}

				c.inq <- r.msg
			case r := <-resultq:
				// Unsolicited message from Discord.
				switch {
				case r.err != nil:
					c.errq <- r.err
				case r.msg.Opcode == opPing:
					// Respond immediately to ping.
					r.msg.Opcode = opPong
					if err := r.msg.writeMessage(c.conn); err != nil {
						c.errq <- err
						continue
					}
				case r.msg.Opcode == opClose:
					c.errq <- fmt.Errorf("connection terminated")
				default:
					c.errq <- fmt.Errorf("unexpected opcode: %d, payload: %v", r.msg.Opcode, r.msg.Payload)
				}
			}
		}

		notify.Debug("Discord exited loop...")
	}()
}

func (c *client) receive() (message, error) {
	var msg message

	header := make([]byte, headerLen)
	switch n, err := c.conn.Read(header); {
	case err != nil:
		return msg, err
	case n != headerLen:
		return msg, fmt.Errorf("wanted %d-byte header, read %d bytes", headerLen, n)
	}

	reader := bytes.NewReader(header)
	binary.Read(reader, binary.LittleEndian, &msg.Opcode)
	var payloadLen int32
	binary.Read(reader, binary.LittleEndian, &payloadLen)

	msg.Payload = make([]byte, payloadLen)
	_, err := io.ReadFull(c.conn, msg.Payload)
	return msg, err
}

func (c *client) send(i interface{}) {
	p, err := json.Marshal(i)
	if err != nil {
		c.errq <- err
		return
	}

	c.outq <- p

	msg, ok := <-c.inq
	if !ok {
		c.errq <- fmt.Errorf("socket closed while waiting for response")
		return
	}

	notify.Debug("Discord received opcode %s", msg.Opcode.String())

	f := frame{}
	err = json.Unmarshal(msg.Payload, &f)
	if err == nil {
		notify.Debug("Discord received frame %s, %T", f.Cmd, f.Args)
	}
}
