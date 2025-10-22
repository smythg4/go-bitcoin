package transactions

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"go-bitcoin/internal/encoding"
	"go-bitcoin/internal/keys"
	"go-bitcoin/internal/script"
	"io"
	"slices"
)

type Transaction struct {
	Version   uint32
	Inputs    []TxIn
	Outputs   []TxOut
	Locktime  uint32
	IsTestnet bool
	IsSegwit  bool

	// private cached values
	cachedHashPrevOuts []byte
	cachedHashSequence []byte
	cachedHashOutputs  []byte
}

func NewTransaction(version uint32, inputs []TxIn, outputs []TxOut, locktime uint32, isTestNet, isSegwit bool) Transaction {
	return Transaction{
		Version:   uint32(version),
		Inputs:    inputs,
		Outputs:   outputs,
		Locktime:  locktime,
		IsTestnet: isTestNet,
		IsSegwit:  isSegwit,
	}
}

func (t Transaction) String() string {
	id, _ := t.Id()
	return fmt.Sprintf("tx: %s\n   version:\t%d\n   tx_ins:\t%v\n   tx_outs:\t%v\n   locktime:\t%d\n   isSegwit:\t%v",
		id, t.Version, t.Inputs, t.Outputs, t.Locktime, t.IsSegwit)
}

func (t *Transaction) Id() (string, error) {
	// Human readable hexadecimal of the transaction hash
	hash, err := t.hash()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", hash), nil
}

func (t *Transaction) hash() ([]byte, error) {
	// Binary hash of the legacy serialization
	serialized, err := t.SerializeLegacy()
	if err != nil {
		return nil, err
	}
	hash := encoding.Hash256(serialized)
	slices.Reverse(hash)
	return hash, nil
}

func (t *Transaction) Serialize() ([]byte, error) {
	// returns the byte serialization of the transaction
	if t.IsSegwit {
		return t.SerializeSegwit()
	} else {
		return t.SerializeLegacy()
	}
}

func (t *Transaction) SerializeLegacy() ([]byte, error) {
	// returns the byte serialization of the legacy transaction
	var result bytes.Buffer

	buf := make([]byte, 4)

	// version
	binary.LittleEndian.PutUint32(buf[:4], uint32(t.Version))
	n, err := result.Write(buf[:4])
	if err != nil || n != 4 {
		return nil, fmt.Errorf("tx serialization error (version) - %w", err)
	}

	// inputs len
	inputLen := uint64(len(t.Inputs))
	inputLenBytes, err := encoding.EncodeVarInt(inputLen)
	if err != nil {
		return nil, err
	}
	_, err = result.Write(inputLenBytes)
	if err != nil {
		return nil, fmt.Errorf("tx serialization error (inputs length) - %w", err)
	}
	// inputs slice
	for i, tx := range t.Inputs {
		data, err := tx.Serialize()
		if err != nil {
			return nil, fmt.Errorf("tx serialization error (input read %d) - %w", i, err)
		}
		_, err = result.Write(data)
		if err != nil {
			return nil, fmt.Errorf("tx serialization error (input write %d) - %w", i, err)
		}
	}

	// outputs len
	outputLen := uint64(len(t.Outputs))
	outputLenBytes, err := encoding.EncodeVarInt(outputLen)
	if err != nil {
		return nil, err
	}
	_, err = result.Write(outputLenBytes)
	if err != nil {
		return nil, fmt.Errorf("tx serialization error (outputs length) - %w", err)
	}
	for i, tx := range t.Outputs {
		data, err := tx.Serialize()
		if err != nil {
			return nil, fmt.Errorf("tx serialization error (output read %d) - %w", i, err)
		}
		_, err = result.Write(data)
		if err != nil {
			return nil, fmt.Errorf("tx serialization error (output write %d) - %w", i, err)
		}
	}

	// locktime
	binary.LittleEndian.PutUint32(buf[:4], uint32(t.Locktime))
	n, err = result.Write(buf[:4])
	if err != nil || n != 4 {
		return nil, fmt.Errorf("tx serialization error (locktime) - %w", err)
	}

	return result.Bytes(), nil
}

