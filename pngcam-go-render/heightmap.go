package main

import (
	"image"
	"image/png"
	"math"
	"os"
)

type Heightmap struct {
	height  []float32
	options *Options
}

func NewHeightmap(opt *Options) *Heightmap {
	hm := Heightmap{}
	hm.height = make([]float32, opt.width*opt.height)
	hm.options = opt

	// initialise to minimum height
	for y := 0; y < opt.height; y++ {
		for x := 0; x < opt.width; x++ {
			n := y*opt.width + x
			hm.height[n] = 0
		}
	}

	return &hm
}

func (hm *Heightmap) WritePNG(path string) error {
	opt := hm.options

	img := image.NewRGBA(image.Rect(0, 0, opt.width, opt.height))

	for y := 0; y < opt.height; y++ {
		for x := 0; x < opt.width; x++ {
			n := y*opt.width + x

			z := hm.height[n]
			if z > 1 {
				z = 1
			}
			if z < 0 {
				z = 0
			}
			brightness := int(16777215 * z)

			img.Pix[n*4] = uint8(brightness >> 16)
			img.Pix[n*4+1] = uint8((brightness >> 8) & 0xff)
			img.Pix[n*4+2] = uint8(brightness & 0xff)
			img.Pix[n*4+3] = 255
		}
	}

	out, err := os.Create(path)
	if err != nil {
		return err
	}
	png.Encode(out, img)
	out.Close()
	return err
}

// X,Y should be in pixels
// Z should range from 0..1
func (hm *Heightmap) DrawTriangle(a, b, c [3]float32) {
	// min/max X position for each Y position
	leftX := make(map[int]int)
	rightX := make(map[int]int)
	// Z coordinate for corresponding leftX/rightX
	leftZ := make(map[int]float32)
	rightZ := make(map[int]float32)

	// min/max Y position
	minY := hm.options.height
	maxY := -1

	// 1. work out where the outline of the triangle is
	perimeterCb := func(x, y int, z float32) {
		cur, got := leftX[y]
		if !got || x < cur {
			leftX[y] = x
			leftZ[y] = z
		}
		cur, got = rightX[y]
		if !got || x > cur {
			rightX[y] = x
			rightZ[y] = z
		}
		if y < minY {
			minY = y
		}
		if y > maxY {
			maxY = y
		}
	}
	hm.IterateLine(a, b, perimeterCb)
	hm.IterateLine(b, c, perimeterCb)
	hm.IterateLine(c, a, perimeterCb)

	// 2. fill in scanlines
	for y := minY; y <= maxY; y++ {
		startX := leftX[y]
		endX := rightX[y]
		startZ := leftZ[y]
		endZ := rightZ[y]
		dx := float32(endX - startX)
		dz := endZ - startZ
		for x := startX; x <= endX; x++ {
			k := float32(1.0)
			if dx != 0 {
				k = float32(x-startX) / dx
			}

			z := startZ + dz*k
			hm.PlotPixel(x, y, z)
		}
	}
}

// X,Y should be in pixels
// Z should range from 0..1
func (hm *Heightmap) DrawTriangleOnOneLine(a, b, c [3]float32, ycoord int) {
	// min/max X position for the central Y position
	leftX := hm.options.width
	rightX := 0
	// Z coordinate for corresponding leftX/rightX
	leftZ := float32(0)
	rightZ := float32(0)

	// 1. work out where the outline of the triangle is
	perimeterCb := func(x, y int, z float32) {
		if y != 0 {
			return
		}
		if x < leftX {
			leftX = x
			leftZ = z
		}
		if x > rightX {
			rightX = x
			rightZ = z
		}
	}
	hm.IterateLine(a, b, perimeterCb)
	hm.IterateLine(b, c, perimeterCb)
	hm.IterateLine(c, a, perimeterCb)

	if leftX >= hm.options.width {
		return
	}

	// 2. draw single scanline
	dx := float32(rightX - leftX)
	dz := rightZ - leftZ
	for x := leftX; x <= rightX; x++ {
		k := float32(1.0)
		if dx != 0 {
			k = float32(x-leftX) / dx
		}

		z := leftZ + dz*k
		hm.PlotPixel(x, ycoord, z)
	}
}

func (hm *Heightmap) PlotPixel(x, y int, z float32) {
	opt := hm.options

	if x < 0 || x >= opt.width || y < 0 || y >= opt.height {
		return
	}

	n := y*opt.width + x

	if z > hm.height[n] {
		hm.height[n] = z
	}
}

func (hm *Heightmap) IterateLine(a, b [3]float32, cb func(int, int, float32)) {
	// visit the first point
	cb(int(a[0]), int(a[1]), a[2])

	dx := b[0] - a[0]
	dy := b[1] - a[1]
	dz := b[2] - a[2]
	length := float32(math.Sqrt(float64(dx*dx + dy*dy))) // 2d length

	// if the line has 0px length, only plot the 1st pixel, and avoid dividing by 0
	if length < 1 {
		return
	}

	dx /= length
	dy /= length
	dz /= length

	x, y, z := a[0], a[1], a[2]

	// visit each point on the line, stepping 1px at a time along a diagonal
	for i := 1; i <= int(length); i++ {
		x += dx
		y += dy
		z += dz
		cb(int(x), int(y), z)
	}
}
