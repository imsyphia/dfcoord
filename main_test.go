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

func BenchmarkGetVectorsInlined(b *testing.B) {
	x := newXoroshiro(0, 0)
	nn := newNormalNoise(x)
	var c = coord{123, 123, 123}
	for i := 0; i < b.N; i++ {
		var c1, c2 coord
		for i := range c {
			c1[i] = wrap(c[i])
			c2[i] = wrap(c[i] * secondScale)
		}
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
