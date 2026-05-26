package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"peer-vote/blockchain/adapters"
	"peer-vote/blockchain/domain"
	"peer-vote/blockchain/engine"
	consensusAdapters "peer-vote/consensus/adapters"
	consensusEngine "peer-vote/consensus/engine"
	"peer-vote/crypto"
	mempoolDomain "peer-vote/mempool/domain"
	networkAdapters "peer-vote/network/adapters"
	networkEngine "peer-vote/network/engine"
	"peer-vote/node/config"
	nodeEngine "peer-vote/node/engine"
	nodePorts "peer-vote/node/ports"
	votingAdapters "peer-vote/voting/adapters"
	votingEngine "peer-vote/voting/engine"
)

func main() {
	// Parse command-line flags
	configPath := flag.String("config", "config.json", "Path to configuration file")
	generateKey := flag.Bool("generate-key", false, "Generate a new validator keypair and exit")
	keyPath := flag.String("key-path", "validator.key", "Path to save/load validator key")
	flag.Parse()

	// Handle key generation
	if *generateKey {
		if err := generateValidatorKey(*keyPath); err != nil {
			log.Fatalf("Failed to generate validator key: %v", err)
		}
		fmt.Printf("Validator key generated and saved to: %s\n", *keyPath)
		return
	}

	// Load configuration
	log.Printf("Loading configuration from: %s\n", *configPath)
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Load or generate validator keypair
	log.Printf("Loading validator keypair from: %s\n", cfg.ValidatorKeyPath)
	keyPair, err := loadOrGenerateKeyPair(cfg.ValidatorKeyPath)
	if err != nil {
		log.Fatalf("Failed to load validator keypair: %v", err)
	}

	// Create signer
	signer := crypto.NewEd25519Signer(keyPair)
	log.Printf("Validator public key: %x\n", signer.PublicKey())

	// Initialize storage
	log.Printf("Initializing block storage in: %s\n", cfg.DataDir)
	blockStore, err := adapters.NewBlockStore(cfg.DataDir)
	if err != nil {
		log.Fatalf("Failed to initialize block storage: %v", err)
	}

	// Initialize or load blockchain
	var blockchain *domain.Chain
	var state *domain.State

	// Try to load existing blockchain from storage
	blocks, err := blockStore.GetAll()
	if err != nil {
		log.Fatalf("Failed to load blocks from storage: %v", err)
	}

	if len(blocks) == 0 {
		// No existing blockchain - create genesis block from config
		log.Println("No existing blockchain found, creating genesis block...")

		// Use default genesis config (same across all nodes)
		genesisConfig := domain.DefaultGenesisConfig()
		genesis, err := domain.CreateGenesisFromConfig(genesisConfig)
		if err != nil {
			log.Fatalf("Failed to create genesis block: %v", err)
		}

		// Verify genesis block
		if err := domain.VerifyGenesisBlock(genesis); err != nil {
			log.Fatalf("Genesis block verification failed: %v", err)
		}

		// Store genesis block
		if err := blockStore.Store(genesis); err != nil {
			log.Fatalf("Failed to store genesis block: %v", err)
		}

		// Initialize blockchain with genesis
		blockchain, err = domain.NewChain(genesis)
		if err != nil {
			log.Fatalf("Failed to create blockchain: %v", err)
		}
		state, err = domain.NewState(blockchain)
		if err != nil {
			log.Fatalf("Failed to create state: %v", err)
		}

		log.Printf("Genesis block created with hash: %s\n", genesis.Hash().String())
	} else {
		// Load existing blockchain
		log.Printf("Loading existing blockchain with %d blocks...\n", len(blocks))

		// Verify genesis block
		if err := domain.VerifyGenesisBlock(blocks[0]); err != nil {
			log.Fatalf("Invalid genesis block in storage: %v", err)
		}

		// Initialize blockchain with genesis
		blockchain, err = domain.NewChain(blocks[0])
		if err != nil {
			log.Fatalf("Failed to create blockchain: %v", err)
		}

		// Verify and append remaining blocks
		for i := 1; i < len(blocks); i++ {
			if err := blockchain.Append(blocks[i]); err != nil {
				log.Fatalf("Failed to append block %d to chain: %v", i, err)
			}
		}

		// Verify chain integrity
		if err := blockchain.VerifyIntegrity(); err != nil {
			log.Fatalf("Blockchain integrity check failed: %v", err)
		}

		// Reconstruct state from blocks
		state, err = domain.NewState(blockchain)
		if err != nil {
			log.Fatalf("Failed to create state: %v", err)
		}
		if err := state.Reconstruct(blocks); err != nil {
			log.Fatalf("Failed to reconstruct state from blocks: %v", err)
		}

		log.Printf("Blockchain loaded successfully, current height: %d\n", blockchain.Height())
	}

	// Initialize mempool
	mempool := mempoolDomain.NewMempool()

	// Load validator registry
	log.Printf("Loading validator registry from: %s\n", cfg.ValidatorConfig)
	validatorRegistry, err := consensusAdapters.NewConfigValidatorRegistry(cfg.ValidatorConfig)
	if err != nil {
		log.Fatalf("Failed to load validator registry: %v", err)
	}
	log.Printf("Loaded %d validators\n", validatorRegistry.Count())

	// Load eligibility list
	log.Printf("Loading eligibility list from: %s\n", cfg.EligibilityListPath)
	eligibilityChecker, err := votingAdapters.LoadEligibilityListFromFile(cfg.EligibilityListPath)
	if err != nil {
		log.Fatalf("Failed to load eligibility list: %v", err)
	}
	eligibleVoters, _ := eligibilityChecker.GetEligibleVoters()
	log.Printf("Loaded %d eligible voters\n", len(eligibleVoters))

	// Initialize vote validator
	voteValidator := votingEngine.NewVoteValidator(eligibilityChecker)

	// Initialize block builder
	blockBuilder := engine.NewBlockBuilder(signer)

	// Initialize consensus engine
	poaEngine := consensusEngine.NewPoAEngine(
		validatorRegistry,
		blockBuilder,
		cfg.BlockInterval.Duration,
	)

	// Initialize finalizer
	finalizer := consensusEngine.NewFinalizer(state, blockStore, mempool)

	// Initialize P2P network
	log.Printf("Initializing P2P network on port %d...\n", cfg.P2PPort)
	broadcaster, err := networkAdapters.NewP2PNetwork(cfg.P2PPort, cfg.BootstrapPeers)
	if err != nil {
		log.Fatalf("Failed to initialize P2P network: %v", err)
	}

	// Initialize sync manager
	syncManager, err := networkEngine.NewSyncManager(state, blockStore)
	if err != nil {
		log.Fatalf("Failed to initialize sync manager: %v", err)
	}

	// Initialize clock
	clock := nodePorts.NewRealClock()

	// Create node
	log.Println("Creating node...")
	node, err := nodeEngine.NewNode(
		cfg,
		signer,
		blockchain,
		state,
		mempool,
		poaEngine,
		syncManager,
		voteValidator,
		finalizer,
		blockStore,
		broadcaster,
		eligibilityChecker,
		clock,
	)
	if err != nil {
		log.Fatalf("Failed to create node: %v", err)
	}

	// Start node
	log.Println("Starting node...")
	if err := node.Start(); err != nil {
		log.Fatalf("Failed to start node: %v", err)
	}

	log.Println("Node started successfully!")
	log.Printf("Current blockchain height: %d\n", blockchain.Height())
	log.Printf("Mempool size: %d\n", mempool.Size())
	log.Println("Press Ctrl+C to shutdown...")

	// Wait for OS signals for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Block until signal received
	sig := <-sigChan
	log.Printf("\nReceived signal: %v\n", sig)
	log.Println("Initiating graceful shutdown...")

	// Shutdown node
	if err := node.Shutdown(); err != nil {
		log.Printf("Error during shutdown: %v\n", err)
	}

	log.Println("Node shutdown complete. Goodbye!")
}

