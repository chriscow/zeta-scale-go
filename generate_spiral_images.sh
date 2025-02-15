#!/bin/bash

# Array of aggressiveness values to test
aggressive_values=(0.0 1.0 2.0 3.0 3.5 4.0)

# Default parameters
imag_value=6300000.0
size=2048
maxN=65000000000

echo "Starting spiral image generation..."
echo "This will create $(( ${#aggressive_values[@]} * 2 )) images..."

go run cmd/spiral/main.go \
    -imag=${imag_value} \
    -maxN=${maxN} \
    -size=${size} \
    -output="no-downsample.png"

go run cmd/spiral/main.go \
    -imag=${imag_value} \
    -maxN=${maxN} \
    -size=${size} \
    -output="no-downsample-points.png" \
    -points=true

for aggressive in "${aggressive_values[@]}"; do
    echo "Generating images for aggressiveness ${aggressive}..."
    
    # Generate regular image with lines
    echo "  Creating line version..."
    go run cmd/spiral/main.go \
        -imag=${imag_value} \
        -maxN=${maxN} \
        -size=${size} \
        -aggressive=${aggressive} \
        -downsample=true \
        -output="downsample_${aggressive}.png"
    
    # Generate points-only version
    echo "  Creating points-only version..."
    go run cmd/spiral/main.go \
        -imag=${imag_value} \
        -maxN=${maxN} \
        -size=${size} \
        -aggressive=${aggressive} \
        -downsample=true \
        -points=true \
        -output="downsample_${aggressive}-points.png"
done

echo "Done! Generated $(( ${#aggressive_values[@]} * 2 )) images:"
for aggressive in "${aggressive_values[@]}"; do
    echo "  - downsample_${aggressive}.png"
    echo "  - downsample_${aggressive}-points.png"
done 