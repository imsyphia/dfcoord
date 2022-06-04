package main

import (
	"math"
)

// A lot of initialization logic is bypassed and/or hardcoded by assuming all noises have
// a first octave of 0 and one octave of amplitude 1. For the purposes of this package,
// there's no need to reimplement the rest. If this file is created into a separate
// package then I suppose it may make sense to do so.

// todo: pass pointers to noises (or at least to perlin's arrays) instead of values,
// copying large arrays is expensive

const (
	octaveStr   = "octave_0"
	secondScale = 1.0181268882175227
	vf          = 5.0 / 6.0
)

type normalNoise struct {
	n1          perlin
	n2          perlin
	valueFactor float64
}

func newNormalNoise(r xoroshiro) normalNoise {
	n1 := newPerlin(r.forkFixed().fromHash(octaveStr))
	n2 := newPerlin(r.forkFixed().fromHash(octaveStr))

	return normalNoise{n1, n2, vf}
}

func (n normalNoise) boundsNoise1(c coord) coordBounds {
	return n.n1.cuboidBounds(wrapCoord(c))
}

func (n normalNoise) boundsNoise2(c coord) coordBounds {
	b := n.n2.cuboidBounds(wrapCoord(scaleCoord(c)))
	return coordBounds{descaleCoord(b.lo), descaleCoord(b.hi)}
}

// cuboidBounds returns the intersection of the bounds of the two noises
func (n normalNoise) cuboidBounds(c coord) (cbr coordBounds) {
	cb1 := n.boundsNoise1(c)
	cb2 := n.boundsNoise2(c)

	cbr.lo.x = math.Max(cb1.lo.x, cb2.lo.x)
	cbr.lo.y = math.Max(cb1.lo.y, cb2.lo.y)
	cbr.lo.z = math.Max(cb1.lo.z, cb2.lo.z)

	cbr.hi.x = math.Min(cb1.hi.x, cb2.hi.x)
	cbr.hi.y = math.Min(cb1.hi.y, cb2.hi.y)
	cbr.hi.z = math.Min(cb1.hi.z, cb2.hi.z)

	return cbr
}

func (n normalNoise) getValue(c coord) float64 {
	v1 := n.n1.noise(wrapCoord(c))
	v2 := n.n2.noise(wrapCoord(scaleCoord(c)))
	return (v1 + v2) * vf
}

func (n normalNoise) getVectors(c coord) ([8]byte, [8]byte) {
	var c1, c2 coord

	c1 = wrapCoord(c)
	c2 = wrapCoord(scaleCoord(c))

	return n.n1.vectors(c1), n.n2.vectors(c2)
}

func (n normalNoise) getNoiseCoords(c coord) (coord, coord) {
	var c1, c2 coord

	c1.x = wrap(c.x + n.n1.o.x)
	c1.y = wrap(c.y + n.n1.o.y)
	c1.z = wrap(c.z + n.n1.o.z)

	c2.x = wrap(c.x + n.n1.o.x*secondScale)
	c2.y = wrap(c.y + n.n1.o.y*secondScale)
	c2.z = wrap(c.z + n.n1.o.z*secondScale)

	return c1, c2
}

func wrapCoord(c coord) coord {
	var r coord

	r.x = wrap(c.x)
	r.y = wrap(c.y)
	r.z = wrap(c.z)

	return r
}

func scaleCoord(c coord) coord {
	var r coord

	r.x = c.x * secondScale
	r.y = c.y * secondScale
	r.z = c.z * secondScale

	return r
}

func descaleCoord(c coord) coord {
	var r coord

	r.x = c.x / secondScale
	r.y = c.y / secondScale
	r.z = c.z / secondScale

	return r
}

type perlin struct {
	p  []byte // precomputed random array used for calculating vectors of a point
	pv []byte
	o  coord // offset
}

func newPerlin(r xoroshiro) (n perlin) {
	n.p, n.pv = make([]byte, 256), make([]byte, 256)

	n.o.x = r.float64() * 256.0
	n.o.y = r.float64() * 256.0
	n.o.z = r.float64() * 256.0

	for i := range n.p {
		n.p[i] = byte(i)
	}

	// fisher-yates shuffle
	for i := range n.p {
		j := int(r.boundedInt32(int32(256 - i)))
		b := n.p[i]
		n.p[i] = n.p[i+j]
		n.p[i+j] = b
	}

	for i, v := range n.p {
		n.pv[i] = gradByte[gradients[v&0xF]]
	}

	return n
}

func compGradients(rn [256]byte) [256]byte {
	var pr [256]byte
	for i, v := range rn {
		pr[i] = gradByte[gradients[v&0xF]]
	}
	return pr
}

