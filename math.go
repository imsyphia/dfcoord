package main

func lerp(x, a, b float64) float64 {
	return a + x*(b-a)
}

func lerp2(x, y, a, b, c, d float64) float64 {
	return lerp(y, lerp(x, a, b), lerp(x, c, d))
}

func lerp3(x, y, z, a, b, c, d, e, f, g, h float64) float64 {
	return lerp(z, lerp2(x, y, a, b, c, d), lerp2(x, y, e, f, g, h))
}

func smoothStep(x float64) float64 {
	return x * x * x * (x*(x*6.0-15.0) + 10.0)
}

func rotateLeft(l int64, dist int) int64 {
	return (l << dist) | int64(uint64(l)>>(64-dist))
}

func invLerp(x, a, b float64) float64 {
	return (x - a) / (b - a)
}

func newtonRoot(init float64, iter int, f func(x float64) float64, df func(x float64) float64) float64 {
	x := init
	for i := 0; i < iter; i++ {
		x = x - f(x)/df(x)
	}
	return x
}
