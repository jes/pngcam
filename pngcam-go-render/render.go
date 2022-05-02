package main

import (
	"fmt"
	"github.com/hschendel/stl"
	"math"
	"os"
)

type Renderer struct {
	options   Options
	mesh      *stl.Solid
	mmWidth   float32
	mmHeight  float32
	mmDepth   float32
	heightmap *Heightmap
}

const (
	X = 0
	Y = 1
	Z = 2
)

func NewRenderer(opt *Options) (*Renderer, error) {
	r := Renderer{}
	r.options = *opt

	solid, err := stl.ReadFile(opt.stlFile)
	if err != nil {
		return nil, err
	}
	r.mesh = solid

	r.ProcessMesh()

	if !opt.quiet {
		fmt.Fprintf(os.Stderr, "%dx%d px depth map. %gx%g mm work piece.\n", opt.width, opt.height, r.mmWidth, r.mmHeight)
		fmt.Fprintf(os.Stderr, "Work piece is %g tall in Z axis.\n", r.mmDepth)
		fmt.Fprintf(os.Stderr, "X resolution is %g px/mm. Y resolution is %g px/mm.\n", float32(opt.width)/r.mmWidth, float32(opt.height)/r.mmHeight)
	}

	r.heightmap = NewHeightmap(opt)

	return &r, nil
}

func (r *Renderer) ProcessMesh() {
	// rotate to the required side
	if r.options.bottom {
		r.mesh.Rotate(stl.Vec3{0, 0, 0}, stl.Vec3{0, 1, 0}, stl.Pi)
	}

	var min, max stl.Vec3
	min[X] = float32(math.Inf(1))
	min[Y] = float32(math.Inf(1))
	min[Z] = float32(math.Inf(1))
	max[X] = float32(math.Inf(-1))
	max[Y] = float32(math.Inf(-1))
	max[Z] = float32(math.Inf(-1))

	for i := range r.mesh.Triangles {
		t := r.mesh.Triangles[i]
		for j := range t.Vertices {
			v := t.Vertices[j]
			if v[X] < min[X] {
				min[X] = v[X]
			}
			if v[Y] < min[Y] {
				min[Y] = v[Y]
			}
			if v[Z] < min[Z] {
				min[Z] = v[Z]
			}
			if v[X] > max[X] {
				max[X] = v[X]
			}
			if v[Y] > max[Y] {
				max[Y] = v[Y]
			}
			if v[Z] > max[Z] {
				max[Z] = v[Z]
			}
		}
	}

	r.mmWidth = max[X] - min[X]
	r.mmHeight = max[Y] - min[Y]
	r.mmDepth = max[Z] - min[Z]

	// translate to origin
	min[X] = -min[X]
	min[Y] = -min[Y]
	min[Z] = -min[Z]
	r.mesh.Translate(min)
}

func (r *Renderer) Render() {
	for i := range r.mesh.Triangles {
		t := r.mesh.Triangles[i]
		r.heightmap.DrawTriangle(r.MmToPx(t.Vertices[0]), r.MmToPx(t.Vertices[1]), r.MmToPx(t.Vertices[2]))
	}
	err := r.heightmap.WritePNG(r.options.pngFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "write %s: %v\n", r.options.pngFile, err)
	}
}

func (r *Renderer) MmToPx(v stl.Vec3) [3]float32 {
	var vNew [3]float32
	vNew[X] = v[X] * float32(r.options.width) / r.mmWidth
	vNew[Y] = float32(r.options.height-1) - v[Y]*float32(r.options.height)/r.mmHeight
	vNew[Z] = v[Z] / r.mmDepth
	return vNew
}
