#!/bin/bash

# Progressive Load Test
# Gradually increases load to find system limits

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo "=========================================="
echo "Progressive Load Test"
echo "=========================================="
echo ""
echo "This test will gradually increase load to find system limits"
echo ""

# Test configurations
LOAD_LEVELS=(10 25 50 100 200)
DURATION=20  # seconds per test

# Results file
RESULTS_FILE="../logs/progressive_load_results.txt"
mkdir -p ../logs
echo "Progressive Load Test Results - $(date)" > $RESULTS_FILE
echo "========================================" >> $RESULTS_FILE
echo "" >> $RESULTS_FILE

# Build if needed
if [ ! -f "../bin/bootstrap" ] || [ ! -f "../bin/node" ]; then
    echo "Building binaries..."
    ./build.sh
fi

# Generate voters if needed
MAX_VOTERS=${LOAD_LEVELS[-1]}
if [ ! -d "../keys/voters" ] || [ $(ls -1 ../keys/voters/*.key 2>/dev/null | wc -l) -lt $MAX_VOTERS ]; then
    echo "Generating $MAX_VOTERS voter keypairs..."
    ./generate_voters.sh $MAX_VOTERS
    ./update_eligibility.sh
fi

echo "Starting network..."
echo ""

# Start bootstrap
../bin/bootstrap -port 4000 > ../logs/bootstrap.log 2>&1 &
BOOTSTRAP_PID=$!
sleep 3

BOOTSTRAP_ADDR=$(grep "/ip4/127.0.0.1/tcp/4000/p2p/" ../logs/bootstrap.log | head -1 | sed 's/.*\(\/ip4\/127\.0\.0\.1\/tcp\/4000\/p2p\/[^ ]*\).*/\1/')

# Update configs
for config in ../configs/config{1,2,3}.json; do
    if command -v jq &> /dev/null; then
        jq ".bootstrap_peers = [\"$BOOTSTRAP_ADDR\"]" $config > ${config}.tmp && mv ${config}.tmp $config
    else
        sed -i.bak "s|\"bootstrap_peers\": \[.*\]|\"bootstrap_peers\": [\"$BOOTSTRAP_ADDR\"]|" $config
        rm -f ${config}.bak
    fi
done

# Start validators
for i in 1 2 3; do
    (cd .. && ./bin/node -config configs/config${i}.json) > ../logs/node${i}.log 2>&1 &
    sleep 2
done

echo "Network started. Waiting for stabilization..."
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
echo "Running Progressive Load Tests"
echo "=========================================="
echo ""

# Run tests at each load level
for NUM_VOTERS in "${LOAD_LEVELS[@]}"; do
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "Testing with $NUM_VOTERS concurrent voters"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo ""
    
    # Record initial state
    INITIAL_BLOCKS=$(grep -h "finalized successfully" ../logs/node1.log | wc -l | tr -d ' ')
    INITIAL_VOTES=$(grep -h "finalized successfully" ../logs/node1.log | tail -1 | grep -o "voted count: [0-9]*" | awk '{print $3}')
    if [ -z "$INITIAL_VOTES" ]; then
        INITIAL_VOTES=0
    fi
    
    START_TIME=$(date +%s)
    
    # Submit votes
    echo "Submitting $NUM_VOTERS votes..."
    VOTE_COUNT=0
    for key_file in ../keys/voters/*.key; do
        if [ $VOTE_COUNT -ge $NUM_VOTERS ]; then
            break
        fi
        
        VOTER_ID=$(basename "$key_file" .key)
        CANDIDATE="candidate-$((RANDOM % 3 + 1))"
        TARGET_INDEX=$((VOTE_COUNT % 3))
        TARGET_PEER=${TARGET_PEERS[$TARGET_INDEX]}
        
        (cd ../.. && ./simulation/bin/submit-vote \
            -voter "$VOTER_ID" \
            -choice "$CANDIDATE" \
            -key "simulation/keys/voters/${VOTER_ID}.key" \
            -peer "$TARGET_PEER") > ../logs/vote_${VOTER_ID}_load${NUM_VOTERS}.log 2>&1 &
        
        VOTE_COUNT=$((VOTE_COUNT + 1))
    done
    
    SUBMIT_END=$(date +%s)
    SUBMIT_DURATION=$((SUBMIT_END - START_TIME))
    
    echo "  Votes submitted in ${SUBMIT_DURATION}s"
    echo "  Waiting ${DURATION}s for processing..."
    sleep $DURATION
    
    END_TIME=$(date +%s)
    TOTAL_DURATION=$((END_TIME - START_TIME))
    
    # Measure results
    FINAL_BLOCKS=$(grep -h "finalized successfully" ../logs/node1.log | wc -l | tr -d ' ')
    FINAL_VOTES=$(grep -h "finalized successfully" ../logs/node1.log | tail -1 | grep -o "voted count: [0-9]*" | awk '{print $3}')
    if [ -z "$FINAL_VOTES" ]; then
        FINAL_VOTES=0
    fi
    
    BLOCKS_CREATED=$((FINAL_BLOCKS - INITIAL_BLOCKS))
    VOTES_FINALIZED=$((FINAL_VOTES - INITIAL_VOTES))
    
    VOTES_SUBMITTED=$(grep -l "Vote submitted successfully" ../logs/vote_*_load${NUM_VOTERS}.log 2>/dev/null | wc -l | tr -d ' ')
    VOTE_LOSS=$((VOTES_SUBMITTED - VOTES_FINALIZED))
    
    THROUGHPUT=$(echo "scale=2; $VOTES_FINALIZED / $TOTAL_DURATION" | bc)
    
    if [ $BLOCKS_CREATED -gt 0 ]; then
        AVG_VOTES_PER_BLOCK=$(echo "scale=2; $VOTES_FINALIZED / $BLOCKS_CREATED" | bc)
    else
        AVG_VOTES_PER_BLOCK=0
    fi
    
    # Display results
    echo ""
    echo "  Results:"
    echo "    Votes submitted: $VOTES_SUBMITTED"
    echo "    Votes finalized: $VOTES_FINALIZED"
    echo "    Vote loss: $VOTE_LOSS"
    echo "    Blocks created: $BLOCKS_CREATED"
    echo "    Avg votes/block: $AVG_VOTES_PER_BLOCK"
    echo "    Throughput: $THROUGHPUT votes/sec"
    echo ""
    
    # Save to results file
    echo "Load Level: $NUM_VOTERS voters" >> $RESULTS_FILE
    echo "  Duration: ${TOTAL_DURATION}s" >> $RESULTS_FILE
    echo "  Votes submitted: $VOTES_SUBMITTED" >> $RESULTS_FILE
    echo "  Votes finalized: $VOTES_FINALIZED" >> $RESULTS_FILE
    echo "  Vote loss: $VOTE_LOSS" >> $RESULTS_FILE
    echo "  Blocks created: $BLOCKS_CREATED" >> $RESULTS_FILE
    echo "  Avg votes/block: $AVG_VOTES_PER_BLOCK" >> $RESULTS_FILE
    echo "  Throughput: $THROUGHPUT votes/sec" >> $RESULTS_FILE
    echo "" >> $RESULTS_FILE
    
    # Status
    if [ $VOTE_LOSS -eq 0 ]; then
        echo "  ✅ Status: EXCELLENT - No vote loss"
    elif [ $VOTE_LOSS -lt 5 ]; then
        echo "  ✅ Status: GOOD - Minimal vote loss"
    elif [ $VOTE_LOSS -lt 10 ]; then
        echo "  ⚠️  Status: ACCEPTABLE - Some vote loss"
    else
        echo "  ❌ Status: POOR - Significant vote loss"
        echo ""
        echo "  ⚠️  System limit may have been reached"
    fi
    
    echo ""
    
    # Clean up vote logs for this level
    rm -f ../logs/vote_*_load${NUM_VOTERS}.log
    
    # Short pause between tests
    sleep 5
done

echo "=========================================="
echo "Progressive Load Test Complete"
echo "=========================================="
echo ""
echo "Results saved to: $RESULTS_FILE"
echo ""

# Display summary
echo "Summary:"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
cat $RESULTS_FILE | grep -E "Load Level|Throughput|Vote loss"
echo ""

echo "To stop all nodes:"
echo "  pkill -f 'bin/bootstrap'; pkill -f 'bin/node'"
echo ""

read -p "Press Enter to stop all nodes and exit..."

# Cleanup
pkill -f 'bin/bootstrap' 2>/dev/null || true
pkill -f 'bin/node' 2>/dev/null || true

echo "Done!"
