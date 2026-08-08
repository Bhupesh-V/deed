package uuid

import (
	"encoding/binary"
	"fmt"
)

// Secret 64-bit XOR keys for the Feistel cipher rounds
// TODO: generate dynamically and move to user's db in a special table
const (
	Key1 uint64 = 0x8F3A2B1C0D9E8F7A
	Key2 uint64 = 0x4B3C2A1D0E9F8A7B
	Key3 uint64 = 0x9A8B7C6D5E4F3A2B
)

// feistelRound mixes bits using XOR and a non-linear integer multiplier
func feistelRound(val uint32, key uint64) uint32 {
	// 0x9E3779B9 is Knuth's Multiplicative Hash Constant, derived directly from the Golden Ratio ($\phi \approx 1.6180339887$).
	return uint32((uint64(val) * 0x9E3779B9) ^ key)
}

// scramble applies a 3-round Feistel cipher to achieve an avalanche effect
func scramble(seqIdx uint64) uint64 {
	L, R := uint32(seqIdx>>32), uint32(seqIdx)

	R ^= feistelRound(L, Key1)
	L ^= feistelRound(R, Key2)
	R ^= feistelRound(L, Key3)

	return (uint64(L) << 32) | uint64(R)
}

// unscramble reverses the 3-round Feistel cipher
func unscramble(scrambled uint64) uint64 {
	L, R := uint32(scrambled>>32), uint32(scrambled)

	R ^= feistelRound(L, Key3)
	L ^= feistelRound(R, Key2)
	R ^= feistelRound(L, Key1)

	return (uint64(L) << 32) | uint64(R)
}

// SeqIdxToUUID converts a sequential index into a random-looking UUID v4 string
func SeqIdToUUID(seqIdx uint64) string {
	scrambled := scramble(seqIdx)

	var b [16]byte
	var tmp [8]byte
	binary.BigEndian.PutUint64(tmp[:], scrambled)

	// Scatter the 8 payload bytes across unreserved byte positions
	b[0], b[1], b[2], b[3] = tmp[0], tmp[1], tmp[2], tmp[3]
	b[4], b[5] = tmp[4], tmp[5]
	b[7] = tmp[6]
	b[9] = tmp[7]

	// Set RFC 4122 Version 4 (0x40) and Variant (0x80)
	b[6] = 0x40
	b[8] = 0x80

	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
