package transactions

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"go-bitcoin/internal/script"
	"io"
	"slices"
)

type TxIn struct {
	PrevTx    []byte
	PrevIdx   uint32
	ScriptSig script.Script
	Sequence  uint32
}

func (t TxIn) String() string {
	return fmt.Sprintf("%x:%d", t.PrevTx, t.PrevIdx)
}

func ParseTxIn(r io.Reader) (TxIn, error) {
	prevTx := make([]byte, 32)

	// prev tx hash (256 bit hash)
	n, err := r.Read(prevTx)
	if err != nil || n != 32 {
		return TxIn{}, fmt.Errorf("txin parse error - %w", err)
	}
	slices.Reverse(prevTx)

	// prev index
	buf := make([]byte, 4)
	n, err = r.Read(buf)
	if err != nil || n != 4 {
		return TxIn{}, fmt.Errorf("txin parse error - %w", err)
	}
	prevIdx := binary.LittleEndian.Uint32(buf)

	// ScriptSig
	script, err := script.ParseScript(r)
	if err != nil {
		return TxIn{}, err
	}

	// Sequence
	n, err = r.Read(buf)
	if err != nil || n != 4 {
		return TxIn{}, fmt.Errorf("txin parse error - %w", err)
	}
	seq := binary.LittleEndian.Uint32(buf)

	return TxIn{
		PrevTx:    prevTx,
		PrevIdx:   prevIdx,
		ScriptSig: script,
		Sequence:  seq,
	}, nil
}

func (t *TxIn) Serialize() ([]byte, error) {
	// returns the byte serialization of the transaction input
	var result bytes.Buffer

	// previous transaction hash
	revPrevTx := make([]byte, len(t.PrevTx))
	copy(revPrevTx, t.PrevTx)
	slices.Reverse(revPrevTx)
	if _, err := result.Write(revPrevTx); err != nil {
		return nil, err
	}

	// previous transaction index
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, t.PrevIdx)
	if _, err := result.Write(buf); err != nil {
		return nil, err
	}

	// ScriptSig
	scriptBytes, err := t.ScriptSig.Serialize()
	if err != nil {
		return nil, err
	}
	if _, err := result.Write(scriptBytes); err != nil {
		return nil, err
	}

	// sequence (uses old 4 byte buffer)
	binary.LittleEndian.PutUint32(buf, t.Sequence)
	if _, err := result.Write(buf); err != nil {
		return nil, err
	}

	return result.Bytes(), nil
}

type TxOut struct {
	Amount       uint64
	ScriptPubKey script.Script
}

func (t TxOut) String() string {
	pubKey, _ := t.ScriptPubKey.Serialize()
	return fmt.Sprintf("%x:%x", t.Amount, pubKey)
}

func ParseTxOut(r io.Reader) (TxOut, error) {
	// amount
	buf := make([]byte, 8)
	n, err := r.Read(buf)
	if err != nil || n != 8 {
		return TxOut{}, fmt.Errorf("txout parse error - %w", err)
	}
	amount := binary.LittleEndian.Uint64(buf)

	// scriptpubkey
	script, err := script.ParseScript(r)
	if err != nil {
		return TxOut{}, fmt.Errorf("txout parse error - %w", err)
	}

	return TxOut{
		Amount:       amount,
		ScriptPubKey: script,
	}, nil
}

func (t *TxOut) Serialize() ([]byte, error) {
	// returns the byte serialization of the transaction output
	var result bytes.Buffer

	// Amount
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, t.Amount)
	if _, err := result.Write(buf); err != nil {
		return nil, err
	}

	// ScriptPubKey
	scriptBytes, err := t.ScriptPubKey.Serialize()
	if err != nil {
		return nil, err
	}
	if _, err := result.Write(scriptBytes); err != nil {
		return nil, err
	}

	return result.Bytes(), nil
}
