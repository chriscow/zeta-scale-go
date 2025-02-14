package main

import (
	"encoding/json"
	"flag"
	"log"
	"sync"

	"github.com/nats-io/nats.go"
)

// 3. Reducer Worker The reducer worker listens for partial results and combines
// them into a final result.

// PartialSumResponse must match the type defined in worker.go.
type PartialSumResponse struct {
	PartialSum complex64 `json:"partialSum"`
}

func main() {
	natsURL := flag.String("nats", nats.DefaultURL, "NATS server URL")
	reduceSubject := flag.String("reduceSubject", "euler.maclaurin.reduce", "Subject to subscribe")
	finalSubject := flag.String("finalSubject", "euler.maclaurin.final", "Subject for final result")
	flag.Parse()

	nc, err := nats.Connect(*natsURL)
	if err != nil {
		log.Fatalf("Error connecting to NATS: %v", err)
	}
	defer nc.Close()
	log.Printf("Reducer connected to NATS at %s", *natsURL)

	var finalSum complex64
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Subscribe to the reduce subject to collect partial results.
	_, err = nc.Subscribe(*reduceSubject, func(msg *nats.Msg) {
		var resp PartialSumResponse
		if err := json.Unmarshal(msg.Data, &resp); err != nil {
			log.Printf("Error unmarshalling partial result: %v", err)
			return
		}

		mu.Lock()
		finalSum += resp.PartialSum
		mu.Unlock()

		log.Printf("Received partial sum: partialSum=(%.6f, %.6f)", real(resp.PartialSum), imag(resp.PartialSum))
		wg.Done()
	})
	if err != nil {
		log.Fatalf("Error subscribing to reduce subject: %v", err)
	}

	// Wait for all partial results to be collected.
	wg.Add(10) // Adjust this number based on the expected number of partial results.
	wg.Wait()

	// Publish the final result.
	finalResp := PartialSumResponse{
		PartialSum: finalSum,
	}
	finalRespData, err := json.Marshal(finalResp)
	if err != nil {
		log.Fatalf("Error marshalling final result: %v", err)
	}

	nc.Publish(*finalSubject, finalRespData)
	log.Printf("Published final result: partialSum=(%.6f, %.6f)", real(finalSum), imag(finalSum))
}
