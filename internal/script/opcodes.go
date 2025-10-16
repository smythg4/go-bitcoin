package script

import (
	"bytes"
	"crypto/sha1"
	"go-bitcoin/internal/eccmath"
	"go-bitcoin/internal/encoding"
	"go-bitcoin/internal/keys"
	"math/big"
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
	OP_3         byte = 0x53
	OP_4         byte = 0x54
	OP_5         byte = 0x55
	OP_6         byte = 0x56
	OP_7         byte = 0x57
	OP_8         byte = 0x58
	OP_9         byte = 0x59
	OP_10        byte = 0x5a
	OP_11        byte = 0x5b
	OP_12        byte = 0x5c
	OP_13        byte = 0x5d
	OP_14        byte = 0x5e
	OP_15        byte = 0x5f
	OP_16        byte = 0x60

	// flow control
	OP_IF     byte = 0x63
	OP_NOTIF  byte = 0x64
	OP_ELSE   byte = 0x67
	OP_ENDIF  byte = 0x68
	OP_VERIFY byte = 0x69
	OP_RETURN byte = 0x6a

	// stack operations
	OP_DUP          byte = 0x76
	OP_DROP         byte = 0x75
	OP_2DROP        byte = 0x6d
	OP_2DUP         byte = 0x6e
	OP_SWAP         byte = 0x7c
	OP_TOALSTACK    byte = 0x6b
	OP_FROMALTSTACK byte = 0x6c

	// comparison
	OP_EQUAL       byte = 0x87
	OP_EQUALVERIFY byte = 0x88

	// logical
	OP_NOT byte = 0x91

	// arithmetic
	OP_ADD byte = 0x93
	OP_SUB byte = 0x94
	OP_MUL byte = 0x95 // disabled
	OP_DIV byte = 0x96 // disabled

	// crypto
	OP_RIPEMD160      byte = 0xa6
	OP_SHA1           byte = 0xa7
	OP_SHA256         byte = 0xa8
	OP_HASH160        byte = 0xa9
	OP_HASH256        byte = 0xaa
	OP_CHECKSIG       byte = 0xac
	OP_CHECKSIGVERIFY byte = 0xad
	OP_CHECKMULTISIG  byte = 0xae
)

type ScriptEngine struct {
	stack    []ScriptCommand
	altstack []ScriptCommand
	commands []ScriptCommand
	pc       int
	z        []byte
}

func NewScriptEngine(script Script) ScriptEngine {
	return ScriptEngine{
		stack:    []ScriptCommand{},
		commands: script.CommandStack,
		pc:       0,
	}
}

func (se *ScriptEngine) pop() (ScriptCommand, bool) {
	if len(se.stack) < 1 {
		return ScriptCommand{}, false
	}
	top := se.stack[len(se.stack)-1]
	se.stack = se.stack[:len(se.stack)-1]
	return top, true
}

func (se *ScriptEngine) peek() (ScriptCommand, bool) {
	if len(se.stack) < 1 {
		return ScriptCommand{}, false
	}
	top := se.stack[len(se.stack)-1]
	return top, true
}

func (se *ScriptEngine) pushData(data []byte) {
	se.stack = append(se.stack, ScriptCommand{
		Data:   data,
		IsData: true,
	})
}

func (se *ScriptEngine) push(cmd ScriptCommand) {
	se.stack = append(se.stack, cmd)
}

// execute the entire script
func (se *ScriptEngine) Execute(z []byte) bool {
	se.z = z

	for se.pc < len(se.commands) {
		cmd := se.commands[se.pc]
		se.pc++

		if cmd.IsData {
			// data elements just get pushed
			se.push(cmd)
		} else {
			// OpCodes get executed
			if !se.ExecuteCommand(cmd) {
				return false // opcode failed
			}
		}
	}

	// script succeeds if top of stack is non-zero
	return se.verifyFinalStack()
}

func (se *ScriptEngine) verifyFinalStack() bool {
	top, ok := se.pop()
	if !ok {
		return false
	}
	return !isAllZeros(top.Data)
}

func isAllZeros(data []byte) bool {
	// check if data is zero (all bytes == 0)
	for _, b := range data {
		if b != 0 {
			return false
		}
	}
	return true
}

