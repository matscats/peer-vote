#!/bin/bash

# Simple Multi-Node Test
# Starts 3 nodes and monitors their operation

set -e

echo "========================================="
echo "Simple Multi-Node Test"
echo "========================================="
echo ""

# Cleanup function
cleanup() {
    echo ""
    echo "Cleaning up..."
    
    # Kill all node processes
    pkill -f "go run cmd/node/main.go" || true
    
    # Wait for processes to terminate
    sleep 2
    
    echo "Cleanup complete"
}

# Set trap to cleanup on exit
trap cleanup EXIT

# Clean up any existing data
echo "Cleaning up existing data..."
rm -rf data1 data2 data3
rm -f node1.log node2.log node3.log voter*.key

echo ""
echo "Starting Node 1 (Bootstrap)..."
go run cmd/node/main.go -config config1.json > node1.log 2>&1 &
sleep 3

# Get Node 1 peer ID from logs
NODE1_PEER_ID=$(grep "Created temporary peer with ID" node1.log | head -1 | awk '{print $NF}' || echo "")
if [ -z "$NODE1_PEER_ID" ]; then
    # Try alternative log format
    NODE1_PEER_ID=$(grep -oP 'peer.*ID.*:\s*\K\S+' node1.log | head -1 || echo "")
fi

echo "Node 1 started"
if [ ! -z "$NODE1_PEER_ID" ]; then
    echo "  Peer ID: $NODE1_PEER_ID"
fi

# Check if Node 1 is running
if ! pgrep -f "config1.json" > /dev/null; then
    echo "ERROR: Node 1 failed to start!"
    echo "Last 30 lines of node1.log:"
    tail -30 node1.log
    exit 1
fi

echo ""
echo "Starting Node 2..."
go run cmd/node/main.go -config config2.json > node2.log 2>&1 &
sleep 2

echo "Node 2 started"

# Check if Node 2 is running
if ! pgrep -f "config2.json" > /dev/null; then
    echo "ERROR: Node 2 failed to start!"
    echo "Last 30 lines of node2.log:"
    tail -30 node2.log
    exit 1
fi

echo ""
echo "Starting Node 3..."
go run cmd/node/main.go -config config3.json > node3.log 2>&1 &
sleep 2

echo "Node 3 started"

# Check if Node 3 is running
if ! pgrep -f "config3.json" > /dev/null; then
    echo "ERROR: Node 3 failed to start!"
    echo "Last 30 lines of node3.log:"
    tail -30 node3.log
    exit 1
fi

echo ""
echo "All nodes started successfully!"
echo ""
echo "Waiting for nodes to initialize and discover each other..."
sleep 5

echo ""
echo "========================================="
echo "Node Status"
echo "========================================="
echo ""

echo "Node 1 (port 4001):"
echo "  Height: $(grep -oP 'current height: \K\d+' node1.log | tail -1 || echo '0')"
echo "  Genesis: $(grep -c 'Genesis block created' node1.log || echo '0') created"
echo ""

echo "Node 2 (port 4002):"
echo "  Height: $(grep -oP 'current height: \K\d+' node2.log | tail -1 || echo '0')"
echo "  Genesis: $(grep -c 'Genesis block created' node2.log || echo '0') created"
echo ""

echo "Node 3 (port 4003):"
echo "  Height: $(grep -oP 'current height: \K\d+' node3.log | tail -1 || echo '0')"
echo "  Genesis: $(grep -c 'Genesis block created' node3.log || echo '0') created"
echo ""

echo "========================================="
echo "Monitoring (Ctrl+C to stop)"
echo "========================================="
echo ""
echo "Watching for block proposals and finalizations..."
echo "Log files: node1.log, node2.log, node3.log"
echo ""

# Monitor logs for interesting events
tail -f node1.log node2.log node3.log | grep --line-buffered -E "(Block finalized|Block proposal|Vote received|Leader selected|Approval received)" &
TAIL_PID=$!

# Wait for user interrupt
wait $TAIL_PID
