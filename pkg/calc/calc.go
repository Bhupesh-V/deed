package calc

import (
	"math"
	"math/big"
)

// A massive 64-bit prime number (2 Quintillion - god's chillar)
// It must be larger than our maximum expected row count. Ain't nobody requesting 2 Quintillion rows
const LargePrime int64 = 2305843009213693951

// Permute maps a sequential counter to a pseudo-random, non-repeating index.
func Permute(counter int64, maxRows int64) int64 {
	if maxRows <= 0 {
		return 0
	}

	c := big.NewInt(counter)
	p := big.NewInt(LargePrime)
	m := big.NewInt(maxRows)

	// (counter * LargePrime) % maxRows
	prod := new(big.Int).Mul(c, p)
	rem := new(big.Int).Mod(prod, m)

	return rem.Int64()
}

func TotalCombinations(total, need float64) int64 {
	return int64(math.Pow(total, need))
}

// HashCounter maps a sequential counter within [min, max] to a pseudo-random,
// non-repeating index within the same [min, max] range.
func HashCounter(counter, min, max int64) int64 {
	if max < min {
		return min
	}

	// Calculate total number of elements in the range [min, max]
	maxRows := (max - min) + 1
	if maxRows <= 0 {
		return min
	}

	// Normalize counter to a 0-indexed scale [0, maxRows - 1]
	normalizedCounter := counter - min

	rem := Permute(normalizedCounter, maxRows)

	return rem + min
}

// GetNumericDatasetSize determines total unique steps available by utilizing the explicit number base system (radix).
func GetNumericDatasetSize(precision, scale, radix int64) int64 {
	// 1. If Radix is 2, it is a binary system (PostgreSQL Integer types like smallint, int, bigint)
	if radix == 2 {
		// Calculate the full binary range space: 2^precision
		size := math.Pow(2, float64(precision))

		// Prevent Int64 overflow for bigint columns (precision 64)
		if size > float64(math.MaxInt64) || math.IsInf(size, 0) {
			return math.MaxInt64
		}
		return int64(size)
	}

	// 2. If Radix is 10, it is a decimal system (PostgreSQL NUMERIC / DECIMAL types)
	p := float64(precision)
	s := float64(scale)

	// Calculate absolute physical maximum allowed by precision/scale
	absMax := math.Pow(10, p-s) - math.Pow(10, -s)
	absMin := -absMax

	minBound := absMin
	maxBound := absMax

	// Apply the universal step formula: ((Max - Min) * 10^Scale) + 1
	multiplier := math.Pow(10, s)
	size := ((maxBound - minBound) * multiplier) + 1

	// Round to nearest integer to clear out floating-point inaccuracies
	return int64(math.Round(size))
}

var (
	big2  = big.NewInt(2)
	big13 = big.NewInt(13)
)

// TotalCombinationsBig calculates base^exp (e.g., 52^N)
func TotalCombinationsBig(base, exp int64) *big.Int {
	b := big.NewInt(base)
	e := big.NewInt(exp)
	return new(big.Int).Exp(b, e, nil)
}

// PermuteBig creates a strict 1-to-1 permutation across [0, maxRows-1].
// Mathematically guaranteed to have ZERO collisions/duplicates.
func PermuteBig(counter, maxRows *big.Int) *big.Int {
	if maxRows.Sign() <= 0 {
		return big.NewInt(0)
	}

	// Generate multiplier A roughly equal to (maxRows / 3)
	A := new(big.Int).Div(maxRows, big.NewInt(3))

	// Ensure A is coprime to 52^N (must NOT be divisible by 2 or 13)
	if new(big.Int).Mod(A, big2).Sign() == 0 {
		A.Add(A, big.NewInt(1)) // Make odd
	}
	if new(big.Int).Mod(A, big13).Sign() == 0 {
		A.Add(A, big2) // Shift away from multiple of 13
	}

	// Add an offset C so counter=0 doesn't start at "AAAA..."
	C := new(big.Int).Div(maxRows, big.NewInt(7))
	if new(big.Int).Mod(C, big2).Sign() == 0 {
		C.Add(C, big.NewInt(1))
	}

	// Compute (counter * A + C) % maxRows
	result := new(big.Int).Mul(counter, A)
	result.Add(result, C)
	return result.Mod(result, maxRows)
}
