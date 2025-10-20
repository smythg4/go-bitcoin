package block

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"go-bitcoin/internal/encoding"
	"io"
	"math/big"
	"slices"
	"time"
)

const LOWEST_BITS uint32 = 0x1d00ffff // maximum target (difficulty 1)

var TESTNET_GENESIS_BLOCK = []byte{
	0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x3b, 0xa3, 0xed, 0xfd,
	0x7a, 0x7b, 0x12, 0xb2, 0x7a, 0xc7, 0x2c, 0x3e,
	0x67, 0x76, 0x8f, 0x61, 0x7f, 0xc8, 0x1b, 0xc3,
	0x88, 0x8a, 0x51, 0x32, 0x3a, 0x9f, 0xb8, 0xaa,
	0x4b, 0x1e, 0x5e, 0x4a, 0xda, 0xe5, 0x49, 0x4d,
	0xff, 0xff, 0x00, 0x1d, 0x1a, 0xa4, 0xae, 0x18,
}

var MAINNET_GENESIS_BLOCK = []byte{
	0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x3b, 0xa3, 0xed, 0xfd,
	0x7a, 0x7b, 0x12, 0xb2, 0x7a, 0xc7, 0x2c, 0x3e,
	0x67, 0x76, 0x8f, 0x61, 0x7f, 0xc8, 0x1b, 0xc3,
	0x88, 0x8a, 0x51, 0x32, 0x3a, 0x9f, 0xb8, 0xaa,
	0x4b, 0x1e, 0x5e, 0x4a, 0x29, 0xab, 0x5f, 0x49,
	0xff, 0xff, 0x00, 0x1d, 0x1d, 0xac, 0x2b, 0x7c,
}

type Block struct {
	Version    uint32   // 4 bytes LE
	PrevBlock  [32]byte // LE
	MerkleRoot [32]byte // LE
	TimeStamp  uint32   // 4 bytes LE, Unix epoch seconds
	Bits       uint32   // 4 bytes LE, compact difficulty target
	Nonce      uint32   // 4 bytes LE, proof of work nonce
	TxHashes   [][32]byte
}

func NewBlock(version uint32, prevBlock, merkleRoot [32]byte, timeStamp uint32, bits, nonce uint32, txHashes [][32]byte) Block {
	return Block{
		Version:    version,
		PrevBlock:  prevBlock,
		MerkleRoot: merkleRoot,
		TimeStamp:  timeStamp,
		Bits:       bits,
		Nonce:      nonce,
		TxHashes:   txHashes, // hashes have to be ordered
	}
}

func ParseBlock(r io.Reader) (Block, error) {
	var b Block
	buf := make([]byte, 4)

	// Version (4 bytes)
	if _, err := io.ReadFull(r, buf); err != nil {
		return Block{}, err
	}
	b.Version = binary.LittleEndian.Uint32(buf)

	// PrevBlock (32 bytes)
	if _, err := io.ReadFull(r, b.PrevBlock[:]); err != nil {
		return Block{}, err
	}

	// MerkleRoot (32 bytes)
	if _, err := io.ReadFull(r, b.MerkleRoot[:]); err != nil {
		return Block{}, err
	}

	// TimeStamp (4 bytes)
	if _, err := io.ReadFull(r, buf); err != nil {
		return Block{}, err
	}
	b.TimeStamp = binary.LittleEndian.Uint32(buf)

	// Bits (4 bytes)
	if _, err := io.ReadFull(r, buf); err != nil {
		return Block{}, err
	}
	b.Bits = binary.LittleEndian.Uint32(buf)

	// Nonce (4 bytes)
	if _, err := io.ReadFull(r, buf); err != nil {
		return Block{}, err
	}
	b.Nonce = binary.LittleEndian.Uint32(buf)

	return b, nil
}

func (b *Block) Serialize() ([]byte, error) {
	// error signature remains for API consistency. This should never fail.
	buf := make([]byte, 80)
	binary.LittleEndian.PutUint32(buf[0:4], b.Version)

	copy(buf[4:36], b.PrevBlock[:])
	copy(buf[36:68], b.MerkleRoot[:])

	binary.LittleEndian.PutUint32(buf[68:72], b.TimeStamp)
	binary.LittleEndian.PutUint32(buf[72:76], b.Bits)
	binary.LittleEndian.PutUint32(buf[76:80], b.Nonce)
	return buf, nil
}

func (b *Block) Time() time.Time {
	return time.Unix(int64(b.TimeStamp), 0)
}

