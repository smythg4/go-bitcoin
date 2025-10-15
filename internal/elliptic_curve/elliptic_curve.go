package ellipticcurve

import (
	"errors"
	"fmt"
	"go-bitcoin/internal/field_elements"
)

// bitcoin uses elliptic curve secp256k1: y^2 = x^3 + 7

const P int = 223

type Curve struct {
	a, b field_elements.FieldElement
	p    int
}

func NewCurve(a, b, p int) *Curve {
	return &Curve{
		a: field_elements.NewFieldElement(a, p),
		b: field_elements.NewFieldElement(b, p),
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
// 	a: field_elements.NewFieldElement(0, P),
// 	b: field_elements.NewFieldElement(7, P),
// }

type Point struct {
	x, y       field_elements.FieldElement
	curve      *Curve
	isInfinity bool
}

func NewPoint(x, y int, c *Curve) (Point, error) {
	yFe := field_elements.NewFieldElement(y, c.p)
	xFe := field_elements.NewFieldElement(x, c.p)

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

func getSlope(p1, p2 Point) (field_elements.FieldElement, error) {
	if !p1.curve.Equals(*p2.curve) {
		return field_elements.FieldElement{}, fmt.Errorf("%v, %v are not on the same curve", p1, p2)
	}

	numerator, err := p2.y.Sub(p1.y)
	if err != nil {
		return field_elements.FieldElement{}, err
	}
	denom, err := p2.x.Sub(p1.x)
	if err != nil {
		return field_elements.FieldElement{}, err
	}
	if denom.IsZero() {
		return field_elements.FieldElement{}, errors.New("divide by zero")
	}

	return numerator.Div(denom)
}

func doubleSlope(p Point) (field_elements.FieldElement, error) {
	xSqr := p.x.Pow(2)
	threeXSqr, err := xSqr.Mul(field_elements.NewFieldElement(3, p.curve.p))
	if err != nil {
		return field_elements.FieldElement{}, nil
	}

	numerator, err := threeXSqr.Add(p.curve.a)
	if err != nil {
		return field_elements.FieldElement{}, err
	}

	denom, err := p.y.Mul(field_elements.NewFieldElement(2, p.curve.p))
	if err != nil {
		return field_elements.FieldElement{}, err
	}

	if denom.IsZero() {
		return field_elements.FieldElement{}, errors.New("divide by zero")
	}

	return numerator.Div(denom)
}

func doublePoint(p Point) (Point, error) {
	slope, err := doubleSlope(p)
	if err != nil {
		return Point{}, err
	}
	sSqr := slope.Pow(2)
	twoX, err := p.x.Mul(field_elements.NewFieldElement(2, p.curve.p))
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
	res := p
	var err error
	for i := 0; i < n-1; i++ {
		res, err = res.Add(p)
		if err != nil {
			return Point{}, err
		}
	}
	return res, nil
}
