package domain

import (
	"fmt"
	"sync"
)

// Chain represents an immutable, ordered sequence of blocks linked by cryptographic hashes
// This is a domain entity that enforces chain integrity invariants
type Chain struct {
	blocks []*Block
	tip    *Block
	mu     sync.RWMutex
}

// NewChain creates a new Chain with the given genesis block
func NewChain(genesis *Block) (*Chain, error) {
	if genesis == nil {
		return nil, fmt.Errorf("genesis block cannot be nil")
	}

	if genesis.Height() != 0 {
		return nil, fmt.Errorf("genesis block must have height 0")
	}

	return &Chain{
		blocks: []*Block{genesis},
		tip:    genesis,
	}, nil
}

// Append adds a new block to the chain
// Enforces chain integrity: previous hash must match tip hash, height must be tip+1
func (c *Chain) Append(block *Block) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if block == nil {
		return fmt.Errorf("cannot append nil block")
	}

	// Verify block links to current tip
	if block.PreviousHash() != c.tip.Hash() {
		return fmt.Errorf("block previous hash does not match chain tip: expected %s, got %s",
			c.tip.Hash().String(), block.PreviousHash().String())
	}

	// Verify height is exactly one greater than tip
	expectedHeight := c.tip.Height() + 1
	if block.Height() != expectedHeight {
		return fmt.Errorf("block height must be %d, got %d", expectedHeight, block.Height())
	}

	// Verify block integrity
	if err := block.Verify(); err != nil {
		return fmt.Errorf("block verification failed: %w", err)
	}

	// Append block to chain
	c.blocks = append(c.blocks, block)
	c.tip = block

	return nil
}

// Tip returns the current chain tip (most recent block)
func (c *Chain) Tip() *Block {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.tip
}

// GetBlock retrieves a block by height
func (c *Chain) GetBlock(height uint64) (*Block, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if height >= uint64(len(c.blocks)) {
		return nil, fmt.Errorf("block height %d not found (chain height: %d)", height, c.Height())
	}

	return c.blocks[height], nil
}

// VerifyIntegrity verifies the entire chain's hash integrity
// Checks that each block's previous hash matches the previous block's hash
func (c *Chain) VerifyIntegrity() error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if len(c.blocks) == 0 {
		return fmt.Errorf("chain is empty")
	}

	// Verify genesis block
	if c.blocks[0].Height() != 0 {
		return fmt.Errorf("genesis block must have height 0")
	}

	// Verify each block links to its predecessor
	for i := 1; i < len(c.blocks); i++ {
		currentBlock := c.blocks[i]
		previousBlock := c.blocks[i-1]

		// Verify height sequence
		if currentBlock.Height() != previousBlock.Height()+1 {
			return fmt.Errorf("height mismatch at block %d: expected %d, got %d",
				i, previousBlock.Height()+1, currentBlock.Height())
		}

		// Verify hash chain
		if currentBlock.PreviousHash() != previousBlock.Hash() {
			return fmt.Errorf("hash chain broken at block %d: previous hash %s does not match block %d hash %s",
				i, currentBlock.PreviousHash().String(), i-1, previousBlock.Hash().String())
		}

		// Verify block integrity
		if err := currentBlock.Verify(); err != nil {
			return fmt.Errorf("block %d verification failed: %w", i, err)
		}
	}

	return nil
}

// Height returns the current chain height (height of the tip block)
func (c *Chain) Height() uint64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.tip.Height()
}

// GetAllBlocks returns all blocks in the chain
// This is useful for state reconstruction and persistence
func (c *Chain) GetAllBlocks() []*Block {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Return a copy to prevent external modification
	blocks := make([]*Block, len(c.blocks))
	copy(blocks, c.blocks)
	return blocks
}
