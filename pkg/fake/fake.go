package fake

import (
	"deed/internal/models"
	"deed/pkg/calc"
	"fmt"
	"math"
	"math/big"
	"math/bits"
	"math/rand"
	"regexp/syntax"
	"strings"
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

// Float generates a deterministic float64 based on row index, precision, and scale.
// Example: precision=5, scale=2 produces numbers up to 999.99 (3 int digits, 2 frac digits).
func (f *Fake) Float(idx int64, precision, scale int) float64 {
	if precision <= 0 {
		precision = 10
	}
	if scale < 0 {
		scale = 2
	}

	if precision > 15 {
		precision = 15
	}
	if scale > precision {
		scale = precision
	}

	integerDigits := precision - scale

	// Max value for the integer component as int64 (e.g., 3 digits -> 999)
	maxIntPart := int64(math.Pow10(integerDigits)) - 1
	if maxIntPart <= 0 {
		maxIntPart = 1
	}

	// Generate integer component
	intPart := calc.Permute(idx, maxIntPart)

	// Generate fractional component using offset (e.g., +67)
	var fracPart int64
	scaleFactor := math.Pow10(scale)
	if scale > 0 {
		maxFracPart := int64(scaleFactor)
		// hehe ⁶🤷‍♂️⁷
		fracPart = calc.Permute(idx+int64(models.SixSeven), maxFracPart)
	}

	// Assemble and round to requested scale
	rawVal := float64(intPart) + (float64(fracPart) / scaleFactor)
	return math.Round(rawVal*scaleFactor) / scaleFactor
}

// Regex generates a string matching pattern for row idx, seeded purely by
// idx so it's deterministic, but with no uniqueness guarantee — repeated
// idx values will produce the same output. Use this for non-UNIQUE columns;
func (f *Fake) Regex(idx int64, pattern string) (string, error) {
	re, err := syntax.Parse(pattern, syntax.Perl)
	if err != nil {
		return "", err
	}

	rng := rand.New(rand.NewSource(idx))

	var sb strings.Builder
	f.buildRegexString(&sb, re, rng)
	return sb.String(), nil
}

func (f *Fake) buildRegexString(sb *strings.Builder, re *syntax.Regexp, rng *rand.Rand) {
	switch re.Op {
	case syntax.OpLiteral:
		sb.WriteString(string(re.Rune))

	case syntax.OpConcat:
		for _, sub := range re.Sub {
			f.buildRegexString(sb, sub, rng)
		}

	case syntax.OpAlternate:
		if len(re.Sub) > 0 {
			choice := rng.Intn(len(re.Sub))
			f.buildRegexString(sb, re.Sub[choice], rng)
		}

	case syntax.OpCharClass:
		var totalRunes int
		for i := 0; i < len(re.Rune); i += 2 {
			totalRunes += int(re.Rune[i+1] - re.Rune[i] + 1)
		}
		if totalRunes > 0 {
			pick := rng.Intn(totalRunes)
			for i := 0; i < len(re.Rune); i += 2 {
				count := int(re.Rune[i+1] - re.Rune[i] + 1)
				if pick < count {
					sb.WriteRune(re.Rune[i] + rune(pick))
					break
				}
				pick -= count
			}
		}

	case syntax.OpStar:
		count := rng.Intn(maxStarRepetitions)
		for range count {
			f.buildRegexString(sb, re.Sub[0], rng)
		}

	case syntax.OpPlus:
		count := 1 + rng.Intn(maxPlusExtraCount)
		for range count {
			f.buildRegexString(sb, re.Sub[0], rng)
		}

	case syntax.OpQuest:
		if rng.Intn(2) == 1 {
			f.buildRegexString(sb, re.Sub[0], rng)
		}

	case syntax.OpRepeat:
		minR, maxR := repeatBounds(re.Min, re.Max)
		diff := maxR - minR + 1
		count := minR + rng.Intn(diff)
		for range count {
			f.buildRegexString(sb, re.Sub[0], rng)
		}

	case syntax.OpCapture:
		f.buildRegexString(sb, re.Sub[0], rng)

	case syntax.OpAnyChar, syntax.OpAnyCharNotNL:
		sb.WriteByte(defaultAnyCharAlphabet[rng.Intn(len(defaultAnyCharAlphabet))])
	}
}

// UniqueRegex generates a string matching pattern for row idx, guaranteed
// COLLISION-FREE: every idx in [0, EstimateRegexCapacity(pattern)) maps to a distinct output.
func (f *Fake) UniqueRegex(idx int64, pattern string) (string, error) {
	re, err := syntax.Parse(pattern, syntax.Perl)
	if err != nil {
		return "", err
	}

	capacity := max(regexCapacity(re), 1)
	scrambled := calc.Permute(idx, capacity)

	var sb strings.Builder
	decodeRegex(&sb, re, scrambled)
	return sb.String(), nil
}

// EstimateRegexCapacity returns the exact number of distinct strings
// UniqueRegex can produce for pattern (see regexCapacity), i.e. the count
// beyond which row indexes start wrapping around and repeating values.
// row indexes start wrapping around and repeating values.
func EstimateRegexCapacity(pattern string) (int64, error) {
	re, err := syntax.Parse(pattern, syntax.Perl)
	if err != nil {
		return 0, err
	}
	return regexCapacity(re), nil
}

// Default bounds for regex generation to keep synthetic data within practical database limits.
const (
	maxStarRepetitions     = 4 // OpStar (*): 0 to 3 repetitions
	maxPlusExtraCount      = 3 // OpPlus (+): 1 to 3 repetitions
	maxUnboundedRepeatCap  = 3 // OpRepeat ({n,}): caps unbounded repeats to min + 3
	repeatThresholdGap     = 5 // Threshold to detect unbounded or excessively large repeat ranges ({n, m})
	defaultAnyCharAlphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789 -_.,!@#"

	// capacityCeiling bounds every intermediate capacity computation well
	// under math.MaxInt64, leaving headroom so combining two already-clamped
	// values (multiply or add) can never itself overflow int64. Deliberately
	// NOT math.MaxInt64/4 or any other multiple of calc.LargePrime: when
	// capacity clamps to exactly that value, calc.Permute(idx, capacity)
	// degenerates to (idx*LargePrime) mod LargePrime, which is 0 for every
	// idx — collapsing every row to the same output instead of scrambling
	// them apart.
	capacityCeiling int64 = 1_000_000_000_000_000_000
)

func safeMul(a, b int64) int64 {
	if a == 0 || b == 0 {
		return 0
	}
	if a > capacityCeiling/b {
		return capacityCeiling
	}
	return a * b
}

func safeAdd(a, b int64) int64 {
	if a > capacityCeiling-b {
		return capacityCeiling
	}
	return a + b
}

func safePow(base int64, exp int) int64 {
	result := int64(1)
	for range exp {
		result = safeMul(result, base)
		if result >= capacityCeiling {
			return capacityCeiling
		}
	}
	return result
}

// repeatBounds applies the same {min,max} capping used everywhere below, so
// capacity and decode logic always agree on the reachable repetition range.
func repeatBounds(minR, maxR int) (int, int) {
	if maxR < 0 || maxR > minR+repeatThresholdGap {
		maxR = minR + maxUnboundedRepeatCap
	}
	return minR, maxR
}

// regexCapacity computes the exact number of distinct strings reachable from
// re, honoring the same repetition caps buildRegexString/decodeRegex use.
func regexCapacity(re *syntax.Regexp) int64 {
	switch re.Op {
	case syntax.OpLiteral:
		return 1

	case syntax.OpConcat:
		total := int64(1)
		for _, sub := range re.Sub {
			total = safeMul(total, regexCapacity(sub))
		}
		return total

	case syntax.OpAlternate:
		total := int64(0)
		for _, sub := range re.Sub {
			total = safeAdd(total, regexCapacity(sub))
		}
		return total

	case syntax.OpCharClass:
		var total int64
		for i := 0; i < len(re.Rune); i += 2 {
			total += int64(re.Rune[i+1] - re.Rune[i] + 1)
		}
		return total

	case syntax.OpStar:
		minR, maxR := repeatBounds(0, maxStarRepetitions-1)
		return repeatCapacity(re.Sub[0], minR, maxR)

	case syntax.OpPlus:
		minR, maxR := repeatBounds(1, maxPlusExtraCount)
		return repeatCapacity(re.Sub[0], minR, maxR)

	case syntax.OpQuest:
		return safeAdd(1, regexCapacity(re.Sub[0]))

	case syntax.OpRepeat:
		minR, maxR := repeatBounds(re.Min, re.Max)
		return repeatCapacity(re.Sub[0], minR, maxR)

	case syntax.OpCapture:
		return regexCapacity(re.Sub[0])

	case syntax.OpAnyChar, syntax.OpAnyCharNotNL:
		return int64(len(defaultAnyCharAlphabet))

	default:
		return 1
	}
}

// repeatCapacity sums the capacity of every reachable repetition count k in
// [minR,maxR] — each k produces a different-length (hence disjoint) output.
func repeatCapacity(sub *syntax.Regexp, minR, maxR int) int64 {
	subCap := regexCapacity(sub)
	total := int64(0)
	for k := minR; k <= maxR; k++ {
		total = safeAdd(total, safePow(subCap, k))
	}
	return total
}

// decodeRegex "unranks" index (0 <= index < regexCapacity(re)) into its
// corresponding string, mirroring regexCapacity's structure exactly so the
// two stay in lockstep.
func decodeRegex(sb *strings.Builder, re *syntax.Regexp, index int64) {
	switch re.Op {
	case syntax.OpLiteral:
		sb.WriteString(string(re.Rune))

	case syntax.OpConcat:
		decodeConcat(sb, re.Sub, index)

	case syntax.OpAlternate:
		remaining := index
		for _, sub := range re.Sub {
			c := regexCapacity(sub)
			if remaining < c {
				decodeRegex(sb, sub, remaining)
				return
			}
			remaining -= c
		}

	case syntax.OpCharClass:
		remaining := index
		for i := 0; i < len(re.Rune); i += 2 {
			count := int64(re.Rune[i+1] - re.Rune[i] + 1)
			if remaining < count {
				sb.WriteRune(re.Rune[i] + rune(remaining))
				return
			}
			remaining -= count
		}

	case syntax.OpStar:
		minR, maxR := repeatBounds(0, maxStarRepetitions-1)
		decodeRepeat(sb, re.Sub[0], index, minR, maxR)

	case syntax.OpPlus:
		minR, maxR := repeatBounds(1, maxPlusExtraCount)
		decodeRepeat(sb, re.Sub[0], index, minR, maxR)

	case syntax.OpQuest:
		if index == 0 {
			return
		}
		decodeRegex(sb, re.Sub[0], index-1)

	case syntax.OpRepeat:
		minR, maxR := repeatBounds(re.Min, re.Max)
		decodeRepeat(sb, re.Sub[0], index, minR, maxR)

	case syntax.OpCapture:
		decodeRegex(sb, re.Sub[0], index)

	case syntax.OpAnyChar, syntax.OpAnyCharNotNL:
		sb.WriteByte(defaultAnyCharAlphabet[int(index)%len(defaultAnyCharAlphabet)])
	}
}

// decodeConcat mixed-radix decodes index across subs, last sub varying
// fastest — the inverse of the product accumulation in regexCapacity.
func decodeConcat(sb *strings.Builder, subs []*syntax.Regexp, index int64) {
	digits := make([]int64, len(subs))
	remaining := index
	for i := len(subs) - 1; i >= 0; i-- {
		c := regexCapacity(subs[i])
		if c <= 0 {
			continue
		}
		digits[i] = remaining % c
		remaining /= c
	}
	for i, sub := range subs {
		decodeRegex(sb, sub, digits[i])
	}
}

// decodeRepeat picks which repetition-count bucket index falls into (each
// bucket k has capacity subCap^k), then mixed-radix decodes k copies of sub.
func decodeRepeat(sb *strings.Builder, sub *syntax.Regexp, index int64, minR, maxR int) {
	subCap := regexCapacity(sub)
	remaining := index
	for k := minR; k <= maxR; k++ {
		bucketCap := safePow(subCap, k)
		if remaining < bucketCap {
			subs := make([]*syntax.Regexp, k)
			for i := range subs {
				subs[i] = sub
			}
			decodeConcat(sb, subs, remaining)
			return
		}
		remaining -= bucketCap
	}
}
