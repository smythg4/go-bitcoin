package encoding

import (
	"bytes"
	"fmt"
	"testing"
)

func TestMerkleConstruction(t *testing.T) {
	// Create 27 dummy transaction hashes
	numTxs := 27
	hashes := make([][]byte, numTxs)
	for i := 0; i < numTxs; i++ {
		hash := make([]byte, 32)
		hash[0] = byte(i) // Give each hash a unique first byte
		hashes[i] = hash
	}

	mt, err := NewMerkleTree(hashes)
	if err != nil {
		t.Fatal(err)
	}

	// Verify tree structure
	if mt.total != numTxs {
		t.Errorf("expected total=%d, got %d", numTxs, mt.total)
	}

	// Verify leaves are at bottom (maxDepth level)
	if len(mt.nodes[mt.maxDepth]) != numTxs {
		t.Errorf("expected %d leaves at maxDepth, got %d", numTxs, len(mt.nodes[mt.maxDepth]))
	}

	// Verify root is at top (level 0)
	if len(mt.nodes[0]) != 1 {
		t.Errorf("expected 1 root hash, got %d", len(mt.nodes[0]))
	}

	fmt.Printf("Built merkle tree with %d transactions, depth=%d\n", numTxs, mt.maxDepth)
	fmt.Printf("Root: %x\n", mt.nodes[0][0])
	fmt.Println(mt)
}

func TestMerkleNavigation(t *testing.T) {
	// Create 27 dummy transaction hashes
	numTxs := 27
	hashes := make([][]byte, numTxs)
	for i := 0; i < numTxs; i++ {
		hash := make([]byte, 32)
		hash[0] = byte(i) // Give each hash a unique first byte
		hashes[i] = hash
	}

	mt, err := NewMerkleTree(hashes)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println("Going left...")
	for !mt.IsLeaf() {
		fmt.Printf("%x\n", mt.GetCurrentNode())
		mt.Left()
	}
	fmt.Printf("%x\n", mt.GetCurrentNode())

	fmt.Println("Going back up...")
	for !bytes.Equal(mt.GetCurrentNode(), mt.Root()) {
		fmt.Printf("%x\n", mt.GetCurrentNode())
		mt.Up()
	}
	fmt.Printf("%x\n", mt.GetCurrentNode())

	fmt.Println("Going right...")
	for !mt.IsLeaf() {
		fmt.Printf("%x\n", mt.GetCurrentNode())
		if mt.RightExists() {
			mt.Right()
		} else {
			fmt.Println("No right node to hit, going left...")
			mt.Left() // Fall back to left when right doesn't exist
		}
	}
	fmt.Printf("%x\n", mt.GetCurrentNode())

	fmt.Println("Going back up...")
	for !bytes.Equal(mt.GetCurrentNode(), mt.Root()) {
		fmt.Printf("%x\n", mt.GetCurrentNode())
		mt.Up()
	}
	fmt.Printf("%x\n", mt.GetCurrentNode())

	fmt.Println("Alternating directions...")
	goRight := false
	for !mt.IsLeaf() {
		fmt.Printf("%x\n", mt.GetCurrentNode())
		if mt.RightExists() && goRight {
			fmt.Println("going right...")
			mt.Right()
			goRight = false
		} else {
			fmt.Println("going left...")
			mt.Left()
			goRight = true
		}
	}
	fmt.Printf("%x\n", mt.GetCurrentNode())
}
