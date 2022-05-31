package main

import (
	"dfcoord/internal/channels"
	"math"
	"runtime"
	"strconv"
)

// Keeping generated data in one namespace improves compatibility.
const namespace = "syph"

type noiseInfo struct {
	dimSeed int64
	rl      string
}

type noiseLocInfo struct {
	dimSeed int64
	rl      string
	axis    axis
	b1, b2  coordBounds
	y       float64
}

type dfParams struct {
	dimSeed int64
	rl      string
	axis    axis
	x, y, z float64
	m, b    float64
}

type axis int

const (
	axisX = iota
	axisZ
)

// The function rd acts as a reduce callback over the infinite, generated stream of parameter lists.
// genFromDimSeed will terminate when rd returns false.
func genFromDimSeed[T any](dimSeed int64, rd func(a T, first bool, d dfParams) (accum T, cont bool)) T {
	cpu := runtime.NumCPU()

	nStop := make(chan struct{})
	nOut := make(chan noiseInfo, cpu)

	workerIn := nOut
	workerOuts := make([]chan dfParams, cpu)
	for i := range workerOuts {
		workerOuts[i] = make(chan dfParams, 2)
	}

	params := make(chan dfParams, cpu)

	channels.MergeS(params, workerOuts)

	// noiseInfo producer
	go func() {
		defer close(nOut)
		for i := int64(0); ; i++ {
			select {
			case _, ok := <-nStop:
				if !ok {
					return
				}
			case nOut <- noiseInfo{dimSeed, namespace + ":" + strconv.FormatInt(i, 36)}:
			}
		}
	}()

	for _, c := range workerOuts {
		in := workerIn
		out := c
		go func() {
			defer close(out)
			for p := range in {
				for _, v := range genFromNoiseInfo(p) {
					out <- v
				}
			}
		}()
	}

	var accum T
	first := true
	for p := range params {
		var cont bool
		accum, cont = rd(accum, first, p)
		first = false
		if !cont {
			close(nStop)
			for range params {
			}
		}
	}

	return accum
}

const (
	searchMin = -128.0
	searchMax = 128.0
)

func genFromNoiseInfo(d noiseInfo) (p []dfParams) {
	xh := newXoroshiro(upgradeSeedTo128Bit(d.dimSeed)).forkFixed().fromHash(d.rl)
	nn := newNormalNoise(xh)
	p = make([]dfParams, 0)
	for x := searchMin; x <= searchMax; x++ {
		for y := searchMin; y <= searchMax; y++ {
			for z := searchMin; z <= searchMax; z++ {
				// todo: clean up and optimize
				c := coord{x, y, z}
				b1 := nn.boundsNoise1(c)
				b2 := nn.boundsNoise2(c)
				l1 := b1.lo[1] > b2.lo[1]
				l2 := b1.hi[1] > b2.hi[1]
				v1, v2 := nn.getVectors(c)
				var vux, vlx, vuz, vlz bool
				// todo: maybe change to always calculate for both cases and check
				// the bounds later to make branch prediction more reliable
				if l1 && l2 {
					vux, vlx, vuz, vlz = is12AlignedVectorSet(v1, v2)
				} else if !l1 && !l2 {
					vux, vlx, vuz, vlz = is12AlignedVectorSet(v2, v1)
				} else {
					// case where the bounds of the second fit within the bounds of the first;
					// it *could* be checked but it's fairly rare and just extra work to implement
					continue
				}
				if vux || vlx || vuz || vlz {
					var axis axis
					if vux || vlx {
						axis = axisX
					} else {
						axis = axisZ
					}
					var yr float64
					if l1 && l2 {
						if vux || vuz {
							yr = b2.hi[1]
						} else {
							yr = b1.lo[1]
						}
					} else {
						if vux || vuz {
							yr = b1.hi[1]
						} else {
							yr = b2.lo[1]
						}
					}
					params := genFromNoiseLoc(noiseLocInfo{d.dimSeed, d.rl, axis, b1, b2, yr})
					// it is arguably a bug if the parameters result in NaN or
					// Inf but the easiest solution is to ignore them for now
					valid := validateParams(params)
					if valid {
						p = append(p, params)
					}
				}
			}
		}
	}
	return p
}

