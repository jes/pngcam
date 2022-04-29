package main

import (
    "image"
    "image/png"
    "math"
    "os"
)

type HeightmapImage struct {
    img image.Image
    x_MmPerPx float64
    y_MmPerPx float64
    options *Options
}

type ToolpointsMap struct {
    w int
    h int
    height []float64
    x_MmPerPx float64
    y_MmPerPx float64
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
        x_MmPerPx: opt.width / float64(img.Bounds().Max.X),
        y_MmPerPx: opt.height / float64(img.Bounds().Max.Y),
        options: opt,
    }, nil
}

func (hm *HeightmapImage) ToToolpointsMap() *ToolpointsMap {
    opt := hm.options

    w := hm.img.Bounds().Max.X
    h := hm.img.Bounds().Max.Y

    tpm := &ToolpointsMap{
        w: w,
        h: h,
        height: make([]float64, w*h),
        x_MmPerPx: hm.x_MmPerPx,
        y_MmPerPx: hm.y_MmPerPx,
        options: opt,
    }

    for y := 0; y < h; y++ {
        for x := 0; x < w; x++ {
            tpm.SetPx(x, y, hm.CutDepth(float64(x) * tpm.x_MmPerPx, float64(y) * tpm.y_MmPerPx))
        }
    }

    return tpm
}

func (hm *HeightmapImage) CutDepth(x, y float64) float64 {
    opt := hm.options
    tool := opt.tool

    belowBottomDepth := -opt.depth - tool.Radius() + opt.stockToLeave

    maxDepth := belowBottomDepth

    for sy := -tool.Radius(); sy <= tool.Radius(); sy += hm.y_MmPerPx {
        for sx := -tool.Radius(); sx <= tool.Radius(); sx += hm.x_MmPerPx {
            r := math.Sqrt(sx*sx + sy*sy)
            if r > tool.Radius() {
                continue
            }

            z := opt.stockToLeave - tool.HeightAtRadius(r)

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

    px := int(x / hm.x_MmPerPx)
    py := int(y / hm.y_MmPerPx)

    r,g,b,_ := hm.img.At(px, py).RGBA()
    r /= 257
    g /= 257
    b /= 257
    brightness := float64(65536*r + 256*g + b) / 16777215;

    return brightness * opt.depth - opt.depth
}

func (hm *HeightmapImage) IsBottom(x, y float64) bool {
    return hm.GetDepth(x,y) == 0
}

func (m *ToolpointsMap) SetMm(x, y, z float64) {
    px,py := m.MmToPx(x,y)
    m.SetPx(px, py, z)
}

func (m *ToolpointsMap) GetMm(x, y float64) float64 {
    px,py := m.MmToPx(x,y)
    return m.GetPx(px, py)
}

func (m *ToolpointsMap) MmToPx(x, y float64) (int, int) {
    return int(x / m.x_MmPerPx), int(y / m.y_MmPerPx)
}

func (m *ToolpointsMap) SetPx(x, y int, z float64) {
    if x < 0 || y < 0 || x >= m.w || y >= m.h {
        return
    }
    m.height[y*m.w + x] = z
}

func (m *ToolpointsMap) GetPx(x, y int) float64 {
    if x < 0 || y < 0 || x >= m.w || y >= m.h {
        return math.Inf(-1)
    }
    return m.height[y*m.w + x]
}

func (m *ToolpointsMap) WritePNG(path string) {
    img := image.NewRGBA(image.Rect(0, 0, m.w, m.h))

    for y := 0; y < m.h; y++ {
        for x := 0; x < m.w; x++ {
            n := y*m.w + x
            img.Pix[n*4] = uint8(255 * (m.height[n]/m.options.depth+1))
            img.Pix[n*4+1] = img.Pix[n*4]
            img.Pix[n*4+2] = img.Pix[n*4]
            img.Pix[n*4+3] = 255
        }
    }

    out, _ := os.Create(path)
    png.Encode(out, img)
    out.Close()
}
