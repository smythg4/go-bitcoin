package encoding

import (
	"fmt"
	"math"
)

type MerkleTree struct {
	total        int
	maxDepth     int
	nodes        [][][]byte
	currentDepth int
	currentIndex int
}

func NewMerkleTree(hashes [][]byte) (*MerkleTree, error) {
	total := len(hashes)
	md := int(math.Ceil(math.Log2(float64(total))))
	mt := &MerkleTree{
		total:        total,
		maxDepth:     md,
		nodes:        make([][][]byte, md+1),
		currentDepth: 0,
		currentIndex: 0,
	}
	currLevelHashes := hashes
	for i := mt.maxDepth; i >= 0; i-- {
		mt.nodes[i] = currLevelHashes
		if i > 0 {
			currLevelHashes = MerkleParentLevel(currLevelHashes)
		}
	}

	return mt, nil
}

// func PopulateTree(flagBits []byte, hashes [][]byte) (*MerkleTree, error) {
// 	for
// }

func (mt *MerkleTree) Up() {
	if mt.currentDepth == 0 {
		return
	}
	mt.currentDepth -= 1
	mt.currentIndex /= 2
}

func (mt *MerkleTree) Left() {
	if mt.IsLeaf() {
		return // Can't go down from leaf
	}

	mt.currentDepth += 1
	mt.currentIndex *= 2
}

func (mt *MerkleTree) Right() {
	if mt.IsLeaf() || !mt.RightExists() {
		return // Can't go down from leaf or if right doesn't exist
	}
	mt.currentDepth += 1
	mt.currentIndex = mt.currentIndex*2 + 1
}

func (mt *MerkleTree) Root() []byte {
	return mt.nodes[0][0]
}

func (mt *MerkleTree) SetCurrentNode(value [32]byte) {
	mt.nodes[mt.currentDepth][mt.currentIndex] = value[:]
}

func (mt *MerkleTree) GetCurrentNode() []byte {
	return mt.nodes[mt.currentDepth][mt.currentIndex]
}

func (mt *MerkleTree) GetLeftNode() []byte {
	if mt.IsLeaf() {
		return nil
	}
	return mt.nodes[mt.currentDepth+1][mt.currentIndex*2]
}

func (mt *MerkleTree) GetRightNode() []byte {
	if mt.IsLeaf() || !mt.RightExists() {
		return nil
	}
	return mt.nodes[mt.currentDepth+1][mt.currentIndex*2+1]
}

func (mt *MerkleTree) IsLeaf() bool {
	return mt.currentDepth == mt.maxDepth
}

func (mt *MerkleTree) RightExists() bool {
	return len(mt.nodes[mt.currentDepth+1]) > mt.currentIndex*2+1
}

func (mt MerkleTree) String() string {
	result := ""
	for i := 0; i <= mt.maxDepth; i++ { // ← Include maxDepth level (leaves)
		result += fmt.Sprintf("Level %d (%d hashes):\n", i, len(mt.nodes[i]))
		for j, hash := range mt.nodes[i] {
			if j > 0 && j%4 == 0 { // 4 hashes per line for readability
				result += "\n"
			}
			result += fmt.Sprintf("  %x...", hash[:4]) // First 4 bytes
		}
		result += "\n\n"
	}
	return result
}

func MerkleParent(l, r []byte) []byte {
	combined := append(l, r...)
	return Hash256(combined)
}

func MerkleParentLevel(hashes [][]byte) [][]byte {
	if len(hashes)%2 != 0 {
		hashes = append(hashes, hashes[len(hashes)-1])
	}
	plevel := make([][]byte, 0, len(hashes)/2)
	for i := 0; i < len(hashes); i += 2 {
		plevel = append(plevel, MerkleParent(hashes[i], hashes[i+1]))
	}
	return plevel
}

func MerkleRoot(hashes [][]byte) []byte {
	if len(hashes) == 0 {
		return nil // or panic("empty hash list")
	}
	currentHashes := hashes
	for len(currentHashes) > 1 {
		currentHashes = MerkleParentLevel(currentHashes)
	}
	return currentHashes[0]
}
