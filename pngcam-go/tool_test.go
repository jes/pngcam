package main

import (
    "math"
    "testing"
)

func TestBall(t *testing.T) {
    tool, err := NewTool("ball", 10)
    if err != nil {
        t.Errorf("can't create ball tool: %v", err)
    }

    if tool.Radius() != 5 {
        t.Errorf("tool radius was %v, expected 5", tool.Radius())
    }

    checkHeightAtRadius(t, tool, 0, 0)
    checkHeightAtRadius(t, tool, 5, 5)
    checkHeightAtRadius(t, tool, 1, 0.1010205)
    checkHeightAtRadius(t, tool, 3, 1)
    checkHeightAtRadius(t, tool, 6, math.Inf(1))

    tool, err = NewTool("flat", 10)
    if err != nil {
        t.Errorf("can't create flat tool: %v", err)
    }

    if tool.Radius() != 5 {
        t.Errorf("tool radius was %v, expected 5", tool.Radius())
    }

    checkHeightAtRadius(t, tool, 0, 0)
    checkHeightAtRadius(t, tool, 5, 0)
    checkHeightAtRadius(t, tool, 1, 0)
    checkHeightAtRadius(t, tool, 3, 0)
    checkHeightAtRadius(t, tool, 6, math.Inf(1))
}

func checkHeightAtRadius(t *testing.T, tool Tool, r float64, wantheight float64) {
    epsilon := 0.00001

    h := tool.HeightAtRadius(r)

    if math.Abs(h-wantheight) > epsilon {
        t.Errorf("height at radius %v should be %v, got %v", r, wantheight, h)
    }
}
