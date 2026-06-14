package engine

import (
	"bytes"
	"fmt"

	"peer-vote/consensus/domain"
	"peer-vote/consensus/ports"
	"peer-vote/crypto"
)

// LeaderSelector implements deterministic round-robin leader selection
// This engine selects the leader for each round based on block height
// Formula: leader = validators[height % validator_count]
type LeaderSelector struct {
	registry ports.ValidatorRegistry
}

// NewLeaderSelector creates a new LeaderSelector with the given validator registry
func NewLeaderSelector(registry ports.ValidatorRegistry) *LeaderSelector {
	return &LeaderSelector{
		registry: registry,
	}
}

// SelectLeader selects the leader for the given block height.
// Uses deterministic round-robin: validators[height % count].
// Returns error if no validators are registered
func (ls *LeaderSelector) SelectLeader(height uint64) (*domain.Validator, error) {
	return ls.SelectLeaderForRound(height, 0)
}

// SelectLeaderForRound selects the leader for a block height and retry round.
// The round offset lets the same height move to another authorized leader after
// a timeout, without changing the block height.
func (ls *LeaderSelector) SelectLeaderForRound(height, roundOffset uint64) (*domain.Validator, error) {
	validators := ls.registry.GetAll()
	if len(validators) == 0 {
		return nil, fmt.Errorf("no validators registered")
	}

	index := (height + roundOffset) % uint64(len(validators))
	return validators[index], nil
}

// IsLeader checks if the given public key is the leader for the specified height
// Returns true if the public key matches the selected leader, false otherwise
func (ls *LeaderSelector) IsLeader(height uint64, pubKey crypto.PublicKey) bool {
	return ls.IsLeaderForRound(height, 0, pubKey)
}

// IsLeaderForRound checks if the public key matches the selected leader for a
// given height and retry round.
func (ls *LeaderSelector) IsLeaderForRound(height, roundOffset uint64, pubKey crypto.PublicKey) bool {
	leader, err := ls.SelectLeaderForRound(height, roundOffset)
	if err != nil {
		return false
	}

	return bytes.Equal(leader.PublicKey(), pubKey)
}
