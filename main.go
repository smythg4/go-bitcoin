package main

import (
	"bytes"
	"fmt"
	"go-bitcoin/internal/block"
	"go-bitcoin/internal/network"
	"log"
	"net"
)

const TESTNET_PORT int = 18333
const MAINNET_PORT int = 8333
const MAINNET_SEEDS string = "seed.bitcoin.sipa.be"
const TESTNET_SEEDS string = "testnet-seed.bitcoin.jonasschnelli.ch"

func main() {
	dns := MAINNET_SEEDS
	port := MAINNET_PORT
	genBlockReader := bytes.NewReader(block.MAINNET_GENESIS_BLOCK)

	ips, err := net.LookupIP(dns)
	if err != nil {
		log.Fatal(err)
	}
	var node *network.SimpleNode

	for _, ip := range ips {
		if ip.To4() == nil {
			continue
		}

		addr := fmt.Sprintf("%s:%d", ip.String(), port)
		fmt.Printf("Trying %s...\n", addr)
		node, err = network.NewSimpleNode(ip.String(), port, false, false)
		if err != nil {
			fmt.Printf("  Failed: %v\n", err)
			continue
		}
		fmt.Printf("  Connected!\n")
		// Use this node connection

		break
	}
	defer node.Close()

	previous, err := block.ParseBlock(genBlockReader)
	if err != nil {
		log.Fatal(err)
	}
	prevHash, err := previous.Hash()
	if err != nil {
		log.Fatal(err)
	}
	first := previous

	count := 1
	expectedBits := block.LOWEST_BITS

	err = node.Handshake()
	if err != nil {
		log.Fatal(err)
	}
	for i := 0; i < 100; i++ {
		getHeaders, err := network.NewGetHeadersMessage(70015, 1, [32]byte(prevHash), nil)
		if err != nil {
			log.Fatal(err)
		}
		if err := node.Send(&getHeaders); err != nil {
			log.Fatal(err)
		}
		// Wait for headers response
		env, err := node.ReceiveHeaders()
		if err != nil {
			log.Fatal(err)
		}
		// Parse headers
		headers, err := network.ParseHeadersMessage(bytes.NewReader(env.Payload))
		if err != nil {
			log.Fatal(err)
		}
		for _, header := range headers.Blocks {
			//fmt.Println(header.ID())
			if !header.CheckProofOfWork() {
				fmt.Printf("bad PoW at block %d\n", count)
			}
			if header.PrevBlock != [32]byte(prevHash) {
				fmt.Printf("discontinous block at %d\n", count)
			}

			// Calculate new difficulty at boundary blocks
			if count%2016 == 0 {
				expectedBits = header.CalcNewBits(first, previous)
				fmt.Printf("Block %d: Adjusting difficulty\n", count)
				fmt.Printf("   %x\n", expectedBits)
				first = header
			}
			if header.Bits != expectedBits {
				fmt.Printf("bad bits at block %d\n", count)
			}
			previous = header
			prevHash, _ = previous.Hash()
			count += 1
		}
		// fmt.Printf("Received %d headers!\n", len(headers.Blocks))
		// for i, b := range headers.Blocks[:min(5, len(headers.Blocks))] {
		// 	fmt.Printf("Block %d: %s\n", i, b.ID())
		// }
	}

}
