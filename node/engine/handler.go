package engine

import (
	"fmt"
	"log"
	"time"
)

// handleVoteReceived processes a VoteReceived event
// Steps:
// 1. Validate the vote (signature, eligibility, no duplicate)
// 2. Add vote to mempool
// 3. Broadcast vote to network (if not already from network)
func (n *Node) handleVoteReceived(e *VoteReceived) error {
	log.Printf("Handling VoteReceived event from %s for voter %s\n", e.From, e.Vote.VoterID())

	// Step 1: Validate vote signature
	// Note: Votes received from the network are already validated during deserialization
	// (NewVote validates the signature), but we verify again for extra safety
	if err := e.Vote.Verify(e.Vote.PublicKey()); err != nil {
		return fmt.Errorf("vote signature validation failed for voter %s: %w", e.Vote.VoterID(), err)
	}

	// Check if voter has already voted (in state or mempool)
	if n.state.HasVoted(e.Vote.VoterID()) {
		return fmt.Errorf("voter %s has already voted (in blockchain)", e.Vote.VoterID())
	}

	if n.mempool.Contains(e.Vote.VoterID()) {
		return fmt.Errorf("voter %s already has a vote in mempool", e.Vote.VoterID())
	}

	// Check eligibility
	eligible, err := n.eligibility.IsEligible(e.Vote.VoterID())
	if err != nil {
		return fmt.Errorf("eligibility check failed: %w", err)
	}
	if !eligible {
		return fmt.Errorf("voter %s is not eligible to vote", e.Vote.VoterID())
	}

	// Step 2: Add to mempool
	if err := n.mempool.Add(e.Vote); err != nil {
		return fmt.Errorf("failed to add vote to mempool: %w", err)
	}

	log.Printf("Vote from %s added to mempool (mempool size: %d)\n", e.Vote.VoterID(), n.mempool.Size())

	// Step 3: Broadcast vote to network (best-effort, don't fail if broadcast fails)
	// Only broadcast if the vote came from local submission, not from network
	if e.From == "local" {
		if err := n.broadcaster.BroadcastVote(e.Vote); err != nil {
			log.Printf("WARNING: Failed to broadcast vote: %v\n", err)
			// Don't return error - vote is already in mempool
		} else {
			log.Printf("Vote from %s broadcast to network\n", e.Vote.VoterID())
		}
	}

	return nil
}

// handleBlockProposal processes a BlockProposalReceived event
// Steps:
// 1. Validate the block proposal (correct leader, valid structure, chain continuity)
// 2. Validate all votes in the block
// 3. Process any buffered approvals for this block
// 4. Record our approval
// 5. Broadcast approval to network
func (n *Node) handleBlockProposal(e *BlockProposalReceived) error {
	log.Printf("Handling BlockProposalReceived event from %s for block at height %d\n", e.From, e.Block.Height())

	// Step 1: Validate the block proposal using consensus engine
	if err := n.consensus.ValidateProposal(e.Block, n.blockchain); err != nil {
		return fmt.Errorf("block proposal validation failed: %w", err)
	}

	log.Printf("Block proposal at height %d validated successfully\n", e.Block.Height())

	// Step 2: Validate all votes in the block
	for i, vote := range e.Block.Votes() {
		// Validate vote signature
		if err := vote.Verify(vote.PublicKey()); err != nil {
			return fmt.Errorf("invalid signature for vote %d from voter %s: %w", i, vote.VoterID(), err)
		}

		// Check if voter has already voted
		if n.state.HasVoted(vote.VoterID()) {
			return fmt.Errorf("block contains vote from voter %s who has already voted", vote.VoterID())
		}

		// Check eligibility
		eligible, err := n.eligibility.IsEligible(vote.VoterID())
		if err != nil {
			return fmt.Errorf("eligibility check failed for vote %d: %w", i, err)
		}
		if !eligible {
			return fmt.Errorf("block contains vote from ineligible voter %s", vote.VoterID())
		}
	}

	log.Printf("All %d votes in block validated successfully\n", len(e.Block.Votes()))

	// Step 3: Process any buffered approvals that arrived before this proposal
	if err := n.processPendingApprovals(e.Block.Hash()); err != nil {
		log.Printf("WARNING: Error processing buffered approvals: %v\n", err)
		// Don't fail - continue with our own approval
	}

	// Step 4: Record our approval in the consensus engine
	if err := n.consensus.RecordApproval(e.Block.Hash(), n.signer.PublicKey()); err != nil {
		return fmt.Errorf("failed to record approval: %w", err)
	}

	log.Printf("Recorded approval for block %s\n", e.Block.Hash().String())

	// Step 5: Broadcast approval to network
	if err := n.broadcaster.BroadcastBlockApproval(e.Block.Hash(), n.signer.PublicKey()); err != nil {
		log.Printf("WARNING: Failed to broadcast block approval: %v\n", err)
		// Don't return error - approval is already recorded locally
	} else {
		log.Printf("Block approval broadcast to network\n")
	}

	// Check if we can finalize the block now
	// (This might happen if we're the last validator to approve)
	return n.checkAndFinalizeBlock()
}

