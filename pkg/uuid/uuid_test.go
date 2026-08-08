package uuid

import (
	"math"
	"strings"
	"testing"
)

// TestRFC4122Compliance verifies that generated strings strictly adhere
// to Version 4 (nibble 4 at byte 6) and Variant (binary 10xx at byte 8).
func TestRFC4122Compliance(t *testing.T) {
	testCases := []uint64{0, 15, 55555, math.MaxUint64}

	for _, idx := range testCases {
		uuidStr := SeqIdToUUID(idx)

		// Remove hyphens to check raw hex positions
		clean := strings.ReplaceAll(uuidStr, "-", "")
		if len(clean) != 32 {
			t.Fatalf("Expected 32 hex chars, got %d for UUID %s", len(clean), uuidStr)
		}

		// Check Version 4 bit: Byte index 6 must start with '4' (bits 48-51 = 0100)
		// In clean hex string, bytes 6-7 are represented by indices 12 and 13.
		// Specifically, the high nibble of byte 6 is at index 12.
		versionNibble := clean[12:13]
		if versionNibble != "4" {
			t.Errorf("Index %d: Invalid UUID version nibble. Expected '4', got '%s' in UUID %s", idx, versionNibble, uuidStr)
		}

		// Check Variant bit: Byte index 8 must have its upper 2 bits set to 10 (binary).
		// In hex, this means bytes 8 can be 8, 9, a, or b (binary 1000, 1001, 1010, 1011).
		// Byte 8 is represented by clean string indices 16 and 17.
		variantByteHex := clean[16:18]
		validVariants := map[string]bool{"80": true, "81": true, "82": true, "83": true, "84": true, "85": true, "86": true, "87": true, "88": true, "89": true, "8a": true, "8b": true, "8c": true, "8d": true, "8e": true, "8f": true}
		if !validVariants[variantByteHex] {
			t.Errorf("Index %d: Invalid UUID variant byte. Got '%s' in UUID %s", idx, variantByteHex, uuidStr)
		}
	}
}

// TestDeterminism checks that the same index always produces the exact same UUID string.
func TestDeterminism(t *testing.T) {
	var idx uint64 = 123456789
	firstRun := SeqIdToUUID(idx)
	secondRun := SeqIdToUUID(idx)

	if firstRun != secondRun {
		t.Errorf("Non-deterministic output: got '%s' then '%s' for index %d", firstRun, secondRun, idx)
	}
}
