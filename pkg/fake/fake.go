package fake

import (
	"deed/pkg/calc"
	"fmt"
	"sync"
	"sync/atomic"
)

const upperStr = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
const lowerStr = "abcdefghijklmnopqrstuvwxyz"
const alphabet = upperStr + lowerStr // Length is 52

type Fake struct {
	c sync.Map
}

func New() *Fake {
	return &Fake{}
}

// LetterN returns Unique n character across runs
func (f *Fake) LetterN(key string, n uint) (string, error) {
	if n == 0 {
		n = 1
	}

	// Fast path: check if key already exists (0 heap allocations)
	val, ok := f.c.Load(key)
	if !ok {
		// Slow path: allocate only when key is missing for the first time
		val, _ = f.c.LoadOrStore(key, new(atomic.Int64))
	}

	counter := val.(*atomic.Int64).Add(1) - 1
	// fmt.Printf("counter: %d\n", counter)

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
