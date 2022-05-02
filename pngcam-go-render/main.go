package main

import (
	"flag"
	"fmt"
	"os"
	"runtime/pprof"
)

func main() {
	width := flag.Int("width", 400, "Set the width of the part in pixels.")
	height := flag.Int("height", 400, "Set the height of the part in pixels.")
	png := flag.String("png", "", "Output PNG filename.")
	quiet := flag.Bool("quiet", false, "Suppress output of dimensions, resolutions, and progress.")
	cpuProfile := flag.String("cpuprofile", "", "Write CPU profile to file.")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "Pngcam-render is a program by James Stanley. You can email me at james@incoherency.co.uk or read my blog at https://incoherency.co.uk/\n")
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

	args := flag.Args()
	if len(args) != 1 {
		fmt.Fprintf(os.Stderr, "usage: pngcam-render STLFILE\n")
		os.Exit(1)
	}
	stlFile := args[0]

	if *png == "" {
		*png = stlFile + ".png"
	}

	opt := Options{
		width:   *width,
		height:  *height,
		quiet:   *quiet,
		stlFile: stlFile,
		pngFile: *png,
	}

	renderer, err := NewRenderer(&opt)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	renderer.Render()
}