// handleBlockApproval processes a BlockApprovalReceived event
// Steps:
// 1. Try to record the approval in the consensus engine
// 2. If no active round exists, buffer the approval for later processing
// 3. Check if we have enough approvals to finalize
func (n *Node) handleBlockApproval(e *BlockApprovalReceived) error {
	log.Printf("Handling BlockApprovalReceived event from %s\n", e.From)

	// Step 1: Try to record the approval
	if err := n.consensus.RecordApproval(e.BlockHash, e.Validator); err != nil {
		// If there's no active round, buffer the approval for when the proposal arrives
		if err.Error() == "no active round" {
			n.bufferApproval(e.BlockHash, e.Validator)
			return nil
		}

		// If the approval is for a different block, it might be stale
		if err.Error() == "approval for different block: expected" {
			// Silently ignore - this is normal when blocks finalize quickly
			return nil
		}

		return fmt.Errorf("failed to record approval: %w", err)
	}

	log.Printf("Recorded approval from validator for block %s\n", e.BlockHash.String())

	// Step 2: Check if we can finalize
	return n.checkAndFinalizeBlock()
}

// checkAndFinalizeBlock checks if the current block has enough approvals to finalize
// If yes, it finalizes the block and resets the consensus round
func (n *Node) checkAndFinalizeBlock() error {
	// Check if we have enough approvals
	canFinalize, block, err := n.consensus.CheckFinalization()
	if err != nil {
		return fmt.Errorf("failed to check finalization: %w", err)
	}

	if !canFinalize {
		// Not enough approvals yet
		return nil
	}

	log.Printf("Block at height %d has enough approvals, finalizing...\n", block.Height())

	// Finalize the block
	if err := n.finalizer.Finalize(block); err != nil {
		return fmt.Errorf("failed to finalize block: %w", err)
	}

	log.Printf("Block at height %d finalized successfully (chain height: %d, voted count: %d)\n",
		block.Height(), n.blockchain.Height(), n.state.VotedCount())

	// Reset consensus round
	n.consensus.ResetRound()

	return nil
}

