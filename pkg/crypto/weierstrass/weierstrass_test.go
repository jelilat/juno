// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style license that can
// be found in the LICENCE file.

package weierstrass

import (
	"crypto/rand"
	"math/big"
	"testing"
)

// isInfinity returns true if the x and y coordinates of a point
// represent the point at infinity.
func isInfinity(x, y *big.Int) bool {
	return x.Sign() == 0 && y.Sign() == 0
}

// BenchmarkMarshalUnmarshal runs benchmarks on marshalling and
// un-marshalling points.
func BenchmarkMarshalUnmarshal(b *testing.B) {
	curve := Stark()
	_, x, y, _ := GenerateKey(curve, rand.Reader)

	b.Run("Uncompressed", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			buf := Marshal(curve, x, y)
			xx, yy := Unmarshal(curve, buf)
			if xx.Cmp(x) != 0 || yy.Cmp(y) != 0 {
				b.Error("Unmarshal output different from Marshal input")
			}
		}
	})

	b.Run("Compressed", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			buf := MarshalCompressed(curve, x, y)
			xx, yy := UnmarshalCompressed(curve, buf)
			if xx.Cmp(x) != 0 || yy.Cmp(y) != 0 {
				b.Error("Unmarshal output different from Marshal input")
			}
		}
	})
}

// BenchmarkScalarBaseMult runs a benchmark on the scalar base
// multiplication operation on a curve.
func BenchmarkScalarBaseMult(b *testing.B) {
	curve := Stark()
	pvt, _, _, _ := GenerateKey(curve, rand.Reader)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		x, _ := curve.ScalarBaseMult(pvt)
		// Prevent the compiler from optimizing out the operation.
		pvt[0] ^= byte(x.Bits()[0])
	}
}

// BenchmarkScalarBaseMult runs a benchmark on the scalar multiplication
// operation on a curve.
func BenchmarkScalarMult(b *testing.B) {
	curve := Stark()
	_, x, y, _ := GenerateKey(curve, rand.Reader)
	pvt, _, _, _ := GenerateKey(curve, rand.Reader)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		x, y = curve.ScalarMult(x, y, pvt)
	}
}

// TestOffCurve checks whether a point that is not on a curve can be
// detected.
func TestOffCurve(t *testing.T) {
	curve := Stark()
	x, y := new(big.Int).SetInt64(1), new(big.Int).SetInt64(1)
	if curve.IsOnCurve(x, y) {
		t.Error("point off curve is claimed to be on the curve")
	}

	// XXX: Marshal curve. Note that this is essentially the body of the
	// Marshal function but given that will panic when the point is off
	// the curve, it is unsuitable here.
	byteLen := (curve.Params().BitSize + 7) / 8
	b := make([]byte, 1+2*byteLen)
	b[0] = 4 // uncompressed point
	x.FillBytes(b[1 : 1+byteLen])
	y.FillBytes(b[1+byteLen : 1+2*byteLen])

	x1, y1 := Unmarshal(curve, b)
	if x1 != nil || y1 != nil {
		t.Error("unmarshaling a point not on the curve succeeded")
	}
}

// TestInfinity checks whether the methods on a curve return valid
// values when some input value is ∞.
func TestInfinity(t *testing.T) {
	curve := Stark()
	x0, y0 := new(big.Int), new(big.Int)
	xG, yG := curve.Params().Gx, curve.Params().Gy

	if !isInfinity(curve.ScalarMult(xG, yG, curve.Params().N.Bytes())) {
		t.Errorf("x^q != ∞")
	}

	if !isInfinity(curve.ScalarMult(xG, yG, []byte{0})) {
		t.Errorf("x^0 != ∞")
	}

	if !isInfinity(curve.ScalarMult(x0, y0, []byte{1, 2, 3})) {
		t.Errorf("∞^k != ∞")
	}

	if !isInfinity(curve.ScalarMult(x0, y0, []byte{0})) {
		t.Errorf("∞^0 != ∞")
	}

	if !isInfinity(curve.ScalarBaseMult(curve.Params().N.Bytes())) {
		t.Errorf("b^q != ∞")
	}

	if !isInfinity(curve.ScalarBaseMult([]byte{0})) {
		t.Errorf("b^0 != ∞")
	}

	if !isInfinity(curve.Double(x0, y0)) {
		t.Errorf("2∞ != ∞")
	}

	nMinusOne := new(big.Int).Sub(curve.Params().N, big.NewInt(1))
	x, y := curve.ScalarMult(xG, yG, nMinusOne.Bytes())
	x, y = curve.Add(x, y, xG, yG)
	if !isInfinity(x, y) {
		t.Errorf("x^(q-1) + x != ∞")
	}

	x, y = curve.Add(xG, yG, x0, y0)
	if x.Cmp(xG) != 0 || y.Cmp(yG) != 0 {
		t.Error("x+∞ != x")
	}

	x, y = curve.Add(x0, y0, xG, yG)
	if x.Cmp(xG) != 0 || y.Cmp(yG) != 0 {
		t.Error("∞+x != x")
	}

	if curve.IsOnCurve(x0, y0) {
		t.Error("IsOnCurve(∞) == true")
	}

	xx, yy := Unmarshal(curve, Marshal(curve, x0, y0))
	if xx != nil || yy != nil {
		t.Error("Unmarshal(Marshal(∞)) did not return an error")
	}

	// We don't test UnmarshalCompressed(MarshalCompressed(∞)) because
	// there are two valid points with x = 0.
	xx, yy = Unmarshal(curve, []byte{0x00})
	if xx != nil || yy != nil {
		t.Error("Unmarshal(∞) did not return an error")
	}
	byteLen := (curve.Params().BitSize + 7) / 8
	buf := make([]byte, byteLen*2+1)
	buf[0] = 4 // Uncompressed format.
	if xx, yy := Unmarshal(curve, buf); xx != nil || yy != nil {
		t.Errorf("Unmarshal((0,0)) did not return an error")
	}
}

