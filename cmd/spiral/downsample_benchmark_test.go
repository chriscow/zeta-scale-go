package main

import (
	"fmt"
	"math"
	"math/cmplx"
	"testing"
)

// generateTestLinks creates a spiral of complex points for testing
func generateTestLinks(n int) []complex128 {
	links := make([]complex128, n)
	for i := 0; i < n; i++ {
		t := float64(i) / float64(n)
		r := t * 10
		theta := t * 20 * 2 * math.Pi
		links[i] = cmplx.Rect(r, theta)
	}
	return links
}

// generateDownsampledLinks creates a spiral pattern while downsampling inline
func generateDownsampledLinks(n int, outputSize int, aggressiveness float64) []complex128 {
	// Pre-allocate a smaller buffer based on expected reduction
	// Start with a conservative estimate
	estimatedSize := n / 10
	if estimatedSize < 1000 {
		estimatedSize = 1000
	}
	result := make([]complex128, 0, estimatedSize)

	// Keep track of accumulated points for averaging
	var sum complex128
	count := 0
	lastPoint := complex(0, 0)
	var lastR, lastTheta float64

	// Parameters for determining when to emit points
	pixelSpreadThreshold := 1.0 // Base threshold
	if aggressiveness > 0.0 {
		pixelSpreadThreshold += (aggressiveness * 2.0)
	}

	for i := 0; i < n; i++ {
		t := float64(i) / float64(n)
		r := t * 10
		theta := t * 20 * 2 * 3.14159
		point := cmplx.Rect(r, theta)

		// Determine if we should emit a point based on:
		// 1. Distance in polar coordinates
		// 2. Visual distance in the output space
		rDiff := math.Abs(r - lastR)
		thetaDiff := math.Abs(theta - lastTheta)

		// Convert to pixel space
		x := real(point)
		y := imag(point)
		lastX := real(lastPoint)
		lastY := imag(lastPoint)

		// Normalize to output size
		pixelX := x * float64(outputSize) / 20.0 // 20.0 is our view bounds
		pixelY := y * float64(outputSize) / 20.0
		lastPixelX := lastX * float64(outputSize) / 20.0
		lastPixelY := lastY * float64(outputSize) / 20.0

		pixelDist := math.Sqrt(math.Pow(pixelX-lastPixelX, 2) + math.Pow(pixelY-lastPixelY, 2))

		// Emit point if:
		// 1. Significant change in r or theta
		// 2. Moved more than threshold pixels
		// 3. Last point in sequence
		if count == 0 || // Always emit first point
			pixelDist > pixelSpreadThreshold || // Moved enough pixels
			rDiff > 0.1 || // Significant radial change
			thetaDiff > 0.1 || // Significant angular change
			i == n-1 { // Last point

			if count > 0 {
				// Emit averaged point
				avg := sum / complex(float64(count), 0)
				result = append(result, avg)
				// Reset accumulator
				sum = point
				count = 1
			} else {
				// First point
				sum = point
				count = 1
			}

			lastPoint = point
			lastR = r
			lastTheta = theta
		} else {
			// Accumulate point
			sum += point
			count++
		}
	}

	// Emit any remaining accumulated points
	if count > 0 {
		avg := sum / complex(float64(count), 0)
		result = append(result, avg)
	}

	return result
}

func BenchmarkDownsampleVsNoDownsample(b *testing.B) {
	// Test cases with different sizes and aggressiveness levels
	testCases := []struct {
		name           string
		size           int
		aggressiveness float64
	}{
		{"Small_NoDownsample", 100_000, 0.0},
		{"Small_LightDownsample", 100_000, 0.5},
		{"Small_HeavyDownsample", 100_000, 4.0},
		{"Medium_NoDownsample", 1_000_000, 0.0},
		{"Medium_LightDownsample", 1_000_000, 0.5},
		{"Medium_HeavyDownsample", 1_000_000, 4.0},
		{"Large_NoDownsample", 65_000_000, 0.0},
		{"Large_LightDownsample", 65_000_000, 0.5},
		{"Large_HeavyDownsample", 65_000_000, 4.0},
	}

	for _, tc := range testCases {
		// Generate test data
		links := generateTestLinks(tc.size)
		outputSize := 2048 // Standard output size

		b.Run(tc.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if tc.aggressiveness > 0 {
					downsampleComplex(links, outputSize, tc.aggressiveness, false)
				} else {
					// Just iterate through the links to simulate "no downsampling"
					for j := 0; j < len(links); j++ {
						_ = links[j]
					}
				}
			}
		})

		// Log memory stats after each test case
		b.ReportMetric(float64(len(links)*16)/1024, "KB_before") // 16 bytes per complex128
		result := downsampleComplex(links, outputSize, tc.aggressiveness, false)
		b.ReportMetric(float64(len(result)*16)/1024, "KB_after")
		b.ReportMetric(float64(len(links))/float64(len(result)), "reduction_ratio")
	}
}

