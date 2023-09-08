package discord

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

type (
	message struct {
		// The Discord-RPC opcode.
		Opcode opcode

		// Payload is the JSON string encoded as UTF-8.
		Payload payload
	}

	opcode int32

	payload []byte

	result struct {
		msg message
		err error
	}
)

const (
	opHandshake = opcode(iota)
	opFrame
	opClose
	opPing
	opPong
)

func (o opcode) String() string {
	switch o {
	case opHandshake:
		return "Handshake"
	case opFrame:
		return "Frame"
	case opClose:
		return "Close"
	case opPing:
		return "Ping"
	case opPong:
		return "Pong"
	default:
		return fmt.Sprintf("opode: %d", o)
	}
}

// headerLen is the number of bytes in the Discord IPC message header.
const headerLen = 8

// encode serializes the message in wire format.
func (m message) encode() []byte {
	buf := bytes.NewBuffer(make([]byte, 0, len(m.Payload)+headerLen))
	binary.Write(buf, binary.LittleEndian, m.Opcode)
	binary.Write(buf, binary.LittleEndian, int32(len(m.Payload)))
	buf.Write(m.Payload)
	return buf.Bytes()
}

// writeMessage writes a message to the socket.
func (m message) writeMessage(w io.Writer) error {
	buf := m.encode()
	switch n, err := w.Write(buf); {
	case err != nil:
		return err
	case n != len(buf):
		return fmt.Errorf("wanted to write %d bytes, wrote %d bytes", len(buf), n)
	}
	return nil
}
