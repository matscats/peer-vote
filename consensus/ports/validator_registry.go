package ports

import (
	"peer-vote/consensus/domain"
	"peer-vote/crypto"
)

// ValidatorRegistry is a port interface for managing the list of authorized validators
// This interface defines what the consensus domain needs from validator management
type ValidatorRegistry interface {
	// GetAll returns all validators in the registry
	GetAll() []*domain.Validator

	// GetByPublicKey retrieves a validator by their public key
	GetByPublicKey(pubKey crypto.PublicKey) (*domain.Validator, error)

	// IsValidator checks if a public key belongs to an authorized validator
	IsValidator(pubKey crypto.PublicKey) bool

	// Count returns the total number of validators
	Count() int
}
