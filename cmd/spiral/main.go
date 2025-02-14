package main

import (
	"flag"
	"fmt"
	"image/color"
	"image/draw"
	"image/png"
	"io/ioutil"
	"log"
	"math"
	"math/cmplx"
	"os"
	"sync"
	"time"

	"image"

	"github.com/golang/freetype/truetype"
	"github.com/llgcode/draw2d"
	"github.com/llgcode/draw2d/draw2dimg"
)

// Constants for the Euler-Maclaurin summation
var (
	MinN      = 100
	MaxN      = 65_000_000_000
	ChunkSize = 100_000
)

func init() {
	// Load the font file from macOS fonts folder.
	fontBytes, err := ioutil.ReadFile("/Library/Fonts/Arial Unicode.ttf")
	if err != nil {
		log.Fatalf("failed to read font file: %v", err)
	}

	// Parse the font using the freetype/truetype package.
	parsedFont, err := truetype.Parse(fontBytes)
	if err != nil {
		log.Fatalf("failed to parse font: %v", err)
	}

	// Register the font so that draw2d can use it.
	draw2d.RegisterFont(draw2d.FontData{
		Name:   "Arial",
		Family: draw2d.FontFamilySans,
		Style:  draw2d.FontStyleNormal,
	}, parsedFont)
}

// computePartialSumWithLinks computes the sum from [start, end) and returns
//  1. The final partial sum for that chunk
//  2. All intermediate partial sums in that range (the "links" for that chunk)
func computePartialSumWithLinks(start, end int, s complex128) (complex128, []complex128) {
	partialSum := complex(0, 0)
	var linkList []complex128

	for k := start; k < end; k++ {
		term := cmplx.Pow(complex(float64(k), 0), -s)
		partialSum += term
		linkList = append(linkList, partialSum)
	}
	return partialSum, linkList
}

// calculateSpiralPartialSums performs the multi-threaded computation and
// returns the total sum and the properly chained links.
func calculateSpiralPartialSums(s complex128) (complex128, []complex128) {
	// Determine how many terms N
	N := int(cmplx.Abs(s))
	println("N", N)
	if N < MinN {
		N = MinN
	} else if N > MaxN {
		N = MaxN
	}
	println("N", N)

	// Figure out how many chunks we need
	numChunks := (N + ChunkSize - 1) / ChunkSize

	// Prepare slices to hold each chunk's result
	partialSums := make([]complex128, numChunks)
	allChunkLinks := make([][]complex128, numChunks)

	var wg sync.WaitGroup
	wg.Add(numChunks)

	// Launch goroutines to compute partial sums
	for i := 0; i < numChunks; i++ {
		start := i*ChunkSize + 1
		end := start + ChunkSize
		if end > N {
			end = N
		}

		go func(idx, st, ed int) {
			defer wg.Done()
			sumVal, linkVals := computePartialSumWithLinks(st, ed, s)
			partialSums[idx] = sumVal
			allChunkLinks[idx] = linkVals
		}(i, start, end)
	}

	// Wait for goroutines to finish
	wg.Wait()

	// Now chain the results in the correct order
	var totalSum complex128
	var chainedLinks []complex128
	runningSum := complex(0, 0)

	for i := 0; i < numChunks; i++ {
		// Adjust this chunk's links by the runningSum so that they are continuous
		for j := range allChunkLinks[i] {
			allChunkLinks[i][j] += runningSum
		}
		// Update the running sum by the chunk's final partial sum
		runningSum += partialSums[i]
		// Append the newly adjusted chunk links to the big list
		chainedLinks = append(chainedLinks, allChunkLinks[i]...)
	}

	// runningSum is effectively the total sum of the first N terms
	totalSum = runningSum

	// Apply Euler-Maclaurin correction terms
	term1 := cmplx.Pow(complex(float64(N), 0), 1-s) / (s - 1)
	term2 := 0.5 * cmplx.Pow(complex(float64(N), 0), -s)
	totalSum += term1 + term2

	// Also add corrections to the final link
	if len(chainedLinks) > 0 {
		chainedLinks[len(chainedLinks)-1] += term1 + term2
	}

	return totalSum, chainedLinks
}

