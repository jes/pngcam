package main

import (
	"fmt"
	"image"
	"image/png"
	"math"
	"os"
)

type HeightmapImage struct {
	img     image.Image
	options *Options
}

type ToolpointsMap struct {
	w             int
	h             int
	hm            *HeightmapImage
	height        []float64
	initialHeight float64
	options       *Options
}

func OpenHeightmapImage(path string, opt *Options) (*HeightmapImage, error) {
	reader, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	img, _, err := image.Decode(reader)
	if err != nil {
		return nil, err
	}

	return &HeightmapImage{
		img:     img,
		options: opt,
	}, nil
}

func (hm *HeightmapImage) ToToolpointsMap() *ToolpointsMap {
	opt := hm.options

	w := hm.img.Bounds().Max.X
	h := hm.img.Bounds().Max.Y

	tpm := NewToolpointsMap(w, h, opt, math.NaN())
	tpm.hm = hm

	return tpm
}

func (hm *HeightmapImage) CutDepth(x, y float64) float64 {
	opt := hm.options
	tool := opt.tool

	belowBottomDepth := -opt.depth - tool.Radius() + opt.stockToLeave

	maxDepth := belowBottomDepth

	toolRadiusSqr := tool.Radius() * tool.Radius()

	if opt.rotary {
		for sy := -90.0; sy <= 90.0; sy += opt.y_MmPerPx { // we pretend the y range of 360 degrees is 360 "millimetres"
			for sx := -tool.Radius(); sx <= tool.Radius(); sx += opt.x_MmPerPx {
				workpieceZ := opt.depth + hm.GetDepth(x+sx, -1-y+sy) // -y because the heightmap y axis is inverted (?) (but why -1 degree?)
				realY := workpieceZ * math.Sin(sy*math.Pi/180.0)
				realZ := workpieceZ * math.Cos(sy*math.Pi/180.0)

				rSqr := sx*sx + realY*realY
				if rSqr > toolRadiusSqr {
					continue
				}

				// TODO: what about if !opt.cutBelowBottom || !hm.IsBottom(x+sx, ...) ?

				d := opt.stockToLeave - tool.HeightAtRadiusSqr(rSqr) + realZ
				if d > maxDepth {
					maxDepth = d
				}
			}
		}
	} else {
		for sy := -tool.Radius(); sy <= tool.Radius(); sy += opt.y_MmPerPx {
			for sx := -tool.Radius(); sx <= tool.Radius(); sx += opt.x_MmPerPx {
				rSqr := sx*sx + sy*sy
				if rSqr > toolRadiusSqr {
					continue
				}

				if !opt.cutBelowBottom || !hm.IsBottom(x+sx, y+sy) {
					d := opt.stockToLeave - tool.HeightAtRadiusSqr(rSqr) + hm.GetDepth(x+sx, y+sy)
					if d > maxDepth {
						maxDepth = d
					}
				}
			}
		}
	}

	return maxDepth
}

func (hm *HeightmapImage) GetDepth(x, y float64) float64 {
	opt := hm.options

	px, py := opt.MmToPx(x, y)

	return hm.GetDepthPx(px, py)
}

func (hm *HeightmapImage) GetDepthPx(px, py int) float64 {
	opt := hm.options

	if opt.rotary {
		// rotary parts wrap around
		py = ((py % opt.heightPx) + opt.heightPx) % opt.heightPx // https://stackoverflow.com/a/59299881
	}

	r, g, b, _ := hm.img.At(px, py).RGBA()
	// XXX: why 257? https://stackoverflow.com/a/41185404 but doesn't really
	// explain - empirically it doesn't make any difference whether it is 256 or
	// 257, presumably it rounds to the same result
	r /= 257
	g /= 257
	b /= 257
	brightness := float64(65536*r+256*g+b) / 16777215

	return brightness*opt.depth - opt.depth
}

func (hm *HeightmapImage) IsBottom(x, y float64) bool {
	epsilon := 0.00001

	return hm.GetDepth(x, y) < -hm.options.depth+epsilon
}

func NewToolpointsMap(w, h int, options *Options, init float64) *ToolpointsMap {
	tpm := ToolpointsMap{
		w:             w,
		h:             h,
		hm:            nil,
		height:        make([]float64, w*h),
		options:       options,
		initialHeight: init,
	}

	for i := 0; i < w*h; i++ {
		tpm.height[i] = init
	}

	return &tpm
}

func (m *ToolpointsMap) SetMm(x, y, z float64) {
	px, py := m.options.MmToPx(x, y)
	m.SetPx(px, py, z)
}

func (m *ToolpointsMap) GetMm(x, y float64) float64 {
	px, py := m.options.MmToPx(x, y)
	return m.GetPx(px, py)
}

func (m *ToolpointsMap) SetPx(x, y int, z float64) {
	if x < 0 || y < 0 || x >= m.w || y >= m.h {
		return
	}
	m.height[y*m.w+x] = z
}

func (m *ToolpointsMap) GetPx(x, y int) float64 {
	if x < 0 || y < 0 || x >= m.w || y >= m.h {
		if m.hm == nil {
			return math.Inf(-1)
		} else {
			return m.hm.CutDepth(m.options.PxToMm(x, y))
		}
	}
	if math.IsNaN(m.height[y*m.w+x]) {
		if m.hm != nil {
			m.height[y*m.w+x] = m.hm.CutDepth(m.options.PxToMm(x, y))
		}
	}
	return m.height[y*m.w+x]
}

