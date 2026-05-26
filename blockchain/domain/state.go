package domain

import (
	"fmt"
	"sync"
)

// State represents the current snapshot of all finalized votes and blockchain data
// This is a domain entity that tracks the voted set and maintains chain reference
type State struct {
	chain    *Chain
	votedSet map[string]bool // voterID -> has voted
	mu       sync.RWMutex
}

// NewState creates a new State with the given chain
func NewState(chain *Chain) (*State, error) {
	if chain == nil {
		return nil, fmt.Errorf("chain cannot be nil")
	}

	state := &State{
		chain:    chain,
		votedSet: make(map[string]bool),
	}

	// Initialize voted set from genesis block (if it contains votes)
	genesis := chain.Tip()
	if genesis != nil {
		for _, vote := range genesis.Votes() {
			state.votedSet[vote.VoterID()] = true
		}
	}

	return state, nil
}

// ApplyBlock applies a block to the state, updating the voted set
// This method should be called when a block is finalized
func (s *State) ApplyBlock(block *Block) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if block == nil {
		return fmt.Errorf("cannot apply nil block")
	}

	// Verify block is the next block in the chain
	if block.Height() != s.chain.Height()+1 {
		return fmt.Errorf("block height mismatch: expected %d, got %d",
			s.chain.Height()+1, block.Height())
	}

	// Check for duplicate voters in the block
	for _, vote := range block.Votes() {
		voterID := vote.VoterID()
		if s.votedSet[voterID] {
			return fmt.Errorf("voter %s has already voted", voterID)
		}
	}

	// Update voted set
	for _, vote := range block.Votes() {
		s.votedSet[vote.VoterID()] = true
	}

	return nil
}

// HasVoted checks if a voter has already cast a vote
func (s *State) HasVoted(voterID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.votedSet[voterID]
}

// Reconstruct reconstructs the state from a sequence of blocks
// This is used during startup to rebuild state from persisted blocks
func (s *State) Reconstruct(blocks []*Block) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(blocks) == 0 {
		return fmt.Errorf("cannot reconstruct from empty block list")
	}

	// Clear current voted set
	s.votedSet = make(map[string]bool)

	// Process each block in order
	for i, block := range blocks {
		if block == nil {
			return fmt.Errorf("block at index %d is nil", i)
		}

		// Verify block height matches index
		if block.Height() != uint64(i) {
			return fmt.Errorf("block height mismatch at index %d: expected %d, got %d",
				i, i, block.Height())
		}

		// Check for duplicate voters
		for _, vote := range block.Votes() {
			voterID := vote.VoterID()
			if s.votedSet[voterID] {
				return fmt.Errorf("duplicate voter %s found during reconstruction at block %d",
					voterID, block.Height())
			}
			s.votedSet[voterID] = true
		}
	}

	return nil
}

// Chain returns the blockchain chain
func (s *State) Chain() *Chain {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.chain
}

// VotedSet returns a copy of the voted set
// Returns a copy to prevent external modification
func (s *State) VotedSet() map[string]bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Return a copy
	votedSetCopy := make(map[string]bool, len(s.votedSet))
	for voterID, voted := range s.votedSet {
		votedSetCopy[voterID] = voted
	}
	return votedSetCopy
}

// VotedCount returns the number of voters who have cast votes
func (s *State) VotedCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.votedSet)
}
