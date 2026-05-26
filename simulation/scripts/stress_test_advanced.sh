#!/bin/bash

# Advanced Stress Test for Blockchain Voting System
# Tests network capacity, throughput, latency, and reliability

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# Configuration
NUM_VOTERS=${1:-50}           # Number of concurrent voters
VOTE_RATE=${2:-10}            # Votes per second (0 = as fast as possible)
TEST_DURATION=${3:-30}        # Duration in seconds
CANDIDATES=("candidate-a" "candidate-b" "candidate-c")

echo "=========================================="
echo "Advanced Stress Test"
echo "=========================================="
echo ""
echo "Configuration:"
echo "  Concurrent voters: $NUM_VOTERS"
echo "  Target rate: $VOTE_RATE votes/sec (0 = unlimited)"
echo "  Test duration: ${TEST_DURATION}s"
echo "  Total expected votes: $((NUM_VOTERS))"
echo ""

# Calculate vote interval
if [ $VOTE_RATE -eq 0 ]; then
    VOTE_INTERVAL=0
    echo "  Mode: Maximum throughput"
else
    VOTE_INTERVAL=$(echo "scale=3; 1.0 / $VOTE_RATE" | bc)
    echo "  Vote interval: ${VOTE_INTERVAL}s"
fi
echo ""

# Clean up old data
echo "Cleaning up old simulation data..."
rm -rf ../data/* ../logs/*
echo ""

# Check if binaries exist
if [ ! -f "../bin/bootstrap" ] || [ ! -f "../bin/node" ] || [ ! -f "../bin/submit-vote" ]; then
    echo "Building binaries..."
    ./build.sh
    if [ $? -ne 0 ]; then
        echo "ERROR: Build failed"
        exit 1
    fi
fi

# Check if voters exist
if [ ! -d "../keys/voters" ] || [ $(ls -1 ../keys/voters/*.key 2>/dev/null | wc -l) -lt $NUM_VOTERS ]; then
    echo "Generating $NUM_VOTERS voter keypairs..."
    ./generate_voters.sh $NUM_VOTERS
    ./update_eligibility.sh
fi

echo "=========================================="
echo "Starting Network"
echo "=========================================="
echo ""

# Start Bootstrap Node
echo "Starting bootstrap node..."
../bin/bootstrap -port 4000 > ../logs/bootstrap.log 2>&1 &
BOOTSTRAP_PID=$!
sleep 3

# Get bootstrap address
BOOTSTRAP_ADDR=$(grep "/ip4/127.0.0.1/tcp/4000/p2p/" ../logs/bootstrap.log | head -1 | sed 's/.*\(\/ip4\/127\.0\.0\.1\/tcp\/4000\/p2p\/[^ ]*\).*/\1/')

if [ -z "$BOOTSTRAP_ADDR" ]; then
    echo "ERROR: Could not get bootstrap address"
    kill $BOOTSTRAP_PID 2>/dev/null
    exit 1
fi

echo "Bootstrap address: $BOOTSTRAP_ADDR"
echo ""

# Update configs with bootstrap address
for config in ../configs/config{1,2,3}.json; do
    if command -v jq &> /dev/null; then
        jq ".bootstrap_peers = [\"$BOOTSTRAP_ADDR\"]" $config > ${config}.tmp && mv ${config}.tmp $config
    else
        sed -i.bak "s|\"bootstrap_peers\": \[.*\]|\"bootstrap_peers\": [\"$BOOTSTRAP_ADDR\"]|" $config
        rm -f ${config}.bak
    fi
done

# Start Validator Nodes
echo "Starting validator nodes..."
for i in 1 2 3; do
    (cd .. && ./bin/node -config configs/config${i}.json) > ../logs/node${i}.log 2>&1 &
    NODE_PID=$!
    echo "  Node $i PID: $NODE_PID"
    sleep 2
done

echo ""
echo "Waiting for network to stabilize (10 seconds)..."
sleep 10

# Get node peer IDs
NODE1_ID=$(grep "P2P Host created with ID:" ../logs/node1.log | awk '{print $NF}')
NODE2_ID=$(grep "P2P Host created with ID:" ../logs/node2.log | awk '{print $NF}')
NODE3_ID=$(grep "P2P Host created with ID:" ../logs/node3.log | awk '{print $NF}')

