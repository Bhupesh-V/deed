package fake

import (
	"math/big"
	"testing"
)

func TestLetterN(t *testing.T) {
	tests := []struct {
		name        string
		key         string
		n           uint
		wantLen     int
		wantErr     bool
		expectedStr string // non-empty if checking exact output
	}{
		{
			name:    "default length n=0 becomes n=1",
			key:     "test_n0",
			n:       0,
			wantLen: 1,
			wantErr: false,
		},
		// {
		// 	name:        "first call for n=1 returns first alphabet char 'A'",
		// 	key:         "test_first_char",
		// 	n:           1,
		// 	wantLen:     1,
		// 	wantErr:     false,
		// 	expectedStr: "A",
		// },
		{
			name:    "valid generation for n=2",
			key:     "test_n2",
			n:       2,
			wantLen: 2,
			wantErr: false,
		},
		{
			name:    "valid generation for n=5",
			key:     "test_n5",
			n:       5,
			wantLen: 5,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := New()
			got, err := f.LetterN(tt.key, tt.n)

			if (err != nil) != tt.wantErr {
				t.Fatalf("LetterN(%q, %d) error = %v, wantErr %v", tt.key, tt.n, err, tt.wantErr)
			}

			if !tt.wantErr {
				if len(got) != tt.wantLen {
					t.Errorf("LetterN(%q, %d) returned string of length %d; want %d", tt.key, tt.n, len(got), tt.wantLen)
				}

				if tt.expectedStr != "" && got != tt.expectedStr {
					t.Errorf("LetterN(%q, %d) = %q; want %q", tt.key, tt.n, got, tt.expectedStr)
				}
			}
		})
	}
}

func TestLetterN_WrapAround(t *testing.T) {
	f := New()
	key := "test_key_wraparound"
	n := uint(1) // maxRows = 52

	firstVal, err := f.LetterN(key, n)
	if err != nil {
		t.Fatalf("unexpected error on first call: %v", err)
	}

	// Consume remaining 51 combinations
	for i := 0; i < 51; i++ {
		if _, err := f.LetterN(key, n); err != nil {
			t.Fatalf("unexpected error at call %d: %v", i+2, err)
		}
	}

	// 53rd call: Wraps around modulo 52, returning firstVal with no error
	wrapVal, err := f.LetterN(key, n)
	if err != nil {
		t.Fatalf("expected graceful wrap-around on 53rd call, got error: %v", err)
	}

	if wrapVal != firstVal {
		t.Errorf("expected wrap-around value %q, got %q", firstVal, wrapVal)
	}
}

// TestLetterN_Uniqueness verifies that sequential calls on the same key produce non-repeating strings
func TestLetterN_Uniqueness(t *testing.T) {
	f := New()
	key := "unique_sequence"
	n := uint(2)
	maxRows := int64(52 * 52) // 2704 combinations
	seen := make(map[string]int64, maxRows)

	for counter := int64(0); counter < maxRows; counter++ {
		str, err := f.LetterN(key, n)
		if err != nil {
			t.Fatalf("LetterN(%q, %d) unexpected error at step %d: %v", key, n, counter, err)
		}

		if prevCounter, exists := seen[str]; exists {
			t.Fatalf("Collision detected! Duplicate string %q generated at call %d (previously seen at %d)", str, counter, prevCounter)
		}

		seen[str] = counter
	}

	if int64(len(seen)) != maxRows {
		t.Errorf("Expected %d unique strings, generated %d", maxRows, len(seen))
	}
}

// TestLetterN_AlphabetCharactersOnly ensures generated output only contains base-52 characters
func TestLetterN_AlphabetCharactersOnly(t *testing.T) {
	f := New()
	key := "alpha_check"

	validChars := make(map[rune]bool)
	for _, ch := range alphabet {
		validChars[ch] = true
	}

	for counter := 0; counter < 100; counter++ {
		str, err := f.LetterN(key, 3)
		if err != nil {
			t.Fatalf("Unexpected error at call %d: %v", counter, err)
		}

		for _, char := range str {
			if !validChars[char] {
				t.Errorf("LetterN generated invalid character %q in %q at call %d", char, str, counter)
			}
		}
	}
}

// TestLetterN_KeyIsolation verifies that different keys maintain independent counters
func TestLetterN_KeyIsolation(t *testing.T) {
	f := New()

	// Assert key isolation by comparing first calls across keys
	valA, _ := f.LetterN("key_a", 1)
	valB, _ := f.LetterN("key_b", 1)

	if valA != valB {
		t.Errorf("Expected identical initial state output for isolated keys, got %q and %q", valA, valB)
	}
}

// TestSeqIdxToLetterN_Determinism verifies that the same ID and length
// always produce the exact same string output.
func TestSeqIdxToLetterN_Determinism(t *testing.T) {
	f := New()
	idx := big.NewInt(12345)
	n := uint(8)

	str1, err1 := f.SeqIdToLetterN(idx, n)
	str2, err2 := f.SeqIdToLetterN(idx, n)

	if err1 != nil || err2 != nil {
		t.Fatalf("unexpected error: %v, %v", err1, err2)
	}

	if str1 != str2 {
		t.Errorf("non-deterministic output: got '%s' and '%s' for same index %s", str1, str2, idx)
	}
}

// TestSeqIdxToLetterN_ZeroLengthHandling checks that n = 0 defaults to length 1.
func TestSeqIdxToLetterN_ZeroLengthHandling(t *testing.T) {
	f := New()
	idx := big.NewInt(5)

	str, err := f.SeqIdToLetterN(idx, 0)
	if err != nil {
		t.Fatalf("unexpected error when n=0: %v", err)
	}

	if len(str) != 1 {
		t.Errorf("expected string length 1 when n=0, got length %d (str: %s)", len(str), str)
	}
}
