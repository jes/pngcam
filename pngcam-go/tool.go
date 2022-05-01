package main

import (
    "fmt"
    "math"
)

type Tool interface {
    Radius() float64
    HeightAtRadius(float64) float64
}

type BallEndMill struct { radius float64 }
type FlatEndMill struct { radius float64 }

func NewTool(tooltype string, diameter float64) (Tool, error) {
    if tooltype == "flat" {
        return &FlatEndMill{radius: diameter/2}, nil
    } else if tooltype == "ball" {
        return &BallEndMill{radius: diameter/2}, nil
    } else {
        return nil, fmt.Errorf("unrecognised tool type: %s", tooltype)
    }
}

func (t *BallEndMill) Radius() float64 { return t.radius }
func (t *FlatEndMill) Radius() float64 { return t.radius }

func (t *BallEndMill) HeightAtRadius(r float64) float64 {
    if r > t.radius {
        return math.Inf(1)
    }

    return t.radius - math.Sqrt(t.radius*t.radius - r*r);
}

func (t *FlatEndMill) HeightAtRadius(r float64) float64 {
    if r > t.radius {
        return math.Inf(1)
    }

    return 0
}
