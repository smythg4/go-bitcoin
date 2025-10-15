package eccmath

import (
	"crypto/rand"
	"fmt"
	"go-bitcoin/internal/encoding"
	"math/big"
)

func MakeSecp256k1() *Curve {
	p := new(big.Int).Lsh(big.NewInt(1), 256)     // 2^256
	p.Sub(p, new(big.Int).Lsh(big.NewInt(1), 32)) // - 2^32
	p.Sub(p, big.NewInt(977))                     // - 977

	return &Curve{
		a: NewFieldElement(big.NewInt(0), p),
		b: NewFieldElement(big.NewInt(7), p),
		p: p,
	}
}

type CurveParams struct {
	Curve *Curve
	G     Point
	N     *big.Int
}

type Secp256k1Group struct {
	curve *Curve
	G     Point    // generator
	N     *big.Int // Order of G
}

func NewBitcoin() *Secp256k1Group {
	sep := MakeSecp256k1()
	gx := big.NewInt(0)
	gx.SetString("79BE667EF9DCBBAC55A06295CE870B07029BFCDB2DCE28D959F2815B16F81798", 16)
	gy := big.NewInt(0)
	gy.SetString("483ADA7726A3C4655DA4FBFC0E1108A8FD17B448A68554199C47D08FFB10D4B8", 16)

	n := big.NewInt(0)
	n.SetString("FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFEBAAEDCE6AF48A03BBFD25E8CD0364141", 16)
	g := mustPoint(sep.NewPoint(gx, gy))
	return &Secp256k1Group{
		curve: sep,
		G:     g,
		N:     n,
	}
}

func (s *Secp256k1Group) ScalarBaseMultiply(k *big.Int) Point {
	res, err := s.G.ScalarMulBig(k)
	if err != nil {
		panic(err)
	}
	return res
}

func (s *Secp256k1Group) Contains(p Point) bool {
	// n * P should equal infinity
	result, _ := p.ScalarMulBig(s.N)
	return result.IsInf()
}

func (s *Secp256k1Group) Sign(key *big.Int, z *big.Int) (Signature, error) {
	// consider implementing RFC 6979 in the future for deterministic ks
	k, err := randFieldElement(s.N)
	if err != nil {
		return Signature{}, fmt.Errorf("failed to generate random k: %w", err)
	}
	//k := big.NewInt(1234567890)

	R := s.ScalarBaseMultiply(k)

	r := new(big.Int).Mod(R.x.num, s.N)

	k_inv := new(big.Int).ModInverse(k, s.N)

	r_times_priv := new(big.Int).Mul(r, key)
	z_plus_r_priv := new(big.Int).Add(z, r_times_priv)

	sig_s := new(big.Int).Mul(z_plus_r_priv, k_inv)
	sig_s.Mod(sig_s, s.N)

	return Signature{r: r, s: sig_s}, nil
}

func randFieldElement(max *big.Int) (*big.Int, error) {
	// generate random in [0, max)

	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return nil, err
	}

	// ensure n >= 1
	if n.Cmp(big.NewInt(0)) == 0 {
		n.SetInt64(1)
	}

	return n, nil
}

type S256Field struct {
	FieldElement
}

func NewS256Field(num, prime *big.Int) S256Field {
	return S256Field{
		NewFieldElement(num, prime),
	}
}

func (fe S256Field) Sqrt() S256Field {
	exp := new(big.Int).Add(fe.prime, big.NewInt(1))
	exp.Div(exp, big.NewInt(4))
	return S256Field{
		NewFieldElement(new(big.Int).Exp(fe.num, exp, fe.prime), fe.prime),
	}
}

type S256Point struct {
	Point Point
	group *Secp256k1Group
}

func NewS256Point(p Point, g *Secp256k1Group) S256Point {
	return S256Point{
		Point: p,
		group: g,
	}
}

func (p S256Point) String() string {
	return fmt.Sprintf("S256Point(x=%064x, y=%064x)", p.Point.x.num, p.Point.y.num)
}

func (p *S256Point) Verify(z *big.Int, sig Signature) bool {
	N := p.group.N

	// s^-1 mod N
	s_inv := new(big.Int).ModInverse(sig.s, N)

	// u = z * s^-1 mod N
	u := new(big.Int).Mul(z, s_inv)
	u.Mod(u, N)

	// v = r * s^-1 mod N
	v := new(big.Int).Mul(sig.r, s_inv)
	v.Mod(v, N)

	// Point operations: R = u*G + v*P
	uG := p.group.ScalarBaseMultiply(u)
	vP, err := p.Point.ScalarMulBig(v)
	if err != nil {
		return false
	}
	total, err := uG.Add(vP)
	if err != nil {
		return false
	}

	// Verify R.x mod N == r
	return new(big.Int).Mod(total.x.num, N).Cmp(sig.r) == 0
}

func (p *S256Point) Serialize(compressed bool) []byte {
	if !compressed {
		// uncompressed
		result := make([]byte, 65)
		result[0] = 0x04 // uncompressed prefix

		xBytes := p.Point.x.num.Bytes()
		yBytes := p.Point.y.num.Bytes()

		// copy x
		copy(result[33-len(xBytes):33], xBytes)

		// copy y
		copy(result[65-len(yBytes):65], yBytes)

		return result
	} else {
		// compressed
		result := make([]byte, 33)
		if p.Point.y.num.Bit(0) == 0 {
			// even y -> 0x02
			result[0] = 0x02
		} else {
			// odd y -> 0x03
			result[0] = 0x03
		}
		xBytes := p.Point.x.num.Bytes()
		// copy x
		copy(result[33-len(xBytes):33], xBytes)

		return result
	}
}

func (p *S256Point) Deserialize(data []byte) (S256Point, error) {
	if len(data) >= 65 && data[0] == 0x04 {
		// Uncompressed format
		x := new(big.Int).SetBytes(data[1:33])  // bytes 1-32
		y := new(big.Int).SetBytes(data[33:65]) // bytes 33-64

		point, err := p.group.curve.NewPoint(x, y)
		if err != nil {
			return S256Point{}, err
		}
		return NewS256Point(point, p.group), nil
	} else if data[0] == 0x03 || data[0] == 0x02 {
		isEven := data[0] == 0x02
		x := new(big.Int).SetBytes(data[1:33]) // bytes 1-32

		xFe := NewS256Field(x, p.group.curve.p)
		seven := NewS256Field(big.NewInt(7), p.group.curve.p)

		// y^2 = x^3 + 7
		xCubedFe := S256Field{xFe.Pow(3)}
		y2Fe, err := xCubedFe.FieldElement.Add(seven.FieldElement)
		if err != nil {
			return S256Point{}, err
		}

		// take square root
		yFe := S256Field{y2Fe}.Sqrt()

		if (yFe.num.Bit(0) == 0) != isEven {
			// wrong parity - take other root (p - y)
			yFe.num = new(big.Int).Sub(p.group.curve.p, yFe.num)
		}

		point, err := p.group.curve.NewPoint(x, yFe.num)
		if err != nil {
			return S256Point{}, err
		}
		return NewS256Point(point, p.group), nil
	}

	return S256Point{}, fmt.Errorf("invalid SEC format")
}

func (p *S256Point) Address(compressed, testnet bool) string {
	data := p.Serialize(compressed)
	h160 := encoding.Hash160(data)
	prefix := 0x00
	if testnet {
		prefix = 0x6f // testnet prefix
	}
	return encoding.EncodeBase58Checksum(append([]byte{byte(prefix)}, h160...))
}
