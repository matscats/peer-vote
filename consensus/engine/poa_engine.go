package engine

import (
	"fmt"
	"time"

	"peer-vote/blockchain/domain"
	blockEngine "peer-vote/blockchain/engine"
	consensusDomain "peer-vote/consensus/domain"
	"peer-vote/consensus/ports"
	"peer-vote/crypto"
	mempoolDomain "peer-vote/mempool/domain"
)

// PoAEngine implements the Proof of Authority consensus algorithm
// This engine orchestrates the consensus process using a state machine approach
// It coordinates leader selection, block proposal, validation, and finalization
type PoAEngine struct {
	stateMachine      *consensusDomain.StateMachine
	leaderSelector    *LeaderSelector
	validatorRegistry ports.ValidatorRegistry
	blockBuilder      *blockEngine.BlockBuilder

	blockInterval     time.Duration
	majorityThreshold int
	roundOffset       uint64
}

// NewPoAEngine creates a new PoAEngine with the given components
func NewPoAEngine(
	registry ports.ValidatorRegistry,
	blockBuilder *blockEngine.BlockBuilder,
	blockInterval time.Duration,
) *PoAEngine {
	// Calculate majority threshold: (validator_count / 2) + 1
	validatorCount := registry.Count()
	majorityThreshold := (validatorCount / 2) + 1

	return &PoAEngine{
		stateMachine:      consensusDomain.NewStateMachine(),
		leaderSelector:    NewLeaderSelector(registry),
		validatorRegistry: registry,
		blockBuilder:      blockBuilder,
		blockInterval:     blockInterval,
		majorityThreshold: majorityThreshold,
		roundOffset:       0,
	}
}

// ProposeBlock is called by the node when the block interval elapses
// If this node is the leader for the next height, it proposes a new block
// Returns the proposed block if this node is the leader, nil otherwise
func (e *PoAEngine) ProposeBlock(mempool *mempoolDomain.Mempool, chain *domain.Chain, signer crypto.Signer) (*domain.Block, error) {
	// Calculate next block height
	nextHeight := chain.Height() + 1

	// Check if we are the leader for this height
	if !e.leaderSelector.IsLeaderForRound(nextHeight, e.roundOffset, signer.PublicKey()) {
		return nil, nil // Not our turn to propose
	}

	// Transition to proposing state
	if err := e.stateMachine.Transition(consensusDomain.StateProposing); err != nil {
		return nil, fmt.Errorf("failed to transition to proposing state: %w", err)
	}

	// Get pending votes from mempool
	votes := mempool.GetPending()

	// Build the block
	block, err := e.blockBuilder.Build(nextHeight, chain.Tip().Hash(), votes)
	if err != nil {
		e.stateMachine.Transition(consensusDomain.StateIdle) // Rollback state
		return nil, fmt.Errorf("failed to build block: %w", err)
	}

	// Create a new round for this proposal
	leader, _ := e.leaderSelector.SelectLeaderForRound(nextHeight, e.roundOffset)
	round := consensusDomain.NewRound(nextHeight, leader)
	if err := round.SetProposal(block); err != nil {
		e.stateMachine.Transition(consensusDomain.StateIdle) // Rollback state
		return nil, fmt.Errorf("failed to set proposal: %w", err)
	}

	// Set the round in the state machine
	if err := e.stateMachine.SetRound(round); err != nil {
		e.stateMachine.Transition(consensusDomain.StateIdle) // Rollback state
		return nil, fmt.Errorf("failed to set round: %w", err)
	}

	return block, nil
}