func (se *ScriptEngine) ExecuteCommand(cmd ScriptCommand) bool {
	switch cmd.Opcode {
	case OP_DUP:
		return se.OpDup()
	case OP_2DUP:
		return se.Op2Dup()
	case OP_1, OP_2, OP_3, OP_4, OP_5, OP_6, OP_7, OP_8, OP_9, OP_10, OP_11, OP_12, OP_13, OP_14, OP_15, OP_16:
		num := int64(cmd.Opcode - 0x50)
		se.pushData(encodeNum(num))
		return true
	case OP_ADD:
		return se.OpAdd()
	case OP_SUB:
		return se.OpSub()
	case OP_MUL:
		return se.OpMul()
	case OP_SHA1:
		return se.OpSha1()
	case OP_HASH256:
		return se.OpHash256()
	case OP_HASH160:
		return se.OpHash160()
	case OP_TOALSTACK:
		return se.OpToAltStack()
	case OP_FROMALTSTACK:
		return se.OpFromAltStack()
	case OP_DROP:
		return se.OpDrop()
	case OP_2DROP:
		return se.Op2Drop()
	case OP_IF:
		return se.OpIf()
	case OP_NOTIF:
		return se.OpNotIf()
	case OP_CHECKSIG:
		return se.OpCheckSig()
	case OP_CHECKSIGVERIFY:
		return se.OpCheckSigVerify()
	case OP_NOT:
		return se.OpNot()
	case OP_EQUAL:
		return se.OpEqual()
	case OP_EQUALVERIFY:
		return se.OpEqualVerify()
	case OP_VERIFY:
		return se.OpVerify()
	case OP_SWAP:
		return se.OpSwap()
	default:
		return false
	}
}

func (se *ScriptEngine) OpDup() bool {
	top, ok := se.peek()
	if !ok {
		return false
	}
	se.push(top)
	return true
}

func (se *ScriptEngine) Op2Dup() bool {
	if len(se.stack) < 2 {
		return false
	}

	//get top two items
	second := se.stack[len(se.stack)-2]
	first := se.stack[len(se.stack)-1]

	se.push(second)
	se.push(first)
	return true
}

func (se *ScriptEngine) OpHash256() bool {
	element, ok := se.pop()
	if !ok {
		return false
	}
	hash := encoding.Hash256(element.Data)
	se.pushData(hash)
	return true
}

func (se *ScriptEngine) OpHash160() bool {
	element, ok := se.pop()
	if !ok {
		return false
	}
	hash := encoding.Hash160(element.Data)
	se.pushData(hash)
	return true
}

func (se *ScriptEngine) OpToAltStack() bool {
	item, ok := se.pop()
	if !ok {
		return false
	}
	se.altstack = append(se.altstack, item)
	return true
}

func (se *ScriptEngine) OpFromAltStack() bool {
	if len(se.altstack) == 0 {
		return false
	}
	item := se.altstack[len(se.altstack)-1]
	se.altstack = se.altstack[:len(se.altstack)-1]
	se.push(item)
	return true
}

func (se *ScriptEngine) OpDrop() bool {
	_, ok := se.pop()
	return ok
}

func (se *ScriptEngine) Op2Drop() bool {
	return se.OpDrop() && se.OpDrop()
}

func (se *ScriptEngine) OpIf() bool {
	condition, ok := se.pop()
	if !ok {
		return false
	}

	// check if condition is true
	isTrue := !isAllZeros(condition.Data)

	if !isTrue {
		// skip to OP_ELSE or OP_ENDIF
		se.skipToElseOrEndif()
	}
	// if true, continue executing
	return true
}

func (se *ScriptEngine) skipToElseOrEndif() {
	depth := 1 // track nested IF/ENDIF blocks

	for se.pc < len(se.commands) {
		cmd := se.commands[se.pc]
		se.pc++

		if cmd.Opcode == OP_IF || cmd.Opcode == OP_NOTIF {
			depth++ // nested if
		} else if cmd.Opcode == OP_ENDIF {
			depth--
			if depth == 0 {
				return // found matching ENDIF
			}
		} else if cmd.Opcode == OP_ELSE && depth == 1 {
			return // found match ELSE at same level
		}
	}
}

