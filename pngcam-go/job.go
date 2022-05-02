package main

import (
	"fmt"
	"math"
	"os"
	"strings"
)

type Job struct {
	options      *Options
	toolpoints   *ToolpointsMap
	readStock    *ToolpointsMap
	writeStock   *ToolpointsMap
	mainToolpath Toolpath
}

func NewJob(opt *Options) (*Job, error) {
	j := Job{}
	j.options = opt

	hm, err := OpenHeightmapImage(opt.heightmapPath, opt)
	if err != nil {
		return nil, err
	}

	opt.x_MmPerPx = opt.width / float64(hm.img.Bounds().Max.X)
	opt.y_MmPerPx = opt.height / float64(hm.img.Bounds().Max.Y)
	opt.widthPx = hm.img.Bounds().Max.X
	opt.heightPx = hm.img.Bounds().Max.Y

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

	if !opt.quiet {
		unit := "mm"
		if opt.imperial {
			unit = "inches"
		}
		fmt.Fprintf(os.Stderr, "%dx%d px height map. %gx%g %s work piece.\n", opt.widthPx, opt.heightPx, opt.width, opt.height, unit)
		fmt.Fprintf(os.Stderr, "X resolution is %g px/%s. Y resolution is %g px/%s.\n", 1/opt.x_MmPerPx, unit, 1/opt.y_MmPerPx, unit)
		fmt.Fprintf(os.Stderr, "Step-over is %g %s = %g px in X and %g px in Y.\n", opt.stepOver, unit, opt.stepOver/opt.x_MmPerPx, opt.stepOver/opt.y_MmPerPx)
	}

	j.MakeToolpath()

	return &j, nil
}

func (j *Job) MakeToolpath() {
	j.mainToolpath = NewToolpath()

	opt := j.options

	xLimit := opt.width
	yLimit := opt.height

	xStep := opt.x_MmPerPx
	yStep := 0.0
	if opt.direction == Vertical {
		xStep = 0.0
		yStep = opt.y_MmPerPx
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

	if !opt.quiet {
		fmt.Fprintf(os.Stderr, "Generating path: 0%%")
	}

	for x >= zero && y >= zero && x < xLimit && y < yLimit {
		seg := NewToolpathSegment()

		// TODO: use CutPath() instead of this weird dual-loop thing
		for x >= zero && y >= zero && x < xLimit && y < yLimit {
			seg.Append(Toolpoint{x, y, j.toolpoints.GetMm(x, y), CuttingFeed})

			x += xStep
			y += yStep
		}

		if opt.omitTop {
			j.mainToolpath.AppendToolpath(seg.OmitTop().Simplified())
		} else {
			j.mainToolpath.Append(seg.Simplified())
		}

		pct := 0.0
		if opt.direction == Horizontal {
			y += opt.stepOver
			pct = float64(100*(y-zero)) / (yLimit - zero)
		} else {
			x += opt.stepOver
			pct = float64(100*(x-zero)) / (xLimit - zero)
		}

		xStep = -xStep
		yStep = -yStep
		x += xStep
		y += yStep

		if !opt.quiet {
			fmt.Fprintf(os.Stderr, "   \rGenerating path: %.0f%%", pct)
		}
	}

	if !opt.quiet {
		fmt.Fprintf(os.Stderr, "   \rGenerating path: done\n")
	}
}

func (j *Job) Gcode() string {
	opt := j.options

	path := j.Roughing()

	if !opt.roughingOnly {
		path.AppendToolpath(j.Finishing())
	}

	if opt.rampEntry {
		path = path.RampEntry(*opt)
	}

	gcode := path.ToGcode(*opt)
	cycleTime := path.CycleTime(*opt)

	if j.writeStock != nil {
		j.writeStock.PlotToolpath(path)
		err := j.writeStock.WritePNG(opt.writeStockPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "write %s: %v\n", opt.writeStockPath, err)
		}
	}

	if !opt.quiet {
		fmt.Fprintf(os.Stderr, "Cycle time estimate: %g secs\n", cycleTime)
	}

	return j.Preamble() + gcode + j.Postamble()
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

	fmt.Fprintf(&gcode, "G1 Z%.04f F%g\n", opt.safeZ+opt.zOffset, opt.rapidFeed)

	return gcode.String()
}

func (j *Job) Postamble() string {
	return "M5\nM2\n" // stop spindle, end program
}

func (j *Job) Finishing() *Toolpath {
	return j.CombineSegments(j.mainToolpath.Simplified().Sorted())
}