func (t *Transaction) SerializeSegwit() ([]byte, error) {
	// returns the byte serialization of the Segwit transaction
	var result bytes.Buffer

	// marker and flag bytes
	n, err := result.Write([]byte{0x00, 0x01})
	if err != nil || n != 2 {
		return nil, fmt.Errorf("tx serialization error (marker/flag) - %w", err)
	}

	buf := make([]byte, 4)
	// version
	binary.LittleEndian.PutUint32(buf[:4], uint32(t.Version))
	n, err = result.Write(buf[:4])
	if err != nil || n != 4 {
		return nil, fmt.Errorf("tx serialization error (version) - %w", err)
	}

	// inputs len
	inputLen := uint64(len(t.Inputs))
	inputLenBytes, err := encoding.EncodeVarInt(inputLen)
	if err != nil {
		return nil, err
	}
	_, err = result.Write(inputLenBytes)
	if err != nil {
		return nil, fmt.Errorf("tx serialization error (inputs length) - %w", err)
	}
	// inputs slice
	for i, tx := range t.Inputs {
		data, err := tx.Serialize()
		if err != nil {
			return nil, fmt.Errorf("tx serialization error (input read %d) - %w", i, err)
		}
		_, err = result.Write(data)
		if err != nil {
			return nil, fmt.Errorf("tx serialization error (input write %d) - %w", i, err)
		}
	}

	// outputs len
	outputLen := uint64(len(t.Outputs))
	outputLenBytes, err := encoding.EncodeVarInt(outputLen)
	if err != nil {
		return nil, err
	}
	_, err = result.Write(outputLenBytes)
	if err != nil {
		return nil, fmt.Errorf("tx serialization error (outputs length) - %w", err)
	}
	for i, tx := range t.Outputs {
		data, err := tx.Serialize()
		if err != nil {
			return nil, fmt.Errorf("tx serialization error (output read %d) - %w", i, err)
		}
		_, err = result.Write(data)
		if err != nil {
			return nil, fmt.Errorf("tx serialization error (output write %d) - %w", i, err)
		}
	}
	// witness
	for _, txin := range t.Inputs {
		numItemBytes, err := encoding.EncodeVarInt(uint64(len(txin.Witness)))
		if err != nil {
			return nil, err
		}
		// write the varint number of items
		if _, err := result.Write(numItemBytes); err != nil {
			return nil, err
		}
		for _, item := range txin.Witness {
			itemLenBytes, err := encoding.EncodeVarInt(uint64(len(item)))
			if err != nil {
				return nil, err
			}
			// write the varint length of this item
			if _, err := result.Write(itemLenBytes); err != nil {
				return nil, err
			}
			// write this item
			if _, err := result.Write(item); err != nil {
				return nil, err
			}
		}
	}
	// locktime
	binary.LittleEndian.PutUint32(buf[:4], uint32(t.Locktime))
	n, err = result.Write(buf[:4])
	if err != nil || n != 4 {
		return nil, fmt.Errorf("tx serialization error (locktime) - %w", err)
	}

	return result.Bytes(), nil
}

func ParseTransaction(r io.Reader) (Transaction, error) {
	// version
	buf := make([]byte, 5)
	n, err := r.Read(buf)
	if err != nil || n != 5 {
		return Transaction{}, fmt.Errorf("tx parse error (version and marker) - %w", err)
	}
	version := binary.LittleEndian.Uint32(buf[:4])

	if buf[4] == 0x00 {
		// marker byte for SegWit
		return ParseSegwitTransaction(r, version)
	} else {
		return ParseLegacyTransaction(r, version, buf[4])
	}
}

