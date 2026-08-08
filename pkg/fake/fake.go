package fake

import (
	"deed/pkg/calc"
	"fmt"
	"math/big"
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

	val, ok := f.c.Load(key)
	if !ok {
		val, _ = f.c.LoadOrStore(key, new(atomic.Int64))
	}

	counterInt := val.(*atomic.Int64).Add(1) - 1
	counter := big.NewInt(counterInt)
	total := int64(len(alphabet))

	maxRows := calc.TotalCombinationsBig(total, int64(n))

	if counter.Cmp(maxRows) >= 0 {
		return "", fmt.Errorf("Counter %d out of bounds for length %d", counterInt, n)
	}

	// Permute across the full big.Int space
	randomIndex := calc.PermuteBig(counter, maxRows)

	// Base-52 decode using big.Int Mod/Div operations
	out := make([]byte, n)
	tempIndex := new(big.Int).Set(randomIndex)
	big52 := big.NewInt(total)
	mod := new(big.Int)

	for i := int(n) - 1; i >= 0; i-- {
		// tempIndex = tempIndex / 52, mod = tempIndex % 52
		tempIndex.DivMod(tempIndex, big52, mod)
		out[i] = alphabet[mod.Int64()]
	}

	return string(out), nil
}

func (f *Fake) SeqIdToLetterN(seqIdx *big.Int, n uint) (string, error) {
	if n == 0 {
		n = 1
	}

	total := int64(len(alphabet))
	maxRows := calc.TotalCombinationsBig(total, int64(n))

	// Check bounds
	if seqIdx.Cmp(maxRows) >= 0 || seqIdx.Sign() < 0 {
		return "", fmt.Errorf("index %s out of bounds for length %d", seqIdx.String(), n)
	}

	// Permute the counter across the space
	randomIndex := calc.PermuteBig(seqIdx, maxRows)

	// Base-52 encode into alphabet characters
	out := make([]byte, n)
	tempIndex := new(big.Int).Set(randomIndex)
	big52 := big.NewInt(total)
	mod := new(big.Int)

	for i := int(n) - 1; i >= 0; i-- {
		tempIndex.DivMod(tempIndex, big52, mod)
		out[i] = alphabet[mod.Int64()]
	}

	return string(out), nil
}