// handleBlockInterval is called when the block interval timer elapses
// Steps:
// 1. Check if we're the leader for the next block
// 2. If yes, propose a new block with pending votes
// 3. If mempool is empty, wait for votes up to empty_block_wait_timeout (Tendermint-style)
// 4. Broadcast the block proposal to network
func (n *Node) handleBlockInterval() error {
	log.Printf("Block interval elapsed (current height: %d, mempool size: %d)\n",
		n.blockchain.Height(), n.mempool.Size())

	// Check if we're currently syncing - don't propose if syncing
	if n.syncManager.IsSyncing() {
		log.Println("Node is syncing, skipping block proposal")
		return nil
	}

	// Try to propose a block (will return nil if we're not the leader)
	block, err := n.consensus.ProposeBlock(n.mempool, n.blockchain, n.signer)
	if err != nil {
		return fmt.Errorf("failed to propose block: %w", err)
	}

	if block == nil {
		// We're not the leader for this round
		log.Println("Not the leader for this round, skipping block proposal")
		return nil
	}

	// Tendermint-style: If block is empty, wait for votes before proposing
	if len(block.Votes()) == 0 {
		log.Println("Mempool empty, waiting for votes to arrive...")

		// Get timeout from config (default 1 second if not set)
		waitTimeout := n.config.EmptyBlockWaitTimeout.Duration
		if waitTimeout == 0 {
			waitTimeout = 1 * time.Second
		}

		deadline := n.clock.Now().Add(waitTimeout)
		checkInterval := 100 * time.Millisecond

		for n.clock.Now().Before(deadline) {
			time.Sleep(checkInterval)

			if n.mempool.Size() > 0 {
				log.Printf("Votes arrived! Mempool now has %d votes, rebuilding block...\n", n.mempool.Size())

				// Rebuild block with new votes
				block, err = n.consensus.ProposeBlock(n.mempool, n.blockchain, n.signer)
				if err != nil {
					return fmt.Errorf("failed to rebuild block: %w", err)
				}
				break
			}
		}

		if len(block.Votes()) == 0 {
			log.Printf("Wait timeout expired (%v), proposing empty block to maintain liveness\n", waitTimeout)
		}
	}

	log.Printf("Proposed new block at height %d with %d votes\n", block.Height(), len(block.Votes()))

	// Process our own proposal locally to advance the state machine
	// This ensures the state transitions from Proposing -> Validating
	if err := n.consensus.ValidateProposal(block, n.blockchain); err != nil {
		return fmt.Errorf("failed to validate own proposal: %w", err)
	}

	log.Printf("Validated own block proposal at height %d\n", block.Height())

	// Broadcast the block proposal to network
	if err := n.broadcaster.BroadcastBlockProposal(block); err != nil {
		log.Printf("WARNING: Failed to broadcast block proposal: %v\n", err)
		// Don't return error - we can still process the block locally
	} else {
		log.Printf("Block proposal broadcast to network\n")
	}

	// As the proposer, we automatically approve our own block
	if err := n.consensus.RecordApproval(block.Hash(), n.signer.PublicKey()); err != nil {
		return fmt.Errorf("failed to record self-approval: %w", err)
	}

	log.Printf("Recorded self-approval for proposed block\n")

	// Broadcast our approval
	if err := n.broadcaster.BroadcastBlockApproval(block.Hash(), n.signer.PublicKey()); err != nil {
		log.Printf("WARNING: Failed to broadcast self-approval: %v\n", err)
	}

	// Check if we can finalize (might happen if we're the only validator)
	return n.checkAndFinalizeBlock()
}

// handleSyncRequest processes a SyncRequested event
// This initiates state synchronization with network peers
func (n *Node) handleSyncRequest(e *SyncRequested) error {
	log.Printf("Handling SyncRequested event from height %d to %d\n", e.FromHeight, e.ToHeight)

	// For now, we'll implement a simple sync that just logs the request
	// Full implementation would require requesting blocks from peers
	// This is a placeholder for the sync functionality

	currentHeight := n.blockchain.Height()
	if e.ToHeight <= currentHeight {
		log.Printf("Already synced to height %d (requested: %d)\n", currentHeight, e.ToHeight)
		return nil
	}

	log.Printf("Sync requested but not yet implemented (current: %d, target: %d)\n", currentHeight, e.ToHeight)

	// TODO: Implement full sync logic using syncManager
	// This would involve:
	// 1. Request blocks from peers for heights fromHeight to toHeight
	// 2. Validate each block
	// 3. Apply blocks to chain and state
	// 4. Persist blocks to storage

	return nil
}
