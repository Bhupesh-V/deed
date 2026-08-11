package fake

import (
	"deed/pkg/calc"
	"fmt"
	"math/big"
	"math/bits"
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

func (f *Fake) LetterN(key string, n uint) (string, error) {
	if n == 0 {
		n = 1
	}

	val, ok := f.c.Load(key)
	if !ok {
		val, _ = f.c.LoadOrStore(key, new(atomic.Int64))
	}

	counterInt := val.(*atomic.Int64).Add(1) - 1
	return f.generateLetterN(counterInt, n)
}

func (f *Fake) SeqIdToLetterN(seqIdx *big.Int, n uint) (string, error) {
	if n == 0 {
		n = 1
	}
	if seqIdx.Sign() < 0 {
		return "", fmt.Errorf("index %s cannot be negative", seqIdx.String())
	}

	return f.generateLetterN(seqIdx.Int64(), n)
}

func (f *Fake) generateLetterN(idx int64, n uint) (string, error) {
	// Don't calculate base-52 math for more than 6 characters to prevent 64-bit integer overflow.
	// 52^6 (~19.7 billion) gives us more than enough unique patterns for large datasets.
	effN := min(n, 6)

	// Calculate total unique 52-letter combinations possible for this length (52^effN).
	total := uint64(len(alphabet)) // 52
	var maxRows uint64 = 1
	for range effN {
		maxRows *= total
	}

	// Safely wrap around if the requested row index exceeds the total unique combinations.
	// TODO: revisit in future
	wrappedIdx := uint64(idx) % maxRows

	A := maxRows / 3

	// Ensures A is odd.
	// If A were even, it would share a factor of 2 with 52 and only visit half of the letters (even-numbered indices).
	if A%2 == 0 {
		A++
	}
	// Ensures A is not a multiple of 13.
	// If A were divisible by 13, it would share a factor of 13 with 52. Adding 2 moves A off the multiple of 13 while keeping it odd (so it stays not divisible by 2).
	if A%13 == 0 {
		A += 2
	}
	C := (maxRows / 7) | 1

	hi, lo := bits.Mul64(wrappedIdx, A)
	lo, carry := bits.Add64(lo, C, 0)
	hi += carry

	randomIndex := calc.Mod128(hi, lo, maxRows)

	out := make([]byte, n)
	temp := randomIndex

	// Decode the scrambled number into base-52 characters for the first 1–6 positions.
	for i := int(effN) - 1; i >= 0; i-- {
		out[i] = alphabet[temp%total]
		temp /= total
	}

	// For strings longer than 6 characters, quickly fill the rest using a simple position offset.
	for i := int(effN); i < int(n); i++ {
		out[i] = alphabet[(uint64(idx)+uint64(i))%total]
	}

	return string(out), nil
}
