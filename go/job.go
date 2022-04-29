package main

import (
    "fmt"
    "strings"
)

type Job struct {
    options *Options
    toolpoints *ToolpointsMap
    readStock *ToolpointsMap
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

    return &j, nil
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
    return ""
}

func (j *Job) Roughing() string {
    return ""
}
