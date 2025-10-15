# go-bitcoin

A Bitcoin implementation in Go, following **Programming Bitcoin** by Jimmy Song.

## Current Progress

**Chapter 3: Elliptic Curve Cryptography**

### Implemented

**Finite Field Arithmetic** (`internal/field_elements/`)
- Field element operations over prime fields
- Addition, subtraction, multiplication, division
- Modular exponentiation and multiplicative inverse
- Proper handling of negative numbers in modular arithmetic

**Elliptic Curves** (`internal/elliptic_curve/`)
- Point representation on elliptic curves (y² = x³ + ax + b)
- Point validation (curve equation verification)
- Point at infinity handling
- Point addition (general case and vertical line case)
- Point doubling (tangent line case)
- Scalar multiplication (repeated addition)

### Example Usage

```go
// Create a curve y² = x³ + 0x + 7 over F₂₂₃
curve := ellipticcurve.NewCurve(0, 7, 223)

// Create a point on the curve
point, err := ellipticcurve.NewPoint(47, 71, curve)

// Scalar multiplication
result, err := point.ScalarMul(21)  // Returns point at infinity
```

## Project Structure

```
go-bitcoin/
├── main.go
├── go.mod
└── internal/
    ├── field_elements/
    │   └── field_elements.go
    └── elliptic_curve/
        └── elliptic_curve.go
```

## Next Steps

- Chapter 3 continued: Optimized scalar multiplication (double-and-add algorithm)
- Chapter 4+: Moving toward secp256k1 and Bitcoin's specific elliptic curve
