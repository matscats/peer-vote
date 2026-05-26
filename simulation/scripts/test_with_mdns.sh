#!/bin/bash

# Test script for multi-node blockchain with mDNS peer discovery

echo "=========================================="
echo "Multi-Node Blockchain Test with mDNS"
echo "=========================================="
echo ""

# Clean up old data
echo "Cleaning up old data..."
rm -rf data1 data2 data3
rm -f node1.log node2.log node3.log

# Build the project
echo "Building project..."
go build -o bin/node cmd/node/main.go
if [ $? -ne 0 ]; then
    echo "Build failed!"
    exit 1
fi

echo "Build successful!"
echo ""

# Start Node 1 (bootstrap node)
echo "Starting Node 1 (bootstrap)..."
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
echo "Node 1 PID: $NODE1_PID"
echo "Node 2 PID: $NODE2_PID"
echo "Node 3 PID: $NODE3_PID"
echo ""

# Wait for mDNS discovery and first block
echo "Waiting 15 seconds for mDNS discovery and first block..."
sleep 15

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
echo "Checking for mDNS Discovery"
echo "=========================================="
grep -h "mDNS" node*.log | sort -u
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
echo "  tail -f node1.log node2.log node3.log"
echo ""

# Wait for user interrupt
trap "echo 'Stopping all nodes...'; kill $NODE1_PID $NODE2_PID $NODE3_PID 2>/dev/null; exit 0" INT
wait
