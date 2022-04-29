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
    heightmapPath string
    readStockPath string
    writeStockPath string

    safeZ float64
    rapidFeed float64
    xyFeed float64
    zFeed float64
    rpm float64

    width float64
    height float64
    depth float64

    direction Direction

    stepOver float64
    stepDown float64

    tool Tool

    stockToLeave float64

    roughingOnly bool
    omitTop bool
    rampEntry bool
    cutBelowBottom bool
    cutBeyondEdges bool

    imperial bool

    xOffset float64
    yOffset float64
    zOffset float64

    maxVel float64
    maxAccel float64

    quiet bool
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
