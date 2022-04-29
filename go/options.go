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
    safeZ float64
    rapidFeed float64
    xyFeed float64
    zFeed float64

    width float64
    height float64
    depth float64

    direction Direction

    stepOver float64
    stepForward float64
    stepDown float64

    tool Tool

    stockToLeave float64

    omitTop bool
    rampEntry bool
    cutBelowBottom bool
    cutBeyondEdges bool
}

func (opt Options) FeedRate(start Toolpoint, end Toolpoint) float64 {
    dx := end.x - start.x
    dy := end.y - start.y
    dz := end.z - start.z

    xyDist := math.Sqrt(dx*dx + dy*dy)
    zDist := dz
    totalDist := math.Sqrt(xyDist*xyDist + zDist*zDist)

    epsilon := 0.00001

    // vertical upwards movement with no XY component: rapid feed
    if xyDist < epsilon && zDist > 0 {
        return opt.rapidFeed
    }

    if zDist >= 0 || math.Abs(xyDist/zDist) > math.Abs(opt.xyFeed/opt.zFeed) {
        // XY feed is limiting factor
        return opt.xyFeed
    } else {
        // Z feed is limiting factor
        return math.Abs(totalDist/zDist) * opt.zFeed
    }
}

func (opt Options) XStep() float64 {
    if opt.direction == Horizontal {
        return opt.stepForward
    } else {
        return opt.stepOver
    }
}

func (opt Options) YStep() float64 {
    if opt.direction == Horizontal {
        return opt.stepOver
    } else {
        return opt.stepForward
    }
}
