# pngcam

Pngcam takes in a heightmap and gives out Gcode to run a CNC machine.

I wrote a bit about it in https://incoherency.co.uk/blog/stories/cnc-heightmap-toolpaths.html

Pngcam also includes a heightmap rendering program called pngcam-render.

## Building

Dependencies:

 - perl
 - GD

You can install GD on Ubuntu with:

    $ sudo apt install libgd-perl

To build the "semi-fat-packed" Perl scripts:

    $ make

To install it to `/usr/bin/`:

    $ sudo make install

If you want to run it without building, then you can use something like:

    $ PERL5LIB=lib ./pngcam [...]

If you want to use `--write-stock` then you'll need `pngcam-plotter` which is built by the `Makefile`.

## Usage

You'll need to represent your part in a heightmap in a PNG file.
The brightness of a pixel (defined as average of r,g,b) corresponds to the height, such that white is the highest and black
is the lowest.

As an example, let's look at first roughing out a shape with a 6mm end mill, and then move to a 2mm ball-nose end mill to finish up the part.
With both tools we'll get 2 passes over the part: one horizontal, and one vertical.

We'll start with the toolpath for the 6mm end mill. We'll have a maximum step-down of 1mm and step-over of 5mm, at 10000 rpm, and
we'll leave 0.25mm clearance from the final part for the finish pass.

Let's say we want the width of the heightmap to correspond to 100mm in the part, and we want the full brightness range to cover 10mm depth.

    $ pngcam --width 100 --depth 10 --tool-shape flat --tool-diameter 6 --step-down 1 --step-over 5 --speed 10000 --clearance 0.5 heightmap.png > pass1.gcode

And then essentially the same again, but this time with the 2mm ball-nose end mill, with reduced step-over and increased spindle speed.

    $ pngcam --width 100 --depth 10 --tool-shape ball --tool-diameter 2 --step-down 1 --step-over 0.2 --speed 20000 heightmap.png > pass1.gcode

The (0,0,0) point will be at the top left of the input image, with the part existing in the positive X direction and negative Y direction, and
with Z=0 at the top surface of the part (i.e. at "white" in the heightmap).

