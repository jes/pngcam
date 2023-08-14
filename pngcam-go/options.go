package main

import (
	"math"
)

type Direction int

const (
	Horizontal Direction = iota
	Vertical
)

type Options struct {
	heightmapPath  string
	readStockPath  string
	writeStockPath string
	rgb            bool

	safeZ     float64
	rapidFeed float64
	xyFeed    float64
	zFeed     float64
	rpm       float64

	width  float64
	height float64
	depth  float64
	rotary bool

	direction Direction

	stepOver float64
	stepDown float64

	tool Tool

	stockToLeave float64

	roughingOnly   bool
	omitTop        bool
	rampEntry      bool
	cutBelowBottom bool
	cutBeyondEdges bool

	imperial bool

	xOffset float64
	yOffset float64
	zOffset float64

	maxVel   float64
	maxAccel float64

	quiet bool

	x_MmPerPx float64
	y_MmPerPx float64
	widthPx   int
	heightPx  int
}

func (opt Options) FeedRate(start Toolpoint, end Toolpoint) float64 {
	dx := end.x - start.x
	dy := end.y - start.y
	dz := end.z - start.z

	xyDist := math.Sqrt(dx*dx + dy*dy)
	zDist := dz

	if opt.rotary {
		y1 := start.z * math.Sin(start.y*math.Pi/180.0)
		y2 := end.z * math.Sin(end.y*math.Pi/180.0)
		dy = y2 - y1
		xyDist = math.Sqrt(dx*dx + dy*dy)
		// TODO: this calculates the straight-line distance between the 2 points, but
		// actually the movement follows an arc (which may be combined with X and Z
		// moves) - ideally we would calculate the true arc length instead of the
		// straight-line length
	}

	totalDist := math.Sqrt(xyDist*xyDist + zDist*zDist)

	epsilon := 0.00001

	// rapid feed on vertical upwards movement with no XY component
	unitsPerMin := opt.rapidFeed
	if xyDist >= epsilon || zDist < 0 {
		if zDist >= 0 || math.Abs(xyDist/zDist) > math.Abs(opt.xyFeed/opt.zFeed) {
			// XY feed is limiting factor
			unitsPerMin = opt.xyFeed
		} else {
			unitsPerMin = opt.zFeed
		}
	}

	if opt.rotary {
		// in rotary mode we use "inverse time" feed rates
		if totalDist < epsilon {
			return opt.rapidFeed // XXX: what should we do here? probably doesn't matter given that distance = 0
		}
		movesPerMin := unitsPerMin / totalDist
		return movesPerMin
	} else {
		return unitsPerMin
	}
}

func (opt *Options) MmToPx(x, y float64) (int, int) {
	return int(x / opt.x_MmPerPx), int(-y/opt.y_MmPerPx) + opt.heightPx - 1
}

func (opt Options) PxToMm(x, y int) (float64, float64) {
	return float64(x) * opt.x_MmPerPx, float64(opt.heightPx-1-y) * opt.y_MmPerPx
}
