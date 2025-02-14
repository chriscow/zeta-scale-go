package main

import (
	"encoding/json"
	"flag"
	"log"
	"math/cmplx"

	"github.com/nats-io/nats.go"
)

// Certainly! To implement a distributed system where the client divides the
// summation into smaller chunks and uses NATS workers to compute these chunks,
// followed by a final worker to combine the results, you can follow this
// architecture: Client: Divides the summation range into smaller chunks and
// sends each chunk as a separate request to the NATS workers. Worker: Computes
// the partial summation for the assigned chunk and sends the result back to a
// designated subject for reduction. Reducer Worker: Listens for partial
// results, combines them, and computes the final result.

// PartialSumRequest and PartialSumResponse must match the types defined in client.go.
type PartialSumRequest struct {
	Start int       `json:"start"`
	End   int       `json:"end"`
	S     complex64 `json:"s"`
}

type PartialSumResponse struct {
	PartialSum complex64 `json:"partialSum"`
}

func main() {
	natsURL := flag.String("nats", nats.DefaultURL, "NATS server URL")
	subject := flag.String("subject", "euler.maclaurin.partial", "Subject to subscribe")
	reduceSubject := flag.String("reduceSubject", "euler.maclaurin.reduce", "Subject for reduction")
	flag.Parse()

	nc, err := nats.Connect(*natsURL)
	if err != nil {
		log.Fatalf("Error connecting to NATS: %v", err)
	}
	defer nc.Close()
	log.Printf("Worker connected to NATS at %s", *natsURL)

	// Subscribe to the subject with a worker (queue) group.
	_, err = nc.QueueSubscribe(*subject, "workers", func(msg *nats.Msg) {
		log.Printf("Received partial sum request: %s", string(msg.Data))

		var req PartialSumRequest
		if err := json.Unmarshal(msg.Data, &req); err != nil {
			log.Printf("Error unmarshalling request: %v", err)
			return
		}

		// Compute the partial summation.
		partialSum := complex64(0)
		for k := req.Start; k < req.End; k++ {
			partialSum += complex64(cmplx.Pow(complex(float64(k), 0), -complex128(req.S)))
		}

		resp := PartialSumResponse{
			PartialSum: partialSum,
		}

		respData, err := json.Marshal(resp)
		if err != nil {
			log.Printf("Error marshalling response: %v", err)
			return
		}

		// Publish the partial result to the reducer.
		nc.Publish(*reduceSubject, respData)
		log.Printf("Published partial sum result: partialSum=(%.6f, %.6f)", real(partialSum), imag(partialSum))
	})
	if err != nil {
		log.Fatalf("Error subscribing to subject: %v", err)
	}

	// Keep the worker running indefinitely.
	select {}
}
