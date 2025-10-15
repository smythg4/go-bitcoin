package main

import (
	"fmt"
	"go-bitcoin/internal/keys"

	"math/big"
)

func main() {
	// secret
	secret, _ := new(big.Int).SetString("mysupersecret", 58)

	// private key
	privateKey := keys.NewPrivateKey(secret)

	// public key
	publicKey := privateKey.PublicKey()

	fmt.Println("Converting your key into bytes...")
	dataFull := publicKey.Serialize(false)
	dataComp := publicKey.Serialize(true)

	keyFull, err := publicKey.Deserialize(dataFull)
	if err != nil {
		fmt.Println("uncompressed deserialization failed:", err)
	} else if keyFull.Point.Equals(publicKey.Point) {
		fmt.Println("✓ Uncompressed round-trip successful")
	} else {
		fmt.Println("✗ Uncompressed round-trip FAILED")
	}

	keyComp, err := publicKey.Deserialize(dataComp)
	if err != nil {
		fmt.Println("compressed deserialization failed:", err)
	} else if keyComp.Point.Equals(publicKey.Point) {
		fmt.Println("✓ Compressed round-trip successful")
	} else {
		fmt.Println("✗ Compressed round-trip FAILED")
	}

	sig, err := privateKey.Sign(big.NewInt(1234567890))
	if err != nil {
		panic(err)
	}
	fmt.Println(sig)
	bytes := sig.Serialize()
	fmt.Printf("%x\n", bytes)

	fmt.Printf("Address: %v\n", publicKey.Address(false, true))

	fmt.Println()
	fmt.Println()

	fmt.Println("WIF:", privateKey.Serialize(false, true))
}
