package main

import (
    "fmt"
    "strings"
)

type Job struct {
    options *Options
    toolpoints *ToolpointsMap
    readStock *ToolpointsMap
    writeStock *ToolpointsMap
    mainToolpath Toolpath
}

func NewJob(opt *Options) (*Job, error) {
    j := Job{}
    j.options = opt

    hm, err := OpenHeightmapImage(opt.heightmapPath, opt)
    if err != nil {
        return nil, err
    }

    j.toolpoints = hm.ToToolpointsMap()

    if opt.readStockPath != "" {
        readImg, err := OpenHeightmapImage(opt.readStockPath, opt)
        if err != nil {
            return nil, err
        }
        j.readStock = readImg.ToToolpointsMap()
    }

    if opt.writeStockPath != "" {
        j.writeStock = NewToolpointsMap(hm.img.Bounds().Max.X, hm.img.Bounds().Max.Y, opt, 0)
    }

    j.MakeToolpath()

    return &j, nil
}

func (j *Job) MakeToolpath() {
    j.mainToolpath = NewToolpath()

    opt := j.options

    xLimit := opt.width
    yLimit := opt.height

    xStep := j.toolpoints.x_MmPerPx
    yStep := 0.0
    if opt.direction == Vertical {
        xStep = 0.0
        yStep = j.toolpoints.y_MmPerPx
    }

    zero := 0.0

    if opt.cutBeyondEdges {
        extraLimit := opt.tool.Radius()
        zero -= extraLimit
        xLimit += extraLimit
        yLimit += extraLimit
    }

    x := zero
    y := zero

    // TODO: maybe the step over should also follow the contours of the toolpoints map, 1 px at a time? maybe something like:
    // addPathSegment(0,0, 100,0)
    // addPathSegment(100,0, 100,10)
    // addPathSegment(100,10, 0,10)
    // addPathSegment(0,10, 0,20)
    // ...

    for x >= zero && y >= zero && x < xLimit && y < yLimit {
        seg := NewToolpathSegment()

        for x >= zero && y >= zero && x < xLimit && y < yLimit {
            seg.Append(Toolpoint{x, y, j.toolpoints.GetMm(x,y)})

            x += xStep
            y += yStep
        }

        if opt.omitTop {
            j.mainToolpath.AppendToolpath(seg.OmitTop())
        } else {
            j.mainToolpath.Append(seg)
        }

        if opt.direction == Horizontal {
            y += opt.stepOver
        } else {
            x += opt.stepOver
        }

        xStep = -xStep
        yStep = -yStep
        x += xStep
        y += yStep
    }
}

func (j *Job) Gcode() string {
    roughingPath := j.Roughing()
    if j.writeStock != nil {
        j.writeStock.PlotToolpath(roughingPath)
    }

    gcode := ""

    if (j.options.roughingOnly) {
        gcode = j.Preamble() + roughingPath.ToGcode(*j.options) + j.Postamble()
    } else {
        finishingPath := j.Finishing()
        if j.writeStock != nil {
            j.writeStock.PlotToolpath(finishingPath)
        }
        gcode = j.Preamble() + roughingPath.ToGcode(*j.options) + finishingPath.ToGcode(*j.options) + j.Postamble()
    }

    if j.writeStock != nil {
        j.writeStock.WritePNG(j.options.writeStockPath)
    }

    return gcode
}

func (j *Job) Preamble() string {
    opt := j.options

    gcode := strings.Builder{}

    if opt.imperial {
        gcode.WriteString("G20\n") // inches
    } else {
        gcode.WriteString("G21\n") // mm
    }
    gcode.WriteString("G90\n") // absolute coordinates
    gcode.WriteString("G54\n") // work coordinate system

    fmt.Fprintf(&gcode, "M3 S%g\n", opt.rpm)

    return gcode.String()
}

func (j *Job) Postamble() string {
    return "M5\nM2\n" // stop spindle, end program
}

func (j *Job) Finishing() *Toolpath {
    return j.mainToolpath.Simplified()
}

func (j *Job) Roughing() *Toolpath {
    opt := j.options

    deepest := -opt.depth
    if opt.cutBelowBottom {
        deepest -= opt.tool.Radius()
    }

    path := NewToolpath()

    for z := -opt.stepDown; z > deepest; z -= opt.stepDown {
        path.AppendToolpath(j.RoughingLevel(z).Simplified())
    }

    return &path
}

func (j *Job) RoughingLevel(z float64) *Toolpath {
    path := NewToolpath()

    for i := range j.mainToolpath.segments {
        seg := NewToolpathSegment()
        for p := range j.mainToolpath.segments[i].points {
            tp := j.mainToolpath.segments[i].points[p]
            if tp.z < z && (j.readStock == nil || z < j.readStock.GetMm(tp.x,tp.y)) {
                // add this point to this roughing segment
                seg.Append(Toolpoint{tp.x, tp.y, z})
            } else {
                // this point isn't in this segment: append what we have and make a new segment
                if len(seg.points) > 0 {
                    path.Append(seg)
                }
                seg = NewToolpathSegment()
            }
        }

        if len(seg.points) > 0 {
            path.Append(seg)
        }
    }

    return &path
}
