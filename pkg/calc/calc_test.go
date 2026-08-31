package calc

import (
	"math"
	"math/bits"
	"math/rand"
	"testing"
)

func TestMod128(t *testing.T) {
	tests := []struct {
		name string
		hi   uint64
		lo   uint64
		m    uint64
		want uint64
	}{
		{
			name: "zero modulo",
			hi:   10,
			lo:   20,
			m:    0,
			want: 0,
		},
		{
			name: "modulo 1",
			hi:   10,
			lo:   20,
			m:    1,
			want: 0,
		},
		{
			name: "hi is zero (standard uint64 modulo)",
			hi:   0,
			lo:   42,
			m:    10,
			want: 2,
		},
		{
			name: "hi >= m (would overflow bits.Div64)",
			hi:   100,
			lo:   50,
			m:    7,
			want: 5, // (100 * 2^64 + 50) % 7 = 5
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Mod128(tt.hi, tt.lo, tt.m)
			if got != tt.want {
				t.Errorf("Mod128(%d, %d, %d) = %d; want %d", tt.hi, tt.lo, tt.m, got, tt.want)
			}
		})
	}
}

func TestPermute(t *testing.T) {
	// Helper to compute expected output at runtime without constant uint64 overflow
	calcExpected := func(counter, maxRows int64) int64 {
		if maxRows <= 0 {
			return 0
		}
		uCounter := uint64(counter)
		uMax := uint64(maxRows)
		hi, lo := bits.Mul64(uCounter, LargePrime)
		return int64(Mod128(hi, lo, uMax))
	}

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
			expected: calcExpected(1, 10),
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
			expected: calcExpected(-5, 10),
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

func TestGetNumericDatasetSize(t *testing.T) {
	tests := []struct {
		name      string
		precision int64
		scale     int64
		radix     int64
		want      int64
	}{
		{
			name:      "Radix 2: 8-bit Tinyint space",
			precision: 8,
			scale:     0,
			radix:     2,
			want:      256,
		},
		{
			name:      "Radix 2: 16-bit Smallint space",
			precision: 16,
			scale:     0,
			radix:     2,
			want:      65536,
		},
		{
			name:      "Radix 2: 32-bit Integer space",
			precision: 32,
			scale:     0,
			radix:     2,
			want:      4294967296,
		},
		{
			name:      "Radix 2: 64-bit Bigint overflow triggers MaxInt64 guard",
			precision: 64,
			scale:     0,
			radix:     2,
			want:      math.MaxInt64,
		},
		{
			name:      "Radix 10: Precision 3, Scale 0 (Range: -999 to 999)",
			precision: 3,
			scale:     0,
			radix:     10,
			want:      1999,
		},
		{
			name:      "Radix 10: Precision 4, Scale 2 (Range: -99.99 to 99.99 step 0.01)",
			precision: 4,
			scale:     2,
			radix:     10,
			want:      19999,
		},
		{
			name:      "Radix 10: Precision 5, Scale 0 (Range: -99999 to 99999)",
			precision: 5,
			scale:     0,
			radix:     10,
			want:      199999,
		},
		{
			name:      "Radix 10: Precision 6, Scale 3 (Range: -999.999 to 999.999)",
			precision: 6,
			scale:     3,
			radix:     10,
			want:      1999999,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetNumericDatasetSize(tt.precision, tt.scale, tt.radix)
			if got != tt.want {
				t.Errorf("GetNumericDatasetSize(%d, %d, %d) = %d; want %d",
					tt.precision, tt.scale, tt.radix, got, tt.want)
			}
		})
	}
}

func TestHashCounter_EdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		counter int64
		min     int64
		max     int64
		want    int64
	}{
		{
			name:    "Invalid Range (max < min)",
			counter: 5,
			min:     10,
			max:     5,
			want:    10,
		},
		{
			name:    "Single Element Range (min == max)",
			counter: 0,
			min:     42,
			max:     42,
			want:    42,
		},
		{
			name:    "Single Element Range with Non-Zero Counter",
			counter: 100,
			min:     42,
			max:     42,
			want:    42,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HashCounter(tt.counter, tt.min, tt.max)
			if got != tt.want {
				t.Errorf("HashCounter(%d, %d, %d) = %d; want %d",
					tt.counter, tt.min, tt.max, got, tt.want)
			}
		})
	}
}

func TestHashCounter_BoundsAndDeterminism(t *testing.T) {
	min, max := int64(-50), int64(50)

	for i := 0; i < 100; i++ {
		counter := rand.Int63n(1000) - 500
		got := HashCounter(counter, min, max)

		if got < min || got > max {
			t.Errorf("HashCounter(%d, %d, %d) = %d; out of bounds [%d, %d]",
				counter, min, max, got, min, max)
		}

		gotAgain := HashCounter(counter, min, max)
		if got != gotAgain {
			t.Errorf("HashCounter is non-deterministic: first call got %d, second call got %d",
				got, gotAgain)
		}
	}
}

func TestHashCounter_FullRangeUniqueness(t *testing.T) {
	ranges := []struct {
		min int64
		max int64
	}{
		{min: 0, max: 9},
		{min: 1, max: 100},
		{min: -20, max: 20},
		{min: 1000, max: 1050},
	}

	for _, r := range ranges {
		totalElements := int(r.max - r.min + 1)
		seen := make(map[int64]bool, totalElements)

		for counter := r.min; counter <= r.max; counter++ {
			val := HashCounter(counter, r.min, r.max)

			if val < r.min || val > r.max {
				t.Fatalf("Value %d out of range [%d, %d] for counter %d", val, r.min, r.max, counter)
			}

			if seen[val] {
				t.Fatalf("Duplicate value %d generated for range [%d, %d] at counter %d",
					val, r.min, r.max, counter)
			}
			seen[val] = true
		}

		if len(seen) != totalElements {
			t.Errorf("Expected %d unique elements for range [%d, %d], got %d",
				totalElements, r.min, r.max, len(seen))
		}
	}
}

// ----------------------------------------------------------------------------
// Single-Threaded Benchmarks
// ----------------------------------------------------------------------------

func BenchmarkDeterministicRatioPool(b *testing.B) {
	colKey := "users:email"
	b.ReportAllocs()

	for i := 0; b.Loop(); i++ {
		_ = DeterministicRatio(int64(i), colKey)
	}
}

// ----------------------------------------------------------------------------
// Parallel Load Benchmarks (Simulating Worker Pool)
// ----------------------------------------------------------------------------

func BenchmarkDeterministicRatioPool_Parallel(b *testing.B) {
	colKey := "users:email"
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		var i int64
		for pb.Next() {
			i++
			_ = DeterministicRatio(i, colKey)
		}
	})
}
