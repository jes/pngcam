package main

import (
    "testing"
)

func TestToolpathSegment(t *testing.T) {
    opt := Options{
        safeZ: 5,
        rapidFeed: 10000,
        xyFeed: 2000,
        zFeed: 200,
    }

    seg1 := ToolpathSegment{
        points: []Toolpoint{
            {0,0,0},
            {1,1,0},
            {0,1,0},
            {0,1,-10},
            {0,10,-10},
            {0,10,0},
        },
    }
    gcode1 := seg1.ToGcode(opt)
    wantGcode1 := `G1 X1.0000 Y1.0000 Z0.0000 F2000
G1 X0.0000 Y1.0000 Z0.0000 F2000
G1 X0.0000 Y1.0000 Z-10.0000 F200
G1 X0.0000 Y10.0000 Z-10.0000 F2000
G1 X0.0000 Y10.0000 Z0.0000 F10000
`
    if gcode1 != wantGcode1 {
        t.Errorf("incorrect gcode; got:\n%v\n---\nexpected:\n%v", gcode1, wantGcode1)
    }

    seg2 := ToolpathSegment{
        points: []Toolpoint{
            {20,0,0},
            {21,1,0},
            {20,1,0},
            {20,1,-10},
            {20,10,-10},
            {20,10,0},
        },
    }
    gcode2 := seg2.ToGcode(opt)
    wantGcode2 := `G1 X21.0000 Y1.0000 Z0.0000 F2000
G1 X20.0000 Y1.0000 Z0.0000 F2000
G1 X20.0000 Y1.0000 Z-10.0000 F200
G1 X20.0000 Y10.0000 Z-10.0000 F2000
G1 X20.0000 Y10.0000 Z0.0000 F10000
`
    if gcode2 != wantGcode2 {
        t.Errorf("incorrect gcode; got:\n%v\n---\nexpected:\n%v", gcode2, wantGcode2)
    }

    path := Toolpath{segments: []ToolpathSegment{seg1,seg2}}
    gcode := path.ToGcode(opt)
    wantGcode := `G1 Z5.0000 F10000
G1 X0.0000 Y0.0000 F10000
G1 X1.0000 Y1.0000 Z0.0000 F2000
G1 X0.0000 Y1.0000 Z0.0000 F2000
G1 X0.0000 Y1.0000 Z-10.0000 F200
G1 X0.0000 Y10.0000 Z-10.0000 F2000
G1 X0.0000 Y10.0000 Z0.0000 F10000
G1 Z5.0000 F10000
G1 X20.0000 Y0.0000 F10000
G1 X21.0000 Y1.0000 Z0.0000 F2000
G1 X20.0000 Y1.0000 Z0.0000 F2000
G1 X20.0000 Y1.0000 Z-10.0000 F200
G1 X20.0000 Y10.0000 Z-10.0000 F2000
G1 X20.0000 Y10.0000 Z0.0000 F10000
G1 Z5.0000 F10000
`
    if gcode != wantGcode {
        t.Errorf("incorrect gcode; got:\n%v\n---\nexpected:\n%v", gcode, wantGcode)
    }
}
