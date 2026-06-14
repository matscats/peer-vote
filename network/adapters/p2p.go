package adapters

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery/routing"
	"github.com/libp2p/go-libp2p/p2p/discovery/util"
	"github.com/multiformats/go-multiaddr"

	"peer-vote/blockchain/domain"
	"peer-vote/crypto"
	networkdomain "peer-vote/network/domain"
	"peer-vote/network/ports"
	votingdomain "peer-vote/voting/domain"
)

const (
	// Topic names for pubsub
	topicVotes          = "peer-vote/votes"
	topicBlockProposals = "peer-vote/block-proposals"
	topicBlockApprovals = "peer-vote/block-approvals"
	topicSyncRequests   = "peer-vote/sync-requests"
	topicSyncResponses  = "peer-vote/sync-responses"
)

// P2PNetwork implements the Broadcaster port using libp2p
type P2PNetwork struct {
	ctx         context.Context
	cancel      context.CancelFunc
	host        host.Host
	dht         *dht.IpfsDHT
	routingDisc *routing.RoutingDiscovery
	pubsub      *pubsub.PubSub
	topics      map[string]*pubsub.Topic
	subs        map[string]*pubsub.Subscription
	handlers    []ports.MessageHandler
	mu          sync.RWMutex
	started     bool
}

// NewP2PNetwork creates a new P2P network adapter
// port: the port to listen on (e.g., 4001)
// bootstrapPeers: list of bootstrap peer addresses to connect to
func NewP2PNetwork(port int, bootstrapPeers []string) (*P2PNetwork, error) {
	ctx, cancel := context.WithCancel(context.Background())

	// Create libp2p host
	listenAddr := fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", port)

	h, err := libp2p.New(
		libp2p.ListenAddrStrings(listenAddr),
		libp2p.EnableNATService(),
	)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create libp2p host: %w", err)
	}

	log.Printf("P2P Host created with ID: %s\n", h.ID())
	log.Printf("Listening on: %v\n", h.Addrs())

	// Create DHT
	kadDHT, err := dht.New(ctx, h, dht.Mode(dht.ModeServer))
	if err != nil {
		h.Close()
		cancel()
		return nil, fmt.Errorf("failed to create DHT: %w", err)
	}

	// Bootstrap the DHT
	if err = kadDHT.Bootstrap(ctx); err != nil {
		h.Close()
		cancel()
		return nil, fmt.Errorf("failed to bootstrap DHT: %w", err)
	}

	log.Println("DHT bootstrapped successfully")

	// Create routing discovery
	routingDisc := routing.NewRoutingDiscovery(kadDHT)

	// Create pubsub
	ps, err := pubsub.NewGossipSub(ctx, h)
	if err != nil {
		h.Close()
		cancel()
		return nil, fmt.Errorf("failed to create pubsub: %w", err)
	}

	network := &P2PNetwork{
		ctx:         ctx,
		cancel:      cancel,
		host:        h,
		dht:         kadDHT,
		routingDisc: routingDisc,
		pubsub:      ps,
		topics:      make(map[string]*pubsub.Topic),
		subs:        make(map[string]*pubsub.Subscription),
		handlers:    make([]ports.MessageHandler, 0),
		started:     false,
	}

	// Connect to bootstrap peers
	if err := network.connectToBootstrapPeers(bootstrapPeers); err != nil {
		log.Printf("Warning: failed to connect to some bootstrap peers: %v\n", err)
		// Don't fail - DHT can still work with peer discovery
	}

	// Start peer discovery
	go network.discoverPeers()

	return network, nil
}

// connectToBootstrapPeers connects to the provided bootstrap peers
func (n *P2PNetwork) connectToBootstrapPeers(bootstrapPeers []string) error {
	connectedCount := 0

	for _, peerAddr := range bootstrapPeers {
		if peerAddr == "" {
			continue
		}

		// Parse multiaddr
		maddr, err := multiaddr.NewMultiaddr(peerAddr)
		if err != nil {
			log.Printf("Warning: invalid bootstrap peer address %s: %v\n", peerAddr, err)
			continue
		}

		// Try to extract peer info
		peerInfo, err := peer.AddrInfoFromP2pAddr(maddr)
		if err != nil {
			// If no peer ID, we can't connect directly but DHT will help discover
			log.Printf("Warning: bootstrap peer %s does not include peer ID, will rely on DHT discovery\n", peerAddr)
			continue
		}

		// Connect to peer
		if err := n.host.Connect(n.ctx, *peerInfo); err != nil {
			log.Printf("Warning: failed to connect to bootstrap peer %s: %v\n", peerAddr, err)
		} else {
			log.Printf("Connected to bootstrap peer: %s\n", peerInfo.ID)
			connectedCount++
		}
	}

	if connectedCount > 0 {
		log.Printf("Successfully connected to %d bootstrap peer(s)\n", connectedCount)
	}

	return nil
}

