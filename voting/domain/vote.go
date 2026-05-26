package domain

import (
	"fmt"
	"time"

	"peer-vote/crypto"
)

// Vote represents a cryptographically signed ballot cast by an eligible voter
// This is a domain entity with behavior and invariants
type Vote struct {
	voterID   string
	publicKey crypto.PublicKey
	choice    string
	timestamp time.Time
	signature crypto.Signature
}

// NewVote creates a new Vote entity
// Constructor enforces the invariant that a Vote cannot be created without a valid signature
func NewVote(voterID, choice string, timestamp time.Time, sig crypto.Signature, pubKey crypto.PublicKey) (*Vote, error) {
	if voterID == "" {
		return nil, fmt.Errorf("voterID cannot be empty")
	}

	if choice == "" {
		return nil, fmt.Errorf("choice cannot be empty")
	}

	if len(sig) == 0 {
		return nil, fmt.Errorf("signature cannot be empty")
	}

	// Enforce signature validation invariant
	message := buildVoteMessage(voterID, choice, timestamp)
	if err := sig.Verify(pubKey, message); err != nil {
		return nil, fmt.Errorf("invalid signature: %w", err)
	}

	return &Vote{
		voterID:   voterID,
		publicKey: pubKey,
		choice:    choice,
		timestamp: timestamp,
		signature: sig,
	}, nil
}

// buildVoteMessage constructs the message that should be signed
func buildVoteMessage(voterID, choice string, timestamp time.Time) []byte {
	// Create deterministic message for signing
	return []byte(fmt.Sprintf("%s|%s|%d", voterID, choice, timestamp.Unix()))
}

// Verify verifies the vote's signature against the given public key
func (v *Vote) Verify(pubKey crypto.PublicKey) error {
	message := buildVoteMessage(v.voterID, v.choice, v.timestamp)
	return v.signature.Verify(pubKey, message)
}

// VoterID returns the voter's unique identifier
func (v *Vote) VoterID() string {
	return v.voterID
}

// Choice returns the ballot choice
func (v *Vote) Choice() string {
	return v.choice
}

// Timestamp returns the vote submission time
func (v *Vote) Timestamp() time.Time {
	return v.timestamp
}

// Signature returns the cryptographic signature
func (v *Vote) Signature() crypto.Signature {
	return v.signature
}

// PublicKey returns the voter's public key
func (v *Vote) PublicKey() crypto.PublicKey {
	return v.publicKey
}

// NewVoteUnsafe creates a Vote without signature validation
// This is used for deserialization where validation happens at the block level
// WARNING: This should only be used internally for deserialization
func NewVoteUnsafe(voterID, choice string, timestamp time.Time, sig crypto.Signature, pubKey crypto.PublicKey) *Vote {
	return &Vote{
		voterID:   voterID,
		publicKey: pubKey,
		choice:    choice,
		timestamp: timestamp,
		signature: sig,
	}
}
