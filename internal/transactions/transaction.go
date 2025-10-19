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
}

func NewTransaction(version uint32, inputs []TxIn, outputs []TxOut, locktime uint32, isTestNet bool) Transaction {
	return Transaction{
		Version:   uint32(version),
		Inputs:    inputs,
		Outputs:   outputs,
		Locktime:  locktime,
		IsTestnet: isTestNet,
	}
}

func (t Transaction) String() string {
	id, _ := t.id()
	return fmt.Sprintf("tx: %s\n   version:\t%d\n   tx_ins:\t%v\n   tx_outs:\t%v\n   locktime:\t%d",
		id, t.Version, t.Inputs, t.Outputs, t.Locktime)
}

func (t *Transaction) id() (string, error) {
	// Human readable hexadecimal of the transaction hash
	hash, err := t.hash()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", hash), nil
}

func (t *Transaction) hash() ([]byte, error) {
	// Binary hash of the legacy serialization
	serialized, err := t.Serialize()
	if err != nil {
		return nil, err
	}
	hash := encoding.Hash256(serialized)
	slices.Reverse(hash)
	return hash, nil
}

func (t *Transaction) Serialize() ([]byte, error) {
	// returns the byte serialization of the transaction
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

func ParseTransaction(r io.Reader) (Transaction, error) {
	// version
	buf := make([]byte, 4)
	n, err := r.Read(buf)
	if err != nil || n != 4 {
		return Transaction{}, fmt.Errorf("tx parse error (version) - %w", err)
	}
	version := binary.LittleEndian.Uint32(buf)

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
	n, err = r.Read(buf) // reuse 4-byte buffer
	if err != nil || n != 4 {
		return Transaction{}, fmt.Errorf("tx parse error (locktime) - %w", err)
	}
	locktime := binary.LittleEndian.Uint32(buf)

	return Transaction{
		Version:  version,
		Inputs:   txins,
		Outputs:  txouts,
		Locktime: locktime,
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
	binary.LittleEndian.PutUint32(sighashType, 1)
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

func (t *Transaction) verifyInput(inputIndex int) (bool, error) {
	if inputIndex > len(t.Inputs) {
		return false, errors.New("inputIndex out of range")
	}
	input := t.Inputs[inputIndex]

	// get the ScriptPubKey from the output being spent
	scriptPubKey, err := input.ScriptPubKey(t.IsTestnet)
	if err != nil {
		return false, fmt.Errorf("error fetching ScriptPubKey for index %d: %w", inputIndex, err)
	}

	// calculate signature hash for this input
	z, err := t.SigHash(inputIndex)
	if err != nil {
		return false, fmt.Errorf("error generating sighash for index %d: %w", inputIndex, err)
	}

	// combine ScriptSig + ScriptPubKey
	combinedScript := input.ScriptSig.Combine(scriptPubKey)

	// evaluate
	return combinedScript.Evaluate(z), nil
}

func (t *Transaction) Verify() (bool, error) {
	// verify this entire transaction
	_, err := t.Fee(t.IsTestnet)
	if err != nil {
		// this will catch if fee < 0
		return false, fmt.Errorf("error fetching fee: %w", err)
	}

	for i, txin := range t.Inputs {
		valid, err := t.verifyInput(i)
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
	derSigWithHashType := append(derSig, 0x01) // append SIGHASH_ALL

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
