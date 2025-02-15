package main

import (
	"math"
	"testing"
)

func TestDownsample(t *testing.T) {
	// Create a simple test case with 8 points.
	partialSums := []Point{
		{0, 0}, {1, 1}, {2, 2}, {3, 3},
		{4, 4}, {5, 5}, {6, 6}, {7, 7},
	}
	groupSize := 2

	// Expected averages:
	// Group 0: average of (0,0) and (1,1) = (0.5, 0.5)
	// Group 1: average of (2,2) and (3,3) = (2.5, 2.5)
	// Group 2: average of (4,4) and (5,5) = (4.5, 4.5)
	// Group 3: average of (6,6) and (7,7) = (6.5, 6.5)
	want := []Point{
		{0.5, 0.5},
		{2.5, 2.5},
		{4.5, 4.5},
		{6.5, 6.5},
	}

	got := downsample(partialSums, groupSize)
	if len(got) != len(want) {
		t.Errorf("got length %d, want length %d", len(got), len(want))
	}

	// Verify that each downsampled point matches the expected average.
	for i := range got {
		if !floatEquals(got[i].X, want[i].X, 1e-6) || !floatEquals(got[i].Y, want[i].Y, 1e-6) {
			t.Errorf("at index %d, got (%f, %f), want (%f, %f)",
				i, got[i].X, got[i].Y, want[i].X, want[i].Y)
		}
	}
}

func floatEquals(a, b, tolerance float64) bool {
	return math.Abs(a-b) <= tolerance
}

// Test when all points fall in (roughly) the same pixel.
func TestDownsampleComplex_SamePixel(t *testing.T) {
	links := []complex128{
		complex(1, 1),
		complex(1.0001, 1.0001), // Much closer points (0.01% apart)
		complex(1.0002, 1.0002),
	}

	// With a high resolution and aggressiveness=4.0 (maximum), these nearly identical values should be averaged
	got := downsampleComplex(links, 2048, 4.0, true)

	// With high aggressiveness, we expect a single averaged point
	if len(got) != 1 {
		t.Errorf("got %d points, expected 1 point with high aggressiveness", len(got))
		return
	}

	// Calculate expected average
	var sum complex128
	for _, link := range links {
		sum += link
	}
	expectedAvg := sum / complex(float64(len(links)), 0)

	// Verify the point is close to the expected average
	if !cmplxEquals(got[0], expectedAvg, 1e-6) {
		t.Errorf("point mismatch: got %v, want %v", got[0], expectedAvg)
	}
}

func cmplxEquals(a, b complex128, tolerance float64) bool {
	return math.Abs(real(a)-real(b)) <= tolerance && math.Abs(imag(a)-imag(b)) <= tolerance
}

// Test interpolation between two far apart points.
func TestDownsampleComplex_Interpolate(t *testing.T) {
	// Two points far apart so that in view space the gap spans many pixels.
	links := []complex128{
		complex(1, 1),
		complex(100, 100),
	}

	// With aggressiveness=4.0 (maximum), we expect fewer interpolated points
	got := downsampleComplex(links, 100, 4.0, false)

	// We expect some points, but not too many due to high aggressiveness
	if len(got) < 2 {
		t.Errorf("got too few points: %d, expected at least 2", len(got))
	}
	if len(got) > 20 {
		t.Errorf("got too many points: %d, expected 20 or fewer with high aggressiveness", len(got))
	}

	// Check that the first and last values match the input
	if !cmplxEquals(got[0], links[0], 1e-6) {
		t.Errorf("first point mismatch: got %v, want %v", got[0], links[0])
	}
	if !cmplxEquals(got[len(got)-1], links[len(links)-1], 1e-6) {
		t.Errorf("last point mismatch: got %v, want %v", got[len(got)-1], links[len(links)-1])
	}
}
