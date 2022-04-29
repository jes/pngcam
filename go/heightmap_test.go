package main

import (
    "testing"
)

func TestHeightmap(t *testing.T) {
    tool, err := NewTool("ball", 10)
    if err != nil {
        t.Errorf("can't create ball tool: %v", err)
    }

    opt := Options{
        width: 232,
        height: 650,
        depth: 10,

        stepOver: 1,
        stepForward: 1,
        stepDown: 1,

        direction: Horizontal,

        tool: tool,
        stockToLeave: 0,
    }

    heightmap, err := OpenHeightmapImage("../klingon-dagger.png", &opt)
    if err != nil {
        t.Errorf("can't open image: %v", err)
    }

    toolpointsmap := heightmap.ToToolpointsMap()
    for y := 0; y < toolpointsmap.h; y++ {
        for x := 0; x < toolpointsmap.w; x++ {
            z := toolpointsmap.GetPx(x, y)
            if z < -opt.depth {
                t.Errorf("depth below bottom: %v,%v,%v", x,y,z)
            }

            if z > 0 {
                t.Errorf("depth above top: %v,%v,%v", x,y,z)
            }
        }
    }

    toolpointsmap.WritePNG("foo.png")
}
