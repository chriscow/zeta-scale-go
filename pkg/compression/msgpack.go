package compression

import (
	"compress/gzip"
	"log"
	"os"

	"github.com/vmihailenco/msgpack/v5"
)

// MsgPackSpiral represents a spiral for MessagePack encoding
type MsgPackSpiral struct {
	// Metadata for efficient rendering
	Bounds struct {
		MinX float32 `msgpack:"minX"`
		MaxX float32 `msgpack:"maxX"`
		MinY float32 `msgpack:"minY"`
		MaxY float32 `msgpack:"maxY"`
	} `msgpack:"bounds"`

	// Quantization factors
	Scale struct {
		X float32 `msgpack:"x"`
		Y float32 `msgpack:"y"`
	} `msgpack:"scale"`

	// Store points as quantized int16 values for better compression
	// Format: [x1,y1,x2,y2,...]
	Points []int16 `msgpack:"points"`
}

// CompressWithMsgPack compresses the points using MessagePack
func CompressWithMsgPack(points []complex128) (*MsgPackSpiral, error) {
	log.Printf("Starting MessagePack compression of %d points", len(points))

	compressed := &MsgPackSpiral{
		Points: make([]int16, len(points)*2),
	}

	// Calculate bounds while converting points
	compressed.Bounds.MinX = float32(real(points[0]))
	compressed.Bounds.MaxX = float32(real(points[0]))
	compressed.Bounds.MinY = float32(imag(points[0]))
	compressed.Bounds.MaxY = float32(imag(points[0]))

	// First pass: find bounds
	for _, p := range points {
		x, y := float32(real(p)), float32(imag(p))
		compressed.Bounds.MinX = min32(compressed.Bounds.MinX, x)
		compressed.Bounds.MaxX = max32(compressed.Bounds.MaxX, x)
		compressed.Bounds.MinY = min32(compressed.Bounds.MinY, y)
		compressed.Bounds.MaxY = max32(compressed.Bounds.MaxY, y)
	}

	// Calculate scale factors to map to int16 range (-32768 to 32767)
	// Use 90% of range for safety
	const safeRange = 29000
	compressed.Scale.X = (compressed.Bounds.MaxX - compressed.Bounds.MinX) / float32(safeRange)
	compressed.Scale.Y = (compressed.Bounds.MaxY - compressed.Bounds.MinY) / float32(safeRange)

	if compressed.Scale.X == 0 {
		compressed.Scale.X = 1
	}
	if compressed.Scale.Y == 0 {
		compressed.Scale.Y = 1
	}

	// Second pass: quantize points
	for i, p := range points {
		x := float32(real(p))
		y := float32(imag(p))

		// Quantize to int16
		qx := int16((x - compressed.Bounds.MinX) / compressed.Scale.X)
		qy := int16((y - compressed.Bounds.MinY) / compressed.Scale.Y)

		compressed.Points[i*2] = qx
		compressed.Points[i*2+1] = qy
	}

	// Test marshal to ensure it works
	data, err := msgpack.Marshal(compressed)
	if err != nil {
		log.Printf("Error during test marshal: %v", err)
		return nil, err
	}
	log.Printf("Successfully compressed %d points with MessagePack (%.2f MB)",
		len(points), float64(len(data))/(1024*1024))

	return compressed, nil
}

// SaveMsgPack saves the compressed data to a file with gzip compression
func SaveMsgPack(compressed *MsgPackSpiral, filename string) error {
	log.Printf("Starting to save MessagePack data to %s", filename)

	// Encode with MessagePack
	data, err := msgpack.Marshal(compressed)
	if err != nil {
		log.Printf("Error marshaling data: %v", err)
		return err
	}
	log.Printf("MessagePack encoded size: %d bytes", len(data))

	// Save with gzip compression
	file, err := os.Create(filename)
	if err != nil {
		log.Printf("Error creating file: %v", err)
		return err
	}
	defer file.Close()

	gzw := gzip.NewWriter(file)
	defer gzw.Close()

	n, err := gzw.Write(data)
	if err != nil {
		log.Printf("Error writing compressed data: %v", err)
		return err
	}

	if err := gzw.Close(); err != nil {
		log.Printf("Error closing gzip writer: %v", err)
		return err
	}

	log.Printf("Successfully wrote %d bytes of MessagePack data", n)
	return nil
}

// LoadMsgPack loads compressed data from a file
func LoadMsgPack(filename string) (*MsgPackSpiral, error) {
	log.Printf("Starting to load MessagePack data from %s", filename)

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

	// Read all data
	data := make([]byte, 0, 1024*1024) // Pre-allocate 1MB
	buf := make([]byte, 32*1024)       // 32KB read buffer
	totalRead := 0

	for {
		n, err := gzr.Read(buf)
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			log.Printf("Error reading data: %v", err)
			return nil, err
		}
		data = append(data, buf[:n]...)
		totalRead += n
	}

	log.Printf("Read %d bytes of compressed data", totalRead)

	// Decode MessagePack
	var compressed MsgPackSpiral
	err = msgpack.Unmarshal(data, &compressed)
	if err != nil {
		log.Printf("Error unmarshaling data: %v", err)
		return nil, err
	}

	log.Printf("Successfully loaded %d points", len(compressed.Points)/2)
	return &compressed, nil
}

// Decompress converts the compressed data back to points
func (c *MsgPackSpiral) Decompress() []complex128 {
	points := make([]complex128, len(c.Points)/2)
	for i := 0; i < len(points); i++ {
		// Dequantize points
		x := float64(c.Bounds.MinX + (float32(c.Points[i*2]) * c.Scale.X))
		y := float64(c.Bounds.MinY + (float32(c.Points[i*2+1]) * c.Scale.Y))
		points[i] = complex(x, y)
	}
	return points
}

func min32(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}

func max32(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}
