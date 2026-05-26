# Blockchain PoA Voting System - Node

This is the main entry point for running a blockchain node in the Proof of Authority (PoA) voting system.

## Quick Start

### 1. Generate Validator Key

First, generate a validator keypair:

```bash
go run main.go -generate-key -key-path validator.key
```

This will output the public key in hex format. Save this for the validator configuration.

### 2. Configure the Node

Create configuration files based on the examples in the project root:

**config.json** - Main node configuration:
```json
{
  "validator_key_path": "validator.key",
  "p2p_port": 4001,
  "p2p_address": "/ip4/127.0.0.1/tcp/4001",
  "bootstrap_peers": [],
  "network_id": "peer-vote-testnet",
  "block_interval": "5s",
  "validator_config": "validators.json",
  "data_dir": "./data",
  "eligibility_list_path": "eligibility.json"
}
```

**validators.json** - List of authorized validators:
```json
{
  "validators": [
    {
      "publicKey": "your_validator_public_key_hex",
      "address": "/ip4/127.0.0.1/tcp/4001"
    }
  ]
}
```

**eligibility.json** - List of eligible voters:
```json
{
  "eligible_voters": [
    "voter1",
    "voter2",
    "voter3"
  ]
}
```

### 3. Start the Node

Run the node with the configuration file:

```bash
go run main.go -config config.json
```

Or build and run:

```bash
go build -o node main.go
./node -config config.json
```

## Command-Line Flags

- `-config <path>` - Path to configuration file (default: "config.json")
- `-generate-key` - Generate a new validator keypair and exit
- `-key-path <path>` - Path to save/load validator key (default: "validator.key")

## Multi-Node Setup

To run multiple nodes for testing:

1. Generate a keypair for each validator
2. Create separate configuration files for each node with different:
   - `p2p_port` (e.g., 4001, 4002, 4003)
   - `validator_key_path` (e.g., validator1.key, validator2.key, validator3.key)
   - `data_dir` (e.g., ./data1, ./data2, ./data3)
3. Configure `bootstrap_peers` to connect nodes together
4. Ensure all nodes have the same `validators.json` and `eligibility.json`

Example for 3 nodes:

**Node 1 (config1.json):**
```json
{
  "validator_key_path": "validator1.key",
  "p2p_port": 4001,
  "bootstrap_peers": [],
  "data_dir": "./data1",
  ...
}
```

**Node 2 (config2.json):**
```json
{
  "validator_key_path": "validator2.key",
  "p2p_port": 4002,
  "bootstrap_peers": ["/ip4/127.0.0.1/tcp/4001/p2p/<node1_peer_id>"],
  "data_dir": "./data2",
  ...
}
```

**Node 3 (config3.json):**
```json
{
  "validator_key_path": "validator3.key",
  "p2p_port": 4003,
  "bootstrap_peers": ["/ip4/127.0.0.1/tcp/4001/p2p/<node1_peer_id>"],
  "data_dir": "./data3",
  ...
}
```

## Graceful Shutdown

The node handles OS signals (SIGINT, SIGTERM) for graceful shutdown. Press `Ctrl+C` to stop the node cleanly.

## Features

- **Automatic Genesis Block Creation**: If no blockchain exists, the node creates and stores a genesis block
- **State Recovery**: On restart, the node reconstructs its state from persisted blocks
- **Chain Integrity Verification**: The node verifies the entire blockchain on startup
- **P2P Networking**: Connects to other validators via libp2p
- **Event-Driven Architecture**: Processes votes, block proposals, and approvals asynchronously
- **Consensus Participation**: Participates in PoA consensus as a validator

## Logs

The node outputs detailed logs including:
- Configuration loading
- Validator keypair loading
- Genesis block creation/loading
- Blockchain state recovery
- P2P network initialization
- Event processing
- Consensus operations

## Data Storage

The node stores blockchain data in the configured `data_dir`:
- `blk*.dat` - Block data files
- `index.dat` - Block index for fast lookups

## Requirements

- Go 1.21 or higher
- Network connectivity for P2P communication
- Write permissions for data directory
