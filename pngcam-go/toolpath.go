package main

import (
	"fmt"
	"math"
	"strings"
)

type FeedType int

const (
	RapidFeed FeedType = iota
	CuttingFeed
)

type Toolpoint struct {
	x    float64
	y    float64
	z    float64
	feed FeedType
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

func (seg *ToolpathSegment) AppendSegment(more *ToolpathSegment) {
	for i := range more.points {
		seg.Append(more.points[i])
	}
}

func (seg *ToolpathSegment) Simplified() ToolpathSegment {
	newseg := NewToolpathSegment()

	if len(seg.points) == 0 {
		return newseg
	}

	newseg.Append(seg.points[0])

	if len(seg.points) == 1 {
		return newseg
	}

	epsilon := 0.00001

	prev := seg.points[1]

	for i := 2; i < len(seg.points); i++ {
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

	return newseg
}

func (seg *ToolpathSegment) Reversed() ToolpathSegment {
	newseg := NewToolpathSegment()

	for i := len(seg.points) - 1; i >= 0; i-- {
		newseg.points = append(newseg.points, seg.points[i])
	}

	return newseg
}

func (seg *ToolpathSegment) ToGcode(opt Options) string {
	gcode := strings.Builder{}

	// TODO: make the rotary axis name configurable
	yAxisName := "Y"
	if opt.rotary {
		yAxisName = "A"
	}

	for i := range seg.points {
		p := seg.points[i]
		feedRate := opt.rapidFeed
		if p.feed == CuttingFeed && i > 0 {
			feedRate = opt.FeedRate(seg.points[i-1], p)
		}
		fmt.Fprintf(&gcode, "G1 X%.04f %s%.04f Z%.04f F%g\n", p.x+opt.xOffset, yAxisName, p.y+opt.yOffset, p.z+opt.zOffset, feedRate)
	}

	return gcode.String()
}

func (seg *ToolpathSegment) OmitTop() *Toolpath {
	tp := NewToolpath()

	newseg := NewToolpathSegment()

	// XXX: why does this need to be so large? is it because we're not always
	// sampling the cutter in the very centre, so sometimes we think we can cut
	// to z=-0.005 even when it should be exactly 0?
	epsilon := 0.01

	for i := range seg.points {
		if seg.points[i].z > -epsilon {
			tp.Append(newseg)
			newseg = NewToolpathSegment()
		} else {
			newseg.Append(seg.points[i])
		}
	}

	tp.Append(newseg)

	return &tp
}

func (seg *ToolpathSegment) RampEntry() ToolpathSegment {
	if len(seg.points) <= 2 {
		return *seg
	}

	newseg := NewToolpathSegment()

	// when a toolpoint moves down in Z, at more than 30 degrees, ramp it along a straight line going along subsequent
	// segments; range for line can be found by walking along segments that are in a straight line, until we reach a Z
	// point that is halfway between current Z and target Z

	maxPlungeAngle := 30 * math.Pi / 180 // radians from horizontal
	minRampDistance := 0.01              // avoid dividing by 0

	for i := 1; i < len(seg.points)-1; i++ {
		last := seg.points[i-1]
		p := seg.points[i]
		next := seg.points[i+1]

		// don't ramp on rapids
		if p.feed == RapidFeed {
			newseg.Append(p)
			continue
		}

		dxLast := p.x - last.x
		dyLast := p.y - last.y
		dzLast := p.z - last.z
		dxyLast := math.Sqrt(dxLast*dxLast + dyLast*dyLast)

		plungeAngle := math.Atan2(-dzLast, dxyLast)
		if plungeAngle < maxPlungeAngle { // already within allowable range
			newseg.Append(p)
			continue
		}

		// TODO: when the next-next moves are in the same xy direction (but,
		// for example, different Z) then we can consider it as well to see if
		// it provides clearance for a longer ramp
		dxNext := next.x - p.x
		dyNext := next.y - p.y
		dzNext := next.z - p.z
		dxyNext := math.Sqrt(dxNext*dxNext + dyNext*dyNext)

		if dxyNext < minRampDistance { // not enough room
			newseg.Append(p)
			continue
		}

		// now we need to replace p with 2 horizontal moves going downwards at
		// maxPlungeAngle; the first move goes in the direction of p => next,
		// and the second one goes in the opposite direction, to land at p;
		// if there is not enough horizontal distance between p and next then
		// we just ramp in whatever distance there is and accept the overly-
		// steep angle (should we instead do multiple ramps?)

		// the height we need to go down is dzLast, and we're starting from
		// last so the first leg goes down to z=p.z-dzLast/2 and the second leg
		// ends up at p, so we'll just leave p unchanged for that

		// how steep is the next leg of the toolpath?
		availableRampAngle := math.Atan2(dzNext, dxyNext)

		// how steep would our ramp need to be to finish before passing the next point?
		impliedAvailableRampAngle := math.Atan2(-dzLast/2, dxyNext)

		// use whichever limit forces the ramp angle to be steepest, so that
		// we don't exceed any limit
		rampAngle := maxPlungeAngle
		if availableRampAngle > rampAngle {
			rampAngle = availableRampAngle
		}
		if impliedAvailableRampAngle > rampAngle {
			rampAngle = impliedAvailableRampAngle
		}

		dxyRamp := -(dzLast / 2) / math.Tan(rampAngle)
		k := dxyRamp / dxyNext
		dxRamp := k * dxNext
		dyRamp := k * dyNext

		// if splitting this move into 2 ramps causes the second ramp to be steeper
		// than the original move, then just keep the original move instead
		plungeAngle2 := math.Atan2(-dzLast/2, math.Abs(dxyRamp)-dxyLast)
		if plungeAngle2 > plungeAngle {
			newseg.Append(p)
			continue
		}

		newseg.Append(Toolpoint{last.x + dxRamp, last.y + dyRamp, p.z - dzLast/2, CuttingFeed})
		newseg.Append(p)
	}

	newseg.Append(seg.points[len(seg.points)-1])

	return newseg
}

func (seg *ToolpathSegment) CycleTime(opt Options) float64 {
	cycleTime := 0.0

	for i := 1; i < len(seg.points); i++ {
		dx := seg.points[i].x - seg.points[i-1].x
		dy := seg.points[i].y - seg.points[i-1].y
		dz := seg.points[i].z - seg.points[i-1].z
		dist := math.Sqrt(dx*dx + dy*dy + dz*dz)

		feedRate := opt.rapidFeed
		if seg.points[i].feed == CuttingFeed {
			feedRate = opt.FeedRate(seg.points[i-1], seg.points[i])
		}
		if feedRate > opt.maxVel {
			feedRate = opt.maxVel
		}

		// TODO: take opt.maxAccel into account
		// TODO: when cycle time calculation is better, remove the factor of 10 in job.CombineSegments()

		cycleTime += 60 * (dist / feedRate)
	}

	return cycleTime
}

func (tp *Toolpath) Simplified() *Toolpath {
	newtp := NewToolpath()

	for i := range tp.segments {
		newtp.Append(tp.segments[i].Simplified())
	}

	return &newtp
}

func (tp *Toolpath) RampEntry(opt Options) *Toolpath {
	newtp := NewToolpath()

	newtp.Append(tp.AsOneSegment(opt).RampEntry())

	return &newtp
}

func (tp *Toolpath) Sorted() *Toolpath {
	newtp := NewToolpath()

	needsegs := make(map[int]*ToolpathSegment)

	last := Toolpoint{0, 0, 0, RapidFeed}
	gotFirstPoint := false

	// take a copy of every segment we need
	for i := range tp.segments {
		if len(tp.segments[i].points) > 0 {
			if !gotFirstPoint {
				last = tp.segments[i].points[0]
				gotFirstPoint = true
			}
			needsegs[i] = &tp.segments[i]
		}
	}

	// grab the segment which starts nearest to the end point of the last
	// segment, move it into our new toolpath, repeat until done
	for len(needsegs) > 0 {
		minDist := math.Inf(1)
		minIdx := 0
		minReversed := false

		for i, _ := range needsegs {
			seg := needsegs[i]
			dx := seg.points[0].x - last.x
			dy := seg.points[0].y - last.y
			dz := seg.points[0].z - last.z
			dist := math.Sqrt(dx*dx + dy*dy + dz*dz)
			if dist < minDist {
				minDist = dist
				minIdx = i
				minReversed = false
			}

			// try the same segment again, but in reverse
			n := len(seg.points) - 1
			dx = seg.points[n].x - last.x
			dy = seg.points[n].y - last.y
			dz = seg.points[n].z - last.z
			dist = math.Sqrt(dx*dx + dy*dy + dz*dz)
			if dist < minDist {
				minDist = dist
				minIdx = i
				minReversed = true
			}
		}

		minSeg := needsegs[minIdx]
		if minReversed {
			last = minSeg.points[0]
			newtp.Append(minSeg.Reversed())
		} else {
			last = minSeg.points[len(minSeg.points)-1]
			newtp.Append(*minSeg)
		}

		delete(needsegs, minIdx)
	}

	return &newtp
}

func (tp *Toolpath) Append(seg ToolpathSegment) {
	tp.segments = append(tp.segments, seg)
}

func (tp *Toolpath) AppendToolpath(more *Toolpath) {
	for i := range more.segments {
		tp.Append(more.segments[i])
	}
}

func (tp *Toolpath) AsOneSegment(opt Options) *ToolpathSegment {
	seg := NewToolpathSegment()

	if len(tp.segments) == 0 {
		return &seg
	}

	// TODO: use RapidPath()
	for i := range tp.segments {
		if len(tp.segments[i].points) == 0 {
			continue
		}

		p0 := tp.segments[i].points[0]
		pLast := tp.segments[i].points[len(tp.segments[i].points)-1]

		// move to the start point of this segment
		seg.Append(Toolpoint{p0.x, p0.y, opt.safeZ, RapidFeed})

		// rapid down to stepDown above start height?
		if p0.z+opt.stepDown < opt.safeZ {
			seg.Append(Toolpoint{p0.x, p0.y, p0.z + opt.stepDown, RapidFeed})
		}

		// move through the rest of the segment
		seg.AppendSegment(&tp.segments[i])

		// back up to safe Z
		seg.Append(Toolpoint{pLast.x, pLast.y, opt.safeZ, RapidFeed})
	}

	return &seg
}

func (tp *Toolpath) RapidPath(a, b Toolpoint, opt Options) ToolpathSegment {
	seg := NewToolpathSegment()

	// move up to safe Z
	seg.Append(Toolpoint{a.x, a.y, opt.safeZ, RapidFeed})

	// move above next point
	seg.Append(Toolpoint{b.x, b.y, opt.safeZ, RapidFeed})

	// rapid down to safe Z above?
	if b.z+opt.stepDown < opt.safeZ {
		seg.Append(Toolpoint{b.x, b.y, b.z + opt.stepDown, RapidFeed})
	}

	return seg
}

func (tp *Toolpath) ToGcode(opt Options) string {
	return tp.AsOneSegment(opt).ToGcode(opt)
}

func (tp *Toolpath) CycleTime(opt Options) float64 {
	return tp.AsOneSegment(opt).CycleTime(opt)
}
