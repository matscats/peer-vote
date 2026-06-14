package engine

import (
	"fmt"
	"log"
	"sync"
	"time"

	"peer-vote/blockchain/domain"
	"peer-vote/blockchain/ports"
	consensusEngine "peer-vote/consensus/engine"
	"peer-vote/crypto"
	mempoolDomain "peer-vote/mempool/domain"
	networkAdapters "peer-vote/network/adapters"
	"peer-vote/network/engine"
	networkPorts "peer-vote/network/ports"
	"peer-vote/node/config"
	nodePorts "peer-vote/node/ports"
	votingDomain "peer-vote/voting/domain"
	votingEngine "peer-vote/voting/engine"
	votingPorts "peer-vote/voting/ports"
)

// Approval represents a buffered block approval
type Approval struct {
	BlockHash crypto.Hash
	Validator crypto.PublicKey
	Timestamp time.Time
}

// Node is the central orchestrator that runs the event-driven blockchain node
// It maintains the main event loop and coordinates all system components
type Node struct {
	// State
	blockchain *domain.Chain
	state      *domain.State
	mempool    *mempoolDomain.Mempool

	// Engines
	consensus     *consensusEngine.PoAEngine
	syncManager   *engine.SyncManager
	voteValidator *votingEngine.VoteValidator
	finalizer     *consensusEngine.Finalizer

	// Ports
	blockRepo   ports.BlockRepository
	broadcaster networkPorts.Broadcaster
	eligibility votingPorts.EligibilityChecker
	clock       nodePorts.Clock

	// Event channels
	events   chan Event
	shutdown chan struct{}

	// Configuration
	config *config.Config
	signer crypto.Signer

	// Timing
	blockTimeout time.Duration // Timeout for block proposal

	// Pending approvals buffer (for approvals that arrive before proposals)
	pendingApprovals map[string][]Approval // Map of block hash (hex) -> list of approvals
	approvalsMu      sync.RWMutex          // Protects pendingApprovals

	// Pending block proposals received while this node is behind
	pendingBlockProposals map[uint64]*domain.Block
	proposalsMu           sync.RWMutex

	// Synchronization
	mu      sync.RWMutex
	running bool
}

// NewNode creates a new Node with the given configuration and components
func NewNode(
	cfg *config.Config,
	signer crypto.Signer,
	blockchain *domain.Chain,
	state *domain.State,
	mempool *mempoolDomain.Mempool,
	consensus *consensusEngine.PoAEngine,
	syncManager *engine.SyncManager,
	voteValidator *votingEngine.VoteValidator,
	finalizer *consensusEngine.Finalizer,
	blockRepo ports.BlockRepository,
	broadcaster networkPorts.Broadcaster,
	eligibility votingPorts.EligibilityChecker,
	clock nodePorts.Clock,
) (*Node, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}
	if signer == nil {
		return nil, fmt.Errorf("signer cannot be nil")
	}
	if blockchain == nil {
		return nil, fmt.Errorf("blockchain cannot be nil")
	}
	if state == nil {
		return nil, fmt.Errorf("state cannot be nil")
	}
	if mempool == nil {
		return nil, fmt.Errorf("mempool cannot be nil")
	}
	if consensus == nil {
		return nil, fmt.Errorf("consensus cannot be nil")
	}
	if syncManager == nil {
		return nil, fmt.Errorf("syncManager cannot be nil")
	}
	if voteValidator == nil {
		return nil, fmt.Errorf("voteValidator cannot be nil")
	}
	if finalizer == nil {
		return nil, fmt.Errorf("finalizer cannot be nil")
	}
	if blockRepo == nil {
		return nil, fmt.Errorf("blockRepo cannot be nil")
	}
	if broadcaster == nil {
		return nil, fmt.Errorf("broadcaster cannot be nil")
	}
	if eligibility == nil {
		return nil, fmt.Errorf("eligibility cannot be nil")
	}
	if clock == nil {
		// Default to real clock if not provided
		clock = nodePorts.NewRealClock()
	}

	// Calculate block timeout (2x block interval for leader failure detection)
	blockTimeout := cfg.BlockInterval.Duration * 2

	return &Node{
		blockchain:            blockchain,
		state:                 state,
		mempool:               mempool,
		consensus:             consensus,
		syncManager:           syncManager,
		voteValidator:         voteValidator,
		finalizer:             finalizer,
		blockRepo:             blockRepo,
		broadcaster:           broadcaster,
		eligibility:           eligibility,
		clock:                 clock,
		events:                make(chan Event, 100), // Buffered channel for events
		shutdown:              make(chan struct{}),
		config:                cfg,
		signer:                signer,
		blockTimeout:          blockTimeout,
		pendingApprovals:      make(map[string][]Approval),
		pendingBlockProposals: make(map[uint64]*domain.Block),
		running:               false,
	}, nil
}

