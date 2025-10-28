package network

import (
	"bytes"
	"encoding/binary"
	"go-bitcoin/internal/encoding"
	"io"
)

type FilterType byte

const (
	BASIC FilterType = 0x00 // only filter defined by BIP158
)

type GetCFilterMessage struct {
	FType       FilterType
	StartHeight uint32   // 4 bytes, height of first block in requested range
	StopHash    [32]byte // hash of last block in the requested range
}

func (gcfm GetCFilterMessage) Command() string {
	return "getcfilters"
}

func (gcfm *GetCFilterMessage) Serialize() ([]byte, error) {
	result := bytes.NewBuffer(nil)

	if _, err := result.Write([]byte{byte(gcfm.FType)}); err != nil {
		return nil, err
	}

	buf4 := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf4, gcfm.StartHeight)
	if _, err := result.Write(buf4); err != nil {
		return nil, err
	}

	if _, err := result.Write(gcfm.StopHash[:]); err != nil {
		return nil, err
	}

	return result.Bytes(), nil
}

func ParseGetCFilterMessage(r io.Reader) (GetCFilterMessage, error) {
	buf1 := make([]byte, 1)
	if _, err := io.ReadFull(r, buf1); err != nil {
		return GetCFilterMessage{}, err
	}
	ftype := FilterType(buf1[0])

	buf4 := make([]byte, 4)
	if _, err := io.ReadFull(r, buf4); err != nil {
		return GetCFilterMessage{}, err
	}
	height := binary.LittleEndian.Uint32(buf4)

	var hash [32]byte
	if _, err := io.ReadFull(r, hash[:]); err != nil {
		return GetCFilterMessage{}, err
	}

	// The height of the block with hash StopHash MUST be greater than or
	// equal to StartHeight, and the difference MUST be
	// strictly less than 1000.

	return GetCFilterMessage{
		FType:       ftype,
		StartHeight: height,
		StopHash:    hash,
	}, nil
}

type CFilterMessage struct {
	FType       FilterType
	BlockHash   [32]byte
	FilterBytes []byte // varint length prepended value
}

func (cf CFilterMessage) Command() string {
	return "cfilter"
}

func (cf *CFilterMessage) Serialize() ([]byte, error) {
	result := bytes.NewBuffer(nil)

	if _, err := result.Write([]byte{byte(cf.FType)}); err != nil {
		return nil, err
	}

	if _, err := result.Write(cf.BlockHash[:]); err != nil {
		return nil, err
	}

	lenBytes, err := encoding.EncodeVarInt(uint64(len(cf.FilterBytes)))
	if err != nil {
		return nil, err
	}

	if _, err := result.Write(lenBytes); err != nil {
		return nil, err
	}

	if _, err := result.Write(cf.FilterBytes); err != nil {
		return nil, err
	}

	return result.Bytes(), nil
}

func ParseCFilterMessage(r io.Reader) (CFilterMessage, error) {
	buf1 := make([]byte, 1)
	if _, err := io.ReadFull(r, buf1); err != nil {
		return CFilterMessage{}, err
	}
	ftype := FilterType(buf1[0])

	var hash [32]byte
	if _, err := io.ReadFull(r, hash[:]); err != nil {
		return CFilterMessage{}, err
	}

	length, err := encoding.ReadVarInt(r)
	if err != nil {
		return CFilterMessage{}, err
	}

	filterBytes := make([]byte, length)
	if _, err := io.ReadFull(r, filterBytes); err != nil {
		return CFilterMessage{}, err
	}

	return CFilterMessage{
		FType:       ftype,
		BlockHash:   hash,
		FilterBytes: filterBytes,
	}, nil
}

type GetCfHeadersMessage struct {
	FType       FilterType
	StartHeight uint32
	StopHash    [32]byte
}

func (cfh GetCfHeadersMessage) Command() string {
	return "getcfheaders"
}

func (cfh *GetCfHeadersMessage) Serialize() ([]byte, error) {
	result := bytes.NewBuffer(nil)

	if _, err := result.Write([]byte{byte(cfh.FType)}); err != nil {
		return nil, err
	}

	buf4 := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf4, cfh.StartHeight)
	if _, err := result.Write(buf4); err != nil {
		return nil, err
	}

	if _, err := result.Write(cfh.StopHash[:]); err != nil {
		return nil, err
	}

	return result.Bytes(), nil
}

func ParseGetCfHeadersMessage(r io.Reader) (GetCfHeadersMessage, error) {
	buf1 := make([]byte, 1)
	if _, err := io.ReadFull(r, buf1); err != nil {
		return GetCfHeadersMessage{}, err
	}
	ftype := FilterType(buf1[0])

	buf4 := make([]byte, 4)
	if _, err := io.ReadFull(r, buf4); err != nil {
		return GetCfHeadersMessage{}, err
	}
	height := binary.LittleEndian.Uint32(buf4)

	var hash [32]byte
	if _, err := io.ReadFull(r, hash[:]); err != nil {
		return GetCfHeadersMessage{}, err
	}

	// The height of the block with hash StopHash MUST be greater than or
	// equal to StartHeight, and the difference MUST be
	// strictly less than 1000.

	return GetCfHeadersMessage{
		FType:       ftype,
		StartHeight: height,
		StopHash:    hash,
	}, nil
}