// ValidateProposal is called by the node when a block proposal is received
// Validates that the proposal comes from the correct leader and has valid structure
// Returns error if validation fails
func (e *PoAEngine) ValidateProposal(block *domain.Block, chain *domain.Chain) error {
	// Verify block structure and signature
	if err := block.Verify(); err != nil {
		return fmt.Errorf("block verification failed: %w", err)
	}

	// Verify chain continuity: previous hash must match current tip
	if block.PreviousHash() != chain.Tip().Hash() {
		return fmt.Errorf("previous hash mismatch: expected %s, got %s",
			chain.Tip().Hash().String(), block.PreviousHash().String())
	}

	// Verify height is exactly one greater than current tip
	expectedHeight := chain.Height() + 1
	if block.Height() != expectedHeight {
		return fmt.Errorf("invalid height: expected %d, got %d", expectedHeight, block.Height())
	}

	roundOffset, err := e.roundOffsetForProposer(block.Height(), block.Proposer())
	if err != nil {
		return err
	}
	e.roundOffset = roundOffset

	// Transition to validating state
	if err := e.stateMachine.Transition(consensusDomain.StateValidating); err != nil {
		return fmt.Errorf("failed to transition to validating state: %w", err)
	}

	// Create or update the round for this proposal
	// This is important for non-leaders who receive the proposal
	round := e.stateMachine.CurrentRound()
	if round == nil || round.Height != block.Height() {
		// Create a new round for this height
		leader, _ := e.leaderSelector.SelectLeaderForRound(block.Height(), e.roundOffset)
		round = consensusDomain.NewRound(block.Height(), leader)
		if err := e.stateMachine.SetRound(round); err != nil {
			return fmt.Errorf("failed to set round: %w", err)
		}
	}

	// Set the proposal in the round
	if round.Proposal == nil {
		if err := round.SetProposal(block); err != nil {
			return fmt.Errorf("failed to set proposal in round: %w", err)
		}
	}

	return nil
}

// RecordApproval is called by the node when a block approval is received
// Records the approval from a validator for the current round
// Returns error if the approval is invalid or there is no active round
func (e *PoAEngine) RecordApproval(blockHash crypto.Hash, validator crypto.PublicKey) error {
	round := e.stateMachine.CurrentRound()
	if round == nil {
		return fmt.Errorf("no active round")
	}

	if round.Proposal == nil {
		return fmt.Errorf("no proposal set for current round")
	}

	// Verify the approval is for the current proposal
	if round.Proposal.Hash() != blockHash {
		return fmt.Errorf("approval for different block: expected %s, got %s",
			round.Proposal.Hash().String(), blockHash.String())
	}

	// Verify the validator is authorized
	if !e.validatorRegistry.IsValidator(validator) {
		return fmt.Errorf("approval from non-validator")
	}

	// Record the approval
	if err := round.AddApproval(validator); err != nil {
		return fmt.Errorf("failed to add approval: %w", err)
	}

	return nil
}

// CheckFinalization checks if the current round has received enough approvals to finalize
// Returns true and the block if finalization threshold is met, false otherwise
func (e *PoAEngine) CheckFinalization() (bool, *domain.Block, error) {
	round := e.stateMachine.CurrentRound()
	if round == nil || round.Proposal == nil {
		return false, nil, nil
	}

	// Check if we have majority approvals
	approvalCount := round.ApprovalCount()
	if approvalCount >= e.majorityThreshold {
		return true, round.Proposal, nil
	}

	return false, nil, nil
}

// ResetRound resets the consensus state machine to idle and clears the current round
// This should be called after a block is finalized or when starting a new round
func (e *PoAEngine) ResetRound() {
	e.stateMachine.Transition(consensusDomain.StateIdle)
	e.stateMachine.SetRound(nil)
	e.roundOffset = 0
}

// AdvanceRound moves the current height to the next retry leader after a
// timeout, without advancing the blockchain height.
func (e *PoAEngine) AdvanceRound() {
	e.roundOffset++
	e.stateMachine.Transition(consensusDomain.StateIdle)
	e.stateMachine.SetRound(nil)
}

// CurrentRoundOffset returns the retry round used for leader selection.
func (e *PoAEngine) CurrentRoundOffset() uint64 {
	return e.roundOffset
}

// CurrentState returns the current consensus state
func (e *PoAEngine) CurrentState() consensusDomain.ConsensusState {
	return e.stateMachine.CurrentState()
}

// CurrentRound returns the current consensus round
func (e *PoAEngine) CurrentRound() *consensusDomain.Round {
	return e.stateMachine.CurrentRound()
}

func (e *PoAEngine) roundOffsetForProposer(height uint64, proposer crypto.PublicKey) (uint64, error) {
	validatorCount := e.validatorRegistry.Count()
	if validatorCount == 0 {
		return 0, fmt.Errorf("no validators registered")
	}

	for step := uint64(0); step < uint64(validatorCount); step++ {
		candidateOffset := e.roundOffset + step
		if e.leaderSelector.IsLeaderForRound(height, candidateOffset, proposer) {
			return candidateOffset, nil
		}
	}

	return 0, fmt.Errorf("invalid proposer for height %d: not a designated leader in current retry window", height)
}
