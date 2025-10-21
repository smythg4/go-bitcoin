package network

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"go-bitcoin/internal/block"
	"go-bitcoin/internal/encoding"
	"io"
)

type GetHeadersMessage struct {
	Version       int32
	BlockLocators [][32]byte
	HashStop      [32]byte
}

func NewGetHeadersMessage(version int32, blockLocators [][32]byte, hashStop *[32]byte) GetHeadersMessage {
	stop := [32]byte{}
	if hashStop != nil {
		stop = *hashStop
	}

	return GetHeadersMessage{
		Version:       version,
		BlockLocators: blockLocators,
		HashStop:      stop,
	}
}

func (g *GetHeadersMessage) Serialize() ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	// write version
	bufint32 := make([]byte, 4)
	binary.LittleEndian.PutUint32(bufint32, uint32(g.Version))
	if _, err := buf.Write(bufint32); err != nil {
		return nil, err
	}

	// write numHashes (varInt)
	hashes, err := encoding.EncodeVarInt(uint64(len(g.BlockLocators)))
	if err != nil {
		return nil, err
	}
	if _, err := buf.Write(hashes); err != nil {
		return nil, err
	}

	for _, block := range g.BlockLocators {
		if _, err := buf.Write(block[:]); err != nil {
			return nil, err
		}
	}

	if _, err := buf.Write(g.HashStop[:]); err != nil {
		return nil, err
	}

	// if len(buf.Bytes()) != 4+len(hashes)+32+32 {
	// 	return nil, fmt.Errorf("result must be %d bytes long, got %d bytes", 4+len(hashes)+32+32, len(buf.Bytes()))
	// }

	return buf.Bytes(), nil
}

func (g GetHeadersMessage) Command() string {
	return "getheaders"
}

type HeadersMessage struct {
	Blocks []block.Block
}

func ParseHeadersMessage(r io.Reader) (HeadersMessage, error) {
	numHeaders, err := encoding.ReadVarInt(r)
	if err != nil {
		return HeadersMessage{}, err
	}
	blocks := make([]block.Block, numHeaders)
	for i := uint64(0); i < numHeaders; i++ {
		b, err := block.ParseBlock(r)
		if err != nil {
			return HeadersMessage{}, err
		}
		blocks[i] = b
		numTx, err := encoding.ReadVarInt(r)
		if err != nil {
			return HeadersMessage{}, err
		}
		if numTx != 0 {
			return HeadersMessage{}, fmt.Errorf("num transaction must be 0, got %d", numTx)
		}
	}
	return HeadersMessage{
		Blocks: blocks,
	}, nil
}

func (h *HeadersMessage) Serialize() ([]byte, error) {
	buf := bytes.NewBuffer(nil)

	numHeaders, err := encoding.EncodeVarInt(uint64(len(h.Blocks)))
	if err != nil {
		return nil, err
	}
	buf.Write(numHeaders)

	for _, block := range h.Blocks {
		blockBytes, err := block.Serialize()
		if err != nil {
			return nil, err
		}
		buf.Write(blockBytes)
		buf.WriteByte(0x00) // num_txs = 0
	}

	return buf.Bytes(), nil
}

func (h HeadersMessage) Command() string {
	return "headers"
}