func ParseLegacyTransaction(r io.Reader, version uint32, firstByte byte) (Transaction, error) {
	// hacky way to "rewind" the reader for proper varint reading
	r = io.MultiReader(bytes.NewReader([]byte{firstByte}), r)

	// parse TxIn
	len, err := encoding.ReadVarInt(r)
	if err != nil {
		return Transaction{}, err
	}
	var i uint64
	txins := make([]TxIn, 0, len)
	for i = 0; i < len; i++ {
		tx, err := ParseTxIn(r)
		if err != nil {
			return Transaction{}, err
		}
		txins = append(txins, tx)
	}

	// parse TxOut
	len, err = encoding.ReadVarInt(r)
	if err != nil {
		return Transaction{}, err
	}
	txouts := make([]TxOut, 0, len)
	for i = 0; i < len; i++ {
		tx, err := ParseTxOut(r)
		if err != nil {
			return Transaction{}, err
		}
		txouts = append(txouts, tx)
	}

	// locktime
	buf := make([]byte, 4)
	n, err := r.Read(buf)
	if err != nil || n != 4 {
		return Transaction{}, fmt.Errorf("tx parse error (locktime) - %w", err)
	}
	locktime := binary.LittleEndian.Uint32(buf)

	return Transaction{
		Version:  version,
		Inputs:   txins,
		Outputs:  txouts,
		Locktime: locktime,
		IsSegwit: false,
	}, nil
}

func ParseSegwitTransaction(r io.Reader, version uint32) (Transaction, error) {
	// check the flag byte (marker byte already checked)
	flag := make([]byte, 1)
	if _, err := r.Read(flag); err != nil {
		return Transaction{}, err
	}

	// parse TxIn
	len, err := encoding.ReadVarInt(r)
	if err != nil {
		return Transaction{}, err
	}
	var i uint64
	txins := make([]TxIn, 0, len)
	for i = 0; i < len; i++ {
		tx, err := ParseTxIn(r)
		if err != nil {
			return Transaction{}, err
		}
		txins = append(txins, tx)
	}

	// parse TxOut
	len, err = encoding.ReadVarInt(r)
	if err != nil {
		return Transaction{}, err
	}
	txouts := make([]TxOut, 0, len)
	for i = 0; i < len; i++ {
		tx, err := ParseTxOut(r)
		if err != nil {
			return Transaction{}, err
		}
		txouts = append(txouts, tx)
	}

	// parse witnesses
	for i := range txins {
		numItems, err := encoding.ReadVarInt(r)
		if err != nil {
			return Transaction{}, err
		}
		items := make([][]byte, numItems)
		for j := uint64(0); j < numItems; j++ {
			itemLen, err := encoding.ReadVarInt(r)
			if err != nil {
				return Transaction{}, err
			}
			itemBytes := make([]byte, itemLen)
			if _, err := r.Read(itemBytes); err != nil {
				return Transaction{}, err
			}
			items = append(items, itemBytes)
		}
		txins[i].Witness = items
	}

	// parse locktime
	buf := make([]byte, 4)
	n, err := r.Read(buf)
	if err != nil || n != 4 {
		return Transaction{}, fmt.Errorf("tx parse error (locktime) - %w", err)
	}
	locktime := binary.LittleEndian.Uint32(buf)

	return Transaction{
		Version:  version,
		Inputs:   txins,
		Outputs:  txouts,
		Locktime: locktime,
		IsSegwit: true,
	}, nil
}

