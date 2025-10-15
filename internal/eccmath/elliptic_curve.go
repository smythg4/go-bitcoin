package eccmath

import (
	"errors"
	"fmt"
	"math/big"
)

// bitcoin uses elliptic curve secp256k1: y^2 = x^3 + 7

type Curve struct {
	a, b FieldElement
	p    *big.Int
}

func NewCurve(a, b, p *big.Int) *Curve {
	return &Curve{
		a: NewFieldElement(a, p),
		b: NewFieldElement(b, p),
		p: p,
	}
}

func (c Curve) Equals(other Curve) bool {
	return c.a.Equals(other.a) && c.b.Equals(other.b)
}

func (c *Curve) GetInfPoint() Point {
	return Point{
		curve:      c,
		isInfinity: true,
	}
}

// you can use this once you determine a prime constant P
// var Secp256k1 = &Curve{
// 	a: NewFieldElement(0, P),
// 	b: NewFieldElement(7, P),
// }

type Point struct {
	x, y       FieldElement
	curve      *Curve
	isInfinity bool
}

func (c *Curve) NewPoint(x, y *big.Int) (Point, error) {
	yFe := NewFieldElement(y, c.p)
	xFe := NewFieldElement(x, c.p)

	ySqr := yFe.Pow(2)
	xCub := xFe.Pow(3)

	ax, err := c.a.Mul(xFe)
	if err != nil {
		return Point{}, err
	}

	lhs := ySqr
	rhs, err := xCub.Add(ax)
	if err != nil {
		return Point{}, err
	}
	rhs, err = rhs.Add(c.b)
	if err != nil {
		return Point{}, err
	}
	if !lhs.Equals(rhs) {
		return Point{}, fmt.Errorf("(%v, %v) is not on the curve", xFe, yFe)
	}

	return Point{
		x:          xFe,
		y:          yFe,
		curve:      c,
		isInfinity: false,
	}, nil
}

func (p Point) Equals(other Point) bool {
	if p.curve == nil || other.curve == nil {
		return p.curve == other.curve && p.x.Equals(other.x) && p.y.Equals(other.y)
	}
	if p.isInfinity && other.isInfinity && p.curve.Equals(*other.curve) {
		return true
	}
	if p.isInfinity || other.isInfinity {
		return false
	}
	return p.x.Equals(other.x) && p.y.Equals(other.y) &&
		p.curve.Equals(*other.curve)
}

func (p Point) Add(other Point) (Point, error) {
	if !p.curve.Equals(*other.curve) {
		return Point{}, fmt.Errorf("%v, %v are not on the same curve", p, other)
	}

	if p.isInfinity {
		return other, nil
	}
	if other.isInfinity {
		return p, nil
	}

	if p.x.Equals(other.x) && !p.y.Equals(other.y) {
		// if the points form a vertical line, return infinity for the curve
		return p.curve.GetInfPoint(), nil
	}

	if p.x.Equals(other.x) && p.y.Equals(other.y) && p.y.IsZero() {
		// if P1 == P2 and y == 0
		return p.curve.GetInfPoint(), nil
	}

	if p.x.Equals(other.x) && p.y.Equals(other.y) {
		// if P1 == P2
		return doublePoint(p)
	}

	slope, err := getSlope(p, other)
	if err != nil {
		return Point{}, err
	}

	x3, err := slope.Pow(2).Sub(p.x)
	if err != nil {
		return Point{}, err
	}
	x3, err = x3.Sub(other.x)
	if err != nil {
		return Point{}, err
	}

	y3, err := p.x.Sub(x3)
	if err != nil {
		return Point{}, err
	}
	y3, err = y3.Mul(slope)
	if err != nil {
		return Point{}, err
	}
	y3, err = y3.Sub(p.y)
	if err != nil {
		return Point{}, err
	}
	return Point{
		x:          x3,
		y:          y3,
		curve:      p.curve,
		isInfinity: false,
	}, nil
}

func (p Point) String() string {
	if p.isInfinity {
		return fmt.Sprintf("infinity (%d)", p.curve.p)
	}
	return fmt.Sprintf("(%v, %v)\n", p.x, p.y)
}

func getSlope(p1, p2 Point) (FieldElement, error) {
	if !p1.curve.Equals(*p2.curve) {
		return FieldElement{}, fmt.Errorf("%v, %v are not on the same curve", p1, p2)
	}

	numerator, err := p2.y.Sub(p1.y)
	if err != nil {
		return FieldElement{}, err
	}
	denom, err := p2.x.Sub(p1.x)
	if err != nil {
		return FieldElement{}, err
	}
	if denom.IsZero() {
		return FieldElement{}, errors.New("divide by zero")
	}

	return numerator.Div(denom)
}

func doubleSlope(p Point) (FieldElement, error) {
	xSqr := p.x.Pow(2)
	threeXSqr, err := xSqr.Mul(NewFieldElement(big.NewInt(3), p.curve.p))
	if err != nil {
		return FieldElement{}, nil
	}

	numerator, err := threeXSqr.Add(p.curve.a)
	if err != nil {
		return FieldElement{}, err
	}

	denom, err := p.y.Mul(NewFieldElement(big.NewInt(2), p.curve.p))
	if err != nil {
		return FieldElement{}, err
	}

	if denom.IsZero() {
		return FieldElement{}, errors.New("divide by zero")
	}

	return numerator.Div(denom)
}

func doublePoint(p Point) (Point, error) {
	slope, err := doubleSlope(p)
	if err != nil {
		return Point{}, err
	}
	sSqr := slope.Pow(2)
	twoX, err := p.x.Mul(NewFieldElement(big.NewInt(2), p.curve.p))
	if err != nil {
		return Point{}, err
	}
	x3, err := sSqr.Sub(twoX)
	if err != nil {
		return Point{}, err
	}
	x1Lessx3, err := p.x.Sub(x3)
	if err != nil {
		return Point{}, err
	}
	s2, err := x1Lessx3.Mul(slope)
	if err != nil {
		return Point{}, err
	}
	y3, err := s2.Sub(p.y)
	if err != nil {
		return Point{}, err
	}
	return Point{x: x3, y: y3, curve: p.curve, isInfinity: false}, nil
}

func (p Point) ScalarMul(n int) (Point, error) {
	result := p.curve.GetInfPoint()
	addend := p
	var err error

	for n > 0 {
		if n&1 == 1 { // if bit set
			result, err = result.Add(addend)
			if err != nil {
				return Point{}, err
			}
		}
		addend, err = addend.Add(addend) // double the addend
		if err != nil {
			return Point{}, err
		}
		n >>= 1 // shift the bit
	}
	return result, nil
}

func (p Point) ScalarMulBig(n *big.Int) (Point, error) {
	result := p.curve.GetInfPoint()
	addend := p
	var err error

	nCopy := new(big.Int).Set(n)
	zero := big.NewInt(0)
	one := big.NewInt(1)

	for nCopy.Cmp(zero) > 0 {
		if new(big.Int).And(nCopy, one).Cmp(one) == 0 { // if bit set
			result, err = result.Add(addend)
			if err != nil {
				return Point{}, err
			}
		}
		addend, err = addend.Add(addend) // double the addend
		if err != nil {
			return Point{}, err
		}
		nCopy.Rsh(nCopy, 1) // shift the bit
	}
	return result, nil
}

func (p Point) IsInf() bool {
	return p.isInfinity
}

func mustPoint(p Point, err error) Point {
	if err != nil {
		panic(err)
	}
	return p
}
