package fake

import (
	"math"
	"math/big"
	"regexp"
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

func TestFloat_Determinism(t *testing.T) {
	f := &Fake{}
	idx := int64(42)
	precision, scale := 10, 2

	// Generating twice with the same index must yield identical results
	val1 := f.Float(idx, precision, scale)
	val2 := f.Float(idx, precision, scale)

	if val1 != val2 {
		t.Errorf("expected deterministic results, got %f and %f", val1, val2)
	}
}

func TestFloat_PrecisionAndScaleBounds(t *testing.T) {
	f := &Fake{}

	tests := []struct {
		name      string
		precision int
		scale     int
		maxVal    float64
	}{
		{
			name:      "NUMERIC(5,2)", // 3 integer digits -> max < 1000.00
			precision: 5,
			scale:     2,
			maxVal:    1000.00,
		},
		{
			name:      "NUMERIC(3,1)", // 2 integer digits -> max < 100.0
			precision: 3,
			scale:     1,
			maxVal:    100.0,
		},
		{
			name:      "NUMERIC(6,6)", // 0 integer digits -> max < 1.0
			precision: 6,
			scale:     6,
			maxVal:    1.000000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for idx := int64(0); idx < 100; idx++ {
				val := f.Float(idx, tt.precision, tt.scale)

				if val < 0 || val >= tt.maxVal {
					t.Errorf("idx %d: value %f out of bounds for max %f", idx, val, tt.maxVal)
				}
			}
		})
	}
}

func TestFloat_ScaleZero(t *testing.T) {
	f := &Fake{}

	for idx := int64(0); idx < 20; idx++ {
		val := f.Float(idx, 5, 0)

		// Fractional part should always be 0 when scale is 0
		if val != math.Trunc(val) {
			t.Errorf("idx %d: expected whole number, got %f", idx, val)
		}
	}
}

func TestFloat_DefaultsAndEdgeCases(t *testing.T) {
	f := &Fake{}

	t.Run("Negative precision defaults to 10", func(t *testing.T) {
		val := f.Float(1, -5, 2)
		if val >= 100000000.0 { // precision 10, scale 2 -> integer max < 10^8
			t.Errorf("value %f exceeded default precision limits", val)
		}
	})

	t.Run("Negative scale defaults to 2", func(t *testing.T) {
		val := f.Float(1, 10, -1)
		// Check scale by verifying rounding to 2 decimal places
		rounded := math.Round(val*100) / 100
		if val != rounded {
			t.Errorf("expected 2 decimal places, got %f", val)
		}
	})

	t.Run("Precision capped at 15", func(t *testing.T) {
		// Should not panic or overflow even with excessive precision requested
		val := f.Float(1, 30, 2)
		if math.IsNaN(val) || math.IsInf(val, 0) {
			t.Errorf("got invalid float: %f", val)
		}
	})

	t.Run("Negative row index", func(t *testing.T) {
		val := f.Float(-10, 5, 2)
		if math.IsNaN(val) || math.IsInf(val, 0) {
			t.Errorf("got invalid float for negative idx: %f", val)
		}
	})
}

func TestRegex_ConfigPatterns(t *testing.T) {
	tests := []struct {
		field   string
		pattern string
	}{
		// users
		{
			field:   "users.email",
			pattern: `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`,
		},
		{
			field:   "users.username",
			pattern: `^[a-zA-Z0-9_-]{3,30}$`,
		},
		{
			field:   "users.password_hash",
			pattern: `^\$2[ayb]\$[0-9]{2}\$[A-Za-z0-9./]{53}$`,
		},

		// orders
		{
			field:   "orders.order_number",
			pattern: `^ORD-[0-9]{8}-[0-9]{4}$`,
		},
		{
			field:   "orders.status",
			pattern: `^(PENDING|PAID|SHIPPED|COMPLETED|CANCELLED)$`,
		},
		{
			field:   "orders.total_amount",
			pattern: `^[0-9]{1,10}\.[0-9]{2}$`,
		},

		// shipments
		{
			field:   "shipments.tracking_number",
			pattern: `^[A-Z0-9]{8,100}$`,
		},

		// shipment_tracking_events
		{
			field:   "shipment_tracking_events.location",
			pattern: `^[A-Za-z\s.-]+,\s*[A-Z]{2}(\s*[0-9]{5})?$`,
		},
		{
			field:   "shipment_tracking_events.status_description",
			pattern: `^[A-Za-z0-9\s.,-]{1,255}$`,
		},

		// delivery_proofs
		{
			field:   "delivery_proofs.recipient_signature_url",
			pattern: `^https?://[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}(/.*)?\.(png|jpg|jpeg|svg)$`,
		},
		{
			field:   "delivery_proofs.photo_url",
			pattern: `^https?://[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}(/.*)?\.(png|jpg|jpeg|webp)$`,
		},

		// proof_verifications
		{
			field:   "proof_verifications.confidence_score",
			pattern: `^(100\.00|[0-9]{1,2}\.[0-9]{2})$`,
		},
	}

	f := &Fake{}

	for _, tt := range tests {
		t.Run(tt.field, func(t *testing.T) {
			// Compile pattern to validate against generated strings
			re, err := regexp.Compile(tt.pattern)
			if err != nil {
				t.Fatalf("invalid regex pattern in test case: %v", err)
			}

			for idx := int64(0); idx < 50; idx++ {
				val, err := f.Regex(idx, tt.pattern)
				if err != nil {
					t.Fatalf("idx %d: unexpected error generating string: %v", idx, err)
				}

				// 1. Verify generated string matches pattern
				if !re.MatchString(val) {
					t.Errorf("idx %d: generated %q does not match pattern %q", idx, val, tt.pattern)
				}

				// 2. Verify determinism
				valRepeat, _ := f.Regex(idx, tt.pattern)
				if val != valRepeat {
					t.Errorf("idx %d: expected deterministic output, got %q and %q", idx, val, valRepeat)
				}
			}
		})
	}
}