// loadOrGenerateKeyPair loads a keypair from file, or generates a new one if the file doesn't exist
func loadOrGenerateKeyPair(path string) (*crypto.KeyPair, error) {
	// Check if key file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		log.Printf("Key file not found, generating new keypair...\n")
		keyPair, err := crypto.GenerateKeyPair()
		if err != nil {
			return nil, fmt.Errorf("failed to generate keypair: %w", err)
		}

		// Save the keypair
		if err := saveKeyPair(keyPair, path); err != nil {
			return nil, fmt.Errorf("failed to save keypair: %w", err)
		}

		log.Printf("New keypair generated and saved to: %s\n", path)
		return keyPair, nil
	}

	// Load existing keypair
	return crypto.LoadKeyPair(path)
}

// saveKeyPair saves a keypair to a file
func saveKeyPair(keyPair *crypto.KeyPair, path string) error {
	// Convert private key to hex string
	keyHex := fmt.Sprintf("%x", keyPair.Private)

	// Write to file
	if err := os.WriteFile(path, []byte(keyHex), 0600); err != nil {
		return fmt.Errorf("failed to write key file: %w", err)
	}

	return nil
}

// generateValidatorKey generates a new validator keypair and saves it to a file
func generateValidatorKey(path string) error {
	keyPair, err := crypto.GenerateKeyPair()
	if err != nil {
		return fmt.Errorf("failed to generate keypair: %w", err)
	}

	if err := saveKeyPair(keyPair, path); err != nil {
		return fmt.Errorf("failed to save keypair: %w", err)
	}

	fmt.Printf("Public key: %x\n", keyPair.Public)
	return nil
}
