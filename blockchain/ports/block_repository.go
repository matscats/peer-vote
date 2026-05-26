package ports

import (
	"peer-vote/blockchain/domain"
	"peer-vote/crypto"
)

// BlockRepository is a port interface for persisting and retrieving blocks
// This interface defines what the blockchain domain needs from storage
type BlockRepository interface {
	// Store persists a block to storage
	Store(block *domain.Block) error

	// GetByHeight retrieves a block by its height
	GetByHeight(height uint64) (*domain.Block, error)

	// GetByHash retrieves a block by its hash
	GetByHash(hash crypto.Hash) (*domain.Block, error)

	// GetTip retrieves the current chain tip (highest block)
	GetTip() (*domain.Block, error)

	// GetAll retrieves all blocks in the chain
	GetAll() ([]*domain.Block, error)
}
