package adapters

import (
	"fmt"
	"sync"
)

// EligibilityList is a list-based implementation of the EligibilityChecker port
// It maintains an in-memory set of eligible voter IDs
type EligibilityList struct {
	eligibleVoters map[string]bool
	mu             sync.RWMutex
}

// NewEligibilityList creates a new EligibilityList with the given list of eligible voters
func NewEligibilityList(voters []string) *EligibilityList {
	eligibleVoters := make(map[string]bool, len(voters))
	for _, voterID := range voters {
		eligibleVoters[voterID] = true
	}

	return &EligibilityList{
		eligibleVoters: eligibleVoters,
	}
}

// IsEligible checks if a voter is eligible to vote
func (e *EligibilityList) IsEligible(voterID string) (bool, error) {
	if voterID == "" {
		return false, fmt.Errorf("voterID cannot be empty")
	}

	e.mu.RLock()
	defer e.mu.RUnlock()

	return e.eligibleVoters[voterID], nil
}

// GetEligibleVoters returns the list of all eligible voters
func (e *EligibilityList) GetEligibleVoters() ([]string, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	voters := make([]string, 0, len(e.eligibleVoters))
	for voterID := range e.eligibleVoters {
		voters = append(voters, voterID)
	}

	return voters, nil
}
