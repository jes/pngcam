package main

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

type Tool interface {
	Radius() float64
	HeightAtRadius(float64) float64
	HeightAtRadiusSqr(float64) float64
}

type BallEndMill struct{ radius float64 }
type FlatEndMill struct{ radius float64 }
type VBit struct {
	radius float64
	angle  float64 // included angle in degrees
}

func NewTool(tooltype string, diameter float64) (Tool, error) {
	if tooltype == "flat" {
		return &FlatEndMill{radius: diameter / 2}, nil
	} else if tooltype == "ball" {
		return &BallEndMill{radius: diameter / 2}, nil
	} else if strings.HasPrefix(tooltype, "vbit") {
		angle, err := strconv.ParseFloat(tooltype[4:], 64)
		if err != nil {
			return nil, err
		}
		return &VBit{radius: diameter / 2, angle: angle}, nil
	} else {
		return nil, fmt.Errorf("unrecognised tool type: %s", tooltype)
	}
}

func (t *BallEndMill) Radius() float64 { return t.radius }
func (t *FlatEndMill) Radius() float64 { return t.radius }
func (t *VBit) Radius() float64        { return t.radius }

func (t *BallEndMill) HeightAtRadius(r float64) float64 {
	return t.HeightAtRadiusSqr(r * r)
}
func (t *BallEndMill) HeightAtRadiusSqr(rSqr float64) float64 {
	if rSqr > t.radius*t.radius {
		return math.Inf(1)
	}

	return t.radius - math.Sqrt(t.radius*t.radius-rSqr)
}

func (t *FlatEndMill) HeightAtRadius(r float64) float64 {
	return t.HeightAtRadiusSqr(r * r)
}
func (t *FlatEndMill) HeightAtRadiusSqr(rSqr float64) float64 {
	if rSqr > t.radius*t.radius {
		return math.Inf(1)
	}

	return 0
}

func (t *VBit) HeightAtRadius(r float64) float64 {
	if r > t.radius {
		return math.Inf(1)
	}

	return r / math.Tan((t.angle/2)*math.Pi/180)
}
func (t *VBit) HeightAtRadiusSqr(rSqr float64) float64 {
	return t.HeightAtRadius(math.Sqrt(rSqr))
}
