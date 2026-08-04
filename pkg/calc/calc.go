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

// GetNumericDatasetSize determines total unique steps available
func GetNumericDatasetSize(precision, scale int64) int64 {
	p := float64(precision)
	s := float64(scale)

	// 1. Calculate absolute physical maximum allowed by precision/scale
	// Max = 10^(P-S) - 10^(-S)
	absMax := math.Pow(10, p-s) - math.Pow(10, -s)
	absMin := -absMax

	// 2. Adjust bounds based on CHECK constraints
	minBound := absMin
	maxBound := absMax

	// if meta.HasMinCheck {
	// 	// If constraint is strictly positive (> 0), the actual minimum
	// 	// representable number is one fractional step above the check value.
	// 	smallestStep := math.Pow(10, -s)
	// 	if meta.MinCheckVal == 0 {
	// 		minBound = smallestStep // e.g., 0.0001 if scale is 4
	// 	} else {
	// 		minBound = meta.MinCheckVal + smallestStep
	// 	}
	// }

	// 3. Apply the universal formula: ((Max - Min) * 10^Scale) + 1
	multiplier := math.Pow(10, s)
	size := ((maxBound - minBound) * multiplier) + 1

	// Round to nearest integer to clear out tiny floating-point math inaccuracies
	return int64(math.Round(size))
}
