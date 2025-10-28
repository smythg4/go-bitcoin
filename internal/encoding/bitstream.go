package encoding

import "fmt"

// BitStreams are readable and writable streams of individual bits.
type BitStream struct {
	data      []byte
	ByteIndex int
	BitIndex  int
}

// Instantiates a new writable bit stream
func NewBitStream() BitStream {
	return BitStream{
		data:      make([]byte, 1),
		ByteIndex: 0,
		BitIndex:  0,
	}
}

// Instantiates a new bit stream reading data from byte slice.
func NewBitStreamFromSlice(data []byte) BitStream {
	return BitStream{
		data:      data,
		ByteIndex: 0,
		BitIndex:  0,
	}
}

// Appends the bit b to the end of the stream.
func (bs *BitStream) WriteBit(b bool) {
	bVal := byte(0x00)
	if b {
		bVal = byte(0x01)
	}
	currByte := bs.data[bs.ByteIndex]

	currByte |= (bVal << (7 - bs.BitIndex))
	bs.data[bs.ByteIndex] = currByte

	bs.BitIndex++
	if bs.BitIndex%8 == 0 {
		bs.BitIndex = 0
		bs.ByteIndex++
	}
	if bs.ByteIndex >= len(bs.data) {
		// automatically extend the byte slice if you've reached the limit
		bs.data = append(bs.data, 0x00)
	}
}

// Reads the next available bit from the stream.
func (bs *BitStream) ReadBit() byte {
	res := bs.data[bs.ByteIndex] & (1 << (7 - bs.BitIndex))
	bs.BitIndex++

	if bs.BitIndex >= 8 {
		bs.BitIndex = 0
		bs.ByteIndex++
	}
	if res != 0 {
		return 0x01
	}
	return 0x00
}

// Appends the k least significant bits of integer n to the end of the stream in big-endian
// bit order
func (bs *BitStream) WriteBitsBigEndian(n, k int) error {
	if k <= 0 {
		return fmt.Errorf("invalid input for n, k = %d, %d", n, k)
	}

	// write bits
	for i := k - 1; i >= 0; i-- {
		// extract the ith bit from n
		bit := (n >> i) & 1
		bs.WriteBit(bit == 1)
	}

	return nil
}

// Reads the next available k bits from the stream and interprets them as the least
// significant bits of a big-endian integer
func (bs *BitStream) ReadBitsBigEndian(k int) (int, error) {
	if k <= 0 {
		return 0, nil
	}

	result := 0
	for i := 0; i < k; i++ {
		bit := bs.ReadBit()
		result = (result << 1) | int(bit)
	}
	return result, nil
}

func (bs *BitStream) Bytes() []byte {
	// If we never advanced past the first byte and haven't written any bits, return empty
	if bs.ByteIndex == 0 && bs.BitIndex == 0 {
		return []byte{}
	}

	// If we're mid-byte, include the current byte
	if bs.BitIndex > 0 {
		return bs.data[:bs.ByteIndex+1]
	}

	// Otherwise, return up to current byte
	return bs.data[:bs.ByteIndex]
}