func isAlignedVectorSet(s1 [8]byte, s2 [8]byte) (x bool, z bool) {

	si := uint64(s1[0])<<60 | uint64(s1[1])<<56 | uint64(s1[2])<<52 | uint64(s1[3])<<48 |
		uint64(s1[4])<<44 | uint64(s1[5])<<40 | uint64(s1[6])<<36 | uint64(s1[7])<<32 |
		uint64(s2[0])<<28 | uint64(s2[1])<<24 | uint64(s2[2])<<20 | uint64(s2[3])<<16 |
		uint64(s2[4])<<12 | uint64(s2[5])<<8 | uint64(s2[6])<<4 | uint64(s2[7])

	// vector order is xyz 000, 100, 010, 110, 001, 101, 011, 111

	// a valid vector set is one where all vectors are on the t-y plane, where t can be x or z
	// the vectors in a pair along r axis must be exactly equivalent, where r is the axis that t is not (so x if z, etc)

	// z direction check, all 0s if all checks pass
	isz := si&0xCCCCCCCCCCCCCCCC ^ 0x8888888888888888

	// x direction check, all 0s if all checks pass
	isx := si & 0xCCCCCCCCCCCCCCCC

	// z pair checks, all 0s if all checks pass
	pz := (si>>16 ^ si) & 0x0000FFFF0000FFFF

	// x pair checks, all 0s if all checks pass
	px := (si>>4 ^ si) & 0x0F0F0F0F0F0F0F0F

	// 0 if valid vector set on xy plane
	xs := isx | pz

	// 0 if valid vector set on zy plane
	zs := isz | px

	return xs == 0, zs == 0
}

func is12AlignedVectorSet(s1 [8]byte, s2 [8]byte) (ux bool, lx bool, uz bool, lz bool) {

	// this is a 12 vector check for the case where y of s1 > y of s2
	// note to self: do not forget the case where s2 is contained within s1

	si := uint64(s1[0])<<60 | uint64(s1[1])<<56 | uint64(s1[2])<<52 | uint64(s1[3])<<48 |
		uint64(s1[4])<<44 | uint64(s1[5])<<40 | uint64(s1[6])<<36 | uint64(s1[7])<<32 |
		uint64(s2[0])<<28 | uint64(s2[1])<<24 | uint64(s2[2])<<20 | uint64(s2[3])<<16 |
		uint64(s2[4])<<12 | uint64(s2[5])<<8 | uint64(s2[6])<<4 | uint64(s2[7])

	// vector order is xyz 000, 100, 010, 110, 001, 101, 011, 111

	// a valid vector set is one where all vectors are on the t-y plane, where t can be x or z
	// the vectors in a pair along r axis must be exactly equivalent, where r is the axis that t is not (so x if z, etc)

	// i guess i'll check for the first + upper plane of second, then second + lower plane of first? idk
	// performance doesn't matter *that* much because current time is ~10ns and obtaining vectors takes ~80ns

	// z direction check, all 0s if all checks pass
	isz := si&0xCCCCCCCCCCCCCCCC ^ 0x8888888888888888

	iszUpper := isz & 0xFFFFFFFF00FF00FF // ignore the lower vectors of the second set
	iszLower := isz & 0xFF00FF00FFFFFFFF // ignore the upper vectors of the first set

	// x direction check, all 0s if all checks pass
	isx := si & 0xCCCCCCCCCCCCCCCC

	isxUpper := isx & 0xFFFFFFFF00FF00FF // ignore the lower vectors of the second set
	isxLower := isx & 0xFF00FF00FFFFFFFF // ignore the upper vectors of the first set

	// z pair checks, all 0s if all checks pass
	pz := (si>>16 ^ si) // check that vectors match and do not mask away garbage

	pzUpper := pz & 0x0000FFFF000000FF // mask away garbage and lower vectors of second set
	pzLower := pz & 0x0000FF000000FFFF // mask away garbage and upper vectors of first set

	// x pair checks, all 0s if all checks pass
	px := (si>>4 ^ si) & 0x0F0F0F0F0F0F0F0F // XOR the bits that should match and mask the garbage

	pxUpper := px & 0x0F0F0F0F000F000F // mask away garbage and lower vectors of second set
	pxLower := px & 0x0F000F000F0F0F0F // mask away garbage and upper vectors of first set

	// 0 if valid vector set on xy plane

	xsUpper := isxUpper | pzUpper
	xsLower := isxLower | pzLower

	// 0 if valid vector set on zy plane

	zsUpper := iszUpper | pxUpper
	zsLower := iszLower | pxLower

	return xsUpper == 0, xsLower == 0, zsUpper == 0, zsLower == 0
}