// discoverPeers continuously discovers and connects to peers using DHT
func (n *P2PNetwork) discoverPeers() {
	const rendezvousString = "peer-vote-network"

	log.Printf("Starting peer discovery with rendezvous: %s\n", rendezvousString)

	// Announce ourselves
	util.Advertise(n.ctx, n.routingDisc, rendezvousString)
	log.Println("Announced presence to DHT")

	// Look for peers continuously
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-n.ctx.Done():
			return
		case <-ticker.C:
			log.Println("Searching for peers via DHT...")

			peerChan, err := n.routingDisc.FindPeers(n.ctx, rendezvousString)
			if err != nil {
				log.Printf("Error finding peers: %v\n", err)
				continue
			}

			peersFound := 0
			for peer := range peerChan {
				if peer.ID == n.host.ID() {
					continue // Skip ourselves
				}

				// Check if already connected
				if n.host.Network().Connectedness(peer.ID) == 1 { // Connected
					continue
				}

				log.Printf("DHT discovered peer: %s\n", peer.ID)
				peersFound++

				// Try to connect
				ctx, cancel := context.WithTimeout(n.ctx, 10*time.Second)
				if err := n.host.Connect(ctx, peer); err != nil {
					log.Printf("Failed to connect to peer %s: %v\n", peer.ID, err)
				} else {
					log.Printf("Successfully connected to peer %s via DHT\n", peer.ID)
				}
				cancel()
			}

			connectedPeers := len(n.host.Network().Peers())
			log.Printf("DHT discovery round complete. Found: %d new peers, Total connected: %d\n",
				peersFound, connectedPeers)
		}
	}
}

// Start initializes topics and starts listening for messages
func (n *P2PNetwork) Start() error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.started {
		return fmt.Errorf("network already started")
	}

	// Join topics
	topicNames := []string{
		topicVotes,
		topicBlockProposals,
		topicBlockApprovals,
		topicSyncRequests,
		topicSyncResponses,
	}

	for _, topicName := range topicNames {
		topic, err := n.pubsub.Join(topicName)
		if err != nil {
			return fmt.Errorf("failed to join topic %s: %w", topicName, err)
		}
		n.topics[topicName] = topic

		// Subscribe to topic
		sub, err := topic.Subscribe()
		if err != nil {
			return fmt.Errorf("failed to subscribe to topic %s: %w", topicName, err)
		}
		n.subs[topicName] = sub

		// Start message handler for this topic
		go n.handleTopicMessages(topicName, sub)
	}

	n.started = true
	return nil
}

// Stop gracefully shuts down the network
func (n *P2PNetwork) Stop() error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if !n.started {
		return nil
	}

	// Cancel subscriptions
	for _, sub := range n.subs {
		sub.Cancel()
	}

	// Close topics
	for _, topic := range n.topics {
		if err := topic.Close(); err != nil {
			fmt.Printf("Warning: failed to close topic: %v\n", err)
		}
	}

	// Close host
	if err := n.host.Close(); err != nil {
		return fmt.Errorf("failed to close host: %w", err)
	}

	// Cancel context
	n.cancel()

	n.started = false
	return nil
}

// Subscribe registers a message handler
func (n *P2PNetwork) Subscribe(handler ports.MessageHandler) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.handlers = append(n.handlers, handler)
	return nil
}

// BroadcastVote broadcasts a vote to all peers
func (n *P2PNetwork) BroadcastVote(vote *votingdomain.Vote) error {
	data, err := networkdomain.SerializeVote(vote)
	if err != nil {
		return fmt.Errorf("failed to serialize vote: %w", err)
	}

	return n.publishToTopic(topicVotes, data)
}

// BroadcastBlockProposal broadcasts a block proposal to all validators
func (n *P2PNetwork) BroadcastBlockProposal(block *domain.Block) error {
	data, err := networkdomain.SerializeBlock(block)
	if err != nil {
		return fmt.Errorf("failed to serialize block: %w", err)
	}

	return n.publishToTopic(topicBlockProposals, data)
}

// BroadcastBlockApproval broadcasts a block approval to all validators
func (n *P2PNetwork) BroadcastBlockApproval(blockHash crypto.Hash, validator crypto.PublicKey) error {
	data, err := networkdomain.SerializeApproval(blockHash, validator)
	if err != nil {
		return fmt.Errorf("failed to serialize approval: %w", err)
	}

	return n.publishToTopic(topicBlockApprovals, data)
}

