package main

import (
	"flag"
	"fmt"
	"os"
	"runtime/pprof"
)

func main() {
	toolShape := flag.String("tool-shape", "ball", "Set the shape of the end mill.")
	toolDiameter := flag.Float64("tool-diameter", 6, "Set the diameter of the end mill in mm.")

	stepDown := flag.Float64("step-down", 100, "Set the maximum step-down in mm. Where the natural toolpath would exceed a cut of this depth, multiple passes are taken instead.")
	stepOver := flag.Float64("step-over", 5, "Set the distance to move the tool over per pass in mm.")
	xyFeed := flag.Float64("xy-feed-rate", 400, "Set the maximum feed rate in X/Y plane in mm/min.")
	zFeed := flag.Float64("z-feed-rate", 50, "Set the maximum feed rate in Z axis in mm/min.")
	rapidFeed := flag.Float64("rapid-feed-rate", 10000, "Set the maximum feed rate for rapid travel moves in mm/min.")
	rpm := flag.Float64("speed", 10000, "Set the spindle speed in RPM.")

	roughingOnly := flag.Bool("roughing-only", false, "Only do the roughing pass (based on --step-down) and do not do the finish pass. This is useful if you want to use different parameters, or a different tool, for the roughing pass comapred to the finish pass.")
	clearance := flag.Float64("clearance", 0, "Set the clearance to leave around the part in mm. Intended so that you can come back again with a finish pass to clean up the part.")
	safeZ := flag.Float64("rapid-clearance", 5, "Set the Z clearance to leave above the part during rapid moves.")
	route := flag.String("route", "horizontal", "Set whether the tool will move in horizontal or vertical lines.")
	xOffset := flag.Float64("x-offset", 0, "Set the offset to add to X coordinates.")
	yOffset := flag.Float64("y-offset", 0, "Set the offset to add to Y coordinates.")
	zOffset := flag.Float64("z-offset", 0, "Set the offset to add to Z coordinates.")
	rampEntry := flag.Bool("ramp-entry", false, "Add horizontal movements to plunge cuts where possible, to reduce cutting forces.")

	width := flag.Float64("width", 100, "Set the width of the image in mm.")
	height := flag.Float64("height", 100, "Set the height of the image in mm.")
	depth := flag.Float64("depth", 10, "Set the total depth of the part in mm.")
	diameter := flag.Float64("diameter", 0, "Set the diameter of the part for rotary carving.")
	rotary := flag.Bool("rotary", false, "Rotary carving.")

	cutBelowBottom := flag.Bool("deep-black", false, "Let the tool cut below the full depth if this would allow better reproduction of the non-black parts of the heightmap. Only really applicable with a ball-nose end mill.")
	cutBeyondEdges := flag.Bool("beyond-edges", false, "Let the tool cut beyond the edges of the heightmap.")
	omitTop := flag.Bool("omit-top", false, "Don't bother cutting top surfaces that are at the upper limit of the heightmap.")
	imperial := flag.Bool("imperial", false, "All units in inches instead of mm, and inches/min instead of mm/min. G-code output has G20 instead of G21.")

	readStockPath := flag.String("read-stock", "", "Read stock heightmap from PNG file, to save cutting air in roughing passes.")
	writeStockPath := flag.String("write-stock", "", "Write output heightmap to PNG file, to use with --read-stock.")

	maxVel := flag.Float64("max-vel", 4000, "Max. velocity in mm/min for cycle time estimation.")
	maxAccel := flag.Float64("max-accel", 50, "Max. acceleration in mm/sec^2 for cycle time estimation.")

	quiet := flag.Bool("quiet", false, "Suppress output of dimensions, resolutions, and progress.")

	cpuProfile := flag.String("cpuprofile", "", "Write CPU profile to file.")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "Pngcam is a program by James Stanley. You can email me at james@incoherency.co.uk or read my blog at https://incoherency.co.uk/\n")
	}

	flag.Parse()

	if *cpuProfile != "" {
		f, err := os.Create(*cpuProfile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	tool, err := NewTool(*toolShape, *toolDiameter)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	dir := Horizontal
	if *route == "vertical" {
		dir = Vertical
	} else if *route != "horizontal" {
		fmt.Fprintf(os.Stderr, "unrecognised route: %s\n", *route)
		os.Exit(1)
	}

	args := flag.Args()
	if len(args) != 1 {
		fmt.Fprintf(os.Stderr, "usage: pngcam HEIGHTMAPFILE\n")
		os.Exit(1)
	}
	heightmapPath := args[0]

	if *diameter != 0 {
		if !*rotary {
			fmt.Fprintf(os.Stderr, "can't use diameter in non-rotary mode")
			os.Exit(1)
		}
		*depth = *diameter / 2.0
	}

	if *rotary {
		// rotary parts are always 360 degrees around (should this be configurable? e.g. to allow partial rotation?)
		*height = 360.0
		*safeZ += *depth
	}

	opt := Options{
		heightmapPath:  heightmapPath,
		readStockPath:  *readStockPath,
		writeStockPath: *writeStockPath,

		safeZ:     *safeZ,
		rapidFeed: *rapidFeed,
		xyFeed:    *xyFeed,
		zFeed:     *zFeed,
		rpm:       *rpm,

		width:  *width,
		height: *height,
		depth:  *depth,
		rotary: *rotary,

		direction: dir,

		stepOver: *stepOver,
		stepDown: *stepDown,

		tool: tool,

		stockToLeave: *clearance,

		roughingOnly:   *roughingOnly,
		omitTop:        *omitTop,
		rampEntry:      *rampEntry,
		cutBelowBottom: *cutBelowBottom,
		cutBeyondEdges: *cutBeyondEdges,

		imperial: *imperial,

		xOffset: *xOffset,
		yOffset: *yOffset,
		zOffset: *zOffset,

		maxVel:   *maxVel,
		maxAccel: *maxAccel,

		quiet: *quiet,
	}

	job, err := NewJob(&opt)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	os.Stdout.WriteString(job.Gcode())
}
