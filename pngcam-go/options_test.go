package main

import (
	"math"
	"testing"
)

func TestFeedRate(t *testing.T) {
	opt := Options{
		safeZ:     5,
		rapidFeed: 10000,
		xyFeed:    2000,
		zFeed:     200,
	}

	// vertical up: rapid feed
	checkFeedRate(t, opt, 0, 0, 0, 0, 0, 10, opt.rapidFeed)

	// vertical down: z feed
	checkFeedRate(t, opt, 0, 0, 10, 0, 0, 0, opt.zFeed)

	// xy motion: xy feed
	checkFeedRate(t, opt, 0, 0, 0, 10, 0, 0, opt.xyFeed)
	checkFeedRate(t, opt, 10, 0, 0, 10, 10, 0, opt.xyFeed)

	// shallow diagonal motion up/down: xy feed
	checkFeedRate(t, opt, 0, 0, 0, 10, 10, 1, opt.xyFeed)
	checkFeedRate(t, opt, 0, 0, 0, 10, 10, -1, opt.xyFeed)

	// steep diagonal motion up: xyfeed
	checkFeedRate(t, opt, 0, 0, 0, 1, 1, 10, opt.xyFeed)

	// steep diagonal motion down: interpolated feed
	checkFeedRate(t, opt, 0, 0, 0, 1, 0, -10, math.Sqrt(1*1+10*10)/10*opt.zFeed)
}

func checkFeedRate(t *testing.T, opt Options, x1 float64, y1 float64, z1 float64, x2 float64, y2 float64, z2 float64, wantfeed float64) {
	epsilon := 0.00001

	feed := opt.FeedRate(Toolpoint{x1, y1, z1, CuttingFeed}, Toolpoint{x2, y2, z2, CuttingFeed})

	if math.Abs(feed-wantfeed) > epsilon {
		t.Errorf("feed rate from (%f,%f,%f) to (%f,%f,%f) should be %f, got %f", x1, y1, z1, x2, y2, z2, wantfeed, feed)
	}
}
