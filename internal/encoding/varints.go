package encoding

import (
	"encoding/binary"
	"fmt"
	"io"
)

func ReadVarInt(r io.Reader) (uint64, error) {
	// read a variable integer from a stream

	buf := make([]byte, 8)

	// read first byte
	if _, err := io.ReadFull(r, buf[:1]); err != nil {
		return 0, fmt.Errorf("varint reader error: %w", err)
	}

	switch buf[0] {
	case 0xfd:
		if _, err := io.ReadFull(r, buf[:2]); err != nil {
			return 0, err
		}
		return uint64(binary.LittleEndian.Uint16(buf[:2])), nil
	case 0xfe:
		if _, err := io.ReadFull(r, buf[:4]); err != nil {
			return 0, err
		}
		return uint64(binary.LittleEndian.Uint32(buf[:4])), nil
	case 0xff:
		if _, err := io.ReadFull(r, buf[:8]); err != nil {
			return 0, err
		}
		return binary.LittleEndian.Uint64(buf[:8]), nil
	default:
		return uint64(buf[0]), nil
	}
}

func EncodeVarInt(i uint64) ([]byte, error) {
	// encodes an int as a varint
	if i < 0xfd {
		return []byte{byte(i)}, nil
	} else if i < 0x10000 {
		result := make([]byte, 3)
		result[0] = byte(0xfd) // prefix
		binary.LittleEndian.PutUint16(result[1:], uint16(i))
		return result, nil
	} else if i < 0x100000000 {
		result := make([]byte, 5)
		result[0] = byte(0xfe) // prefix
		binary.LittleEndian.PutUint32(result[1:], uint32(i))
		return result, nil
	} else if i < 0x10000000000000000-1 {
		result := make([]byte, 9)
		result[0] = byte(0xff) // prefix
		binary.LittleEndian.PutUint64(result[1:], uint64(i))
		return result, nil
	}
	return nil, fmt.Errorf("varint encoding error - %d invalid input", i)
}
