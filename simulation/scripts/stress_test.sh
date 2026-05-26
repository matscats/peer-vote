#!/bin/bash

# Stress test - send many votes quickly to test block batching

NUM_VOTERS=${1:-20}
VOTE_INTERVAL=${2:-0.1}  # Very fast - 0.1 seconds between votes

echo "=========================================="
echo "Stress Test - Multiple Votes per Block"
echo "=========================================="
echo ""
echo "Configuration:"
echo "  Voters: $NUM_VOTERS"
echo "  Vote interval: ${VOTE_INTERVAL}s"
echo ""

cd "$(dirname "$0")"

# Clean up
echo "Cleaning up..."
pkill -9 -f 'bin/bootstrap' 2>/dev/null
pkill -9 -f 'bin/node' 2>/dev/null
pkill -9 -f 'submit-vote' 2>/dev/null
sleep 2

rm -rf ../data/* ../logs/*

# Build if needed
if [ ! -f "../bin/node" ]; then
    echo "Building binaries..."
    ./build.sh
fi

# Check voters
if [ ! -d "../keys/voters" ] || [ $(ls -1 ../keys/voters/*.key 2>/dev/null | wc -l) -lt $NUM_VOTERS ]; then
    echo "Generating voter keypairs..."
    ./generate_voters.sh $NUM_VOTERS
    ./update_eligibility.sh
fi

echo ""
echo "=========================================="
echo "Starting Network"
echo "=========================================="
echo ""

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
echo "Waiting for network to stabilize (10 seconds)..."
sleep 10

echo ""
echo "=========================================="
echo "Sending $NUM_VOTERS Votes Rapidly"
echo "=========================================="
echo ""

# Get node peer address
NODE_ID=$(grep "P2P Host created with ID:" ../logs/node1.log | awk '{print $NF}')
TARGET_PEER="/ip4/127.0.0.1/tcp/4001/p2p/${NODE_ID}"

if [ -z "$NODE_ID" ]; then
    echo "❌ Could not get node peer ID"
    pkill -9 -f 'bin/bootstrap'; pkill -9 -f 'bin/node'
    exit 1
fi

echo "Target peer: $TARGET_PEER"
echo ""

# Submit votes VERY QUICKLY
CANDIDATES=("candidate-a" "candidate-b" "candidate-c")
VOTE_COUNT=0

echo "Sending votes at ${VOTE_INTERVAL}s intervals..."
START_TIME=$(date +%s)

for key_file in ../keys/voters/*.key; do
    if [ $VOTE_COUNT -ge $NUM_VOTERS ]; then
        break
    fi
    
    VOTER_ID=$(basename "$key_file" .key)
    CANDIDATE=${CANDIDATES[$RANDOM % ${#CANDIDATES[@]}]}
    
    # Submit vote in background (don't wait)
    (cd ../.. && ./simulation/bin/submit-vote \
        -voter "$VOTER_ID" \
        -choice "$CANDIDATE" \
        -key "simulation/keys/voters/${VOTER_ID}.key" \
        -peer "$TARGET_PEER") > ../logs/vote_${VOTER_ID}.log 2>&1 &
    
    VOTE_COUNT=$((VOTE_COUNT + 1))
    
    # Very short sleep
    sleep $VOTE_INTERVAL
done

END_TIME=$(date +%s)
ELAPSED=$((END_TIME - START_TIME))

echo ""
echo "✅ All $NUM_VOTERS votes submitted in ${ELAPSED}s"
echo ""

# Wait for processing
echo "Waiting for votes to be processed (20 seconds)..."
sleep 20

echo ""
echo "=========================================="
echo "Results"
echo "=========================================="
echo ""

# Check how many votes were accepted
VOTES_ACCEPTED=$(grep "Vote submitted successfully" ../logs/vote_*.log | wc -l)
VOTES_IN_MEMPOOL=$(grep "added to mempool" ../logs/node1.log | wc -l)
BLOCKS_FINALIZED=$(grep "finalized successfully" ../logs/node1.log | wc -l)

echo "Votes submitted successfully: $VOTES_ACCEPTED"
echo "Votes added to mempool: $VOTES_IN_MEMPOOL"
echo "Blocks finalized: $BLOCKS_FINALIZED"
echo ""

# Analyze blockchain
echo "Analyzing blockchain..."
go run analyze_blocks.go ../data/data1

echo ""
echo "=========================================="
echo "Detailed Block Analysis"
echo "=========================================="
echo ""

# Show blocks with multiple votes
grep "Proposed new block" ../logs/node*.log | grep -v "with 0 votes" | sort

echo ""
echo "Logs available in: simulation/logs/"
echo ""
echo "To stop all nodes:"
echo "  pkill -9 -f 'bin/bootstrap'; pkill -9 -f 'bin/node'"
echo ""

# Stop nodes
echo "Stopping nodes..."
pkill -9 -f 'bin/bootstrap'
pkill -9 -f 'bin/node'
sleep 2

echo "✅ Test complete!"
