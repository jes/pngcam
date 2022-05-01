package main

import (
    "image"
    "image/png"
    "math"
    "os"
)

type HeightmapImage struct {
    img image.Image
    options *Options
}

type ToolpointsMap struct {
    w int
    h int
    hm *HeightmapImage
    height []float64
    options *Options
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
        img: img,
        options: opt,
    }, nil
}

func (hm *HeightmapImage) ToToolpointsMap() *ToolpointsMap {
    opt := hm.options

    w := hm.img.Bounds().Max.X
    h := hm.img.Bounds().Max.Y

    tpm := NewToolpointsMap(w,h, opt, math.NaN())
    tpm.hm = hm

    return tpm
}

func (hm *HeightmapImage) CutDepth(x, y float64) float64 {
    opt := hm.options
    tool := opt.tool

    belowBottomDepth := -opt.depth - tool.Radius() + opt.stockToLeave

    maxDepth := belowBottomDepth

    toolRadiusSqr := tool.Radius()*tool.Radius()

    for sy := -tool.Radius(); sy <= tool.Radius(); sy += opt.y_MmPerPx {
        for sx := -tool.Radius(); sx <= tool.Radius(); sx += opt.x_MmPerPx {
            rSqr := sx*sx + sy*sy
            if rSqr > toolRadiusSqr {
                continue
            }

            z := opt.stockToLeave - tool.HeightAtRadiusSqr(rSqr)

            if !opt.cutBelowBottom || !hm.IsBottom(x+sx, y+sy) {
                d := z + hm.GetDepth(x+sx, y+sy)
                if d > maxDepth {
                    maxDepth = d
                }
            }
        }
    }

    return maxDepth
}

func (hm *HeightmapImage) GetDepth(x, y float64) float64 {
    opt := hm.options

    px,py := opt.MmToPx(x, y)

    r,g,b,_ := hm.img.At(px, py).RGBA()
    r /= 257
    g /= 257
    b /= 257
    brightness := float64(65536*r + 256*g + b) / 16777215;

    return brightness * opt.depth - opt.depth
}

func (hm *HeightmapImage) IsBottom(x, y float64) bool {
    epsilon := 0.00001

    return hm.GetDepth(x,y) < -hm.options.depth+epsilon
}

func NewToolpointsMap(w,h int, options *Options, init float64) *ToolpointsMap {
    tpm := ToolpointsMap{
        w: w,
        h: h,
        hm: nil,
        height: make([]float64, w*h),
        options: options,
    }

    for i := 0; i < w*h; i++ {
        tpm.height[i] = init
    }

    return &tpm
}

func (m *ToolpointsMap) SetMm(x, y, z float64) {
    px,py := m.options.MmToPx(x,y)
    m.SetPx(px, py, z)
}

func (m *ToolpointsMap) GetMm(x, y float64) float64 {
    px,py := m.options.MmToPx(x,y)
    return m.GetPx(px, py)
}

func (m *ToolpointsMap) SetPx(x, y int, z float64) {
    if x < 0 || y < 0 || x >= m.w || y >= m.h {
        return
    }
    m.height[y*m.w + x] = z
}

func (m *ToolpointsMap) GetPx(x, y int) float64 {
    if x < 0 || y < 0 || x >= m.w || y >= m.h {
        if m.hm == nil {
            return math.Inf(-1)
        } else {
            return m.hm.CutDepth(m.options.PxToMm(x, y))
        }
    }
    if math.IsNaN(m.height[y*m.w + x]) {
        if m.hm != nil {
            m.height[y*m.w + x] = m.hm.CutDepth(m.options.PxToMm(x, y))
        }
    }
    return m.height[y*m.w + x]
}

func (m *ToolpointsMap) WritePNG(path string) {
    img := image.NewRGBA(image.Rect(0, 0, m.w, m.h))

    for y := 0; y < m.h; y++ {
        for x := 0; x < m.w; x++ {
            n := y*m.w + x

            z := m.height[n]
            if z > 0 { z = 0 }
            if z < -m.options.depth { z = -m.options.depth }
            brightness := int(16777215 * (z/m.options.depth+1))

            img.Pix[n*4] = uint8(brightness >> 16)
            img.Pix[n*4+1] = uint8((brightness >> 8) & 0xff)
            img.Pix[n*4+2] = uint8(brightness & 0xff)
            img.Pix[n*4+3] = 255
        }
    }

    out, _ := os.Create(path)
    png.Encode(out, img)
    out.Close()
}

func (m *ToolpointsMap) PlotPixel(x, y, z float64) {
    curZ := m.GetMm(x, y)
    if math.IsNaN(curZ) || z < curZ {
        m.SetMm(x, y, z)
    }
}

func (m *ToolpointsMap) PlotPoint(x, y, z float64) {
    opt := m.options
    tool := opt.tool

    // pretend tool is 1px larger so that we don't leave tall spikes between rows
    r := tool.Radius() + opt.x_MmPerPx

    toolRadiusSqr := tool.Radius()*tool.Radius()

    for sy := -r; sy <= r; sy += opt.y_MmPerPx {
        for sx := -r; sx <= r; sx += opt.x_MmPerPx {
            rSqr := sx*sx + sy*sy
            if rSqr > toolRadiusSqr {
                continue
            }
            zOffset := tool.HeightAtRadiusSqr(rSqr)
            m.PlotPixel(x+sx, y+sy, z+zOffset)
        }
    }
}

func (m *ToolpointsMap) PlotLine(x1,y1,z1, x2,y2,z2 float64) {
    dx := x2-x1
    dy := y2-y1
    dz := z2-z1

    xyDist := math.Sqrt(dx*dx + dy*dy)

    xStep := dx / xyDist
    yStep := dy / xyDist
    zStep := dz / xyDist

    // TODO: might be wrong if x_MmPerPx is substantially different to y_MmPerPx
    for k := 0.0; k <= xyDist; k += m.options.x_MmPerPx {
        m.PlotPoint(x1 + xStep*k, y1 + yStep*k, z1 + zStep*k)
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
