package engine

import (
	"fmt"

	"peer-vote/blockchain/domain"
	"peer-vote/crypto"
	voteDomain "peer-vote/voting/domain"
)

// BlockBuilder is responsible for constructing and signing new blocks
// This engine coordinates block creation with the proposer's cryptographic key
type BlockBuilder struct {
	signer crypto.Signer
}

// NewBlockBuilder creates a new BlockBuilder with the given signer
func NewBlockBuilder(signer crypto.Signer) *BlockBuilder {
	return &BlockBuilder{
		signer: signer,
	}
}

// Build constructs and signs a new block with the given parameters
// The block is created with the proposer's public key and signed with their private key
// Returns error if block creation or signing fails
func (b *BlockBuilder) Build(height uint64, prevHash crypto.Hash, votes []*voteDomain.Vote) (*domain.Block, error) {
	// Create the block with the proposer's public key
	block, err := domain.NewBlock(height, prevHash, votes, b.signer.PublicKey())
	if err != nil {
		return nil, fmt.Errorf("failed to create block: %w", err)
	}

	// Sign the block with the proposer's private key
	if err := block.Sign(b.signer); err != nil {
		return nil, fmt.Errorf("failed to sign block: %w", err)
	}

	return block, nil
}
