package main

import (
    "fmt"
    "math"
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

func (seg *ToolpathSegment) Append(t Toolpoint) {
    seg.points = append(seg.points, t)
}

func (seg *ToolpathSegment) Simplified() *ToolpathSegment {
    newseg := NewToolpathSegment()

    if len(seg.points) == 0 {
        return &newseg
    }

    newseg.Append(seg.points[0])

    if len(seg.points) == 1 {
        return &newseg
    }

    epsilon := 0.00001

    prev := seg.points[1]

    for i := 2 ; i < len(seg.points); i++ {
        first := newseg.points[len(newseg.points)-1]
        cur := seg.points[i]

        prev_xy := math.Atan2(prev.y-first.y, prev.x-first.x)
        cur_xy := math.Atan2(cur.y-prev.y, cur.x-prev.x)
        prev_xz := math.Atan2(prev.z-first.z, prev.x-first.x)
        cur_xz := math.Atan2(cur.z-prev.z, cur.x-prev.x)
        prev_yz := math.Atan2(prev.z-first.z, prev.y-first.y)
        cur_yz := math.Atan2(cur.z-prev.z, cur.y-prev.y)

        // if the route first->prev has the same angle as prev->cur, then first->prev->cur is
        // a straight line, so we can remove prev and just go straight from first->cur

        if math.Abs(cur_xy-prev_xy) > epsilon || math.Abs(cur_xz-prev_xz) > epsilon || math.Abs(cur_yz-prev_yz) > epsilon {
            newseg.Append(prev)
        }
        prev = cur
    }

    newseg.Append(prev)

    return &newseg
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

func (tp *Toolpath) Simplified() *Toolpath {
    newtp := NewToolpath()

    for i := 0; i < len(tp.segments); i++ {
        newtp.Append(*(tp.segments[i].Simplified()))
    }

    return &newtp
}

func (tp *Toolpath) Append(seg ToolpathSegment) {
    tp.segments = append(tp.segments, seg)
}

func (tp *Toolpath) AppendToolpath(more *Toolpath) {
    for i := 0; i < len(more.segments); i++ {
        tp.Append(more.segments[i])
    }
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
