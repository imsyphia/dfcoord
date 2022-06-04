package main

import (
	"testing"
)

func BenchmarkNewNoise(b *testing.B) {
	x := newXoroshiro(0, 0)
	for i := 0; i < b.N; i++ {
		n := newNormalNoise(x)
		_ = n
	}
}

func BenchmarkGetVectors(b *testing.B) {
	x := newXoroshiro(0, 0)
	nn := newNormalNoise(x)
	for i := 0; i < b.N; i++ {
		nn.getVectors(coord{123, 123, 123})
	}
}

func BenchmarkPerlinVectorsInt(b *testing.B) {
	x := newXoroshiro(0, 0)
	p := newNormalNoise(x).n1
	for i := 0; i < b.N; i++ {
		p.vectorsInt(coord{123, 123, 123})
	}
}

func BenchmarkGetVectorsInlined(b *testing.B) {
	x := newXoroshiro(0, 0)
	nn := newNormalNoise(x)
	var c = coord{123, 123, 123}
	for i := 0; i < b.N; i++ {
		var c1, c2 coord
		c1 = wrapCoord(c)
		c2 = wrapCoord(scaleCoord(c))
		v1 := nn.n1.vectors(c1)
		v2 := nn.n2.vectors(c2)
		_, _ = v1, v2
	}
}

func BenchmarkIsAlignedVectorSet(b *testing.B) {
	s1 := [8]byte{1, 3, 4, 7, 2, 11, 11, 3}
	s2 := [8]byte{8, 3, 1, 7, 2, 11, 11, 3}
	for i := 0; i < b.N; i++ {
		_, _ = isAlignedVectorSet(s1, s2)
	}
}

func BenchmarkBounds(b *testing.B) {
	x := newXoroshiro(0, 0)
	nn := newNormalNoise(x)
	c := coord{123, 123, 123}
	for i := 0; i < b.N; i++ {
		_ = nn.boundsNoise1(c)
	}
}

func BenchmarkFromDimSeed(b *testing.B) {
	dimSeed := 0

	reduce := func(a twoParams, first bool, d dfParams) (twoParams, bool) {
		if d.axis == axisX {
			if !a.okx {
				a.x = d
				a.okx = true
			}
		} else {
			if !a.okz {
				a.z = d
				a.okz = true
			}
		}
		cont := !(a.okx && a.okz)
		return a, cont
	}

	_ = genFromDimSeed(int64(dimSeed), reduce)
}
