package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/p2p/discovery/routing"
	"github.com/libp2p/go-libp2p/p2p/discovery/util"
)

func main() {
	// Parse command-line flags
	port := flag.Int("port", 4000, "Port to listen on")
	flag.Parse()

	log.Printf("Starting Bootstrap Node on port %d...\n", *port)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create libp2p host
	listenAddr := fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", *port)
	h, err := libp2p.New(
		libp2p.ListenAddrStrings(listenAddr),
		libp2p.EnableNATService(),
	)
	if err != nil {
		log.Fatalf("Failed to create libp2p host: %v", err)
	}
	defer h.Close()

	log.Printf("Bootstrap Node created with ID: %s\n", h.ID())
	log.Printf("Listening on: %v\n", h.Addrs())

	// Print full multiaddrs with peer ID for other nodes to use
	log.Println("\n=== Bootstrap Node Addresses ===")
	for _, addr := range h.Addrs() {
		fullAddr := fmt.Sprintf("%s/p2p/%s", addr, h.ID())
		log.Printf("  %s\n", fullAddr)
	}
	log.Println("================================\n")

	// Create DHT in server mode
	kadDHT, err := dht.New(ctx, h, dht.Mode(dht.ModeServer))
	if err != nil {
		log.Fatalf("Failed to create DHT: %v", err)
	}

	// Bootstrap the DHT
	if err = kadDHT.Bootstrap(ctx); err != nil {
		log.Fatalf("Failed to bootstrap DHT: %v", err)
	}

	log.Println("DHT bootstrapped successfully")

	// Create routing discovery
	routingDisc := routing.NewRoutingDiscovery(kadDHT)

	// Announce ourselves on the rendezvous point
	const rendezvousString = "peer-vote-network"
	log.Printf("Announcing presence on rendezvous: %s\n", rendezvousString)
	util.Advertise(ctx, routingDisc, rendezvousString)

	// Start periodic status reporting
	go reportStatus(ctx, h, kadDHT)

	log.Println("Bootstrap node is running!")
	log.Println("Other nodes can connect using the addresses above")
	log.Println("Press Ctrl+C to shutdown...")

	// Wait for OS signals for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Block until signal received
	sig := <-sigChan
	log.Printf("\nReceived signal: %v\n", sig)
	log.Println("Shutting down bootstrap node...")
}

// reportStatus periodically reports the status of the bootstrap node
func reportStatus(ctx context.Context, h host.Host, dht *dht.IpfsDHT) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			peers := h.Network().Peers()
			log.Printf("Status: %d peer(s) connected\n", len(peers))

			if len(peers) > 0 {
				log.Println("Connected peers:")
				for _, p := range peers {
					log.Printf("  - %s\n", p)
				}
			}

			// Report DHT routing table size
			routingTableSize := dht.RoutingTable().Size()
			log.Printf("DHT routing table size: %d\n", routingTableSize)
		}
	}
}
