package main

import (
	"encoding/json"
	"flag"
	"log"
	"math"
	"time"

	"github.com/nats-io/nats.go"
)

// Explanation Client: Divides the summation into chunks and sends each chunk to
// the workers. It waits for the final result from the reducer. Worker: Computes
// the partial summation for its assigned chunk and publishes the result to a
// reduction subject. Reducer Worker: Listens for partial results, combines
// them, and publishes the final result. It uses a sync.WaitGroup to wait for
// all expected partial results before publishing the final result. NATS
// Subjects: "euler.maclaurin.partial": Used by the client to send partial sum
// requests to workers. "euler.maclaurin.reduce": Used by workers to send
// partial results to the reducer. "euler.maclaurin.final": Used by the reducer
// to publish the final result. This setup allows you to distribute the
// computation across multiple workers, leveraging parallelism both in the cloud
// and on the GPU. Adjust the number of chunks and the wg.Add() value in the
// reducer based on your specific requirements and the number of workers you
// have.

// The Euler–Maclaurin formula is a method for approximating sums, and it
// involves both a direct summation of terms and correction terms that account
// for the difference between the sum and the integral approximation. When
// considering parallelization, it's important to understand which parts of the
// computation can be distributed independently. Independence of Partial
// Summations Direct Summation: The direct summation of terms ∑ k = 1 N − 1 k −
// s ∑ k=1 N−1 ​
//  k
// −s
//   can be divided into independent chunks. Each worker can compute a partial sum over a subset of the range without needing the results from other workers. This is because the summation is simply an accumulation of terms, and the order of addition does not affect the final result.
// Correction Terms: The correction terms in the Euler–Maclaurin formula, such
// as N 1 − s / ( s − 1 ) N 1−s
//  /(s−1) and the secondary correction sum involving Bernoulli numbers, are typically computed once for the entire range and are not easily parallelizable in the same way as the direct summation. These terms depend on the entire range and the properties of the function being approximated.
// Parallelization Strategy Given the above, you can parallelize the direct
// summation across multiple workers, but the correction terms should be applied
// after all partial sums have been computed and combined. Here's a refined
// strategy: Workers: Each worker computes a partial sum for a specific range of
// terms. These partial sums are independent and can be computed in parallel.
// Reducer: After collecting all partial sums, the reducer combines them and
// then applies the correction terms to compute the final result. This approach
// ensures that the parallelization is valid and that the final result is
// accurate. The correction terms are applied only once, after all partial sums
// have been aggregated, ensuring that the approximation is consistent with the
// Euler–Maclaurin formula.

// PartialSumRequest represents a request to compute a partial sum.
type PartialSumRequest struct {
	Start int       `json:"start"`
	End   int       `json:"end"`
	S     complex64 `json:"s"`
}

// PartialSumResponse represents a response with a partial sum result.
type PartialSumResponse struct {
	PartialSum complex64 `json:"partialSum"`
}

func main() {
	natsURL := flag.String("nats", nats.DefaultURL, "NATS server URL")
	subject := flag.String("subject", "euler.maclaurin.partial", "Subject to publish")
	reduceSubject := flag.String("reduceSubject", "euler.maclaurin.reduce", "Subject for reduction")
	flag.Parse()

	nc, err := nats.Connect(*natsURL)
	if err != nil {
		log.Fatalf("Error connecting to NATS: %v", err)
	}
	defer nc.Close()
	log.Printf("Client connected to NATS at %s", *natsURL)

	// Define the complex number for which to compute the summation.
	s := complex(0.5, 14.135)
	N := int(math.Abs(float64(s)))
	chunkSize := 1000 // Define the size of each chunk.

	// Send requests for each chunk.
	for start := 1; start < N; start += chunkSize {
		end := start + chunkSize
		if end > N {
			end = N
		}

		req := PartialSumRequest{
			Start: start,
			End:   end,
			S:     complex64(s),
		}
		reqData, err := json.Marshal(req)
		if err != nil {
			log.Fatalf("Error marshalling request: %v", err)
		}

		// Publish the request to the workers.
		nc.Publish(*subject, reqData)
		log.Printf("Published partial sum request: start=%d, end=%d", start, end)
	}

	// Optionally, wait for the final result from the reducer.
	msg, err := nc.Request(*reduceSubject, nil, 10*time.Second)
	if err != nil {
		log.Fatalf("Error waiting for final result: %v", err)
	}

	var finalResult PartialSumResponse
	if err := json.Unmarshal(msg.Data, &finalResult); err != nil {
		log.Fatalf("Error unmarshalling final result: %v", err)
	}

	log.Printf("Final result: partialSum=(%.6f, %.6f)", real(finalResult.PartialSum), imag(finalResult.PartialSum))
}
