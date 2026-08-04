package fake

import (
	"deed/pkg/calc"
	"fmt"
)

const upperStr = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
const lowerStr = "abcdefghijklmnopqrstuvwxyz"
const alphabet = upperStr + lowerStr // Length is 52

// LetterN returns Unique n character across runs
func LetterN(counter int64, n uint) (string, error) {
	if n == 0 {
		n = 1
	}

	total := len(alphabet)
	// Calculate total possible permutations for this length (52^n)
	maxRows := calc.TotalCombinations(float64(total), float64(n))

	// Safely wrap counter if it exceeds the maximum possible combinations
	if counter >= maxRows {
		return "", fmt.Errorf("Counter %d out of bounds! Maximum permutations for length %d is %d", counter, n, maxRows)
	}

	// Scramble the sequential counter into a pseudo-random index
	randomIndex := calc.Permute(counter, maxRows)

	// Decode the random index into our base-52 character string
	out := make([]byte, n)
	tempIndex := randomIndex
	for i := int(n) - 1; i >= 0; i-- {
		out[i] = alphabet[tempIndex%52]
		tempIndex /= 52
	}

	return string(out), nil
}