// BroadcastSyncRequest requests a range of missing blocks from peers.
func (n *P2PNetwork) BroadcastSyncRequest(fromHeight, toHeight uint64) error {
	data, err := networkdomain.SerializeSyncRequest(fromHeight, toHeight)
	if err != nil {
		return fmt.Errorf("failed to serialize sync request: %w", err)
	}

	return n.publishToTopic(topicSyncRequests, data)
}

// BroadcastSyncResponse broadcasts blocks requested by a syncing peer.
func (n *P2PNetwork) BroadcastSyncResponse(blocks []*domain.Block) error {
	data, err := networkdomain.SerializeSyncResponse(blocks)
	if err != nil {
		return fmt.Errorf("failed to serialize sync response: %w", err)
	}

	return n.publishToTopic(topicSyncResponses, data)
}

// publishToTopic publishes data to a specific topic
func (n *P2PNetwork) publishToTopic(topicName string, data []byte) error {
	n.mu.RLock()
	topic, exists := n.topics[topicName]
	n.mu.RUnlock()

	if !exists {
		return fmt.Errorf("topic %s not found", topicName)
	}

	if err := topic.Publish(n.ctx, data); err != nil {
		return fmt.Errorf("failed to publish to topic %s: %w", topicName, err)
	}

	return nil
}

// handleTopicMessages listens for messages on a topic and dispatches to handlers
func (n *P2PNetwork) handleTopicMessages(topicName string, sub *pubsub.Subscription) {
	for {
		msg, err := sub.Next(n.ctx)
		if err != nil {
			// Context cancelled or subscription closed
			return
		}

		// Skip messages from ourselves
		if msg.ReceivedFrom == n.host.ID() {
			continue
		}

		// Dispatch message based on topic
		if err := n.dispatchMessage(topicName, msg); err != nil {
			fmt.Printf("Error handling message from topic %s: %v\n", topicName, err)
		}
	}
}

// dispatchMessage deserializes and dispatches a message to registered handlers
func (n *P2PNetwork) dispatchMessage(topicName string, msg *pubsub.Message) error {
	n.mu.RLock()
	handlers := make([]ports.MessageHandler, len(n.handlers))
	copy(handlers, n.handlers)
	n.mu.RUnlock()

	from := msg.ReceivedFrom.String()

	switch topicName {
	case topicVotes:
		vote, err := networkdomain.DeserializeVote(msg.Data)
		if err != nil {
			return fmt.Errorf("failed to deserialize vote: %w", err)
		}

		for _, handler := range handlers {
			if err := handler.HandleVote(vote, from); err != nil {
				fmt.Printf("Handler error for vote: %v\n", err)
			}
		}

	case topicBlockProposals:
		block, err := networkdomain.DeserializeBlock(msg.Data)
		if err != nil {
			return fmt.Errorf("failed to deserialize block: %w", err)
		}

		for _, handler := range handlers {
			if err := handler.HandleBlockProposal(block, from); err != nil {
				fmt.Printf("Handler error for block proposal: %v\n", err)
			}
		}

	case topicBlockApprovals:
		blockHash, validator, err := networkdomain.DeserializeApproval(msg.Data)
		if err != nil {
			return fmt.Errorf("failed to deserialize approval: %w", err)
		}

		for _, handler := range handlers {
			if err := handler.HandleBlockApproval(blockHash, validator, from); err != nil {
				fmt.Printf("Handler error for block approval: %v\n", err)
			}
		}

	case topicSyncRequests:
		fromHeight, toHeight, err := networkdomain.DeserializeSyncRequest(msg.Data)
		if err != nil {
			return fmt.Errorf("failed to deserialize sync request: %w", err)
		}

		for _, handler := range handlers {
			if err := handler.HandleSyncRequest(fromHeight, toHeight, from); err != nil {
				fmt.Printf("Handler error for sync request: %v\n", err)
			}
		}

	case topicSyncResponses:
		blocks, err := networkdomain.DeserializeSyncResponse(msg.Data)
		if err != nil {
			return fmt.Errorf("failed to deserialize sync response: %w", err)
		}

		for _, handler := range handlers {
			if err := handler.HandleSyncResponse(blocks, from); err != nil {
				fmt.Printf("Handler error for sync response: %v\n", err)
			}
		}

	default:
		return fmt.Errorf("unknown topic: %s", topicName)
	}

	return nil
}

// GetHostID returns the host's peer ID
func (n *P2PNetwork) GetHostID() peer.ID {
	return n.host.ID()
}

// GetListenAddresses returns the addresses the host is listening on
func (n *P2PNetwork) GetListenAddresses() []multiaddr.Multiaddr {
	return n.host.Addrs()
}

// GetConnectedPeers returns the list of connected peer IDs
func (n *P2PNetwork) GetConnectedPeers() []peer.ID {
	return n.host.Network().Peers()
}
