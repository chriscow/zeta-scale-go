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
	t.Logf("Input points min/max bounds: (%.6f,%.6f) to (%.6f,%.6f)",
		real(links[0]), imag(links[0]),
		real(links[len(links)-1]), imag(links[len(links)-1]))

	// Calculate the relative spread for debugging
	maxRange := math.Max(real(links[len(links)-1])-real(links[0]),
		imag(links[len(links)-1])-imag(links[0]))
	baseRange := math.Max(0.01, maxRange)
	relativeSpread := maxRange / baseRange
	t.Logf("Relative spread calculation: maxRange=%.6f, baseRange=%.6f, relativeSpread=%.6f",
		maxRange, baseRange, relativeSpread)

	// With a high resolution and aggressiveness=0.0, these nearly identical values should map to the same pixel
	got := downsampleComplex(links, 2048, 0.0, true)

	// Log key diagnostic information
	if len(got) > 10 {
		t.Logf("First 5 output points: %v", got[:5])
		t.Logf("Last 5 output points: %v", got[len(got)-5:])
		t.Logf("Total points: %d (expected 1)", len(got))
	} else {
		t.Logf("All output points: %v", got)
	}

	// We expect exactly one point - the average of all input points
	want := 1
	if len(got) != want {
		t.Errorf("got length %d, want length %d", len(got), want)
		return
	}

	// Calculate expected average
	var sum complex128
	for _, link := range links {
		sum += link
	}
	expectedAvg := sum / complex(float64(len(links)), 0)
	t.Logf("Expected average point: %.6f + %.6fi", real(expectedAvg), imag(expectedAvg))

	// Verify the value is close to the expected average
	if math.Abs(real(got[0])-real(expectedAvg)) > 1e-6 || math.Abs(imag(got[0])-imag(expectedAvg)) > 1e-6 {
		t.Errorf("got point %.6f + %.6fi, want %.6f + %.6fi",
			real(got[0]), imag(got[0]), real(expectedAvg), imag(expectedAvg))
	}
}

// Test interpolation between two far apart points.
func TestDownsampleComplex_Interpolate(t *testing.T) {
	// Two points far apart so that in view space the gap spans many pixels.
	links := []complex128{
		complex(1, 1),
		complex(100, 100),
	}
	// With view bounds computed from the data, the normalized pixel coordinates will be:
	// first point: (0,0) and second point: (outputSize, outputSize) for outputSize=100.
	// The gap distance = sqrt((100-0)²+(100-0)²) ≈ 141.421356. This produces 141-1 = 140 interpolated points.
	got := downsampleComplex(links, 100, 0.5, false)
	want := 1 + 140 + 1 // (first group + interpolated points + last group)
	if len(got) != want {
		t.Errorf("got length %d, want length %d", len(got), want)
	}
	// Also check that the first and last values match the input.
	if math.Abs(real(got[0])-1) > 1e-6 || math.Abs(imag(got[0])-1) > 1e-6 {
		t.Errorf("first downsampled value incorrect: got (%.6f,%.6f), want (1,1)", real(got[0]), imag(got[0]))
	}
	if math.Abs(real(got[len(got)-1])-100) > 1e-6 || math.Abs(imag(got[len(got)-1])-100) > 1e-6 {
		t.Errorf("last downsampled value incorrect: got (%.6f,%.6f), want (100,100)", real(got[len(got)-1]), imag(got[len(got)-1]))
	}
}