func (t *Transaction) SigHash(inputIndex int) ([]byte, error) {
	// get the scriptpubkey from the input
	prevScriptPubKey, err := t.Inputs[inputIndex].ScriptPubKey(t.IsTestnet)
	if err != nil {
		return nil, err
	}

	// check if this is P2SH - use redeemScript if so
	if script.IsP2sh(prevScriptPubKey.CommandStack) {
		scriptSig := t.Inputs[inputIndex].ScriptSig
		if len(scriptSig.CommandStack) == 0 {
			return nil, errors.New("empty ScriptSig for P2SH input")
		}
		// last element of ScriptSig is serialized redeemScript
		lastCmd := scriptSig.CommandStack[len(scriptSig.CommandStack)-1]
		if !lastCmd.IsData {
			return nil, errors.New("invalid P2SH ScriptSig: last element not data")
		}
		// In transaction.go, around line 205:
		redeemScriptData := lastCmd.Data
		// Prepend the length as a varint
		length, err := encoding.EncodeVarInt(uint64(len(redeemScriptData)))
		if err != nil {
			return nil, fmt.Errorf("failed to encode redeemScript length: %w", err)
		}
		scriptWithLength := append(length, redeemScriptData...)
		redeemScript, err := script.ParseScript(bytes.NewReader(scriptWithLength))
		if err != nil {
			return nil, fmt.Errorf("failed to parse redeemScript: %w", err)
		}
		prevScriptPubKey = redeemScript
	}
	// create a modified transaction for signing
	// 1. for the input at inputIndex, replace ScriptSig with prevScriptPubKey
	// 2. for all other inputs, set ScriptSig to empty

	// make a copy of inputs with modifications
	modifiedInputs := make([]TxIn, len(t.Inputs))
	for i, input := range t.Inputs {
		modifiedInputs[i] = TxIn{
			PrevTx:   input.PrevTx,
			PrevIdx:  input.PrevIdx,
			Sequence: input.Sequence,
		}

		if i == inputIndex {
			// this is the input we're signing - use prevScriptPubKey
			modifiedInputs[i].ScriptSig = prevScriptPubKey
		} else {
			// all other inputs get empty script
			modifiedInputs[i].ScriptSig = script.NewScript([]script.ScriptCommand{})
		}
	}

	// create modified transaction
	modifiedTx := Transaction{
		Version:   t.Version,
		Inputs:    modifiedInputs,
		Outputs:   t.Outputs,
		Locktime:  t.Locktime,
		IsTestnet: t.IsTestnet,
	}

	// serialize the modified transaction
	serialized, err := modifiedTx.Serialize()
	if err != nil {
		return nil, err
	}

	// append sighash type (SIGHASH_ALL  = 0x01000000)
	sighashType := make([]byte, 4)
	binary.LittleEndian.PutUint32(sighashType, encoding.SIGHASH_ALL)
	serialized = append(serialized, sighashType...)

	// double SHA256
	hash := encoding.Hash256(serialized)

	return hash, nil
}

func (t *Transaction) Fee(testNet bool) (uint64, error) {
	// returns the fee of this transaction in satoshi

	// sum all input values
	inputSum := uint64(0)
	for _, tx := range t.Inputs {
		val, err := tx.Value(testNet)
		if err != nil {
			return 0, err
		}
		inputSum += val
	}

	// sum all output values
	outputSum := uint64(0)
	for _, output := range t.Outputs {
		outputSum += output.Amount
	}

	if outputSum > inputSum {
		return 0, fmt.Errorf("invalid transaction: outputs (%d) > inputs (%d)", outputSum, inputSum)
	}
	// fee is the difference
	return inputSum - outputSum, nil
}

func (t *Transaction) VerifyInput(inputIndex int) (bool, error) {
	if inputIndex >= len(t.Inputs) {
		return false, errors.New("inputIndex out of range")
	}
	input := t.Inputs[inputIndex]

	// get the ScriptPubKey from the output being spent
	scriptPubKey, err := input.ScriptPubKey(t.IsTestnet)
	if err != nil {
		return false, fmt.Errorf("error fetching ScriptPubKey for index %d: %w", inputIndex, err)
	}

	var z []byte
	var witness [][]byte

	if scriptPubKey.IsP2wpkhScriptPubKey() {
		// native p2wpkh
		// scriptsig empty, witness contains signature data
		z, err = t.SigHashBIP143(inputIndex, nil, nil)
		if err != nil {
			return false, fmt.Errorf("error generating BIP143 sighash: %w", err)
		}
		witness = input.Witness
	} else if scriptPubKey.IsP2shScriptPubKey() {
		// Could be nested SegWit (P2SH-wrapped P2WPKH)
		// Extract redeemScript from ScriptSig (last element)
		if len(input.ScriptSig.CommandStack) == 0 {
			return false, errors.New("empty ScriptSig for P2SH input")
		}
		command := input.ScriptSig.CommandStack[len(input.ScriptSig.CommandStack)-1]
		rawRedeemLen := len(command.Data)
		redeemLenBytes, err := encoding.EncodeVarInt(uint64(rawRedeemLen))
		if err != nil {
			return false, err
		}
		rawRedeem := append(redeemLenBytes, command.Data...)
		redeemScript, err := script.ParseScript(bytes.NewReader(rawRedeem))
		if err != nil {
			return false, err
		}
		if redeemScript.IsP2wpkhScriptPubKey() {
			z, err = t.SigHashBIP143(inputIndex, &redeemScript, nil)
			if err != nil {
				return false, fmt.Errorf("error generating sighash for index %d: %w", inputIndex, err)
			}
			witness = input.Witness
		} else {
			z, err = t.SigHashBIP143(inputIndex, &redeemScript, nil)
			if err != nil {
				return false, fmt.Errorf("error generating sighash for index %d: %w", inputIndex, err)
			}
		}
	} else {
		// legacy P2PKH or other...
		z, err = t.SigHash(inputIndex)
		if err != nil {
			return false, fmt.Errorf("error generating sighash for index %d: %w", inputIndex, err)
		}
	}

	// combine ScriptSig + ScriptPubKey
	combinedScript := input.ScriptSig.Combine(scriptPubKey)

	// evaluate
	return combinedScript.Evaluate(z, witness), nil
}

