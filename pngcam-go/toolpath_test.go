package main

import (
	"testing"
)

func nonEmptySegments(tp *Toolpath) []ToolpathSegment {
	segs := []ToolpathSegment{}
	for _, seg := range tp.segments {
		if len(seg.points) > 0 {
			segs = append(segs, seg)
		}
	}
	return segs
}

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

func TestOmitTopAndBottom(t *testing.T) {
	seg := ToolpathSegment{
		points: []Toolpoint{
			{0, 0, 0, CuttingFeed},
			{1, 0, -1, CuttingFeed},
			{2, 0, -10, CuttingFeed},
			{3, 0, -5, CuttingFeed},
			{4, 0, 0, CuttingFeed},
		},
	}

	optTop := &Options{omitTop: true, depth: 10}
	gotTop := seg.OmitTopAndBottom(optTop)
	gotTopSegs := nonEmptySegments(gotTop)
	if len(gotTopSegs) != 1 || len(gotTopSegs[0].points) != 3 {
		t.Fatalf("omit-top should keep one 3-point segment, got %#v", gotTopSegs)
	}

	optBottom := &Options{omitBottom: true, depth: 10}
	gotBottom := seg.OmitTopAndBottom(optBottom)
	gotBottomSegs := nonEmptySegments(gotBottom)
	if len(gotBottomSegs) != 2 {
		t.Fatalf("omit-bottom should split into two segments, got %d", len(gotBottomSegs))
	}
	if len(gotBottomSegs[0].points) != 2 || len(gotBottomSegs[1].points) != 2 {
		t.Fatalf("omit-bottom should keep two 2-point segments, got %#v", gotBottomSegs)
	}

	optBoth := &Options{omitTop: true, omitBottom: true, depth: 10}
	gotBoth := seg.OmitTopAndBottom(optBoth)
	gotBothSegs := nonEmptySegments(gotBoth)
	if len(gotBothSegs) != 2 {
		t.Fatalf("omit-top+omit-bottom should split into two segments, got %d", len(gotBothSegs))
	}
	if len(gotBothSegs[0].points) != 1 || len(gotBothSegs[1].points) != 1 {
		t.Fatalf("omit-top+omit-bottom should keep two single-point segments, got %#v", gotBothSegs)
	}
}
