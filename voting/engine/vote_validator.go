package engine

import (
	"fmt"

	"peer-vote/crypto"
	"peer-vote/voting/domain"
	"peer-vote/voting/ports"
)

// VotedSetChecker is an interface for checking if a voter has already voted
// This allows the VoteValidator to check against the blockchain state
type VotedSetChecker interface {
	HasVoted(voterID string) bool
}

// VoteValidator is an engine that orchestrates vote validation logic
// It validates signature, eligibility, and duplicate checks
type VoteValidator struct {
	eligibility ports.EligibilityChecker
}

// NewVoteValidator creates a new VoteValidator
func NewVoteValidator(eligibility ports.EligibilityChecker) *VoteValidator {
	return &VoteValidator{
		eligibility: eligibility,
	}
}

// Validate validates a vote against all rules:
// 1. Signature validation (already enforced by Vote constructor)
// 2. Eligibility check
// 3. Duplicate vote check
func (v *VoteValidator) Validate(vote *domain.Vote, pubKey crypto.PublicKey, state VotedSetChecker) error {
	// 1. Verify signature (Requirements 1.1, 1.3)
	if err := vote.Verify(pubKey); err != nil {
		return fmt.Errorf("signature verification failed: %w", err)
	}

	// 2. Check eligibility (Requirements 1.2, 18.5)
	eligible, err := v.eligibility.IsEligible(vote.VoterID())
	if err != nil {
		return fmt.Errorf("eligibility check failed: %w", err)
	}
	if !eligible {
		return fmt.Errorf("voter %s is not eligible to vote", vote.VoterID())
	}

	// 3. Check for duplicate vote (Requirements 1.2, 1.4)
	if state.HasVoted(vote.VoterID()) {
		return fmt.Errorf("voter %s has already voted", vote.VoterID())
	}

	return nil
}