func (t *Transaction) Verify() (bool, error) {
	// verify this entire transaction
	_, err := t.Fee(t.IsTestnet)
	if err != nil {
		// this will catch if fee < 0
		return false, fmt.Errorf("error fetching fee: %w", err)
	}

	for i, txin := range t.Inputs {
		valid, err := t.VerifyInput(i)
		if err != nil {
			return false, fmt.Errorf("error verifying input %s: %w", txin, err)
		}
		if !valid {
			return false, nil
		}
	}
	return true, nil
}

func (t *Transaction) SignInput(inputIndex int, privKey keys.PrivateKey, compressed bool) error {
	// sign the transaction
	z, err := t.SigHash(inputIndex)
	if err != nil {
		return err
	}

	sig, err := privKey.SignHash(z)
	if err != nil {
		return err
	}

	derSig := sig.Serialize()
	sighashType := make([]byte, 4)
	binary.LittleEndian.PutUint32(sighashType, encoding.SIGHASH_ALL)
	derSigWithHashType := append(derSig, sighashType...)

	publicKey := privKey.PublicKey()
	secPubKey := publicKey.Serialize(compressed)

	scriptSig := script.NewScript([]script.ScriptCommand{
		{IsData: true, Data: derSigWithHashType},
		{IsData: true, Data: secPubKey},
	})

	t.Inputs[inputIndex].ScriptSig = scriptSig
	return nil
}

func (t *Transaction) SignInputs(privKey keys.PrivateKey, compressed bool) error {
	for i, txin := range t.Inputs {
		err := t.SignInput(i, privKey, compressed)
		if err != nil {
			return fmt.Errorf("error signing input %s: %w", txin, err)
		}
	}
	return nil
}

func (t *Transaction) isCoinbase() bool {
	// coinbase transactions must have exactly one input
	if len(t.Inputs) != 1 {
		return false
	}
	// the one input must have a previous transaction of 32 bytes of 00
	if !slices.Equal(t.Inputs[0].PrevTx, bytes.Repeat([]byte{0x00}, 32)) {
		return false
	}
	// the one input must have a previous index of ffffffff
	if t.Inputs[0].PrevIdx != 0xffffffff {
		return false
	}
	return true
}

func (t *Transaction) coinbaseHeight() int64 {
	if !t.isCoinbase() {
		return -1
	}
	element := t.Inputs[0].ScriptSig.CommandStack[0]
	return script.DecodeNum(element.Data)
}