func (se *ScriptEngine) OpNotIf() bool {
	condition, ok := se.pop()
	if !ok {
		return false
	}

	// check if condition is false
	isFalse := isAllZeros(condition.Data)

	if !isFalse {
		// skip to OP_ELSE or OP_ENDIF
		se.skipToElseOrEndif()
	}
	// if false, continue executing
	return true
}

func (se *ScriptEngine) OpCheckSig() bool {
	// pop public key
	pubkeyCmd, ok := se.pop()
	if !ok {
		return false
	}

	// pop signature (includes sighash type byte at the end)
	sigCmd, ok := se.pop()
	if !ok {
		return false
	}

	// handle empty signature case
	if len(sigCmd.Data) == 0 {
		se.pushData([]byte{}) // no sig -> push false
		return true
	}

	// strip last byte (sighash type, usually 0x01 for SIGHASH_ALL)
	derSig := sigCmd.Data[:len(sigCmd.Data)-1]

	// parse DER signature
	sig, err := eccmath.ParseSignature(bytes.NewReader(derSig))
	if err != nil {
		se.pushData([]byte{}) // invalid sig -> push false
		return true
	}

	// parse SEC public key
	pubkey, err := keys.ParsePublicKey(bytes.NewReader(pubkeyCmd.Data))
	if err != nil {
		se.pushData([]byte{}) // invalid pubkey -> push false
		return true
	}

	// convert sighash to big.Int
	z := new(big.Int).SetBytes(se.z)

	// verify signature
	if pubkey.Verify(z, sig) {
		se.pushData([]byte{0x01}) // verified! -> push true
	} else {
		se.pushData([]byte{}) // verification failed -> push false
	}

	return true
}

func (se *ScriptEngine) OpCheckSigVerify() bool {
	return se.OpCheckSig() && se.OpVerify()
}

func (se *ScriptEngine) OpEqual() bool {
	item1, ok := se.pop()
	if !ok {
		return false
	}
	item2, ok := se.pop()
	if !ok {
		return false
	}
	if bytes.Equal(item1.Data, item2.Data) {
		se.pushData([]byte{0x01})
	} else {
		se.pushData([]byte{0x00})
	}
	return true
}

func (se *ScriptEngine) OpEqualVerify() bool {
	return se.OpEqual() && se.OpVerify()
}

func (se *ScriptEngine) OpVerify() bool {
	item, ok := se.pop()
	if !ok {
		return false
	}
	// fail if all zeros (false), succeed if non-zero (true)
	return !isAllZeros(item.Data)
}

func (se *ScriptEngine) OpSwap() bool {
	item1, ok := se.pop()
	if !ok {
		return false
	}
	item2, ok := se.pop()
	if !ok {
		// script fails, so no need to push the opcode back on the stack
		return false
	}
	se.push(item1)
	se.push(item2)
	return true
}

func (se *ScriptEngine) OpAdd() bool {
	a, ok := se.pop()
	if !ok {
		return false
	}
	b, ok := se.pop()
	if !ok {
		return false
	}

	numA := decodeNum(a.Data)
	numB := decodeNum(b.Data)
	result := encodeNum(numA + numB)

	se.pushData(result)
	return true
}

func (se *ScriptEngine) OpSub() bool {
	a, ok := se.pop()
	if !ok {
		return false
	}
	b, ok := se.pop()
	if !ok {
		return false
	}

	numA := decodeNum(a.Data)
	numB := decodeNum(b.Data)
	result := encodeNum(numA - numB)

	se.pushData(result)
	return true
}

func (se *ScriptEngine) OpMul() bool {
	a, ok := se.pop()
	if !ok {
		return false
	}
	b, ok := se.pop()
	if !ok {
		return false
	}

	numA := decodeNum(a.Data)
	numB := decodeNum(b.Data)
	result := encodeNum(numA * numB)

	se.pushData(result)
	return true
}

func (se *ScriptEngine) OpNot() bool {
	item, ok := se.pop()
	if !ok {
		return false
	}

	num := decodeNum(item.Data)

	if num == 0 {
		se.pushData(encodeNum(1))
	} else {
		se.pushData(encodeNum(0))
	}
	return true
}

func (se *ScriptEngine) OpSha1() bool {
	element, ok := se.pop()
	if !ok {
		return false
	}

	// compute SHA1 hash
	hash := sha1.Sum(element.Data)

	se.pushData(hash[:])
	return true
}
