#!/bin/bash

# Malicious Behavior Test
# Tests system's ability to detect and reject malicious actions

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo "=========================================="
echo "Malicious Behavior Test"
echo "=========================================="
echo ""
echo "Testing system's defense against:"
echo "  1. Double voting attempts"
echo "  2. Invalid signature attacks"
echo "  3. Replay attacks"
echo "  4. Unauthorized voter attempts"
echo ""

# Clean up
rm -rf ../data/* ../logs/*

# Build if needed
if [ ! -f "../bin/bootstrap" ] || [ ! -f "../bin/node" ]; then
    ./build.sh
fi

# Generate voters
if [ ! -d "../keys/voters" ] || [ $(ls -1 ../keys/voters/*.key 2>/dev/null | wc -l) -lt 5 ]; then
    ./generate_voters.sh 5
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

sleep 10

NODE1_ID=$(grep "P2P Host created with ID:" ../logs/node1.log | awk '{print $NF}')
TARGET_PEER="/ip4/127.0.0.1/tcp/4001/p2p/${NODE1_ID}"

echo "Network ready"
echo ""

# Test 1: Double Voting
echo "=========================================="
echo "Test 1: Double Voting Attack"
echo "=========================================="
echo ""
echo "Attempting to vote twice with same voter..."

VOTER_ID="voter001"
CANDIDATE="candidate-a"

echo "  First vote..."
(cd ../.. && ./simulation/bin/submit-vote \
    -voter "$VOTER_ID" \
    -choice "$CANDIDATE" \
    -key "simulation/keys/voters/${VOTER_ID}.key" \
    -peer "$TARGET_PEER") > ../logs/vote_${VOTER_ID}_first.log 2>&1

sleep 2

echo "  Second vote (should be rejected)..."
(cd ../.. && ./simulation/bin/submit-vote \
    -voter "$VOTER_ID" \
    -choice "candidate-b" \
    -key "simulation/keys/voters/${VOTER_ID}.key" \
    -peer "$TARGET_PEER") > ../logs/vote_${VOTER_ID}_second.log 2>&1

sleep 5

# Check results
FIRST_SUCCESS=$(grep -c "Vote submitted successfully" ../logs/vote_${VOTER_ID}_first.log 2>/dev/null || echo 0)
SECOND_SUCCESS=$(grep -c "Vote submitted successfully" ../logs/vote_${VOTER_ID}_second.log 2>/dev/null || echo 0)
REJECTION=$(grep -c "already voted" ../logs/node1.log 2>/dev/null || echo 0)

echo ""
echo "Results:"
if [ $FIRST_SUCCESS -eq 1 ] && [ $SECOND_SUCCESS -eq 1 ] && [ $REJECTION -gt 0 ]; then
    echo "  ✅ First vote: Accepted"
    echo "  ✅ Second vote: Submitted but rejected by validators"
    echo "  ✅ System correctly detected double voting"
elif [ $FIRST_SUCCESS -eq 1 ] && [ $SECOND_SUCCESS -eq 0 ]; then
    echo "  ✅ First vote: Accepted"
    echo "  ✅ Second vote: Rejected at submission"
    echo "  ✅ System correctly prevented double voting"
else
    echo "  ❌ Unexpected behavior"
fi

echo ""

# Test 2: Invalid Signature
echo "=========================================="
echo "Test 2: Invalid Signature Attack"
echo "=========================================="
echo ""
echo "Attempting to vote with wrong signature..."

VOTER_ID="voter002"
WRONG_KEY="voter003"  # Using voter003's key to sign voter002's vote

echo "  Voting as $VOTER_ID but signing with $WRONG_KEY's key..."

# This will create a vote with mismatched voterID and signature
(cd ../.. && ./simulation/bin/submit-vote \
    -voter "$VOTER_ID" \
    -choice "candidate-a" \
    -key "simulation/keys/voters/${WRONG_KEY}.key" \
    -peer "$TARGET_PEER") > ../logs/vote_invalid_sig.log 2>&1

sleep 5

# Check if rejected
INVALID_SIG_REJECTION=$(grep -c "signature validation failed\|invalid signature" ../logs/node*.log 2>/dev/null || echo 0)

echo ""
echo "Results:"
if [ $INVALID_SIG_REJECTION -gt 0 ]; then
    echo "  ✅ Invalid signature detected and rejected"
    echo "  ✅ System verified cryptographic integrity"
else
    echo "  ⚠️  Note: System may accept vote if publicKey matches signature"
    echo "     (voterID validation is a known limitation)"
fi

echo ""

# Test 3: Unauthorized Voter
echo "=========================================="
echo "Test 3: Unauthorized Voter Attack"
echo "=========================================="
echo ""
echo "Attempting to vote with non-eligible voter..."

# Create a temporary voter not in eligibility list
TEMP_KEY="../keys/unauthorized_voter.key"
if [ ! -f "$TEMP_KEY" ]; then
    echo "  Generating unauthorized voter key..."
    (cd ../.. && go run cmd/node/main.go -generate-key -key-path "simulation/keys/unauthorized_voter.key") > /dev/null 2>&1 || true
fi

if [ -f "$TEMP_KEY" ]; then
    echo "  Attempting vote from unauthorized voter..."
    (cd ../.. && ./simulation/bin/submit-vote \
        -voter "unauthorized_voter" \
        -choice "candidate-a" \
        -key "simulation/keys/unauthorized_voter.key" \
        -peer "$TARGET_PEER") > ../logs/vote_unauthorized.log 2>&1
    
    sleep 5
    
    UNAUTHORIZED_REJECTION=$(grep -c "not eligible\|eligibility check failed" ../logs/node*.log 2>/dev/null || echo 0)
    
    echo ""
    echo "Results:"
    if [ $UNAUTHORIZED_REJECTION -gt 0 ]; then
        echo "  ✅ Unauthorized voter detected and rejected"
        echo "  ✅ Eligibility check working correctly"
    else
        echo "  ❌ Unauthorized voter may have been accepted"
    fi
else
    echo "  ⚠️  Could not generate unauthorized voter key"
fi

echo ""

# Test 4: Rapid Fire Attack (Spam)
echo "=========================================="
echo "Test 4: Spam Attack"
echo "=========================================="
echo ""
echo "Attempting to spam network with rapid votes..."

echo "  Sending 10 votes in rapid succession..."
for i in {1..10}; do
    (cd ../.. && ./simulation/bin/submit-vote \
        -voter "voter004" \
        -choice "candidate-a" \
        -key "simulation/keys/voters/voter004.key" \
        -peer "$TARGET_PEER") > ../logs/vote_spam_${i}.log 2>&1 &
done

sleep 10

SPAM_ACCEPTED=$(grep -c "Vote submitted successfully" ../logs/vote_spam_*.log 2>/dev/null || echo 0)
SPAM_IN_BLOCKCHAIN=$(grep "voter004" ../logs/node1.log 2>/dev/null | grep -c "added to mempool" || echo 0)

echo ""
echo "Results:"
echo "  Votes submitted: 10"
echo "  Votes accepted by network: $SPAM_ACCEPTED"
echo "  Votes in mempool: $SPAM_IN_BLOCKCHAIN"

if [ $SPAM_IN_BLOCKCHAIN -le 1 ]; then
    echo "  ✅ System accepted only first vote, rejected duplicates"
    echo "  ✅ Spam attack mitigated"
else
    echo "  ⚠️  Multiple votes may have been accepted"
fi

echo ""

# Final Summary
echo "=========================================="
echo "Security Test Summary"
echo "=========================================="
echo ""

TOTAL_TESTS=4
PASSED=0

# Count passed tests
[ $REJECTION -gt 0 ] && PASSED=$((PASSED + 1))
[ $INVALID_SIG_REJECTION -gt 0 ] && PASSED=$((PASSED + 1))
[ $UNAUTHORIZED_REJECTION -gt 0 ] && PASSED=$((PASSED + 1))
[ $SPAM_IN_BLOCKCHAIN -le 1 ] && PASSED=$((PASSED + 1))

echo "Tests Passed: $PASSED/$TOTAL_TESTS"
echo ""

echo "Security Mechanisms Validated:"
echo "  1. Double voting prevention: $([ $REJECTION -gt 0 ] && echo '✅' || echo '❌')"
echo "  2. Signature verification: $([ $INVALID_SIG_REJECTION -gt 0 ] && echo '✅' || echo '⚠️')"
echo "  3. Eligibility checking: $([ $UNAUTHORIZED_REJECTION -gt 0 ] && echo '✅' || echo '❌')"
echo "  4. Spam protection: $([ $SPAM_IN_BLOCKCHAIN -le 1 ] && echo '✅' || echo '❌')"
echo ""

# Check blockchain integrity
FINAL_VOTES=$(grep -h "finalized successfully" ../logs/node1.log 2>/dev/null | tail -1 | grep -o "voted count: [0-9]*" | awk '{print $3}')
if [ -z "$FINAL_VOTES" ]; then
    FINAL_VOTES=0
fi

echo "Blockchain State:"
echo "  Valid votes finalized: $FINAL_VOTES"
ech