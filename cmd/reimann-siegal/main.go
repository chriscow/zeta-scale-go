// main.go — main entrypoint using the Reimann–Siegel approach with parallel channels.
package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"log"
	"math"
	"math/cmplx"
	"os"
	"time"
)

// PsiThirdDerivative is a helper that computes the third derivative of the Psi function,
// translated directly from Zeta.cs. Verbose logging is added.
func PsiThirdDerivative(x float64) float64 {
	pi := math.Pi
	term1 := math.Pow(pi, 3) * math.Pow(4*x-2, 3) * math.Sin(pi*(2*x*x-2*x-1.0/8)) / math.Cos(2*pi*x)
	term2 := -6 * math.Pow(pi, 3) * math.Pow(4*x-2, 2) * math.Sin(2*pi*x) * math.Cos(pi*(2*x*x-2*x-1.0/8)) / math.Pow(math.Cos(2*pi*x), 2)
	term3 := -24 * math.Pow(pi, 3) * (4*x - 2) * math.Pow(math.Sin(2*pi*x), 2) * math.Sin(pi*(2*x*x-2*x-1.0/8)) / math.Pow(math.Cos(2*pi*x), 3)
	term4 := -12 * math.Pow(pi, 3) * (4*x - 2) * math.Sin(pi*(2*x*x-2*x-1.0/8)) / math.Cos(2*pi*x)
	term5 := -4 * math.Pow(pi, 2) * (4*x - 2) * math.Cos(pi*(2*x*x-2*x-1.0/8)) / math.Cos(2*pi*x)
	term6 := -math.Pow(pi, 2) * (32*x - 16) * math.Cos(pi*(2*x*x-2*x-1.0/8)) / math.Cos(2*pi*x)
	term7 := 48 * math.Pow(pi, 3) * math.Pow(math.Sin(2*pi*x), 3) * math.Cos(pi*(2*x*x-2*x-1.0/8)) / math.Pow(math.Cos(2*pi*x), 4)
	term8 := -24 * math.Pow(pi, 2) * math.Sin(2*pi*x) * math.Sin(pi*(2*x*x-2*x-1.0/8)) / math.Pow(math.Cos(2*pi*x), 2)
	term9 := 40 * math.Pow(pi, 3) * math.Sin(2*pi*x) * math.Cos(pi*(2*x*x-2*x-1.0/8)) / math.Pow(math.Cos(2*pi*x), 2)

	result := term1 + term2 + term3 + term4 + term5 + term6 + term7 + term8 + term9
	log.Printf("PsiThirdDerivative(%f) = %f", x, result)
	return result
}

// reimannSiegelWithLinks implements a Reimann–Siegel approximation for ζ(0.5+it),
// computing a chain of partial sums (“links”) based on the formula in Zeta.cs.
// It uses Go channels to compute each term of the sum in parallel.
func reimannSiegelWithLinks(s complex128) (complex128, []complex128) {
	t := imag(s)
	if t <= 0 {
		log.Fatalf("Imaginary part must be > 0, got %f", t)
	}
	// v = floor(sqrt(t/(2π)))
	v := int(math.Floor(math.Sqrt(t / (2 * math.Pi))))
	log.Printf("Computed v = %d for t = %f", v, t)

	// Define V(t) as in the C# version with extra correction terms.
	V := t/2*math.Log(t/(2*math.Pi)) - t/2 - math.Pi/8
	V += 1/(48*t) + 7/(5760*math.Pow(t, 3)) + 31/(80640*math.Pow(t, 5)) + 127/(430080*math.Pow(t, 7)) + 511/(1216512*math.Pow(t, 9))
	log.Printf("Computed V(t) = %f", V)

	// Compute correction terms. Here T = sqrt(t/(2π)) - v.
	T_val := math.Sqrt(t/(2*math.Pi)) - float64(v)
	// c0 = phi(T) with φ(u) = cos(2π*(u² - u - 1/16)) / cos(2π*u)
	c0 := math.Cos(2*math.Pi*(T_val*T_val-T_val-1.0/16.0)) / math.Cos(2*math.Pi*T_val)
	c1 := -PsiThirdDerivative(T_val) / (96 * math.Pow(math.Pi, 2)) * math.Pow(t/(2*math.Pi), -0.5)
	log.Printf("Computed c0 = %f, c1 = %f", c0, c1)

	sign := 1.0
	if (v-1)%2 != 0 {
		sign = -1.0
	}
	b := sign * math.Pow(2*math.Pi/t, 0.25) * (c0 + c1)
	log.Printf("Computed correction term b = %f", b)

	// Compute each term in the series: a_k = 1/sqrt(k+1) * cos(V - t*log(k+1)) for k = 0,…, v-1.
	type termResult struct {
		index int
		value float64
	}
	termCh := make(chan termResult, v)
	for k := 0; k < v; k++ {
		go func(k int) {
			term := 1.0 / math.Sqrt(float64(k+1)) * math.Cos(V-t*math.Log(float64(k+1)))
			log.Printf("Computed term for k=%d: %f", k, term)
			termCh <- termResult{k, term}
		}(k)
	}

	// Collect all computed terms.
	terms := make([]float64, v)
	for i := 0; i < v; i++ {
		res := <-termCh
		terms[res.index] = res.value
	}
	close(termCh)

	// Compute cumulative partial sums for the links.
	links := make([]complex128, v+1) // v partial links plus one for the final corrected value.
	cumulative := 0.0
	for k := 0; k < v; k++ {
		cumulative += terms[k]
		links[k] = complex(2*cumulative, 0)
		log.Printf("Cumulative sum at k=%d: %f", k, 2*cumulative)
	}

	// Final result: total = 2*cumulative + b. Multiply by exp(-iV) as in the C# version.
	totalReal := 2*cumulative + b
	factor := cmplx.Exp(complex(0, -V))
	total := totalReal * factor
	// Adjust all links by the same factor.
	for k := 0; k < v; k++ {
		links[k] *= factor
	}
	// Append final total as the last link.
	links[v] = total
	log.Printf("Final Reimann–Siegel result: %v", total)
	return total, links
}

