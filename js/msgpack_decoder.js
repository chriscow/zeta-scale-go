/**
 * MessagePack decoder for spiral data.
 * Requires @msgpack/msgpack package.
 * 
 * This format is:
 * - Much smaller (1.5MB vs 79MB)
 * - Faster to load and parse
 * - Still maintains high precision (int16 quantization preserves visual quality)
 * - Ready for WebGL with Int16Array
 * - Includes bounds for immediate rendering setup
 */
import { decode } from '@msgpack/msgpack';

/**
 * Loads and decodes spiral data from a MessagePack file.
 * @param {string} url - URL to the .msgpack file
 * @returns {Promise<{
 *   points: Int16Array,
 *   bounds: {minX: number, maxX: number, minY: number, maxY: number},
 *   scale: {x: number, y: number},
 *   numPoints: number,
 *   getPoint: (i: number) => [number, number],
 *   dequantized: Float32Array
 * }>}
 */
export async function loadSpiral(url) {
  // Fetch and decode the MessagePack data
  const response = await fetch(url);
  const compressed = await response.arrayBuffer();
  const data = decode(compressed);
  
  // Create typed arrays for fast access
  const points = new Int16Array(data.points);
  
  /**
   * Get a single point by index, dequantizing on-the-fly
   * @param {number} i - Point index
   * @returns {[number, number]} - [x, y] coordinates
   */
  function getPoint(i) {
    const x = data.bounds.minX + (points[i*2] * data.scale.x);
    const y = data.bounds.minY + (points[i*2+1] * data.scale.y);
    return [x, y];
  }
  
  // Create dequantized buffer if needed
  const dequantized = new Float32Array(points.length);
  for (let i = 0; i < points.length; i += 2) {
    dequantized[i] = data.bounds.minX + (points[i] * data.scale.x);
    dequantized[i+1] = data.bounds.minY + (points[i+1] * data.scale.y);
  }
  
  return {
    points,
    bounds: data.bounds,
    scale: data.scale,
    numPoints: points.length / 2,
    getPoint,
    dequantized
  };
}

/**
 * Example WebGL renderer for spiral data
 * @param {HTMLCanvasElement} canvas - Target canvas element
 * @param {object} spiral - Decoded spiral data from loadSpiral()
 */
export function renderSpiral(canvas, spiral) {
  const gl = canvas.getContext('webgl2');
  if (!gl) {
    throw new Error('WebGL2 not supported');
  }

  // Create vertex shader
  const vertexShader = gl.createShader(gl.VERTEX_SHADER);
  gl.shaderSource(vertexShader, `#version 300 es
    in vec2 position;
    uniform vec4 bounds;  // minX, maxX, minY, maxY
    uniform vec2 scale;
    
    void main() {
      // Dequantize point
      float x = bounds.x + (float(position.x) * scale.x);
      float y = bounds.z + (float(position.y) * scale.y);
      
      // Map to clip space (-1 to 1)
      x = (x - bounds.x) / (bounds.y - bounds.x) * 2.0 - 1.0;
      y = (y - bounds.z) / (bounds.w - bounds.z) * 2.0 - 1.0;
      
      gl_Position = vec4(x, y, 0.0, 1.0);
      gl_PointSize = 1.0;
    }
  `);
  gl.compileShader(vertexShader);

  // Create fragment shader
  const fragmentShader = gl.createShader(gl.FRAGMENT_SHADER);
  gl.shaderSource(fragmentShader, `#version 300 es
    precision highp float;
    out vec4 fragColor;
    
    void main() {
      fragColor = vec4(1.0, 1.0, 1.0, 0.5);
    }
  `);
  gl.compileShader(fragmentShader);

  // Create program
  const program = gl.createProgram();
  gl.attachShader(program, vertexShader);
  gl.attachShader(program, fragmentShader);
  gl.linkProgram(program);
  gl.useProgram(program);

  // Create buffer and upload points
  const buffer = gl.createBuffer();
  gl.bindBuffer(gl.ARRAY_BUFFER, buffer);
  gl.bufferData(gl.ARRAY_BUFFER, spiral.points, gl.STATIC_DRAW);

  // Set up vertex attributes
  const positionLoc = gl.getAttribLocation(program, 'position');
  gl.enableVertexAttribArray(positionLoc);
  gl.vertexAttribIPointer(positionLoc, 2, gl.SHORT, 0, 0);

  // Set uniforms
  const boundsLoc = gl.getUniformLocation(program, 'bounds');
  gl.uniform4f(boundsLoc, 
    spiral.bounds.minX, spiral.bounds.maxX,
    spiral.bounds.minY, spiral.bounds.maxY
  );
  
  const scaleLoc = gl.getUniformLocation(program, 'scale');
  gl.uniform2f(scaleLoc, spiral.scale.x, spiral.scale.y);

  // Clear and draw
  gl.clearColor(0.1, 0.1, 0.1, 1.0);
  gl.clear(gl.COLOR_BUFFER_BIT);
  gl.drawArrays(gl.POINTS, 0, spiral.numPoints);
}

// Example usage:
/*
async function init() {
  const canvas = document.querySelector('#spiral-canvas');
  const spiral = await loadSpiral('spiral_full.msgpack');
  renderSpiral(canvas, spiral);
}
*/ 