func isNumber(x float64) bool {
	return !(math.IsInf(x, 0) || math.IsNaN(x))
}

func validateParams(p dfParams) bool {
	return isNumber(p.m) && isNumber(p.b) && isNumber(p.x) && isNumber(p.y) && isNumber(p.z)
}

func genFromNoiseLoc(res noiseLocInfo) dfParams {
	// this whole funcion likely needs to be refactored, I wrote it once and haven't touched it since
	derivative := func(f func(float64) float64, d float64) func(float64) float64 {
		return func(x float64) float64 {
			return (f(x+d) - f(x-d)) / (2 * d)
		}
	}

	nn := newNormalNoise(newXoroshiro(upgradeSeedTo128Bit(res.dimSeed)).forkFixed().fromHash(res.rl))

	var px, py, pz float64
	py = res.y

	var domain = struct {
		min float64
		max float64
	}{}
	var noiseGetter func(float64) float64
	if res.axis == axisX {
		zMid := (math.Max(res.b1.lo[2], res.b2.lo[2]) + math.Min(res.b1.hi[2], res.b2.hi[2])) / 2
		pz = zMid
		xMin := math.Max(res.b1.lo[0], res.b2.lo[0])
		xMax := math.Min(res.b1.hi[0], res.b2.hi[0])
		noiseGetter = func(x float64) float64 {
			return nn.getValue(coord{x + xMin, res.y, zMid})
		}
		domain.min, domain.max = 0, xMax-xMin
	}

	if res.axis == axisZ {
		xMid := (math.Max(res.b1.lo[0], res.b2.lo[0]) + math.Min(res.b1.hi[0], res.b2.hi[0])) / 2
		px = xMid
		zMin := math.Max(res.b1.lo[2], res.b2.lo[2])
		zMax := math.Min(res.b1.hi[2], res.b2.hi[2])
		noiseGetter = func(x float64) float64 {
			return nn.getValue(coord{xMid, res.y, x + zMin})
		}
		domain.min, domain.max = 0, zMax-zMin
	}

	dNoiseGetter := derivative(noiseGetter, 0.00000001)

	// generates a linear appoximation of the inverse at x
	lineAtX := func(x float64) func(float64) float64 {
		d := 1 / dNoiseGetter(x)
		o := noiseGetter(x)
		return func(y float64) float64 {
			return d*(y-o) + y
		}
	}

	// we want to find the minimum of this function, as it represents the error
	// value outside of domain results in undefined behavior
	functionToBeOptimizedFor := func(x float64) float64 {
		l := lineAtX(x)

		e := func(p float64) float64 {
			// approximate the error of the inverse function using only the regular one
			return l(noiseGetter(x+p)) - (x + p)
		}

		points := []float64{
			e(0.001),  // 1 million blocks
			e(0.0001), // 100k, etc
			e(0.00001),
			e(-0.00001),
			e(-0.0001),
			e(-0.001),
		}

		var t float64 = 0
		for _, v := range points {
			t += v
		}
		return math.Abs(t)
	}

	// approximate a minimum point
	increment := domain.max / 101
	least := math.MaxFloat64
	for i := 0; i < 100; i++ {
		x := float64(i+1) * increment
		if functionToBeOptimizedFor(x) < least {
			least = x
		}
	}

	ddNoiseGetter := derivative(dNoiseGetter, 0.00000001)

	// find zero of derivative via newton's method, which should be the minimum
	// terminate early if x is not in domain
	pt := least
	for i := 0; i < 100; i++ {
		k := pt - dNoiseGetter(pt)/ddNoiseGetter(pt)
		if k < (domain.min+0.001) || pt > (domain.max-0.001) {
			break
		}
		pt = k
	}

	slope := 1 / dNoiseGetter(pt)
	offset := (1 / dNoiseGetter(pt)) * (-noiseGetter(pt))

	if res.axis == axisX {
		px = math.Max(res.b1.lo[0], res.b2.lo[0]) + pt
	}

	if res.axis == axisZ {
		pz = math.Max(res.b1.lo[2], res.b2.lo[2]) + pt
	}

	return dfParams{res.dimSeed, res.rl, res.axis, px, py, pz, slope, offset}
}
