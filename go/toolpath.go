package main

import (
    "fmt"
    "strings"
)

type Toolpoint struct {
    x float64
    y float64
    z float64
}

type ToolpathSegment struct {
    points []Toolpoint
}

type Toolpath struct {
    segments []ToolpathSegment
}

func NewToolpathSegment() ToolpathSegment {
    return ToolpathSegment{
        points: []Toolpoint{},
    }
}

func NewToolpath() Toolpath {
    return Toolpath{
        segments: []ToolpathSegment{},
    }
}

func (seg *ToolpathSegment) ToGcode(opt Options) string {
    gcode := strings.Builder{}

    // start at i=1 because we assume we're starting from point 0
    for i := 1; i < len(seg.points); i++ {
        p := seg.points[i]
        fmt.Fprintf(&gcode, "G1 X%.04f Y%.04f Z%.04f F%g\n", p.x, p.y, p.z, opt.FeedRate(seg.points[i-1], p))
    }

    return gcode.String()
}

func (tp *Toolpath) ToGcode(opt Options) string {
    gcode := strings.Builder{}

    // hop up to safe Z
    fmt.Fprintf(&gcode, "G1 Z%.04f F%g\n", opt.safeZ, opt.rapidFeed)

    for i := 0; i < len(tp.segments); i++ {
        p0 := tp.segments[i].points[0]

        // move to the start point of this segment
        fmt.Fprintf(&gcode, "G1 X%.04f Y%.04f F%g\n", p0.x, p0.y, opt.rapidFeed)

        // rapid down to safe Z above start height?
        if p0.z+opt.safeZ < opt.safeZ {
            fmt.Fprintf(&gcode, "G1 Z%.04f F%g\n", p0.z+opt.safeZ, opt.rapidFeed)
        }

        // feed down to start height
        // TODO: ramp entry
        fmt.Fprintf(&gcode, "G1 Z%.04f F%g\n", p0.z, opt.zFeed)

        // move through the rest of the segment
        gcode.WriteString(tp.segments[i].ToGcode(opt))

        // back up to safe Z
        fmt.Fprintf(&gcode, "G1 Z%.04f F%g\n", opt.safeZ, opt.rapidFeed)
    }

    return gcode.String()
}