TARGET_PEERS=(
    "/ip4/127.0.0.1/tcp/4001/p2p/${NODE1_ID}"
    "/ip4/127.0.0.1/tcp/4002/p2p/${NODE2_ID}"
    "/ip4/127.0.0.1/tcp/4003/p2p/${NODE3_ID}"
)

echo ""
echo "=========================================="
echo "Starting Stress Test"
echo "=========================================="
echo ""
echo "Submitting $NUM_VOTERS votes..."
echo ""

# Record start time
START_TIME=$(date +%s)
START_TIME_MS=$(date +%s%3N)

# Submit votes in parallel
VOTE_COUNT=0
for key_file in ../keys/voters/*.key; do
    if [ $VOTE_COUNT -ge $NUM_VOTERS ]; then
        break
    fi
    
    VOTER_ID=$(basename "$key_file" .key)
    
    # Randomly select a candidate
    CANDIDATE=${CANDIDATES[$RANDOM % ${#CANDIDATES[@]}]}
    
    # Distribute votes across validators (round-robin)
    TARGET_INDEX=$((VOTE_COUNT % 3))
    TARGET_PEER=${TARGET_PEERS[$TARGET_INDEX]}
    
    # Submit vote in background
    (cd ../.. && ./simulation/bin/submit-vote \
        -voter "$VOTER_ID" \
        -choice "$CANDIDATE" \
        -key "simulation/keys/voters/${VOTER_ID}.key" \
        -peer "$TARGET_PEER") > ../logs/vote_${VOTER_ID}.log 2>&1 &
    
    VOTE_COUNT=$((VOTE_COUNT + 1))
    
    # Progress indicator
    if [ $((VOTE_COUNT % 10)) -eq 0 ]; then
        echo "  Submitted: $VOTE_COUNT/$NUM_VOTERS votes"
    fi
    
    # Rate limiting
    if [ "$VOTE_INTERVAL" != "0" ] && [ "$VOTE_INTERVAL" != "" ]; then
        sleep $VOTE_INTERVAL
    fi
done

SUBMIT_END_TIME=$(date +%s)
SUBMIT_DURATION=$((SUBMIT_END_TIME - START_TIME))

echo ""
echo "All $NUM_VOTERS votes submitted in ${SUBMIT_DURATION}s"
echo "Submission rate: $(echo "scale=2; $NUM_VOTERS / $SUBMIT_DURATION" | bc) votes/sec"
echo ""

# Wait for votes to be processed
echo "Waiting for votes to be finalized (${TEST_DURATION}s)..."
sleep $TEST_DURATION

END_TIME=$(date +%s)
END_TIME_MS=$(date +%s%3N)
TOTAL_DURATION=$((END_TIME - START_TIME))

echo ""
echo "=========================================="
echo "Analyzing Results"
echo "=========================================="
echo ""

# Count votes in blockchain
VOTES_IN_BLOCKCHAIN=$(grep -h "finalized successfully" ../logs/node1.log | tail -1 | grep -o "voted count: [0-9]*" | awk '{print $3}')
if [ -z "$VOTES_IN_BLOCKCHAIN" ]; then
    VOTES_IN_BLOCKCHAIN=0
fi

# Count blocks finalized
BLOCKS_FINALIZED=$(grep -h "finalized successfully" ../logs/node1.log | wc -l | tr -d ' ')

# Count successful vote submissions
VOTES_SUBMITTED=$(grep -l "Vote submitted successfully" ../logs/vote_*.log 2>/dev/null | wc -l | tr -d ' ')

# Count failed submissions
VOTES_FAILED=$(grep -l "Failed" ../logs/vote_*.log 2>/dev/null | wc -l | tr -d ' ')

# Calculate metrics
VOTE_LOSS=$((VOTES_SUBMITTED - VOTES_IN_BLOCKCHAIN))
VOTE_LOSS_PERCENT=$(echo "scale=2; ($VOTE_LOSS * 100) / $VOTES_SUBMITTED" | bc)

THROUGHPUT=$(echo "scale=2; $VOTES_IN_BLOCKCHAIN / $TOTAL_DURATION" | bc)

# Average votes per block
if [ $BLOCKS_FINALIZED -gt 0 ]; then
    AVG_VOTES_PER_BLOCK=$(echo "scale=2; $VOTES_IN_BLOCKCHAIN / $BLOCKS_FINALIZED" | bc)
else
    AVG_VOTES_PER_BLOCK=0
fi

# Calculate latency (time from first vote to last finalization)
FIRST_VOTE_TIME=$(grep "Vote submitted successfully" ../logs/vote_*.log 2>/dev/null | head -1 | awk '{print $1, $2}')
LAST_FINALIZATION_TIME=$(grep "finalized successfully" ../logs/node1.log | tail -1 | awk '{print $1, $2}')

if [ -n "$FIRST_VOTE_TIME" ] && [ -n "$LAST_FINALIZATION_TIME" ]; then
    FIRST_VOTE_TS=$(date -j -f "%Y/%m/%d %H:%M:%S" "$FIRST_VOTE_TIME" +%s 2>/dev/null || echo $START_TIME)
    LAST_FINAL_TS=$(date -j -f "%Y/%m/%d %H:%M:%S" "$LAST_FINALIZATION_TIME" +%s 2>/dev/null || echo $END_TIME)
    LATENCY=$((LAST_FINAL_TS - FIRST_VOTE_TS))
else
    LATENCY=$TOTAL_DURATION
fi

# Generate report
echo "📊 STRESS TEST RESULTS"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "Test Configuration:"
echo "  Duration: ${TOTAL_DURATION}s"
echo "  Target voters: $NUM_VOTERS"
echo "  Target rate: $VOTE_RATE votes/sec"
echo ""
echo "Submission Results:"
echo "  ✅ Submitted successfully: $VOTES_SUBMITTED"
echo "  ❌ Failed submissions: $VOTES_FAILED"
echo "  ⏱️  Submission time: ${SUBMIT_DURATION}s"
echo "  📈 Actual submission rate: $(echo "scale=2; $VOTES_SUBMITTED / $SUBMIT_DURATION" | bc) votes/sec"
echo ""
echo "Blockchain Results:"
echo "  📦 Blocks finalized: $BLOCKS_FINALIZED"
echo "  🗳️  Votes in blockchain: $VOTES_IN_BLOCKCHAIN"
echo "  📊 Average votes/block: $AVG_VOTES_PER_BLOCK"
echo ""
echo "Performance Metrics:"
echo "  🚀 Throughput: $THROUGHPUT votes/sec"
echo "  ⏱️  End-to-end latency: ${LATENCY}s"
echo "  📉 Vote loss: $VOTE_LOSS votes ($VOTE_LOSS_PERCENT%)"
echo ""

# Performance assessment
if [ $VOTE_LOSS -eq 0 ]; then
    echo "✅ EXCELLENT: No votes lost!"
elif [ $VOTE_LOSS -lt 5 ]; then
    echo "✅ GOOD: Minimal vote loss (<5)"
elif [ $VOTE_LOSS -lt 10 ]; then
    echo "⚠️  ACCEPTABLE: Some vote loss (<10)"
else
    echo "❌ POOR: Significant vote loss (>10)"
fi

echo ""

# Network statistics
echo "Network Statistics:"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
for i in 1 2 3; do
    PEERS=$(grep "Connected peers:" ../logs/node${i}.log | tail -1 | grep -o "Connected peers: [0-9]*" | awk '{print $3}')
    echo "  Node $i: ${PEERS:-0} peers connected"
done

echo ""
echo "Block Distribution:"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Count blocks proposed by each node
for i in 1 2 3; do
    PROPOSED=$(grep "Proposed new block" ../logs/node${i}.log | wc -l | tr -d ' ')
    echo "  Node $i proposed: $PROPOSED blocks"
done

echo ""
echo "Vote Distribution by Candidate:"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

for candidate in "${CANDIDATES[@]}"; do
    COUNT=$(grep -l "choice=$candidate" ../logs/vote_*.log 2>/dev/null | wc -l | tr -d ' ')
    echo "  $candidate: $COUNT votes"
done

echo ""
echo "=========================================="
echo "Test Complete"
echo "=========================================="
echo ""
echo "Logs available in: ../logs/"
echo "Blockchain data in: ../data/"
echo ""
echo "To analyze blocks in detail:"
echo "  go run analyze_blocks.go"
echo ""
echo "To visualize network:"
echo "  ./visualize.sh"
echo ""
echo "To stop all nodes:"
echo "  pkill -f 'bin/bootstrap'; pkill -f 'bin/node'"
echo ""

# Keep nodes running for analysis
read -p "Press Enter to stop all nodes and exit..."

# Cleanup
echo "Stopping all nodes..."
pkill -f 'bin/bootstrap' 2>/dev/null || true
pkill -f 'bin/node' 2>/dev/null || true

echo "Done!"
