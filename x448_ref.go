// The MIT License (MIT)
//
// Copyright (c) 2011 Stanford University.
// Copyright (c) 2014-2015 Cryptography Research, Inc.
// Copyright (c) 2015 Yawning Angel.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package x448

// This should really use 64 bit limbs, but Go is fucking retarded and doesn't
// have __(u)int128_t, so the 32 bit code it is, at a hefty performance
// penalty.  Fuck my life, I'm going to have to bust out PeachPy to get this
// to go fast aren't I.
//
// This is equivalent to the non-unrolled reference code, though the compiler
// is free to unroll as it sees fit.  If performance is horrendous I'll
// manually unroll things.

const (
	wBits     = 32
	lBits     = (wBits * 7 / 8)
	x448Limbs = (448 / lBits)
	lMask     = (1 << lBits) - 1
)

type limbUint uint32
type limbSint int32

type gf struct {
	limb [x448Limbs]uint32
}

var zero = gf{[x448Limbs]uint32{0}}
var one = gf{[x448Limbs]uint32{1}}
var p = gf{[x448Limbs]uint32{
	lMask, lMask, lMask, lMask, lMask, lMask, lMask, lMask,
	lMask - 1, lMask, lMask, lMask, lMask, lMask, lMask, lMask,
}}

// cpy copies x = y.
func (x *gf) cpy(y *gf) {
	for i, v := range y.limb { // XXX: Unroll
		x.limb[i] = v
	}
}

// mul multiplies c = a * b. (PERF)
func (c *gf) mul(a, b *gf) {
	var aa gf
	aa.cpy(a)

	var accum [x448Limbs]uint64
	for i, bv := range b.limb { // XXX: Unroll
		for j, aav := range aa.limb { // XXX: Unroll
			accum[(i+j)%x448Limbs] += (uint64)(bv) * (uint64)(aav)
		}
		aa.limb[(x448Limbs-1-i)^(x448Limbs/2)] += aa.limb[x448Limbs-1-i]
	}

	accum[x448Limbs-1] += accum[x448Limbs-2] >> lBits
	accum[x448Limbs-2] &= lMask
	accum[x448Limbs/2] += accum[x448Limbs-1] >> lBits
	for j := uint(0); j < x448Limbs; j++ { // XXX: Unroll
		accum[j] += accum[(j-1)%x448Limbs] >> lBits
		accum[(j-1)%x448Limbs] &= lMask
	}
	for j, accv := range accum { // XXX: Unroll
		c.limb[j] = (uint32)(accv)
	}
}

// sqr squares (c = x * x).  Just calls multiply. (PERF)
func (c *gf) sqr(x *gf) {
	c.mul(x, x)
}

// isqrt inverse square roots (y = 1/sqrt(x)), using an addition chain.
func (y *gf) isqrt(x *gf) {
	var a, b, c gf
	c.sqr(x)

	// XXX/Yawning, could unroll, but this is called only once.

	// STEP(b,x,1);
	b.mul(x, &c)
	c.cpy(&b)
	for i := 0; i < 1; i++ {
		c.sqr(&c)
	}

	// STEP(b,x,3);
	b.mul(x, &c)
	c.cpy(&b)
	for i := 0; i < 3; i++ {
		c.sqr(&c)
	}

	//STEP(a,b,3);
	a.mul(&b, &c)
	c.cpy(&a)
	for i := 0; i < 3; i++ {
		c.sqr(&c)
	}

	// STEP(a,b,9);
	a.mul(&b, &c)
	c.cpy(&a)
	for i := 0; i < 9; i++ {
		c.sqr(&c)
	}

	// STEP(b,a,1);
	b.mul(&a, &c)
	c.cpy(&b)
	for i := 0; i < 1; i++ {
		c.sqr(&c)
	}

	// STEP(a,x,18);
	a.mul(x, &c)
	c.cpy(&a)
	for i := 0; i < 18; i++ {
		c.sqr(&c)
	}

	// STEP(a,b,37);
	a.mul(&b, &c)
	c.cpy(&a)
	for i := 0; i < 37; i++ {
		c.sqr(&c)
	}

	// STEP(b,a,37);
	b.mul(&a, &c)
	c.cpy(&b)
	for i := 0; i < 37; i++ {
		c.sqr(&c)
	}

	// STEP(b,a,111);
	b.mul(&a, &c)
	c.cpy(&b)
	for i := 0; i < 111; i++ {
		c.sqr(&c)
	}

	// STEP(a,b,1);
	a.mul(&b, &c)
	c.cpy(&a)
	for i := 0; i < 1; i++ {
		c.sqr(&c)
	}

	// STEP(b,x,223);
	b.mul(x, &c)
	c.cpy(&b)
	for i := 0; i < 223; i++ {
		c.sqr(&c)
	}

	y.mul(&a, &c)
}

