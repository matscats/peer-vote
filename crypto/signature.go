package crypto

import (
	"crypto/ed25519"
	"fmt"
)

// Signature represents a cryptographic signature
type Signature []byte

// Verify verifies the signature against the given public key and message
func (s Signature) Verify(pubKey PublicKey, message []byte) error {
	if !ed25519.Verify(ed25519.PublicKey(pubKey), message, s) {
		return fmt.Errorf("signature verification failed")
	}
	return nil
}
