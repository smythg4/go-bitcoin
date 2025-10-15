package eccmath

import (
	"bytes"
	"fmt"
	"math/big"
)

type Signature struct {
	r *big.Int
	s *big.Int
}

func NewSignature(r, s int) Signature {
	return Signature{
		r: big.NewInt(int64(r)),
		s: big.NewInt(int64(s)),
	}
}

func (s Signature) String() string {
	return fmt.Sprintf("Signature(0x%064x, 0x%064x)", s.r, s.s)
}

func (s Signature) Serialize() []byte {
	rBytes := s.r.Bytes()
	sBytes := s.s.Bytes()

	// strip off all the leading 0s
	rBytes = bytes.TrimLeft(rBytes, "\x00")
	sBytes = bytes.TrimLeft(sBytes, "\x00")

	if len(rBytes) > 0 && rBytes[0]&0x80 != 0 {
		// prepend with 0x00 if high bit >= 0x80
		rBytes = append([]byte{0x00}, rBytes...)
	}
	if len(sBytes) > 0 && sBytes[0]&0x80 != 0 {
		// prepend with 0x00 if high bit >= 0x80
		sBytes = append([]byte{0x00}, sBytes...)
	}

	// Build DER structure
	result := []byte{0x30} // SEQUENCE marker

	// Total length (will calculate and insert)
	totalLen := 1 + 1 + len(rBytes) + 1 + 1 + len(sBytes)
	result = append(result, byte(totalLen))

	// r integer
	result = append(result, 0x02) // INTEGER marker
	result = append(result, byte(len(rBytes)))
	result = append(result, rBytes...)

	// s integer
	result = append(result, 0x02) // INTEGER marker
	result = append(result, byte(len(sBytes)))
	result = append(result, sBytes...)

	return result
}
