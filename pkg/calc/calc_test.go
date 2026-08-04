package calc

import (
	"math"
	"testing"
)

func TestPermute(t *testing.T) {
	tests := []struct {
		name     string
		counter  int64
		maxRows  int64
		expected int64
	}{
		{
			name:     "zero counter",
			counter:  0,
			maxRows:  100,
			expected: 0,
		},
		{
			name:     "standard positive counter",
			counter:  1,
			maxRows:  10,
			expected: (1 * LargePrime) % 10,
		},
		{
			name:     "counter equal to maxRows",
			counter:  100,
			maxRows:  100,
			expected: 0,
		},
		{
			name:     "single row maxRows",
			counter:  42,
			maxRows:  1,
			expected: 0,
		},
		{
			name:     "negative counter handling",
			counter:  -5,
			maxRows:  10,
			expected: Permute(-5, 10), // Evaluates to 5 using two's complement unsigned math
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Permute(tt.counter, tt.maxRows)
			if got != tt.expected {
				t.Errorf("Permute(%d, %d) = %d; want %d", tt.counter, tt.maxRows, got, tt.expected)
			}
		})
	}
}

// TestPermute_Properties verifies that Permute stays strictly within valid bounds [0, maxRows)
func TestPermute_Properties(t *testing.T) {
	maxRows := int64(100)

	t.Run("always within bounds [0, maxRows)", func(t *testing.T) {
		for i := int64(-100); i <= 100; i++ {
			got := Permute(i, maxRows)
			if got < 0 || got >= maxRows {
				t.Errorf("Permute(%d, %d) = %d; out of bounds [0, %d)", i, maxRows, got, maxRows-1)
			}
		}
	})
}

func TestTotalCombinations(t *testing.T) {
	tests := []struct {
		name     string
		total    float64
		need     float64
		expected int64
	}{
		{
			name:     "standard power 2^3",
			total:    2.0,
			need:     3.0,
			expected: 8,
		},
		{
			name:     "power of zero 10^0",
			total:    10.0,
			need:     0.0,
			expected: 1,
		},
		{
			name:     "zero base 0^5",
			total:    0.0,
			need:     5.0,
			expected: 0,
		},
		{
			name:     "fractional floating result cast to int64",
			total:    3.5,
			need:     2.0,
			expected: int64(math.Pow(3.5, 2.0)), // 12.25 -> 12
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TotalCombinations(tt.total, tt.need)
			if got != tt.expected {
				t.Errorf("TotalCombinations(%f, %f) = %d; want %d", tt.total, tt.need, got, tt.expected)
			}
		})
	}
}
