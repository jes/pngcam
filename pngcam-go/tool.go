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
	LengthToIntersection(float64, float64, float64) float64
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
func (t *BallEndMill) LengthToIntersection(xOffset float64, angle float64, z float64) float64 {
	// how much does the tool radius shrink by due to the distance from centre line in x axis?
	radiusChange := t.radius - math.Sqrt(t.radius*t.radius-xOffset*xOffset)

	// what is the length of the line segment from the origin, at angle "angle", that intersects with a circle whose bottom is at z=z?
	// sine law: a/sinA = b/sinB = c/sinC
	// we have a triange with angle A = abs(angle), side length a = tool.radius, side length b = z+tool.radius
	A := math.Abs(angle * math.Pi / 180.0)
	a := t.radius - radiusChange
	b := z + t.radius + radiusChange
	// we want to know side length c
	// first use the sine law to find angle B
	// b/sinB = a/sinA, sinB = b.sin(A)/a, B = asin(b.sin(A)/a)
	B := math.Asin(b * math.Sin(A) / a)
	if math.IsNaN(B) {
		// tool can not touch workpiece at this angle
		return math.NaN()
	}
	B = math.Pi - B // of the 2 possible solutions, we want the larger angle for B
	// now we know that the angles add up to 180 degrees (pi radians), find angle C
	C := math.Pi - (B + A)
	// now use the sine law to find side length c
	// c/sinC = a/sinA, c = a.sin(C)/sin(A)
	c := a * math.Sin(C) / math.Sin(A)

	return c
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
func (t *FlatEndMill) LengthToIntersection(xOffset float64, angle float64, z float64) float64 {
	h := z / math.Cos(angle*math.Pi/180.0)
	yOffset := h * math.Sin(angle*math.Pi/180.0)
	rSqr := xOffset*xOffset + yOffset*yOffset
	if rSqr > t.radius*t.radius {
		return math.NaN()
	}
	return h
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
func (t *VBit) LengthToIntersection(xOffset float64, angle float64, z float64) float64 {
	return 0
}
