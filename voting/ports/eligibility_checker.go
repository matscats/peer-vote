package ports

// EligibilityChecker is a port interface for checking voter eligibility
// This follows hexagonal architecture - the domain defines what it needs,
// and infrastructure provides the implementation
type EligibilityChecker interface {
	// IsEligible checks if a voter is eligible to vote
	IsEligible(voterID string) (bool, error)

	// GetEligibleVoters returns the list of all eligible voters
	GetEligibleVoters() ([]string, error)
}
