# peer-vote

A decentralized electronic voting system built on a permissioned blockchain with Proof of Authority (PoA) consensus. Votes are cryptographically signed, validated against an eligibility list, and immutably recorded on-chain through a P2P validator network.

## Architecture

The system is organized into domain-driven modules:

```
peer-vote/
├── blockchain/     # Chain, block, and genesis domain + file-based storage
├── consensus/      # PoA engine, leader selection, round state machine, finalization
├── crypto/         # Key pairs, signing, signature verification, hashing
├── mempool/        # In-memory vote queue before block inclusion
├── network/        # libp2p P2P adapter, pub/sub messaging, sync manager
├── node/           # Central orchestrator — event loop wiring all modules together
├── voting/         # Vote domain entity, validator, eligibility checking
├── cmd/
│   ├── node/       # Node binary entry point
│   ├── bootstrap/  # Bootstrap (seed) node entry point
│   └── submit-vote/# CLI tool for casting a vote
└── simulation/     # Test scripts and simulation tooling
```

Key design choices:
- **Hexagonal architecture** — each module exposes ports (interfaces) and adapters (implementations), keeping domain logic free of infrastructure concerns.
- **Event-driven node** — the `Node` engine runs a single event loop that dispatches votes, block proposals, approvals, and timeouts to the appropriate handler.
- **Immutable vote entities** — a `Vote` can only be constructed with a valid cryptographic signature; the constructor enforces this invariant.

## Prerequisites

- Go 1.21 or higher
- Network access for P2P communication (TCP)

## Quick Start

### 1. Build the binaries

```bash
go build -o bin/node      ./cmd/node
go build -o bin/bootstrap ./cmd/bootstrap
go build -o bin/vote      ./cmd/submit-vote
```

### 2. Generate validator keys

Each validator node needs its own keypair:

```bash
./bin/node -generate-key -key-path validator1.key
./bin/node -generate-key -key-path validator2.key
./bin/node -generate-key -key-path validator3.key
```

Each command prints the corresponding public key — save these for `validators.json`.

### 3. Create configuration files

**validators.json** — the authorized validator set (same file on every node):
```json
{
  "validators": [
    { "publicKey": "<node1_pubkey_hex>", "address": "/ip4/127.0.0.1/tcp/4001" },
    { "publicKey": "<node2_pubkey_hex>", "address": "/ip4/127.0.0.1/tcp/4002" },
    { "publicKey": "<node3_pubkey_hex>", "address": "/ip4/127.0.0.1/tcp/4003" }
  ]
}
```

**eligibility.json** — the list of voter IDs allowed to cast ballots:
```json
{
  "eligible_voters": ["alice", "bob", "carol"]
}
```

**config1.json** — per-node configuration:
```json
{
  "validator_key_path": "validator1.key",
  "p2p_port": 4001,
  "p2p_address": "/ip4/127.0.0.1/tcp/4001",
  "bootstrap_peers": [],
  "network_id": "peer-vote-testnet",
  "block_interval": "5s",
  "validator_config": "validators.json",
  "data_dir": "./data1",
  "eligibility_list_path": "eligibility.json"
}
```

For nodes 2 and 3, change `validator_key_path`, `p2p_port`, `p2p_address`, `data_dir`, and add node 1's multiaddr to `bootstrap_peers`:
```json
"bootstrap_peers": ["/ip4/127.0.0.1/tcp/4001/p2p/<node1_peer_id>"]
```

### 4. Start the nodes

```bash
./bin/node -config config1.json
./bin/node -config config2.json
./bin/node -config config3.json
```

Node 1's peer ID is printed on startup — use it in the `bootstrap_peers` of the other configs.

### 5. Submit a vote

```bash
./bin/vote -voter alice -choice "CandidateA" -node /ip4/127.0.0.1/tcp/4001/p2p/<peer_id>
```

## How It Works

1. A voter signs their ballot with their private key and submits it to any node.
2. The receiving node validates the vote: checks the eligibility list, verifies the signature, and adds it to the mempool.
3. The elected leader for the current round proposes a block containing pending votes.
4. Other validators verify the proposed block and broadcast their approvals.
5. Once a quorum of approvals is collected, the finalizer commits the block to the chain.
6. On restart, the node replays persisted blocks to reconstruct state and verifies chain integrity.

## Consensus

The system uses **Proof of Authority** with a round-robin leader selection. Each round has four states: `Idle → Proposing → Validating → Finalizing`. A round timeout resets the state machine, preventing livelock when the leader is unavailable.

## Data Storage

Block data is stored in the configured `data_dir`:
- `blk*.dat` — raw block files
- `index.dat` — block index for O(1) lookups by hash or height

## Simulation & Testing

The `simulation/` directory contains shell scripts for functional, security, performance, scalability, and fault-tolerance tests. See [`simulation/README.md`](simulation/README.md) for details.

## License

MIT
