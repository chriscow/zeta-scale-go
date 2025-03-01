package main

import (
	"fmt"
	"math/cmplx"
	"testing"
)

func BenchmarkCalculateSpiralPartialSums(b *testing.B) {
	// Test parameters
	s := complex(0.5, 6_300_000.0)

	// Test different chunk sizes in powers of 2
	chunkSizes := []int{1, 2, 4, 8, 16, 32, 64, 128, 256, 512, 1024, 2048, 4096}

	for _, numChunks := range chunkSizes {
		b.Run(fmt.Sprintf("chunks=%d", numChunks), func(b *testing.B) {
			// Calculate N
			N := int(cmplx.Abs(s))
			if N < MinN {
				N = MinN
			} else if N > MaxN {
				N = MaxN
			}

			// Override the dynamic chunk size for this test
			originalChunkSize := ChunkSize
			ChunkSize = (N + numChunks - 1) / numChunks

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				result, links := calculateSpiralPartialSums(s)
				// Prevent compiler optimization
				if real(result) == 0 && len(links) == 0 {
					b.Fatal("unexpected zero result")
				}
			}

			// Restore original chunk size
			ChunkSize = originalChunkSize
		})
	}
}
