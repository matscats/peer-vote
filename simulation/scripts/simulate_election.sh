#!/bin/bash

# Complete election simulation script
# This script:
# 1. Starts bootstrap node and validator nodes
# 2. Waits for network to stabilize
# 3. Simulates voters submitting votes
# 4. Monitors block finalization
# 5. Analyzes results

VOTERS_DIR="../keys/voters"
NUM_VOTERS=${1:-10}
VOTE_INTERVAL=${2:-2}  # Seconds between votes
CANDIDATES=("candidate-a" "candidate-b" "candidate-c")

echo "=========================================="
echo "Election Simulation"
echo "=========================================="
echo ""
echo "Configuration:"
echo "  Voters: $NUM_VOTERS"
echo "  Vote interval: ${VOTE_INTERVAL}s"
echo "  Candidates: ${CANDIDATES[@]}"
echo ""

cd "$(dirname "$0")"

# Clean up old data
echo "Cleaning up old simulation data..."
rm -rf ../data/* ../logs/*
echo ""

# Check if binaries exist
if [ ! -f "../bin/bootstrap" ] || [ ! -f "../bin/node" ]; then
    echo "Building binaries..."
    ./build.sh
    if [ $? -ne 0 ]; then
        echo "ERROR: Build failed"
        exit 1
    fi
fi

# Check if voters exist
if [ ! -d "$VOTERS_DIR" ] || [ $(ls -1 "$VOTERS_DIR"/*.key 2>/dev/null | wc -l) -lt $NUM_VOTERS ]; then
    echo "Generating voter keypairs..."
    ./generate_voters.sh $NUM_VOTERS
    ./update_eligibility.sh
fi

echo "=========================================="
echo "Starting Blockchain Network"
echo "=========================================="
echo ""

# Start Bootstrap Node
echo "Starting bootstrap node..."
../bin/bootstrap -port 4000 > ../logs/bootstrap.log 2>&1 &
BOOTSTRAP_PID=$!
echo "Bootstrap PID: $BOOTSTRAP_PID"
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
    echo "Node $i PID: $NODE_PID"
    sleep 2
done

echo ""
echo "Waiting for network to stabilize (10 seconds)..."
sleep 10

echo ""
echo "=========================================="
echo "Submitting Votes"
echo "=========================================="
echo ""

# Get peer IDs for all nodes
NODE1_ID=$(grep "P2P Host created with ID:" ../logs/node1.log | awk '{print $NF}')
NODE2_ID=$(grep "P2P Host created with ID:" ../logs/node2.log | awk '{print $NF}')
NODE3_ID=$(grep "P2P Host created with ID:" ../logs/node3.log | awk '{print $NF}')

# Create array of target peers
TARGET_PEERS=(
    "/ip4/127.0.0.1/tcp/4001/p2p/${NODE1_ID}"
    "/ip4/127.0.0.1/tcp/4002/p2p/${NODE2_ID}"
    "/ip4/127.0.0.1/tcp/4003/p2p/${NODE3_ID}"
)

echo "Target peers for votes:"
echo "  Node 1: ${TARGET_PEERS[0]}"
echo "  Node 2: ${TARGET_PEERS[1]}"
echo "  Node 3: ${TARGET_PEERS[2]}"
echo ""

# Submit votes from simulation directory
VOTE_COUNT=0
for key_file in "$VOTERS_DIR"/*.key; do
    if [ $VOTE_COUNT -ge $NUM_VOTERS ]; then
        break
    fi
    
    VOTER_ID=$(basename "$key_file" .key)
    
    # Randomly select a candidate
    CANDIDATE=${CANDIDATES[$RANDOM % ${#CANDIDATES[@]}]}
    
    # Distribute votes across validators (round-robin)
    TARGET_INDEX=$((VOTE_COUNT % 3))
    TARGET_PEER=${TARGET_PEERS[$TARGET_INDEX]}
    
    echo "[$((VOTE_COUNT + 1))/$NUM_VOTERS] $VOTER_ID voting for $CANDIDATE (via node$((TARGET_INDEX + 1)))..."
    
    # Submit vote (run from project root)
    (cd ../.. && ./simulation/bin/submit-vote -voter "$VOTER_ID" -choice "$CANDIDATE" -key "simulation/keys/voters/${VOTER_ID}.key" -peer "$TARGET_PEER") > ../logs/vote_${VOTER_ID}.log 2>&1 &
    
    VOTE_COUNT=$((VOTE_COUNT + 1))
    
    # Wait between votes
    sleep $VOTE_INTERVAL
done

echo ""
echo "All votes submitted!"
echo ""

# Wait for votes to be processed
echo "Waiting for votes to be finalized (20 seconds)..."
sleep 20

echo ""
echo "=========================================="
echo "Election Results"
echo "=========================================="
echo ""

# Analyze results
./analyze_results.sh

echo ""
echo "=========================================="
echo "Simulation Complete"
echo "=========================================="
echo ""
echo "Logs available in: ../logs/"
echo "Blockchain data in: ../data/"
echo ""
echo "To stop all nodes, run:"
echo "  pkill -f 'bin/bootstrap'; pkill -f 'bin/node'"
echo ""

# Keep nodes running
echo "Nodes are still running. Press Ctrl+C to stop."
trap "echo 'Stopping all nodes...'; pkill -f 'bin/bootstrap'; pkill -f 'bin/node'; exit 0" INT
wait
