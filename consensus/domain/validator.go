package domain

import (
	"peer-vote/crypto"
)

// Validator represents an authorized node with permission to propose and validate blocks
type Validator struct {
	publicKey crypto.PublicKey
	address   string // Network address
}

// NewValidator creates a new Validator entity
func NewValidator(pubKey crypto.PublicKey, addr string) *Validator {
	return &Validator{
		publicKey: pubKey,
		address:   addr,
	}
}

// PublicKey returns the validator's public key
func (v *Validator) PublicKey() crypto.PublicKey {
	return v.publicKey
}

// Address returns the validator's network address
func (v *Validator) Address() string {
	return v.address
}
