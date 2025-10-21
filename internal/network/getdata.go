package network

import (
	"bytes"
	"encoding/binary"
	"go-bitcoin/internal/encoding"
)

type DataType uint32

const (
	DATA_TYPE_ERROR DataType = iota
	DATA_TYPE_TX
	DATA_TYPE_BLOCK
	DATA_TYPE_FILTERED_BLOCK
	DATA_TYPE_CMPCT_BLOCK
)

type DataItem struct {
	Type       DataType
	Identifier [32]byte
}

type GetDataMessage struct {
	Data []DataItem
}

func NewGetDataMessage() GetDataMessage {
	return GetDataMessage{
		Data: []DataItem{},
	}
}

func (gd *GetDataMessage) AddData(dType DataType, id [32]byte) {
	gd.Data = append(gd.Data, DataItem{
		Type:       dType,
		Identifier: id,
	})
}

func (gd *GetDataMessage) Serialize() ([]byte, error) {
	buf := bytes.NewBuffer(nil)

	// number of items (VarInt)
	count, err := encoding.EncodeVarInt(uint64(len(gd.Data)))
	if err != nil {
		return nil, err
	}
	buf.Write(count)

	// loop through each data item
	for _, item := range gd.Data {
		// data type (4 bytes LE)
		binary.Write(buf, binary.LittleEndian, item.Type)

		// id (32 bytes, already LE from parsing)
		buf.Write(item.Identifier[:])
	}

	return buf.Bytes(), nil
}

func (gd GetDataMessage) Command() string {
	return "getdata"
}
