package domain

import (
	"fmt"
	"sync"

	"peer-vote/blockchain/domain"
	"peer-vote/crypto"
)

// Round represents a consensus cycle in which one validator acts as leader to propose a block
// Thread-safe implementation using sync.RWMutex
type Round struct {
	Height    uint64
	Leader    *Validator
	Proposal  *domain.Block
	Approvals map[string]bool // Map of hex-encoded public key -> approval status
	mu        sync.RWMutex
}

// NewRound creates a new Round entity
func NewRound(height uint64, leader *Validator) *Round {
	return &Round{
		Height:    height,
		Leader:    leader,
		Approvals: make(map[string]bool),
	}
}

// SetProposal sets the proposed block for this round
// Returns error if a proposal has already been set
func (r *Round) SetProposal(block *domain.Block) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.Proposal != nil {
		return fmt.Errorf("proposal already set for round at height %d", r.Height)
	}

	if block.Height() != r.Height {
		return fmt.Errorf("block height %d does not match round height %d", block.Height(), r.Height)
	}

	r.Proposal = block
	return nil
}

// AddApproval records an approval from a validator
// Returns error if the validator has already approved
func (r *Round) AddApproval(validator crypto.PublicKey) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.Proposal == nil {
		return fmt.Errorf("cannot add approval: no proposal set for round at height %d", r.Height)
	}

	// Use hex encoding of public key as map key for consistent comparison
	keyStr := fmt.Sprintf("%x", validator)

	if r.Approvals[keyStr] {
		return fmt.Errorf("validator has already approved this round")
	}

	r.Approvals[keyStr] = true
	return nil
}

// ApprovalCount returns the number of approvals received
func (r *Round) ApprovalCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.Approvals)
}

// HasApproval checks if a validator has approved this round
func (r *Round) HasApproval(validator crypto.PublicKey) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	keyStr := fmt.Sprintf("%x", validator)
	return r.Approvals[keyStr]
}
