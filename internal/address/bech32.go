package address

import (
	"bytes"
	"fmt"
	"strings"
)

const charset = "qpzry9x8gf2tvdw0s3jn54khce6mua7l" // lookup table

var generator = []int{0x3b6a57b2, 0x26508e6d, 0x1ea119fa, 0x3d4233dd, 0x2a1462b3}

// Encode encodes hrp(human-readable part) and data(32bit data array), returns Bech32 / or error
// if hrp is uppercase, return uppercase Bech32
func Encode(hrp string, data []int) (string, error) {
	// validate hrp
	if (len(hrp) + len(data) + 7) > 90 {
		return "", fmt.Errorf("too long: hrp length=%d, data length=%d", len(hrp), len(data))
	}
	if len(hrp) < 1 {
		return "", fmt.Errorf("invalid hrp: hrp=%v", hrp)
	}
	for p, c := range hrp {
		if c < 33 || c > 126 {
			return "", fmt.Errorf("invalid character human-readable part : hrp[%d]=%d", p, c)
		}
	}
	if strings.ToUpper(hrp) != hrp && strings.ToLower(hrp) != hrp {
		return "", fmt.Errorf("mix case: hrp=%v", hrp)
	}
	lower := strings.ToLower(hrp) == hrp
	hrp = strings.ToLower(hrp)
	combined := append(data, createChecksum(hrp, data)...)
	var ret bytes.Buffer
	ret.WriteString(hrp)
	ret.WriteString("1")
	for idx, p := range combined {
		if p < 0 || p >= len(charset) {
			return "", fmt.Errorf("invalid data: data[%d]=%d", idx, p)
		}
		ret.WriteByte(charset[p])
	}
	if lower {
		return ret.String(), nil
	}
	return strings.ToUpper(ret.String()), nil
}

func encodeBech32(witnessVersion byte, witnessProgram []byte, hrp string) (string, error) {
	// Convert witness version and program to 5-bit groups
	data := []int{int(witnessVersion)}

	// convert the witness program from 8-bit to 5-bit
	programInts := make([]int, len(witnessProgram))
	for i, b := range witnessProgram {
		programInts[i] = int(b)
	}

	converted, err := convertbits(programInts, 8, 5, true)
	if err != nil {
		return "", err
	}

	// concatenate version + converted program
	data = append(data, converted...)

	return Encode(hrp, data)
}

func polymod(values []int) int {
	chk := 1
	for _, v := range values {
		top := chk >> 25
		chk = (chk&0x1ffffff)<<5 ^ v
		for i := 0; i < 5; i++ {
			if (top>>uint(i))&1 == 1 {
				chk ^= generator[i]
			}
		}
	}
	return chk
}

func hrpExpand(hrp string) []int {
	ret := []int{}

	for _, c := range hrp {
		ret = append(ret, int(c>>5))
	}
	ret = append(ret, 0)
	for _, c := range hrp {
		ret = append(ret, int(c&31))
	}
	return ret
}

func verifyChecksum(hrp string, data []int) bool {
	return polymod(append(hrpExpand(hrp), data...)) == 1
}

func createChecksum(hrp string, data []int) []int {
	values := append(append(hrpExpand(hrp), data...), []int{0, 0, 0, 0, 0, 0}...)
	mod := polymod(values) ^ 1
	ret := make([]int, 6)
	for p := 0; p < len(ret); p++ {
		ret[p] = (mod >> uint(5*(5-p))) & 31
	}
	return ret
}

func convertbits(data []int, frombits, tobits uint, pad bool) ([]int, error) {
	acc := 0
	bits := uint(0)
	ret := []int{}
	maxv := (1 << tobits) - 1

	for idx, value := range data {
		if value < 0 || (value>>frombits) != 0 {
			return nil, fmt.Errorf("invalid data range: data[%d]=%d (frombits=%d)", idx, value, frombits)
		}
		acc = (acc << frombits) | value
		bits += frombits
		for bits >= tobits {
			bits -= tobits
			ret = append(ret, (acc>>bits)&maxv)
		}
	}
	if pad {
		if bits > 0 {
			ret = append(ret, (acc<<(tobits-bits))&maxv)
		}
	} else if bits >= frombits {
		return nil, fmt.Errorf("illegal zero padding")
	} else if ((acc << (tobits - bits)) & maxv) != 0 {
		return nil, fmt.Errorf("non-zero padding")
	}

	return ret, nil
}
