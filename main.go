package main

import (
	"fmt"
	"go-bitcoin/internal/transactions"
	"log"
)

func main() {
	// // Create a private key from a secret
	// secret := big.NewInt(0xdeadbeef54321)
	// privateKey := keys.NewPrivateKey(secret)

	// // Generate public key
	// publicKey := privateKey.PublicKey()

	// // Generate Bitcoin addresses
	// mainnetAddr := publicKey.Address(true, false) // compressed, mainnet
	// testnetAddr := publicKey.Address(true, true)  // compressed, testnet

	// fmt.Printf("Mainnet address: %s\n", mainnetAddr)
	// fmt.Printf("Testnet address: %s\n", testnetAddr)

	// // Export private key in WIF format
	// wif := privateKey.Serialize(true, false) // compressed, mainnet
	// fmt.Printf("Private key (WIF): %s\n", wif)

	// // Sign a message
	// z := big.NewInt(1234567890)
	// sig, _ := privateKey.Sign(z)

	// // Verify signature
	// valid := publicKey.Verify(z, sig)
	// fmt.Printf("Signature valid: %v\n", valid)

	// // Serialize signature in DER format
	// derSig := sig.Serialize()
	// fmt.Printf("DER signature: %x\n", derSig)

	// testVarInt := uint64(255)
	// data, err := encoding.EncodeVarInt(testVarInt)
	// if err != nil {
	// 	panic(err)
	// }
	// fmt.Println("Original int: ", testVarInt)
	// fmt.Printf("Encoded: %x\n", data)
	// back, err := encoding.ReadVarInt(bytes.NewReader(data))
	// if err != nil {
	// 	panic(err)
	// }
	// fmt.Printf("Decoded: %d\n", back)

	fmt.Println()
	fetcher := transactions.NewTxFetcher()
	txId := "e0fc453aa494912627ca3d93c411e8b5f1c8e8d81d5a07af023d45426f224fab"
	tx, err := fetcher.Fetch(txId, true, true)
	if err != nil {
		fmt.Printf("Error fetching TxID (%s)\n%v\n", txId, err)
		log.Fatal()
	}

	fmt.Println(tx)
}