// plotLinks renders the chain of links as a PNG image.
// It divides the work among a fixed number of worker goroutines which send
// their rendered image fragments back via a channel and then these are composited.
func plotLinks(links []complex128, zeta complex128) {
	const numWorkers = 24
	const outputSize = 2048

	if len(links) == 0 {
		log.Println("No links provided for plotting.")
		return
	}
	// Determine the x- and y-ranges.
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
	log.Printf("Link X range: [%f, %f], Y range: [%f, %f]", minX, maxX, minY, maxY)

	// Define a worker result to capture each worker's RGBA image.
	type workerResult struct {
		index int
		img   *image.RGBA
	}
	workerCh := make(chan workerResult, numWorkers)

	chunkSize := (len(links) + numWorkers - 1) / numWorkers
	for i := 0; i < numWorkers; i++ {
		start := i * chunkSize
		end := start + chunkSize
		if end > len(links) {
			end = len(links)
		}
		go func(worker, start, end int) {
			log.Printf("Worker %d processing links from index %d to %d", worker, start, end)
			img := image.NewRGBA(image.Rect(0, 0, outputSize, outputSize))
			// Clear image: initialize with fully transparent pixels.
			for y := 0; y < outputSize; y++ {
				for x := 0; x < outputSize; x++ {
					img.Set(x, y, color.RGBA{0, 0, 0, 0})
				}
			}
			var prevX, prevY float64
			for j := start; j < end; j++ {
				// Normalize coordinates to the image space.
				x := (real(links[j]) - minX) / (maxX - minX) * float64(outputSize)
				y := (imag(links[j]) - minY) / (maxY - minY) * float64(outputSize)
				y = float64(outputSize) - y // invert Y (image coordinates)
				if j == start {
					prevX = x
					prevY = y
				} else {
					// Draw a simple line between the previous point and the current.
					drawLine(img, int(prevX), int(prevY), int(x), int(y), color.RGBA{255, 255, 255, 64})
					prevX = x
					prevY = y
				}
			}
			workerCh <- workerResult{worker, img}
		}(i, start, end)
	}

	// Collect all worker images.
	workerImages := make([]*image.RGBA, numWorkers)
	for i := 0; i < numWorkers; i++ {
		res := <-workerCh
		log.Printf("Received image from worker %d", res.index)
		workerImages[res.index] = res.img
	}
	close(workerCh)

	// Create the base image with a dark grey background.
	finalImage := image.NewRGBA(image.Rect(0, 0, outputSize, outputSize))
	draw.Draw(finalImage, finalImage.Bounds(), &image.Uniform{color.RGBA{30, 30, 30, 255}}, image.Point{}, draw.Src)

	// Composite each worker's image on top.
	for i, img := range workerImages {
		log.Printf("Compositing worker %d image", i)
		draw.Draw(finalImage, img.Bounds(), img, image.Point{}, draw.Over)
	}

	// Save the final image.
	outFile, err := os.Create("reimann_links.png")
	if err != nil {
		log.Fatalf("failed to create output file: %v", err)
	}
	defer outFile.Close()

	if err := png.Encode(outFile, finalImage); err != nil {
		log.Fatalf("failed to encode image: %v", err)
	}
	log.Println("Image saved as reimann_links.png")
}

// drawLine implements a simple Bresenham line algorithm.
func drawLine(img *image.RGBA, x0, y0, x1, y1 int, col color.RGBA) {
	dx := math.Abs(float64(x1 - x0))
	dy := math.Abs(float64(y1 - y0))
	sx := -1
	sy := -1
	if x0 < x1 {
		sx = 1
	}
	if y0 < y1 {
		sy = 1
	}
	errVal := int(dx - dy)
	for {
		img.Set(x0, y0, col)
		if x0 == x1 && y0 == y1 {
			break
		}
		e2 := 2 * errVal
		if e2 > -int(dy) {
			errVal -= int(dy)
			x0 += sx
		}
		if e2 < int(dx) {
			errVal += int(dx)
			y0 += sy
		}
	}
}

func main() {
	// Read the command-line flag for the imaginary part.
	imagPart := flag.Float64("imag", 14.135, "Imaginary part for input complex number (will use ζ(0.5 + i*imag))")
	flag.Parse()

	s := complex(0.5, *imagPart)
	log.Printf("Input complex number: %v", s)

	startTime := time.Now()
	total, links := reimannSiegelWithLinks(s)
	elapsed := time.Since(startTime)
	fps := 1.0 / elapsed.Seconds()
	fmt.Printf("Reimann–Siegel result: (%.6f, %.6f)\n", real(total), imag(total))
	fmt.Printf("Calculation time: %v, FPS: %.2f\n", elapsed, fps)
	fmt.Printf("Number of links: %d\n", len(links))

	// Prepend the origin (0,0) to the links (if desired for plotting).
	links = append([]complex128{complex(0, 0)}, links...)

	startPlot := time.Now()
	log.Println("Plotting links using Reimann–Siegel approach...")
	plotLinks(links, total)
	plotElapsed := time.Since(startPlot)
	fps = 1.0 / plotElapsed.Seconds()
	fmt.Printf("Plotting time: %v, FPS: %.2f\n", plotElapsed, fps)
}
