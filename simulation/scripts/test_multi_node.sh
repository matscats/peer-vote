#!/bin/bash

# Multi-Node Integration Test Script
# Tests vote submission, propagation, and finalization across 3 nodes

set -e

echo "========================================="
echo "Multi-Node Integration Test"
echo "========================================="
echo ""

# Cleanup function
cleanup() {
    echo ""
    echo "Cleaning up..."
    
    # Kill all node processes
    if [ ! -z "$NODE1_PID" ]; then
        kill $NODE1_PID 2>/dev/null || true
    fi
    if [ ! -z "$NODE2_PID" ]; then
        kill $NODE2_PID 2>/dev/null || true
    fi
    if [ ! -z "$NODE3_PID" ]; then
        kill $NODE3_PID 2>/dev/null || true
    fi
    
    # Wait for processes to terminate
    sleep 2
    
    # Remove data directories
    rm -rf data1 data2 data3
    
    # Remove log files
    rm -f node1.log node2.log node3.log
    
    echo "Cleanup complete"
}

# Set trap to cleanup on exit
trap cleanup EXIT

# Clean up any existing data
echo "Cleaning up existing data..."
rm -rf data1 data2 data3
rm -f node1.log node2.log node3.log

echo ""
echo "Step 1: Starting Node 1 (Bootstrap)..."
go run cmd/node/main.go -config config1.json > node1.log 2>&1 &
NODE1_PID=$!
echo "Node 1 started with PID: $NODE1_PID"

# Wait for Node 1 to initialize
echo "Waiting for Node 1 to initialize..."
sleep 3

# Check if Node 1 is still running
if ! kill -0 $NODE1_PID 2>/dev/null; then
    echo "ERROR: Node 1 failed to start!"
    echo "Last 20 lines of node1.log:"
    tail -20 node1.log
    exit 1
fi

echo "Node 1 initialized successfully"
echo ""

echo "Step 2: Starting Node 2..."
go run cmd/node/main.go -config config2.json > node2.log 2>&1 &
NODE2_PID=$!
echo "Node 2 started with PID: $NODE2_PID"

# Wait for Node 2 to initialize
sleep 2

# Check if Node 2 is still running
if ! kill -0 $NODE2_PID 2>/dev/null; then
    echo "ERROR: Node 2 failed to start!"
    echo "Last 20 lines of node2.log:"
    tail -20 node2.log
    exit 1
fi

echo "Node 2 initialized successfully"
echo ""

echo "Step 3: Starting Node 3..."
go run cmd/node/main.go -config config3.json > node3.log 2>&1 &
NODE3_PID=$!
echo "Node 3 started with PID: $NODE3_PID"

# Wait for Node 3 to initialize
sleep 2

# Check if Node 3 is still running
if ! kill -0 $NODE3_PID 2>/dev/null; then
    echo "ERROR: Node 3 failed to start!"
    echo "Last 20 lines of node3.log:"
    tail -20 node3.log
    exit 1
fi

echo "Node 3 initialized successfully"
echo ""

echo "Step 4: Waiting for nodes to discover each other..."
sleep 3
echo "Nodes should now be connected"
echo ""

echo "Step 5: Generating voter keys..."
# Generate voter keys
go run cmd/node/main.go -generate-key -key-path voter1.key > /dev/null 2>&1
go run cmd/node/main.go -generate-key -key-path voter2.key > /dev/null 2>&1
go run cmd/node/main.go -generate-key -key-path voter3.key > /dev/null 2>&1
echo "Voter keys generated"
echo ""

echo "Step 6: Submitting votes to different nodes..."
echo "  - Submitting vote from voter1 to Node 1..."
go run cmd/submit-vote/main.go -voter voter1 -choice candidate-a -key voter1.key -peer /ip4/127.0.0.1/tcp/4001 2>&1 | grep -E "(Vote submitted|ERROR)" || true

sleep 1

echo "  - Submitting vote from voter2 to Node 2..."
go run cmd/submit-vote/main.go -voter voter2 -choice candidate-b -key voter2.key -peer /ip4/127.0.0.1/tcp/4002 2>&1 | grep -E "(Vote submitted|ERROR)" || true

sleep 1

echo "  - Submitting vote from voter3 to Node 3..."
go run cmd/submit-vote/main.go -voter voter3 -choice candidate-a -key voter3.key -peer /ip4/127.0.0.1/tcp/4003 2>&1 | grep -E "(Vote submitted|ERROR)" || true

echo ""
echo "Step 7: Waiting for block proposal and finalization..."
echo "  (Block interval is 5 seconds, waiting 10 seconds...)"
sleep 10

echo ""
echo "Step 8: Checking results..."
echo ""

# Check Node 1 logs
echo "Node 1 status:"
if grep -q "Block finalized" node1.log; then
    echo "  ✓ Block finalized"
    FINALIZED_HEIGHT=$(grep "Block finalized" node1.log | tail -1 | grep -oP 'height=\K\d+' || echo "unknown")
    echo "    Height: $FINALIZED_HEIGHT"
else
    echo "  ✗ No block finalized"
fi

if grep -q "Vote received" node1.log; then
    VOTE_COUNT=$(grep -c "Vote received" node1.log)
    echo "  ✓ Votes received: $VOTE_COUNT"
else
    echo "  ✗ No votes received"
fi

echo ""

# Check Node 2 logs
echo "Node 2 status:"
if grep -q "Block finalized" node2.log; then
    echo "  ✓ Block finalized"
    FINALIZED_HEIGHT=$(grep "Block finalized" node2.log | tail -1 | grep -oP 'height=\K\d+' || echo "unknown")
    echo "    Height: $FINALIZED_HEIGHT"
else
    echo "  ✗ No block finalized"
fi

if grep -q "Vote received" node2.log; then
    VOTE_COUNT=$(grep -c "Vote received" node2.log)
    echo "  ✓ Votes received: $VOTE_COUNT"
else
    echo "  ✗ No votes received"
fi

echo ""

# Check Node 3 logs
echo "Node 3 status:"
if grep -q "Block finalized" node3.log; then
    echo "  ✓ Block finalized"
    FINALIZED_HEIGHT=$(grep "Block finalized" node3.log | tail -1 | grep -oP 'height=\K\d+' || echo "unknown")
    echo "    Height: $FINALIZED_HEIGHT"
else
    echo "  ✗ No block finalized"
fi

if grep -q "Vote received" node3.log; then
    VOTE_COUNT=$(grep -c "Vote received" node3.log)
    echo "  ✓ Votes received: $VOTE_COUNT"
else
    echo "  ✗ No votes received"
fi

echo ""
echo "========================================="
echo "Test Complete"
echo "========================================="
echo ""
echo "Log files available for inspection:"
echo "  - node1.log"
echo "  - node2.log"
echo "  - node3.log"
echo ""
echo "To view logs in real-time, run:"
echo "  tail -f node1.log node2.log node3.log"
echo ""

# Keep nodes running for manual inspection if desired
read -p "Press Enter to shutdown nodes and cleanup..."