func (j *Job) Roughing() *Toolpath {
	opt := j.options

	deepest := -opt.depth
	if opt.cutBelowBottom {
		deepest -= opt.tool.Radius()
	}

	path := NewToolpath()

	for z := -opt.stepDown; z > deepest; z -= opt.stepDown {
		path.AppendToolpath(j.RoughingLevel(z).Simplified().Sorted())
	}

	return &path
}

func (j *Job) RoughingLevel(z float64) *Toolpath {
	path := NewToolpath()

	for i := range j.mainToolpath.segments {
		seg := NewToolpathSegment()
		for p := range j.mainToolpath.segments[i].points {
			tp := j.mainToolpath.segments[i].points[p]
			if tp.z < z && (j.readStock == nil || z < j.readStock.GetMm(tp.x, tp.y)) {
				// add this point to this roughing segment
				seg.Append(Toolpoint{tp.x, tp.y, z, CuttingFeed})
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

	return j.CombineSegments(path.Sorted())
}

func (j *Job) CombineSegments(tp *Toolpath) *Toolpath {
	opt := j.options

	if len(tp.segments) <= 1 {
		return tp
	}

	newtp := NewToolpath()

	// TODO: what happens when there are 0-length segments?
	// TODO: if opt.omitTop, do we need to take pains to omit top?

	seg := tp.segments[0]

	for i := 1; i < len(tp.segments); i++ {
		prev := seg.points[len(seg.points)-1]
		cur := tp.segments[i].points[0]

		rapidPath := tp.RapidPath(prev, cur, *opt)
		deepestZ := prev.z
		if cur.z < deepestZ {
			deepestZ = cur.z
		}
		cutPath := j.CutPath(prev, cur, deepestZ)

		// as well as a straight line from prev to cur, try axis-aligned lines
		// in x-first and y-first configuration
		xCur := Toolpoint{x: cur.x, y: prev.y, z: math.Max(deepestZ, j.toolpoints.GetMm(cur.x, prev.y))}
		yCur := Toolpoint{x: prev.x, y: cur.y, z: math.Max(deepestZ, j.toolpoints.GetMm(prev.x, cur.y))}
		xYCutPath := j.CutPath(prev, xCur, deepestZ)
		xYCutPath2 := j.CutPath(xCur, cur, deepestZ)
		xYCutPath.AppendSegment(&xYCutPath2)
		yXCutPath := j.CutPath(prev, yCur, deepestZ)
		yXCutPath2 := j.CutPath(yCur, cur, deepestZ)
		yXCutPath.AppendSegment(&yXCutPath2)

		if xYCutPath.CycleTime(*opt) < cutPath.CycleTime(*opt) {
			cutPath = xYCutPath
		}
		if yXCutPath.CycleTime(*opt) < cutPath.CycleTime(*opt) {
			cutPath = yXCutPath
		}

		// when we have a cutting path that is faster than the rapid path, use it instead
		// TODO: when cycle time estimates are more accurate, lose the factor of 10
		if cutPath.CycleTime(*opt) < 10*rapidPath.CycleTime(*opt) {
			seg.AppendSegment(&cutPath)
		} else {
			newtp.Append(seg)
			seg = NewToolpathSegment()
		}
		seg.AppendSegment(&tp.segments[i])
	}

	if len(seg.points) > 0 {
		newtp.Append(seg)
	}

	return &newtp
}

func (j *Job) CutPath(a, b Toolpoint, deepestZ float64) ToolpathSegment {
	x := a.x
	y := a.y

	dx := b.x - a.x
	dy := b.y - a.y
	dist := math.Sqrt(dx*dx + dy*dy)

	if dist > 2*j.options.stepOver {
		// on long travels, we would take unsightly gouges out of the surface
		// pattern, due to the stepOver, even though we technically keep the
		// tool on the surface of the model; to mitigate this, we lift the tool
		// up by the "nominal deviation" on long travel moves
		r1 := j.options.tool.Radius()
		r2 := j.options.stepOver / 2
		deepestZ += math.Sqrt(r1*r1 - r2*r2)
	}

	dx /= dist
	dy /= dist

	seg := NewToolpathSegment()

	// TODO: allow rapid through the middle of material that we've already cut,
	// using the writeStock.GetMm() to decide?

	// TODO: might be wrong if x_MmPerPx is substantially different to y_MmPerPx
	for k := 0.0; k <= dist; k += j.options.x_MmPerPx {
		x = a.x + k*dx
		y = a.y + k*dy

		z := j.toolpoints.GetMm(x, y)
		if z < deepestZ {
			z = deepestZ
		}

		seg.Append(Toolpoint{x, y, z, CuttingFeed})
	}

	return seg
}