// BenchmarkDownsampleComplexEdgeCases tests performance in specific scenarios
func BenchmarkDownsampleComplexEdgeCases(b *testing.B) {
	testCases := []struct {
		name      string
		generator func(int) []complex128
		size      int
	}{
		{
			name: "AllSamePixel",
			generator: func(n int) []complex128 {
				links := make([]complex128, n)
				for i := 0; i < n; i++ {
					// All points very close together
					x := 1.0 + float64(i)*0.0001
					y := 1.0 + float64(i)*0.0001
					links[i] = complex(x, y)
				}
				return links
			},
			size: 10000,
		},
		{
			name: "WideSpread",
			generator: func(n int) []complex128 {
				links := make([]complex128, n)
				for i := 0; i < n; i++ {
					// Points spread far apart
					x := float64(i) * 10
					y := float64(i) * 10
					links[i] = complex(x, y)
				}
				return links
			},
			size: 10000,
		},
	}

	for _, tc := range testCases {
		links := tc.generator(tc.size)
		outputSize := 2048

		b.Run(tc.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				downsampleComplex(links, outputSize, 0.5, false)
			}
		})

		// Log memory stats
		b.ReportMetric(float64(len(links)*16)/1024, "KB_before")
		result := downsampleComplex(links, outputSize, 0.5, false)
		b.ReportMetric(float64(len(result)*16)/1024, "KB_after")
		b.ReportMetric(float64(len(links))/float64(len(result)), "reduction_ratio")
	}
}

func BenchmarkInlineVsPostDownsample(b *testing.B) {
	testCases := []struct {
		name           string
		size           int
		aggressiveness float64
	}{
		{"Small_Light", 100_000, 0.5},
		{"Small_Heavy", 100_000, 4.0},
		{"Medium_Light", 1_000_000, 0.5},
		{"Medium_Heavy", 1_000_000, 4.0},
		{"Large_Light", 65_000_000, 0.5},
		{"Large_Heavy", 65_000_000, 4.0},
	}

	outputSize := 2048

	for _, tc := range testCases {
		b.Run("Post_"+tc.name, func(b *testing.B) {
			links := generateTestLinks(tc.size)
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				downsampleComplex(links, outputSize, tc.aggressiveness, false)
			}
			// Report memory for first run
			if i := 0; i == 0 {
				b.ReportMetric(float64(len(links)*16)/1024, "KB_initial")
				result := downsampleComplex(links, outputSize, tc.aggressiveness, false)
				b.ReportMetric(float64(len(result)*16)/1024, "KB_final")
				b.ReportMetric(float64(len(links))/float64(len(result)), "reduction_ratio")
			}
		})

		b.Run("Inline_"+tc.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				result := generateDownsampledLinks(tc.size, outputSize, tc.aggressiveness)
				if i == 0 {
					// Report memory for first run only
					b.ReportMetric(float64(tc.size*16)/1024, "KB_wouldbe")
					b.ReportMetric(float64(len(result)*16)/1024, "KB_actual")
					b.ReportMetric(float64(tc.size)/float64(len(result)), "reduction_ratio")
				}
			}
		})
	}
}

func BenchmarkDownsampleComplex(b *testing.B) {
	sizes := []int{1000, 10000, 100000, 1000000}
	aggressiveness := []float64{0.0, 1.0, 2.0, 3.0, 4.0}
	outputSize := 2048

	for _, size := range sizes {
		links := generateTestLinks(size)

		for _, agg := range aggressiveness {
			b.Run("Serial/Size="+formatInt(size)+"/Agg="+formatFloat(agg), func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					downsampleComplexSerial(links, outputSize, agg, false)
				}
			})

			b.Run("Parallel/Size="+formatInt(size)+"/Agg="+formatFloat(agg), func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					downsampleComplex(links, outputSize, agg, false)
				}
			})
		}
	}
}

// Helper functions to format numbers for test names
func formatInt(n int) string {
	if n >= 1000000 {
		return fmt.Sprintf("%.1fM", float64(n)/1000000)
	} else if n >= 1000 {
		return fmt.Sprintf("%.1fK", float64(n)/1000)
	}
	return fmt.Sprintf("%d", n)
}

func formatFloat(f float64) string {
	return fmt.Sprintf("%.1f", f)
}
