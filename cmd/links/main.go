package main

import (
	"flag"
	"fmt"
	"math/cmplx"
	"sync"
)

// Constants for the Euler-Maclaurin summation
const (
	MinN      = 100
	MaxN      = 1000000
	ChunkSize = 1000
)

// Function to compute a partial sum for a given range and track links
func computePartialSumWithLinks(start, end int, s complex128, results chan<- complex128, links chan<- []complex128, wg *sync.WaitGroup) {
	defer wg.Done()
	partialSum := complex(0, 0)
	var linkList []complex128

	for k := start; k < end; k++ {
		term := cmplx.Pow(complex(float64(k), 0), -s)
		partialSum += term
		linkList = append(linkList, partialSum) // Track the cumulative sum as a link
	}

	results <- partialSum
	links <- linkList
}

// Function to compute the Euler-Maclaurin summation and track links
func eulerMaclaurinWithLinks(s complex128) (complex128, []complex128) {
	N := int(cmplx.Abs(s))
	if N < MinN {
		N = MinN
	} else if N > MaxN {
		N = MaxN
	}

	// Channels to collect partial sums and links
	results := make(chan complex128, (N+ChunkSize-1)/ChunkSize)
	links := make(chan []complex128, (N+ChunkSize-1)/ChunkSize)
	var wg sync.WaitGroup

	// Launch goroutines to compute partial sums and track links
	for start := 1; start < N; start += ChunkSize {
		end := start + ChunkSize
		if end > N {
			end = N
		}
		wg.Add(1)
		go computePartialSumWithLinks(start, end, s, results, links, &wg)
	}

	// Wait for all goroutines to finish
	go func() {
		wg.Wait()
		close(results)
		close(links)
	}()

	// Aggregate the partial sums and links
	totalSum := complex(0, 0)
	var allLinks []complex128
	for partialSum := range results {
		totalSum += partialSum
	}
	for linkList := range links {
		allLinks = append(allLinks, linkList...)
	}

	// Apply correction terms
	term1 := cmplx.Pow(complex(float64(N), 0), 1-s) / (s - 1)
	term2 := 0.5 * cmplx.Pow(complex(float64(N), 0), -s)
	totalSum += term1 + term2

	return totalSum, allLinks
}

func main() {
	// Define a flag for the imaginary part of the complex number
	imagPart := flag.Float64("imag", 14.135, "Imaginary part of the complex number")
	flag.Parse()

	// Example complex number with real part 0.5 and user-provided imaginary part
	s := complex(0.5, *imagPart)

	// Compute the Euler-Maclaurin summation and track links
	result, links := eulerMaclaurinWithLinks(s)

	// Output the result
	fmt.Printf("Euler-Maclaurin result: (%.6f, %.6f)\n", real(result), imag(result))

	// Output the links
	fmt.Println("Links:")
	for i, link := range links {
		fmt.Printf("Link %d: (%.6f, %.6f)\n", i+1, real(link), imag(link))
	}
}
