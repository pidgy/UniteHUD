package discord

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/Microsoft/go-winio"
)

type client struct {
	addr string

	errq chan error

	conn io.ReadWriteCloser
}

const (
	id = "1148787786816696350"
)

var (
	socket = struct {
		min, max int
	}{0, 9}

	timeout = time.Second
)

func connect() (client, error) {
	c := client{
		errq: make(chan error, 1),
	}

	for s := socket.min; s <= socket.max; s++ {
		c.addr = fmt.Sprintf(`\\?\pipe\discord-ipc-%d`, s)

		conn, err := winio.DialPipe(c.addr, &timeout)
		if err != nil {
			continue
		}

		c.conn = conn

		return c, nil
	}

	return c, fmt.Errorf("socket error")
}

func (c *client) cleanup() {
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}

	if c.errq != nil {
		close(c.errq)
		c.errq = nil
	}
}

func (c *client) error() error {
	if c.conn == nil {
		return nil
	}

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

func (c *client) receive() (message, error) {
	var msg message

	header := make([]byte, headerLen)
	n, err := c.conn.Read(header)
	if err != nil {
		return msg, err
	}

	if n != headerLen {
		return msg, fmt.Errorf("expected to read %d-byte header, got %d bytes", headerLen, n)
	}

	reader := bytes.NewReader(header)
	binary.Read(reader, binary.LittleEndian, &msg.opcode)

	var payloadLen int32
	binary.Read(reader, binary.LittleEndian, &payloadLen)

	msg.Payload = make([]byte, payloadLen)

	_, err = io.ReadFull(c.conn, msg.Payload)
	if err != nil {
		return msg, err
	}

	return msg, nil
}

func (c *client) send(o opper) {
	p, err := json.Marshal(o)
	if err != nil {
		c.errq <- err
		return
	}

	m := message{
		opcode:  o.opcode(),
		Payload: p,
	}

	err = m.writeMessage(c.conn)
	if err != nil {
		c.errq <- err
		return
	}

	for {
		m, err := c.receive()
		if err != nil {
			c.errq <- err
			return
		}

		switch m.opcode {
		case o.opreceived():
			return
		case opPing:
			m.opcode = opPong
			err := m.writeMessage(c.conn)
			if err != nil {
				c.errq <- err
			}
		case opClose:
			c.errq <- fmt.Errorf("connection terminated")
			return
		default:
			c.errq <- fmt.Errorf("%s received while waiting for %s", o.opcode(), m.opcode)
			return
		}
	}
}
