package main

import (
	"fmt"
	"go-bitcoin/internal/keys"

	"math/big"
)

func main() {
	// Create a private key from a secret
	secret := big.NewInt(0xdeadbeef54321)
	privateKey := keys.NewPrivateKey(secret)

	// Generate public key
	publicKey := privateKey.PublicKey()

	// Generate Bitcoin addresses
	mainnetAddr := publicKey.Address(true, false) // compressed, mainnet
	testnetAddr := publicKey.Address(true, true)  // compressed, testnet

	fmt.Printf("Mainnet address: %s\n", mainnetAddr)
	fmt.Printf("Testnet address: %s\n", testnetAddr)

	// Export private key in WIF format
	wif := privateKey.Serialize(true, false) // compressed, mainnet
	fmt.Printf("Private key (WIF): %s\n", wif)

	// Sign a message
	z := big.NewInt(1234567890)
	sig, _ := privateKey.Sign(z)

	// Verify signature
	valid := publicKey.Verify(z, sig)
	fmt.Printf("Signature valid: %v\n", valid)

	// Serialize signature in DER format
	derSig := sig.Serialize()
	fmt.Printf("DER signature: %x\n", derSig)
}