func (t *Transaction) SigHashBIP143(inputIndex int, redeemScript *script.Script, witnessScript *script.Script) ([]byte, error) {
	txin := t.Inputs[inputIndex]

	// per BIP143 spec
	s := bytes.NewBuffer(nil)

	buf4 := make([]byte, 4)
	buf8 := make([]byte, 8)

	var scriptCode []byte
	var err error
	// version bytes
	binary.LittleEndian.PutUint32(buf4, t.Version)
	if _, err := s.Write(buf4); err != nil {
		return nil, err
	}

	if _, err := s.Write(t.hashPrevOuts()); err != nil {
		return nil, err
	}
	if _, err := s.Write(t.hashSequence()); err != nil {
		return nil, err
	}
	prevout := make([]byte, len(txin.PrevTx))
	copy(prevout, txin.PrevTx)
	slices.Reverse(prevout)
	if _, err := s.Write(prevout); err != nil {
		return nil, err
	}
	binary.LittleEndian.PutUint32(buf4, txin.PrevIdx)
	if _, err := s.Write(buf4); err != nil {
		return nil, err
	}
	if witnessScript != nil {
		scriptCode, err = witnessScript.Serialize()
		if err != nil {
			return nil, err
		}
	} else if redeemScript != nil {
		scr := script.P2pkhScript(redeemScript.CommandStack[1].Data)
		scriptCode, err = scr.Serialize()
		if err != nil {
			return nil, err
		}
	} else {
		pk, err := txin.ScriptPubKey(t.IsTestnet)
		if err != nil {
			return nil, err
		}
		scr := script.P2pkhScript(pk.CommandStack[1].Data)
		scriptCode, err = scr.Serialize()
		if err != nil {
			return nil, err
		}
	}
	if _, err := s.Write(scriptCode); err != nil {
		return nil, err
	}

	val, err := txin.Value(t.IsTestnet)
	if err != nil {
		return nil, err
	}
	binary.LittleEndian.PutUint64(buf8, val)
	if _, err := s.Write(buf8); err != nil {
		return nil, err
	}

	binary.LittleEndian.PutUint32(buf4, txin.Sequence)
	if _, err := s.Write(buf4); err != nil {
		return nil, err
	}

	outHash, err := t.hashOutputs()
	if err != nil {
		return nil, err
	}
	if _, err := s.Write(outHash); err != nil {
		return nil, err
	}

	binary.LittleEndian.PutUint32(buf4, t.Locktime)
	if _, err := s.Write(buf4); err != nil {
		return nil, err
	}

	binary.LittleEndian.PutUint32(buf4, encoding.SIGHASH_ALL)
	if _, err := s.Write(buf4); err != nil {
		return nil, err
	}

	return encoding.Hash256(s.Bytes()), nil
}

func (t *Transaction) hashPrevOuts() []byte {
	if t.cachedHashOutputs == nil {
		allPrevOuts := []byte{}
		allSequence := []byte{}
		buf4 := make([]byte, 4)
		for _, txin := range t.Inputs {
			prevout := make([]byte, len(txin.PrevTx))
			copy(prevout, txin.PrevTx)
			slices.Reverse(prevout)
			allPrevOuts = append(allPrevOuts, prevout...)
			binary.LittleEndian.PutUint32(buf4, txin.PrevIdx)
			allPrevOuts = append(allPrevOuts, buf4...)
			binary.LittleEndian.PutUint32(buf4, txin.Sequence)
			allSequence = append(allSequence, buf4...)
		}
		t.cachedHashPrevOuts = encoding.Hash256(allPrevOuts)
		t.cachedHashSequence = encoding.Hash256(allSequence)
	}
	return t.cachedHashPrevOuts
}

func (t *Transaction) hashSequence() []byte {
	if t.cachedHashSequence == nil {
		_ = t.hashPrevOuts() // this should populate it
	}
	return t.cachedHashSequence
}

func (t *Transaction) hashOutputs() ([]byte, error) {
	if t.cachedHashOutputs == nil {
		allOutputs := []byte{}
		for _, txout := range t.Outputs {
			ser, err := txout.Serialize()
			if err != nil {
				return nil, err
			}
			allOutputs = append(allOutputs, ser...)
		}
		t.cachedHashOutputs = encoding.Hash256(allOutputs)
	}
	return t.cachedHashOutputs, nil
}
