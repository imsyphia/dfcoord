package main

import (
	"crypto/md5"
	"encoding/binary"
)

type xoroshiro struct {
	*xoroshiroRandom
}

func newXoroshiro(lo, hi int64) xoroshiro {
	return xoroshiro{newXoroshiroRandom(lo, hi)}
}

func (r xoroshiro) forkFixed() xoroshiroFixedFactory {
	return xoroshiroFixedFactory{r.next(), r.next()}
}

func (x xoroshiro) bits(i int) uint64 {
	return uint64(x.next()) >> (64 - i)
}

func (r xoroshiro) float64() float64 {
	return float64(r.bits(53)) * 1.1102230246251565e-16
}

func (r xoroshiro) int32() int32 {
	return int32(r.next())
}

func (r xoroshiro) boundedInt32(i int32) int32 {
	if i <= 0 {
		return 0
	}

	// u/int64(uint32(x)) is to convert s to u/int64 while filling with zeroes instead of sign bit

	l := int64(uint32(r.int32()))
	m := l * int64(uint32(i))
	n := m & 0xFFFFFFFF

	if n < int64(i) {
		j := int64(uint32(^i+1)) % int64(uint32(i))
		for n < int64(j) {
			l = int64(uint32(r.int32()))
			m = l * int64(i)
			n = m & 0xFFFFFFFF
		}
	}
	o := m >> 32
	return int32(o)
}

func upgradeSeedTo128Bit(l int64) (lo, hi int64) {
	m := l ^ 0x6A09E667F3BCC909
	n := m + -7046029254386353131
	return mixStafford13(m), mixStafford13(n)
}

func mixStafford13(l int64) int64 {
	l = (int64(uint64(l)>>30) ^ l) * -4658895280553007687
	l = (int64(uint64(l)>>27) ^ l) * -7723592293110705685
	return int64(uint64(l)>>31) ^ l
}

type xoroshiroFixedFactory struct {
	lo, hi int64
}

func (r xoroshiroFixedFactory) fromHash(s string) xoroshiro {
	b := md5.Sum([]byte(s))
	lo := int64(binary.BigEndian.Uint64(b[0:]))
	hi := int64(binary.BigEndian.Uint64(b[8:]))
	return newXoroshiro(lo^r.lo, hi^r.hi)
}

type xoroshiroRandom struct {
	lo, hi int64
}

func newXoroshiroRandom(lo, hi int64) *xoroshiroRandom {
	r := new(xoroshiroRandom)
	r.lo = lo
	r.hi = hi
	if (r.lo | r.hi) == 0 {
		r.lo = -7046029254386353131
		r.hi = 7640891576956012809
	}
	return r
}

func (r *xoroshiroRandom) next() int64 {
	lo := r.lo
	hi := r.hi
	res := rotateLeft(lo+hi, 17) + lo
	hi ^= lo
	r.lo = rotateLeft(lo, 49) ^ hi ^ (hi << 21)
	r.hi = rotateLeft(hi, 28)
	return res
}
