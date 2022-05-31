package main

import "testing"

func BenchmarkNext(b *testing.B) {
	x := newXoroshiroRandom(0, 0)
	for i := 0; i < b.N; i++ {
		_ = x.next()
	}
}

func BenchmarkFloat64(b *testing.B) {
	x := newXoroshiro(0, 0)
	for i := 0; i < b.N; i++ {
		_ = x.float64()
	}
}

func BenchmarkBoundedInt32(b *testing.B) {
	x := newXoroshiro(0, 0)
	for i := 0; i < b.N; i++ {
		_ = x.boundedInt32(139842934)
	}
}

func BenchmarkFromHash(b *testing.B) {
	x := newXoroshiro(0, 0).forkFixed()
	s := "testing string"
	for i := 0; i < b.N; i++ {
		_ = x.fromHash(s)
	}
}
