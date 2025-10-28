package transactions

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"go-bitcoin/internal/encoding"
	"go-bitcoin/internal/script"
	"io"
	"slices"
)

type TxIn struct {
	PrevTx    []byte
	PrevIdx   uint32
	ScriptSig script.Script
	Sequence  uint32
	Witness   [][]byte
}

func NewTxIn(prevTx []byte, prevIdx, sequence uint32) TxIn {
	return TxIn{
		PrevTx:   prevTx,
		PrevIdx:  prevIdx,
		Sequence: sequence,
	}
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
	// Check if this is a coinbase input (prevTx is all zeros and prevIdx is 0xffffffff)
	isCoinbase := prevIdx == 0xffffffff
	if isCoinbase {
		for _, b := range prevTx {
			if b != 0 {
				isCoinbase = false
				break
			}
		}
	}

	var scriptSig script.Script
	if isCoinbase {
		// Coinbase scriptSig contains arbitrary data, not valid script
		// Read it as raw bytes without parsing
		scriptLen, err := encoding.ReadVarInt(r)
		if err != nil {
			return TxIn{}, err
		}
		scriptBytes := make([]byte, scriptLen)
		if _, err := io.ReadFull(r, scriptBytes); err != nil {
			return TxIn{}, err
		}
		// Store as a single data command (arbitrary bytes)
		// Special case: empty scriptSig should have no commands for proper roundtrip
		if scriptLen == 0 {
			scriptSig = script.NewScript([]script.ScriptCommand{})
		} else {
			scriptSig = script.NewScript([]script.ScriptCommand{
				{Data: scriptBytes, IsData: true},
			})
		}
	} else {
		// Regular input - parse as Bitcoin script
		var err error
		scriptSig, err = script.ParseScript(r)
		if err != nil {
			return TxIn{}, err
		}
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
		ScriptSig: scriptSig,
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

func (t *TxIn) fetchTx(testNet bool) (*Transaction, error) {
	fetcher := NewTxFetcher()
	// PrevTx is stored in display order (big-endian)
	// Can use directly for API call
	hex := fmt.Sprintf("%x", t.PrevTx)
	return fetcher.Fetch(hex, testNet, false)
}

func (t *TxIn) Value(testNet bool) (uint64, error) {
	// get the output value by looking up the tx hash.
	// returns amount in Satoshi
	tx, err := t.fetchTx(testNet)
	if err != nil {
		return 0, err
	}
	return tx.Outputs[t.PrevIdx].Amount, nil
}

func (t *TxIn) ScriptPubKey(testNet bool) (script.Script, error) {
	// get the ScriptPubKey by looking up the tx hash. Returns a Script object.
	tx, err := t.fetchTx(testNet)
	if err != nil {
		return script.Script{}, err
	}
	return tx.Outputs[t.PrevIdx].ScriptPubKey, nil
}

type TxOut struct {
	Amount         uint64
	ScriptPubKey   script.Script
	rawScriptBytes []byte // Raw script bytes even if unparseable
}

// RawScriptBytes returns the raw script bytes for filter generation
// Falls back to serializing ScriptPubKey if raw bytes weren't stored
func (t *TxOut) RawScriptBytes() ([]byte, error) {
	if len(t.rawScriptBytes) > 0 {
		return t.rawScriptBytes, nil
	}
	// Fallback for older code paths
	return t.ScriptPubKey.RawBytes()
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

	// scriptpubkey - read raw bytes first
	scriptBytes, err := script.ReadScriptBytes(r)
	if err != nil {
		return TxOut{}, fmt.Errorf("txout parse error - %w", err)
	}

	// Try to parse the script, but use empty script if parsing fails
	// (some blocks have intentionally malformed scripts)
	scriptObj := script.Script{}
	if len(scriptBytes) > 0 {
		// Create a reader with the varint length prefix + script bytes
		varIntLen, _ := encoding.EncodeVarInt(uint64(len(scriptBytes)))
		scriptReader := bytes.NewReader(append(varIntLen, scriptBytes...))
		parsedScript, err := script.ParseScript(scriptReader)
		if err == nil {
			scriptObj = parsedScript
		}
		// If parsing fails, we keep the empty script but the raw bytes are still available
	}

	return TxOut{
		Amount:       amount,
		ScriptPubKey: scriptObj,
		rawScriptBytes: scriptBytes, // Store raw bytes for filter generation
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
