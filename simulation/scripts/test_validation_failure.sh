#!/bin/bash

# Test validation failures - simulate various failure scenarios

echo "=========================================="
echo "Validation Failure Test"
echo "=========================================="
echo ""

cd "$(dirname "$0")"

# Clean up
echo "Cleaning up..."
pkill -9 -f 'bin/bootstrap' 2>/dev/null
pkill -9 -f 'bin/node' 2>/dev/null
pkill -9 -f 'submit-vote' 2>/dev/null
sleep 2

rm -rf ../data/* ../logs/*

# Build
if [ ! -f "../bin/node" ]; then
    echo "Building binaries..."
    ./build.sh
fi

# Generate voters if needed
if [ ! -d "../keys/voters" ] || [ $(ls -1 ../keys/voters/*.key 2>/dev/null | wc -l) -lt 5 ]; then
    echo "Generating voter keypairs..."
    ./generate_voters.sh 5
    ./update_eligibility.sh
fi

echo ""
echo "=========================================="
echo "Test Scenario Selection"
echo "=========================================="
echo ""
echo "1. Invalid signature (tampered vote)"
echo "2. Ineligible voter (not in eligibility list)"
echo "3. Double voting (same voter votes twice)"
echo "4. Invalid block signature (tampered block)"
echo ""

SCENARIO=${1:-1}

case $SCENARIO in
    1)
        echo "Testing: Invalid Signature"
        TEST_TYPE="invalid_signature"
        ;;
    2)
        echo "Testing: Ineligible Voter"
        TEST_TYPE="ineligible_voter"
        ;;
    3)
        echo "Testing: Double Voting"
        TEST_TYPE="double_voting"
        ;;
    4)
        echo "Testing: Invalid Block"
        TEST_TYPE="invalid_block"
        ;;
    *)
        echo "Invalid scenario"
        exit 1
        ;;
esac

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

# Get node peer address
NODE_ID=$(grep "P2P Host created with ID:" ../logs/node1.log | awk '{print $NF}')
TARGET_PEER="/ip4/127.0.0.1/tcp/4001/p2p/${NODE_ID}"

if [ -z "$NODE_ID" ]; then
    echo "❌ Could not get node peer ID"
    pkill -9 -f 'bin/bootstrap'; pkill -9 -f 'bin/node'
    exit 1
fi

echo ""
echo "=========================================="
echo "Executing Test: $TEST_TYPE"
echo "=========================================="
echo ""

case $TEST_TYPE in
    "invalid_signature")
        echo "Scenario: Sending vote with tampered signature"
        echo ""
        
        # First send a valid vote
        echo "1. Sending valid vote from voter001..."
        (cd ../.. && ./simulation/bin/submit-vote \
            -voter "voter001" \
            -choice "candidate-a" \
            -key "simulation/keys/voters/voter001.key" \
            -peer "$TARGET_PEER") > ../logs/vote_voter001.log 2>&1 &
        
        sleep 3
        
        # Now we'll try to send a vote with invalid signature
        # We'll use voter002's key but claim to be voter003
        echo "2. Attempting to send vote with wrong signature (voter002 key, voter003 ID)..."
        echo "   This should FAIL validation..."
        
        # This will fail because the signature won't match the voterID
        (cd ../.. && ./simulation/bin/submit-vote \
            -voter "voter003" \
            -choice "candidate-b" \
            -key "simulation/keys/voters/voter002.key" \
            -peer "$TARGET_PEER") > ../logs/vote_invalid_sig.log 2>&1 &
        
        sleep 5
        ;;
        
    "ineligible_voter")
        echo "Scenario: Vote from ineligible voter"
        echo ""
        
        # Create a voter that's not in eligibility list
        echo "1. Creating voter key that's NOT in eligibility list..."
        go run ../../cmd/node/main.go -generate-key -key-path ../keys/ineligible_voter.key > /dev/null 2>&1
        
        echo "2. Attempting to vote with ineligible voter..."
        echo "   This should FAIL eligibility check..."
        
        (cd ../.. && ./simulation/bin/submit-vote \
            -voter "ineligible_voter" \
            -choice "candidate-a" \
            -key "simulation/keys/ineligible_voter.key" \
            -peer "$TARGET_PEER") > ../logs/vote_ineligible.log 2>&1 &
        
        sleep 5
        ;;
        
    "double_voting")
        echo "Scenario: Same voter tries to vote twice"
        echo ""
        
        echo "1. Sending first vote from voter001..."
        (cd ../.. && ./simulation/bin/submit-vote \
            -voter "voter001" \
            -choice "candidate-a" \
            -key "simulation/keys/voters/voter001.key" \
            -peer "$TARGET_PEER") > ../logs/vote_voter001_first.log 2>&1 &
        
        sleep 5
        
        echo "2. Waiting for first vote to be finalized..."
        sleep 10
        
        echo "3. Attempting second vote from voter001..."
        echo "   This should FAIL duplicate check..."
        
        (cd ../.. && ./simulation/bin/submit-vote \
            -voter "voter001" \
            -choice "candidate-b" \
            -key "simulation/keys/voters/voter001.key" \
            -peer "$TARGET_PEER") > ../logs/vote_voter001_second.log 2>&1 &
        
        sleep 5
        ;;
        
    "invalid_block")
        echo "Scenario: This would require modifying node code to send invalid block"
        echo "Skipping for now - would need to create malicious node"
        ;;
esac

echo ""
echo "Waiting for processing (10 seconds)..."
sleep 10

echo ""
echo "=========================================="
echo "Results"
echo "=========================================="
echo ""

# Check what happened
case $TEST_TYPE in
    "invalid_signature")
        echo "Valid vote (voter001):"
        if grep -q "Vote submitted successfully" ../logs/vote_voter001.log; then
            echo "  ✅ Submitted successfully"
        else
            echo "  ❌ Failed to submit"
            tail -3 ../logs/vote_voter001.log
        fi
        
        echo ""
        echo "Invalid signature vote:"
        if grep -q "Vote submitted successfully" ../logs/vote_invalid_sig.log; then
            echo "  ❌ PROBLEM: Invalid vote was accepted!"
        else
            echo "  ✅ Correctly rejected"
            echo "  Error:"
            tail -3 ../logs/vote_invalid_sig.log | grep -E "Failed|invalid|error" || tail -1 ../logs/vote_invalid_sig.log
        fi
        
        echo ""
        echo "Node logs (checking for validation errors):"
        grep -i "signature validation failed\|invalid signature" ../logs/node*.log | head -5
        ;;
        
    "ineligible_voter")
        echo "Ineligible voter attempt:"
        if grep -q "Vote submitted successfully" ../logs/vote_ineligible.log; then
            echo "  ⚠️  Vote was submitted to network"
        else
            echo "  ❌ Failed at submission"
            tail -3 ../logs/vote_ineligible.log
        fi
        
        echo ""
        echo "Node logs (checking for eligibility errors):"
        grep -i "not eligible\|eligibility" ../logs/node*.log | head -5
        ;;
        
    "double_voting")
        echo "First vote:"
        if grep -q "Vote submitted successfully" ../logs/vote_voter001_first.log; then
            echo "  ✅ Submitted successfully"
        else
            echo "  ❌ Failed"
        fi
        
        echo ""
        echo "Second vote (duplicate):"
        if grep -q "Vote submitted successfully" ../logs/vote_voter001_second.log; then
            echo "  ⚠️  Submitted to network"
        else
            echo "  ❌ Failed at submission"
        fi
        
        echo ""
        echo "Node logs (checking for duplicate errors):"
        grep -i "already voted\|duplicate" ../logs/node*.log | head -5
        ;;
esac

echo ""
echo "Blockchain analysis:"
go run analyze_blocks.go ../data/data1

echo ""
echo "=========================================="
echo "Full Node Logs"
echo "=========================================="
echo ""
echo "Errors in node logs:"
grep -i "error\|failed\|invalid" ../logs/node1.log | tail -10

echo ""
echo "Logs available in: simulation/logs/"
echo ""

# Stop nodes
echo "Stopping nodes..."
pkill -9 -f 'bin/bootstrap'
pkill -9 -f 'bin/node'
sleep 2

echo "✅ Test complete!"
