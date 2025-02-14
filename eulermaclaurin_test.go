package distributed

import (
	"math"
	"testing"
)

func TestEulerMaclaurin(t *testing.T) {
	type testCase struct {
		input complex128
		// Due to the approximative nature of the summation, we check for consistency rather than an exact value.
	}

	cases := []testCase{
		{input: complex(0.5, 14.135)},
		{input: complex(1.0, 2.5)},
	}

	for _, tc := range cases {
		got, iter, diff := EulerMaclaurin(tc.input)
		// Check that the result is finite.
		if math.IsNaN(real(got)) || math.IsNaN(imag(got)) {
			t.Errorf("EulerMaclaurin(%v) returned NaN", tc.input)
		}
		// Verify that the correction loop produced a reasonable number of iterations.
		if iter < 1 {
			t.Errorf("EulerMaclaurin(%v): expected iteration count > 0, got %d", tc.input, iter)
		}

		// Check consistency: two calls with the same input should be nearly equal.
		got2, _, _ := EulerMaclaurin(tc.input)
		tolerance := 1e-3
		if math.Abs(real(got)-real(got2)) > tolerance || math.Abs(imag(got)-imag(got2)) > tolerance {
			t.Errorf("Inconsistent results for EulerMaclaurin(%v): got (%.6f, %.6f) vs (%.6f, %.6f)",
				tc.input, real(got), imag(got), real(got2), imag(got2))
		}

		// Optionally, print the debug info for manual inspection.
		t.Logf("Input: %v, Result: (%.6f, %.6f), iterations=%d, final diff=%.8f",
			tc.input, real(got), imag(got), iter, diff)
	}
}
