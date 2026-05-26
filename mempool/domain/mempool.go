package domain

import (
	"fmt"
	"sync"

	"peer-vote/voting/domain"
)

// Mempool temporarily stores unconfirmed votes in memory
// This is a domain entity following DDD principles
// Uses a map for O(1) lookup by voterID and a slice to maintain insertion order
// All operations are thread-safe using sync.RWMutex
type Mempool struct {
	votes map[string]*domain.Vote // voterID -> vote for O(1) lookup
	order []string                // Maintains insertion order
	mu    sync.RWMutex
}

// NewMempool creates a new Mempool entity
func NewMempool() *Mempool {
	return &Mempool{
		votes: make(map[string]*domain.Vote),
		order: make([]string, 0),
	}
}

// Add adds a vote to the mempool
// Returns an error if a vote from the same voter already exists
func (m *Mempool) Add(vote *domain.Vote) error {
	if vote == nil {
		return fmt.Errorf("vote cannot be nil")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	voterID := vote.VoterID()

	// Check if vote from this voter already exists
	if _, exists := m.votes[voterID]; exists {
		return fmt.Errorf("vote from voter %s already in mempool", voterID)
	}

	// Add to map and order slice
	m.votes[voterID] = vote
	m.order = append(m.order, voterID)

	return nil
}

// GetPending returns all pending votes in the order they were received
func (m *Mempool) GetPending() []*domain.Vote {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return votes in insertion order
	pending := make([]*domain.Vote, 0, len(m.order))
	for _, voterID := range m.order {
		if vote, exists := m.votes[voterID]; exists {
			pending = append(pending, vote)
		}
	}

	return pending
}

// Remove removes the specified votes from the mempool
// This is typically called when votes are included in a finalized block
func (m *Mempool) Remove(votes []*domain.Vote) error {
	if len(votes) == 0 {
		return nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Remove votes from map
	for _, vote := range votes {
		if vote != nil {
			delete(m.votes, vote.VoterID())
		}
	}

	// Rebuild order slice without removed votes
	newOrder := make([]string, 0, len(m.votes))
	for _, voterID := range m.order {
		if _, exists := m.votes[voterID]; exists {
			newOrder = append(newOrder, voterID)
		}
	}
	m.order = newOrder

	return nil
}

// Contains checks if a vote from the specified voter exists in the mempool
func (m *Mempool) Contains(voterID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	_, exists := m.votes[voterID]
	return exists
}

// Size returns the number of votes currently in the mempool
func (m *Mempool) Size() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.votes)
}
