# Simulation Environment

This directory contains all the necessary files and scripts to run simulations of the blockchain voting system.

## Directory Structure

```
simulation/
├── bin/              # Compiled binaries (bootstrap, node)
├── configs/          # Configuration files for nodes and validators
├── keys/             # Validator keypairs
├── scripts/          # Test and simulation scripts
├── data/             # Blockchain data directories (data1, data2, data3, etc.)
├── logs/             # Node and bootstrap logs
└── docs/             # Simulation documentation and results
```

## Quick Start

### 1. Build the Binaries

From the project root:

```bash
go build -o simulation/bin/bootstrap cmd/bootstrap/main.go
go build -o simulation/bin/node cmd/node/main.go
```

Or use the build script:

```bash
cd simulation/scripts
./build.sh
```

### 2. Run a Multi-Node Simulation

```bash
cd simulation/scripts
./test_with_bootstrap.sh
```

This will:
- Start a bootstrap node for peer discovery
- Start 3 validator nodes
- Show consensus in action with block proposals and finalization

### 3. Submit Votes

While the simulation is running, you can submit votes:

```bash
cd simulation/scripts
./submit_vote.sh voter1 candidate-a
```

## Available Scripts

### Core Scripts

- **`test_with_bootstrap.sh`** - Complete multi-node test with DHT bootstrap
- **`start_bootstrap.sh`** - Start only the bootstrap node
- **`build.sh`** - Build all binaries
- **`visualize.sh`** - Generate network visualization from logs (for TCC)

### Testing Scripts

- **`quick_test.sh`** - Quick system test with 3 votes
- **`stress_test.sh`** - Stress test with rapid vote submission
- **`simulate_election.sh`** - Full election simulation with multiple voters

### Legacy Scripts (for reference)

- `test_multi_node.sh` - Original multi-node test
- `test_simple.sh` - Simple monitoring script
- `test_with_mdns.sh` - mDNS-based discovery (deprecated)

## Configuration Files

### Node Configurations

- `config1.json`, `config2.json`, `config3.json` - Individual node configs
- `config.example.json` - Template for creating new node configs

### Validator Registry

- `validators.json` - List of authorized validators (public keys)
- `validators.example.json` - Template

### Voter Eligibility

- `eligibility.json` - List of eligible voters
- `eligibility.example.json` - Template

## Creating New Simulations

### Adding More Nodes

1. Copy `config.example.json` to `config4.json`
2. Update the port and data directory
3. Generate a new validator key:
   ```bash
   ../bin/node -generate-key -key-path ../keys/validator4.key
   ```
4. Add the public key to `validators.json`
5. Update the bootstrap peers in the config

### Custom Scenarios

Create new scripts in `simulation/scripts/` for specific test scenarios:

- Different numbers of validators
- Network partitions
- Byzantine behavior simulation
- Load testing with many votes

## Monitoring

### Real-time Logs

```bash
tail -f logs/node*.log logs/bootstrap.log
```

### Check Block Finalization

```bash
grep "finalized" logs/node*.log
```

### Check Peer Connections

```bash
grep "Connected peers" logs/node*.log
```

## Data Management

### Clean All Data

```bash
rm -rf data/* logs/*
```

### Backup Simulation Results

```bash
tar -czf results_$(date +%Y%m%d_%H%M%S).tar.gz data/ logs/
```

## Troubleshooting

### Nodes Not Connecting

1. Check bootstrap node is running: `ps aux | grep bootstrap`
2. Verify bootstrap address in node configs
3. Check firewall settings

### Blocks Not Finalizing

1. Ensure at least 2/3 validators are running
2. Check for errors in logs: `grep ERROR logs/node*.log`
3. Verify validator keys match the registry

### Port Conflicts

If ports are already in use, update the `p2p_port` in config files.

## Performance Metrics

Track these metrics for your TCC:

- **Block finalization time**: Time from proposal to finalization
- **Throughput**: Votes processed per second
- **Network latency**: Time for message propagation
- **Consensus rounds**: Number of rounds to reach consensus
- **Peer discovery time**: Time to connect all nodes

## For TCC Documentation

### Network Visualization

Generate interactive visualizations of your simulation:

```bash
cd simulation/scripts
./visualize.sh
```

This creates:
- **`visualization.html`** - Interactive web page with network map, statistics, and timeline
- **`network_data.json`** - Raw data for custom analysis

See [VISUALIZATION.md](VISUALIZATION.md) for detailed documentation.

### Documentation Files

The `docs/` directory contains:

- Test results and observations
- Performance benchmarks
- Network topology diagrams
- Consensus flow documentation

Add your simulation results here for easy reference in your thesis.
