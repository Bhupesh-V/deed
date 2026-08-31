package calc

import (
	"hash/fnv"
	"math"
	"math/bits"
	"sync"
)

// A massive 64-bit prime number (2 Quintillion - god's chillar)
// It must be larger than our maximum expected row count. Ain't nobody requesting 2 Quintillion rows
const LargePrime uint64 = 2305843009213693951

// Mod128 computes (hi*2^64 + lo) % m without bits.Div64 overflow panics
func Mod128(hi, lo, m uint64) uint64 {
	if m <= 1 {
		return 0
	}
	if hi == 0 {
		return lo % m
	}

	R := (math.MaxUint64%m + 1) % m

	for hi > 0 {
		hi = hi % m
		hR, lR := bits.Mul64(hi, R)

		var carry uint64
		lo, carry = bits.Add64(lR, lo%m, 0)
		hi = hR + carry
	}

	return lo % m
}

// Permute maps counter to non-repeating index with 0 heap allocations
func Permute(counter int64, maxRows int64) int64 {
	if maxRows <= 0 {
		return 0
	}

	uCounter := uint64(counter)
	uMax := uint64(maxRows)

	hi, lo := bits.Mul64(uCounter, LargePrime)
	rem := Mod128(hi, lo, uMax)

	return int64(rem)
}

func TotalCombinations(total, need float64) int64 {
	return int64(math.Pow(total, need))
}

func HashCounter(counter, min, max int64) int64 {
	if max < min {
		return min
	}

	maxRows := (max - min) + 1
	if maxRows <= 0 {
		return min
	}

	normalizedCounter := counter - min
	rem := Permute(normalizedCounter, maxRows)

	return rem + min
}

func GetNumericDatasetSize(precision, scale, radix int64) int64 {
	if radix == 2 {
		size := math.Pow(2, float64(precision))
		if size > float64(math.MaxInt64) || math.IsInf(size, 0) {
			return math.MaxInt64
		}
		return int64(size)
	}

	p := float64(precision)
	s := float64(scale)
	absMax := math.Pow(10, p-s) - math.Pow(10, -s)
	absMin := -absMax

	multiplier := math.Pow(10, s)
	size := ((absMax - absMin) * multiplier) + 1

	return int64(math.Round(size))
}

// func TotalCombinationsBig(base, exp int64) *big.Int {
// 	b := big.NewInt(base)
// 	e := big.NewInt(exp)
// 	return new(big.Int).Exp(b, e, nil)
// }

// func PermuteBig(counter, maxRows *big.Int) *big.Int {
// 	if maxRows.Sign() <= 0 {
// 		return big.NewInt(0)
// 	}

// 	A := new(big.Int).Div(maxRows, big.NewInt(3))
// 	big2 := big.NewInt(2)
// 	big13 := big.NewInt(13)

// 	if new(big.Int).Mod(A, big2).Sign() == 0 {
// 		A.Add(A, big.NewInt(1))
// 	}
// 	if new(big.Int).Mod(A, big13).Sign() == 0 {
// 		A.Add(A, big2)
// 	}

// 	C := new(big.Int).Div(maxRows, big.NewInt(7))
// 	if new(big.Int).Mod(C, big2).Sign() == 0 {
// 		C.Add(C, big.NewInt(1))
// 	}

// 	result := new(big.Int).Mul(counter, A)
// 	result.Add(result, C)
// 	return result.Mod(result, maxRows)
// }

var fnvPool = sync.Pool{
	New: func() any {
		return fnv.New64a()
	},
}

// DeterministicRatio calculates a deterministic float32 ratio in [0.0, 1.0) for a given row and column.
// It uses an inlined 64-bit FNV-1a hash algorithm mapped down to 24-bit floating-point precision.
func DeterministicRatio(rowIndex int64, colKey string) float32 {
	// 14695981039346656037: Official 64-bit FNV-1a offset basis (initial hash seed state).
	var h uint64 = 14695981039346656037

	// Hash the bytes of the column identifier (e.g., "users:email").
	for i := 0; i < len(colKey); i++ {
		h ^= uint64(colKey[i])

		// 1099511628211: Official 64-bit FNV prime multiplier.
		// Multiplying by this prime scatters bits unpredictably (avalanche effect).
		h *= 1099511628211
	}

	// Mix the 64-bit row index into the hash state.
	h ^= uint64(rowIndex)

	// Multiply by the FNV prime a second time to fully scramble the row index bits.
	h *= 1099511628211

	// 0xFFFFFF (Hex) = 16,777,215 (Decimal) = 24 ones in binary (111111111111111111111111).
	// Performs a bitwise AND (&) to keep only the lower 24 bits of the 64-bit hash.
	// 24 bits is chosen because float32 has exactly 24 bits of significand/precision.
	lower24Bits := h & 0xFFFFFF

	// 1 << 24 = 2^24 = 16,777,216.
	// Dividing the 24-bit integer by 16,777,216 normalizes the value into a float32 in [0.0, 1.0).
	return float32(lower24Bits) / float32(1<<24)
}
