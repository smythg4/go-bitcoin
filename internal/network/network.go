package network

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"go-bitcoin/internal/encoding"
	"io"
)

type Message interface {
	Serialize() ([]byte, error)
	Command() string
}

type MagicNum = uint32

const MAINNET_MAGIC MagicNum = 0xf9beb4d9
const TESTNET_MAGIC MagicNum = 0x0b110907

type NetworkEnvelope struct {
	Magic           MagicNum
	Command         string
	PayloadLen      uint32
	PayloadChecksum uint32
	Payload         []byte
}

func NewNetworkEnvelope(command string, payload []byte, testNet bool) (NetworkEnvelope, error) {
	if len(command) > 12 {
		// length in bytes
		return NetworkEnvelope{}, fmt.Errorf("command too long: %d bytes (max 12)", len(command))
	}

	hash := encoding.Hash256(payload)
	checksum := binary.LittleEndian.Uint32(hash[:4])

	magic := MAINNET_MAGIC
	if testNet {
		magic = TESTNET_MAGIC
	}

	return NetworkEnvelope{
		Magic:           magic,
		Command:         command, // stored unpadded
		PayloadLen:      uint32(len(payload)),
		PayloadChecksum: checksum,
		Payload:         payload,
	}, nil
}

func (n NetworkEnvelope) String() string {
	return fmt.Sprintf("%s: %x", n.Command, n.Payload)
}

func (n *NetworkEnvelope) commandBytes() [12]byte {
	var cmd [12]byte
	copy(cmd[:], n.Command) // Copies command, rest stays zero (null-padded)
	return cmd
}

func ParseNetworkEnvelope(r io.Reader) (NetworkEnvelope, error) {
	// if len(data) < 24 {
	// 	return NetworkEnvelope{}, fmt.Errorf("envelope header too short: %d bytes (need 24)", len(data))
	// }

	magicBytes := make([]byte, 4)
	_, err := io.ReadFull(r, magicBytes)
	if err != nil {
		return NetworkEnvelope{}, err
	}
	magic := binary.LittleEndian.Uint32(magicBytes)

	// parse command and strip null padding
	commandBytes := make([]byte, 12)
	_, err = io.ReadFull(r, commandBytes)
	if err != nil {
		return NetworkEnvelope{}, err
	}
	command := string(bytes.TrimRight(commandBytes, "\x00"))

	payloadLenBytes := make([]byte, 4)
	_, err = io.ReadFull(r, payloadLenBytes)
	if err != nil {
		return NetworkEnvelope{}, err
	}
	payloadLen := binary.LittleEndian.Uint32(payloadLenBytes)

	checksumBytes := make([]byte, 4)
	_, err = io.ReadFull(r, checksumBytes)
	if err != nil {
		return NetworkEnvelope{}, err
	}
	checksum := binary.LittleEndian.Uint32(checksumBytes)

	payload := make([]byte, payloadLen)
	_, err = io.ReadFull(r, payload)
	if err != nil {
		return NetworkEnvelope{}, err
	}

	// validate checksum
	hash := encoding.Hash256(payload)
	expectedChecksum := binary.LittleEndian.Uint32(hash[:4])
	if checksum != expectedChecksum {
		return NetworkEnvelope{}, fmt.Errorf("checksum mismatch: got %08x, expected %08x", checksum, expectedChecksum)
	}

	return NetworkEnvelope{
		Magic:           magic,
		Command:         command,
		PayloadLen:      payloadLen,
		PayloadChecksum: checksum,
		Payload:         payload,
	}, nil
}

func (n *NetworkEnvelope) Serialize() ([]byte, error) {
	buf := make([]byte, 4+12+4+4+n.PayloadLen)

	binary.BigEndian.PutUint32(buf[0:4], n.Magic)

	commandBytes := n.commandBytes()
	copy(buf[4:16], commandBytes[:])

	binary.LittleEndian.PutUint32(buf[16:20], n.PayloadLen)

	binary.LittleEndian.PutUint32(buf[20:24], n.PayloadChecksum)

	if len(buf[24:]) < int(n.PayloadLen) {
		return nil, fmt.Errorf("not enough space left in buffer: %d bytes (need %d bytes)", len(buf[24:]), n.PayloadLen)
	}
	copy(buf[24:], n.Payload)
	return buf, nil
}
