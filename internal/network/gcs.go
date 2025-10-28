package network

import (
	"fmt"
	"go-bitcoin/internal/encoding"
	"slices"
)

type GolombCodedSet struct {
	NumItems uint64 // number of itmes in the filter
	P        uint   // false positive rate parameter (19 for BIP 158 basic filter)
	M        uint64 // modulus (784931 for BIP 158 basic filter)
	data     []byte // compressed filter data
}

// NewGCS constructs a new GCS filter from a list of items
func NewGCS(items [][]byte, k0, k1 uint64) (*GolombCodedSet, error) {
	n := uint64(len(items))

	// BIP158 defaults
	m := uint64(784931)
	p := uint(19)

	// hash all items to the range [0, n*m)
	setItems, err := hashedSetConstruct(items, k0, k1, m)
	if err != nil {
		return nil, fmt.Errorf("failed to generate hash set: %w", err)
	}

	// sort hashed values
	slices.Sort(setItems)

	// delta encode and Golomb encode
	outputStream := encoding.NewBitStream()
	lastVal := uint64(0)
	for _, item := range setItems {
		delta := item - lastVal
		golombEncode(&outputStream, int(delta), int(p))
		lastVal = item
	}

	return &GolombCodedSet{
		NumItems: n,
		P:        p,
		M:        m,
		data:     outputStream.Bytes(),
	}, nil
}

// Match checks if a single item might be in the filter
func (g *GolombCodedSet) Match(item []byte, k0, k1 uint64) (bool, error) {
	targetHash, err := hashToRange(item, g.NumItems, g.M, k0, k1)
	if err != nil {
		return false, fmt.Errorf("failed to hash target: %w", err)
	}

	stream := encoding.NewBitStreamFromSlice(g.data)
	lastVal := uint64(0)

	for i := uint64(0); i < g.NumItems; i++ {
		delta, err := golombDecode(&stream, int(g.P))
		if err != nil {
			return false, fmt.Errorf("failed to decode: %w", err)
		}

		currentVal := lastVal + delta
		if currentVal == targetHash {
			return true, nil // found it!
		}
		if currentVal > targetHash {
			return false, nil // passed it, not in filter
		}
		lastVal = currentVal
	}

	return false, nil // reach end without finding it
}

// MatchAny checks if any of the items might be in the filter
func (g *GolombCodedSet) MatchAny(items [][]byte, k0, k1 uint64) (bool, error) {
	for _, item := range items {
		match, err := g.Match(item, k0, k1)
		if err != nil {
			return false, err
		}
		if match {
			return true, nil
		}
	}
	return false, nil
}

// Serialize returns the complete BIP 158 filter: [N (varint)][filter_data]
func (g *GolombCodedSet) Serialize() ([]byte, error) {
	nBytes, err := encoding.EncodeVarInt(g.NumItems)
	if err != nil {
		return nil, err
	}
	return append(nBytes, g.data...), nil
}

func hashToRange(item []byte, n, m uint64, k0, k1 uint64) (uint64, error) {
	// BIP 158 constraints to avoid overflow
	if n >= (1 << 32) {
		return 0, fmt.Errorf("n (%d) must be < 2^32", n)
	}
	if m >= (1 << 32) {
		return 0, fmt.Errorf("m (%d) must be < 2^32", m)
	}
	if n == 0 {
		return 0, nil // Empty filter edge case
	}
	f := n * m
	hash := encoding.SipHash24(k0, k1, item)

	// BIP 158 uses multiply-and-shift instead of modulo for fast reduction
	// This computes (hash * f) >> 64, which maps hash to range [0, f)
	return fastReduction(hash, f), nil
}

// fastReduction implements the multiply-and-shift technique from BIP 158
// It computes (v * n) >> 64, which efficiently maps v to range [0, n)
// This avoids the expensive modulo operation while maintaining uniform distribution
func fastReduction(v, n uint64) uint64 {
	// Split into 32-bit components for 64x64->128 bit multiplication
	vHi := v >> 32
	vLo := uint64(uint32(v))
	nHi := n >> 32
	nLo := uint64(uint32(n))

	// Compute partial products
	vnpHi := vHi * nHi
	vnpMid := vHi * nLo
	npvMid := nHi * vLo
	vnpLo := vLo * nLo

	// Combine partial products with carry propagation
	carry := (uint64(uint32(vnpMid)) + uint64(uint32(npvMid)) + (vnpLo >> 32)) >> 32

	// Return upper 64 bits of the 128-bit product
	return vnpHi + (vnpMid >> 32) + (npvMid >> 32) + carry
}

func hashedSetConstruct(rawItems [][]byte, k0, k1 uint64, m uint64) ([]uint64, error) {
	n := uint64(len(rawItems))

	setItems := make([]uint64, len(rawItems))

	for i := range rawItems {
		setValue, err := hashToRange(rawItems[i], n, m, k0, k1)
		if err != nil {
			return nil, fmt.Errorf("failed to hash rawItem[%d] = %x: %w", i, rawItems[i], err)
		}
		setItems[i] = setValue
	}

	return setItems, nil
}

func golombEncode(s *encoding.BitStream, x int, p int) {
	q := x >> p

	for q > 0 {
		s.WriteBit(true)
		q--
	}
	s.WriteBit(false)

	s.WriteBitsBigEndian(x, p)
}

func golombDecode(s *encoding.BitStream, p int) (uint64, error) {
	q := 0
	for s.ReadBit() == 0x01 {
		q++
	}

	r, err := s.ReadBitsBigEndian(p)
	if err != nil {
		return 0, fmt.Errorf("failed to read %d bits from stream: %w", p, err)
	}

	x := (q << p) + r
	return uint64(x), nil
}
