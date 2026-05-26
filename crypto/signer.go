package crypto

import (
	"crypto/ed25519"
	"fmt"
)

// Signer is an interface for signing messages
type Signer interface {
	Sign(message []byte) (Signature, error)
	PublicKey() PublicKey
}

// Ed25519Signer implements the Signer interface using Ed25519
type Ed25519Signer struct {
	keyPair *KeyPair
}

// NewEd25519Signer creates a new Ed25519Signer with the given key pair
func NewEd25519Signer(keyPair *KeyPair) *Ed25519Signer {
	return &Ed25519Signer{
		keyPair: keyPair,
	}
}

// Sign signs the given message using Ed25519
func (s *Ed25519Signer) Sign(message []byte) (Signature, error) {
	if s.keyPair == nil || s.keyPair.Private == nil {
		return nil, fmt.Errorf("no private key available for signing")
	}

	sig := ed25519.Sign(ed25519.PrivateKey(s.keyPair.Private), message)
	return Signature(sig), nil
}

// PublicKey returns the public key associated with this signer
func (s *Ed25519Signer) PublicKey() PublicKey {
	return s.keyPair.Public
}
