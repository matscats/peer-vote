# Multi-Node Integration Testing Setup

This directory contains configuration files for running a 3-node blockchain PoA voting system for integration testing.

## Configuration Files

### Validator Keys
- `validator1.key` - Private key for Node 1
- `validator2.key` - Private key for Node 2
- `validator3.key` - Private key for Node 3

### Node Configurations
- `config1.json` - Configuration for Node 1 (port 4001, bootstrap node)
- `config2.json` - Configuration for Node 2 (port 4002, connects to Node 1)
- `config3.json` - Configuration for Node 3 (port 4003, connects to Node 1)

### Shared Configurations
- `validators.json` - List of all 3 authorized validators (must be identical across all nodes)
- `eligibility.json` - List of eligible voters (must be identical across all nodes)

## Node Details

### Node 1 (Bootstrap Node)
- **Port**: 4001
- **Data Directory**: `./data1`
- **Public Key**: `383eb67b43343e3fc1bcb711ce84fd9c13ef2f15c4025830dbc2a75666674503`
- **Role**: Bootstrap node (other nodes connect to it)

### Node 2
- **Port**: 4002
- **Data Directory**: `./data2`
- **Public Key**: `5f1a9046481483aa98f5e0a0b05e84b9505ffe82065dc9517d80e63b21a567d0`
- **Bootstrap Peer**: Node 1 at `/ip4/127.0.0.1/tcp/4001`

### Node 3
- **Port**: 4003
- **Data Directory**: `./data3`
- **Public Key**: `ff30a1748f699d0af2575b1bbfbc7946f56ac9a157b674fcabf568cf4773d1c5`
- **Bootstrap Peer**: Node 1 at `/ip4/127.0.0.1/tcp/4001`

## Starting the Network

### Option 1: Start Nodes in Separate Terminals

**Terminal 1 - Start Node 1 (Bootstrap):**
```bash
go run cmd/node/main.go -config config1.json
```

**Terminal 2 - Start Node 2:**
```bash
go run cmd/node/main.go -config config2.json
```

**Terminal 3 - Start Node 3:**
```bash
go run cmd/node/main.go -config config3.json
```

### Option 2: Start Nodes in Background

```bash
# Start Node 1
go run cmd/node/main.go -config config1.json > node1.log 2>&1 &
NODE1_PID=$!

# Wait for Node 1 to initialize
sleep 2

# Start Node 2
go run cmd/node/main.go -config config2.json > node2.log 2>&1 &
NODE2_PID=$!

# Start Node 3
go run cmd/node/main.go -config config3.json > node3.log 2>&1 &
NODE3_PID=$!

# View logs
tail -f node1.log node2.log node3.log

# Stop all nodes
kill $NODE1_PID $NODE2_PID $NODE3_PID
```

## Testing the Network

### 1. Verify Nodes are Connected

Check the logs to ensure nodes have discovered each other via P2P.

### 2. Submit Votes

You can submit votes by sending them to any node. The votes will propagate across the network.

### 3. Verify Consensus

Watch the logs to see:
- Leader selection rotating across nodes
- Block proposals from the designated leader
- Block approvals from validators
- Block finalization when majority approvals are received

### 4. Verify State Synchronization

- Start all 3 nodes
- Let them finalize several blocks
- Stop one node
- Let the remaining nodes finalize more blocks
- Restart the stopped node
- Verify it synchronizes and catches up

## Leader Rotation

With 3 validators, the leader selection follows round-robin:
- **Height 1**: Node 1 (index 1 % 3 = 1)
- **Height 2**: Node 2 (index 2 % 3 = 2)
- **Height 3**: Node 3 (index 3 % 3 = 0)
- **Height 4**: Node 1 (index 4 % 3 = 1)
- And so on...

## Consensus Requirements

- **Majority**: 2 out of 3 validators must approve a block for finalization
- **Block Interval**: 5 seconds (configurable in config files)

## Data Directories

Each node stores its blockchain data in a separate directory:
- Node 1: `./data1/` (contains `blk*.dat` and `index.dat`)
- Node 2: `./data2/`
- Node 3: `./data3/`

## Cleanup

To reset the network and start fresh:

```bash
# Remove all data directories
rm -rf data1 data2 data3

# Remove log files (if using background mode)
rm -f node1.log node2.log node3.log
```

## Troubleshooting

### Nodes Not Connecting

- Ensure Node 1 (bootstrap) is started first
- Check that ports 4001, 4002, 4003 are not in use
- Verify `bootstrap_peers` in config2.json and config3.json point to Node 1

### Blocks Not Finalizing

- Ensure at least 2 out of 3 nodes are running
- Check that all nodes have the same `validators.json`
- Verify network connectivity between nodes

### State Divergence

- Ensure all nodes have the same genesis block
- Verify all nodes have the same `validators.json` and `eligibility.json`
- Check logs for validation errors

## Network Topology

```
Node 1 (Bootstrap)
  :4001
    |
    +--- Node 2 :4002
    |
    +--- Node 3 :4003
```

All nodes connect to Node 1 initially, then discover each other via P2P gossip.
