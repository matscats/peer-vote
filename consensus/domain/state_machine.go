package domain

import (
	"fmt"
	"sync"
)

// ConsensusState represents the current state of the consensus process
type ConsensusState int

const (
	// StateIdle indicates no active consensus round
	StateIdle ConsensusState = iota
	// StateProposing indicates the leader is proposing a block
	StateProposing
	// StateValidating indicates validators are validating the proposal
	StateValidating
	// StateFinalizing indicates the block is being finalized
	StateFinalizing
)

// String returns the string representation of the consensus state
func (s ConsensusState) String() string {
	switch s {
	case StateIdle:
		return "Idle"
	case StateProposing:
		return "Proposing"
	case StateValidating:
		return "Validating"
	case StateFinalizing:
		return "Finalizing"
	default:
		return "Unknown"
	}
}

// StateMachine manages the consensus state transitions
// Thread-safe implementation using sync.RWMutex
type StateMachine struct {
	currentState ConsensusState
	currentRound *Round
	mu           sync.RWMutex
}

// NewStateMachine creates a new StateMachine in the Idle state
func NewStateMachine() *StateMachine {
	return &StateMachine{
		currentState: StateIdle,
	}
}

// Transition attempts to transition to a new state
// Returns error if the transition is invalid
func (sm *StateMachine) Transition(to ConsensusState) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Validate state transition
	if !sm.isValidTransition(sm.currentState, to) {
		return fmt.Errorf("invalid state transition from %s to %s",
			sm.currentState.String(), to.String())
	}

	sm.currentState = to
	return nil
}

// isValidTransition checks if a state transition is valid
// Valid transitions:
//   - Idle -> Proposing (leader starts proposing)
//   - Idle -> Validating (non-leader receives and validates proposal)
//   - Proposing -> Validating (leader validates own proposal)
//   - Validating -> Finalizing (majority approvals received)
//   - Finalizing -> Idle (block finalized, ready for next round)
//   - Any state -> Idle (reset/timeout)
func (sm *StateMachine) isValidTransition(from, to ConsensusState) bool {
	// Allow transition to Idle from any state (reset)
	if to == StateIdle {
		return true
	}

	// Define valid state transitions
	validTransitions := map[ConsensusState][]ConsensusState{
		StateIdle:       {StateProposing, StateValidating}, // Added StateValidating for non-leaders
		StateProposing:  {StateValidating},
		StateValidating: {StateFinalizing},
		StateFinalizing: {StateIdle},
	}

	allowedStates, exists := validTransitions[from]
	if !exists {
		return false
	}

	for _, allowed := range allowedStates {
		if allowed == to {
			return true
		}
	}

	return false
}

// CurrentState returns the current consensus state
func (sm *StateMachine) CurrentState() ConsensusState {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	return sm.currentState
}

// CurrentRound returns the current round
func (sm *StateMachine) CurrentRound() *Round {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	return sm.currentRound
}

// SetRound sets the current round
func (sm *StateMachine) SetRound(round *Round) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.currentRound = round
	return nil
}
