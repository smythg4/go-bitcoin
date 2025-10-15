package eccmath

import (
	"fmt"
	"math/big"
)

type Signature struct {
	r *big.Int
	s *big.Int
}

func NewSignature(r, s int) Signature {
	return Signature{
		r: big.NewInt(int64(r)),
		s: big.NewInt(int64(s)),
	}
}

func (s Signature) String() string {
	return fmt.Sprintf("Signature(0x%064x, 0x%064x)", s.r, s.s)
}
