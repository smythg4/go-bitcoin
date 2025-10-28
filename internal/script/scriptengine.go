package script

import (
	"bytes"
	"crypto/sha1"
	"crypto/sha256"
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
	OP_PUSHDATA4 byte = 0x4e
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

	// locktime
	OP_CHECKLOCKTIMEVERIFY byte = 0xb1
	OP_CHECKSEQUENCEVERIFY byte = 0xb2
)

type ScriptEngine struct {
	stack    []ScriptCommand
	altstack []ScriptCommand
	commands []ScriptCommand
	pc       int
	z        []byte
	witness  [][]byte
	// BIP 65/112 context
	locktime uint32
	sequence uint32
}

func NewScriptEngine(script Script) ScriptEngine {
	return ScriptEngine{
		stack:    []ScriptCommand{},
		commands: script.CommandStack,
		pc:       0,
	}
}

// WithLocktime sets the transaction locktime for OP_CHECKLOCKTIMEVERIFY (BIP 65)
func (se *ScriptEngine) WithLocktime(locktime uint32) *ScriptEngine {
	se.locktime = locktime
	return se
}

// WithSequence sets the input sequence for OP_CHECKSEQUENCEVERIFY (BIP 112)
func (se *ScriptEngine) WithSequence(sequence uint32) *ScriptEngine {
	se.sequence = sequence
	return se
}

