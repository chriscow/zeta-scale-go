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
	ChunkSize = 1_000_000
)

// Function to compute a partial sum for a given range
func computePartialSum(start, end int, s complex128, results chan<- complex128, wg *sync.WaitGroup) {
	defer wg.Done()
	partialSum := complex(0, 0)
	for k := start; k < end; k++ {
		partialSum += cmplx.Pow(complex(float64(k), 0), -s)
	}
	results <- partialSum
}

// Function to compute the Euler-Maclaurin summation
func eulerMaclaurin(s complex128) complex128 {
	N := int(cmplx.Abs(s))
	// if N < MinN {
	// 	N = MinN
	// } else if N > MaxN {
	// 	N = MaxN
	// }

	// Channel to collect partial sums
	results := make(chan complex128, (N+ChunkSize-1)/ChunkSize)
	var wg sync.WaitGroup

	// Launch goroutines to compute partial sums
	for start := 1; start < N; start += ChunkSize {
		end := start + ChunkSize
		if end > N {
			end = N
		}
		wg.Add(1)
		go computePartialSum(start, end, s, results, &wg)
	}

	// Wait for all goroutines to finish
	go func() {
		wg.Wait()
		close(results)
	}()

	// Aggregate the partial sums
	totalSum := complex(0, 0)
	for partialSum := range results {
		totalSum += partialSum
	}

	// Apply correction terms
	term1 := cmplx.Pow(complex(float64(N), 0), 1-s) / (s - 1)
	term2 := 0.5 * cmplx.Pow(complex(float64(N), 0), -s)
	totalSum += term1 + term2

	return totalSum
}

func main() {
	// Define a flag for the imaginary part of the complex number
	imagPart := flag.Float64("imag", 14.135, "Imaginary part of the complex number")
	flag.Parse()

	// Example complex number with real part 0.5 and user-provided imaginary part
	s := complex(0.5, *imagPart)

	// Compute the Euler-Maclaurin summation
	result := eulerMaclaurin(s)

	// Output the result
	fmt.Printf("Euler-Maclaurin result: (%.6f, %.6f)\n", real(result), imag(result))
}
