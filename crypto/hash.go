package crypto

import (
	"crypto/sha256"
	"encoding/hex"
)

// Hash represents a SHA-256 hash value
type Hash [32]byte

// NewHash creates a new Hash from the given data using SHA-256
func NewHash(data []byte) Hash {
	return sha256.Sum256(data)
}

// String returns the hexadecimal string representation of the hash
func (h Hash) String() string {
	return hex.EncodeToString(h[:])
}

// Bytes returns the hash as a byte slice
func (h Hash) Bytes() []byte {
	return h[:]
}
