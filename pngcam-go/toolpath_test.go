package main

import (
	"testing"
)

func TestToolpaths(t *testing.T) {
	opt := Options{
		safeZ:     5,
		rapidFeed: 10000,
		xyFeed:    2000,
		zFeed:     200,
	}

	seg1 := ToolpathSegment{
		points: []Toolpoint{
			{0, 0, 0, CuttingFeed},
			{1, 1, 0, CuttingFeed},
			{0, 1, 0, CuttingFeed},
			{0, 1, -10, CuttingFeed},
			{0, 10, -10, CuttingFeed},
			{0, 10, 0, CuttingFeed},
		},
	}
	gcode1 := seg1.ToGcode(opt)

	if gcode1 == "" {
		t.Errorf("incorrect gcode1; got empty string")
	}

	seg2 := ToolpathSegment{
		points: []Toolpoint{
			{20, 0, 0, CuttingFeed},
			{21, 1, 0, CuttingFeed},
			{20, 1, 0, CuttingFeed},
			{20, 1, -10, CuttingFeed},
			{20, 10, -10, CuttingFeed},
			{20, 10, 0, CuttingFeed},
		},
	}
	gcode2 := seg2.ToGcode(opt)

	if gcode2 == "" {
		t.Errorf("incorrect gcode2; got empty string")
	}

	path := Toolpath{segments: []ToolpathSegment{seg1, seg2}}
	gcode := path.ToGcode(opt)

	if gcode == "" {
		t.Errorf("incorrect gcode; got empty string")
	}
}