// Start initializes the node and begins the event loop
// This method:
// 1. Initializes genesis block if needed
// 2. Starts the P2P network
// 3. Subscribes to network events
// 4. Starts the event loop in a goroutine
func (n *Node) Start() error {
	n.mu.Lock()
	if n.running {
		n.mu.Unlock()
		return fmt.Errorf("node is already running")
	}
	n.running = true
	n.mu.Unlock()

	log.Println("Starting node...")

	// Initialize genesis block if blockchain is empty (height 0 only)
	if n.blockchain.Height() == 0 {
		log.Println("Genesis block already initialized at height 0")
	}

	// Start P2P network
	log.Println("Starting P2P network...")
	if err := n.broadcaster.Start(); err != nil {
		return fmt.Errorf("failed to start P2P network: %w", err)
	}

	// Subscribe to network events
	log.Println("Subscribing to network events...")
	if err := n.broadcaster.Subscribe(n); err != nil {
		return fmt.Errorf("failed to subscribe to network events: %w", err)
	}

	// Start event loop
	log.Println("Starting event loop...")
	go n.eventLoop()
	go n.requestInitialSync()

	log.Printf("Node started successfully at height %d\n", n.blockchain.Height())
	return nil
}

func (n *Node) requestInitialSync() {
	// Give GossipSub subscriptions and peer discovery a short window before asking
	// for missing blocks. Peers that are not ahead will simply ignore the request.
	time.Sleep(2 * time.Second)

	currentHeight := n.blockchain.Height()
	const syncLookahead uint64 = 1024
	if err := n.requestSync(currentHeight+1, currentHeight+syncLookahead); err != nil {
		log.Printf("WARNING: initial sync request failed: %v\n", err)
	}
}

// eventLoop is the main event processing loop
// This is the heart of the node - it reacts to events from multiple sources:
// - Events from the event channel (votes, block proposals, approvals)
// - Block interval ticks (time to propose a new block)
// - Block timeout (advance round if leader fails to propose)
// - Shutdown signal
func (n *Node) eventLoop() {
	// Use clock port for testable time control
	ticker := n.clock.NewTicker(n.config.BlockInterval.Duration)
	defer ticker.Stop()

	// Track last block time for timeout detection
	lastBlockTime := n.clock.Now()
	timeoutTicker := n.clock.NewTicker(n.blockTimeout)
	defer timeoutTicker.Stop()

	log.Printf("Event loop started with block interval: %v, timeout: %v\n", n.config.BlockInterval.Duration, n.blockTimeout)

	// Log connected peers periodically for debugging
	peerLogTicker := n.clock.NewTicker(10 * time.Second)
	defer peerLogTicker.Stop()

	// Clean up old pending approvals periodically (every 30 seconds)
	cleanupTicker := n.clock.NewTicker(30 * time.Second)
	defer cleanupTicker.Stop()

	for {
		select {
		case event := <-n.events:
			// Handle incoming event
			if err := n.handleEvent(event); err != nil {
				log.Printf("ERROR: Event handling failed for %s: %v\n", event.Type(), err)
				// Continue processing other events despite error
			}

		case <-ticker.C():
			// Block interval elapsed - time to propose a block if we're the leader
			if err := n.handleBlockInterval(); err != nil {
				log.Printf("ERROR: Block interval handling failed: %v\n", err)
				// Continue to next interval despite error
			}
			// Reset last block time
			lastBlockTime = n.clock.Now()
			// Reset timeout ticker
			timeoutTicker.Reset(n.blockTimeout)

		case <-peerLogTicker.C():
			// Log connected peers for debugging
			log.Printf("DEBUG: Connected peers: %d, Consensus state: %s\n",
				len(n.broadcaster.(*networkAdapters.P2PNetwork).GetConnectedPeers()),
				n.consensus.CurrentState())

		case <-cleanupTicker.C():
			// Clean up old pending approvals (older than 60 seconds)
			n.cleanupOldApprovals(60 * time.Second)

		case <-timeoutTicker.C():
			// Block timeout elapsed - leader may have failed to propose
			elapsed := n.clock.Now().Sub(lastBlockTime)
			if elapsed >= n.blockTimeout {
				log.Printf("WARNING: Block timeout elapsed (%v), advancing round\n", elapsed)
				// Reset consensus round to allow next leader to propose
				n.consensus.ResetRound()
				// Reset timeout ticker
				timeoutTicker.Reset(n.blockTimeout)
			}

		case <-n.shutdown:
			log.Println("Shutdown signal received, stopping event loop...")
			return
		}
	}
}

