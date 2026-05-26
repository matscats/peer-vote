#!/bin/bash

# Script to analyze election results from blockchain logs

cd "$(dirname "$0")"

LOGS_DIR="../logs"

echo "Analyzing blockchain logs..."
echo ""

# Count finalized blocks
BLOCKS_FINALIZED=$(grep -h "finalized successfully" $LOGS_DIR/node*.log | wc -l)
echo "📦 Blocks finalized: $BLOCKS_FINALIZED"

# Count votes in blocks
VOTES_IN_BLOCKS=$(grep -h "Proposed new block.*with.*votes" $LOGS_DIR/node*.log | awk '{sum += $(NF-1)} END {print sum}')
echo "🗳️  Total votes in blocks: ${VOTES_IN_BLOCKS:-0}"

# Count votes by candidate (from vote submission logs)
echo ""
echo "Vote Distribution:"
echo "─────────────────"

if [ -f "$LOGS_DIR/vote_voter001.log" ]; then
    for candidate in candidate-a candidate-b candidate-c; do
        COUNT=$(grep -l "choice=$candidate" $LOGS_DIR/vote_*.log 2>/dev/null | wc -l)
        echo "  $candidate: $COUNT votes"
    done
else
    echo "  (No vote submission logs found)"
fi

# Show recent finalized blocks
echo ""
echo "Recent Finalized Blocks:"
echo "────────────────────────"
grep -h "finalized successfully" $LOGS_DIR/node*.log | tail -5 | while read line; do
    HEIGHT=$(echo "$line" | grep -o "height [0-9]*" | awk '{print $2}')
    VOTED=$(echo "$line" | grep -o "voted count: [0-9]*" | awk '{print $3}')
    echo "  Block #$HEIGHT - Votes: $VOTED"
done

# Check for errors
echo ""
ERRORS=$(grep -h "ERROR" $LOGS_DIR/node*.log | grep -v "no active round" | wc -l)
if [ $ERRORS -gt 0 ]; then
    echo "⚠️  Errors found: $ERRORS"
    echo ""
    echo "Recent errors:"
    grep -h "ERROR" $LOGS_DIR/node*.log | grep -v "no active round" | tail -3
else
    echo "✅ No critical errors"
fi

# Network statistics
echo ""
echo "Network Statistics:"
echo "───────────────────"

# Peer connections - get the most recent count from each node
for i in 1 2 3; do
    if [ -f "$LOGS_DIR/node${i}.log" ]; then
        PEERS=$(grep "Connected peers:" "$LOGS_DIR/node${i}.log" | tail -1 | grep -o "Connected peers: [0-9]*" | awk '{print $3}')
        if [ -n "$PEERS" ]; then
            echo "  Node $i connected peers: $PEERS"
        fi
    fi
done

# DHT discoveries
DHT_DISCOVERIES=$(grep -h "Successfully connected to peer.*via DHT" $LOGS_DIR/node*.log 2>/dev/null | wc -l | tr -d ' ')
echo "  DHT discoveries: $DHT_DISCOVERIES"

# Consensus state
echo ""
echo "Current Consensus State:"
echo "────────────────────────"
for i in 1 2 3; do
    if [ -f "$LOGS_DIR/node${i}.log" ]; then
        STATE=$(grep "Consensus state:" "$LOGS_DIR/node${i}.log" | tail -1 | awk '{print $NF}')
        if [ -n "$STATE" ]; then
            echo "  Node $i: $STATE"
        else
            # Try alternative format
            STATE=$(grep "DEBUG:.*Consensus state:" "$LOGS_DIR/node${i}.log" | tail -1 | awk '{print $NF}')
            echo "  Node $i: ${STATE:-Unknown}"
        fi
    fi
done

echo ""
