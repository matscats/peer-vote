package engine

import (
	"fmt"
	"sync"

	"peer-vote/blockchain/domain"
	"peer-vote/blockchain/ports"
)

// SyncManager handles state synchronization with network peers
// This engine coordinates blockchain synchronization when a node is behind
type SyncManager struct {
	state     *domain.State
	blockRepo ports.BlockRepository
	syncing   bool
	syncedTip uint64 // The height we've synced to
	mu        sync.RWMutex
}

// NewSyncManager creates a new SyncManager
func NewSyncManager(state *domain.State, blockRepo ports.BlockRepository) (*SyncManager, error) {
	if state == nil {
		return nil, fmt.Errorf("state cannot be nil")
	}
	if blockRepo == nil {
		return nil, fmt.Errorf("blockRepo cannot be nil")
	}

	return &SyncManager{
		state:     state,
		blockRepo: blockRepo,
		syncing:   false,
		syncedTip: state.Chain().Height(),
	}, nil
}

// Sync initiates synchronization with the network
// This method requests missing blocks from peers and validates them before applying
// Returns error if synchronization fails
func (s *SyncManager) Sync(peerTipHeight uint64, blockProvider func(height uint64) (*domain.Block, error)) error {
	s.mu.Lock()
	if s.syncing {
		s.mu.Unlock()
		return fmt.Errorf("sync already in progress")
	}
	s.syncing = true
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.syncing = false
		s.mu.Unlock()
	}()

	currentHeight := s.state.Chain().Height()

	// Check if we're already synced
	if currentHeight >= peerTipHeight {
		return nil // Already synced
	}

	// Request blocks sequentially from current height + 1 to peer tip
	for height := currentHeight + 1; height <= peerTipHeight; height++ {
		// Request block from peer
		block, err := blockProvider(height)
		if err != nil {
			return fmt.Errorf("failed to request block at height %d: %w", height, err)
		}

		// Validate block before applying
		if err := s.validateBlock(block); err != nil {
			return fmt.Errorf("block validation failed at height %d: %w", height, err)
		}

		// Append to chain
		if err := s.state.Chain().Append(block); err != nil {
			return fmt.Errorf("failed to append block at height %d: %w", height, err)
		}

		// Persist block
		if err := s.blockRepo.Store(block); err != nil {
			return fmt.Errorf("failed to persist block at height %d: %w", height, err)
		}

		// Apply block to state
		if err := s.state.ApplyBlock(block); err != nil {
			return fmt.Errorf("failed to apply block at height %d: %w", height, err)
		}

		// Update synced tip
		s.mu.Lock()
		s.syncedTip = height
		s.mu.Unlock()
	}

	return nil
}

// RequestBlocks requests a range of blocks from peers
// This is a helper method that can be used by the node engine to request blocks
// fromHeight: starting height (inclusive)
// toHeight: ending height (inclusive)
func (s *SyncManager) RequestBlocks(fromHeight, toHeight uint64, blockProvider func(height uint64) (*domain.Block, error)) ([]*domain.Block, error) {
	if fromHeight > toHeight {
		return nil, fmt.Errorf("invalid height range: from %d to %d", fromHeight, toHeight)
	}

	blocks := make([]*domain.Block, 0, toHeight-fromHeight+1)

	for height := fromHeight; height <= toHeight; height++ {
		block, err := blockProvider(height)
		if err != nil {
			return nil, fmt.Errorf("failed to request block at height %d: %w", height, err)
		}

		// Validate block
		if err := s.validateBlock(block); err != nil {
			return nil, fmt.Errorf("block validation failed at height %d: %w", height, err)
		}

		blocks = append(blocks, block)
	}

	return blocks, nil
}

// validateBlock validates a block before adding it to the chain
// This includes structural validation and signature verification
func (s *SyncManager) validateBlock(block *domain.Block) error {
	if block == nil {
		return fmt.Errorf("block is nil")
	}

	// Verify block structure and signature
	if err := block.Verify(); err != nil {
		return fmt.Errorf("block verification failed: %w", err)
	}

	// Verify block links to current chain tip
	currentTip := s.state.Chain().Tip()
	if block.PreviousHash() != currentTip.Hash() {
		return fmt.Errorf("block previous hash does not match chain tip: expected %s, got %s",
			currentTip.Hash().String(), block.PreviousHash().String())
	}

	// Verify height is exactly one greater than current tip
	expectedHeight := currentTip.Height() + 1
	if block.Height() != expectedHeight {
		return fmt.Errorf("block height must be %d, got %d", expectedHeight, block.Height())
	}

	return nil
}

// IsSynced returns true if the node is synced with the network
// A node is considered synced if it's not currently syncing and has processed all known blocks
func (s *SyncManager) IsSynced() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return !s.syncing
}

// IsSyncing returns true if synchronization is currently in progress
func (s *SyncManager) IsSyncing() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.syncing
}

// GetSyncedHeight returns the height we've synced to
func (s *SyncManager) GetSyncedHeight() uint64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.syncedTip
}

// GetCurrentHeight returns the current chain height
func (s *SyncManager) GetCurrentHeight() uint64 {
	return s.state.Chain().Height()
}

// NeedsSynchronization checks if the node needs to sync based on peer tip height
func (s *SyncManager) NeedsSynchronization(peerTipHeight uint64) bool {
	currentHeight := s.state.Chain().Height()
	return peerTipHeight > currentHeight
}