// inv inverses (y = 1/x).
func (y *gf) inv(x *gf) {
	var z, w gf
	z.sqr(x)     // x^2
	w.isqrt(&z)  // +- 1/sqrt(x^2) = +- 1/x
	z.sqr(&w)    // 1/x^2
	w.mul(x, &z) // 1/x
	y.cpy(&w)
}

// reduce weakly reduces mod p
func (x *gf) reduce() {
	x.limb[x448Limbs/2] += x.limb[x448Limbs-1] >> lBits
	for j := uint(0); j < x448Limbs; j++ { // XXX: Unroll
		x.limb[j] += x.limb[(j-1)%x448Limbs] >> lBits
		x.limb[(j-1)%x448Limbs] &= lMask
	}
}

// add adds mod p. Conservatively always weak-reduces. (PERF)
func (x *gf) add(y, z *gf) {
	for i, yv := range y.limb { // XXX: Unroll
		x.limb[i] = yv + z.limb[i]
	}
	x.reduce()
}

// sub subtracts mod p.  Conservatively always weak-reduces. (PERF)
func (x *gf) sub(y, z *gf) {
	for i, yv := range y.limb { // XXX: Unroll
		x.limb[i] = yv - z.limb[i] + 2*p.limb[i]
	}
	x.reduce()
}

// condSwap swaps x and y in constant time.
func (x *gf) condSwap(y *gf, swap limbUint) {
	for i, xv := range x.limb { // XXX: Unroll
		s := (xv ^ y.limb[i]) & (uint32)(swap) // Sort of dumb, oh well.
		x.limb[i] ^= s
		y.limb[i] ^= s
	}
}

// mlw multiplies by a signed int.  NOT CONSTANT TIME wrt the sign of the int,
// but that's ok because it's only ever called with w = -edwardsD.  Just uses
// a full multiply. (PERF)
func (a *gf) mlw(b *gf, w int) {
	if w > 0 {
		ww := gf{[x448Limbs]uint32{(uint32)(w)}}
		a.mul(b, &ww)
	} else {
		// This branch is *NEVER* taken with the current code.
		panic("mul called with negative w")
		ww := gf{[x448Limbs]uint32{(uint32)(-w)}}
		a.mul(b, &ww)
		a.sub(&zero, a)
	}
}

// canon canonicalizes.
func (a *gf) canon() {
	a.reduce()

	// Subtract p with borrow.
	var carry int64
	for i, v := range a.limb {
		carry = carry + (int64)(v) - (int64)(p.limb[i])
		a.limb[i] = (uint32)(carry & lMask)
		carry >>= lBits
	}

	addback := carry
	carry = 0

	// Add it back.
	for i, v := range a.limb {
		carry = carry + (int64)(v) + (int64)(p.limb[i]&(uint32)(addback))
		a.limb[i] = uint32(carry & lMask)
		carry >>= lBits
	}
}

// deser deserializes into the limb representation.
func (s *gf) deser(ser *[x448Bytes]byte) int64 {
	var buf uint64
	bits := uint(0)
	k := 0

	for i, v := range ser {
		buf |= (uint64)(v) << bits
		for bits += 8; (bits >= lBits || i == x448Bytes-1) && k < x448Limbs; bits, buf = bits-lBits, buf>>lBits {
			s.limb[k] = (uint32)(buf & lMask)
			k++
		}
	}

	// XXX: Return value never used, this can be omitted.
	var accum int64
	for i, v := range s.limb {
		accum = (accum + (int64)(v) - (int64)(p.limb[i])) >> wBits
	}
	return accum
}

// ser serializes into byte representation.
func (a *gf) ser(ser *[x448Bytes]byte) {
	a.canon()
	k := 0
	bits := uint(0)
	var buf uint64
	for i, v := range a.limb {
		buf |= (uint64)(v) << bits
		for bits += lBits; (bits >= 8 || i == x448Limbs-1) && k < x448Bytes; bits, buf = bits-8, buf>>8 {
			ser[k] = (byte)(buf)
			k++
		}
	}
}