// calculateSingleThreadedPartialSums simply accumulates the sum link by link
func calculateSingleThreadedPartialSums(s complex128, numLinks int) []complex128 {
	links := make([]complex128, numLinks)
	partialSum := complex(0, 0)

	for k := 1; k < numLinks; k++ {
		term := cmplx.Pow(complex(float64(k), 0), -s)
		partialSum += term
		links[k] = partialSum
		log.Printf("Single-threaded link %d: (%.6f, %.6f)", k, real(partialSum), imag(partialSum))
	}
	return links
}

// plotLinks creates and saves a PNG of the link path plus a crosshair at zeta
func plotLinks(links []complex128, zeta complex128, outputFile string, pointsOnly bool) {
	const numWorkers = 24   // Number of goroutines
	const outputSize = 2048 // Final output image width and height

	// Determine the min and max for x and y across all links.
	minX, maxX := real(links[0]), real(links[0])
	minY, maxY := imag(links[0]), imag(links[0])
	for _, link := range links {
		x := real(link)
		y := imag(link)
		if x < minX {
			minX = x
		}
		if x > maxX {
			maxX = x
		}
		if y < minY {
			minY = y
		}
		if y > maxY {
			maxY = y
		}
	}
	log.Printf("Link X range: [%f, %f], Y range: [%f, %f]\n", minX, maxX, minY, maxY)

	// Divide the links among workers.
	chunkSize := (len(links) + numWorkers - 1) / numWorkers

	// Each worker creates an image of the full output size with transparent background.
	workerImages := make([]*image.RGBA, numWorkers)
	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		start := i * chunkSize
		end := start + chunkSize
		if end > len(links) {
			end = len(links)
		}
		wg.Add(1)
		go func(worker, start, end int) {
			defer wg.Done()
			log.Printf("Worker %d drawing links from index %d to %d\n", worker, start, end)
			// Create full-size image with transparent background.
			img := image.NewRGBA(image.Rect(0, 0, outputSize, outputSize))
			// Clear image to transparent.
			gc := draw2dimg.NewGraphicContext(img)
			gc.SetFillColor(color.RGBA{0, 0, 0, 0})
			gc.Clear()

			// Set drawn line properties in white.
			gc.SetStrokeColor(color.RGBA{255, 255, 255, 64})
			if pointsOnly {
				gc.SetFillColor(color.RGBA{255, 255, 255, 255})
			}
			gc.SetLineWidth(0.5)

			// Draw the links in this chunk.
			if end > start {
				for j := start; j < end; j++ {
					x := real(links[j])
					y := imag(links[j])
					// Normalize x and y into [0, outputSize] based on overall range.
					normalizedX := (x - minX) / (maxX - minX) * float64(outputSize)
					normalizedY := (y - minY) / (maxY - minY) * float64(outputSize)
					// Invert Y because image coordinates start at top.
					finalX := normalizedX
					finalY := float64(outputSize) - normalizedY

					if pointsOnly {
						// Draw a small circle for each point
						gc.BeginPath()
						gc.ArcTo(finalX, finalY, 1.0, 1.0, 0, 2*math.Pi)
						gc.Close()
						gc.FillStroke()
					} else {
						if j == start {
							gc.MoveTo(finalX, finalY)
						} else {
							gc.LineTo(finalX, finalY)
						}
					}
				}
				if !pointsOnly {
					gc.Stroke()
				}
			} else {
				log.Printf("Worker %d has no links to draw\n", worker)
			}
			workerImages[worker] = img
		}(i, start, end)
	}
	wg.Wait()
	log.Println("All workers completed processing their chunks.")

	// Create the base final image with a solid dark grey background.
	finalImage := image.NewRGBA(image.Rect(0, 0, outputSize, outputSize))
	draw.Draw(finalImage, finalImage.Bounds(), &image.Uniform{color.RGBA{30, 30, 30, 255}}, image.Point{}, draw.Src)

	// Composite each worker's transparent image on top of the dark grey background.
	for i, img := range workerImages {
		log.Printf("Compositing worker %d image\n", i)
		draw.Draw(finalImage, img.Bounds(), img, image.Point{}, draw.Over)
	}

	// Create an overlay layer for axis markers and text (drawn in white).
	overlay := image.NewRGBA(image.Rect(0, 0, outputSize, outputSize))
	gcOverlay := draw2dimg.NewGraphicContext(overlay)
	gcOverlay.SetFillColor(color.RGBA{0, 0, 0, 0})
	gcOverlay.Clear()

	// Set white color for drawing on the overlay.
	gcOverlay.SetStrokeColor(color.White)
	gcOverlay.SetFillColor(color.White)
	gcOverlay.SetLineWidth(2)
	gcOverlay.SetFontData(draw2d.FontData{
		Name:   "Arial",
		Family: draw2d.FontFamilySans,
		Style:  draw2d.FontStyleNormal,
	})
	gcOverlay.SetFontSize(14)

	// Draw simple axis markers:
	// X-axis: if 0 is in the y-range, draw a horizontal line.
	if minY <= 0 && maxY >= 0 {
		normalizedY := (0 - minY) / (maxY - minY) * float64(outputSize)
		y0 := float64(outputSize) - normalizedY
		gcOverlay.SetLineWidth(1)
		gcOverlay.SetStrokeColor(color.RGBA{30, 30, 30, 66})
		gcOverlay.MoveTo(0, y0)
		gcOverlay.LineTo(float64(outputSize), y0)
		gcOverlay.Stroke()
	}
	// Y-axis: if 0 is in the x-range, draw a vertical line.
	if minX <= 0 && maxX >= 0 {
		normalizedX := (0 - minX) / (maxX - minX) * float64(outputSize)
		gcOverlay.SetLineWidth(1)
		gcOverlay.SetStrokeColor(color.RGBA{30, 30, 30, 66})
		gcOverlay.MoveTo(normalizedX, 0)
		gcOverlay.LineTo(normalizedX, float64(outputSize))
		gcOverlay.Stroke()
	}

	// Composite the overlay onto the final image.
	draw.Draw(finalImage, finalImage.Bounds(), overlay, image.Point{}, draw.Over)

	log.Printf("Final image dimensions: %dx%d\n", finalImage.Bounds().Dx(), finalImage.Bounds().Dy())

	// Save the final image.
	outFile, err := os.Create(outputFile)
	if err != nil {
		log.Fatalf("failed to create output file: %v", err)
	}
	defer outFile.Close()

	if err := png.Encode(outFile, finalImage); err != nil {
		log.Fatalf("failed to encode image: %v", err)
	}

	log.Println("Image saved as", outputFile)
}