func (m *ToolpointsMap) WritePNG(path string, existingStock *HeightmapImage) error {
	m2 := NewToolpointsMap(m.w, m.h, m.options, 0)
	if existingStock != nil {
		for y := 0; y < m2.h; y++ {
			for x := 0; x < m2.w; x++ {
				n := y*m2.w + x
				m2.height[n] = existingStock.GetDepthPx(x, y)
			}
		}
	}

	if !m.options.quiet {
		fmt.Fprintf(os.Stderr, "Writing stock: 0%%")
	}
	for y := 0; y < m.h; y++ {
		for x := 0; x < m.w; x++ {
			xMm, yMm := m.options.PxToMm(x, y)
			if m.height[y*m.w+x] != m.initialHeight {
				m2.PlotToolShape(xMm, yMm, m.height[y*m.w+x])
			}
		}

		if !m.options.quiet {
			pct := float64(100 * y / m.h)
			fmt.Fprintf(os.Stderr, "   \rWriting stock: %.0f%%", pct)
		}
	}

	img := image.NewRGBA(image.Rect(0, 0, m.w, m.h))

	for y := 0; y < m.h; y++ {
		for x := 0; x < m.w; x++ {
			n := y*m.w + x

			z := m2.height[n]
			if z > 0 {
				z = 0
			}
			if z < -m.options.depth {
				z = -m.options.depth
			}
			brightness := int(16777215 * (z/m.options.depth + 1))

			if m.options.rgb {
				img.Pix[n*4] = uint8(brightness >> 16)
				img.Pix[n*4+1] = uint8((brightness >> 8) & 0xff)
				img.Pix[n*4+2] = uint8(brightness & 0xff)
			} else {
				img.Pix[n*4] = uint8(brightness >> 16)
				img.Pix[n*4+1] = uint8(brightness >> 16)
				img.Pix[n*4+2] = uint8(brightness >> 16)
			}
			img.Pix[n*4+3] = 255
		}
	}

	out, err := os.Create(path)
	if err != nil {
		return err
	}
	png.Encode(out, img)
	out.Close()

	if !m.options.quiet {
		fmt.Fprintf(os.Stderr, "   \rWriting stock: done\n")
	}

	return err
}

func (m *ToolpointsMap) PlotPixelMm(x, y, z float64) {
	px, py := m.options.MmToPx(x, y)
	m.PlotPixelPx(px, py, z)
}

func (m *ToolpointsMap) PlotPixelPx(px, py int, z float64) {
	curZ := m.GetPx(px, py)
	if math.IsNaN(curZ) || z < curZ {
		m.SetPx(px, py, z)
	}
}

func (m *ToolpointsMap) PlotToolShape(x, y, z float64) {
	opt := m.options
	tool := opt.tool

	xPx, yPx := opt.MmToPx(x, y)

	r := tool.Radius()
	rPxX := int(r/opt.x_MmPerPx) + 1
	rPxY := int(r/opt.y_MmPerPx) + 1
	if opt.rotary {
		rPxY = int(90.0/opt.y_MmPerPx) + 1
	}

	toolRadiusSqr := r * r

	if opt.rotary {
		for sy := -rPxY; sy <= rPxY; sy++ {
			for sx := -rPxX; sx <= rPxX; sx++ {
				sxMm := float64(sx) * opt.x_MmPerPx
				syDeg := float64(sy) * opt.y_MmPerPx // degrees

				height := tool.LengthToIntersection(sxMm, syDeg, z)
				m.PlotPixelPx(xPx+sx, yPx+sy, height-opt.depth)
			}
		}
	} else {
		for sy := -rPxY; sy <= rPxY; sy++ {
			for sx := -rPxX; sx <= rPxX; sx++ {
				sxMm := float64(sx) * opt.x_MmPerPx
				syMm := float64(sy) * opt.y_MmPerPx

				rSqr := sxMm*sxMm + syMm*syMm
				if rSqr > toolRadiusSqr {
					continue
				}
				zOffset := tool.HeightAtRadiusSqr(rSqr)
				m.PlotPixelPx(xPx+sx, yPx+sy, z+zOffset)
			}
		}
	}
}

func (m *ToolpointsMap) PlotLine(x1, y1, z1, x2, y2, z2 float64) {
	dx := x2 - x1
	dy := y2 - y1
	dz := z2 - z1

	xyDist := math.Sqrt(dx*dx + dy*dy)

	xStep := dx / xyDist
	yStep := dy / xyDist
	zStep := dz / xyDist

	// TODO: might be wrong if x_MmPerPx is substantially different to y_MmPerPx
	for k := 0.0; k <= xyDist; k += m.options.x_MmPerPx {
		m.PlotPixelMm(x1+xStep*k, y1+yStep*k, z1+zStep*k)
	}
}

func (m *ToolpointsMap) PlotToolpathSegment(seg *ToolpathSegment) {
	if len(seg.points) == 0 {
		return
	}

	if len(seg.points) == 1 {
		m.PlotLine(seg.points[0].x, seg.points[0].y, seg.points[0].z, seg.points[0].x, seg.points[0].y, seg.points[0].z)
		return
	}

	for i := 1; i < len(seg.points); i++ {
		m.PlotLine(seg.points[i-1].x, seg.points[i-1].y, seg.points[i-1].z, seg.points[i].x, seg.points[i].y, seg.points[i].z)
	}
}

func (m *ToolpointsMap) PlotToolpath(tp *Toolpath) {
	for i := range tp.segments {
		m.PlotToolpathSegment(&tp.segments[i])
	}
}
