package schnorr

// https://medium.com/coinmonks/schnorr-signatures-in-go-80a7fbfe0fe4

import (
	"gopkg.in/dedis/kyber.v2"
)

type Group interface {
	G() kyber.Point
	RandomScalar() kyber.Scalar
	Mul(kyber.Scalar, kyber.Point) kyber.Point
	PointSub(a, b kyber.Point) kyber.Point
	ScalarSub(a, b kyber.Scalar) kyber.Scalar
	ScalarMul(a, b kyber.Scalar) kyber.Scalar
	Inv(scalar kyber.Scalar) kyber.Scalar
}

type hash func(string, kyber.Point) kyber.Scalar

// m: Message
// x: Private key
func Sign(group Group, m string, x kyber.Scalar, h hash) (kyber.Point, kyber.Scalar) {

	var g = group.G()

	// Pick a random k from allowed set.
	k := group.RandomScalar()

	// r = k * G (a.k.a the same operation as r = g^k)
	r := group.Mul(k, g)

	// Hash(m || r)
	e := h(m, r)

	// s = k - e * x
	s := group.ScalarSub(k, group.ScalarMul(e, x))

	return r, s
}

// m: Message
// S: Signature
func PublicKey(group Group, m string, r kyber.Point, s kyber.Scalar, h hash) kyber.Point {

	var g = group.G()

	// e = Hash(m || r)
	e := h(m, r)

	// y = (r - s * G) * (1 / e)
	y := group.PointSub(r, group.Mul(s, g))
	y = group.Mul(group.Inv(e), y)

	return y
}

// m: Message
// s: Signature
// y: Public key
func Verify(group Group, m string, r kyber.Point, s kyber.Scalar, y kyber.Point, h func(string, kyber.Point) kyber.Scalar) bool {

	var g = group.G()

	// e = Hash(m || r)
	e := h(m, r)

	// Attempt to reconstruct 's * G' with a provided signature; s * G = r - e * y
	sGv := group.PointSub(r, group.Mul(e, y))

	// Construct the actual 's * G'
	sG := group.Mul(s, g)

	// Equality check; ensure signature and public key outputs to s * G.
	return sG.Equal(sGv)
}
