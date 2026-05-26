#!/bin/bash

# Test script for multi-node blockchain with DHT and bootstrap node

echo "=========================================="
echo "Multi-Node Blockchain Test with DHT"
echo "=========================================="
echo ""

# Clean up old data
echo "Cleaning up old data..."
rm -rf data1 data2 data3
rm -f node1.log node2.log node3.log bootstrap.log

# Build the projects
echo "Building projects..."
go build -o bin/bootstrap cmd/bootstrap/main.go
go build -o bin/node cmd/node/main.go
if [ $? -ne 0 ]; then
    echo "Build failed!"
    exit 1
fi

echo "Build successful!"
echo ""

# Start Bootstrap Node
echo "Starting Bootstrap Node..."
./bin/bootstrap -port 4000 > bootstrap.log 2>&1 &
BOOTSTRAP_PID=$!
echo "Bootstrap node started with PID: $BOOTSTRAP_PID"
sleep 3

# Extract bootstrap node peer ID and address from log
echo "Waiting for bootstrap node to initialize..."
sleep 2

# Get the bootstrap node's multiaddr with peer ID
BOOTSTRAP_ADDR=$(grep "/ip4/127.0.0.1/tcp/4000/p2p/" bootstrap.log | head -1 | sed 's/.*\(\/ip4\/127\.0\.0\.1\/tcp\/4000\/p2p\/[^ ]*\).*/\1/')

if [ -z "$BOOTSTRAP_ADDR" ]; then
    echo "ERROR: Could not get bootstrap node address!"
    echo "Bootstrap log:"
    cat bootstrap.log
    kill $BOOTSTRAP_PID 2>/dev/null
    exit 1
fi

echo "Bootstrap node address: $BOOTSTRAP_ADDR"
echo ""

# Update config files with bootstrap address
echo "Updating config files with bootstrap address..."
for config in config1.json config2.json config3.json; do
    # Use jq if available, otherwise use sed
    if command -v jq &> /dev/null; then
        jq ".bootstrap_peers = [\"$BOOTSTRAP_ADDR\"]" $config > ${config}.tmp && mv ${config}.tmp $config
    else
        # Fallback to manual editing
        sed -i.bak "s|\"bootstrap_peers\": \[.*\]|\"bootstrap_peers\": [\"$BOOTSTRAP_ADDR\"]|" $config
        rm -f ${config}.bak
    fi
done

echo "Config files updated!"
echo ""

# Start Node 1
echo "Starting Node 1..."
./bin/node -config config1.json > node1.log 2>&1 &
NODE1_PID=$!
echo "Node 1 started with PID: $NODE1_PID"
sleep 2

# Start Node 2
echo "Starting Node 2..."
./bin/node -config config2.json > node2.log 2>&1 &
NODE2_PID=$!
echo "Node 2 started with PID: $NODE2_PID"
sleep 2

# Start Node 3
echo "Starting Node 3..."
./bin/node -config config3.json > node3.log 2>&1 &
NODE3_PID=$!
echo "Node 3 started with PID: $NODE3_PID"
sleep 3

echo ""
echo "All nodes started!"
echo "Bootstrap PID: $BOOTSTRAP_PID"
echo "Node 1 PID: $NODE1_PID"
echo "Node 2 PID: $NODE2_PID"
echo "Node 3 PID: $NODE3_PID"
echo ""

# Wait for DHT discovery and first blocks
echo "Waiting 20 seconds for DHT discovery and block finalization..."
sleep 20

echo ""
echo "=========================================="
echo "Checking Bootstrap Node"
echo "=========================================="
tail -15 bootstrap.log
echo ""

echo "=========================================="
echo "Checking Node Logs"
echo "=========================================="
echo ""

echo "--- Node 1 Log (last 20 lines) ---"
tail -20 node1.log
echo ""

echo "--- Node 2 Log (last 20 lines) ---"
tail -20 node2.log
echo ""

echo "--- Node 3 Log (last 20 lines) ---"
tail -20 node3.log
echo ""

echo "=========================================="
echo "Checking for DHT Discovery"
echo "=========================================="
grep -h "DHT discovered peer\|Successfully connected to peer" node*.log | sort -u
echo ""

echo "=========================================="
echo "Checking for Connected Peers"
echo "=========================================="
grep -h "Connected peers:" node*.log | tail -3
echo ""

echo "=========================================="
echo "Checking for Block Finalization"
echo "=========================================="
grep -h "Block.*finalized" node*.log | sort -u
echo ""

echo "=========================================="
echo "Checking for Errors"
echo "=========================================="
grep -h "ERROR" node*.log | head -10
echo ""

# Keep nodes running for observation
echo "Nodes are still running. Press Ctrl+C to stop all nodes."
echo "To monitor logs in real-time, run:"
echo "  tail -f bootstrap.log node1.log node2.log node3.log"
echo ""

# Wait for user interrupt
trap "echo 'Stopping all nodes...'; kill $BOOTSTRAP_PID $NODE1_PID $NODE2_PID $NODE3_PID 2>/dev/null; exit 0" INT
wait