type CfHeadersMessage struct {
	FType            FilterType
	StopHash         [32]byte
	PrevFilterHeader [32]byte   // The filter header preceding the first block in the requested range
	FilterHashes     [][32]byte // varint length prepended list of hashes
}

func (cfh CfHeadersMessage) Command() string {
	return "cfheaders"
}

func (cfh *CfHeadersMessage) Serialize() ([]byte, error) {
	result := bytes.NewBuffer(nil)

	if _, err := result.Write([]byte{byte(cfh.FType)}); err != nil {
		return nil, err
	}

	if _, err := result.Write(cfh.StopHash[:]); err != nil {
		return nil, err
	}

	if _, err := result.Write(cfh.PrevFilterHeader[:]); err != nil {
		return nil, err
	}

	lenBytes, err := encoding.EncodeVarInt(uint64(len(cfh.FilterHashes)))
	if err != nil {
		return nil, err
	}

	if _, err := result.Write(lenBytes); err != nil {
		return nil, err
	}

	for i := 0; i < len(cfh.FilterHashes); i++ {
		if _, err := result.Write(cfh.FilterHashes[i][:]); err != nil {
			return nil, err
		}
	}

	return result.Bytes(), nil
}

func ParseCfHeadersMessage(r io.Reader) (CfHeadersMessage, error) {
	buf1 := make([]byte, 1)
	if _, err := io.ReadFull(r, buf1); err != nil {
		return CfHeadersMessage{}, err
	}
	ftype := FilterType(buf1[0])

	var stopHash [32]byte
	if _, err := io.ReadFull(r, stopHash[:]); err != nil {
		return CfHeadersMessage{}, err
	}

	var prevFilter [32]byte
	if _, err := io.ReadFull(r, prevFilter[:]); err != nil {
		return CfHeadersMessage{}, err
	}

	numHashes, err := encoding.ReadVarInt(r)
	if err != nil {
		return CfHeadersMessage{}, err
	}

	filterHash := make([][32]byte, numHashes)
	for i := uint64(0); i < numHashes; i++ {
		if _, err := io.ReadFull(r, filterHash[i][:]); err != nil {
			return CfHeadersMessage{}, err
		}
	}

	return CfHeadersMessage{
		FType:            ftype,
		StopHash:         stopHash,
		PrevFilterHeader: prevFilter,
		FilterHashes:     filterHash,
	}, nil
}

type GetCfCheckPointMessage struct {
	FType    FilterType
	StopHash [32]byte
}

func (cfcp GetCfCheckPointMessage) Command() string {
	return "getcfcheckpt"
}

func (cfcp *GetCfCheckPointMessage) Serialize() ([]byte, error) {
	var result [33]byte
	result[0] = byte(cfcp.FType)

	copy(result[1:33], cfcp.StopHash[:])

	return result[:], nil
}

func ParseGetCfCheckPointMessage(r io.Reader) (GetCfCheckPointMessage, error) {
	buf1 := make([]byte, 1)
	if _, err := io.ReadFull(r, buf1); err != nil {
		return GetCfCheckPointMessage{}, err
	}
	ftype := FilterType(buf1[0])

	var stopHash [32]byte
	if _, err := io.ReadFull(r, stopHash[:]); err != nil {
		return GetCfCheckPointMessage{}, err
	}

	return GetCfCheckPointMessage{
		FType:    ftype,
		StopHash: stopHash,
	}, nil
}

type CfCheckPointMessage struct {
	FType         FilterType
	StopHash      [32]byte
	FilterHeaders [][32]byte // varint length prepended
}

func (cpm CfCheckPointMessage) Command() string {
	return "cfcheckpt"
}

func (cpm CfCheckPointMessage) Serialize() ([]byte, error) {
	result := bytes.NewBuffer(nil)

	if _, err := result.Write([]byte{byte(cpm.FType)}); err != nil {
		return nil, err
	}

	if _, err := result.Write(cpm.StopHash[:]); err != nil {
		return nil, err
	}

	numHeaderBytes, err := encoding.EncodeVarInt(uint64(len(cpm.FilterHeaders)))
	if err != nil {
		return nil, err
	}

	if _, err := result.Write(numHeaderBytes); err != nil {
		return nil, err
	}

	for i := 0; i < len(cpm.FilterHeaders); i++ {
		if _, err := result.Write(cpm.FilterHeaders[i][:]); err != nil {
			return nil, err
		}
	}

	return result.Bytes(), nil
}

func ParseCfCheckPointMessage(r io.Reader) (CfCheckPointMessage, error) {
	buf1 := make([]byte, 1)
	if _, err := io.ReadFull(r, buf1); err != nil {
		return CfCheckPointMessage{}, err
	}
	ftype := FilterType(buf1[0])

	var stopHash [32]byte
	if _, err := io.ReadFull(r, stopHash[:]); err != nil {
		return CfCheckPointMessage{}, err
	}

	numHeaders, err := encoding.ReadVarInt(r)
	if err != nil {
		return CfCheckPointMessage{}, err
	}

	filterHeaders := make([][32]byte, numHeaders)
	for i := uint64(0); i < numHeaders; i++ {
		if _, err := io.ReadFull(r, filterHeaders[i][:]); err != nil {
			return CfCheckPointMessage{}, err
		}
	}

	return CfCheckPointMessage{
		FType:         ftype,
		StopHash:      stopHash,
		FilterHeaders: filterHeaders,
	}, nil
}
