package address

import (
	"fmt"
	"go-bitcoin/internal/encoding"
)

type Network int

const (
	MAINNET Network = iota
	TESTNET
)

// Bech32 HRP returns the HRP for bech32 address
func (n Network) Bech32HRP() string {
	if n == TESTNET {
		return "tb"
	}
	return "bc"
}

func (n Network) P2PKHVersion() byte {
	if n == TESTNET {
		return 0x6F
	}
	return 0x00
}

func (n Network) P2SHVersion() byte {
	if n == TESTNET {
		return 0xC4
	}
	return 0x05
}

type AddrType int

const (
	P2PKH  AddrType = iota // base58check
	P2SH                   // base58check
	P2WPKH                 // bech32, 20 bytes
	P2WSH                  // bech32, 32 bytes
	P2TR                   // bech32m, 32 bytes -- not implemented
)

type Address struct {
	Type    AddrType
	Network Network
	String  string
}

// FromHash160 creates a P2PKH or P2SH address from a hash160
func FromHash160(hash160 []byte, addrType AddrType, net Network) (*Address, error) {
	var prefix byte
	var addrString string

	switch addrType {
	case P2PKH:
		prefix = net.P2PKHVersion()
		addrString = encoding.EncodeBase58Checksum(append([]byte{prefix}, hash160...))
	case P2SH:
		prefix = net.P2SHVersion()
		addrString = encoding.EncodeBase58Checksum(append([]byte{prefix}, hash160...))
	default:
		return nil, fmt.Errorf("unsupported address type: %v", addrType)
	}

	return &Address{
		String:  addrString,
		Type:    addrType,
		Network: net,
	}, nil
}

// FromPublicKey creates an address from a public key
func FromPublicKey(pubkey []byte, addrType AddrType, net Network) (*Address, error) {
	hash160 := encoding.Hash160(pubkey)
	return FromHash160(hash160, addrType, net)
}

// FromWitnessProgram creates a bech32 address from a witness program
func FromWitnessProgram(version byte, program []byte, net Network) (*Address, error) {
	// validate program length
	if len(program) != 20 && len(program) != 32 {
		return nil, fmt.Errorf("invalid witness program length: %d", len(program))
	}

	var addrType AddrType
	if version == 0 {
		if len(program) == 20 {
			addrType = P2WPKH
		} else {
			addrType = P2WSH
		}
	} else {
		return nil, fmt.Errorf("unsupported witness version: %d", version)
	}

	hrp := net.Bech32HRP()
	bech32String, err := encodeBech32(version, program, hrp)
	if err != nil {
		return nil, err
	}

	return &Address{
		String:  bech32String,
		Type:    addrType,
		Network: net,
	}, nil
}
