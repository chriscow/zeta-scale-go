package compression

import (
	"compress/gzip"
	"encoding/binary"
	"log"
	"math"
	"os"
)

// DeltaCompressed represents a spiral compressed using delta encoding
type DeltaCompressed struct {
	// Store first point in full precision
	StartX, StartY float64
	// Store scale factors for the deltas
	ScaleX, ScaleY float64
	// Number of points
	NumPoints uint32
	// Packed deltas using int16 for efficiency
	Deltas []int16
}

// CompressWithDelta compresses the points using delta encoding
func CompressWithDelta(points []complex128) (*DeltaCompressed, error) {
	if len(points) == 0 {
		return nil, nil
	}

	log.Printf("Starting delta compression of %d points", len(points))

	// Initialize with first point
	compressed := &DeltaCompressed{
		StartX:    real(points[0]),
		StartY:    imag(points[0]),
		NumPoints: uint32(len(points)),
	}

	// Calculate ranges to determine optimal scale factors
	minDx, maxDx := 0.0, 0.0
	minDy, maxDy := 0.0, 0.0

	for i := 1; i < len(points); i++ {
		dx := real(points[i]) - real(points[i-1])
		dy := imag(points[i]) - imag(points[i-1])
		minDx = math.Min(minDx, dx)
		maxDx = math.Max(maxDx, dx)
		minDy = math.Min(minDy, dy)
		maxDy = math.Max(maxDy, dy)
	}

	log.Printf("Delta ranges - X: [%f, %f], Y: [%f, %f]", minDx, maxDx, minDy, maxDy)

	// Calculate scale factors to fit within int16 range
	// Use 90% of int16 range to avoid edge cases
	compressed.ScaleX = math.Max(math.Abs(minDx), math.Abs(maxDx)) / 29000.0
	compressed.ScaleY = math.Max(math.Abs(minDy), math.Abs(maxDy)) / 29000.0

	if compressed.ScaleX == 0 {
		compressed.ScaleX = 1.0
	}
	if compressed.ScaleY == 0 {
		compressed.ScaleY = 1.0
	}

	log.Printf("Using scale factors - X: %f, Y: %f", compressed.ScaleX, compressed.ScaleY)

	// Encode deltas
	compressed.Deltas = make([]int16, (len(points)-1)*2)
	for i := 1; i < len(points); i++ {
		dx := real(points[i]) - real(points[i-1])
		dy := imag(points[i]) - imag(points[i-1])

		compressed.Deltas[(i-1)*2] = int16(dx / compressed.ScaleX)
		compressed.Deltas[(i-1)*2+1] = int16(dy / compressed.ScaleY)
	}

	log.Printf("Successfully compressed to %d deltas", len(compressed.Deltas))
	return compressed, nil
}

// SaveDeltaCompressed saves the compressed data to a file with gzip compression
func SaveDeltaCompressed(compressed *DeltaCompressed, filename string) error {
	log.Printf("Starting to save delta compressed data to %s", filename)

	file, err := os.Create(filename)
	if err != nil {
		log.Printf("Error creating file: %v", err)
		return err
	}
	defer file.Close()

	gzw := gzip.NewWriter(file)
	defer gzw.Close()

	// Write header
	if err := binary.Write(gzw, binary.LittleEndian, compressed.StartX); err != nil {
		log.Printf("Error writing StartX: %v", err)
		return err
	}
	if err := binary.Write(gzw, binary.LittleEndian, compressed.StartY); err != nil {
		log.Printf("Error writing StartY: %v", err)
		return err
	}
	if err := binary.Write(gzw, binary.LittleEndian, compressed.ScaleX); err != nil {
		log.Printf("Error writing ScaleX: %v", err)
		return err
	}
	if err := binary.Write(gzw, binary.LittleEndian, compressed.ScaleY); err != nil {
		log.Printf("Error writing ScaleY: %v", err)
		return err
	}
	if err := binary.Write(gzw, binary.LittleEndian, compressed.NumPoints); err != nil {
		log.Printf("Error writing NumPoints: %v", err)
		return err
	}

	// Write deltas
	if err := binary.Write(gzw, binary.LittleEndian, compressed.Deltas); err != nil {
		log.Printf("Error writing Deltas: %v", err)
		return err
	}

	if err := gzw.Close(); err != nil {
		log.Printf("Error closing gzip writer: %v", err)
		return err
	}

	log.Printf("Successfully saved delta compressed data")
	return nil
}

// LoadDeltaCompressed loads compressed data from a file
func LoadDeltaCompressed(filename string) (*DeltaCompressed, error) {
	log.Printf("Starting to load delta compressed data from %s", filename)

	file, err := os.Open(filename)
	if err != nil {
		log.Printf("Error opening file: %v", err)
		return nil, err
	}
	defer file.Close()

	gzr, err := gzip.NewReader(file)
	if err != nil {
		log.Printf("Error creating gzip reader: %v", err)
		return nil, err
	}
	defer gzr.Close()

	compressed := &DeltaCompressed{}

	// Read header
	if err := binary.Read(gzr, binary.LittleEndian, &compressed.StartX); err != nil {
		log.Printf("Error reading StartX: %v", err)
		return nil, err
	}
	if err := binary.Read(gzr, binary.LittleEndian, &compressed.StartY); err != nil {
		log.Printf("Error reading StartY: %v", err)
		return nil, err
	}
	if err := binary.Read(gzr, binary.LittleEndian, &compressed.ScaleX); err != nil {
		log.Printf("Error reading ScaleX: %v", err)
		return nil, err
	}
	if err := binary.Read(gzr, binary.LittleEndian, &compressed.ScaleY); err != nil {
		log.Printf("Error reading ScaleY: %v", err)
		return nil, err
	}
	if err := binary.Read(gzr, binary.LittleEndian, &compressed.NumPoints); err != nil {
		log.Printf("Error reading NumPoints: %v", err)
		return nil, err
	}

	// Read deltas
	compressed.Deltas = make([]int16, (compressed.NumPoints-1)*2)
	if err := binary.Read(gzr, binary.LittleEndian, &compressed.Deltas); err != nil {
		log.Printf("Error reading Deltas: %v", err)
		return nil, err
	}

	log.Printf("Successfully loaded %d points", compressed.NumPoints)
	return compressed, nil
}

// Decompress converts the compressed data back to points
func (c *DeltaCompressed) Decompress() []complex128 {
	points := make([]complex128, c.NumPoints)
	points[0] = complex(c.StartX, c.StartY)

	for i := 1; i < int(c.NumPoints); i++ {
		dx := float64(c.Deltas[(i-1)*2]) * c.ScaleX
		dy := float64(c.Deltas[(i-1)*2+1]) * c.ScaleY
		points[i] = complex(
			real(points[i-1])+dx,
			imag(points[i-1])+dy,
		)
	}

	return points
}
