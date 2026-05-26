package engine

import (
	"fmt"
	"log"

	"peer-vote/blockchain/domain"
	"peer-vote/blockchain/ports"
)

// RecoverFromStorage reconstructs the blockchain state from persisted blocks
// This is called during node startup to restore state after a restart
// Steps:
// 1. Load all blocks from storage
// 2. Verify chain integrity
// 3. Reconstruct the voted set from all blocks
// 4. Detect and reject corrupted state
func RecoverFromStorage(blockRepo ports.BlockRepository) (*domain.Chain, *domain.State, error) {
	log.Println("Starting state recovery from storage...")

	// Step 1: Load all blocks from storage
	blocks, err := blockRepo.GetAll()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load blocks from storage: %w", err)
	}

	if len(blocks) == 0 {
		return nil, nil, fmt.Errorf("no blocks found in storage (genesis block missing)")
	}

	log.Printf("Loaded %d blocks from storage\n", len(blocks))

	// Step 2: Verify genesis block
	genesis := blocks[0]
	if genesis.Height() != 0 {
		return nil, nil, fmt.Errorf("first block is not genesis (height: %d)", genesis.Height())
	}

	// Create chain with genesis block
	chain, err := domain.NewChain(genesis)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create chain with genesis: %w", err)
	}

	// Step 3: Append remaining blocks and verify chain integrity
	for i := 1; i < len(blocks); i++ {
		block := blocks[i]

		// Verify block height matches expected sequence
		if block.Height() != uint64(i) {
			return nil, nil, fmt.Errorf("block height mismatch at index %d: expected %d, got %d",
				i, i, block.Height())
		}

		// Append block to chain (this validates chain continuity)
		if err := chain.Append(block); err != nil {
			return nil, nil, fmt.Errorf("failed to append block at height %d: %w", block.Height(), err)
		}
	}

	log.Printf("Chain integrity verified (height: %d)\n", chain.Height())

	// Step 4: Verify complete chain integrity
	if err := chain.VerifyIntegrity(); err != nil {
		return nil, nil, fmt.Errorf("chain integrity verification failed: %w", err)
	}

	// Step 5: Create state and reconstruct voted set
	state, err := domain.NewState(chain)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create state: %w", err)
	}

	// Reconstruct voted set from all blocks
	if err := state.Reconstruct(blocks); err != nil {
		return nil, nil, fmt.Errorf("failed to reconstruct voted set: %w", err)
	}

	log.Printf("State reconstructed successfully (voted count: %d)\n", state.VotedCount())

	return chain, state, nil
}

// InitializeGenesis creates and persists a genesis block if storage is empty
// This is called during first-time node startup
func InitializeGenesis(blockRepo ports.BlockRepository, genesisBlock *domain.Block) error {
	log.Println("Initializing genesis block...")

	// Verify this is a valid genesis block
	if genesisBlock.Height() != 0 {
		return fmt.Errorf("genesis block must have height 0, got %d", genesisBlock.Height())
	}

	// Verify genesis block integrity
	if err := genesisBlock.Verify(); err != nil {
		return fmt.Errorf("genesis block verification failed: %w", err)
	}

	// Persist genesis block
	if err := blockRepo.Store(genesisBlock); err != nil {
		return fmt.Errorf("failed to persist genesis block: %w", err)
	}

	log.Printf("Genesis block initialized and persisted (hash: %s)\n", genesisBlock.Hash().String())

	return nil
}

// ValidateStorageState checks if the storage state is valid and consistent
// This is called during startup to detect corruption
// Returns error if storage is corrupted or inconsistent
func ValidateStorageState(blockRepo ports.BlockRepository) error {
	log.Println("Validating storage state...")

	// Try to load all blocks
	blocks, err := blockRepo.GetAll()
	if err != nil {
		return fmt.Errorf("failed to load blocks: %w", err)
	}

	if len(blocks) == 0 {
		return fmt.Errorf("storage is empty (no genesis block)")
	}

	// Verify genesis block
	if blocks[0].Height() != 0 {
		return fmt.Errorf("first block is not genesis (height: %d)", blocks[0].Height())
	}

	// Verify block sequence
	for i := 0; i < len(blocks); i++ {
		if blocks[i].Height() != uint64(i) {
			return fmt.Errorf("block sequence broken at index %d: expected height %d, got %d",
				i, i, blocks[i].Height())
		}

		// Verify block integrity
		if err := blocks[i].Verify(); err != nil {
			return fmt.Errorf("block %d failed verification: %w", i, err)
		}

		// Verify hash chain (except for genesis)
		if i > 0 {
			if blocks[i].PreviousHash() != blocks[i-1].Hash() {
				return fmt.Errorf("hash chain broken at block %d: previous hash mismatch", i)
			}
		}
	}

	log.Printf("Storage state validated successfully (%d blocks)\n", len(blocks))

	return nil
}

// RecoverOrInitialize attempts to recover state from storage, or initializes genesis if empty
// This is the main entry point for state initialization during node startup
func RecoverOrInitialize(blockRepo ports.BlockRepository, genesisBlock *domain.Block) (*domain.Chain, *domain.State, error) {
	log.Println("Attempting to recover or initialize blockchain state...")

	// Check if storage has any blocks
	blocks, err := blockRepo.GetAll()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to check storage: %w", err)
	}

	if len(blocks) == 0 {
		// Storage is empty - initialize with genesis
		log.Println("Storage is empty, initializing genesis block...")

		if err := InitializeGenesis(blockRepo, genesisBlock); err != nil {
			return nil, nil, fmt.Errorf("failed to initialize genesis: %w", err)
		}

		// Create chain and state with genesis
		chain, err := domain.NewChain(genesisBlock)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create chain: %w", err)
		}

		state, err := domain.NewState(chain)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create state: %w", err)
		}

		log.Println("Genesis block initialized successfully")
		return chain, state, nil
	}

	// Storage has blocks - validate and recover
	log.Printf("Found %d blocks in storage, validating...\n", len(blocks))

	if err := ValidateStorageState(blockRepo); err != nil {
		return nil, nil, fmt.Errorf("storage validation failed: %w", err)
	}

	// Recover state from storage
	chain, state, err := RecoverFromStorage(blockRepo)
	if err != nil {
		return nil, nil, fmt.Errorf("state recovery failed: %w", err)
	}

	log.Printf("State recovered successfully (height: %d, voted count: %d)\n",
		chain.Height(), state.VotedCount())

	return chain, state, nil
}
