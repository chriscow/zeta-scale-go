# zeta-scale-go: High-Performance Riemann Zeta Function Visualization

A high-performance Go implementation for visualizing the Riemann zeta function through spiral plots. This project generates beautiful mathematical visualizations by computing and plotting partial sums of the Riemann zeta function in the complex plane.

## Features

- **High-Performance Computing**: Utilizes multi-threaded computation for calculating partial sums
- **Adaptive Downsampling**: Intelligent point reduction while maintaining visual quality
- **Beautiful Visualizations**: Generates high-quality PNG images of zeta function spirals
- **Configurable Output**: Multiple parameters to customize the visualization
- **Memory Efficient**: Smart memory management for handling large datasets

## Requirements

- Go 1.23.4 or later
- MacOS (for font loading - can be modified for other platforms)
- Arial Unicode font installed (typically at `/Library/Fonts/Arial Unicode.ttf`)

## Installation

1. Clone the repository:
   ```bash
   git clone https://github.com/yourusername/zest-go.git
   cd zeta-scale-go
   ```

2. Install dependencies:
   ```bash
   go mod download
   ```

## Usage

The main program accepts several command-line flags to customize the visualization:

```bash
go run cmd/spiral/main.go [flags]
```

### Available Flags

- `-imag float`: Imaginary part of the complex number (default: 6,300,000.0)
- `-maxN int`: Maximum number of terms to compute (default: 65,000,000,000)
- `-downsample`: Enable downsampling of links (default: false)
- `-aggressive float`: Downsampling aggressiveness (0.0-1.0, default: 0.5)
- `-output string`: Output filename for the image (default: "combined_links.png")
- `-size int`: Output image size in pixels (default: 2048)
- `-debug`: Enable debug logging (default: false)
- `-points`: Draw points only, no lines (default: false)

### Example Commands

1. Basic visualization:
   ```bash
   go run cmd/spiral/main.go
   ```

2. High-resolution visualization with downsampling:
   ```bash
   go run cmd/spiral/main.go -size 4096 -downsample -aggressive 2.0
   ```

3. Points-only visualization with custom imaginary part:
   ```bash
   go run cmd/spiral/main.go -points -imag 1000000.0
   ```

## Performance Optimization

The program includes several optimizations:

1. **Parallel Processing**: Uses Go's concurrency features for computation
2. **Adaptive Downsampling**: Reduces point count while preserving visual quality
3. **Memory Management**: Efficient handling of large datasets
4. **Worker Pools**: Optimized image composition using worker pools

## Technical Details

### Computation Method

The program uses the Euler-Maclaurin summation formula to compute partial sums of the Riemann zeta function. The computation is split into chunks and processed in parallel using goroutines.

### Visualization

The visualization process includes:
- Complex plane mapping
- Adaptive point interpolation
- Multi-threaded rendering
- Additive blending for smooth visuals

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the [LICENSE] file for details.

## Acknowledgments

- Thanks to the Go team for the excellent concurrency support
- The mathematical visualization community for inspiration 