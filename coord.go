package main

type coord struct {
	x, y, z float64
}

type intCoord struct {
	x, y, z int64
}

type coordBounds struct {
	lo coord
	hi coord
}
