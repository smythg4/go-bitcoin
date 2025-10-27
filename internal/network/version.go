package network

import (
	"bytes"
	"encoding/binary"
	"go-bitcoin/internal/encoding"
	"math/rand/v2"
	"net"
	"time"
)

type NetAddr struct {
	Services uint64
	Address  [16]byte
	Port     uint16
}

func NewNetAddr(services uint64, address [16]byte, port uint16) NetAddr {
	return NetAddr{
		Services: services,
		Address:  address,
		Port:     port,
	}
}

func (na NetAddr) String() string {
	ip := net.IP(na.Address[:])
	return ip.String()
}

func (na *NetAddr) Serialize() []byte {
	serviceBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(serviceBytes, na.Services)
	portBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(portBytes, na.Port)
	return append(serviceBytes, append(na.Address[:], portBytes...)...)
}

type VersionMessage struct {
	Version      int32 // default 70015
	Services     uint64
	TimeStamp    int64 // 64 bit UNIX time
	SenderAddr   NetAddr
	ReceiverAddr NetAddr
	Nonce        uint64
	UserAgent    string
	LatestBlock  int32
	Relay        bool
}

func DefaultVersionMessage(remoteIP net.IP, port uint16) VersionMessage {
	ip16 := remoteIP.To16()
	var addr [16]byte
	copy(addr[:], ip16)
	return VersionMessage{
		Version:   70015,
		Services:  8, // NODE_WITNESS (1<<3)
		TimeStamp: time.Now().Unix(),
		SenderAddr: NetAddr{
			Services: 0,
			Address:  [16]byte{},
			Port:     port,
		},
		ReceiverAddr: NetAddr{
			Services: 0,
			Address:  addr,
			Port:     port,
		},
		Nonce:       rand.Uint64(),
		UserAgent:   "/programmingbitcoin:0.1/",
		LatestBlock: 0,
		Relay:       false,
	}
}

func (vm *VersionMessage) Serialize() ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	// write version
	int32Buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(int32Buf, uint32(vm.Version))
	if _, err := buf.Write(int32Buf); err != nil {
		return nil, err
	}
	// write services
	int64Buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(int64Buf, vm.Services)
	if _, err := buf.Write(int64Buf); err != nil {
		return nil, err
	}

	// write timestamp
	binary.LittleEndian.PutUint64(int64Buf, uint64(vm.TimeStamp))
	if _, err := buf.Write(int64Buf); err != nil {
		return nil, err
	}
	// write receiver and sender addresses
	if _, err := buf.Write(vm.ReceiverAddr.Serialize()); err != nil {
		return nil, err
	}
	if _, err := buf.Write(vm.SenderAddr.Serialize()); err != nil {
		return nil, err
	}

	// write nonce
	binary.LittleEndian.PutUint64(int64Buf, vm.Nonce)
	if _, err := buf.Write(int64Buf); err != nil {
		return nil, err
	}

	// write user agent (prepended with varint length)
	userAgentLen := uint64(len(vm.UserAgent))
	userAgentVarInt, err := encoding.EncodeVarInt(userAgentLen)
	if err != nil {
		return nil, err
	}
	if _, err := buf.Write(userAgentVarInt); err != nil {
		return nil, err
	}
	if _, err := buf.Write([]byte(vm.UserAgent)); err != nil {
		return nil, err
	}

	// write height (is latest block right?)
	binary.LittleEndian.PutUint32(int32Buf, uint32(vm.LatestBlock))
	if _, err := buf.Write(int32Buf); err != nil {
		return nil, err
	}

	// write relay (depends on BIP37 - bloom fields)
	if vm.Relay {
		buf.Write([]byte{byte(0x01)})
	} else {
		buf.Write([]byte{byte(0x00)})
	}

	return buf.Bytes(), nil
}

func (vm VersionMessage) Command() string {
	return "version"
}
