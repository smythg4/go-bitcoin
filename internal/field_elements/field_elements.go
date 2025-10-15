package field_elements

import (
	"errors"
	"fmt"
	"math/big"
)

type FieldElement struct {
	num   *big.Int
	prime *big.Int
}

func (fe FieldElement) String() string {
	return fmt.Sprintf("%d (mod %d)", fe.num, fe.prime)
}

func NewFieldElement(num, prime int) FieldElement {
	n := big.NewInt(int64(num))
	p := big.NewInt(int64(prime))
	n.Mod(n, p)
	return FieldElement{
		num:   n,
		prime: p,
	}
}

func (fe FieldElement) Equals(other FieldElement) bool {
	return fe.num.Cmp(other.num) == 0 && fe.prime.Cmp(other.prime) == 0
}

func (fe FieldElement) Add(other FieldElement) (FieldElement, error) {
	if fe.prime.Cmp(other.prime) != 0 {
		return FieldElement{}, errors.New("cannot add two numbers in different Fields")
	}
	num := new(big.Int)
	num.Add(fe.num, other.num)
	num.Mod(num, fe.prime)
	return FieldElement{num: num, prime: fe.prime}, nil
}

func (fe FieldElement) Sub(other FieldElement) (FieldElement, error) {
	if fe.prime.Cmp(other.prime) != 0 {
		return FieldElement{}, errors.New("cannot subtract two numbers in different Fields")
	}
	num := new(big.Int)
	num.Sub(fe.num, other.num)
	num.Mod(num, fe.prime)
	return FieldElement{num: num, prime: fe.prime}, nil
}

func (fe FieldElement) Mul(other FieldElement) (FieldElement, error) {
	if fe.prime.Cmp(other.prime) != 0 {
		return FieldElement{}, errors.New("cannot multiply two numbers in different Fields")
	}
	num := new(big.Int)
	num.Mul(fe.num, other.num)
	num.Mod(num, fe.prime)
	return FieldElement{num: num, prime: fe.prime}, nil
}

func (fe FieldElement) Inv() FieldElement {
	inv := new(big.Int)
	inv.ModInverse(fe.num, fe.prime)
	return FieldElement{num: inv, prime: fe.prime}
}

func (fe FieldElement) Div(other FieldElement) (FieldElement, error) {
	if fe.prime.Cmp(other.prime) != 0 {
		return FieldElement{}, errors.New("cannot divide two numbers in different Fields")
	}
	otherinv := other.Inv()
	return fe.Mul(otherinv)
}

func (fe FieldElement) Pow(exponent int) FieldElement {
	num := new(big.Int)
	num.Exp(fe.num, big.NewInt(int64(exponent)), fe.prime)
	return FieldElement{num: num, prime: fe.prime}
}

func (fe FieldElement) IsZero() bool {
	return fe.num.Cmp(big.NewInt(0)) == 0
}
