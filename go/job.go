package main

import (
    "fmt"
    "strings"
)

type Job struct {
    options *Options
    toolpoints *ToolpointsMap
    readStock *ToolpointsMap
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
        hm, err = OpenHeightmapImage(opt.readStockPath, opt)
        if err != nil {
            return nil, err
        }
        j.readStock = hm.ToToolpointsMap()
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

    x := 0.0
    y := 0.0

    // TODO: maybe the step over should also follow the contours of the toolpoints map, 1 px at a time? maybe something like:
    // addPathSegment(0,0, 100,0)
    // addPathSegment(100,0, 100,10)
    // addPathSegment(100,10, 0,10)
    // addPathSegment(0,10, 0,20)
    // ...

    for x >= 0.0 && y >= 0.0 && x < xLimit && y < yLimit {
        seg := NewToolpathSegment()

        for x >= 0.0 && y >= 0.0 && x < xLimit && y < yLimit {
            seg.points = append(seg.points, Toolpoint{x, y, j.toolpoints.GetMm(x,y)})

            x += xStep
            y += yStep
        }

        j.mainToolpath.segments = append(j.mainToolpath.segments, seg)

        if opt.direction == Horizontal {
            y += opt.stepOver
            xStep = -xStep
            x += xStep
        } else {
            x += opt.stepOver
            yStep = -yStep
            y += yStep
        }
    }
}

func (j *Job) Gcode() string {
    if (j.options.roughingOnly) {
        return j.Preamble() + j.Roughing() + j.Postamble()
    } else {
        return j.Preamble() + j.Roughing() + j.Finishing() + j.Postamble()
    }
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

func (j *Job) Finishing() string {
    return j.mainToolpath.ToGcode(*j.options)
}

func (j *Job) Roughing() string {
    opt := j.options

    deepest := -opt.depth
    if opt.cutBelowBottom {
        deepest -= opt.tool.Radius()
    }

    gcode := ""

    for z := -opt.stepDown; z > deepest; z -= opt.stepDown {
        gcode += j.RoughingLevel(z).ToGcode(*opt)
    }

    return gcode
}

func (j *Job) RoughingLevel(z float64) *Toolpath {
    path := NewToolpath()

    for i := 0; i < len(j.mainToolpath.segments); i++ {
        seg := NewToolpathSegment()
        for p := 0; p < len(j.mainToolpath.segments[i].points); p++ {
            tp := j.mainToolpath.segments[i].points[p]
            if tp.z < z {
                // add this point to this roughing segment
                seg.points = append(seg.points, Toolpoint{tp.x, tp.y, z})
            } else {
                // this point isn't in this segment: append what we have and make a new segment
                if len(seg.points) > 0 {
                    path.segments = append(path.segments, seg)
                }
                seg = NewToolpathSegment()
            }
        }

        if len(seg.points) > 0 {
            path.segments = append(path.segments, seg)
        }
    }

    return &path
}