// Point represents a 2D point.
type Point struct {
	X, Y float64
}

// downsample simulates averaging groups of points (mimicking the compute shader logic).
// Given a slice of points and a groupSize, it returns a slice of averaged points.
func downsample(points []Point, groupSize int) []Point {
	n := len(points)
	outputCount := n / groupSize
	result := make([]Point, 0, outputCount)
	for i := 0; i < outputCount; i++ {
		var sumX, sumY float64
		start := i * groupSize
		end := start + groupSize
		for j := start; j < end; j++ {
			sumX += points[j].X
			sumY += points[j].Y
		}
		result = append(result, Point{
			X: sumX / float64(groupSize),
			Y: sumY / float64(groupSize),
		})
	}
	return result
}

// downsampleComplex uses the view bounds (computed from all links) and the output image size,
// so that only links that fall within the same pixel are averaged. Additionally, if two adjacent
// groups are separated by more than one pixel, it linearly interpolates extra points.
// aggressiveness controls how much reduction to do (0.0 = minimal, 1.0 = maximum)
func downsampleComplex(links []complex128, outputSize int, aggressiveness float64, debug bool) []complex128 {
	if len(links) == 0 {
		return links
	}

	if debug {
		log.Printf("Starting downsampleComplex with %d links and output size %d (aggressiveness: %.2f)",
			len(links), outputSize, aggressiveness)
	}

	// Determine view bounds from the links.
	minX, maxX := real(links[0]), real(links[0])
	minY, maxY := imag(links[0]), imag(links[0])
	for _, link := range links {
		x := real(link)
		y := imag(link)
		if x < minX {
			minX = x
		}
		if x > maxX {
			maxX = x
		}
		if y < minY {
			minY = y
		}
		if y > maxY {
			maxY = y
		}
	}
	if debug {
		log.Printf("View bounds: minX=%.6f, maxX=%.6f, minY=%.6f, maxY=%.6f", minX, maxX, minY, maxY)
	}

	// Calculate relative distance between points
	// For small ranges (< 0.01), we should consider them close together
	maxRange := math.Max(maxX-minX, maxY-minY) // Use actual range instead of max value
	baseRange := math.Max(0.01, maxRange)      // Use the range itself as base
	relativeSpread := maxRange / baseRange
	if debug {
		log.Printf("Relative calculations: maxRange=%e, baseRange=%e, relativeSpread=%e", maxRange, baseRange, relativeSpread)
	}

	// Scale the maxRelativeSpread based on aggressiveness
	// At aggressiveness=0.0: 0.01% spread (very precise)
	// At aggressiveness=1.0: 0.1% spread (standard)
	// At aggressiveness=2.0: 1% spread (more aggressive)
	// At aggressiveness=3.0: 2% spread (very aggressive)
	// At aggressiveness=3.5: 3% spread (extremely aggressive)
	// At aggressiveness=4.0: 5% spread (maximum)
	maxRelativeSpread := 0.0001 * math.Pow(5, aggressiveness)

	// Add extra smoothing for values between 3.5 and 4.0 to avoid the cliff
	if aggressiveness > 3.5 {
		// Smooth transition from 3% to 5% spread
		t := (aggressiveness - 3.5) / 0.5     // 0 to 1 as we go from 3.5 to 4.0
		maxRelativeSpread = 0.03 + (0.02 * t) // Linear interpolation from 3% to 5%
	}

	// Also consider pixel-space proximity for grouping
	pixelSpreadThreshold := 1.0 + (aggressiveness * 2.0) // 1.0 to 9.0 pixels

	if debug {
		log.Printf("Using maxRelativeSpread=%e based on aggressiveness=%.2f", maxRelativeSpread, aggressiveness)
	}

	// If the relative spread is small enough, average the points
	if relativeSpread <= maxRelativeSpread {
		if debug {
			log.Printf("Points are relatively close: %e <= %e", relativeSpread, maxRelativeSpread)
		}
		var sum complex128
		for _, link := range links {
			sum += link
		}
		avg := sum / complex(float64(len(links)), 0)
		if debug {
			log.Printf("Computed average of %d points: %.6f + %.6fi", len(links), real(avg), imag(avg))
		}
		return []complex128{avg}
	}
	if debug {
		log.Printf("Points are too far apart relatively: %e > %e", relativeSpread, maxRelativeSpread)
	}

	// Helper to compute pixel coordinate for a given link.
	pixelForLink := func(link complex128) (int, int) {
		normalizedX := (real(link) - minX) / (maxX - minX)
		normalizedY := (imag(link) - minY) / (maxY - minY)
		px := int(math.Round(normalizedX * float64(outputSize)))
		py := int(math.Round(normalizedY * float64(outputSize)))
		if debug {
			log.Printf("Mapping point (%.6f,%.6f) to pixel (%d,%d) [normalized: %.6f,%.6f]",
				real(link), imag(link), px, py, normalizedX, normalizedY)
		}
		return px, py
	}

	// We'll accumulate groups of contiguous links that fall into the same pixel.
	type groupData struct {
		sum      complex128
		count    int
		pixelX   int
		pixelY   int
		lastLink complex128 // last link value in this group; used for interpolation
	}

	var downsampled []complex128

	// Initialize the first group.
	initPx, initPy := pixelForLink(links[0])
	currentGroup := groupData{
		sum:      links[0],
		count:    1,
		pixelX:   initPx,
		pixelY:   initPy,
		lastLink: links[0],
	}

	// Flush a group by averaging it.
	flushGroup := func(g groupData) complex128 {
		avg := g.sum / complex(float64(g.count), 0)
		if debug {
			log.Printf("Flushing group: pixel=(%d,%d), count=%d, avg=(%.6f,%.6f)",
				g.pixelX, g.pixelY, g.count, real(avg), imag(avg))
		}
		return avg
	}

	// Iterate over the links.
	for i := 1; i < len(links); i++ {
		link := links[i]
		px, py := pixelForLink(link)

		// Check if this point is close enough to be considered in the same pixel
		if px == currentGroup.pixelX && py == currentGroup.pixelY ||
			(math.Abs(float64(px-currentGroup.pixelX)) <= pixelSpreadThreshold &&
				math.Abs(float64(py-currentGroup.pixelY)) <= pixelSpreadThreshold) {
			// Same pixel or within threshold: accumulate.
			currentGroup.sum += link
			currentGroup.count++
			currentGroup.lastLink = link
			continue
		}

		// Group changed: flush the current group.
		avg := flushGroup(currentGroup)
		downsampled = append(downsampled, avg)

		// Check gap in pixel coordinates from the previous group to the current link.
		dx := px - currentGroup.pixelX
		dy := py - currentGroup.pixelY
		pixelGap := math.Sqrt(float64(dx*dx + dy*dy))

		// Scale the interpolation threshold based on aggressiveness
		// At aggressiveness=0.0: gaps > 1.1 pixels (very detailed)
		// At aggressiveness=1.0: gaps > 5 pixels (standard)
		// At aggressiveness=2.0: gaps > 15 pixels (more aggressive)
		// At aggressiveness=3.0: gaps > 35 pixels (very aggressive)
		// At aggressiveness=3.5: gaps > 55 pixels (extremely aggressive)
		// At aggressiveness=4.0: gaps > 75 pixels (maximum)
		interpolationThreshold := 1.1 * math.Pow(2.5, aggressiveness)

		// Add extra smoothing for values between 3.5 and 4.0
		if aggressiveness > 3.5 {
			// Smooth transition from 55 to 75 pixels
			t := (aggressiveness - 3.5) / 0.5
			interpolationThreshold = 55.0 + (20.0 * t)
		}

		// Only interpolate if the gap is significantly larger than one pixel
		if pixelGap > interpolationThreshold {
			// Interpolate extra points, reducing count more aggressively at higher values
			// Also smooth the steps reduction for high aggressiveness
			steps := int(pixelGap / math.Pow(2, math.Min(aggressiveness, 3.5)))
			if aggressiveness > 3.5 {
				// Further reduce steps linearly from 3.5 to 4.0
				t := (aggressiveness - 3.5) / 0.5
				steps = int(float64(steps) * (1.0 - (0.5 * t))) // Reduce by up to 50% more
			}
			if debug {
				log.Printf("Interpolating %d points between pixel=(%d,%d) and pixel=(%d,%d) (threshold: %.2f)",
					steps, currentGroup.pixelX, currentGroup.pixelY, px, py, interpolationThreshold)
			}
			for s := 1; s <= steps; s++ {
				t := float64(s) / float64(steps+1)
				interp := currentGroup.lastLink*(1-complex(t, 0)) + link*complex(t, 0)
				downsampled = append(downsampled, interp)
			}
		}

		// Start a new group with the current link.
		currentGroup = groupData{
			sum:      link,
			count:    1,
			pixelX:   px,
			pixelY:   py,
			lastLink: link,
		}
	}

	// Flush any remaining group.
	finalAvg := flushGroup(currentGroup)
	downsampled = append(downsampled, finalAvg)

	if debug {
		log.Printf("Downsampled %d points to %d points", len(links), len(downsampled))
	}
	return downsampled
}

