package network

import (
	"bytes"
	"fmt"
	"go-bitcoin/internal/encoding"
	"math/big"
	"testing"
)

func hashToInt(hash []byte) *big.Int {
	return new(big.Int).SetBytes(hash)
}

func TestBloomBasics(t *testing.T) {
	size := 10
	bitField := make([]byte, size)

	for _, word := range [][]byte{[]byte("hello world"), []byte("goodbye")} {
		for _, hash_func := range []func([]byte) []byte{encoding.Hash256, encoding.Hash160} {
			h := hash_func(word)
			bit := hashToInt(h).Mod(hashToInt(h), big.NewInt(int64(size)))
			bitField[bit.Int64()] = 1
		}
	}

	fmt.Println(bitField)

	expected := []byte{1, 1, 1, 0, 0, 0, 0, 0, 0, 1}

	if !bytes.Equal(expected, bitField) {
		t.Fatalf("expected: %v, got %v", expected, bitField)
	}
}

func TestBloomBitField(t *testing.T) {
	fieldSize := 2
	numFunctions := 2
	tweak := 42
	bitFieldSize := fieldSize * 8
	bitField := make([]byte, bitFieldSize)

	for _, word := range [][]byte{[]byte("hello world"), []byte("goodbye")} {
		fmt.Printf("\nWord: %s\n", word)
		for i := 0; i < numFunctions; i++ {
			seed := uint32(i)*uint32(encoding.BIP37_CONSTANT) + uint32(tweak)
			h := encoding.MurmurHash3(word, seed)
			bit := h % uint32(bitFieldSize)

			fmt.Printf("  i=%d, seed=%d, hash=%d, bit=%d\n", i, seed, h, bit)

			bitField[bit] = 1
		}
	}
	expected := []byte{0, 0, 0, 0, 0, 1, 1, 0, 0, 1, 1, 0, 0, 0, 0, 0}
	fmt.Printf("\nResult:   %v\n", bitField)
	fmt.Printf("Expected: %v\n", expected)
	if !bytes.Equal(expected, bitField) {
		t.Fatalf("expected: %v, got %v", expected, bitField)
	}
}

func TestExercise122(t *testing.T) {
	items := [][]byte{[]byte("Hello World"), []byte("Goodbye!")}
	size := 10
	funcCount := 5
	tweak := 99
	bitFieldSize := size * 8
	bitField := make([]byte, bitFieldSize)
	for _, item := range items {
		fmt.Printf("\nItem: %s\n", item)
		for i := 0; i < funcCount; i++ {
			seed := uint32(i)*uint32(encoding.BIP37_CONSTANT) + uint32(tweak)
			h := encoding.MurmurHash3(item, seed)
			bit := h % uint32(bitFieldSize)

			fmt.Printf("  i=%d, seed=%d, hash=%d, bit=%d\n", i, seed, h, bit)

			bitField[bit] = 1
		}
	}
	expected := "4000600a080000010940"
	b, err := encoding.BitFieldToBytes(bitField)
	if err != nil {
		t.Fatal(err)
	}
	actual := fmt.Sprintf("%x", b)

	fmt.Printf("\nResult:   %s\n", actual)
	fmt.Printf("Expected: %s\n", expected)
	if actual != expected {
		t.Fatalf("expected: %s, got %s", expected, actual)
	}
}
