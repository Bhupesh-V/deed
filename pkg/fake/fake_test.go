package fake

import (
	"testing"
)

func TestLetterN(t *testing.T) {
	tests := []struct {
		name        string
		counter     int64
		n           uint
		wantLen     int
		wantErr     bool
		expectedStr string // non-empty if checking exact output
	}{
		{
			name:    "default length n=0 becomes n=1",
			counter: 0,
			n:       0,
			wantLen: 1,
			wantErr: false,
		},
		{
			name:        "counter 0 for n=1 returns first alphabet char 'A'",
			counter:     0,
			n:           1,
			wantLen:     1,
			wantErr:     false,
			expectedStr: "A",
		},
		{
			name:    "valid counter for n=2",
			counter: 5,
			n:       2,
			wantLen: 2,
			wantErr: false,
		},
		{
			name:    "valid counter for n=5",
			counter: 100,
			n:       5,
			wantLen: 5,
			wantErr: false,
		},
		{
			name:    "counter out of bounds for n=1 (maxRows is 52)",
			counter: 52,
			n:       1,
			wantErr: true,
		},
		{
			name:    "counter out of bounds for n=2 (maxRows is 2704)",
			counter: 2704,
			n:       2,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := LetterN(tt.counter, tt.n)

			if (err != nil) != tt.wantErr {
				t.Fatalf("LetterN(%d, %d) error = %v, wantErr %v", tt.counter, tt.n, err, tt.wantErr)
			}

			if !tt.wantErr {
				if len(got) != tt.wantLen {
					t.Errorf("LetterN(%d, %d) returned string of length %d; want %d", tt.counter, tt.n, len(got), tt.wantLen)
				}

				if tt.expectedStr != "" && got != tt.expectedStr {
					t.Errorf("LetterN(%d, %d) = %q; want %q", tt.counter, tt.n, got, tt.expectedStr)
				}
			}
		})
	}
}

// TestLetterN_Uniqueness verifies that sequential counters produce non-repeating strings
func TestLetterN_Uniqueness(t *testing.T) {
	n := uint(2)
	maxRows := int64(52 * 52) // 2704 combinations
	seen := make(map[string]int64, maxRows)

	for counter := int64(0); counter < maxRows; counter++ {
		str, err := LetterN(counter, n)
		if err != nil {
			t.Fatalf("LetterN(%d, %d) unexpected error: %v", counter, n, err)
		}

		if prevCounter, exists := seen[str]; exists {
			t.Fatalf("Collision detected! LetterN produced duplicate string %q for counters %d and %d", str, prevCounter, counter)
		}

		seen[str] = counter
	}

	if int64(len(seen)) != maxRows {
		t.Errorf("Expected %d unique strings, generated %d", maxRows, len(seen))
	}
}

// TestLetterN_AlphabetCharactersOnly ensures generated output only contains base-52 characters
func TestLetterN_AlphabetCharactersOnly(t *testing.T) {
	validChars := make(map[rune]bool)
	for _, ch := range alphabet {
		validChars[ch] = true
	}

	for counter := int64(0); counter < 100; counter++ {
		str, err := LetterN(counter, 3)
		if err != nil {
			t.Fatalf("Unexpected error for counter %d: %v", counter, err)
		}

		for _, char := range str {
			if !validChars[char] {
				t.Errorf("LetterN(%d, 3) generated invalid character %q in %q", counter, char, str)
			}
		}
	}
}