func (b *Block) Hash() ([]byte, error) {
	// should never fail
	serialized, _ := b.Serialize()

	return encoding.Hash256(serialized), nil
}

func (b *Block) ID() string {
	// should never fail
	hash, _ := b.Hash()
	slices.Reverse(hash)
	return fmt.Sprintf("%x", hash)
}

func (b *Block) IsBip9() bool {
	// top 3 bits of version need to be 0b001
	return (b.Version >> 29) == 0b001
}

func (b *Block) IsBip91() bool {
	// 4th bit of version needs to be 1
	return (b.Version>>4)&1 == 1
}

func (b *Block) IsBip141() bool {
	// SegWit (BIP141) - checks bit 1
	return (b.Version>>1)&1 == 1
}

func (b *Block) bitsToTarget() *big.Int {
	exponent := b.Bits >> 24     // take last (high) byte
	coeff := b.Bits & 0x00ffffff // take the other bytes

	target := big.NewInt(int64(coeff))

	// multiply by 256^(exponent - 3)
	if exponent <= 3 {
		target.Rsh(target, uint(8*(3-exponent)))
	} else {
		target.Lsh(target, uint(8*(exponent-3)))
	}

	return target
}

func TargetToBits(target *big.Int) uint32 {
	// turns a target integer back into bits
	rawBytes := target.Bytes()

	// if high bit is set, prepend 0x00 to avoid negative interpretation
	if len(rawBytes) > 0 && rawBytes[0] > 0x7f {
		rawBytes = append([]byte{0x00}, rawBytes...)
	}
	exponent := uint32(len(rawBytes))

	// extract the first 3 bytes as coefficient (BE)
	coefficient := uint32(0)
	if len(rawBytes) >= 1 {
		coefficient |= uint32(rawBytes[0]) << 16
	}
	if len(rawBytes) >= 2 {
		coefficient |= uint32(rawBytes[1]) << 8
	}
	if len(rawBytes) >= 3 {
		coefficient |= uint32(rawBytes[2])
	}

	// pack: top byte = exponent, bottom 3 bytes = coefficient
	return (exponent << 24) | coefficient
}

func (b *Block) Difficulty() *big.Int {
	// 0xffff and 0x1d are constants associated with the Bitcoin difficulty formula
	// difficulty = ( 0xffff * 256 ^ (0x1d-3) ) / target
	target := b.bitsToTarget()
	diffBase := big.NewInt(int64(0xffff))
	diffBase.Lsh(diffBase, uint(8*(0x1d-3)))
	diff := new(big.Int).Div(diffBase, target)
	return diff
}

func (b *Block) CheckProofOfWork() bool {
	hash, _ := b.Hash()
	slices.Reverse(hash)
	// set bytes uses BE ordering
	proof := new(big.Int).SetBytes(hash)
	return proof.Cmp(b.bitsToTarget()) < 0
}

func (b *Block) CalcNewBits(firstBlock, lastBlock Block) uint32 {
	// calculates the new bits given the first and last block of a 2,016 block difficulty adjustment period
	const TWO_WEEKS = 60 * 60 * 24 * 14
	EIGHT_WEEKS := big.NewInt(TWO_WEEKS * 4)
	THREE_HALF_DAYS := big.NewInt(TWO_WEEKS / 4)

	timeDiff := big.NewInt(int64(lastBlock.TimeStamp - firstBlock.TimeStamp))

	if timeDiff.Cmp(EIGHT_WEEKS) > 0 {
		timeDiff = EIGHT_WEEKS
	}
	if timeDiff.Cmp(THREE_HALF_DAYS) < 0 {
		timeDiff = THREE_HALF_DAYS
	}
	newTarget := new(big.Int).Mul(lastBlock.bitsToTarget(), timeDiff)
	newTarget.Div(newTarget, big.NewInt(TWO_WEEKS))

	// Clamp to maximum target (minimum difficulty)
	maxTarget := &Block{Bits: LOWEST_BITS}
	if newTarget.Cmp(maxTarget.bitsToTarget()) > 0 {
		return LOWEST_BITS // Can't be easier than genesis difficulty
	}

	return TargetToBits(newTarget)
}

func (b *Block) ValidateMerkleRoot() bool {
	hashes := make([][]byte, len(b.TxHashes))
	for i, hash := range b.TxHashes {
		reversed := make([]byte, 32)
		copy(reversed, hash[:])
		slices.Reverse(reversed)
		hashes[i] = reversed
	}
	merkleRoot := encoding.MerkleRoot(hashes)
	return bytes.Equal(b.MerkleRoot[:], merkleRoot)
}
