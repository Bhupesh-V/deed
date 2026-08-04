package fake

import (
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
		{
			name:        "first call for n=1 returns first alphabet char 'A'",
			key:         "test_first_char",
			n:           1,
			wantLen:     1,
			wantErr:     false,
			expectedStr: "A",
		},
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

// TestLetterN_OutOfBounds verifies that once a key exhausts max permutations, an error is returned
func TestLetterN_OutOfBounds(t *testing.T) {
	f := New()
	key := "exhaust_key"
	n := uint(1) // maxRows is 52

	// Consume all 52 valid permutations (counters 0 to 51)
	for i := 0; i < 52; i++ {
		_, err := f.LetterN(key, n)
		if err != nil {
			t.Fatalf("Unexpected error on call %d: %v", i, err)
		}
	}

	// The 53rd call (counter 52) must trigger out of bounds error
	_, err := f.LetterN(key, n)
	if err == nil {
		t.Fatalf("Expected out of bounds error on 53rd call for n=1, but got nil")
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

	strA1, _ := f.LetterN("key_a", 1)
	strB1, _ := f.LetterN("key_b", 1)
	strA2, _ := f.LetterN("key_a", 1)

	if strA1 != "A" {
		t.Errorf("Expected first call for key_a to be 'A', got %q", strA1)
	}

	// key_b should start at counter 0 ('A'), isolated from key_a
	if strB1 != "A" {
		t.Errorf("Expected first call for key_b to be 'A', got %q", strB1)
	}

	// key_a's second call should advance independently
	if strA2 == strA1 {
		t.Errorf("Expected key_a to advance counter on second call, got duplicate %q", strA2)
	}
}
