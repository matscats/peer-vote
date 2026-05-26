package domain

import (
	"encoding/hex"
	"fmt"
	"time"

	"peer-vote/crypto"
	votingdomain "peer-vote/voting/domain"
)

// GenesisConfig holds the configuration for creating a genesis block
type GenesisConfig struct {
	Timestamp      time.Time        // Fixed timestamp for deterministic genesis
	ProposerPubKey crypto.PublicKey // Public key of the genesis proposer
	ExpectedHash   string           // Expected hash of the genesis block (hex string)
}

// DefaultGenesisConfig returns the default genesis configuration for the network
// This should be the same across all nodes in the network
func DefaultGenesisConfig() *GenesisConfig {
	// Parse the genesis proposer public key (validator1's public key)
	pubKeyBytes, _ := hex.DecodeString("383eb67b43343e3fc1bcb711ce84fd9c13ef2f15c4025830dbc2a75666674503")

	return &GenesisConfig{
		Timestamp:      time.Unix(0, 0).UTC(), // Fixed epoch timestamp
		ProposerPubKey: pubKeyBytes,
		ExpectedHash:   "", // Will be set after first genesis creation
	}
}

// CreateGenesisBlock creates the genesis block (block 0) with no votes and null previous hash
// The genesis block is signed by the designated genesis validator to ensure authenticity
// All nodes must have the same genesis block hash for network consistency
func CreateGenesisBlock(genesisSigner crypto.Signer) (*Block, error) {
	if genesisSigner == nil {
		return nil, fmt.Errorf("genesis signer cannot be nil")
	}

	// Create genesis block with:
	// - height: 0
	// - previousHash: null (zero hash)
	// - votes: empty slice (no votes in genesis)
	// - proposer: genesis validator's public key
	nullHash := crypto.Hash{}

	// Create block manually with fixed timestamp for deterministic genesis
	genesis := &Block{
		height:       0,
		previousHash: nullHash,
		timestamp:    time.Unix(0, 0).UTC(), // Fixed epoch timestamp for deterministic genesis
		votes:        []*votingdomain.Vote{},
		proposer:     genesisSigner.PublicKey(),
	}

	// Sign the genesis block
	if err := genesis.Sign(genesisSigner); err != nil {
		return nil, fmt.Errorf("failed to sign genesis block: %w", err)
	}

	return genesis, nil
}

// CreateGenesisFromConfig creates a genesis block from configuration without requiring a signer
// This allows nodes to create the same genesis block without sharing private keys
// The block is created unsigned, which is acceptable for genesis as it's verified by hash
func CreateGenesisFromConfig(config *GenesisConfig) (*Block, error) {
	if config == nil {
		return nil, fmt.Errorf("genesis config cannot be nil")
	}

	nullHash := crypto.Hash{}

	// Create genesis block with deterministic parameters
	genesis := &Block{
		height:       0,
		previousHash: nullHash,
		timestamp:    config.Timestamp,
		votes:        []*votingdomain.Vote{},
		proposer:     config.ProposerPubKey,
		signature:    crypto.Signature{}, // Genesis block can be unsigned
	}

	// Compute hash
	genesis.hash = genesis.ComputeHash()

	// Verify against expected hash if provided
	if config.ExpectedHash != "" {
		expectedHashBytes, err := hex.DecodeString(config.ExpectedHash)
		if err != nil {
			return nil, fmt.Errorf("invalid expected hash: %w", err)
		}
		var expectedHash crypto.Hash
		copy(expectedHash[:], expectedHashBytes)

		if genesis.hash != expectedHash {
			return nil, fmt.Errorf("genesis hash mismatch: got %s, expected %s",
				genesis.hash.String(), expectedHash.String())
		}
	}

	return genesis, nil
}

// VerifyGenesisBlock verifies that a block is a valid genesis block
func VerifyGenesisBlock(block *Block) error {
	if block == nil {
		return fmt.Errorf("genesis block cannot be nil")
	}

	// Verify height is 0
	if block.Height() != 0 {
		return fmt.Errorf("genesis block must have height 0, got %d", block.Height())
	}

	// Verify previous hash is null
	nullHash := crypto.Hash{}
	if block.PreviousHash() != nullHash {
		return fmt.Errorf("genesis block must have null previous hash")
	}

	// Verify hash is correct (genesis may not have signature)
	computedHash := block.ComputeHash()
	if block.Hash() != computedHash {
		return fmt.Errorf("genesis block hash mismatch")
	}

	return nil
}

// VerifyGenesisHash checks if the given block matches the expected genesis hash
// This is used to ensure all nodes in the network start with the same genesis block
func VerifyGenesisHash(block *Block, expectedHash crypto.Hash) error {
	if block == nil {
		return fmt.Errorf("genesis block cannot be nil")
	}

	if block.Hash() != expectedHash {
		return fmt.Errorf("genesis hash mismatch: got %s, expected %s",
			block.Hash().String(), expectedHash.String())
	}

	return nil
}

// IsGenesisHash checks if the given hash matches the expected genesis hash
// This is used to ensure all nodes in the network start with the same genesis block
func IsGenesisHash(block *Block, expectedHash crypto.Hash) bool {
	if block == nil {
		return false
	}
	return block.Hash() == expectedHash
}