## Options

    $ pngcam --usage
    Usage: pngcam [options] PNGFILE > GCODEFILE

    This program will read in a heightmap from PNGFILE and write G-code to stdout.

    Tool options:

        --tool-shape flat|ball
            Set the shape of the end mill.
            Default: ball

        --tool-diameter MM
            Set the diameter of the end mill in mm.
            Default: 6

    Tool control options:

        --step-down MM
            Set the maximum step-down in mm. Where the natural toolpath would exceed a cut of this depth, multiple passes are taken instead.
            Default: 100

        --step-over MM
            Set the distance to move the tool over per pass in mm.
            Default: 5

        --step-forward MM
            Set the distance to step forward for each point in the path. If the part contains features that are substantially smaller
            than the step-over, then you can use --step-forward to make sure you don't cut through them.
            Default: takes value from --step-over

        --xy-feed-rate MM/MIN
            Set the maximum feed rate in X/Y plane in mm/min.
            Default: 400

        --z-feed-rate MM/MIN
            Set the maximum feed rate in Z axis in mm/min.
            Default: 50

        --rapid-feed-rate MM/MIN
            Set the maximum feed rate for rapid travel moves in mm/min.
            Default: 10000

        --speed RPM
            Set the spindle speed in RPM.
            Default: 10000

    Path generation options:

        --roughing-only
            Only do the roughing pass (based on --step-down) and do not do the finish pass. This is useful if you
            want to use different parameters, or a different tool, for the roughing pass compared to the finish pass.
            Default: do the finish pass as well as the roughing pass

        --clearance MM
            Set the clearance to leave around the part in mm. Intended so that you can come back again with a finish pass to clean up the part.
            Default: 0

        --rapid-clearance MM
            Set the Z clearance to leave above the part during rapid moves.
            Default: 5

        --route horizontal|vertical|both
            Set whether the tool will move in horizontal lines, vertical lines, or first horizontal followed by vertical.
            Default: both

        --x-offset MM
            Set the offset of X cordinates.
            Default: 0

        --y-offset MM
            Set the offset of Y cordinates.
            Default: 0

        --z-offset MM
            Set the offset of Z cordinates.
            Default: 0

        --ramp-entry
            Add horizontal movements to plunge cuts where possible, to reduce cutting forces.
            Default: plunge straight down

    Heightmap options:

        --width MM
            Set the width of the image in mm. If height is not specified, height will be calculated automatically to maintain aspect ratio. If neither are specified, width=100mm is assumed.
            Default: 100

        --height MM
            Set the height of the image in mm. If width is not specified, width will be calculated automatically to maintain aspect ratio. If neither are specified, width=100mm is assumed.
            Default: N/A

        --depth MM
            Set the total depth of the part in mm.
            Default: 10

        --x-flip
            Flip the image in the X axis. This is useful when you want to cut the same shape on the bottom of a part. The origin will still be at top left of the finished toolpath.

        --y-flip
            Flip the image in the Y axis. This is useful when you want to cut the same shape on the bottom of a part. The origin will still be at top left of the finished toolpath.

        --invert
            Invert the colours in the image, so that white is the deepest cut and black is the shallowest.

        --deep-black
            Let the tool cut below the full depth into black (0,0,0) if this would allow better reproduction of the non-black parts of the heightmap.
            Only really applicable with a ball-nose end mill.
            Default: treat black (0,0,0) as a hard limit on cut depth

        --beyond-edges
            Let the tool cut beyond the edges of the heightmap.
            Default: don't

        --omit-top
            Don't bother cutting top surfaces that are at the upper limit of the heightmap.
            Default: cut them

        --normalise
            Measure the minimum and maximum brightness in the heightmap and stretch all brightness so that the full range of cut depth is achieved.
            Default: no normalisation

        --normalise-ignore-black
            When normalising, ignore black (i.e. stretch all brightnesses apart from black, but leave black alone).
            Note that normalisation is applied before inversion, so if you need this with --invert, you might have to invert the image externally instead.
            Default: don't ignore black

        --imperial
            All units are inches instead of mm, and inches/min instead of mm/min. G-code output has G20 instead of G21.
            Default: not imperial.

        --rgb
            Use R,G,B channels independently to get 24 bits of height data instead of 8.
            Default: greyscale

        --read-stock PNGFILE
            Read stock heightmap from PNGFILE, to save cutting air in roughing passes.
            Default: none

        --write-stock PNGFILE
            Write output heightmap to PNGFILE, to use with --read-stock.
            Default: none

    Cycle time options:

        --max-vel MM/MIN
            Max. velocity in mm/min for cycle time calculation.
            Default: 4000.

        --max-accel MM/SEC^2
            Max. acceleration in mm/sec^2 for cycle time calculation.
            Default: 50.

    Output options:

        --quiet
            Suppress output of dimensions, resolutions, and progress.

## Pngcam-render options

    $ pngcam-render --usage
    Usage: pngcam-render [options] STLFILE

    This program will read in the STLFILE and render it to a heightmap.

    Options:

        --border PX
            Draw a border around the part.
            Default: 32

        --width PX
            Set the width of the part in pixels. If height is not specified, height will be calculated
            automatically to maintain aspect ratio. If neither are specified, width=400px is assumed.
            The output image will be this wide, plus a border on both sides.
            Default: 400

        --height PX
            Set the height of the part in pixels. If width is not specified, width will be calculated
            automatically to maintain aspect ratio. If neither are specified, width=400px is assumed.
            The output image will be this talg, plus a border on both sides.
            Default: N/A

        --bottom
            View from the bottom, as if the part were rotated through 180 degrees around the Y axis.
            Default: viewed from the top

        --png PNGFILE
            Set the name of the output file. If none is give, this will just be the STL file with ".png" appended.
            Default: STLFILE.png

        --rgb
            Use R,G,B channels independently to get 24 bits of height data instead of 8.
            Default: greyscale

        --quiet
            Suppress output of dimensions, resolutions, and progress.

## Tests

To run tests, either:

    $ prove -l t/

or

    $ make test

If a test fails you might want to try diffing the old (expected) and new versions of the G-code files to work out what went wrong.

## Contact

Pngcam is a program by James Stanley. You can email me at james@incoherency.co.uk or read my blog at https://incoherency.co.uk/