func main() {
	// Read command-line flags
	imagPart := flag.Float64("imag", 6_300_000.0, "Imaginary part of the complex number")
	maxN := flag.Int("maxN", 65_000_000_000, "Maximum number of terms")
	downsampleFlag := flag.Bool("downsample", false, "Enable downsampling of links")
	aggressiveness := flag.Float64("aggressive", 0.5, "Downsampling aggressiveness (0.0-1.0)")
	outputFile := flag.String("output", "combined_links.png", "Output filename for the image")
	debugFlag := flag.Bool("debug", false, "Enable debug logging")
	pointsOnlyFlag := flag.Bool("points", false, "Draw points only, no lines")
	flag.Parse()

	// Set MaxN from the command-line flag
	MaxN = *maxN

	start := time.Now()

	// Example complex number with real part 0.5
	s := complex(0.5, *imagPart)

	// Multi-threaded
	result, multiThreadedLinks := calculateSpiralPartialSums(s)

	// Downsample if the flag is set
	if *downsampleFlag {
		// Use the same resolution as the final output image.
		pixelResolution := 2048
		before := len(multiThreadedLinks)
		multiThreadedLinks = downsampleComplex(multiThreadedLinks, pixelResolution, *aggressiveness, *debugFlag)
		after := len(multiThreadedLinks)

		// Calculate downsampling statistics
		reductionRatio := float64(before) / float64(after)
		memoryBefore := before * 16 // complex128 = 16 bytes
		memoryAfter := after * 16
		memorySaved := float64(memoryBefore-memoryAfter) / 1024.0 // Convert to KB

		fmt.Printf("\nDownsampling Statistics (aggressiveness=%.2f):\n", *aggressiveness)
		fmt.Printf("Points reduced: %d → %d\n", before, after)
		fmt.Printf("Reduction ratio: %.2fx\n", reductionRatio)
		fmt.Printf("Memory saved: %.2f KB\n", memorySaved)
		fmt.Printf("Average distance between points: %.6f\n",
			math.Sqrt(math.Pow(real(multiThreadedLinks[len(multiThreadedLinks)-1]-multiThreadedLinks[0]), 2)+
				math.Pow(imag(multiThreadedLinks[len(multiThreadedLinks)-1]-multiThreadedLinks[0]), 2))/float64(len(multiThreadedLinks)))
		fmt.Printf("Maintained visual quality while using %.1f%% fewer points\n",
			100.0*(1.0-float64(after)/float64(before)))
	}

	// Print the final result
	fmt.Printf("\nEuler-Maclaurin result: (%.6f, %.6f)\n", real(result), imag(result))
	elapsed := time.Since(start)
	fps := 1.0 / elapsed.Seconds()
	fmt.Printf("Time taken: %v FPS: %.2f\n", elapsed, fps)

	// Plot
	// prepend a 0,0 link to the multi-threaded links
	start = time.Now()
	println("\nPlotting multi-threaded links")
	multiThreadedLinks = append([]complex128{complex(0, 0)}, multiThreadedLinks...)
	plotLinks(multiThreadedLinks, result, *outputFile, *pointsOnlyFlag) // Pass the points-only flag
	elapsed = time.Since(start)
	fps = 1.0 / elapsed.Seconds()
	fmt.Printf("Time taken: %v FPS: %.2f\n", elapsed, fps)
}
