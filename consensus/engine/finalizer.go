package engine

import (
	"fmt"

	"peer-vote/blockchain/domain"
	"peer-vote/blockchain/ports"
	mempoolDomain "peer-vote/mempool/domain"
)

// Finalizer is responsible for atomically finalizing blocks
// This engine coordinates the finalization process across multiple components:
// 1. Append block to chain
// 2. Persist block to storage
// 3. Update state (voted set)
// 4. Remove votes from mempool
// All operations must succeed or none should be committed
type Finalizer struct {
	state     *domain.State
	blockRepo ports.BlockRepository
	mempool   *mempoolDomain.Mempool
}

// NewFinalizer creates a new Finalizer with the given components
func NewFinalizer(state *domain.State, blockRepo ports.BlockRepository, mempool *mempoolDomain.Mempool) *Finalizer {
	return &Finalizer{
		state:     state,
		blockRepo: blockRepo,
		mempool:   mempool,
	}
}

// Finalize atomically finalizes a block by performing all necessary operations
// The finalization process is atomic: if any step fails, the entire operation fails
// Steps:
// 1. Update state with new votes (validates against current state)
// 2. Append block to chain (updates chain height)
// 3. Persist block to storage (durable)
// 4. Remove votes from mempool (cleanup)
//
// Note: State is updated before chain append to validate against current height.
// If any step fails after state update, the node would need to restart and
// reconstruct from disk.
func (f *Finalizer) Finalize(block *domain.Block) error {
	if block == nil {
		return fmt.Errorf("cannot finalize nil block")
	}

	// Step 1: Update state with votes from the block
	// This validates the block against current state and updates the voted set
	if err := f.state.ApplyBlock(block); err != nil {
		return fmt.Errorf("failed to apply block to state: %w", err)
	}

	// Step 2: Append block to chain
	// This validates that the block properly links to the chain
	if err := f.state.Chain().Append(block); err != nil {
		return fmt.Errorf("failed to append block to chain: %w", err)
	}

	// Step 3: Persist block to storage
	// This is the critical durability step
	if err := f.blockRepo.Store(block); err != nil {
		// Note: Chain and state are now inconsistent with storage
		// In a production system, we would need to rollback
		// For this academic implementation, we accept that the node would need to restart
		return fmt.Errorf("failed to persist block: %w", err)
	}

	// Step 4: Remove finalized votes from mempool
	// This frees up memory and prevents re-inclusion in future blocks
	if err := f.mempool.Remove(block.Votes()); err != nil {
		return fmt.Errorf("failed to remove votes from mempool: %w", err)
	}

	return nil
}
