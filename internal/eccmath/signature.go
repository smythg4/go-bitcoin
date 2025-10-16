package eccmath

import (
	"bytes"
	"errors"
	"fmt"
	"io"
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

func ParseSignature(reader io.Reader) (Signature, error) {
	// check for 0x30 DER sequence marker
	marker := make([]byte, 1)
	n, err := reader.Read(marker)
	if err != nil || n != 1 {
		return Signature{}, fmt.Errorf("failed to read DER marker: %w", err)
	}
	if marker[0] != 0x30 {
		return Signature{}, errors.New("missing 0x30 DER marker")
	}

	// read total length byte -- don't actually use it
	lengthBuf := make([]byte, 1)
	n, err = reader.Read(lengthBuf)
	if err != nil || n != 1 {
		return Signature{}, fmt.Errorf("failed to read total length: %w", err)
	}

	// Parse r
	if _, err := io.ReadFull(reader, marker); err != nil {
		return Signature{}, fmt.Errorf("failed to read r INTEGER marker: %w", err)
	}
	if marker[0] != 0x02 {
		return Signature{}, errors.New("missing 0x02 INTEGER marker for r")
	}

	if _, err := io.ReadFull(reader, lengthBuf); err != nil {
		return Signature{}, fmt.Errorf("failed to read r length: %w", err)
	}
	rLen := int(lengthBuf[0])

	rBytes := make([]byte, rLen)
	if _, err := io.ReadFull(reader, rBytes); err != nil {
		return Signature{}, fmt.Errorf("failed to read r bytes: %w", err)
	}

	// no need to strip leading zeroes if padding was used
	r := new(big.Int).SetBytes(rBytes)

	// Parse s
	if _, err := io.ReadFull(reader, marker); err != nil {
		return Signature{}, fmt.Errorf("failed to read s INTEGER marker: %w", err)
	}
	if marker[0] != 0x02 {
		return Signature{}, errors.New("missing 0x02 INTEGER marker for s")
	}

	if _, err := io.ReadFull(reader, lengthBuf); err != nil {
		return Signature{}, fmt.Errorf("failed to read s length: %w", err)
	}
	sLen := int(lengthBuf[0])

	sBytes := make([]byte, sLen)
	if _, err := io.ReadFull(reader, sBytes); err != nil {
		return Signature{}, fmt.Errorf("failed to read s bytes: %w", err)
	}

	// no need to strip leading zeroes if padding was used
	s := new(big.Int).SetBytes(sBytes)

	return Signature{
		r: r,
		s: s,
	}, nil
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
