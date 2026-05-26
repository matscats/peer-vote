#!/bin/bash

# Quick test script to verify the system is working
# This runs a minimal test with 3 voters

echo "=========================================="
echo "Quick System Test"
echo "=========================================="
echo ""

cd "$(dirname "$0")"

# Clean up
echo "Cleaning up old data..."
rm -rf ../data/* ../logs/*

# Build
echo "Building binaries..."
./build.sh
if [ $? -ne 0 ]; then
    echo "❌ Build failed"
    exit 1
fi
echo "✅ Build successful"
echo ""

# Check voters
if [ ! -d "../keys/voters" ] || [ $(ls -1 ../keys/voters/*.key 2>/dev/null | wc -l) -lt 3 ]; then
    echo "Generating voter keypairs..."
    ./generate_voters.sh 3
    ./update_eligibility.sh
fi

echo "=========================================="
echo "Starting Network"
echo "=========================================="
echo ""

# Kill any existing processes first
echo "Cleaning up any existing processes..."
pkill -9 -f 'bin/bootstrap' 2>/dev/null
pkill -9 -f 'bin/node' 2>/dev/null
pkill -9 -f 'submit-vote' 2>/dev/null
sleep 3

# Verify no processes are running
if pgrep -f 'bin/bootstrap' > /dev/null || pgrep -f 'bin/node' > /dev/null; then
    echo "❌ Failed to kill existing processes"
    ps aux | grep -E "bin/bootstrap|bin/node" | grep -v grep
    exit 1
fi

# Start bootstrap
echo "Starting bootstrap node..."
../bin/bootstrap -port 4000 > ../logs/bootstrap.log 2>&1 &
BOOTSTRAP_PID=$!
sleep 3

# Get bootstrap address
BOOTSTRAP_ADDR=$(grep "/ip4/127.0.0.1/tcp/4000/p2p/" ../logs/bootstrap.log | head -1 | sed 's/.*\(\/ip4\/127\.0\.0\.1\/tcp\/4000\/p2p\/[^ ]*\).*/\1/')

if [ -z "$BOOTSTRAP_ADDR" ]; then
    echo "❌ Could not get bootstrap address"
    kill $BOOTSTRAP_PID 2>/dev/null
    exit 1
fi

echo "✅ Bootstrap running: $BOOTSTRAP_ADDR"

# Update configs
for config in ../configs/config{1,2,3}.json; do
    if command -v jq &> /dev/null; then
        jq ".bootstrap_peers = [\"$BOOTSTRAP_ADDR\"]" $config > ${config}.tmp && mv ${config}.tmp $config
    else
        sed -i.bak "s|\"bootstrap_peers\": \[.*\]|\"bootstrap_peers\": [\"$BOOTSTRAP_ADDR\"]|" $config
        rm -f ${config}.bak
    fi
done

# Start nodes
echo "Starting validator nodes..."
for i in 1 2 3; do
    (cd .. && ./bin/node -config configs/config${i}.json) > ../logs/node${i}.log 2>&1 &
    echo "  Node $i started"
    sleep 2
done

echo ""
echo "Waiting for network to stabilize (15 seconds)..."
sleep 15

echo ""
echo "=========================================="
echo "Submitting Test Votes"
echo "=========================================="
echo ""

# Get peer IDs for all nodes
NODE1_ID=$(grep "P2P Host created with ID:" ../logs/node1.log | awk '{print $NF}')
NODE2_ID=$(grep "P2P Host created with ID:" ../logs/node2.log | awk '{print $NF}')
NODE3_ID=$(grep "P2P Host created with ID:" ../logs/node3.log | awk '{print $NF}')

if [ -z "$NODE1_ID" ]; then
    echo "❌ Could not get node peer IDs"
    echo "Node1 log:"
    tail -20 ../logs/node1.log
    pkill -f 'bin/bootstrap'; pkill -f 'bin/node'
    exit 1
fi

# Create array of target peers
TARGET_PEERS=(
    "/ip4/127.0.0.1/tcp/4001/p2p/${NODE1_ID}"
    "/ip4/127.0.0.1/tcp/4002/p2p/${NODE2_ID}"
    "/ip4/127.0.0.1/tcp/4003/p2p/${NODE3_ID}"
)

echo "Target peers:"
for i in 0 1 2; do
    echo "  Node $((i+1)): ${TARGET_PEERS[$i]}"
done
echo ""

# Submit 3 test votes to different nodes
CANDIDATES=("candidate-a" "candidate-b" "candidate-c")
for i in 1 2 3; do
    VOTER_ID=$(printf "voter%03d" $i)
    CANDIDATE=${CANDIDATES[$((i-1))]}
    TARGET_PEER=${TARGET_PEERS[$((i-1))]}
    
    echo "[$i/3] $VOTER_ID voting for $CANDIDATE (via node$i)..."
    
    (cd ../.. && ./simulation/bin/submit-vote \
        -voter "$VOTER_ID" \
        -choice "$CANDIDATE" \
        -key "simulation/keys/voters/${VOTER_ID}.key" \
        -peer "$TARGET_PEER") > ../logs/vote_${VOTER_ID}.log 2>&1 &
    
    sleep 2
done

echo ""
echo "All votes submitted!"
echo ""

# Wait for processing
echo "Waiting for votes to be processed (30 seconds)..."
sleep 30

echo ""
echo "=========================================="
echo "Test Results"
echo "=========================================="
echo ""

# Check logs
echo "Checking node1 log for votes..."
VOTES_IN_MEMPOOL=$(grep "added to mempool" ../logs/node1.log | wc -l)
BLOCKS_FINALIZED=$(grep "finalized successfully" ../logs/node1.log | wc -l)

echo "  Votes in mempool: $VOTES_IN_MEMPOOL"
echo "  Blocks finalized: $BLOCKS_FINALIZED"
echo ""

# Check vote logs
echo "Checking vote submission logs..."
for i in 1 2 3; do
    VOTER_ID=$(printf "voter%03d" $i)
    if grep -q "Vote submitted successfully" ../logs/vote_${VOTER_ID}.log; then
        echo "  ✅ $VOTER_ID: submitted successfully"
    else
        echo "  ❌ $VOTER_ID: failed"
        echo "     Error: $(tail -1 ../logs/vote_${VOTER_ID}.log)"
    fi
done

echo ""
echo "=========================================="
echo "Summary"
echo "=========================================="
echo ""

if [ $VOTES_IN_MEMPOOL -ge 1 ] && [ $BLOCKS_FINALIZED -ge 1 ]; then
    echo "✅ TEST PASSED"
    echo "   - Votes were accepted"
    echo "   - Blocks were finalized"
    echo "   - System is working!"
else
    echo "⚠️  TEST INCOMPLETE"
    echo "   - Check logs in simulation/logs/"
    echo "   - Votes in mempool: $VOTES_IN_MEMPOOL"
    echo "   - Blocks finalized: $BLOCKS_FINALIZED"
fi

echo ""
echo "Logs available in: simulation/logs/"
echo ""
echo "To stop all nodes:"
echo "  pkill -f 'bin/bootstrap'; pkill -f 'bin/node'"
echo ""

# Keep running
echo "Press Ctrl+C to stop all nodes..."
trap "echo 'Stopping...'; pkill -f 'bin/bootstrap'; pkill -f 'bin/node'; exit 0" INT
wait