func (n perlin) noise(c coord) float64 {
	// assumes zero lfif, lfvf, y

	var oc coord
	oc.x = c.x + n.o.x
	oc.y = c.y + n.o.y
	oc.z = c.z + n.o.z

	var ob intCoord
	ob.x = int64(math.Floor(oc.x))
	ob.y = int64(math.Floor(oc.y))
	ob.z = int64(math.Floor(oc.z))

	var of coord
	of.x = oc.x - float64(ob.x)
	of.y = oc.y - float64(ob.y)
	of.z = oc.z - float64(ob.z)

	// some random numbers
	r := func(i int) int {
		return int(n.p[i&0xFF] & 0xFF)
	}

	xb, yb, zb := int(ob.x), int(ob.y), int(ob.z)

	rx := r(xb)
	rx1 := r(xb + 1)
	rxy := r(rx + yb)
	rx1y := r(rx1 + yb)
	rxy1 := r(rx + yb + 1)
	rx1y1 := r(rx1 + yb + 1)

	xf, yf, zf := of.x, of.y, of.z

	// dot products
	ov000 := gradDot(r(rxy+zb), xf, yf, zf)
	ov100 := gradDot(r(rx1y+zb), xf-1, yf, zf)
	ov010 := gradDot(r(rxy1+zb), xf, yf-1, zf)
	ov110 := gradDot(r(rx1y1+zb), xf-1, yf-1, zf)
	ov001 := gradDot(r(rxy+zb+1), xf, yf, zf-1)
	ov101 := gradDot(r(rx1y+zb+1), xf-1, yf, zf-1)
	ov011 := gradDot(r(rxy1+zb+1), xf, yf-1, zf-1)
	ov111 := gradDot(r(rx1y1+zb+1), xf-1, yf-1, zf-1)

	// smooth
	xfs := smoothStep(xf)
	yfs := smoothStep(yf)
	zfs := smoothStep(zf)

	return lerp3(xfs, yfs, zfs, ov000, ov100, ov010, ov110, ov001, ov101, ov011, ov111)
}

func (n perlin) cuboidBounds(c coord) (b coordBounds) {
	var of coord

	of.x = n.o.x - math.Floor(n.o.x)
	of.y = n.o.y - math.Floor(n.o.y)
	of.z = n.o.z - math.Floor(n.o.z)

	b.lo.x = math.Floor(c.x+of.x) - of.x
	b.lo.y = math.Floor(c.y+of.y) - of.y
	b.lo.z = math.Floor(c.z+of.z) - of.z

	b.hi.x = b.lo.x + 1
	b.hi.y = b.lo.y + 1
	b.hi.z = b.lo.z + 1

	return b
}

func r(p [256]byte, i int) int {
	return int(p[i&0xFF] & 0xFF)
}

func (n perlin) vectors(c coord) [8]byte {
	var oc coord
	var ob intCoord

	oc.x = c.x + n.o.x
	oc.y = c.y + n.o.y
	oc.z = c.z + n.o.z

	ob.x = int64(math.Floor(oc.x))
	ob.y = int64(math.Floor(oc.y))
	ob.z = int64(math.Floor(oc.z))

	r := func(i int) int {
		return int(n.p[i&0xFF] & 0xFF)
	}

	x, y, z := int(ob.x), int(ob.y), int(ob.z)

	rx := r(x)
	rx1 := r(x + 1)
	rxy := r(rx + y)
	rx1y := r(rx1 + y)
	rxy1 := r(rx + y + 1)
	rx1y1 := r(rx1 + y + 1)

	ov000 := n.pv[(rxy+z)&0xFF]
	ov100 := n.pv[(rx1y+z)&0xFF]
	ov010 := n.pv[(rxy1+z)&0xFF]
	ov110 := n.pv[(rx1y1+z)&0xFF]
	ov001 := n.pv[(rxy+z+1)&0xFF]
	ov101 := n.pv[(rx1y+z+1)&0xFF]
	ov011 := n.pv[(rxy1+z+1)&0xFF]
	ov111 := n.pv[(rx1y1+z+1)&0xFF]

	// returns bytes representing vectors for performance
	return [8]byte{ov000, ov100, ov010, ov110, ov001, ov101, ov011, ov111}
}

var gradients = [16][3]int{
	{1, 1, 0}, {-1, 1, 0}, {1, -1, 0}, {-1, -1, 0},
	{1, 0, 1}, {-1, 0, 1}, {1, 0, -1}, {-1, 0, -1},
	{0, 1, 1}, {0, -1, 1}, {0, 1, -1}, {0, -1, -1},
	{1, 1, 0}, {0, -1, 1}, {-1, 1, 0}, {0, -1, -1},
}

// holds a byte value for each unique vector in gradients
var gradByte = map[[3]int]byte{
	{1, 1, 0}:   0,
	{-1, 1, 0}:  1,
	{1, -1, 0}:  2,
	{-1, -1, 0}: 3,
	{1, 0, 1}:   4,
	{-1, 0, 1}:  5,
	{1, 0, -1}:  6,
	{-1, 0, -1}: 7,
	{0, 1, 1}:   8,
	{0, -1, 1}:  9,
	{0, 1, -1}:  10,
	{0, -1, -1}: 11,
}

func gradDot(i int, x, y, z float64) float64 {
	g := gradients[i&0xF]
	var h [3]float64
	for i, v := range g {
		h[i] = float64(v)
	}
	return dot([3]float64{x, y, z}, h)
}

func dot(f1 [3]float64, f2 [3]float64) float64 {
	var k float64
	for i := range f1 {
		k += f1[i] * f2[i]
	}
	return k
}

func wrap(x float64) float64 {
	// if x > 1/2 of 3.3554432e7 then wrap around
	return x - math.Floor(x/3.3554432e7+0.5)*3.3554432e7
}
