package ports

import (
	"peer-vote/blockchain/domain"
	"peer-vote/crypto"
	votingdomain "peer-vote/voting/domain"
)

// Broadcaster defines the port for broadcasting messages across the P2P network
// This interface abstracts the network layer from the domain logic
type Broadcaster interface {
	// BroadcastVote broadcasts a vote to all connected peers
	BroadcastVote(vote *votingdomain.Vote) error

	// BroadcastBlockProposal broadcasts a block proposal to all validators
	BroadcastBlockProposal(block *domain.Block) error

	// BroadcastBlockApproval broadcasts a block approval to all validators
	BroadcastBlockApproval(blockHash crypto.Hash, validator crypto.PublicKey) error

	// Subscribe registers a message handler to receive network events
	Subscribe(handler MessageHandler) error

	// Start initializes and starts the network broadcaster
	Start() error

	// Stop gracefully shuts down the network broadcaster
	Stop() error
}

// MessageHandler defines callbacks for handling incoming network messages
// This interface allows the node engine to react to network events
type MessageHandler interface {
	// HandleVote is called when a vote is received from the network
	HandleVote(vote *votingdomain.Vote, from string) error

	// HandleBlockProposal is called when a block proposal is received
	HandleBlockProposal(block *domain.Block, from string) error

	// HandleBlockApproval is called when a block approval is received
	HandleBlockApproval(blockHash crypto.Hash, validator crypto.PublicKey, from string) error
}