// TestMarshal checks whether a point can be serialised correctly.
func TestMarshal(t *testing.T) {
	curve := Stark()
	_, x, y, err := GenerateKey(curve, rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}
	serialized := Marshal(curve, x, y)
	xx, yy := Unmarshal(curve, serialized)
	if xx == nil {
		t.Fatal("failed to unmarshal")
	}
	if xx.Cmp(x) != 0 || yy.Cmp(y) != 0 {
		t.Fatal("unmarshal returned different values")
	}
}

// TestUnmarshalToLargeCoordinates checks whether a point with (invalid)
// large coordinates will be deserialised.
func TestUnmarshalToLargeCoordinates(t *testing.T) {
	curve := Stark()
	p := curve.Params().P
	byteLen := (p.BitLen() + 7) / 8

	// Set x to be greater than curve's parameter P – specifically, to
	// P+5. Set y to mod_sqrt(x^3 - 3x + B)) so that (x mod P = 5 , y) is
	// on the curve.
	x := new(big.Int).Add(p, big.NewInt(5))
	y := curve.Params().short(x)
	y.ModSqrt(y, p)

	invalid := make([]byte, byteLen*2+1)
	invalid[0] = 4 // uncompressed encoding
	x.FillBytes(invalid[1 : 1+byteLen])
	y.FillBytes(invalid[1+byteLen:])

	if X, Y := Unmarshal(curve, invalid); X != nil || Y != nil {
		t.Error("Unmarshal accepts invalid X coordinate")
	}
}

// TestInvalidCoordinates tests [big.Int] values that are not valid
// field elements i.e. negative or larger than P. They are expected to
// return false from IsOnCurve, all other behaviour is undefined.
func TestInvalidCoordinates(t *testing.T) {
	curve := Stark()
	checkIsOnCurveFalse := func(name string, x, y *big.Int) {
		if curve.IsOnCurve(x, y) {
			t.Errorf("IsOnCurve(%s) unexpectedly returned true", name)
		}
	}

	p := curve.Params().P
	_, x, y, _ := GenerateKey(curve, rand.Reader)
	xx, yy := new(big.Int), new(big.Int)

	// Check if the sign is getting dropped.
	xx.Neg(x)
	checkIsOnCurveFalse("-x, y", xx, y)
	yy.Neg(y)
	checkIsOnCurveFalse("x, -y", x, yy)

	// Check if negative values are reduced modulo P.
	xx.Sub(x, p)
	checkIsOnCurveFalse("x-P, y", xx, y)
	yy.Sub(y, p)
	checkIsOnCurveFalse("x, y-P", x, yy)

	// Check if positive values are reduced modulo P.
	xx.Add(x, p)
	checkIsOnCurveFalse("x+P, y", xx, y)
	yy.Add(y, p)
	checkIsOnCurveFalse("x, y+P", x, yy)

	// Check if the overflow is dropped.
	xx.Add(x, new(big.Int).Lsh(big.NewInt(1), 535))
	checkIsOnCurveFalse("x+2⁵³⁵, y", xx, y)
	yy.Add(y, new(big.Int).Lsh(big.NewInt(1), 535))
	checkIsOnCurveFalse("x, y+2⁵³⁵", x, yy)

	// Check if P is treated like zero (if possible).
	// y^2 = x^3 - 3x + B
	// y = mod_sqrt(x^3 - 3x + B)
	// y = mod_sqrt(B) if x = 0
	// If there is no modsqrt, there is no point with x = 0, can't test
	// x = P.
	if yy := new(big.Int).ModSqrt(curve.Params().B, p); yy != nil {
		if !curve.IsOnCurve(big.NewInt(0), yy) {
			t.Fatal("(0, mod_sqrt(B)) is not on the curve?")
		}
		checkIsOnCurveFalse("P, y", p, yy)
	}
}

// TestMarshallCompressed checks whether points can be compressed and
// deserialised.
func TestMarshallCompressed(t *testing.T) {
	curve := Stark()
	_, x, y, err := GenerateKey(curve, rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}
	if !curve.IsOnCurve(x, y) {
		t.Fatal("invalid test point")
	}

	got := MarshalCompressed(curve, x, y)

	X, Y := UnmarshalCompressed(curve, got)
	if X == nil || Y == nil {
		t.Fatal("UnmarshalCompressed failed unexpectedly")
	}

	if !curve.IsOnCurve(X, Y) {
		t.Error("UnmarshalCompressed returned a point not on the curve")
	}

	if X.Cmp(x) != 0 || Y.Cmp(y) != 0 {
		t.Errorf(
			"point did not round-trip correctly: got (%v, %v), want (%v, %v)",
			X, Y, x, y)
	}
}

// TestLargeIsOnCurve checks whether a point with (invalid) large
// coordinates is reported to be on the curve.
func TestLargeIsOnCurve(t *testing.T) {
	curve := Stark()
	large := big.NewInt(1)
	large.Lsh(large, 1000)
	if curve.IsOnCurve(large, large) {
		t.Error("(2^1000, 2^1000) is reported on the curve")
	}
}
