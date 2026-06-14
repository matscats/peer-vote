package engine

import (
	blockchainDomain "peer-vote/blockchain/domain"
	"peer-vote/crypto"
	votingDomain "peer-vote/voting/domain"
)

// Event represents an event that the node engine processes
// Events drive the node's state machine and trigger actions
type Event interface {
	Type() EventType
}

// EventType identifies the type of event
type EventType int

const (
	EventVoteReceived EventType = iota
	EventBlockProposalReceived
	EventBlockApprovalReceived
	EventSyncRequested
	EventSyncResponseReceived
)

// String returns the string representation of the event type
func (et EventType) String() string {
	switch et {
	case EventVoteReceived:
		return "VoteReceived"
	case EventBlockProposalReceived:
		return "BlockProposalReceived"
	case EventBlockApprovalReceived:
		return "BlockApprovalReceived"
	case EventSyncRequested:
		return "SyncRequested"
	case EventSyncResponseReceived:
		return "SyncResponseReceived"
	default:
		return "Unknown"
	}
}

// VoteReceived event is triggered when a vote is received from the network
type VoteReceived struct {
	Vote *votingDomain.Vote
	From string // Peer ID
}

// Type returns the event type
func (e *VoteReceived) Type() EventType {
	return EventVoteReceived
}

// BlockProposalReceived event is triggered when a block proposal is received from the network
type BlockProposalReceived struct {
	Block *blockchainDomain.Block
	From  string // Peer ID
}

// Type returns the event type
func (e *BlockProposalReceived) Type() EventType {
	return EventBlockProposalReceived
}

// BlockApprovalReceived event is triggered when a block approval is received from the network
type BlockApprovalReceived struct {
	BlockHash crypto.Hash
	Validator crypto.PublicKey
	From      string // Peer ID
}

// Type returns the event type
func (e *BlockApprovalReceived) Type() EventType {
	return EventBlockApprovalReceived
}

// SyncRequested event is triggered when state synchronization is needed
type SyncRequested struct {
	FromHeight uint64
	ToHeight   uint64
}

// Type returns the event type
func (e *SyncRequested) Type() EventType {
	return EventSyncRequested
}

// SyncResponseReceived event is triggered when blocks are received for state synchronization
type SyncResponseReceived struct {
	Blocks []*blockchainDomain.Block
	From   string
}

// Type returns the event type
func (e *SyncResponseReceived) Type() EventType {
	return EventSyncResponseReceived
}
