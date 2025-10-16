package script

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"go-bitcoin/internal/encoding"
	"io"
)

// Script Op Codes
const (
	OP_O         byte = 0x00
	OP_PUSHDATA1 byte = 0x4c
	OP_PUSHDATA2 byte = 0x4d
	OP_PUSHDATA4 byte = 0x4f
	OP_1NEGATE   byte = 0x4f
	OP_1         byte = 0x51
	OP_2         byte = 0x52
	OP_16        byte = 0x60

	// flow control
	OP_IF     byte = 0x63
	OP_NOTIF  byte = 0x64
	OP_ELSE   byte = 0x67
	OP_ENDIF  byte = 0x68
	OP_VERIFY byte = 0x69
	OP_RETURN byte = 0x6a

	// stack operations
	OP_DUP   byte = 0x76
	OP_DROP  byte = 0x75
	OP_2DROP byte = 0x6d
	OP_SWAP  byte = 0x7c

	// comparison
	OP_EQUAL       byte = 0x87
	OP_EQUALVERIFY byte = 0x88

	// arithmetic
	OP_ADD byte = 0x93

	// crypto
	OP_RIPEMD160     byte = 0xa6
	OP_SHA1          byte = 0xa7
	OP_SHA256        byte = 0xa8
	OP_HASH160       byte = 0xa9
	OP_HASH256       byte = 0xaa
	OP_CHECKSIG      byte = 0xac
	OP_CHECKMULTISIG byte = 0xae
)

type ScriptCommand struct {
	Opcode byte
	Data   []byte
	IsData bool // true if data is set, false if it's an Opcode
}

type Script struct {
	commandStack []ScriptCommand
}

func NewScript(cmds []ScriptCommand) Script {
	return Script{
		commandStack: cmds,
	}
}

func ParseScript(r io.Reader) (Script, error) {
	s := NewScript([]ScriptCommand{})
	length, err := encoding.ReadVarInt(r)
	if err != nil {
		return Script{}, fmt.Errorf("script parsing error (read) - %w", err)
	}
	count := uint64(0)
	for count < length {
		buf := make([]byte, 1)
		n, err := r.Read(buf)
		if err != nil || n != 1 {
			return Script{}, fmt.Errorf("script parsing error (length) - %w", err)
		}
		currentByte := buf[0]

		count++
		if currentByte >= 1 && currentByte <= 75 {
			// next bytes are an element to add to the stack
			elemLen := int(currentByte)
			buf := make([]byte, elemLen)
			n, err := r.Read(buf)
			if err != nil {
				return Script{}, fmt.Errorf("script parsing error (append) - %w", err)
			}
			if n != elemLen {
				return Script{}, fmt.Errorf("script parsing error: element length (%d) != bytes read (%d)", elemLen, n)
			}

			// add as data
			s.commandStack = append(s.commandStack, ScriptCommand{
				Data:   buf,
				IsData: true,
			})
			count += uint64(n)
		} else {
			switch currentByte {
			case OP_PUSHDATA1:
				// next byte tells us how many bytes to push onto stack
				buf := make([]byte, 1)
				n, err := r.Read(buf)
				if err != nil || n != 1 {
					return Script{}, fmt.Errorf("script parsing error: OP_PUSHDATA1 - %w", err)
				}
				dataLen := int(buf[0])
				buf = make([]byte, dataLen)
				n, err = r.Read(buf)
				if err != nil || n != dataLen {
					return Script{}, fmt.Errorf("script parsing error: OP_PUSHDATA1 - %w", err)
				}

				// add as data
				s.commandStack = append(s.commandStack, ScriptCommand{
					Data:   buf,
					IsData: true,
				})
				count += uint64(n + 1)
			case OP_PUSHDATA2:
				// next two bytes tells us how many bytes to push onto stack
				buf := make([]byte, 2)
				n, err := r.Read(buf)
				if err != nil || n != 2 {
					return Script{}, fmt.Errorf("script parsing error: OP_PUSHDATA2 - %w", err)
				}
				dataLen := int(binary.LittleEndian.Uint16(buf))
				buf = make([]byte, dataLen)
				n, err = r.Read(buf)
				if err != nil || n != dataLen {
					return Script{}, fmt.Errorf("script parsing error: OP_PUSHDATA2 - %w", err)
				}

				// add as data
				s.commandStack = append(s.commandStack, ScriptCommand{
					Data:   buf,
					IsData: true,
				})
				count += uint64(n + 2)
			case OP_PUSHDATA4:
				// next four bytes tells us how many bytes to push onto stack
				buf := make([]byte, 4)
				n, err := r.Read(buf)
				if err != nil || n != 4 {
					return Script{}, fmt.Errorf("script parsing error: OP_PUSHDATA4 - %w", err)
				}
				dataLen := int(binary.LittleEndian.Uint32(buf))
				buf = make([]byte, dataLen)
				n, err = r.Read(buf)
				if err != nil || n != dataLen {
					return Script{}, fmt.Errorf("script parsing error: OP_PUSHDATA4 - %w", err)
				}

				// add as data
				s.commandStack = append(s.commandStack, ScriptCommand{
					Data:   buf,
					IsData: true,
				})
				count += uint64(n + 4)
			default:
				// just another op_code to push onto the stack
				s.commandStack = append(s.commandStack, ScriptCommand{
					Opcode: currentByte,
					IsData: false,
				})
			}
		}
	}
	if count != length {
		return Script{}, fmt.Errorf("script parsing error: script length (%d) != bytes parsed (%d)", length, count)
	}
	return s, nil
}

func (s *Script) Serialize() ([]byte, error) {
	var result bytes.Buffer

	for _, cmd := range s.commandStack {
		if cmd.IsData {
			dataLen := len(cmd.Data)

			if dataLen <= 75 {
				// length fits in one byte
				if err := result.WriteByte(byte(dataLen)); err != nil {
					return nil, err
				}
				if _, err := result.Write(cmd.Data); err != nil {
					return nil, err
				}
			} else if dataLen <= 0xff {
				// use OP_PUSHDATA1
				if err := result.WriteByte(OP_PUSHDATA1); err != nil {
					return nil, err
				}
				if err := result.WriteByte(byte(dataLen)); err != nil {
					return nil, err
				}
				if _, err := result.Write(cmd.Data); err != nil {
					return nil, err
				}
			} else if dataLen <= 0xffff {
				// use OP_PUSHDATA2
				if err := result.WriteByte(OP_PUSHDATA2); err != nil {
					return nil, err
				}
				lenBytes := make([]byte, 2)
				binary.LittleEndian.PutUint16(lenBytes, uint16(dataLen))
				if _, err := result.Write(lenBytes); err != nil {
					return nil, err
				}
				if _, err := result.Write(cmd.Data); err != nil {
					return nil, err
				}
			} else {
				// use OP_PUSHDATA4
				if err := result.WriteByte(OP_PUSHDATA4); err != nil {
					return nil, err
				}
				lenBytes := make([]byte, 4)
				binary.LittleEndian.PutUint32(lenBytes, uint32(dataLen))
				if _, err := result.Write(lenBytes); err != nil {
					return nil, err
				}
				if _, err := result.Write(cmd.Data); err != nil {
					return nil, err
				}
			}
		} else {
			// just write the opcode
			if err := result.WriteByte(cmd.Opcode); err != nil {
				return nil, err
			}
		}
	}

	// prepend with varint length
	serialized := result.Bytes()
	length, err := encoding.EncodeVarInt(uint64(len(serialized)))
	if err != nil {
		return nil, fmt.Errorf("script serialization error: varint length - %w", err)
	}
	return append(length, serialized...), nil
}
