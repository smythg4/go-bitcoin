package main

import (
	"fmt"
	"go-bitcoin/internal/keys"
	"math/big"
)

func main() {
	// secret
	secret := big.NewInt(0xdeadbeef54321)

	// private key
	privateKey := keys.NewPrivateKey(secret)

	// public key
	publicKey := privateKey.PublicKey()

	fmt.Println("Converting your key into bytes...")
	dataFull := publicKey.SecSerialize(false)
	dataComp := publicKey.SecSerialize(true)

	keyFull, err := publicKey.SecDeserialize(dataFull)
	if err != nil {
		fmt.Println("uncompressed deserialization failed:", err)
	} else if keyFull.Point.Equals(publicKey.Point) {
		fmt.Println("✓ Uncompressed round-trip successful")
	} else {
		fmt.Println("✗ Uncompressed round-trip FAILED")
	}

	keyComp, err := publicKey.SecDeserialize(dataComp)
	if err != nil {
		fmt.Println("compressed deserialization failed:", err)
	} else if keyComp.Point.Equals(publicKey.Point) {
		fmt.Println("✓ Compressed round-trip successful")
	} else {
		fmt.Println("✗ Compressed round-trip FAILED")
	}
}
