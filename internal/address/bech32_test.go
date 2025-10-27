package address

import (
	"encoding/hex"
	"testing"
)

func TestBech32(t *testing.T) {
	// Known test vector from BIP 173
	program, _ := hex.DecodeString("751e76e8199196d454941c45d1b3a323f1433bd6")

	addr, err := encodeBech32(0, program, "bc")
	if err != nil {
		t.Fatalf("encoding failed: %v", err)
	}

	expected := "bc1qw508d6qejxtdg4y5r3zarvary0c5xw7kv8f3t4"
	if addr != expected {
		t.Errorf("got:  %s\nwant: %s", addr, expected)
	}
}

func TestBech32Mainnet(t *testing.T) {
	tests := []struct {
		name     string
		version  byte
		program  string // hex
		hrp      string
		expected string
	}{
		{
			name:     "P2WPKH mainnet",
			version:  0,
			program:  "751e76e8199196d454941c45d1b3a323f1433bd6",
			hrp:      "bc",
			expected: "bc1qw508d6qejxtdg4y5r3zarvary0c5xw7kv8f3t4",
		},
		{
			name:     "P2WSH mainnet",
			version:  0,
			program:  "1863143c14c5166804bd19203356da136c985678cd4d27a1b8c6329604903262",
			hrp:      "bc",
			expected: "bc1qrp33g0q5c5txsp9arysrx4k6zdkfs4nce4xj0gdcccefvpysxf3qccfmv3",
		},
		{
			name:     "P2WPKH testnet",
			version:  0,
			program:  "751e76e8199196d454941c45d1b3a323f1433bd6",
			hrp:      "tb",
			expected: "tb1qw508d6qejxtdg4y5r3zarvary0c5xw7kxpjzsx",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			program, _ := hex.DecodeString(tt.program)
			result, err := encodeBech32(tt.version, program, tt.hrp)
			if err != nil {
				t.Fatalf("encoding failed: %v", err)
			}
			if result != tt.expected {
				t.Errorf("got:  %s\nwant: %s", result, tt.expected)
			}
		})
	}
}