// WithWitness sets the witness data for SegWit transactions
func (se *ScriptEngine) WithWitness(witness [][]byte) *ScriptEngine {
	se.witness = witness
	return se
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

func IsP2sh(triplet []ScriptCommand) bool {
	return triplet[0].Opcode == OP_HASH160 && len(triplet[1].Data) == 20 && triplet[2].Opcode == OP_EQUAL
}

func IsP2wsh(pair []ScriptCommand) bool {
	return len(pair) == 2 &&
		pair[0].Opcode == OP_O &&
		pair[1].IsData &&
		len(pair[1].Data) == 32
}

func (se *ScriptEngine) P2sh(redeemScript, hash ScriptCommand) bool {
	// we know first command is OP_HASH160
	ok := se.OpHash160()
	if !ok {
		return false
	}

	se.push(hash) // hash from the cmd stack

	// the next command will be OP_EQUAL -- I used OP_EQUALVERIFY to knock out the final check of 0/1
	ok = se.OpEqualVerify()
	if !ok {
		return false
	}
	// Prepend varint length before parsing
	redeemScriptData := redeemScript.Data
	length, err := encoding.EncodeVarInt(uint64(len(redeemScriptData)))
	if err != nil {
		return false
	}
	scriptWithLength := append(length, redeemScriptData...)
	parsedRs, err := ParseScript(bytes.NewBuffer(scriptWithLength)) // do I need to prepend the length?
	if err != nil {
		return false
	}

	se.commands = append(se.commands, parsedRs.CommandStack...)

	return true
}

func (se *ScriptEngine) P2wsh(hash256 ScriptCommand) bool {
	if len(se.witness) == 0 {
		return false
	}

	// Last witness item is the witnessScript
	witnessScript := se.witness[len(se.witness)-1]

	// Validate: SHA256(witnessScript) == hash256
	actualHash := sha256.Sum256(witnessScript)
	if !bytes.Equal(actualHash[:], hash256.Data) {
		return false
	}

	// Push all witness items except last onto stack
	for i := 0; i < len(se.witness)-1; i++ {
		se.pushData(se.witness[i])
	}

	// Parse witnessScript and inject commands
	length, err := encoding.EncodeVarInt(uint64(len(witnessScript)))
	if err != nil {
		return false
	}
	scriptBytes := append(length, witnessScript...)
	parsedWitnessScript, err := ParseScript(bytes.NewReader(scriptBytes))
	if err != nil {
		return false
	}

	// Inject witnessScript commands into execution
	se.commands = append(se.commands, parsedWitnessScript.CommandStack...)

	return true
}

func (se *ScriptEngine) P2wpkh(hash160 ScriptCommand) bool {
	if len(se.witness) != 2 {
		return false
	}

	// Push witness items onto stack
	se.pushData(se.witness[0]) // signature
	se.pushData(se.witness[1]) // pubkey

	// Create and inject P2PKH script commands
	p2pkhScript := P2pkhScript(hash160.Data)
	se.commands = append(se.commands, p2pkhScript.CommandStack...)

	return true
}

// execute the entire script
func (se *ScriptEngine) Execute(z []byte) bool {
	se.z = z

	for se.pc < len(se.commands) {
		cmd := se.commands[se.pc]
		se.pc++

		if se.pc+2 <= len(se.commands) && IsP2sh(se.commands[se.pc-1:se.pc+2]) {
			// look for BIP0016 sequence of commands
			redeemScript, ok := se.peek() // copy the redeemScript for later use
			if !ok {
				return false
			}
			hash := se.commands[se.pc]
			if !se.P2sh(redeemScript, hash) {
				return false
			}
			se.pc += 2 // already advanced it 1 earlier
			continue
		}
		if cmd.IsData {
			// data elements just get pushed
			se.push(cmd)
		} else {
			// OpCodes get executed
			if !se.ExecuteCommand(cmd) {
				return false // opcode failed
			}
		}

		// after execution, check stack for witness programs
		if len(se.stack) == 2 &&
			len(se.stack[0].Data) == 0 && // OP_O pushes empty bytes
			len(se.stack[1].Data) == 20 { // P2WPKH
			hash160, _ := se.pop()
			se.pop() // remove OP_O
			if !se.P2wpkh(hash160) {
				return false
			}
			continue
		}
		if len(se.stack) == 2 &&
			len(se.stack[0].Data) == 0 &&
			len(se.stack[1].Data) == 32 { // P2WSH
			hash256, _ := se.pop() // add error handling
			se.pop()               // remove OP_O
			if !se.P2wsh(hash256) {
				return false
			}
			continue
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
	case OP_O: // 0x00 - already defined in your constants
		se.pushData([]byte{}) // OP_0 pushes empty byte array
		return true
	case OP_DUP:
		return se.OpDup()
	case OP_2DUP:
		return se.Op2Dup()
	case OP_1, OP_2, OP_3, OP_4, OP_5, OP_6, OP_7, OP_8, OP_9, OP_10, OP_11, OP_12, OP_13, OP_14, OP_15, OP_16:
		num := int64(cmd.Opcode - 0x50)
		se.pushData(EncodeNum(num))
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
	case OP_CHECKMULTISIG:
		return se.OpCheckMultiSig()
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
	case OP_CHECKLOCKTIMEVERIFY:
		return se.OpCheckLocktimeVerify()
	case OP_CHECKSEQUENCEVERIFY:
		return se.OpCheckSequenceVerify()
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

func checkSigHelper(pubkeyCmd, sigCmd ScriptCommand, z *big.Int) bool {
	if len(sigCmd.Data) == 0 {
		return false
	}
	derSig := sigCmd.Data[:len(sigCmd.Data)-1] // strip sighash type byte

	sig, err := eccmath.ParseSignature(bytes.NewReader(derSig))
	if err != nil {
		return false
	}

	pubkey, err := keys.ParsePublicKey(bytes.NewReader(pubkeyCmd.Data))
	if err != nil {
		return false
	}

	return pubkey.Verify(z, sig)
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

	// convert sighash to big.Int
	z := new(big.Int).SetBytes(se.z)

	if checkSigHelper(pubkeyCmd, sigCmd, z) {
		se.pushData([]byte{0x01}) // verified! -> push true
	} else {
		se.pushData([]byte{}) // verification failed -> push false
	}

	return true
}

func (se *ScriptEngine) OpCheckSigVerify() bool {
	return se.OpCheckSig() && se.OpVerify()
}

func (se *ScriptEngine) OpCheckMultiSig() bool {
	top, ok := se.pop()
	if !ok {
		return false
	}

	// get n public keys off the stack
	n := int(DecodeNum(top.Data))
	if len(se.stack) < n+1 {
		return false
	}
	secPubkeys := make([]ScriptCommand, 0, n)
	for i := 0; i < n; i++ {
		top, ok = se.pop()
		if !ok {
			return false // should never happen
		}
		secPubkeys = append(secPubkeys, top)
	}

	// get m signatures off the stack
	top, ok = se.pop()
	if !ok {
		return false
	}
	m := int(DecodeNum(top.Data))
	if len(se.stack) < m+1 {
		return false
	}
	derSignatures := make([]ScriptCommand, 0, m)
	for i := 0; i < m; i++ {
		top, ok = se.pop()
		if !ok {
			return false
		}
		derSignatures = append(derSignatures, top)
	}
	// off by one filler element
	top, ok = se.pop()
	if !ok {
		return false
	}

	// TODO: do the verifications
	z := new(big.Int).SetBytes(se.z)

	sigIndex := 0
	pubkeyIndex := 0

	// try to match all m signatures
	for sigIndex < m && pubkeyIndex < n {
		if checkSigHelper(secPubkeys[pubkeyIndex], derSignatures[sigIndex], z) {
			// signature matched this pubkey - move to next signature
			sigIndex++
		}
		// always move to the next pubkey
		pubkeyIndex++
	}

	// success if we match all m signatures
	if sigIndex == m {
		se.pushData([]byte{0x01})
	} else {
		se.pushData([]byte{0x00})
	}

	return true
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

	numA := DecodeNum(a.Data)
	numB := DecodeNum(b.Data)
	result := EncodeNum(numA + numB)

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

	numA := DecodeNum(a.Data)
	numB := DecodeNum(b.Data)
	result := EncodeNum(numA - numB)

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

	numA := DecodeNum(a.Data)
	numB := DecodeNum(b.Data)
	result := EncodeNum(numA * numB)

	se.pushData(result)
	return true
}

func (se *ScriptEngine) OpNot() bool {
	item, ok := se.pop()
	if !ok {
		return false
	}

	num := DecodeNum(item.Data)

	if num == 0 {
		se.pushData(EncodeNum(1))
	} else {
		se.pushData(EncodeNum(0))
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

// OpCheckLocktimeVerify implements OP_CHECKLOCKTIMEVERIFY (BIP 65)
// Marks transaction as invalid if the top stack item is greater than the transaction's locktime field
// or if the sequence number is 0xffffffff (finalized)
func (se *ScriptEngine) OpCheckLocktimeVerify() bool {
	// BIP 65: OP_CHECKLOCKTIMEVERIFY

	if len(se.stack) < 1 {
		return false
	}

	// Peek at top stack element (don't pop - CLTV doesn't consume the value)
	element, ok := se.peek()
	if !ok {
		return false
	}

	// Decode the locktime threshold from stack
	stackLocktime := DecodeNum(element.Data)

	// 1. Check if the stack value is negative (BIP 65 rule)
	if stackLocktime < 0 {
		return false
	}

	// 2. Check if input sequence is final (0xffffffff means locktime is disabled)
	// BIP 65: nSequence must be < 0xffffffff
	if se.sequence == 0xffffffff {
		return false
	}

	// 3. Check that stack locktime and tx locktime are the same type
	// Types: block height (< 500000000) or Unix timestamp (>= 500000000)
	const LOCKTIME_THRESHOLD = 500000000

	stackIsTimestamp := stackLocktime >= LOCKTIME_THRESHOLD
	txIsTimestamp := se.locktime >= LOCKTIME_THRESHOLD

	// They must both be block heights or both be timestamps
	if stackIsTimestamp != txIsTimestamp {
		return false
	}

	// 4. Check that transaction locktime >= stack locktime
	// This means the transaction is locked until at least the stack value
	if int64(se.locktime) < stackLocktime {
		return false
	}

	// Success - CLTV is a "verify" operation, so it doesn't modify the stack
	return true
}

// OpCheckSequenceVerify implements OP_CHECKSEQUENCEVERIFY (BIP 112)
// Relative lock-time using consensus-enforced sequence numbers (BIP 68)
func (se *ScriptEngine) OpCheckSequenceVerify() bool {
	// BIP 112: OP_CHECKSEQUENCEVERIFY

	if len(se.stack) < 1 {
		return false
	}

	// Peek at top stack element (don't pop - CSV doesn't consume the value)
	element, ok := se.peek()
	if !ok {
		return false
	}

	// Decode the sequence value from stack
	stackSequence := DecodeNum(element.Data)

	// 1. Check if the stack value is negative (BIP 112 rule)
	if stackSequence < 0 {
		return false
	}

	// BIP 112: If bit 31 of stack value is set, CSV succeeds immediately
	// This allows scripts to opt-out of relative lock-time
	const SEQUENCE_LOCKTIME_DISABLE_FLAG = uint32(1 << 31)
	if uint32(stackSequence)&SEQUENCE_LOCKTIME_DISABLE_FLAG != 0 {
		return true
	}

	// 2. Check if tx input sequence has disable flag set (bit 31)
	// If bit 31 of nSequence is set, BIP 68 is disabled, so CSV fails
	if se.sequence&SEQUENCE_LOCKTIME_DISABLE_FLAG != 0 {
		return false
	}

	// 3. Check that stack and sequence have same lock-time type (bit 22)
	// Bit 22: 0 = block-based, 1 = time-based (512-second granularity)
	const SEQUENCE_LOCKTIME_TYPE_FLAG = uint32(1 << 22)

	stackType := uint32(stackSequence) & SEQUENCE_LOCKTIME_TYPE_FLAG
	sequenceType := se.sequence & SEQUENCE_LOCKTIME_TYPE_FLAG

	if stackType != sequenceType {
		return false
	}

	// 4. Compare the masked values (lower 16 bits)
	// Mask extracts bits 0-15 (the actual lock-time value)
	const SEQUENCE_LOCKTIME_MASK = 0x0000ffff

	stackValue := uint32(stackSequence) & SEQUENCE_LOCKTIME_MASK
	sequenceValue := se.sequence & SEQUENCE_LOCKTIME_MASK

	// Sequence must be >= stack value (input must have aged enough)
	if sequenceValue < stackValue {
		return false
	}

	// Success - CSV is a "verify" operation, so it doesn't modify the stack
	return true
}
