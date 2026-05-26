package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/libp2p/go-libp2p"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"

	"peer-vote/crypto"
	"peer-vote/network/domain"
	votingdomain "peer-vote/voting/domain"
)

func main() {
	// Parse command-line flags
	voterID := flag.String("voter", "", "Voter ID (required)")
	choice := flag.String("choice", "", "Vote choice (required)")
	keyPath := flag.String("key", "", "Path to voter's private key (required)")
	targetPeer := flag.String("peer", "/ip4/127.0.0.1/tcp/4001", "Target peer multiaddr")
	flag.Parse()

	// Validate required flags
	if *voterID == "" || *choice == "" || *keyPath == "" {
		log.Fatal("Usage: submit-vote -voter <voter_id> -choice <choice> -key <key_path> [-peer <multiaddr>] [-network <network_id>]")
	}

	// Load voter's keypair
	log.Printf("Loading voter keypair from: %s\n", *keyPath)
	keyPair, err := crypto.LoadKeyPair(*keyPath)
	if err != nil {
		log.Fatalf("Failed to load keypair: %v", err)
	}

	// Create signer
	signer := crypto.NewEd25519Signer(keyPair)

	// Create vote
	log.Printf("Creating vote: voter=%s, choice=%s\n", *voterID, *choice)
	timestamp := time.Now()

	// Build and sign the vote message
	message := []byte(fmt.Sprintf("%s|%s|%d", *voterID, *choice, timestamp.Unix()))
	signature, err := signer.Sign(message)
	if err != nil {
		log.Fatalf("Failed to sign vote: %v", err)
	}

	// Create vote with signature
	vote, err := votingdomain.NewVote(*voterID, *choice, timestamp, signature, signer.PublicKey())
	if err != nil {
		log.Fatalf("Failed to create vote: %v", err)
	}

	// Serialize vote
	voteData, err := domain.SerializeVote(vote)
	if err != nil {
		log.Fatalf("Failed to serialize vote: %v", err)
	}

	// Create libp2p host
	ctx := context.Background()
	h, err := libp2p.New(
		libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"),
	)
	if err != nil {
		log.Fatalf("Failed to create libp2p host: %v", err)
	}
	defer h.Close()

	log.Printf("Created temporary peer with ID: %s\n", h.ID())

	// Connect to target peer
	log.Printf("Connecting to target peer: %s\n", *targetPeer)
	if err := connectToPeer(ctx, h, *targetPeer); err != nil {
		log.Fatalf("Failed to connect to peer: %v", err)
	}

	// Create pubsub
	ps, err := pubsub.NewGossipSub(ctx, h)
	if err != nil {
		log.Fatalf("Failed to create pubsub: %v", err)
	}

	// Join vote topic
	topicName := "peer-vote/votes"
	log.Printf("Joining topic: %s\n", topicName)
	topic, err := ps.Join(topicName)
	if err != nil {
		log.Fatalf("Failed to join topic: %v", err)
	}

	// Wait a bit for topic to propagate
	time.Sleep(2 * time.Second)

	// Publish vote
	log.Println("Publishing vote to network...")
	if err := topic.Publish(ctx, voteData); err != nil {
		log.Fatalf("Failed to publish vote: %v", err)
	}

	log.Println("Vote submitted successfully!")
	log.Printf("Vote details: voter=%s, choice=%s, timestamp=%s\n",
		vote.VoterID(), vote.Choice(), vote.Timestamp().Format(time.RFC3339))

	// Wait a bit to ensure message is sent
	time.Sleep(1 * time.Second)
}

// connectToPeer connects to a peer given its multiaddr
func connectToPeer(ctx context.Context, h host.Host, peerAddr string) error {
	// Parse multiaddr
	maddr, err := multiaddr.NewMultiaddr(peerAddr)
	if err != nil {
		return fmt.Errorf("invalid multiaddr: %w", err)
	}

	// Extract peer info
	peerInfo, err := peer.AddrInfoFromP2pAddr(maddr)
	if err != nil {
		return fmt.Errorf("failed to get peer info: %w", err)
	}

	// Connect to peer
	if err := h.Connect(ctx, *peerInfo); err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	return nil
}