// handleEvent dispatches events to their appropriate handlers
func (n *Node) handleEvent(event Event) error {
	switch e := event.(type) {
	case *VoteReceived:
		return n.handleVoteReceived(e)
	case *BlockProposalReceived:
		return n.handleBlockProposal(e)
	case *BlockApprovalReceived:
		return n.handleBlockApproval(e)
	case *SyncRequested:
		return n.handleSyncRequest(e)
	case *SyncResponseReceived:
		return n.handleSyncResponse(e)
	default:
		return fmt.Errorf("unknown event type: %v", event.Type())
	}
}

// Shutdown gracefully shuts down the node
// This method:
// 1. Signals the event loop to stop
// 2. Drains the event queue
// 3. Stops the P2P network
// 4. Waits for cleanup to complete
func (n *Node) Shutdown() error {
	n.mu.Lock()
	if !n.running {
		n.mu.Unlock()
		return fmt.Errorf("node is not running")
	}
	n.running = false
	n.mu.Unlock()

	log.Println("Shutting down node...")

	// Signal event loop to stop
	close(n.shutdown)

	// Give event loop time to finish current operation and drain queue
	time.Sleep(500 * time.Millisecond)

	// Drain any remaining events in the queue
	remainingEvents := len(n.events)
	if remainingEvents > 0 {
		log.Printf("WARNING: Draining %d remaining events from queue\n", remainingEvents)
		for i := 0; i < remainingEvents; i++ {
			select {
			case <-n.events:
				// Discard event
			default:
				// Channel is empty, exit loop
				goto drainComplete
			}
		}
	}
drainComplete:

	// Stop P2P network
	if err := n.broadcaster.Stop(); err != nil {
		log.Printf("WARNING: Failed to stop P2P network: %v\n", err)
		// Continue with shutdown despite error
	}

	log.Println("Node shutdown complete")
	return nil
}

// GetState returns the current blockchain state
func (n *Node) GetState() *domain.State {
	return n.state
}

// GetBlockchain returns the blockchain
func (n *Node) GetBlockchain() *domain.Chain {
	return n.blockchain
}

// GetMempool returns the mempool
func (n *Node) GetMempool() *mempoolDomain.Mempool {
	return n.mempool
}

// IsRunning returns true if the node is currently running
func (n *Node) IsRunning() bool {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.running
}

// SubmitVote allows external submission of a vote (e.g., from API)
// This creates a VoteReceived event and adds it to the event queue
func (n *Node) SubmitVote(vote *votingDomain.Vote) error {
	if !n.IsRunning() {
		return fmt.Errorf("node is not running")
	}

	// Create event and add to queue
	event := &VoteReceived{
		Vote: vote,
		From: "local", // Indicates vote was submitted locally
	}

	select {
	case n.events <- event:
		return nil
	default:
		return fmt.Errorf("event queue is full")
	}
}

// Network event handlers (implements networkPorts.MessageHandler interface)

// HandleVote is called when a vote is received from the network
func (n *Node) HandleVote(vote *votingDomain.Vote, from string) error {
	// Create event and add to queue
	event := &VoteReceived{
		Vote: vote,
		From: from,
	}

	select {
	case n.events <- event:
		return nil
	default:
		return fmt.Errorf("event queue is full")
	}
}

