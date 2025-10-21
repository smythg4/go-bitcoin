package script

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"go-bitcoin/internal/encoding"
	"io"
)

type ScriptCommand struct {
	Opcode byte
	Data   []byte
	IsData bool // true if data is set, false if it's an Opcode
}

type Script struct {
	CommandStack []ScriptCommand
}

func NewScript(cmds []ScriptCommand) Script {
	return Script{
		CommandStack: cmds,
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
			s.CommandStack = append(s.CommandStack, ScriptCommand{
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
				s.CommandStack = append(s.CommandStack, ScriptCommand{
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
				s.CommandStack = append(s.CommandStack, ScriptCommand{
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
				s.CommandStack = append(s.CommandStack, ScriptCommand{
					Data:   buf,
					IsData: true,
				})
				count += uint64(n + 4)
			default:
				// just another op_code to push onto the stack
				s.CommandStack = append(s.CommandStack, ScriptCommand{
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

	for _, cmd := range s.CommandStack {
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

func (s Script) Combine(scriptPubKey Script) Script {
	// used to stack ScriptSig with ScriptPubKey
	// check that s is ScriptSig?

	combined := make([]ScriptCommand, 0, len(s.CommandStack)+len(scriptPubKey.CommandStack))
	combined = append(combined, s.CommandStack...)
	combined = append(combined, scriptPubKey.CommandStack...)
	return Script{
		CommandStack: combined,
	}
}

func (s *Script) Evaluate(sighash []byte) bool {
	engine := NewScriptEngine(*s)
	return engine.Execute(sighash)
}

func EncodeNum(n int64) []byte {
	// converts a Go int64 to Bitcoin Script's little-endian signed integer format
	if n == 0 {
		return []byte{}
	}
	absN := n
	negative := n < 0
	if negative {
		absN = -n
	}

	result := []byte{}
	for absN > 0 {
		result = append(result, byte(absN&0xff))
		absN >>= 8
	}

	// if the high bit is set, add an extra byte for the sign
	if result[len(result)-1]&0x80 != 0 {
		if negative {
			result = append(result, 0x80)
		} else {
			result = append(result, 0x00)
		}
	} else if negative {
		// set the sign bit on the last byte
		result[len(result)-1] |= 0x80
	}

	return result
}

func DecodeNum(data []byte) int64 {
	// converts Bitcoin Script's little-endian signed integer format to Go's int64
	if len(data) == 0 {
		return 0
	}

	// check sign bit (high bit of last byte)
	negative := data[len(data)-1]&0x80 != 0

	// convert from little-endian bytes to int64
	var result int64
	for i := len(data) - 1; i >= 0; i-- {
		result <<= 8
		if i == len(data)-1 {
			// strip sign bit from last byte
			result |= int64(data[i] & 0x7f)
		} else {
			result |= int64(data[i])
		}
	}

	if negative {
		return -result
	}

	return result
}

func P2pkhScript(h160 []byte) Script {
	// take a hash160 and returns the p2pkh script ScriptPubKey
	c1 := ScriptCommand{
		Opcode: OP_DUP,
		IsData: false,
	}
	c2 := ScriptCommand{
		Opcode: OP_HASH160,
		IsData: false,
	}
	c3 := ScriptCommand{
		IsData: true,
		Data:   h160,
	}
	c4 := ScriptCommand{
		Opcode: OP_EQUALVERIFY,
		IsData: false,
	}
	c5 := ScriptCommand{
		Opcode: OP_CHECKSIG,
		IsData: false,
	}
	cmds := []ScriptCommand{c1, c2, c3, c4, c5}
	return NewScript(cmds)
}

func P2pkhAddress(h160 []byte, testNet bool) string {
	prefix := 0x00
	if testNet {
		prefix = 0x6f // testnet prefix
	}
	return encoding.EncodeBase58Checksum(append([]byte{byte(prefix)}, h160...))
}

func P2shAddress(h160 []byte, testNet bool) string {
	prefix := 0x05
	if testNet {
		prefix = 0xc4 // testnet prefix
	}
	return encoding.EncodeBase58Checksum(append([]byte{byte(prefix)}, h160...))
}

func (s *Script) Address(testnet bool) (string, error) {
	if len(s.CommandStack) < 3 {
		return "", errors.New("not enough commands")
	}
	if IsP2sh(s.CommandStack[0:3]) {
		h160 := s.CommandStack[1].Data
		return P2shAddress(h160, testnet), nil
	} else {
		// assume p2pkh otherwise
		h160 := s.CommandStack[2].Data
		return P2pkhAddress(h160, testnet), nil
	}
}