// HandleBlockProposal is called when a block proposal is received from the network
func (n *Node) HandleBlockProposal(block *domain.Block, from string) error {
	// Create event and add to queue
	event := &BlockProposalReceived{
		Block: block,
		From:  from,
	}

	select {
	case n.events <- event:
		return nil
	default:
		return fmt.Errorf("event queue is full")
	}
}

// HandleBlockApproval is called when a block approval is received from the network
func (n *Node) HandleBlockApproval(blockHash crypto.Hash, validator crypto.PublicKey, from string) error {
	// Create event and add to queue
	event := &BlockApprovalReceived{
		BlockHash: blockHash,
		Validator: validator,
		From:      from,
	}

	select {
	case n.events <- event:
		return nil
	default:
		return fmt.Errorf("event queue is full")
	}
}

// HandleSyncRequest is called when a sync request is received from the network
func (n *Node) HandleSyncRequest(fromHeight, toHeight uint64, from string) error {
	event := &SyncRequested{
		FromHeight: fromHeight,
		ToHeight:   toHeight,
	}

	select {
	case n.events <- event:
		return nil
	default:
		return fmt.Errorf("event queue is full")
	}
}

// HandleSyncResponse is called when a sync response is received from the network
func (n *Node) HandleSyncResponse(blocks []*domain.Block, from string) error {
	event := &SyncResponseReceived{
		Blocks: blocks,
		From:   from,
	}

	select {
	case n.events <- event:
		return nil
	default:
		return fmt.Errorf("event queue is full")
	}
}

// cleanupOldApprovals removes pending approvals older than the specified duration
// This prevents memory leaks from approvals that will never be processed
func (n *Node) cleanupOldApprovals(maxAge time.Duration) {
	n.approvalsMu.Lock()
	defer n.approvalsMu.Unlock()

	now := n.clock.Now()
	removed := 0

	for blockHash, approvals := range n.pendingApprovals {
		// Filter out old approvals
		validApprovals := make([]Approval, 0, len(approvals))
		for _, approval := range approvals {
			if now.Sub(approval.Timestamp) < maxAge {
				validApprovals = append(validApprovals, approval)
			} else {
				removed++
			}
		}

		// Update or delete the entry
		if len(validApprovals) > 0 {
			n.pendingApprovals[blockHash] = validApprovals
		} else {
			delete(n.pendingApprovals, blockHash)
		}
	}

	if removed > 0 {
		log.Printf("Cleaned up %d old pending approvals (total pending: %d)\n", removed, n.countPendingApprovals())
	}
}

// countPendingApprovals returns the total number of pending approvals
// Must be called with approvalsMu held
func (n *Node) countPendingApprovals() int {
	count := 0
	for _, approvals := range n.pendingApprovals {
		count += len(approvals)
	}
	return count
}

// bufferApproval stores an approval for later processing when the proposal arrives
func (n *Node) bufferApproval(blockHash crypto.Hash, validator crypto.PublicKey) {
	n.approvalsMu.Lock()
	defer n.approvalsMu.Unlock()

	hashStr := fmt.Sprintf("%x", blockHash)
	approval := Approval{
		BlockHash: blockHash,
		Validator: validator,
		Timestamp: n.clock.Now(),
	}

	n.pendingApprovals[hashStr] = append(n.pendingApprovals[hashStr], approval)
	log.Printf("Buffered approval for block %s (total buffered for this block: %d)\n",
		blockHash.String(), len(n.pendingApprovals[hashStr]))
}

// processPendingApprovals processes any buffered approvals for the given block hash
func (n *Node) processPendingApprovals(blockHash crypto.Hash) error {
	n.approvalsMu.Lock()
	hashStr := fmt.Sprintf("%x", blockHash)
	approvals := n.pendingApprovals[hashStr]
	delete(n.pendingApprovals, hashStr) // Remove from buffer
	n.approvalsMu.Unlock()

	if len(approvals) == 0 {
		return nil
	}

	log.Printf("Processing %d buffered approvals for block %s\n", len(approvals), blockHash.String())

	// Process each buffered approval
	for _, approval := range approvals {
		if err := n.consensus.RecordApproval(approval.BlockHash, approval.Validator); err != nil {
			// Log error but continue processing other approvals
			log.Printf("WARNING: Failed to process buffered approval: %v\n", err)
			continue
		}
		log.Printf("Processed buffered approval from validator for block %s\n", blockHash.String())
	}

	return nil
}
